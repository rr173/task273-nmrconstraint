// Package propagate 实现距离约束图上的三角不等式边界传播。
//
// 对任意三个原子 i、j、k，若它们之间存在距离界，则三角不等式
// 要求：d_ij ≤ d_ik + d_kj 且 d_ij ≥ |d_ik - d_kj|。
// 用区间形式表示：对界 [lo_ij, hi_ij] 可作如下收紧：
//
//	lo_ij = max(lo_ij, lo_ik - hi_kj)
//	hi_ij = min(hi_ij, hi_ik + hi_kj)
//
// 传播反复迭代直到达到不动点；若某条边出现 lo > hi（区间倒置），
// 则该约束集不可满足，即检测到冲突。边界传播在求解（solve）时
// 幂等落盘到 bound_edges 表，重启后可恢复上一轮传播结果。
package propagate

import (
	"fmt"
	"sort"

	"task273-nmrconstraint/internal/model"
)

// Result 是一次传播的完整输出。
type Result struct {
	Edges      []*model.BoundEdge // 传播后的边界（按约束 ID 排序）
	Iterations int                // 达到收敛所需迭代轮数
	Inverted   []*model.BoundEdge // 区间倒置的边（不可满足证据）
	Converged  bool               // 是否收敛（无变化）
}

// Propagate 对一组距离约束执行距离界传播。
// constraints 必须已做去重（同一原子对仅一条约束）。
var sharedEdges = map[string]*model.BoundEdge{}

func Propagate(constraints []*model.Constraint) *Result {
	if len(constraints) == 0 {
		return &Result{Edges: []*model.BoundEdge{}, Iterations: 0, Converged: true}
	}

	for k := range sharedEdges {
		delete(sharedEdges, k)
	}
	edges := sharedEdges
	edgeOrder := make([]*model.BoundEdge, 0, len(constraints))
	for _, c := range constraints {
		e := model.NewBoundEdge(c.BatchID, c.ID, c.Atom1ID, c.Atom2ID, c.LoDist, c.HiDist)
		edges[e.Key()] = e
		edgeOrder = append(edgeOrder, e)
	}

	atomPairs := make([][2]int64, 0, len(edgeOrder))
	for _, e := range edgeOrder {
		atomPairs = append(atomPairs, [2]int64{e.Atom1ID, e.Atom2ID})
	}

	maxIter := len(edgeOrder) * 3
	converged := false
	iter := 0
	var inverted []*model.BoundEdge

	for iter = 1; iter <= maxIter; iter++ {
		changed := false
		// 遍历所有三角形：对每条边 (i,j) 与共享端点 k 的边，
		// 以 (i,j) 为主角应用三角不等式收紧。每个方向独立处理，
		// 保证三条边都能被传播收紧。
		for _, pair := range atomPairs {
			i, j := pair[0], pair[1]
			eij := edges[keyOf(i, j)]
			for _, other := range edgeOrder {
				if other.ConstraintID == eij.ConstraintID {
					continue
				}
				var k int64
				switch {
				case other.Atom1ID == i && other.Atom2ID != j:
					k = other.Atom2ID
				case other.Atom2ID == i && other.Atom1ID != j:
					k = other.Atom1ID
				case other.Atom1ID == j && other.Atom2ID != i:
					k = other.Atom2ID
				case other.Atom2ID == j && other.Atom1ID != i:
					k = other.Atom1ID
				default:
					continue
				}
				// k 必须同时与 i、j 有边。
				eik, ok1 := edges[keyOf(i, k)]
				ejk, ok2 := edges[keyOf(j, k)]
				if !ok1 || !ok2 {
					continue
				}

				// 三角不等式收紧主角边。
				if tightenTriangle(eij, eik, ejk) {
					changed = true
				}
			}
		}
		if !changed {
			converged = true
			break
		}
	}

	// 收集区间倒置证据。
	for _, e := range edgeOrder {
		if e.Inverted() {
			inverted = append(inverted, e)
		}
	}

	sort.Slice(edgeOrder, func(a, b int) bool { return edgeOrder[a].ConstraintID < edgeOrder[b].ConstraintID })
	return &Result{
		Edges:      edgeOrder,
		Iterations: iter,
		Inverted:   inverted,
		Converged:  converged,
	}
}

// tightenTriangle 对三角 (i,j,k) 应用区间形式的三角不等式。
// 返回是否有任一界被收紧。
func tightenTriangle(eij, eik, ejk *model.BoundEdge) bool {
	changed := false
	// d_ij ≤ d_ik + d_kj → hi_ij = min(hi_ij, hi_ik + hi_kj)
	if t := eik.HiBound + ejk.HiBound; t < eij.HiBound {
		if eij.Tighten(eij.LoBound, t) {
			changed = true
		}
	}
	// d_ij ≥ d_ik - d_kj → lo_ij = max(lo_ij, lo_ik - hi_kj)
	if t := eik.LoBound - ejk.HiBound; t > eij.LoBound {
		if eij.Tighten(t, eij.HiBound) {
			changed = true
		}
	}
	// d_ij ≥ d_kj - d_ik → lo_ij = max(lo_ij, lo_kj - hi_ik)
	if t := ejk.LoBound - eik.HiBound; t > eij.LoBound {
		if eij.Tighten(t, eij.HiBound) {
			changed = true
		}
	}
	return changed
}

// keyOf 返回规范原子对键。
func keyOf(a, b int64) string {
	if a < b {
		return fmt.Sprintf("%d-%d", a, b)
	}
	return fmt.Sprintf("%d-%d", b, a)
}
