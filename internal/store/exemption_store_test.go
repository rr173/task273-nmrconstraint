package store

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"task273-nmrconstraint/internal/model"
)

// TestExemptionConcurrentSingleWinner 复现"二十位研究者同时豁免同一条冲突约束"的场景：
// UNIQUE(batch_id, constraint_id) 保证仅一条记录落盘，其余重复请求必须报告
// model.ErrExemptionExists，而非伪造成功或产生多条豁免记录。
func TestExemptionConcurrentSingleWinner(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "concurrent-exempt.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() {
		_ = db.Close()
		_ = os.Remove(dbPath)
	}()

	batchStore := NewBatchStore(db)
	atomStore := NewAtomStore(db)
	peakStore := NewPeakStore(db)
	consStore := NewConstraintStore(db)
	exStore := NewExemptionStore(db)

	b, err := batchStore.Create(model.NewBatch("concurrent-exempt", "twelve researchers"))
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	atoms, err := atomStore.Create(b.ID, []*model.Atom{
		model.NewAtom(0, "A", "GLY", "C"),
		model.NewAtom(0, "B", "GLY", "N"),
	})
	if err != nil {
		t.Fatalf("create atoms: %v", err)
	}
	peaks, err := peakStore.Create(b.ID, []*model.NoePeak{
		model.NewNoePeak(0, "pAB", atoms[0].ID, atoms[1].ID, 5.0, 0.9),
	})
	if err != nil {
		t.Fatalf("create peaks: %v", err)
	}
	cons, err := consStore.Create(b.ID, []*model.Constraint{
		model.NewConstraint(0, peaks[0].ID, atoms[0].ID, atoms[1].ID, 4.0, 6.0),
	})
	if err != nil {
		t.Fatalf("create constraints: %v", err)
	}
	constraintID := cons[0].ID

	const researchers = 20
	var (
		wg        sync.WaitGroup
		start     = make(chan struct{})
		succCount int64
		existsErr int64
		otherErr  int64
	)
	wg.Add(researchers)
	for i := 0; i < researchers; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			e := model.NewExemption(b.ID, constraintID, "peak overlap")
			e.CreatedAt = time.Now().UTC()
			_, err := exStore.Create(e)
			switch {
			case err == nil:
				atomic.AddInt64(&succCount, 1)
			case errors.Is(err, model.ErrExemptionExists):
				atomic.AddInt64(&existsErr, 1)
			default:
				atomic.AddInt64(&otherErr, 1)
				t.Errorf("researcher %d: unexpected error %v", i, err)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	if got := atomic.LoadInt64(&succCount); got != 1 {
		t.Fatalf("expected exactly 1 successful exemption, got %d", got)
	}
	if got := atomic.LoadInt64(&existsErr); got != researchers-1 {
		t.Fatalf("expected %d already-exempted errors, got %d", researchers-1, got)
	}
	if got := atomic.LoadInt64(&otherErr); got != 0 {
		t.Fatalf("expected 0 other errors, got %d", got)
	}

	// 仅有唯一一条豁免记录落盘。
	list, err := exStore.List(b.ID)
	if err != nil {
		t.Fatalf("list exemptions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 persisted exemption, got %d", len(list))
	}
	if list[0].ConstraintID != constraintID {
		t.Fatalf("persisted exemption constraint=%d, want %d", list[0].ConstraintID, constraintID)
	}
	if list[0].ID == 0 {
		t.Fatalf("persisted exemption has zero id (phantom record)")
	}
}
