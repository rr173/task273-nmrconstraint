package store

import "strings"

// boolToInt 将布尔值转为 SQLite 整数。
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// isUniqueViolation 判断 SQLite 错误是否为 UNIQUE 约束违反。
// modernc.org/sqlite 的违反消息形如 "UNIQUE constraint failed: atoms.batch_id, atoms.name"。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
