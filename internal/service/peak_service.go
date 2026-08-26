package service

import (
	"sort"

	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/store"
)

// PeakService 负责 NOE 峰的导入、置信度维护与状态标记。
type PeakService struct {
	peaks   *store.PeakStore
	batches *store.BatchStore
}

// NewPeakService 构造峰服务。
func NewPeakService(peaks *store.PeakStore, batches *store.BatchStore) *PeakService {
	return &PeakService{peaks: peaks, batches: batches}
}

// Import 批量导入 NOE 峰（批次须 receiving）。
func (s *PeakService) Import(batchID int64, peaks []*model.NoePeak) ([]*model.NoePeak, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status != model.BatchReceiving {
		return nil, model.ErrInvalidInput("peaks can only be imported while batch is receiving")
	}
	for _, p := range peaks {
		if err := p.Validate(); err != nil {
			return nil, err
		}
	}
	return s.peaks.Create(batchID, peaks)
}

// List 列出批次全部峰。
func (s *PeakService) List(batchID int64) ([]*model.NoePeak, error) {
	return s.peaks.List(batchID)
}

// SetConfidence 更新峰的观察置信度。
func (s *PeakService) SetConfidence(id int64, confidence float64) (*model.NoePeak, error) {
	p, err := s.peaks.Get(id)
	if err != nil {
		return nil, err
	}
	if err := p.SetConfidence(confidence); err != nil {
		return nil, err
	}
	if err := s.peaks.UpdateConfidence(id, confidence); err != nil {
		return nil, err
	}
	return s.peaks.Get(id)
}

// MarkOverlap 标记峰为重叠可疑。
func (s *PeakService) MarkOverlap(id int64) (*model.NoePeak, error) {
	p, err := s.peaks.Get(id)
	if err != nil {
		return nil, err
	}
	p.MarkOverlap()
	if err := s.peaks.UpdateStatus(id, p.Status, true); err != nil {
		return nil, err
	}
	return s.peaks.Get(id)
}

// Exclude 排除峰。
func (s *PeakService) Exclude(id int64) (*model.NoePeak, error) {
	p, err := s.peaks.Get(id)
	if err != nil {
		return nil, err
	}
	p.Exclude()
	if err := s.peaks.UpdateStatus(id, p.Status, p.OverlapSuspected); err != nil {
		return nil, err
	}
	return s.peaks.Get(id)
}

// ConfidenceVersion 计算批次的观察置信度版本：
// 有效峰计数 + 10×重叠可疑峰计数，供快照记录。
func (s *PeakService) ConfidenceVersion(batchID int64) (int, error) {
	peaks, err := s.peaks.List(batchID)
	if err != nil {
		return 0, err
	}
	valid := 0
	overlap := 0
	for _, p := range peaks {
		if p.Status == model.PeakExcluded {
			continue
		}
		valid++
		if p.OverlapSuspected {
			overlap++
		}
	}
	return valid + overlap*10, nil
}

// SortPeaks 按 ID 升序排序（响应稳定性）。
func SortPeaks(peaks []*model.NoePeak) {
	sort.Slice(peaks, func(i, j int) bool { return peaks[i].ID < peaks[j].ID })
}
