package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"task273-nmrconstraint/internal/model"
)

// ConstraintStore 提供距离约束的持久化。
type ConstraintStore struct {
	db *DB
}

// NewConstraintStore 构造约束存储。
func NewConstraintStore(db *DB) *ConstraintStore { return &ConstraintStore{db: db} }

// Create 批量插入距离约束。
func (s *ConstraintStore) Create(batchID int64, cons []*model.Constraint) ([]*model.Constraint, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	for _, c := range cons {
		res, err := tx.Exec(
			`INSERT INTO constraints(batch_id, peak_id, atom1_id, atom2_id, lo_dist, hi_dist, status, created_at)
			 VALUES(?,?,?,?,?,?,?,?)`,
			batchID, c.PeakID, c.Atom1ID, c.Atom2ID, c.LoDist, c.HiDist, string(c.Status), c.CreatedAt.Format(time.RFC3339Nano),
		)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, fmt.Errorf("insert constraint: %v", err)
			}
			return nil, fmt.Errorf("insert constraint: %v", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		c.ID = id
		c.BatchID = batchID
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return cons, nil
}

// List 列出批次全部约束。
func (s *ConstraintStore) List(batchID int64) ([]*model.Constraint, error) {
	rows, err := s.db.Query(`SELECT id, batch_id, peak_id, atom1_id, atom2_id, lo_dist, hi_dist, status, created_at FROM constraints WHERE batch_id = ? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Constraint
	for rows.Next() {
		c, err := scanConstraint(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Get 按 ID 读取约束。
func (s *ConstraintStore) Get(id int64) (*model.Constraint, error) {
	row := s.db.QueryRow(`SELECT id, batch_id, peak_id, atom1_id, atom2_id, lo_dist, hi_dist, status, created_at FROM constraints WHERE id = ?`, id)
	return scanConstraint(row)
}

// GetInBatch 按批次与 ID 读取约束。
func (s *ConstraintStore) GetInBatch(batchID, id int64) (*model.Constraint, error) {
	row := s.db.QueryRow(`SELECT id, batch_id, peak_id, atom1_id, atom2_id, lo_dist, hi_dist, status, created_at FROM constraints WHERE batch_id = ? AND id = ?`, batchID, id)
	return scanConstraint(row)
}

// ByIDs 按 ID 集合读取约束（缺一即错）。
func (s *ConstraintStore) ByIDs(batchID int64, ids []int64) ([]*model.Constraint, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ph := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+1)
	args = append(args, batchID)
	for _, id := range ids {
		args = append(args, id)
	}
	q := `SELECT id, batch_id, peak_id, atom1_id, atom2_id, lo_dist, hi_dist, status, created_at FROM constraints WHERE batch_id = ? AND id IN (` + ph + `) ORDER BY id`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Constraint
	for rows.Next() {
		c, err := scanConstraint(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) != len(ids) {
		return nil, model.ErrConstraintNotFound
	}
	return out, nil
}

// UpdateStatus 更新约束状态。
func (s *ConstraintStore) UpdateStatus(id int64, status model.ConstraintStatus) error {
	_, err := s.db.Exec(`UPDATE constraints SET status = ? WHERE id = ?`, string(status), id)
	return err
}

// ResetSolveStatus 将冲突/有效状态重置为 raw，用于重新求解。
func (s *ConstraintStore) ResetSolveStatus(batchID int64) error {
	_, err := s.db.Exec(`UPDATE constraints SET status = ? WHERE batch_id = ? AND status IN (?, ?)`,
		string(model.ConstraintRaw), batchID, string(model.ConstraintConflicted), string(model.ConstraintValid))
	return err
}

// ActiveCount 统计批次内参与求解的约束数（排除 excluded）。
func (s *ConstraintStore) ActiveCount(batchID int64) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM constraints WHERE batch_id = ? AND status != ?`, batchID, string(model.ConstraintExcluded)).Scan(&n)
	return n, err
}

func scanConstraint(row scanner) (*model.Constraint, error) {
	var c model.Constraint
	var status, created string
	if err := row.Scan(&c.ID, &c.BatchID, &c.PeakID, &c.Atom1ID, &c.Atom2ID, &c.LoDist, &c.HiDist, &status, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrConstraintNotFound
		}
		return nil, err
	}
	c.Status = model.ConstraintStatus(status)
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return &c, nil
}
