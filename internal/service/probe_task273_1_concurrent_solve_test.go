package service

import (
	"context"
	"fmt"
	"sync"

	"path/filepath"
	"testing"

	"task273-nmrconstraint/internal/constraint"
	"task273-nmrconstraint/internal/mapping"
	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/snapshot"
	"task273-nmrconstraint/internal/store"
)

type probeStack struct {
	db      *store.DB
	batches *BatchService
	atoms   *mapping.Service
	peaks   *PeakService
	cons    *constraint.Service
	diag    *DiagnosisService
	snaps   *snapshot.Service
}

func newProbeStack(t *testing.T) *probeStack {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "probe.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	bs := store.NewBatchStore(db)
	as := store.NewAtomStore(db)
	ps := store.NewPeakStore(db)
	cs := store.NewConstraintStore(db)
	bds := store.NewBoundStore(db)
	ss := store.NewConflictSetStore(db)
	es := store.NewExemptionStore(db)
	sn := store.NewSnapshotStore(db)
	peaks := NewPeakService(ps, bs)
	return &probeStack{
		db:      db,
		batches: NewBatchService(bs),
		atoms:   mapping.NewService(as, bs),
		peaks:   peaks,
		cons:    constraint.NewService(bs, as, ps, cs),
		diag:    NewDiagnosisService(bs, cs, bds, ss, es),
		snaps:   newSnapshotService(bs, cs, bds, sn, es, peaks.ConfidenceVersion),
	}
}

func seedTriangle(t *testing.T, st *probeStack) (int64, []*model.Constraint) {
	t.Helper()
	b, err := st.batches.Create("NMR-probe", "triangle")
	if err != nil {
		t.Fatal(err)
	}
	atoms, err := st.atoms.Import(b.ID, []*model.Atom{
		model.NewAtom(0, "A", "GLY", "C"),
		model.NewAtom(0, "B", "GLY", "N"),
		model.NewAtom(0, "C", "ALA", "C"),
		model.NewAtom(0, "D", "ALA", "N"),
	})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*model.Atom{}
	for _, a := range atoms {
		byName[a.Name] = a
	}
	peaks, err := st.peaks.Import(b.ID, []*model.NoePeak{
		model.NewNoePeak(0, "pAB", byName["A"].ID, byName["B"].ID, 5.0, 0.95),
		model.NewNoePeak(0, "pAC", byName["A"].ID, byName["C"].ID, 4.0, 0.90),
		model.NewNoePeak(0, "pBC", byName["B"].ID, byName["C"].ID, 0.2, 0.85),
	})
	if err != nil {
		t.Fatal(err)
	}
	peakBy := map[string]*model.NoePeak{}
	for _, p := range peaks {
		peakBy[p.Name] = p
	}
	cons, err := st.cons.CreateFromPeaks(b.ID, []*model.Constraint{
		model.NewConstraint(0, peakBy["pAB"].ID, byName["A"].ID, byName["B"].ID, 4.0, 6.0),
		model.NewConstraint(0, peakBy["pAC"].ID, byName["A"].ID, byName["C"].ID, 3.0, 5.0),
		model.NewConstraint(0, peakBy["pBC"].ID, byName["B"].ID, byName["C"].ID, 12.0, 15.0),
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.ID, cons
}

func newSnapshotService(batches *store.BatchStore, cons *store.ConstraintStore, bounds *store.BoundStore, snaps *store.SnapshotStore, exemps *store.ExemptionStore, peakConf func(int64) (int, error)) *snapshot.Service {
	return snapshot.NewService(batches, cons, bounds, snaps, exemps, peakConf)
}

func TestConcurrentSolveIsolatesBatches(t *testing.T) {
	st := newProbeStack(t)
	const workers = 20
	ids := make([]int64, workers)
	for i := 0; i < workers; i++ {
		id, _ := seedTriangle(t, st)
		ids[i] = id
	}
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(batchID int64) {
			defer wg.Done()
			res, err := st.diag.Solve(context.Background(), batchID)
			if err != nil {
				errCh <- fmt.Errorf("batch %d solve: %w", batchID, err)
				return
			}
			if res.BatchID != batchID {
				errCh <- fmt.Errorf("result batch %d want %d", res.BatchID, batchID)
				return
			}
			if !res.HasConflict {
				errCh <- fmt.Errorf("batch %d want conflict", batchID)
				return
			}
			edges, err := st.diag.ListBounds(batchID)
			if err != nil {
				errCh <- err
				return
			}
			if len(edges) != 3 {
				errCh <- fmt.Errorf("batch %d bounds=%d want 3", batchID, len(edges))
			}
		}(ids[i])
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}
