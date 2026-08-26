package model

import (
	"time"
)

// Constraint 是一条原子间距离约束：[LoDist, HiDist] 区间。
// 由 NOE 峰归属而来，是距离图与三角不等式传播的基本单元。
type Constraint struct {
	ID        int64            `json:"id"`
	BatchID   int64            `json:"batch_id"`
	PeakID    int64            `json:"peak_id"`
	Atom1ID   int64            `json:"atom1_id"`
	Atom2ID   int64            `json:"atom2_id"`
	LoDist    float64          `json:"lo_dist"`
	HiDist    float64          `json:"hi_dist"`
	Status    ConstraintStatus `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
}

// NewConstraint 构造距离约束，初始为 raw。
func NewConstraint(batchID, peakID, atom1ID, atom2ID int64, lo, hi float64) *Constraint {
	return &Constraint{
		BatchID:   batchID,
		PeakID:    peakID,
		Atom1ID:   atom1ID,
		Atom2ID:   atom2ID,
		LoDist:    lo,
		HiDist:    hi,
		Status:    ConstraintRaw,
		CreatedAt: time.Now().UTC(),
	}
}

// Validate 校验距离区间与端点。
func (c *Constraint) Validate() error {
	if c.Atom1ID == c.Atom2ID {
		return ErrSelfConstraint
	}
	if c.Atom1ID <= 0 || c.Atom2ID <= 0 {
		return ErrInvalidInput("constraint atom ids must be positive")
	}
	if c.PeakID <= 0 {
		return ErrInvalidInput("constraint peak id must be positive")
	}
	if c.LoDist < 0 || c.HiDist < c.LoDist {
		return ErrInvalidInterval
	}
	return nil
}

// MarkConflicted 将约束标记为冲突。
func (c *Constraint) MarkConflicted() {
	c.Status = ConstraintConflicted
}

// MarkValid 将约束标记为有效。
func (c *Constraint) MarkValid() {
	c.Status = ConstraintValid
}

// Exclude 排除约束。
func (c *Constraint) Exclude() {
	c.Status = ConstraintExcluded
}

// Restore 将约束从 excluded 恢复为 raw，允许重新参与求解。
func (c *Constraint) Restore() {
	c.Status = ConstraintRaw
}

// Active 判断约束是否应参与求解（排除态不参与）。
func (c *Constraint) Active() bool {
	return c.Status != ConstraintExcluded
}
