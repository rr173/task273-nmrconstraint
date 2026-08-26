package httpapi

import (
	"net/http"

	"task273-nmrconstraint/internal/constraint"
	"task273-nmrconstraint/internal/mapping"
	"task273-nmrconstraint/internal/snapshot"
	"task273-nmrconstraint/internal/service"
	"task273-nmrconstraint/internal/store"
)

// Server 聚合全部依赖并注册路由。
type Server struct {
	batches  *service.BatchService
	atoms    *mapping.Service
	peaks    *service.PeakService
	cons     *constraint.Service
	diag     *service.DiagnosisService
	snaps    *snapshot.Service
	store    *store.DB
	mux      *http.ServeMux
}

// NewServer 构造 HTTP 服务。
func NewServer(db *store.DB, batches *service.BatchService, atoms *mapping.Service, peaks *service.PeakService, cons *constraint.Service, diag *service.DiagnosisService, snaps *snapshot.Service) *Server {
	s := &Server{
		batches: batches,
		atoms:   atoms,
		peaks:   peaks,
		cons:    cons,
		diag:    diag,
		snaps:   snaps,
		store:   db,
		mux:     http.NewServeMux(),
	}
	s.routes()
	return s
}

// Handler 返回 http.Handler。
func (s *Server) Handler() http.Handler {
	return s.mux
}

// routes 注册全部 /api 路由。
func (s *Server) routes() {
	// 健康
	s.mux.HandleFunc("GET /api/health", s.handleHealth)

	// 批次
	s.mux.HandleFunc("POST /api/batches", s.handleCreateBatch)
	s.mux.HandleFunc("GET /api/batches", s.handleListBatches)
	s.mux.HandleFunc("GET /api/batches/{id}", s.handleGetBatch)
	s.mux.HandleFunc("PATCH /api/batches/{id}", s.handleUpdateBatch)
	s.mux.HandleFunc("POST /api/batches/{id}/advance", s.handleAdvanceBatch)

	// 原子
	s.mux.HandleFunc("POST /api/batches/{id}/atoms", s.handleImportAtoms)
	s.mux.HandleFunc("GET /api/batches/{id}/atoms", s.handleListAtoms)
	s.mux.HandleFunc("GET /api/atoms/{id}", s.handleGetAtom)
	s.mux.HandleFunc("POST /api/atoms/{id}/exclude", s.handleExcludeAtom)
	s.mux.HandleFunc("POST /api/atoms/{id}/activate", s.handleActivateAtom)

	// NOE 峰
	s.mux.HandleFunc("POST /api/batches/{id}/peaks", s.handleImportPeaks)
	s.mux.HandleFunc("GET /api/batches/{id}/peaks", s.handleListPeaks)
	s.mux.HandleFunc("PATCH /api/peaks/{id}", s.handleUpdatePeak)
	s.mux.HandleFunc("POST /api/peaks/{id}/overlap", s.handleMarkOverlap)
	s.mux.HandleFunc("POST /api/peaks/{id}/exclude", s.handleExcludePeak)

	// 约束
	s.mux.HandleFunc("POST /api/batches/{id}/constraints", s.handleCreateConstraints)
	s.mux.HandleFunc("GET /api/batches/{id}/constraints", s.handleListConstraints)
	s.mux.HandleFunc("GET /api/constraints/{id}", s.handleGetConstraint)
	s.mux.HandleFunc("POST /api/constraints/{id}/exclude", s.handleExcludeConstraint)
	s.mux.HandleFunc("POST /api/constraints/{id}/restore", s.handleRestoreConstraint)

	// 求解与冲突
	s.mux.HandleFunc("POST /api/batches/{id}/solve", s.handleSolve)
	s.mux.HandleFunc("GET /api/batches/{id}/bounds", s.handleListBounds)
	s.mux.HandleFunc("GET /api/batches/{id}/violations", s.handleListViolations)
	s.mux.HandleFunc("GET /api/batches/{id}/conflicts", s.handleListConflicted)

	// 冲突集与豁免
	s.mux.HandleFunc("POST /api/batches/{id}/conflictsets", s.handleBuildConflictSets)
	s.mux.HandleFunc("GET /api/batches/{id}/conflictsets", s.handleListConflictSets)
	s.mux.HandleFunc("GET /api/conflictsets/{id}", s.handleGetConflictSet)
	s.mux.HandleFunc("POST /api/conflictsets/{id}/minimize", s.handleMinimizeConflictSet)
	s.mux.HandleFunc("POST /api/batches/{id}/exemptions", s.handleCreateExemption)
	s.mux.HandleFunc("GET /api/batches/{id}/exemptions", s.handleListExemptions)

	// 快照
	s.mux.HandleFunc("POST /api/batches/{id}/snapshots", s.handleCreateSnapshot)
	s.mux.HandleFunc("GET /api/batches/{id}/snapshots", s.handleListSnapshots)
	s.mux.HandleFunc("GET /api/snapshots/{id}", s.handleGetSnapshot)
	s.mux.HandleFunc("POST /api/snapshots/{id}/publish", s.handlePublishSnapshot)
}
