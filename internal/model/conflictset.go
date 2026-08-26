package model

import "time"

// ConflictSet 是一组共同导致不可满足的约束集合。
// 经最小化（MUC 求解）后得到最小冲突集。
type ConflictSet struct {
	ID         int64            `json:"id"`
	BatchID    int64            `json:"batch_id"`
	Kind       ConflictKind     `json:"kind"`
	Status     ConflictSetStatus `json:"status"`
	Minimized  bool             `json:"minimized"`
	MemberCount int             `json:"member_count"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// NewConflictSet 构造候选冲突集。
func NewConflictSet(batchID int64, kind ConflictKind) *ConflictSet {
	now := time.Now().UTC()
	return &ConflictSet{
		BatchID:   batchID,
		Kind:      kind,
		Status:    ConflictSetCandidate,
		Minimized: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// MarkMinimized 标记最小化完成。
func (c *ConflictSet) MarkMinimized() {
	c.Minimized = true
	c.Status = ConflictSetConfirmed
	c.UpdatedAt = time.Now().UTC()
}

// Exempt 豁免冲突集（其成员约束获得豁免记录）。
func (c *ConflictSet) Exempt() {
	c.Status = ConflictSetExempted
	c.UpdatedAt = time.Now().UTC()
}

// ConflictMember 是冲突集的成员关系。
type ConflictMember struct {
	ID           int64  `json:"id"`
	ConflictSetID int64 `json:"conflict_set_id"`
	ConstraintID int64  `json:"constraint_id"`
	// Removed 表示该约束在最小化过程中被判定为可移除（非必要成员）。
	Removed bool `json:"removed"`
}

// NewConflictMember 构造成员。
func NewConflictMember(setID, constraintID int64) *ConflictMember {
	return &ConflictMember{
		ConflictSetID: setID,
		ConstraintID:  constraintID,
		Removed:       false,
	}
}
