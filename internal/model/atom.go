package model

import (
	"strings"
	"time"
)

// Atom 是结构中的一个原子映射，作为距离约束的端点。
type Atom struct {
	ID        int64      `json:"id"`
	BatchID   int64      `json:"batch_id"`
	Name      string     `json:"name"`
	Residue   string     `json:"residue"`
	Element   string     `json:"element"`
	Status    AtomStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

// NewAtom 构造原子映射。
func NewAtom(batchID int64, name, residue, element string) *Atom {
	return &Atom{
		BatchID:   batchID,
		Name:      name,
		Residue:   residue,
		Element:   element,
		Status:    AtomRaw,
		CreatedAt: time.Now().UTC(),
	}
}

// Validate 校验原子字段合法性。
func (a *Atom) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return ErrInvalidInput("atom name must not be empty")
	}
	if strings.TrimSpace(a.Element) == "" {
		return ErrInvalidInput("atom element must not be empty")
	}
	return nil
}

// Activate 将原子置为 active。
func (a *Atom) Activate() {
	a.Status = AtomActive
}

// Exclude 排除原子（其上的约束将不参与求解）。
func (a *Atom) Exclude() {
	a.Status = AtomExcluded
}
