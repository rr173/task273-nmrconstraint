package model

import "time"

// Exemption 记录研究者对某条冲突约束的豁免决定。
// 豁免后该约束被排除出后续求解，批次可能转为可发布。
type Exemption struct {
	ID           int64     `json:"id"`
	BatchID      int64     `json:"batch_id"`
	ConstraintID int64     `json:"constraint_id"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
}

// NewExemption 构造豁免记录。
func NewExemption(batchID, constraintID int64, reason string) *Exemption {
	return &Exemption{
		BatchID:      batchID,
		ConstraintID: constraintID,
		Reason:       reason,
		CreatedAt:    time.Now().UTC(),
	}
}
