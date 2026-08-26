package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"task273-nmrconstraint/internal/model"
)

// envelope 是统一响应包装。
type envelope struct {
	Data any `json:"data,omitempty"`
}

// errorBody 是错误响应体。
type errorBody struct {
	Error string `json:"error"`
}

// writeJSON 以 200 写 JSON。
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope{Data: data}); err != nil {
		log.Printf("write json: %v", err)
	}
}

// writeErr 将领域错误映射为 HTTP 状态码。
func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		status = http.StatusRequestTimeout
	case errors.Is(err, model.ErrBatchNotFound),
		errors.Is(err, model.ErrAtomNotFound),
		errors.Is(err, model.ErrPeakNotFound),
		errors.Is(err, model.ErrConstraintNotFound),
		errors.Is(err, model.ErrConflictSetNotFound),
		errors.Is(err, model.ErrSnapshotNotFound),
		errors.Is(err, model.ErrExemptionNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrInvalidTransition),
		errors.Is(err, model.ErrSealedBatchImmutable),
		errors.Is(err, model.ErrPublishedSnapshot),
		errors.Is(err, model.ErrSnapshotNotDraft),
		errors.Is(err, model.ErrPublishNotReleasable),
		errors.Is(err, model.ErrMinimizeDone),
		errors.Is(err, model.ErrExemptionExists):
		status = http.StatusConflict
	case errors.Is(err, model.ErrInvalidInterval),
		errors.Is(err, model.ErrSelfConstraint),
		errors.Is(err, model.ErrAtomNotFoundRef),
		errors.Is(err, model.ErrPeakNotFoundRef),
		errors.Is(err, model.ErrDuplicateAtomName),
		errors.Is(err, model.ErrDuplicatePeakName),
		errors.Is(err, model.ErrDuplicateConstraint),
		errors.Is(err, model.ErrPeakPairMismatch),
		errors.Is(err, model.ErrConfidenceOutOfRange),
		errors.Is(err, model.ErrIntensityInvalid),
		errors.Is(err, model.ErrEmptyConstraintSet),
		errors.Is(err, model.ErrConflictNotPresent),
		errors.Is(err, model.ErrNotSolvable):
		status = http.StatusUnprocessableEntity
	}
	var invalid *model.InvalidInputError
	if errors.As(err, &invalid) {
		status = http.StatusUnprocessableEntity
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Error: err.Error()})
}

// decodeJSON 解析请求体并拒绝未知字段。
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
