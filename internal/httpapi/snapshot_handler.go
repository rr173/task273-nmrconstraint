package httpapi

import (
	"net/http"

	"task273-nmrconstraint/internal/model"
)

type createSnapshotReq struct {
	Name string `json:"name"`
}

// handleCreateSnapshot POST /api/batches/{id}/snapshots
func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	var req createSnapshotReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, model.ErrInvalidInput("invalid request body: "+err.Error()))
		return
	}
	if req.Name == "" {
		writeErr(w, model.ErrInvalidInput("snapshot name is required"))
		return
	}
	snap, err := s.snaps.CreateDraft(id, req.Name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, snap)
}

// handleListSnapshots GET /api/batches/{id}/snapshots
func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	list, err := s.snaps.List(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetSnapshot GET /api/snapshots/{id}
func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid snapshot id"))
		return
	}
	snap, items, err := s.snaps.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"snapshot": snap,
		"items":    items,
	})
}

// handlePublishSnapshot POST /api/snapshots/{id}/publish
func (s *Server) handlePublishSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid snapshot id"))
		return
	}
	snap, _, err := s.snaps.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	published, err := s.snaps.Publish(snap.BatchID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, published)
}
