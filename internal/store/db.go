package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DB 封装 SQLite 连接并记录打开时间，供恢复验证使用。
type DB struct {
	*sql.DB
	Path       string
	OpenedAt   time.Time
}

// Open 打开（或创建）SQLite 数据库并应用迁移。
func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(32)
	db := &DB{DB: sqlDB, Path: path, OpenedAt: time.Now().UTC()}
	if err := db.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := db.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// Close 关闭数据库连接。
func (db *DB) Close() error {
	return db.DB.Close()
}
