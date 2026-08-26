package diagnose

import (
	"testing"

	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/propagate"
)

// TestMinimalConflictSet 验证 MUC：删除任一必要成员后冲突消失。
func TestMinimalConflictSet(t *testing.T) {
	// 三角 ABC：BC 与 AB/AC 冲突。
	candidates := []*model.Constraint{
		mkConstraint(1, 1, 1, 2, 4.0, 6.0),   // AB
		mkConstraint(2, 1, 1, 3, 3.0, 5.0),   // AC
		mkConstraint(3, 1, 2, 3, 12.0, 15.0), // BC
	}
	minimal, ok := MinimalConflictSet(candidates)
	if !ok {
		t.Fatalf("expected minimal conflict set")
	}
	if len(minimal) < 2 {
		t.Fatalf("expected at least 2 necessary constraints, got %d", len(minimal))
	}
	// 任何必要约束单独移除后应可满足。
	for i := range minimal {
		probe := make([]*model.Constraint, 0, len(minimal)-1)
		for j, c := range minimal {
			if j != i {
				probe = append(probe, c)
			}
		}
		if conflict, _, _ := HasConflict(probe); conflict {
			t.Fatalf("removing constraint %d should resolve conflict", minimal[i].ID)
		}
	}
}

// TestMinimalConflictSetSatisfiableInput 验证可满足输入返回 false。
func TestMinimalConflictSetSatisfiableInput(t *testing.T) {
	candidates := []*model.Constraint{
		mkConstraint(1, 1, 1, 2, 4.0, 6.0),
		mkConstraint(2, 1, 1, 3, 3.0, 5.0),
		mkConstraint(3, 1, 2, 3, 2.0, 4.0),
	}
	_, ok := MinimalConflictSet(candidates)
	if ok {
		t.Fatalf("satisfiable input must not yield a conflict set")
	}
}

// TestMinimalConflictSetEmpty 验证空输入安全。
func TestMinimalConflictSetEmpty(t *testing.T) {
	_, ok := MinimalConflictSet(nil)
	if ok {
		t.Fatalf("empty input must not yield a conflict set")
	}
}

// TestConflictKind 验证冲突类型判定（倒置优先于三角违反）。
func TestConflictKind(t *testing.T) {
	// 三角违反输入：BC 的 [12,15] 使传播后 AB 与 BC 区间倒置。
	cons := []*model.Constraint{
		mkConstraint(1, 1, 1, 2, 4.0, 6.0),
		mkConstraint(2, 1, 1, 3, 3.0, 5.0),
		mkConstraint(3, 1, 2, 3, 12.0, 15.0),
	}
	res := propagate.Propagate(cons)
	if len(res.Inverted) == 0 {
		t.Fatalf("expected inverted edges")
	}
	kind := ConflictKindOf(res.Inverted, nil)
	if kind != model.ConflictInterval {
		t.Fatalf("expected interval kind, got %v", kind)
	}
}
