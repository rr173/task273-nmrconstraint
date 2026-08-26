// Package mapping 维护结构批次中的原子映射。
// 原子是距离约束的端点；本包负责导入、激活与排除，
// 并保证批次内原子名唯一、端点引用有效。
package mapping

import (
	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/store"
)

// Service 是原子映射的领域服务。
type Service struct {
	atoms *store.AtomStore
	batches *store.BatchStore
}

// NewService 构造原子映射服务。
func NewService(atoms *store.AtomStore, batches *store.BatchStore) *Service {
	return &Service{atoms: atoms, batches: batches}
}

// Import 批量导入原子；批次必须处于 receiving 状态。
func (s *Service) Import(batchID int64, atoms []*model.Atom) ([]*model.Atom, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status != model.BatchReceiving {
		return nil, model.ErrInvalidInput("atoms can only be imported while batch is receiving")
	}
	for _, a := range atoms {
		if err := a.Validate(); err != nil {
			return nil, err
		}
	}
	return s.atoms.Create(batchID, atoms)
}

// List 列出批次全部原子。
func (s *Service) List(batchID int64) ([]*model.Atom, error) {
	return s.atoms.List(batchID)
}

// Get 读取单个原子。
func (s *Service) Get(id int64) (*model.Atom, error) {
	return s.atoms.Get(id)
}

// Exclude 排除原子：其端点上的约束将不再参与后续求解。
func (s *Service) Exclude(id int64) (*model.Atom, error) {
	a, err := s.atoms.Get(id)
	if err != nil {
		return nil, err
	}
	a.Exclude()
	if err := s.atoms.UpdateStatus(id, a.Status); err != nil {
		return nil, err
	}
	return a, nil
}

// Activate 重新激活原子。
func (s *Service) Activate(id int64) (*model.Atom, error) {
	a, err := s.atoms.Get(id)
	if err != nil {
		return nil, err
	}
	a.Activate()
	if err := s.atoms.UpdateStatus(id, a.Status); err != nil {
		return nil, err
	}
	return a, nil
}
