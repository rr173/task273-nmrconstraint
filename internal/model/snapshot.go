package model

import "time"

// Snapshot 是一次诊断结果的不可变快照：记录发布时的约束状态与传播边界。
type Snapshot struct {
	ID         int64          `json:"id"`
	BatchID    int64          `json:"batch_id"`
	Name       string         `json:"name"`
	Status     SnapshotStatus `json:"status"`
	ConfidenceVersion int     `json:"confidence_version"`
	CreatedAt  time.Time      `json:"created_at"`
	PublishedAt *time.Time    `json:"published_at,omitempty"`
}

// NewSnapshot 构造草稿快照。
func NewSnapshot(batchID int64, name string, confidenceVersion int) *Snapshot {
	return &Snapshot{
		BatchID:           batchID,
		Name:              name,
		Status:            SnapshotDraft,
		ConfidenceVersion: confidenceVersion,
		CreatedAt:         time.Now().UTC(),
	}
}

// Publish 发布快照。
func (s *Snapshot) Publish() error {
	if s.Status != SnapshotDraft {
		return ErrSnapshotNotDraft
	}
	s.Status = SnapshotPublished
	now := time.Now().UTC()
	s.PublishedAt = &now
	return nil
}

// SnapshotItem 是快照中某条约束在发布时刻的状态拷贝。
type SnapshotItem struct {
	ID           int64            `json:"id"`
	SnapshotID   int64            `json:"snapshot_id"`
	ConstraintID int64            `json:"constraint_id"`
	Atom1ID      int64            `json:"atom1_id"`
	Atom2ID      int64            `json:"atom2_id"`
	LoDist       float64          `json:"lo_dist"`
	HiDist       float64          `json:"hi_dist"`
	LoBound      float64          `json:"lo_bound"`
	HiBound      float64          `json:"hi_bound"`
	Status       ConstraintStatus `json:"status"`
}

// NewSnapshotItem 构造快照条目。
func NewSnapshotItem(snapshotID int64, c *Constraint, e *BoundEdge) *SnapshotItem {
	loBound, hiBound := e.LoBound, e.HiBound
	return &SnapshotItem{
		SnapshotID:   snapshotID,
		ConstraintID: c.ID,
		Atom1ID:      c.Atom1ID,
		Atom2ID:      c.Atom2ID,
		LoDist:       c.LoDist,
		HiDist:       c.HiDist,
		LoBound:      loBound,
		HiBound:      hiBound,
		Status:       c.Status,
	}
}
