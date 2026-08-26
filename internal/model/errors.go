package model

import "errors"

// 领域错误全集。HTTP 层据此映射为 400/404/409/422。
var (
	ErrBatchNotFound        = errors.New("batch not found")
	ErrAtomNotFound         = errors.New("atom not found")
	ErrPeakNotFound         = errors.New("noe peak not found")
	ErrConstraintNotFound   = errors.New("constraint not found")
	ErrConflictSetNotFound  = errors.New("conflict set not found")
	ErrSnapshotNotFound     = errors.New("snapshot not found")
	ErrExemptionNotFound    = errors.New("exemption not found")
	ErrInvalidTransition    = errors.New("invalid batch status transition")
	ErrSealedBatchImmutable = errors.New("sealed batch is immutable")
	ErrPublishedSnapshot    = errors.New("published snapshot is immutable")
	ErrInvalidInterval      = errors.New("distance interval is invalid: lo must be >= 0 and hi >= lo")
	ErrSelfConstraint       = errors.New("constraint cannot reference the same atom twice")
	ErrAtomNotFoundRef      = errors.New("constraint references a non-existent atom")
	ErrPeakNotFoundRef      = errors.New("constraint references a non-existent peak")
	ErrDuplicateAtomName    = errors.New("atom name already exists in batch")
	ErrDuplicatePeakName    = errors.New("peak name already exists in batch")
	ErrDuplicateConstraint  = errors.New("duplicate constraint for the same atom pair")
	ErrNotSolvable          = errors.New("batch must be in solving/conflicted/releasable state to solve")
	ErrEmptyConstraintSet   = errors.New("no constraints to solve")
	ErrConflictNotPresent   = errors.New("no active conflicts to build a conflict set")
	ErrMinimizeDone         = errors.New("conflict set already minimized")
	ErrExemptionExists      = errors.New("constraint already exempted in this batch")
	ErrPublishNotReleasable = errors.New("only a releasable batch can publish a snapshot")
	ErrSnapshotNotDraft     = errors.New("only draft snapshots can be published")
	ErrPeakPairMismatch     = errors.New("peak atom pair does not match constraint atom pair")
	ErrConfidenceOutOfRange = errors.New("confidence must be within [0,1]")
	ErrIntensityInvalid     = errors.New("peak intensity must be positive")
)

// InvalidInputError 携带用户可见消息的校验类错误，HTTP 层映射为 422。
type InvalidInputError struct {
	Message string
}

func (e *InvalidInputError) Error() string { return e.Message }

// ErrInvalidInput 构造一个校验错误。
func ErrInvalidInput(msg string) error {
	return &InvalidInputError{Message: msg}
}
