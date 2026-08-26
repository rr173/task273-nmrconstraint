package store

import (
	"database/sql"
	"strings"
	"time"

	"task273-nmrconstraint/internal/model"
)

// PeakStore 提供 NOE 峰的持久化。
type PeakStore struct {
	db *DB
}

// NewPeakStore 构造峰存储。
func NewPeakStore(db *DB) *PeakStore { return &PeakStore{db: db} }

// Create 批量插入 NOE 峰。
func (s *PeakStore) Create(batchID int64, peaks []*model.NoePeak) ([]*model.NoePeak, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	for _, p := range peaks {
		res, err := tx.Exec(
			`INSERT INTO noe_peaks(batch_id, name, atom1_id, atom2_id, intensity, confidence, overlap_suspected, status, created_at)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			batchID, p.Name, p.Atom1ID, p.Atom2ID, p.Intensity, p.Confidence, boolToInt(p.OverlapSuspected), string(p.Status), p.CreatedAt.Format(time.RFC3339Nano),
		)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, model.ErrDuplicatePeakName
			}
			return nil, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		p.ID = id
		p.BatchID = batchID
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return peaks, nil
}

// List 列出批次全部峰。
func (s *PeakStore) List(batchID int64) ([]*model.NoePeak, error) {
	rows, err := s.db.Query(`SELECT id, batch_id, name, atom1_id, atom2_id, intensity, confidence, overlap_suspected, status, created_at FROM noe_peaks WHERE batch_id = ? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.NoePeak
	for rows.Next() {
		p, err := scanPeak(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Get 按 ID 读取峰。
func (s *PeakStore) Get(id int64) (*model.NoePeak, error) {
	row := s.db.QueryRow(`SELECT id, batch_id, name, atom1_id, atom2_id, intensity, confidence, overlap_suspected, status, created_at FROM noe_peaks WHERE id = ?`, id)
	return scanPeak(row)
}

// UpdateConfidence 更新峰置信度。
func (s *PeakStore) UpdateConfidence(id int64, confidence float64) error {
	_, err := s.db.Exec(`UPDATE noe_peaks SET confidence = ?, status = ? WHERE id = ?`, confidence, string(model.PeakValid), id)
	return err
}

// UpdateStatus 更新峰状态与重叠标记。
func (s *PeakStore) UpdateStatus(id int64, status model.PeakStatus, overlap bool) error {
	_, err := s.db.Exec(`UPDATE noe_peaks SET status = ?, overlap_suspected = ? WHERE id = ?`, string(status), boolToInt(overlap), id)
	return err
}

// ByIDs 按 ID 集合读取峰（保持顺序，缺一即错）。
func (s *PeakStore) ByIDs(batchID int64, ids []int64) ([]*model.NoePeak, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ph := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+1)
	args = append(args, batchID)
	for _, id := range ids {
		args = append(args, id)
	}
	q := `SELECT id, batch_id, name, atom1_id, atom2_id, intensity, confidence, overlap_suspected, status, created_at FROM noe_peaks WHERE batch_id = ? AND id IN (` + ph + `) ORDER BY id`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.NoePeak
	for rows.Next() {
		p, err := scanPeak(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) != len(ids) {
		return nil, model.ErrPeakNotFound
	}
	return out, nil
}

func scanPeak(row scanner) (*model.NoePeak, error) {
	var p model.NoePeak
	var status, created string
	var overlap int
	if err := row.Scan(&p.ID, &p.BatchID, &p.Name, &p.Atom1ID, &p.Atom2ID, &p.Intensity, &p.Confidence, &overlap, &status, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrPeakNotFound
		}
		return nil, err
	}
	p.OverlapSuspected = overlap != 0
	p.Status = model.PeakStatus(status)
	p.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return &p, nil
}
