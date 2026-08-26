package httpapi

import (
	"net/http"

	"task273-nmrconstraint/internal/model"
)

type createBatchReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type advanceBatchReq struct {
	Status model.BatchStatus `json:"status"`
}

// handleCreateBatch POST /api/batches
func (s *Server) handleCreateBatch(w http.ResponseWriter, r *http.Request) {
	var req createBatchReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, model.ErrInvalidInput("invalid request body: "+err.Error()))
		return
	}
	b, err := s.batches.Create(req.Name, req.Description)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// handleListBatches GET /api/batches
func (s *Server) handleListBatches(w http.ResponseWriter, r *http.Request) {
	list, err := s.batches.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetBatch GET /api/batches/{id}
func (s *Server) handleGetBatch(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	b, err := s.batches.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// handleUpdateBatch PATCH /api/batches/{id}
func (s *Server) handleUpdateBatch(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	var req createBatchReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, model.ErrInvalidInput("invalid request body: "+err.Error()))
		return
	}
	b, err := s.batches.UpdateMeta(id, req.Name, req.Description)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// handleAdvanceBatch POST /api/batches/{id}/advance
func (s *Server) handleAdvanceBatch(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	var req advanceBatchReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, model.ErrInvalidInput("invalid request body: "+err.Error()))
		return
	}
	b, err := s.batches.Advance(id, req.Status)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}
