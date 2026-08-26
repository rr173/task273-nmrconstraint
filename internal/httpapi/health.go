package httpapi

import (
	"net/http"
	"strconv"
)

// pathID 从路由路径参数中解析整数 ID。
func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

// handleHealth 返回服务健康状态。
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"db":     s.store.Path,
	})
}
