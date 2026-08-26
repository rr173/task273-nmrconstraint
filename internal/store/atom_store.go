package store

import (
	"database/sql"
	"time"

	"task273-nmrconstraint/internal/model"
)

// AtomStore 提供原子的持久化。
type AtomStore struct {
	db *DB
}

// NewAtomStore 构造原子存储。
func NewAtomStore(db *DB) *AtomStore { return &AtomStore{db: db} }

// Create 批量插入原子；任一名字重复则整体失败并回滚。
func (s *AtomStore) Create(batchID int64, atoms []*model.Atom) ([]*model.Atom, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit() }()

	for _, a := range atoms {
		res, err := tx.Exec(
			`INSERT INTO atoms(batch_id, name, residue, element, status, created_at) VALUES(?,?,?,?,?,?)`,
			batchID, a.Name, a.Residue, a.Element, string(a.Status), a.CreatedAt.Format(time.RFC3339Nano),
		)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, model.ErrDuplicateAtomName
			}
			return nil, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		a.ID = id
		a.BatchID = batchID
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return atoms, nil
}

// List 列出批次全部原子。
func (s *AtomStore) List(batchID int64) ([]*model.Atom, error) {
	rows, err := s.db.Query(`SELECT id, batch_id, name, residue, element, status, created_at FROM atoms WHERE batch_id = ? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Atom
	for rows.Next() {
		a, err := scanAtom(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Get 按 ID 读取原子。
func (s *AtomStore) Get(id int64) (*model.Atom, error) {
	row := s.db.QueryRow(`SELECT id, batch_id, name, residue, element, status, created_at FROM atoms WHERE id = ?`, id)
	return scanAtom(row)
}

// UpdateStatus 更新原子状态。
func (s *AtomStore) UpdateStatus(id int64, status model.AtomStatus) error {
	_, err := s.db.Exec(`UPDATE atoms SET status = ? WHERE id = ?`, string(status), id)
	return err
}

// Exists 判断批次内原子是否存在且未排除。
func (s *AtomStore) Exists(batchID, id int64) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM atoms WHERE batch_id = ? AND id = ? AND status != ?`, batchID, id, string(model.AtomExcluded)).Scan(&n)
	return n > 0, err
}

func scanAtom(row scanner) (*model.Atom, error) {
	var a model.Atom
	var status, created string
	if err := row.Scan(&a.ID, &a.BatchID, &a.Name, &a.Residue, &a.Element, &status, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrAtomNotFound
		}
		return nil, err
	}
	a.Status = model.AtomStatus(status)
	a.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return &a, nil
}
