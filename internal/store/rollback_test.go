package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"task273-nmrconstraint/internal/model"
)

// newDB 在临时库上构造已迁移的 DB，测试结束自动清理。
func newDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "store_test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestAtomCreateDuplicateRollsBackBatch 验证批量导入原子时若中途出现重复名，
// 整批回滚：不得留下半批原子，且不得让后续导入卡死或误失败。
func TestAtomCreateDuplicateRollsBackBatch(t *testing.T) {
	db := newDB(t)
	atoms := NewAtomStore(db)
	batches := NewBatchStore(db)

	b, err := batches.Create(model.NewBatch("repro", ""))
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}

	// 导入含重复名的批次：前两条合法，第三条与第一条同名，应整体失败。
	_, err = atoms.Create(b.ID, []*model.Atom{
		model.NewAtom(0, "A", "GLY", "C"),
		model.NewAtom(0, "B", "GLY", "N"),
		model.NewAtom(0, "A", "ALA", "C"), // 重复名 A
	})
	if !errors.Is(err, model.ErrDuplicateAtomName) {
		t.Fatalf("expected ErrDuplicateAtomName, got %v", err)
	}

	// 关键断言：失败后批次内不得残留任何半批原子。
	left, err := atoms.List(b.ID)
	if err != nil {
		t.Fatalf("list after rollback: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected 0 atoms after rollback, got %d: %+v", len(left), left)
	}

	// 验证连接未卡死：随后导入全新名字必须成功。
	created, err := atoms.Create(b.ID, []*model.Atom{
		model.NewAtom(0, "X", "GLY", "C"),
		model.NewAtom(0, "Y", "GLY", "N"),
	})
	if err != nil {
		t.Fatalf("subsequent import should succeed after rollback, got %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("expected 2 atoms created, got %d", len(created))
	}
}

// TestAtomCreateDuplicateAcrossBatch 验证不同批次间同名原子合法（UNIQUE 仅覆盖 batch_id+name）。
func TestAtomCreateDuplicateAcrossBatch(t *testing.T) {
	db := newDB(t)
	atoms := NewAtomStore(db)
	batches := NewBatchStore(db)

	b1, err := batches.Create(model.NewBatch("b1", ""))
	if err != nil {
		t.Fatalf("create batch1: %v", err)
	}
	b2, err := batches.Create(model.NewBatch("b2", ""))
	if err != nil {
		t.Fatalf("create batch2: %v", err)
	}

	for _, bid := range []int64{b1.ID, b2.ID} {
		if _, err := atoms.Create(bid, []*model.Atom{
			model.NewAtom(0, "A", "GLY", "C"),
		}); err != nil {
			t.Fatalf("batch %d: import should succeed, got %v", bid, err)
		}
	}
}

// TestConstraintCreateDuplicateRollsBackBatch 验证约束批量导入的中途重复同样整批回滚。
func TestConstraintCreateDuplicateRollsBackBatch(t *testing.T) {
	db := newDB(t)
	batches := NewBatchStore(db)
	atoms := NewAtomStore(db)
	peaks := NewPeakStore(db)
	cons := NewConstraintStore(db)

	b, err := batches.Create(model.NewBatch("repro", ""))
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}

	a, err := atoms.Create(b.ID, []*model.Atom{
		model.NewAtom(0, "A", "GLY", "C"),
		model.NewAtom(0, "B", "GLY", "N"),
	})
	if err != nil {
		t.Fatalf("import atoms: %v", err)
	}
	p, err := peaks.Create(b.ID, []*model.NoePeak{
		model.NewNoePeak(0, "pAB", a[0].ID, a[1].ID, 5.0, 0.9),
	})
	if err != nil {
		t.Fatalf("import peaks: %v", err)
	}

	// 同一对原子重复约束应整体回滚，不留半批。
	_, err = cons.Create(b.ID, []*model.Constraint{
		model.NewConstraint(0, p[0].ID, a[0].ID, a[1].ID, 3.0, 5.0),
		model.NewConstraint(0, p[0].ID, a[0].ID, a[1].ID, 4.0, 6.0), // 重复原子对
	})
	if !errors.Is(err, model.ErrDuplicateConstraint) {
		t.Fatalf("expected ErrDuplicateConstraint, got %v", err)
	}

	list, err := cons.List(b.ID)
	if err != nil {
		t.Fatalf("list constraints: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 constraints after rollback, got %d", len(list))
	}
}

// TestBoundReplaceBatchRollsBackOnCancel 验证传播边界原子替换被取消时整批回滚，不留半表。
func TestBoundReplaceBatchRollsBackOnCancel(t *testing.T) {
	db := newDB(t)
	batches := NewBatchStore(db)
	atoms := NewAtomStore(db)
	peaks := NewPeakStore(db)
	cons := NewConstraintStore(db)
	bounds := NewBoundStore(db)

	b, err := batches.Create(model.NewBatch("repro", ""))
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	a, err := atoms.Create(b.ID, []*model.Atom{
		model.NewAtom(0, "A", "GLY", "C"),
		model.NewAtom(0, "B", "GLY", "N"),
		model.NewAtom(0, "C", "ALA", "C"),
	})
	if err != nil {
		t.Fatalf("import atoms: %v", err)
	}
	p, err := peaks.Create(b.ID, []*model.NoePeak{
		model.NewNoePeak(0, "pAB", a[0].ID, a[1].ID, 5.0, 0.9),
		model.NewNoePeak(0, "pAC", a[0].ID, a[2].ID, 4.0, 0.9),
	})
	if err != nil {
		t.Fatalf("import peaks: %v", err)
	}
	c, err := cons.Create(b.ID, []*model.Constraint{
		model.NewConstraint(0, p[0].ID, a[0].ID, a[1].ID, 4.0, 6.0),
		model.NewConstraint(0, p[1].ID, a[0].ID, a[2].ID, 3.0, 5.0),
	})
	if err != nil {
		t.Fatalf("create constraints: %v", err)
	}

	// 先写入既有边界作为基线数据。
	edges := []*model.BoundEdge{
		{BatchID: b.ID, ConstraintID: c[0].ID, Atom1ID: a[0].ID, Atom2ID: a[1].ID, LoBound: 4.0, HiBound: 6.0, UpdatedAt: time.Now().UTC()},
		{BatchID: b.ID, ConstraintID: c[1].ID, Atom1ID: a[0].ID, Atom2ID: a[2].ID, LoBound: 3.0, HiBound: 5.0, UpdatedAt: time.Now().UTC()},
	}
	if err := bounds.ReplaceBatch(context.Background(), b.ID, edges); err != nil {
		t.Fatalf("seed bounds: %v", err)
	}

	// 取消上下文：DELETE 已执行，插入阶段应因取消回滚，不得丢掉既有边界。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repl := []*model.BoundEdge{
		{BatchID: b.ID, ConstraintID: c[0].ID, Atom1ID: a[0].ID, Atom2ID: a[1].ID, LoBound: 1.0, HiBound: 2.0, UpdatedAt: time.Now().UTC()},
	}
	if err := bounds.ReplaceBatch(ctx, b.ID, repl); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	left, err := bounds.List(b.ID)
	if err != nil {
		t.Fatalf("list bounds: %v", err)
	}
	if len(left) != 2 {
		t.Fatalf("expected 2 bounds preserved after rollback, got %d", len(left))
	}
}
