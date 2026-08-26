package store

import (
	"database/sql"
	"fmt"
	"time"

	"task273-nmrconstraint/internal/model"
)

// ExemptionStore 提供豁免记录的持久化。
type ExemptionStore struct {
	db *DB
}

// NewExemptionStore 构造豁免存储。
func NewExemptionStore(db *DB) *ExemptionStore { return &ExemptionStore{db: db} }

// Create 插入豁免记录（同批同约束唯一）。
func (s *ExemptionStore) Create(e *model.Exemption) (*model.Exemption, error) {
	res, err := s.db.Exec(
		`INSERT INTO exemptions(batch_id, constraint_id, reason, created_at) VALUES(?,?,?,?)`,
		e.BatchID, e.ConstraintID, e.Reason, e.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("exemption: %v", err)
		}
		return nil, fmt.Errorf("exemption: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	e.ID = id
	return e, nil
}

// List 列出批次全部豁免。
func (s *ExemptionStore) List(batchID int64) ([]*model.Exemption, error) {
	rows, err := s.db.Query(`SELECT id, batch_id, constraint_id, reason, created_at FROM exemptions WHERE batch_id = ? ORDER BY id DESC`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Exemption
	for rows.Next() {
		e, err := scanExemption(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanExemption(row scanner) (*model.Exemption, error) {
	var e model.Exemption
	var created string
	if err := row.Scan(&e.ID, &e.BatchID, &e.ConstraintID, &e.Reason, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrExemptionNotFound
		}
		return nil, err
	}
	e.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return &e, nil
}
