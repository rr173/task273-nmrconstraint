package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"task273-nmrconstraint/internal/model"
)

// newTestDB 打开一个临时 SQLite 库并应用迁移。
func newTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "snapshot_test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedSnapshotWithBounds 建批次、3 条约束与传播边界，返回批次/约束/快照 store。
// BC 边上界被模拟传播收紧到 11（现场值），用于建立快照冻结基线。
func seedSnapshotWithBounds(t *testing.T, db *DB) (batchID int64, cons []*model.Constraint, snaps *SnapshotStore, bnd *BoundStore) {
	t.Helper()
	bs := NewBatchStore(db)
	b, err := bs.Create(model.NewBatch("snapshot-freeze", "snapshot freeze regression"))
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	// constraints 表的外键要求 peak 存在；先建一对原子与峰以合法挂载约束。
	as := NewAtomStore(db)
	atoms, err := as.Create(b.ID, []*model.Atom{
		model.NewAtom(0, "A", "GLY", "C"),
		model.NewAtom(0, "B", "GLY", "N"),
		model.NewAtom(0, "C", "ALA", "C"),
	})
	if err != nil {
		t.Fatalf("create atoms: %v", err)
	}
	ps := NewPeakStore(db)
	peaks, err := ps.Create(b.ID, []*model.NoePeak{
		model.NewNoePeak(0, "pAB", atoms[0].ID, atoms[1].ID, 5.0, 0.95),
		model.NewNoePeak(0, "pAC", atoms[0].ID, atoms[2].ID, 4.0, 0.90),
		model.NewNoePeak(0, "pBC", atoms[1].ID, atoms[2].ID, 0.2, 0.85),
	})
	if err != nil {
		t.Fatalf("create peaks: %v", err)
	}
	cs := NewConstraintStore(db)
	cons, err = cs.Create(b.ID, []*model.Constraint{
		model.NewConstraint(0, peaks[0].ID, atoms[0].ID, atoms[1].ID, 4.0, 6.0),
		model.NewConstraint(0, peaks[1].ID, atoms[0].ID, atoms[2].ID, 3.0, 5.0),
		model.NewConstraint(0, peaks[2].ID, atoms[1].ID, atoms[2].ID, 12.0, 15.0),
	})
	if err != nil {
		t.Fatalf("create constraints: %v", err)
	}
	bnd = NewBoundStore(db)
	edges := []*model.BoundEdge{
		model.NewBoundEdge(b.ID, cons[0].ID, atoms[0].ID, atoms[1].ID, 4.0, 6.0),
		model.NewBoundEdge(b.ID, cons[1].ID, atoms[0].ID, atoms[2].ID, 3.0, 5.0),
		model.NewBoundEdge(b.ID, cons[2].ID, atoms[1].ID, atoms[2].ID, 12.0, 15.0),
	}
	// 模拟传播收紧：BC 上界由 15 收到 11（现场值），快照应冻结到 11。
	edges[2].Tighten(12.0, 11.0)
	if err := bnd.ReplaceBatch(context.Background(), b.ID, edges); err != nil {
		t.Fatalf("replace bounds: %v", err)
	}
	snaps = NewSnapshotStore(db)
	return b.ID, cons, snaps, bnd
}

// TestSnapshotItemsFreezeBoundsAtCreation 验证快照条目的 lo_bound/hi_bound
// 在创建时刻冻结：之后豁免、重新求解等改动现场传播边界，不得让已保存的快照
// 条目跟着现场 bound_edges 漂移（曾因 Items() LEFT JOIN bound_edges 而 regress）。
func TestSnapshotItemsFreezeBoundsAtCreation(t *testing.T) {
	db := newTestDB(t)
	batchID, cons, snaps, bnd := seedSnapshotWithBounds(t, db)

	// 创建草稿快照——此刻 BC 边上界冻结为 11。
	snap := model.NewSnapshot(batchID, "v1", 1)
	live, err := bnd.List(batchID)
	if err != nil {
		t.Fatalf("list bounds: %v", err)
	}
	edgeByC := map[int64]*model.BoundEdge{}
	for _, e := range live {
		edgeByC[e.ConstraintID] = e
	}
	items := make([]*model.SnapshotItem, 0, len(cons))
	for _, c := range cons {
		items = append(items, model.NewSnapshotItem(snap.ID, c, edgeByC[c.ID]))
	}
	if _, err := snaps.Create(snap, items); err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	// 读回冻结值，建立基线。
	got, err := snaps.Items(snap.ID)
	if err != nil {
		t.Fatalf("snapshot items: %v", err)
	}
	if len(got) != len(cons) {
		t.Fatalf("expected %d items, got %d", len(cons), len(got))
	}
	baseline := map[int64]model.SnapshotItem{}
	for _, it := range got {
		baseline[it.ConstraintID] = *it
	}
	if baseline[cons[2].ID].HiBound != 11.0 {
		t.Fatalf("expected frozen BC hi_bound=11.0 at creation, got %v", baseline[cons[2].ID].HiBound)
	}

	// 模拟豁免后重新求解：替换现场 bound_edges，删掉 BC 边、收紧 AB。
	// 现场数据彻底改变，但已冻结的快照条目不得随之变化。
	newEdges := []*model.BoundEdge{
		model.NewBoundEdge(batchID, cons[0].ID, 1, 2, 4.0, 5.0), // AB 收紧
		model.NewBoundEdge(batchID, cons[1].ID, 1, 3, 3.0, 5.0), // AC 不变
		// BC 被豁免，从现场边界中消失。
	}
	if err := bnd.ReplaceBatch(context.Background(), batchID, newEdges); err != nil {
		t.Fatalf("replace bounds after exempt: %v", err)
	}

	// 现场边界确实变了——保证后续断言不是空操作。
	live2, err := bnd.List(batchID)
	if err != nil {
		t.Fatalf("list bounds after mutate: %v", err)
	}
	if len(live2) != 2 {
		t.Fatalf("expected 2 live edges after exempt, got %d", len(live2))
	}

	// 快照条目必须与创建时基线逐字段一致。
	got2, err := snaps.Items(snap.ID)
	if err != nil {
		t.Fatalf("snapshot items after mutate: %v", err)
	}
	if len(got2) != len(cons) {
		t.Fatalf("expected %d items after mutate, got %d", len(cons), len(got2))
	}
	for _, it := range got2 {
		base, ok := baseline[it.ConstraintID]
		if !ok {
			t.Fatalf("unexpected item constraint %d after mutate", it.ConstraintID)
		}
		if it.LoBound != base.LoBound || it.HiBound != base.HiBound {
			t.Fatalf("snapshot item constraint %d bound drifted after live bound mutation: lo %v->%v hi %v->%v",
				it.ConstraintID, base.LoBound, it.LoBound, base.HiBound, it.HiBound)
		}
		if it.LoDist != base.LoDist || it.HiDist != base.HiDist {
			t.Fatalf("snapshot item constraint %d dist drifted", it.ConstraintID)
		}
		if it.Status != base.Status {
			t.Fatalf("snapshot item constraint %d status drifted", it.ConstraintID)
		}
	}

	_ = time.Now
}
