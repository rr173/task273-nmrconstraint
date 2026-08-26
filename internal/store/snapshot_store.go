package store

import (
	"database/sql"
	"time"

	"task273-nmrconstraint/internal/model"
)

// SnapshotStore 提供诊断快照及其条目的持久化。
type SnapshotStore struct {
	db *DB
}

// NewSnapshotStore 构造快照存储。
func NewSnapshotStore(db *DB) *SnapshotStore { return &SnapshotStore{db: db} }

// Create 创建草稿快照并写入约束状态条目（同一事务）。
func (s *SnapshotStore) Create(snap *model.Snapshot, items []*model.SnapshotItem) (*model.Snapshot, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Commit()
		}
	}()

	res, err := tx.Exec(
		`INSERT INTO snapshots(batch_id, name, status, confidence_version, created_at, published_at) VALUES(?,?,?,?,?,?)`,
		snap.BatchID, snap.Name, string(snap.Status), snap.ConfidenceVersion, snap.CreatedAt.Format(time.RFC3339Nano), nil,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	snap.ID = id
	for _, it := range items {
		if _, err := tx.Exec(
			`INSERT INTO snapshot_items(snapshot_id, constraint_id, atom1_id, atom2_id, lo_dist, hi_dist, lo_bound, hi_bound, status)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			id, it.ConstraintID, it.Atom1ID, it.Atom2ID, it.LoDist, it.HiDist, it.LoBound, it.HiBound, string(it.Status),
		); err != nil {
			return nil, err
		}
		it.ID, it.SnapshotID = 0, id
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return snap, nil
}

// List 列出批次全部快照。
func (s *SnapshotStore) List(batchID int64) ([]*model.Snapshot, error) {
	rows, err := s.db.Query(`SELECT id, batch_id, name, status, confidence_version, created_at, published_at FROM snapshots WHERE batch_id = ? ORDER BY id DESC`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Snapshot
	for rows.Next() {
		snap, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, rows.Err()
}

// Get 按 ID 读取快照。
func (s *SnapshotStore) Get(id int64) (*model.Snapshot, error) {
	row := s.db.QueryRow(`SELECT id, batch_id, name, status, confidence_version, created_at, published_at FROM snapshots WHERE id = ?`, id)
	return scanSnapshot(row)
}

// Publish 发布快照（发布者已校验 draft 状态）。
func (s *SnapshotStore) Publish(id int64) error {
	_, err := s.db.Exec(`UPDATE snapshots SET status = ?, published_at = ? WHERE id = ?`,
		string(model.SnapshotPublished), time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

// UpdateStatus 更新快照状态（如标记 superseded）。
func (s *SnapshotStore) UpdateStatus(id int64, status model.SnapshotStatus) error {
	_, err := s.db.Exec(`UPDATE snapshots SET status = ? WHERE id = ?`, string(status), id)
	return err
}

// Items 列出快照条目。
func (s *SnapshotStore) Items(snapshotID int64) ([]*model.SnapshotItem, error) {
	rows, err := s.db.Query(`SELECT id, snapshot_id, constraint_id, atom1_id, atom2_id, lo_dist, hi_dist, lo_bound, hi_bound, status FROM snapshot_items WHERE snapshot_id = ? ORDER BY id`, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.SnapshotItem
	for rows.Next() {
		var it model.SnapshotItem
		var status string
		if err := rows.Scan(&it.ID, &it.SnapshotID, &it.ConstraintID, &it.Atom1ID, &it.Atom2ID, &it.LoDist, &it.HiDist, &it.LoBound, &it.HiBound, &status); err != nil {
			return nil, err
		}
		it.Status = model.ConstraintStatus(status)
		out = append(out, &it)
	}
	return out, rows.Err()
}

func scanSnapshot(row scanner) (*model.Snapshot, error) {
	var s model.Snapshot
	var status, created string
	var published sql.NullString
	if err := row.Scan(&s.ID, &s.BatchID, &s.Name, &status, &s.ConfidenceVersion, &created, &published); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrSnapshotNotFound
		}
		return nil, err
	}
	s.Status = model.SnapshotStatus(status)
	s.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	if published.Valid {
		t, _ := time.Parse(time.RFC3339Nano, published.String)
		s.PublishedAt = &t
	}
	return &s, nil
}
