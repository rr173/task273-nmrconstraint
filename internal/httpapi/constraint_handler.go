package httpapi

import (
	"net/http"

	"task273-nmrconstraint/internal/model"
)

type importConstraintsReq struct {
	Constraints []constraintInput `json:"constraints"`
}

type constraintInput struct {
	PeakID  int64   `json:"peak_id"`
	Atom1ID int64   `json:"atom1_id"`
	Atom2ID int64   `json:"atom2_id"`
	LoDist  float64 `json:"lo_dist"`
	HiDist  float64 `json:"hi_dist"`
}

// handleCreateConstraints POST /api/batches/{id}/constraints
func (s *Server) handleCreateConstraints(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	var req importConstraintsReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, model.ErrInvalidInput("invalid request body: "+err.Error()))
		return
	}
	if len(req.Constraints) == 0 {
		writeErr(w, model.ErrInvalidInput("constraints must not be empty"))
		return
	}
	cons := make([]*model.Constraint, 0, len(req.Constraints))
	for _, in := range req.Constraints {
		cons = append(cons, model.NewConstraint(id, in.PeakID, in.Atom1ID, in.Atom2ID, in.LoDist, in.HiDist))
	}
	created, err := s.cons.CreateFromPeaks(id, cons)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleListConstraints GET /api/batches/{id}/constraints
func (s *Server) handleListConstraints(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	list, err := s.cons.List(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetConstraint GET /api/constraints/{id}
func (s *Server) handleGetConstraint(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid constraint id"))
		return
	}
	c, err := s.cons.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// handleExcludeConstraint POST /api/constraints/{id}/exclude
func (s *Server) handleExcludeConstraint(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid constraint id"))
		return
	}
	c, err := s.cons.Exclude(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// handleRestoreConstraint POST /api/constraints/{id}/restore
func (s *Server) handleRestoreConstraint(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid constraint id"))
		return
	}
	c, err := s.cons.Restore(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
