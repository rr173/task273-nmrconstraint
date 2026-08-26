// Package snapshot 负责诊断快照的创建与发布。
// 快照在发布时刻固化约束状态与传播边界，之后不可修改；
// 新一次发布会使旧快照进入 superseded（替代）状态。
package snapshot

import (
	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/store"
)

// Service 是快照发布领域服务。
type Service struct {
	batches  *store.BatchStore
	cons     *store.ConstraintStore
	bounds   *store.BoundStore
	snaps    *store.SnapshotStore
	exemps   *store.ExemptionStore
	peakConf func(batchID int64) (int, error) // 注入：读取当前置信度版本
}

// NewService 构造快照服务。
func NewService(batches *store.BatchStore, cons *store.ConstraintStore, bounds *store.BoundStore, snaps *store.SnapshotStore, exemps *store.ExemptionStore, peakConf func(batchID int64) (int, error)) *Service {
	return &Service{
		batches:  batches,
		cons:     cons,
		bounds:   bounds,
		snaps:    snaps,
		exemps:   exemps,
		peakConf: peakConf,
	}
}

// CreateDraft 为批次创建草稿快照，固化当前全部约束的传播边界。
func (s *Service) CreateDraft(batchID int64, name string) (*model.Snapshot, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status == model.BatchSealed {
		return nil, model.ErrSealedBatchImmutable
	}
	constraints, err := s.cons.List(batchID)
	if err != nil {
		return nil, err
	}
	edges, err := s.bounds.List(batchID)
	if err != nil {
		return nil, err
	}
	edgeByConstraint := make(map[int64]*model.BoundEdge, len(edges))
	for _, e := range edges {
		edgeByConstraint[e.ConstraintID] = e
	}
	version := 1
	if s.peakConf != nil {
		if v, err := s.peakConf(batchID); err == nil {
			version = v
		}
	}
	snap := model.NewSnapshot(batchID, name, version)
	items := make([]*model.SnapshotItem, 0, len(constraints))
	for _, c := range constraints {
		items = append(items, model.NewSnapshotItem(snap.ID, c, edgeByConstraint[c.ID]))
	}
	return s.snaps.Create(snap, items)
}

// List 列出批次全部快照。
func (s *Service) List(batchID int64) ([]*model.Snapshot, error) {
	return s.snaps.List(batchID)
}

// Get 读取快照详情（含条目）。
func (s *Service) Get(id int64) (*model.Snapshot, []*model.SnapshotItem, error) {
	snap, err := s.snaps.Get(id)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.snaps.Items(id)
	if err != nil {
		return nil, nil, err
	}
	return snap, items, nil
}

// Publish 发布快照：批次须处于可发布状态，且发布后旧快照转 superseded。
func (s *Service) Publish(batchID, snapshotID int64) (*model.Snapshot, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status != model.BatchReleasable {
		return nil, model.ErrPublishNotReleasable
	}
	snap, err := s.snaps.Get(snapshotID)
	if err != nil {
		return nil, err
	}
	if snap.BatchID != batchID {
		return nil, model.ErrSnapshotNotFound
	}
	if err := snap.Publish(); err != nil {
		return nil, err
	}
	if err := s.snaps.Publish(snapshotID); err != nil {
		return nil, err
	}
	live, err := s.bounds.List(batchID)
	if err != nil {
		return nil, err
	}
	_ = live
	// 批次推进到 published。
	_ = b.Advance(model.BatchPublished)
	if err := s.batches.UpdateStatus(batchID, b.Status, nil); err != nil {
		return nil, err
	}
	// 将同一批次其它已发布快照置为 superseded。
	if err := s.supersedeOthers(batchID, snapshotID); err != nil {
		return nil, err
	}
	return snap, nil
}

// supersedeOthers 将同批其它已发布快照标记为替代。
func (s *Service) supersedeOthers(batchID, exceptID int64) error {
	all, err := s.snaps.List(batchID)
	if err != nil {
		return err
	}
	for _, sn := range all {
		if sn.ID == exceptID || sn.Status != model.SnapshotPublished {
			continue
		}
		if err := s.snaps.UpdateStatus(sn.ID, model.SnapshotSuperseded); err != nil {
			return err
		}
	}
	return nil
}
