package planning

// ApproveSpec validates a draft and returns an approved copy.
func ApproveSpec(spec Spec, approval Approval) (Spec, error) {
	if spec.Status != SpecDraft {
		return Spec{}, domainError(ErrInvalidTransition, []ArtifactID{spec.ID}, "only draft specs can be approved")
	}
	if approval.State != ApprovalApproved {
		return Spec{}, domainError(ErrApprovalRequired, []ArtifactID{spec.ID}, "approval decision must be approved")
	}

	candidate := cloneSpec(spec)
	candidate.Approval = approval
	candidate.Status = SpecApproved
	if findings := ValidateSpec(candidate); hasErrorFindings(findings) {
		return Spec{}, domainError(ErrInvalidArtifact, []ArtifactID{spec.ID}, findingMessages(findings)...)
	}
	return candidate, nil
}

// ReopenSpec returns a non-draft spec to draft for an explicit new pass.
func ReopenSpec(spec Spec) (Spec, error) {
	if spec.Status == SpecDraft {
		return Spec{}, domainError(ErrInvalidTransition, []ArtifactID{spec.ID}, "spec is already draft")
	}
	if !spec.Status.valid() {
		return Spec{}, domainError(ErrInvalidTransition, []ArtifactID{spec.ID}, "spec status is invalid")
	}

	reopened := cloneSpec(spec)
	reopened.Status = SpecDraft
	reopened.Approval = Approval{State: ApprovalPending}
	reopened.ExecutionID = nil
	return reopened, nil
}

func findingMessages(findings []Finding) []string {
	messages := make([]string, 0, len(findings))
	for _, finding := range findings {
		if finding.Severity == SeverityError {
			messages = append(messages, finding.Message)
		}
	}
	return messages
}
