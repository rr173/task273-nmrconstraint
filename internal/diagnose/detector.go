// Package diagnose 负责从传播结果中定位冲突：三角不等式违反
// 与区间倒置，并基于删减法求解最小冲突集（MUC）。
package diagnose

import (
	"fmt"
	"sort"

	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/propagate"
)

// Violation 描述一条三角不等式违反：边 (i,j) 的下界大于
// 经过中间原子 k 的可达上界。
type Violation struct {
	ConstraintID int64   `json:"constraint_id"`
	Atom1ID      int64   `json:"atom1_id"`
	Atom2ID      int64   `json:"atom2_id"`
	MidAtomID    int64   `json:"mid_atom_id"`
	LoBound      float64 `json:"lo_bound"`
	MaxReachable float64 `json:"max_reachable"`
}

// DetectViolations 在传播后的边界上检测三角不等式违反。
// 对每条边 (i,j) 和每个中间原子 k：若 lo_ij > hi_ik + hi_kj，
// 则 (i,j) 不可能被 (i,k) 与 (k,j) 同时满足。
func DetectViolations(edges []*model.BoundEdge) []Violation {
	byKey := make(map[string]*model.BoundEdge, len(edges))
	for _, e := range edges {
		if e.Inverted() {
			continue
		}
		byKey[e.Key()] = e
	}
	var out []Violation
	for _, e := range edges {
		for _, other := range edges {
			if other.ConstraintID == e.ConstraintID {
				continue
			}
			var k int64
			switch {
			case other.Atom1ID == e.Atom1ID && other.Atom2ID != e.Atom2ID:
				k = other.Atom2ID
			case other.Atom2ID == e.Atom1ID && other.Atom1ID != e.Atom2ID:
				k = other.Atom1ID
			case other.Atom1ID == e.Atom2ID && other.Atom2ID != e.Atom1ID:
				k = other.Atom2ID
			case other.Atom2ID == e.Atom2ID && other.Atom1ID != e.Atom1ID:
				k = other.Atom1ID
			default:
				continue
			}
			edgeItoK, ok := byKey[keyOf(e.Atom1ID, k)]
			if !ok {
				continue
			}
			edgeKtoJ, ok := byKey[keyOf(k, e.Atom2ID)]
			if !ok {
				continue
			}
			maxReachable := edgeItoK.HiBound + edgeKtoJ.HiBound
			if e.LoBound > maxReachable+1e-9 {
				out = append(out, Violation{
					ConstraintID: e.ConstraintID,
					Atom1ID:      e.Atom1ID,
					Atom2ID:      e.Atom2ID,
					MidAtomID:    k,
					LoBound:      e.LoBound,
					MaxReachable: maxReachable,
				})
			}
		}
	}
	// 去重：同一约束对多个中间原子可能报告多次。
	seen := map[string]bool{}
	var dedup []Violation
	sort.Slice(out, func(i, j int) bool {
		if out[i].ConstraintID != out[j].ConstraintID {
			return out[i].ConstraintID < out[j].ConstraintID
		}
		return out[i].MidAtomID < out[j].MidAtomID
	})
	for _, v := range out {
		k := keyOf(v.ConstraintID, v.MidAtomID)
		if seen[k] {
			continue
		}
		seen[k] = true
		dedup = append(dedup, v)
	}
	return dedup
}

// HasConflict 判断一组约束经传播后是否不可满足（区间倒置或三角违反）。
func HasConflict(constraints []*model.Constraint) (bool, []*model.BoundEdge, []Violation) {
	res := propagate.Propagate(constraints)
	if len(res.Inverted) > 0 {
		return true, res.Edges, nil
	}
	violations := DetectViolations(res.Edges)
	return len(violations) > 0, res.Edges, violations
}

// ConflictKindOf 根据证据判断冲突类型。
func ConflictKindOf(inverted []*model.BoundEdge, violations []Violation) model.ConflictKind {
	if len(inverted) > 0 {
		return model.ConflictInterval
	}
	return model.ConflictTriangle
}

func keyOf(a, b int64) string {
	if a < b {
		return fmt.Sprintf("%d:%d", a, b)
	}
	return fmt.Sprintf("%d:%d", b, a)
}
