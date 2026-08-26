package httpapi

import (
	"net/http"

	"task273-nmrconstraint/internal/model"
)

type createExemptionReq struct {
	ConstraintID int64  `json:"constraint_id"`
	Reason       string `json:"reason"`
}

// handleSolve POST /api/batches/{id}/solve
func (s *Server) handleSolve(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	result, err := s.diag.Solve(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleListBounds GET /api/batches/{id}/bounds
func (s *Server) handleListBounds(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	edges, err := s.diag.ListBounds(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, edges)
}

// handleListViolations GET /api/batches/{id}/violations
func (s *Server) handleListViolations(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	violations, err := s.diag.ListViolations(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, violations)
}

// handleListConflicted GET /api/batches/{id}/conflicts
func (s *Server) handleListConflicted(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	cons, err := s.diag.ListConflicted(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cons)
}

// handleBuildConflictSets POST /api/batches/{id}/conflictsets
func (s *Server) handleBuildConflictSets(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	sets, err := s.diag.BuildConflictSets(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sets)
}

// handleListConflictSets GET /api/batches/{id}/conflictsets
func (s *Server) handleListConflictSets(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	sets, err := s.diag.ListConflictSets(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sets)
}

// handleGetConflictSet GET /api/conflictsets/{id}
func (s *Server) handleGetConflictSet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid conflict set id"))
		return
	}
	cs, members, err := s.diag.GetConflictSet(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"conflict_set": cs,
		"members":      members,
	})
}

// handleMinimizeConflictSet POST /api/conflictsets/{id}/minimize
func (s *Server) handleMinimizeConflictSet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid conflict set id"))
		return
	}
	cs, members, err := s.diag.Minimize(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"conflict_set": cs,
		"members":      members,
	})
}

// handleCreateExemption POST /api/batches/{id}/exemptions
func (s *Server) handleCreateExemption(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	var req createExemptionReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, model.ErrInvalidInput("invalid request body: "+err.Error()))
		return
	}
	if req.ConstraintID <= 0 {
		writeErr(w, model.ErrInvalidInput("constraint_id is required"))
		return
	}
	e, err := s.diag.Exempt(id, req.ConstraintID, req.Reason)
	if e != nil {
		writeJSON(w, http.StatusCreated, e)
		return
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// handleListExemptions GET /api/batches/{id}/exemptions
func (s *Server) handleListExemptions(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	list, err := s.diag.Exemptions(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
