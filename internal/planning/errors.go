package planning

import (
	"fmt"
	"strings"
)

// ErrorCode identifies a stable Planning domain failure.
type ErrorCode string

const (
	ErrInvalidIdentifier ErrorCode = "invalid_identifier"
	ErrInvalidArtifact   ErrorCode = "invalid_artifact"
	ErrInvalidOwnership  ErrorCode = "invalid_ownership"
	ErrInvalidTransition ErrorCode = "invalid_transition"
	ErrApprovalRequired  ErrorCode = "approval_required"
	ErrDependencyMissing ErrorCode = "dependency_missing"
	ErrDependencyCycle   ErrorCode = "dependency_cycle"
	ErrNotReady          ErrorCode = "not_ready"
	ErrInvalidExecution  ErrorCode = "invalid_execution"
	ErrInvalidSlice      ErrorCode = "invalid_slice"
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
	if domain, ok := err.(*DomainError); ok {
		return domain.Code
	}
	return ErrorCode(fmt.Sprintf("%T", err))
}
