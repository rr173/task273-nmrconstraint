package model

import (
	"strings"
	"time"
)

// NoePeak 是一个 NOE（核欧氏效应）交叉峰：连接两个原子并携带强度与观察置信度。
type NoePeak struct {
	ID              int64      `json:"id"`
	BatchID         int64      `json:"batch_id"`
	Name            string     `json:"name"`
	Atom1ID         int64      `json:"atom1_id"`
	Atom2ID         int64      `json:"atom2_id"`
	Intensity       float64    `json:"intensity"`
	Confidence      float64    `json:"confidence"`
	OverlapSuspected bool      `json:"overlap_suspected"`
	Status          PeakStatus `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
}

// NewNoePeak 构造 NOE 峰。
func NewNoePeak(batchID int64, name string, atom1ID, atom2ID int64, intensity, confidence float64) *NoePeak {
	return &NoePeak{
		BatchID:    batchID,
		Name:       name,
		Atom1ID:    atom1ID,
		Atom2ID:    atom2ID,
		Intensity:  intensity,
		Confidence: confidence,
		Status:     PeakRaw,
		CreatedAt:  time.Now().UTC(),
	}
}

// Validate 校验 NOE 峰字段。
func (p *NoePeak) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrInvalidInput("peak name must not be empty")
	}
	if p.Atom1ID == p.Atom2ID {
		return ErrSelfConstraint
	}
	if p.Atom1ID <= 0 || p.Atom2ID <= 0 {
		return ErrInvalidInput("peak atom ids must be positive")
	}
	if p.Intensity <= 0 {
		return ErrIntensityInvalid
	}
	if p.Confidence < 0 || p.Confidence > 1 {
		return ErrConfidenceOutOfRange
	}
	return nil
}

// MarkOverlap 标记峰为重叠可疑。
func (p *NoePeak) MarkOverlap() {
	p.OverlapSuspected = true
	p.Status = PeakOverlap
}

// SetConfidence 更新观察置信度。
func (p *NoePeak) SetConfidence(c float64) error {
	if c < 0 || c > 1 {
		return ErrConfidenceOutOfRange
	}
	p.Confidence = c
	if p.Status == PeakRaw {
		p.Status = PeakValid
	}
	return nil
}

// Exclude 排除该峰（其派生约束将不再参与求解）。
func (p *NoePeak) Exclude() {
	p.Status = PeakExcluded
}
