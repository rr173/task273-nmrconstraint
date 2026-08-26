package propagate

import (
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
