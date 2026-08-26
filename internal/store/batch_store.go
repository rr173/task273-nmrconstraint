package store

import (
	"database/sql"
	"time"

	"task273-nmrconstraint/internal/model"
)

// BatchStore 提供批次的持久化。
type BatchStore struct {
	db *DB
}

// NewBatchStore 构造批次存储。
func NewBatchStore(db *DB) *BatchStore { return &BatchStore{db: db} }

// Create 插入新批次并返回带 ID 的实体。
func (s *BatchStore) Create(b *model.Batch) (*model.Batch, error) {
	res, err := s.db.Exec(
		`INSERT INTO batches(name, description, status, created_at, updated_at, sealed_at) VALUES(?,?,?,?,?,?)`,
		b.Name, b.Description, string(b.Status), b.CreatedAt.Format(time.RFC3339Nano), b.UpdatedAt.Format(time.RFC3339Nano), nil,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	b.ID = id
	return b, nil
}

// Get 按 ID 读取批次。
func (s *BatchStore) Get(id int64) (*model.Batch, error) {
	row := s.db.QueryRow(`SELECT id, name, description, status, created_at, updated_at, sealed_at FROM batches WHERE id = ?`, id)
	return scanBatch(row)
}

// List 列出全部批次，按创建时间倒序。
func (s *BatchStore) List() ([]*model.Batch, error) {
	rows, err := s.db.Query(`SELECT id, name, description, status, created_at, updated_at, sealed_at FROM batches ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Batch
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// UpdateStatus 更新批次状态（含时间戳）。
func (s *BatchStore) UpdateStatus(id int64, status model.BatchStatus, sealedAt *time.Time) error {
	_, err := s.db.Exec(
		`UPDATE batches SET status = ?, updated_at = ?, sealed_at = ? WHERE id = ?`,
		string(status), time.Now().UTC().Format(time.RFC3339Nano), sealedAt, id,
	)
	return err
}

// UpdateMeta 更新批次名称与描述。
func (s *BatchStore) UpdateMeta(id int64, name, description string) error {
	_, err := s.db.Exec(
		`UPDATE batches SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		name, description, time.Now().UTC().Format(time.RFC3339Nano), id,
	)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanBatch(row scanner) (*model.Batch, error) {
	var b model.Batch
	var status string
	var created, updated string
	var sealed sql.NullString
	if err := row.Scan(&b.ID, &b.Name, &b.Description, &status, &created, &updated, &sealed); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrBatchNotFound
		}
		return nil, err
	}
	b.Status = model.BatchStatus(status)
	b.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	b.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	if sealed.Valid {
		t, _ := time.Parse(time.RFC3339Nano, sealed.String)
		b.SealedAt = &t
	}
	return &b, nil
}
