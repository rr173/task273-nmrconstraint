package model

import (
	"fmt"
	"time"
)

// BoundEdge 是距离图上一条边在传播后的距离界。
// 三角不等式传播会逐步收紧 LoBound/HiBound，直到收敛或区间倒置。
type BoundEdge struct {
	ID         int64     `json:"id"`
	BatchID    int64     `json:"batch_id"`
	ConstraintID int64   `json:"constraint_id"`
	Atom1ID    int64     `json:"atom1_id"`
	Atom2ID    int64     `json:"atom2_id"`
	LoBound    float64   `json:"lo_bound"`
	HiBound    float64   `json:"hi_bound"`
	Tightened  bool      `json:"tightened"`
	Iteration  int       `json:"iteration"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewBoundEdge 以约束区间初始化一条边。
func NewBoundEdge(batchID, constraintID, atom1ID, atom2ID int64, lo, hi float64) *BoundEdge {
	return &BoundEdge{
		BatchID:      batchID,
		ConstraintID: constraintID,
		Atom1ID:      atom1ID,
		Atom2ID:      atom2ID,
		LoBound:      lo,
		HiBound:      hi,
		Tightened:    false,
		Iteration:    0,
		UpdatedAt:    time.Now().UTC(),
	}
}

// Inverted 判断区间是否倒置（传播导致 lo > hi，即该约束不可满足）。
func (e *BoundEdge) Inverted() bool {
	return e.LoBound > e.HiBound+1e-9
}

// Key 返回规范化原子对键（小 id 在前），用于三角闭合。
func (e *BoundEdge) Key() string {
	if e.Atom1ID < e.Atom2ID {
		return fmt.Sprintf("%d-%d", e.Atom1ID, e.Atom2ID)
	}
	return fmt.Sprintf("%d-%d", e.Atom2ID, e.Atom1ID)
}

// Tighten 用新的上下界收紧边，返回是否发生变化。
func (e *BoundEdge) Tighten(lo, hi float64) bool {
	oldLo, oldHi := e.LoBound, e.HiBound
	if lo > e.LoBound {
		e.LoBound = lo
	}
	if hi < e.HiBound {
		e.HiBound = hi
	}
	changed := oldLo != e.LoBound || oldHi != e.HiBound
	if changed {
		e.Tightened = true
		e.UpdatedAt = time.Now().UTC()
	}
	return changed
}
