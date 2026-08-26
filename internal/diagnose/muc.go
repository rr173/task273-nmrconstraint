package diagnose

import (
	"sort"

	"task273-nmrconstraint/internal/model"
)

// MinimalConflictSet 用贪心删减法求一个最小冲突集（MUC）。
//
// 输入：候选冲突约束集合（全部参与传播后仍不可满足）。
// 输出：C' ⊆ C 使得 C' 仍不可满足，且删除 C' 中任何一条
// 约束后即变为可满足——即 C' 是极小的不可满足子集。
//
// 算法（单轮删减）：
//  1. 依次考察 C 中每条约束 c；
//  2. 暂时从 C 中移除 c，对剩余集合重新传播；
//  3. 若剩余集合仍冲突，则 c 不是必要成员，永久移除；
//     若剩余集合可满足，则 c 是必要成员，放回；
//  4. 考察完所有约束后，剩余集合即为一个 MUC。
func MinimalConflictSet(candidates []*model.Constraint) ([]*model.Constraint, bool) {
	if len(candidates) == 0 {
		return nil, false
	}
	// 输入本身必须冲突。
	if conflict, _, _ := HasConflict(candidates); !conflict {
		return nil, false
	}

	working := make([]*model.Constraint, len(candidates))
	copy(working, candidates)

	for i := 0; i < len(working); i++ {
		probe := make([]*model.Constraint, 0, len(working)-1)
		for j, c := range working {
			if j != i {
				probe = append(probe, c)
			}
		}
		conflict, _, _ := HasConflict(probe)
		if !conflict {
			// 移除 c 后变得可满足：c 是必要成员，保留。
			continue
		}
		// c 可移除：从工作集删除。
		working = append(working[:i], working[i+1:]...)
		i--
	}

	sort.Slice(working, func(a, b int) bool { return working[a].ID < working[b].ID })
	return working, true
}

// InvolvedConstraintIDs 汇总冲突证据涉及的约束 ID 集合。
func InvolvedConstraintIDs(inverted []*model.BoundEdge, violations []Violation) []int64 {
	ids := map[int64]bool{}
	for _, e := range inverted {
		ids[e.ConstraintID] = true
	}
	for _, v := range violations {
		ids[v.ConstraintID] = true
	}
	out := make([]int64, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
