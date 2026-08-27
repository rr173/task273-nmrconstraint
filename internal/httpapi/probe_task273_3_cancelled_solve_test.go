package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"path/filepath"
	"testing"

	"task273-nmrconstraint/internal/constraint"
	"task273-nmrconstraint/internal/mapping"
	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/service"
	"task273-nmrconstraint/internal/snapshot"
	"task273-nmrconstraint/internal/store"
)

type probeStack struct {
	db      *store.DB
	batches *service.BatchService
	atoms   *mapping.Service
	peaks   *service.PeakService
	cons    *constraint.Service
	diag    *service.DiagnosisService
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
	peaks := service.NewPeakService(ps, bs)
	return &probeStack{
		db:      db,
		batches: service.NewBatchService(bs),
		atoms:   mapping.NewService(as, bs),
		peaks:   peaks,
		cons:    constraint.NewService(bs, as, ps, cs),
		diag:    service.NewDiagnosisService(bs, cs, bds, ss, es),
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

func newAPI(t *testing.T) (http.Handler, *probeStack) {
	t.Helper()
	st := newProbeStack(t)
	h := NewServer(
		st.db,
		st.batches,
		st.atoms,
		st.peaks,
		st.cons,
		st.diag,
		st.snaps,
	).Handler()
	return h, st
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any, ctx context.Context) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(raw)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	req := httptest.NewRequest(method, path, rdr).WithContext(ctx)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestCancelledSolveDoesNotPersistBounds(t *testing.T) {
	h, st := newAPI(t)
	batchID, _ := seedTriangle(t, st)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rr := doJSON(t, h, http.MethodPost, fmt.Sprintf("/api/batches/%d/solve", batchID), map[string]any{}, ctx)
	if rr.Code != http.StatusRequestTimeout {
		t.Fatalf("cancelled solve status=%d body=%s want 408", rr.Code, rr.Body.String())
	}
	edges, err := st.diag.ListBounds(batchID)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 0 {
		t.Fatalf("cancelled solve persisted %d bounds", len(edges))
	}
	b, err := st.batches.Get(batchID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != model.BatchReceiving {
		t.Fatalf("batch status=%s want receiving", b.Status)
	}
}
