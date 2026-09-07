package application

import (
	"context"
	"fmt"
	"strings"
)

// PermissionCollaboration authorizes remote source repair.
const PermissionCollaboration = "planning.collaboration"

// EventCollaborationRepaired records a completed remote source repair.
const EventCollaborationRepaired = "planning.collaboration.repaired"

// CollaborationRepairInput specifies a canonical source split. ExpectedRevision
// binds a confirmed request to the source shown in the preview.
type CollaborationRepairInput struct {
	Source           ExternalReference `json:"source"`
	Specs            []string          `json:"specs"`
	ExpectedRevision string            `json:"expected_revision,omitempty"`
	Confirmed        bool              `json:"confirmed"`
}

// CollaborationRepairResult preserves both the preview and mutation evidence.
type CollaborationRepairResult struct {
	SchemaVersion int                          `json:"schema_version"`
	Request       CollaborationRepairRequest   `json:"request"`
	Specs         []string                     `json:"specs"`
	Action        MutationAction               `json:"action"`
	Applied       bool                         `json:"applied"`
	Evidence      *CollaborationRepairEvidence `json:"evidence,omitempty"`
	Event         *Event                       `json:"event,omitempty"`
}

func authorizeCollaboration(ctx context.Context, source CollaborationSource, authorizer Authorizer) error {
	if source == nil {
		return fmt.Errorf("collaboration source is unavailable")
	}
	if authorizer == nil {
		return fmt.Errorf("collaboration requires an authorizer")
	}
	return authorizer.Require(ctx, PermissionRead)
}

// AssessCollaboration evaluates provider content and contributions without
// requiring a local workspace or changing artifact ownership.
func (s *Service) AssessCollaboration(ctx context.Context, source CollaborationSource, ref ExternalReference, authorizer Authorizer) (CollaborationAssessment, error) {
	if err := authorizeCollaboration(ctx, source, authorizer); err != nil {
		return CollaborationAssessment{}, err
	}
	snapshot, err := source.Read(ctx, ref)
	if err != nil {
		return CollaborationAssessment{}, err
	}
	decision := assessCollaboration(snapshot)
	return CollaborationAssessment{SchemaVersion: IntegrationContractVersion, Kind: "maturity_assessment", GeneratedAt: s.now().UTC().Format("2006-01-02T15:04:05Z07:00"), Source: map[string]any{"reference": snapshot.Source}, Ownership: map[string]any{"canonical_source": "external"}, Decision: decision}, nil
}

// PreviewCollaborationRepair returns the replacement content and source revision.
func (s *Service) PreviewCollaborationRepair(ctx context.Context, source CollaborationSource, input CollaborationRepairInput, authorizer Authorizer) (CollaborationRepairResult, error) {
	var result CollaborationRepairResult
	if err := authorizeCollaboration(ctx, source, authorizer); err != nil {
		return result, err
	}
	specs := []string{}
	seen := map[string]bool{}
	for _, title := range input.Specs {
		title = strings.TrimSpace(title)
		if strings.ContainsAny(title, "\r\n") {
			return result, fmt.Errorf("spec titles must be single lines")
		}
		key := strings.ToLower(title)
		if title != "" && !seen[key] {
			seen[key] = true
			specs = append(specs, title)
		}
	}
	if len(specs) < 2 {
		return result, fmt.Errorf("repair requires at least two distinct spec titles")
	}
	snapshot, err := source.Read(ctx, input.Source)
	if err != nil {
		return result, err
	}
	if snapshot.Source.Revision == "" || (input.ExpectedRevision != "" && snapshot.Source.Revision != input.ExpectedRevision) {
		return result, &IntegrationError{Class: IntegrationRevisionConflict, Operation: "collaboration.repair", Message: "source changed since preview or has no revision"}
	}
	lines := make([]string, len(specs))
	for i, title := range specs {
		lines[i] = "- " + title
	}
	body := setMarkdownSection(snapshot.Content, "Specs", strings.Join(lines, "\n"))
	result = CollaborationRepairResult{SchemaVersion: IntegrationContractVersion, Specs: specs, Action: MutationUpdate, Request: CollaborationRepairRequest{Source: snapshot.Source, ExpectedRevision: snapshot.Source.Revision, Content: body}}
	if body == snapshot.Content {
		result.Action = MutationUnchanged
	}
	return result, nil
}

