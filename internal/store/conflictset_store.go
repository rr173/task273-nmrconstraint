package store

import (
	"database/sql"
	"time"

	"task273-nmrconstraint/internal/model"
)

// ConflictSetStore 提供冲突集及其成员的持久化。
type ConflictSetStore struct {
	db *DB
}

// NewConflictSetStore 构造冲突集存储。
func NewConflictSetStore(db *DB) *ConflictSetStore { return &ConflictSetStore{db: db} }

// Create 插入冲突集并批量写入成员（同一事务）。
func (s *ConflictSetStore) Create(cs *model.ConflictSet, members []*model.ConflictMember) (*model.ConflictSet, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(
		`INSERT INTO conflict_sets(batch_id, kind, status, minimized, member_count, created_at, updated_at) VALUES(?,?,?,?,?,?,?)`,
		cs.BatchID, string(cs.Kind), string(cs.Status), boolToInt(cs.Minimized), len(members), cs.CreatedAt.Format(time.RFC3339Nano), cs.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	cs.ID = id
	cs.MemberCount = len(members)
	for _, m := range members {
		if _, err := tx.Exec(
			`INSERT INTO conflict_members(conflict_set_id, constraint_id, removed) VALUES(?,?,?)`,
			id, m.ConstraintID, boolToInt(m.Removed),
		); err != nil {
			return nil, err
		}
		m.ConflictSetID = id
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return cs, nil
}

// List 列出批次全部冲突集。
func (s *ConflictSetStore) List(batchID int64) ([]*model.ConflictSet, error) {
	rows, err := s.db.Query(`SELECT id, batch_id, kind, status, minimized, member_count, created_at, updated_at FROM conflict_sets WHERE batch_id = ? ORDER BY id DESC`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ConflictSet
	for rows.Next() {
		cs, err := scanConflictSet(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cs)
	}
	return out, rows.Err()
}

// Get 按 ID 读取冲突集。
func (s *ConflictSetStore) Get(id int64) (*model.ConflictSet, error) {
	row := s.db.QueryRow(`SELECT id, batch_id, kind, status, minimized, member_count, created_at, updated_at FROM conflict_sets WHERE id = ?`, id)
	return scanConflictSet(row)
}

// Update 更新冲突集状态与最小化标记。
func (s *ConflictSetStore) Update(cs *model.ConflictSet) error {
	_, err := s.db.Exec(
		`UPDATE conflict_sets SET status = ?, minimized = ?, member_count = ?, updated_at = ? WHERE id = ?`,
		string(cs.Status), boolToInt(cs.Minimized), cs.MemberCount, cs.UpdatedAt.Format(time.RFC3339Nano), cs.ID,
	)
	return err
}

// Members 列出冲突集成员。
func (s *ConflictSetStore) Members(setID int64) ([]*model.ConflictMember, error) {
	rows, err := s.db.Query(`SELECT id, conflict_set_id, constraint_id, removed FROM conflict_members WHERE conflict_set_id = ? ORDER BY id`, setID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ConflictMember
	for rows.Next() {
		var m model.ConflictMember
		var removed int
		if err := rows.Scan(&m.ID, &m.ConflictSetID, &m.ConstraintID, &removed); err != nil {
			return nil, err
		}
		m.Removed = removed != 0
		out = append(out, &m)
	}
	return out, rows.Err()
}

// UpdateMemberRemoved 标记成员在最小化中被移除。
func (s *ConflictSetStore) UpdateMemberRemoved(setID, constraintID int64, removed bool) error {
	_, err := s.db.Exec(`UPDATE conflict_members SET removed = ? WHERE conflict_set_id = ? AND constraint_id = ?`, boolToInt(removed), setID, constraintID)
	return err
}

// HasActiveConflicts 判断批次内是否存在未豁免的确认冲突集。
func (s *ConflictSetStore) HasActiveConflicts(batchID int64) (bool, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(1) FROM conflict_sets WHERE batch_id = ? AND status IN (?, ?)`,
		batchID, string(model.ConflictSetCandidate), string(model.ConflictSetConfirmed),
	).Scan(&n)
	return n > 0, err
}

func scanConflictSet(row scanner) (*model.ConflictSet, error) {
	var cs model.ConflictSet
	var kind, status, created, updated string
	var minimized int
	if err := row.Scan(&cs.ID, &cs.BatchID, &kind, &status, &minimized, &cs.MemberCount, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrConflictSetNotFound
		}
		return nil, err
	}
	cs.Kind = model.ConflictKind(kind)
	cs.Status = model.ConflictSetStatus(status)
	cs.Minimized = minimized != 0
	cs.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	cs.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &cs, nil
}
