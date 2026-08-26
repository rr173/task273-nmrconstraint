package httpapi

import (
	"net/http"

	"task273-nmrconstraint/internal/model"
)

type importAtomsReq struct {
	Atoms []atomInput `json:"atoms"`
}

type atomInput struct {
	Name    string `json:"name"`
	Residue string `json:"residue"`
	Element string `json:"element"`
}

// handleImportAtoms POST /api/batches/{id}/atoms
func (s *Server) handleImportAtoms(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	var req importAtomsReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, model.ErrInvalidInput("invalid request body: "+err.Error()))
		return
	}
	if len(req.Atoms) == 0 {
		writeErr(w, model.ErrInvalidInput("atoms must not be empty"))
		return
	}
	atoms := make([]*model.Atom, 0, len(req.Atoms))
	for _, in := range req.Atoms {
		atoms = append(atoms, model.NewAtom(id, in.Name, in.Residue, in.Element))
	}
	created, err := s.atoms.Import(id, atoms)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleListAtoms GET /api/batches/{id}/atoms
func (s *Server) handleListAtoms(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	list, err := s.atoms.List(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetAtom GET /api/atoms/{id}
func (s *Server) handleGetAtom(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid atom id"))
		return
	}
	a, err := s.atoms.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// handleExcludeAtom POST /api/atoms/{id}/exclude
func (s *Server) handleExcludeAtom(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid atom id"))
		return
	}
	a, err := s.atoms.Exclude(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// handleActivateAtom POST /api/atoms/{id}/activate
func (s *Server) handleActivateAtom(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid atom id"))
		return
	}
	a, err := s.atoms.Activate(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}