// RepairCollaboration requires confirmation, permission, and an audit sink before
// a guarded provider write. Audit failure retains completed mutation evidence.
func (s *Service) RepairCollaboration(ctx context.Context, source CollaborationSource, input CollaborationRepairInput, authorizer Authorizer, events EventSink) (CollaborationRepairResult, error) {
	if !input.Confirmed {
		return CollaborationRepairResult{}, ErrConfirmationRequired
	}
	if input.ExpectedRevision == "" {
		return CollaborationRepairResult{}, &IntegrationError{Class: IntegrationRevisionConflict, Message: "preview revision is required"}
	}
	if authorizer == nil {
		return CollaborationRepairResult{}, fmt.Errorf("collaboration requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionCollaboration); err != nil {
		return CollaborationRepairResult{}, err
	}
	if events == nil {
		return CollaborationRepairResult{}, ErrEventSinkRequired
	}
	result, err := s.PreviewCollaborationRepair(ctx, source, input, authorizer)
	if err != nil || result.Action == MutationUnchanged {
		return result, err
	}
	evidence, err := source.Repair(ctx, result.Request)
	if err != nil {
		return result, err
	}
	result.Evidence = &evidence
	if !evidence.Changed {
		result.Action = MutationUnchanged
		return result, nil
	}
	result.Applied = true
	event := Event{Name: EventCollaborationRepaired, ModuleID: s.moduleID, Source: &evidence.Source, Outcome: MutationUpdate, OccurredAt: s.now().UTC()}
	result.Event = &event
	if err := events.Publish(ctx, event); err != nil {
		return result, &IntegrationError{Class: IntegrationPartialFailure, Operation: "collaboration.audit", Message: "source repaired but audit persistence failed", Err: err}
	}
	return result, nil
}

func assessCollaboration(snapshot CollaborationSourceSnapshot) MaturityDecision {
	body := strings.TrimSpace(snapshot.Content)
	for _, contribution := range snapshot.Contributions {
		if strings.TrimSpace(contribution.Content) != "" {
			body += "\n\n## Comment\n" + strings.TrimSpace(contribution.Content)
		}
	}
	decision := MaturityDecision{State: "not_ready", Confidence: "low", SourceMode: "external", Reason: "The source still has planning gaps that would force promotion to guess.", SuggestedTitles: map[string]any{"initiative": snapshot.Title}}
	checks := []struct{ heading, keyword, strength, gap string }{
		{"Problem", "problem", "clear problem", "missing concrete problem statement"},
		{"Goals", "goal", "clear goals or user value", "missing goals or user value"},
		{"Constraints", "constraint", "clear constraints", "missing constraints"},
		{"Non-Goals", "non-goal", "non-goals are called out", "missing non-goals"},
		{"Proposed Shape", "shape", "solution shape is visible", "missing initial solution shape"},
	}
	for _, check := range checks {
		value := collaborationSection(body, check.heading, check.keyword)
		if check.heading == "Proposed Shape" {
			value = firstNonEmpty(value, collaborationSection(body, "Decision Snapshot", "decision"), collaborationSection(body, "Candidate Approaches", "approach"))
		}
		if value == "" {
			decision.Gaps = append(decision.Gaps, check.gap)
		} else {
			decision.Strengths = append(decision.Strengths, check.strength)
		}
	}
	if len(decision.Gaps) > 0 {
		return decision
	}
	titles, multi := collaborationTitles(body)
	if multi && len(titles) < 2 {
		decision.State, decision.Confidence = "needs_source_repair", "medium"
		decision.Reason = "The source requests multiple specs but does not identify at least two distinct titles."
		decision.SuggestedTitles["specs"] = titles
		return decision
	}
	if len(titles) == 0 {
		titles = []string{snapshot.Title}
	}
	decision.Confidence = "high"
	decision.SuggestedTitles = map[string]any{"specs": titles}
	decision.State, decision.RecommendedPath = "ready_single_spec", "single_spec"
	decision.Reason = "The source is clear enough to promote directly into one bounded spec."
	if len(titles) > 1 {
		decision.State, decision.RecommendedPath = "ready_multi_spec", "multi_spec"
		decision.SuggestedTitles["initiative"] = snapshot.Title
		decision.Reason = fmt.Sprintf("The source is clear enough to promote as an initiative with %d initial specs.", len(titles))
	}
	for _, title := range titles {
		blocked := []string{}
		for _, other := range titles {
			if title != other && strings.Contains(strings.ToLower(body), strings.ToLower(title+" depends on "+other)) {
				blocked = append(blocked, other)
			}
		}
		decision.DependencyGuess = append(decision.DependencyGuess, map[string]any{"spec": title, "blocked_by": blocked})
	}
	return decision
}
