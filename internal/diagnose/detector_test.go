package diagnose

import (
	"testing"

	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/propagate"
)

func mkConstraint(id, batch int64, a1, a2 int64, lo, hi float64) *model.Constraint {
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

// TestHasConflictTriangle 验证三角不等式冲突被识别。
// 三角违反经传播后表现为区间倒置（lo > hi）或三角违反证据。
func TestHasConflictTriangle(t *testing.T) {
	cons := []*model.Constraint{
		mkConstraint(1, 1, 1, 2, 4.0, 6.0),
		mkConstraint(2, 1, 1, 3, 3.0, 5.0),
		mkConstraint(3, 1, 2, 3, 12.0, 15.0),
	}
	conflict, edges, violations := HasConflict(cons)
	if !conflict {
		t.Fatalf("expected conflict, got none")
	}
	if len(edges) != 3 {
		t.Fatalf("expected 3 propagated edges")
	}
	// 必须有倒置边或三角违反证据。
	if len(violations) == 0 {
		inverted := 0
		for _, e := range edges {
			if e.Inverted() {
				inverted++
			}
		}
		if inverted == 0 {
			t.Fatalf("expected inverted edges or violations")
		}
	}
}

// TestHasConflictConsistent 验证一致约束集不误报。
func TestHasConflictConsistent(t *testing.T) {
	cons := []*model.Constraint{
		mkConstraint(1, 1, 1, 2, 4.0, 6.0),
		mkConstraint(2, 1, 1, 3, 3.0, 5.0),
		mkConstraint(3, 1, 2, 3, 2.0, 4.0),
	}
	conflict, _, violations := HasConflict(cons)
	if conflict {
		t.Fatalf("expected no conflict, got %d violations", len(violations))
	}
}

// TestInvolvedConstraintIDs 验证证据 → 约束 ID 集合汇总。
func TestInvolvedConstraintIDs(t *testing.T) {
	res := propagate.Propagate([]*model.Constraint{
		mkConstraint(1, 1, 1, 2, 4.0, 6.0),
		mkConstraint(2, 1, 1, 3, 3.0, 5.0),
		mkConstraint(3, 1, 2, 3, 12.0, 15.0),
	})
	violations := DetectViolations(res.Edges)
	ids := InvolvedConstraintIDs(res.Inverted, violations)
	if len(ids) == 0 {
		t.Fatalf("expected involved constraint ids")
	}
	// 必须包含约束 3。
	found := false
	for _, id := range ids {
		if id == 3 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected constraint 3 in involved ids, got %v", ids)
	}
}
