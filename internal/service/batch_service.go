// Package service 编排批次生命周期与求解流程。
// 它把领域包（mapping/constraint/propagate/diagnose/snapshot）
// 与 store 层粘合为可被 HTTP 层直接调用的用例。
package service

import (
	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/store"
)

// BatchService 负责批次的创建、元数据维护与状态机流转。
type BatchService struct {
	batches *store.BatchStore
}

// NewBatchService 构造批次服务。
func NewBatchService(batches *store.BatchStore) *BatchService {
	return &BatchService{batches: batches}
}

// Create 创建新批次。
func (s *BatchService) Create(name, description string) (*model.Batch, error) {
	b := model.NewBatch(name, description)
	if err := b.ValidateName(); err != nil {
		return nil, err
	}
	return s.batches.Create(b)
}

// List 列出全部批次。
func (s *BatchService) List() ([]*model.Batch, error) {
	return s.batches.List()
}

// Get 读取批次。
func (s *BatchService) Get(id int64) (*model.Batch, error) {
	return s.batches.Get(id)
}

// UpdateMeta 更新批次名称与描述。
func (s *BatchService) UpdateMeta(id int64, name, description string) (*model.Batch, error) {
	b, err := s.batches.Get(id)
	if err != nil {
		return nil, err
	}
	if b.Status == model.BatchSealed {
		return nil, model.ErrSealedBatchImmutable
	}
	if err := s.batches.UpdateMeta(id, name, description); err != nil {
		return nil, err
	}
	return s.batches.Get(id)
}

// Advance 推进批次状态机。
func (s *BatchService) Advance(id int64, to model.BatchStatus) (*model.Batch, error) {
	b, err := s.batches.Get(id)
	if err != nil {
		return nil, err
	}
	sealedAt := b.SealedAt
	if err := b.Advance(to); err != nil {
		return nil, err
	}
	if to == model.BatchSealed {
		now := b.SealedAt
		sealedAt = now
	}
	if err := s.batches.UpdateStatus(id, b.Status, sealedAt); err != nil {
		return nil, err
	}
	return s.batches.Get(id)
}
