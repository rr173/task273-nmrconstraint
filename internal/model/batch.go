package model

import (
	"strings"
	"time"
)

// Batch 是一个结构求解批次：原子、NOE 峰与距离约束的归属容器。
type Batch struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Status      BatchStatus `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	SealedAt    *time.Time  `json:"sealed_at,omitempty"`
}

// NewBatch 构造一个处于 receiving 状态的新批次。
func NewBatch(name, description string) *Batch {
	now := time.Now().UTC()
	return &Batch{
		Name:        name,
		Description: description,
		Status:      BatchReceiving,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// CanTransition 校验批次状态机流转是否合法。
func (b *Batch) CanTransition(to BatchStatus) bool {
	for _, next := range ValidTransitions[b.Status] {
		if next == to {
			return true
		}
	}
	return false
}

// Advance 推进批次状态；封存后拒绝任何修改。
func (b *Batch) Advance(to BatchStatus) error {
	if b.Status == BatchSealed {
		return ErrSealedBatchImmutable
	}
	if !b.CanTransition(to) {
		return ErrInvalidTransition
	}
	b.Status = to
	b.UpdatedAt = time.Now().UTC()
	if to == BatchSealed {
		now := time.Now().UTC()
		b.SealedAt = &now
	}
	return nil
}

// ValidateName 校验批次名非空。
func (b *Batch) ValidateName() error {
	if strings.TrimSpace(b.Name) == "" {
		return ErrInvalidInput("batch name must not be empty")
	}
	return nil
}
