package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/store"
)

// newService 在临时 SQLite 库上组装一个求解服务，并预置一个冲突三角
// 批次（BC 的 [12,15] 与 AB [4,6]、AC [3,5] 违反三角不等式）。
func newSolveFixture(t *testing.T) (*DiagnosisService, *store.BatchStore, *store.BoundStore, *store.ConstraintStore, int64) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "solve.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	batches := store.NewBatchStore(db)
	atoms := store.NewAtomStore(db)
	peaks := store.NewPeakStore(db)
	cons := store.NewConstraintStore(db)
	bounds := store.NewBoundStore(db)
	sets := store.NewConflictSetStore(db)
	exemps := store.NewExemptionStore(db)
	diag := NewDiagnosisService(batches, cons, bounds, sets, exemps)

	batch, err := batches.Create(model.NewBatch("tri-conflict", "BC [12,15] 违反 AB+AC"))
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	imp, err := atoms.Create(batch.ID, []*model.Atom{
		model.NewAtom(0, "A", "GLY", "C"),
		model.NewAtom(0, "B", "GLY", "N"),
		model.NewAtom(0, "C", "ALA", "C"),
	})
	if err != nil {
		t.Fatalf("import atoms: %v", err)
	}
	byName := map[string]*model.Atom{}
	for _, a := range imp {
		byName[a.Name] = a
	}
	pks, err := peaks.Create(batch.ID, []*model.NoePeak{
		model.NewNoePeak(0, "pAB", byName["A"].ID, byName["B"].ID, 5.0, 0.95),
		model.NewNoePeak(0, "pAC", byName["A"].ID, byName["C"].ID, 4.0, 0.90),
		model.NewNoePeak(0, "pBC", byName["B"].ID, byName["C"].ID, 0.2, 0.85),
	})
	if err != nil {
		t.Fatalf("import peaks: %v", err)
	}
	peakByName := map[string]*model.NoePeak{}
	for _, p := range pks {
		peakByName[p.Name] = p
	}
	if _, err := cons.Create(batch.ID, []*model.Constraint{
		model.NewConstraint(0, peakByName["pAB"].ID, byName["A"].ID, byName["B"].ID, 4.0, 6.0),
		model.NewConstraint(0, peakByName["pAC"].ID, byName["A"].ID, byName["C"].ID, 3.0, 5.0),
		model.NewConstraint(0, peakByName["pBC"].ID, byName["B"].ID, byName["C"].ID, 12.0, 15.0),
	}); err != nil {
		t.Fatalf("create constraints: %v", err)
	}
	return diag, batches, bounds, cons, batch.ID
}

// TestSolveCancelledLeavesNoBounds 验证：求解请求被取消后，不得落盘传播边界、
// 不得推进批次状态，且按 context.Canceled 返回。
func TestSolveCancelledLeavesNoBounds(t *testing.T) {
	diag, batches, bounds, _, batchID := newSolveFixture(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 调用方在求解前已取消

	_, err := diag.Solve(ctx, batchID)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// 边界表必须为空：取消后不得留下半成品边界。
	edges, err := bounds.List(batchID)
	if err != nil {
		t.Fatalf("list bounds: %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("cancelled solve must not persist bounds, got %d edges", len(edges))
	}

	// 批次状态必须停留在 receiving：取消不得推进状态机。
	b, err := batches.Get(batchID)
	if err != nil {
		t.Fatalf("get batch: %v", err)
	}
	if b.Status != model.BatchReceiving {
		t.Fatalf("cancelled solve must not advance batch, status=%s", b.Status)
	}
}

// TestSolveDeadlineExceeded 验证：超时上下文按 context.DeadlineExceeded 返回且不留边界。
func TestSolveDeadlineExceeded(t *testing.T) {
	diag, _, bounds, _, batchID := newSolveFixture(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	// 让 deadline 到期后再求解。
	<-ctx.Done()

	_, err := diag.Solve(ctx, batchID)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}

	edges, err := bounds.List(batchID)
	if err != nil {
		t.Fatalf("list bounds: %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("timed-out solve must not persist bounds, got %d edges", len(edges))
	}
}

// TestSolveSuccessPersistsBounds 验证：正常上下文下求解落盘完整边界并推进批次状态。
// 作为对照，确保取消分支没有破坏正常路径。
func TestSolveSuccessPersistsBounds(t *testing.T) {
	diag, batches, bounds, cons, batchID := newSolveFixture(t)

	result, err := diag.Solve(context.Background(), batchID)
	if err != nil {
		t.Fatalf("solve: %v", err)
	}
	if !result.HasConflict {
		t.Fatalf("expected conflict, got none")
	}
	if result.BatchStatus != model.BatchConflicted {
		t.Fatalf("expected conflicted status, got %s", result.BatchStatus)
	}
	if result.EdgeCount != 3 {
		t.Fatalf("expected 3 edges, got %d", result.EdgeCount)
	}

	edges, err := bounds.List(batchID)
	if err != nil {
		t.Fatalf("list bounds: %v", err)
	}
	if len(edges) != 3 {
		t.Fatalf("expected 3 persisted edges, got %d", len(edges))
	}

	// 约束状态落盘：BC 应为 conflicted。
	all, err := cons.List(batchID)
	if err != nil {
		t.Fatalf("list constraints: %v", err)
	}
	conflicted := 0
	for _, c := range all {
		if c.Status == model.ConstraintConflicted {
			conflicted++
		}
	}
	if conflicted == 0 {
		t.Fatalf("expected at least one conflicted constraint")
	}

	// 批次状态推进到 conflicted。
	b, err := batches.Get(batchID)
	if err != nil {
		t.Fatalf("get batch: %v", err)
	}
	if b.Status != model.BatchConflicted {
		t.Fatalf("expected batch conflicted, got %s", b.Status)
	}
}
