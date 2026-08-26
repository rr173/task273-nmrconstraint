package store

// migrate 创建全部表结构。迁移为幂等操作（CREATE TABLE IF NOT EXISTS）。
func (db *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS batches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			sealed_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS atoms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			name TEXT NOT NULL,
			residue TEXT NOT NULL DEFAULT '',
			element TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(batch_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS noe_peaks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			name TEXT NOT NULL,
			atom1_id INTEGER NOT NULL REFERENCES atoms(id),
			atom2_id INTEGER NOT NULL REFERENCES atoms(id),
			intensity REAL NOT NULL,
			confidence REAL NOT NULL,
			overlap_suspected INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(batch_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS constraints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			peak_id INTEGER NOT NULL REFERENCES noe_peaks(id),
			atom1_id INTEGER NOT NULL REFERENCES atoms(id),
			atom2_id INTEGER NOT NULL REFERENCES atoms(id),
			lo_dist REAL NOT NULL,
			hi_dist REAL NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(batch_id, atom1_id, atom2_id)
		)`,
		`CREATE TABLE IF NOT EXISTS bound_edges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			constraint_id INTEGER NOT NULL REFERENCES constraints(id),
			atom1_id INTEGER NOT NULL,
			atom2_id INTEGER NOT NULL,
			lo_bound REAL NOT NULL,
			hi_bound REAL NOT NULL,
			tightened INTEGER NOT NULL DEFAULT 0,
			iteration INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL,
			UNIQUE(batch_id, constraint_id)
		)`,
		`CREATE TABLE IF NOT EXISTS conflict_sets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			minimized INTEGER NOT NULL DEFAULT 0,
			member_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS conflict_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conflict_set_id INTEGER NOT NULL REFERENCES conflict_sets(id),
			constraint_id INTEGER NOT NULL REFERENCES constraints(id),
			removed INTEGER NOT NULL DEFAULT 0,
			UNIQUE(conflict_set_id, constraint_id)
		)`,
		`CREATE TABLE IF NOT EXISTS exemptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			constraint_id INTEGER NOT NULL REFERENCES constraints(id),
			reason TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(batch_id, constraint_id)
		)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			confidence_version INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			published_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS snapshot_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			snapshot_id INTEGER NOT NULL REFERENCES snapshots(id),
			constraint_id INTEGER NOT NULL,
			atom1_id INTEGER NOT NULL,
			atom2_id INTEGER NOT NULL,
			lo_dist REAL NOT NULL,
			hi_dist REAL NOT NULL,
			lo_bound REAL NOT NULL,
			hi_bound REAL NOT NULL,
			status TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_atoms_batch ON atoms(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_peaks_batch ON noe_peaks(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_constraints_batch ON constraints(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_edges_batch ON bound_edges(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_members_set ON conflict_members(conflict_set_id)`,
		`CREATE INDEX IF NOT EXISTS idx_items_snapshot ON snapshot_items(snapshot_id)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
