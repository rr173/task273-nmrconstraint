package store

import (
	"context"
	"database/sql"
	"time"

	"task273-nmrconstraint/internal/model"
)

// BoundStore 提供传播边界的持久化。
type BoundStore struct {
	db *DB
}

// NewBoundStore 构造边界存储。
func NewBoundStore(db *DB) *BoundStore { return &BoundStore{db: db} }

// ReplaceBatch 原子替换一个批次的全部传播边界（求解的幂等落盘）。
// DELETE 与全部 INSERT 在单个事务内执行；ctx 在开始与每条插入前
// 检查，取消时回滚，不得留下半表。ctx 取消返回 context.Cause 对应错误。
func (s *BoundStore) ReplaceBatch(ctx context.Context, batchID int64, edges []*model.BoundEdge) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM bound_edges WHERE batch_id = ?`, batchID); err != nil {
		return err
	}
	for _, e := range edges {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO bound_edges(batch_id, constraint_id, atom1_id, atom2_id, lo_bound, hi_bound, tightened, iteration, updated_at)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			batchID, e.ConstraintID, e.Atom1ID, e.Atom2ID, e.LoBound, e.HiBound, boolToInt(e.Tightened), e.Iteration, e.UpdatedAt.Format(time.RFC3339Nano),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// List 列出批次全部传播边界。
func (s *BoundStore) List(batchID int64) ([]*model.BoundEdge, error) {
	rows, err := s.db.Query(`SELECT id, batch_id, constraint_id, atom1_id, atom2_id, lo_bound, hi_bound, tightened, iteration, updated_at FROM bound_edges WHERE batch_id = ? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.BoundEdge
	for rows.Next() {
		e, err := scanBound(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetByConstraint 按约束 ID 读取传播边界。
func (s *BoundStore) GetByConstraint(batchID, constraintID int64) (*model.BoundEdge, error) {
	row := s.db.QueryRow(`SELECT id, batch_id, constraint_id, atom1_id, atom2_id, lo_bound, hi_bound, tightened, iteration, updated_at FROM bound_edges WHERE batch_id = ? AND constraint_id = ?`, batchID, constraintID)
	return scanBound(row)
}

func scanBound(row scanner) (*model.BoundEdge, error) {
	var e model.BoundEdge
	var tightened int
	var updated string
	if err := row.Scan(&e.ID, &e.BatchID, &e.ConstraintID, &e.Atom1ID, &e.Atom2ID, &e.LoBound, &e.HiBound, &tightened, &e.Iteration, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrConstraintNotFound
		}
		return nil, err
	}
	e.Tightened = tightened != 0
	e.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &e, nil
}
