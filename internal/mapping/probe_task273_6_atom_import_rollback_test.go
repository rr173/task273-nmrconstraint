package mapping

import (
	"path/filepath"
	"testing"

	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/store"
)

func TestDuplicateAtomImportRollsBackAndReleasesTx(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "probe.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	batches := store.NewBatchStore(db)
	atoms := NewService(store.NewAtomStore(db), batches)
	seed := model.NewBatch("NMR-probe", "")
	b, err := batches.Create(seed)
	if err != nil {
		t.Fatal(err)
	}
	_, err = atoms.Import(b.ID, []*model.Atom{
		model.NewAtom(0, "A", "GLY", "C"),
		model.NewAtom(0, "B", "GLY", "N"),
		model.NewAtom(0, "A", "ALA", "C"),
	})
	if err == nil {
		t.Fatal("expected duplicate atom error")
	}
	list, err := atoms.List(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("duplicate import left %d atoms", len(list))
	}
	got, err := atoms.Import(b.ID, []*model.Atom{
		model.NewAtom(0, "A", "GLY", "C"),
		model.NewAtom(0, "B", "GLY", "N"),
		model.NewAtom(0, "C", "ALA", "C"),
		model.NewAtom(0, "D", "ALA", "N"),
	})
	if err != nil {
		t.Fatalf("follow-up import: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("follow-up count=%d", len(got))
	}
}
