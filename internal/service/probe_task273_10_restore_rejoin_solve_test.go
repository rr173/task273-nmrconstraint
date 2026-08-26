package service

import (
	"context"

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

func TestRestorePutsConstraintBackIntoSolve(t *testing.T) {
	st := newProbeStack(t)
	batchID, cons := seedTriangle(t, st)
	if _, err := st.diag.Solve(context.Background(), batchID); err != nil {
		t.Fatal(err)
	}
	target := cons[2].ID
	if _, err := st.diag.Exempt(batchID, target, "peak overlap"); err != nil {
		t.Fatal(err)
	}
	clean, err := st.diag.Solve(context.Background(), batchID)
	if err != nil {
		t.Fatal(err)
	}
	if clean.HasConflict {
		t.Fatal("expected clean after exemption")
	}
	if _, err := st.cons.Restore(target); err != nil {
		t.Fatal(err)
	}
	again, err := st.diag.Solve(context.Background(), batchID)
	if err != nil {
		t.Fatal(err)
	}
	if !again.HasConflict {
		t.Fatal("restored triangle must conflict again")
	}
	if again.BatchStatus != model.BatchConflicted {
		t.Fatalf("status=%s want conflicted", again.BatchStatus)
	}
	edges, err := st.diag.ListBounds(batchID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range edges {
		if e.ConstraintID == target {
			found = true
		}
	}
	if !found {
		t.Fatalf("restored constraint %d missing from bounds (n=%d)", target, len(edges))
	}
}
