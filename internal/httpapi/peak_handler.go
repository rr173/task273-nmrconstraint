package httpapi

import (
	"net/http"

	"task273-nmrconstraint/internal/model"
)

type importPeaksReq struct {
	Peaks []peakInput `json:"peaks"`
}

type peakInput struct {
	Name       string  `json:"name"`
	Atom1ID    int64   `json:"atom1_id"`
	Atom2ID    int64   `json:"atom2_id"`
	Intensity  float64 `json:"intensity"`
	Confidence float64 `json:"confidence"`
}

type updatePeakReq struct {
	Confidence *float64 `json:"confidence,omitempty"`
}

// handleImportPeaks POST /api/batches/{id}/peaks
func (s *Server) handleImportPeaks(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	var req importPeaksReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, model.ErrInvalidInput("invalid request body: "+err.Error()))
		return
	}
	if len(req.Peaks) == 0 {
		writeErr(w, model.ErrInvalidInput("peaks must not be empty"))
		return
	}
	peaks := make([]*model.NoePeak, 0, len(req.Peaks))
	for _, in := range req.Peaks {
		peaks = append(peaks, model.NewNoePeak(id, in.Name, in.Atom1ID, in.Atom2ID, in.Intensity, in.Confidence))
	}
	created, err := s.peaks.Import(id, peaks)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleListPeaks GET /api/batches/{id}/peaks
func (s *Server) handleListPeaks(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid batch id"))
		return
	}
	list, err := s.peaks.List(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleUpdatePeak PATCH /api/peaks/{id}
func (s *Server) handleUpdatePeak(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid peak id"))
		return
	}
	var req updatePeakReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, model.ErrInvalidInput("invalid request body: "+err.Error()))
		return
	}
	if req.Confidence == nil {
		writeErr(w, model.ErrInvalidInput("confidence is required"))
		return
	}
	p, err := s.peaks.SetConfidence(id, *req.Confidence)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// handleMarkOverlap POST /api/peaks/{id}/overlap
func (s *Server) handleMarkOverlap(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid peak id"))
		return
	}
	p, err := s.peaks.MarkOverlap(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// handleExcludePeak POST /api/peaks/{id}/exclude
func (s *Server) handleExcludePeak(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, model.ErrInvalidInput("invalid peak id"))
		return
	}
	p, err := s.peaks.Exclude(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}
