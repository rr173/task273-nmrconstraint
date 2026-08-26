// Package constraint 负责 NOE 峰到距离约束的归属与校验。
// 每条约束是一个原子对上的距离区间 [Lo, Hi]；本包校验区间合法性、
// 端点存在性、峰与约束的原子对一致性，以及同批同原子对的唯一性。
package constraint

import (
	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/store"
)

// Service 是约束归属的领域服务。
type Service struct {
	batches *store.BatchStore
	atoms   *store.AtomStore
	peaks   *store.PeakStore
	cons    *store.ConstraintStore
}

// NewService 构造约束服务。
func NewService(batches *store.BatchStore, atoms *store.AtomStore, peaks *store.PeakStore, cons *store.ConstraintStore) *Service {
	return &Service{
		batches: batches,
		atoms:   atoms,
		peaks:   peaks,
		cons:    cons,
	}
}

// CreateFromPeaks 基于 NOE 峰与距离区间批量创建距离约束。
// 峰与约束的原子对必须一致（允许交换两端）。
func (s *Service) CreateFromPeaks(batchID int64, inputs []*model.Constraint) ([]*model.Constraint, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status != model.BatchReceiving {
		return nil, model.ErrInvalidInput("constraints can only be added while batch is receiving")
	}
	// 预加载峰与原子引用校验。
	peakIDs := make(map[int64]bool, len(inputs))
	for _, c := range inputs {
		if err := c.Validate(); err != nil {
			return nil, err
		}
		peakIDs[c.PeakID] = true
	}
	peakIDsList := make([]int64, 0, len(peakIDs))
	for id := range peakIDs {
		peakIDsList = append(peakIDsList, id)
	}
	peaks, err := s.peaks.ByIDs(batchID, peakIDsList)
	if err != nil {
		return nil, err
	}
	peakMap := make(map[int64]*model.NoePeak, len(peaks))
	for _, p := range peaks {
		peakMap[p.ID] = p
	}
	for _, c := range inputs {
		p, ok := peakMap[c.PeakID]
		if !ok {
			return nil, model.ErrPeakNotFoundRef
		}
		if !samePair(p.Atom1ID, p.Atom2ID, c.Atom1ID, c.Atom2ID) {
			return nil, model.ErrPeakPairMismatch
		}
		ok1, err1 := s.atoms.Exists(batchID, c.Atom1ID)
		if err1 != nil {
			return nil, err1
		}
		ok2, err2 := s.atoms.Exists(batchID, c.Atom2ID)
		if err2 != nil {
			return nil, err2
		}
		if !ok1 || !ok2 {
			return nil, model.ErrAtomNotFoundRef
		}
	}
	return s.cons.Create(batchID, inputs)
}

// List 列出批次全部约束。
func (s *Service) List(batchID int64) ([]*model.Constraint, error) {
	return s.cons.List(batchID)
}

// Get 读取单个约束。
func (s *Service) Get(id int64) (*model.Constraint, error) {
	return s.cons.Get(id)
}

// Exclude 排除约束（不再参与求解）。
func (s *Service) Exclude(id int64) (*model.Constraint, error) {
	c, err := s.cons.Get(id)
	if err != nil {
		return nil, err
	}
	c.Exclude()
	if err := s.cons.UpdateStatus(id, c.Status); err != nil {
		return nil, err
	}
	return c, nil
}

// Restore 恢复被排除的约束。
func (s *Service) Restore(id int64) (*model.Constraint, error) {
	c, err := s.cons.Get(id)
	if err != nil {
		return nil, err
	}
	c.Restore()
	if err := s.cons.UpdateStatus(id, c.Status); err != nil {
		return nil, err
	}
	return c, nil
}

// samePair 判断两个无序原子对是否相同。
func samePair(a1, a2, b1, b2 int64) bool {
	return (a1 == b1 && a2 == b2) || (a1 == b2 && a2 == b1)
}
