package propagate

import (
	"fmt"
	"testing"

	"task273-nmrconstraint/internal/model"
)

// mk 构造一条约束。
func mk(id, batch int64, a1, a2 int64, lo, hi float64) *model.Constraint {
	return &model.Constraint{
		ID:      id,
		BatchID: batch,
		Atom1ID: a1,
		Atom2ID: a2,
		LoDist:  lo,
		HiDist:  hi,
		Status:  model.ConstraintValid,
	}
}

// TestPropagateTriangleConsistent 验证可满足的三角不等式传播后收敛且不倒置。
func TestPropagateTriangleConsistent(t *testing.T) {
	cons := []*model.Constraint{
		mk(1, 1, 1, 2, 4.0, 6.0), // AB
		mk(2, 1, 1, 3, 3.0, 5.0), // AC
		mk(3, 1, 2, 3, 2.0, 4.0), // BC：2 <= 4+5 且 4 <= 6+5，一致
	}
	res := Propagate(cons)
	if !res.Converged {
		t.Fatalf("expected convergence, got iterations=%d", res.Iterations)
	}
	if len(res.Inverted) != 0 {
		t.Fatalf("expected no inverted edges, got %d", len(res.Inverted))
	}
	if len(res.Edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(res.Edges))
	}
}

// TestPropagateTriangleInfeasible 验证违反三角不等式时检测到区间倒置。
// BC 的 [12,15] 与 AB [4,6]、AC [3,5] 冲突：12 > 6+5。
func TestPropagateTriangleInfeasible(t *testing.T) {
	cons := []*model.Constraint{
		mk(1, 1, 1, 2, 4.0, 6.0),  // AB
		mk(2, 1, 1, 3, 3.0, 5.0),  // AC
		mk(3, 1, 2, 3, 12.0, 15.0), // BC 违反
	}
	res := Propagate(cons)
	if !res.Converged {
		t.Fatalf("expected convergence (infeasibility detected at fixpoint), got iterations=%d", res.Iterations)
	}
	if len(res.Inverted) == 0 {
		t.Fatalf("expected inverted edge for infeasible triangle")
	}
}

// TestPropagateTighten 验证传播确实收紧边界。
func TestPropagateTighten(t *testing.T) {
	cons := []*model.Constraint{
		mk(1, 1, 1, 2, 1.0, 10.0), // AB 宽区间
		mk(2, 1, 1, 3, 2.0, 3.0),  // AC
		mk(3, 1, 2, 3, 2.0, 3.0),  // BC
	}
	res := Propagate(cons)
	if len(res.Edges) != 3 {
		t.Fatalf("expected 3 edges")
	}
	// AB 上界应被收紧到 min(10, 3+3)=6。
	for _, e := range res.Edges {
		if e.Atom1ID == 1 && e.Atom2ID == 2 || e.Atom1ID == 2 && e.Atom2ID == 1 {
			if e.HiBound > 6.000001 {
				t.Fatalf("expected AB hi bound <= 6, got %v", e.HiBound)
			}
			if !e.Tightened {
				t.Fatalf("expected AB edge marked tightened")
			}
		}
	}
}

// TestPropagateEmpty 验证空约束集安全返回。
func TestPropagateEmpty(t *testing.T) {
	res := Propagate(nil)
	if !res.Converged || len(res.Edges) != 0 {
		t.Fatalf("empty input should converge with zero edges")
	}
}

// TestPropagateConcurrentBatches 验证多批次并发求解时互不踩内存：
// 每个批次的边界条数与原子对集合必须与自身输入完全对应，
// 不得缺边，也不得混入别批次的边。
func TestPropagateConcurrentBatches(t *testing.T) {
	const batches = 20
	type want struct {
		batch    int64
		edgeKeys map[string]bool
	}
	// 每批用互不相交的原子 id 区间，便于事后核对"是否串到别人批次"。
	wants := make([]want, batches)
	results := make([]*Result, batches)
	for b := 0; b < batches; b++ {
		base := int64(b*3 + 1)
		// 一个一致的小三角：lo/hi 互不违反三角不等式。
		cons := []*model.Constraint{
			mk(int64(b*3+1), int64(b), base, base+1, 4.0, 6.0),
			mk(int64(b*3+2), int64(b), base, base+2, 3.0, 5.0),
			mk(int64(b*3+3), int64(b), base+1, base+2, 2.0, 4.0),
		}
		keys := make(map[string]bool, len(cons))
		for _, c := range cons {
			e := model.NewBoundEdge(c.BatchID, c.ID, c.Atom1ID, c.Atom2ID, c.LoDist, c.HiDist)
			keys[e.Key()] = true
		}
		wants[b] = want{batch: int64(b), edgeKeys: keys}
	}

	start := make(chan struct{})
	done := make(chan error, batches)
	for b := 0; b < batches; b++ {
		b := b
		go func() {
			<-start // 尽量让二十路同时进入 Propagate
			base := int64(b*3 + 1)
			cons := []*model.Constraint{
				mk(int64(b*3+1), int64(b), base, base+1, 4.0, 6.0),
				mk(int64(b*3+2), int64(b), base, base+2, 3.0, 5.0),
				mk(int64(b*3+3), int64(b), base+1, base+2, 2.0, 4.0),
			}
			res := Propagate(cons)
			if len(res.Edges) != len(cons) {
				done <- fmt.Errorf("batch %d: expected %d edges, got %d", b, len(cons), len(res.Edges))
				return
			}
			got := make(map[string]bool, len(res.Edges))
			for _, e := range res.Edges {
				if e.BatchID != int64(b) {
					done <- fmt.Errorf("batch %d: edge leaked from batch %d", b, e.BatchID)
					return
				}
				got[e.Key()] = true
			}
			for k := range wants[b].edgeKeys {
				if !got[k] {
					done <- fmt.Errorf("batch %d: missing edge %s", b, k)
					return
				}
			}
			for k := range got {
				if !wants[b].edgeKeys[k] {
					done <- fmt.Errorf("batch %d: stray edge %s from another batch", b, k)
					return
				}
			}
			results[b] = res
			done <- nil
		}()
	}
	close(start)
	for i := 0; i < batches; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	// 复查二十路收敛、无倒置、无一缺边。
	for b := 0; b < batches; b++ {
		res := results[b]
		if !res.Converged {
			t.Fatalf("batch %d: expected convergence, got iterations=%d", b, res.Iterations)
		}
		if len(res.Inverted) != 0 {
			t.Fatalf("batch %d: expected no inverted edges, got %d", b, len(res.Inverted))
		}
		if len(res.Edges) != 3 {
			t.Fatalf("batch %d: expected 3 edges, got %d", b, len(res.Edges))
		}
	}
}
