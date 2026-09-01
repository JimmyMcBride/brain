package planning

import (
	"errors"
	"strings"
)

// ErrorCode identifies a stable Planning domain failure.
type ErrorCode string

const (
	// ErrInvalidIdentifier reports a non-canonical artifact identifier.
	ErrInvalidIdentifier ErrorCode = "invalid_identifier"
	// ErrInvalidArtifact reports invalid artifact content or relationships.
	ErrInvalidArtifact ErrorCode = "invalid_artifact"
	// ErrInvalidOwnership reports an unsupported ownership mode.
	ErrInvalidOwnership ErrorCode = "invalid_ownership"
	// ErrInvalidTransition reports an unsupported lifecycle transition.
	ErrInvalidTransition ErrorCode = "invalid_transition"
	// ErrApprovalRequired reports a missing approval decision.
	ErrApprovalRequired ErrorCode = "approval_required"
	// ErrDependencyMissing reports a reference to an absent dependency.
	ErrDependencyMissing ErrorCode = "dependency_missing"
	// ErrDependencyCycle reports a cycle in the spec dependency graph.
	ErrDependencyCycle ErrorCode = "dependency_cycle"
	// ErrNotReady reports a spec that cannot begin execution.
	ErrNotReady ErrorCode = "not_ready"
	// ErrInvalidExecution reports inconsistent execution state.
	ErrInvalidExecution ErrorCode = "invalid_execution"
	// ErrInvalidSlice reports an invalid runtime slice or evidence record.
	ErrInvalidSlice ErrorCode = "invalid_slice"
)

// DomainError carries a stable code plus domain identifiers and reasons.
type DomainError struct {
	Code      ErrorCode
	Artifacts []ArtifactID
	Reasons   []string
}

func (e *DomainError) Error() string {
	if e == nil {
		return ""
	}
	message := string(e.Code)
	if len(e.Artifacts) > 0 {
		ids := make([]string, len(e.Artifacts))
		for i, id := range e.Artifacts {
			ids[i] = string(id)
		}
		message += ": " + strings.Join(ids, ", ")
	}
	if len(e.Reasons) > 0 {
		message += ": " + strings.Join(e.Reasons, "; ")
	}
	return message
}

func domainError(code ErrorCode, artifacts []ArtifactID, reasons ...string) error {
	return &DomainError{
		Code:      code,
		Artifacts: append([]ArtifactID(nil), artifacts...),
		Reasons:   append([]string(nil), reasons...),
	}
}

// ErrorCodeOf returns the stable code carried by a domain error.
func ErrorCodeOf(err error) ErrorCode {
	if err == nil {
		return ""
	}
	var domain *DomainError
	if errors.As(err, &domain) {
		return domain.Code
	}
	return ""
}
