package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/JimmyMcBride/brain/planning"
)

var analysisCategories = []string{"Missing Constraints", "Success Criteria Gaps", "Hidden Dependencies", "Risk Gaps", "What/Why vs How Leakage", "Recommended Revisions"}
var checklistProfiles = []string{"general", "ui-flow", "api-integration", "data-migration"}

// PreviewSpecEdit returns a canonical-body replacement without writing.
func (s *Service) PreviewSpecEdit(ctx context.Context, input SpecEditInput) (SpecMutationResult, error) {
	document, err := s.specForMutation(ctx, input.ID)
	if err != nil {
		return SpecMutationResult{}, err
	}
	currentBody := document.Body
	document.Body = strings.TrimRight(strings.ReplaceAll(input.Body, "\r\n", "\n"), "\n") + "\n"
	document.Artifact.Verification = markdownItems(extractMarkdownSection(document.Body, "Verification"))
	if findings := planning.ValidateSpec(document.Artifact); hasPlanningErrors(findings) {
		return SpecMutationResult{}, fmt.Errorf("invalid spec replacement: %s", planningFindingMessages(findings))
	}
	action := MutationUpdate
	if currentBody == document.Body {
		action = MutationUnchanged
	}
	return SpecMutationResult{Action: action, Document: document}, nil
}

// EditSpec applies a confirmed canonical-body replacement.
func (s *Service) EditSpec(ctx context.Context, input SpecEditInput, authorizer Authorizer, events EventSink) (SpecMutationResult, error) {
	preview, err := s.PreviewSpecEdit(ctx, input)
	if err != nil || preview.Action == MutationUnchanged {
		return preview, err
	}
	return s.applySpecDocument(ctx, preview.Document, input.Confirmed, PermissionSpecEdit, EventSpecUpdated, authorizer, events)
}

// PreviewSpecStatus returns a validated lifecycle change without writing.
func (s *Service) PreviewSpecStatus(ctx context.Context, input SpecStatusInput) (SpecMutationResult, error) {
	document, err := s.specForMutation(ctx, input.ID)
	if err != nil {
		return SpecMutationResult{}, err
	}
	if input.Status == document.Artifact.Status {
		return SpecMutationResult{Action: MutationUnchanged, Document: document}, nil
	}
	switch input.Status {
	case planning.SpecDraft:
		document.Artifact, err = planning.ReopenSpec(document.Artifact)
	case planning.SpecApproved:
		document.Artifact, err = planning.ApproveSpec(document.Artifact, planning.Approval{State: planning.ApprovalApproved, Reason: "approved through Planning host"})
	case planning.SpecImplementing:
		return SpecMutationResult{}, fmt.Errorf("start implementing specs with spec execute")
	case planning.SpecDone:
		if document.Artifact.Status != planning.SpecImplementing {
			return SpecMutationResult{}, fmt.Errorf("only implementing specs can be marked done")
		}
		document.Artifact.Status = planning.SpecDone
	default:
		return SpecMutationResult{}, fmt.Errorf("invalid spec status %q", input.Status)
	}
	if err != nil {
		return SpecMutationResult{}, err
	}
	return SpecMutationResult{Action: MutationUpdate, Document: document}, nil
}

// SetSpecStatus applies a confirmed lifecycle change.
func (s *Service) SetSpecStatus(ctx context.Context, input SpecStatusInput, authorizer Authorizer, events EventSink) (SpecMutationResult, error) {
	preview, err := s.PreviewSpecStatus(ctx, input)
	if err != nil || preview.Action == MutationUnchanged {
		return preview, err
	}
	permission := PermissionSpecEdit
	if input.Status == planning.SpecApproved {
		permission = PermissionSpecApprove
	}
	return s.applySpecDocument(ctx, preview.Document, input.Confirmed, permission, EventSpecUpdated, authorizer, events)
}

// PreviewSpecInitiative returns a lightweight initiative metadata change without writing.
func (s *Service) PreviewSpecInitiative(ctx context.Context, input SpecInitiativeInput) (SpecMutationResult, error) {
	document, err := s.specForMutation(ctx, input.ID)
	if err != nil {
		return SpecMutationResult{}, err
	}
	if input.Clear {
		document.Artifact.Initiative = nil
		delete(document.Metadata, "initiative")
		delete(document.Metadata, "initiative_title")
		delete(document.Metadata, "initiative_summary")
	} else {
		if input.Initiative == nil || input.Initiative.Validate() != nil {
			return SpecMutationResult{}, fmt.Errorf("valid initiative --set value is required unless --clear is used")
		}
		id := *input.Initiative
		document.Artifact.Initiative = &id
		document.Metadata["initiative"] = string(id)
		if strings.TrimSpace(input.Title) != "" {
			document.Metadata["initiative_title"] = strings.TrimSpace(input.Title)
		}
		if strings.TrimSpace(input.Summary) != "" {
			document.Metadata["initiative_summary"] = strings.TrimSpace(input.Summary)
		}
	}
	action := MutationUpdate
	current, err := s.repository.GetSpec(ctx, input.ID)
	if err != nil {
		return SpecMutationResult{}, err
	}
	if sameSpecDocument(current, document) {
		action = MutationUnchanged
	}
	return SpecMutationResult{Action: action, Document: document}, nil
}

// SetSpecInitiative applies confirmed initiative metadata.
func (s *Service) SetSpecInitiative(ctx context.Context, input SpecInitiativeInput, authorizer Authorizer, events EventSink) (SpecMutationResult, error) {
	preview, err := s.PreviewSpecInitiative(ctx, input)
	if err != nil || preview.Action == MutationUnchanged {
		return preview, err
	}
	return s.applySpecDocument(ctx, preview.Document, input.Confirmed, PermissionSpecEdit, EventSpecUpdated, authorizer, events)
}

// PreviewSpecAnalysis computes and embeds an additive analysis report without writing.
func (s *Service) PreviewSpecAnalysis(ctx context.Context, id planning.ArtifactID) (SpecAnalysisReport, error) {
	document, err := s.specForMutation(ctx, id)
	if err != nil {
		return SpecAnalysisReport{}, err
	}
	findings := analyzeSpecDocument(document)
	body := setMarkdownSection(document.Body, "Analysis", renderAnalysis(findings))
	action := MutationUpdate
	if body == document.Body {
		action = MutationUnchanged
	}
	document.Body = body
	return SpecAnalysisReport{SpecPath: document.Path, Findings: findings, Action: action, Document: document}, nil
}

// AnalyzeSpec persists a confirmed additive analysis report.
func (s *Service) AnalyzeSpec(ctx context.Context, id planning.ArtifactID, confirmed bool, authorizer Authorizer, events EventSink) (SpecAnalysisReport, error) {
	report, err := s.PreviewSpecAnalysis(ctx, id)
	if err != nil || report.Action == MutationUnchanged {
		return report, err
	}
	result, err := s.applySpecDocument(ctx, report.Document, confirmed, PermissionSpecEdit, EventSpecUpdated, authorizer, events)
	report.Action, report.Document, report.Event = result.Action, result.Document, result.Event
	return report, err
}

// PreviewSpecChecklist computes and embeds one additive checklist profile without writing.
func (s *Service) PreviewSpecChecklist(ctx context.Context, id planning.ArtifactID, profile string) (SpecChecklistReport, error) {
	if !containsString(checklistProfiles, profile) {
		return SpecChecklistReport{}, fmt.Errorf("invalid checklist profile %q", profile)
	}
	document, err := s.specForMutation(ctx, id)
	if err != nil {
		return SpecChecklistReport{}, err
	}
	findings := checklistSpecDocument(document, profile)
	existing := extractMarkdownSection(document.Body, "Checklist")
	next := setMarkdownSubsection(existing, profile, renderChecklist(profile, findings))
	body := setMarkdownSection(document.Body, "Checklist", next)
	action := MutationUpdate
	if body == document.Body {
		action = MutationUnchanged
	}
	document.Body = body
	return SpecChecklistReport{SpecPath: document.Path, Profile: profile, Findings: findings, Action: action, Document: document}, nil
}

// RunSpecChecklist persists one confirmed additive checklist profile.
func (s *Service) RunSpecChecklist(ctx context.Context, id planning.ArtifactID, profile string, confirmed bool, authorizer Authorizer, events EventSink) (SpecChecklistReport, error) {
	report, err := s.PreviewSpecChecklist(ctx, id, profile)
	if err != nil || report.Action == MutationUnchanged {
		return report, err
	}
	result, err := s.applySpecDocument(ctx, report.Document, confirmed, PermissionSpecEdit, EventSpecUpdated, authorizer, events)
	report.Action, report.Document, report.Event = result.Action, result.Document, result.Event
	return report, err
}

// PreviewSpecExecution derives deterministic ephemeral slices without mutation.
func (s *Service) PreviewSpecExecution(ctx context.Context, input SpecExecutionInput) (SpecExecutionResult, error) {
	document, err := s.specForMutation(ctx, input.ID)
	if err != nil {
		return SpecExecutionResult{}, err
	}
	executionID := planning.ArtifactID(string(input.ID) + "-execution")
	executionInput := planning.ExecutionInput{ID: executionID, ExplicitCandidates: executionCandidates(document.Body), FlowCandidates: flowCandidates(document.Body), DefaultVerification: document.Artifact.Verification}
	implementing, plan, err := planning.BeginExecution(document.Artifact, executionInput)
	if err != nil {
		return SpecExecutionResult{}, err
	}
	action := MutationUpdate
	if document.Artifact.Status == planning.SpecImplementing {
		action = MutationUnchanged
	}
	document.Artifact = implementing
	prefix := strings.TrimSpace(input.BranchPrefix)
	if prefix == "" {
		prefix = "feature/"
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	slices := make([]SpecExecutionSlice, 0, len(plan.Slices))
	for _, slice := range plan.Slices {
		slices = append(slices, SpecExecutionSlice{ID: slice.ID, Title: slice.Title, Goal: slice.Goal, Verification: append([]string(nil), slice.Verification...), Position: slice.Position})
	}
	view := SpecExecutionView{SpecPath: document.Path, Status: implementing.Status, SuggestedBranch: prefix + string(input.ID), Slices: slices}
	return SpecExecutionResult{Action: action, Document: document, Execution: view}, nil
}

// BeginSpecExecution applies a confirmed execution transition without persisting slice artifacts.
func (s *Service) BeginSpecExecution(ctx context.Context, input SpecExecutionInput, authorizer Authorizer, events EventSink) (SpecExecutionResult, error) {
	preview, err := s.PreviewSpecExecution(ctx, input)
	if err != nil || preview.Action == MutationUnchanged {
		return preview, err
	}
	result, err := s.applySpecDocument(ctx, preview.Document, input.Confirmed, PermissionExecute, EventSpecExecutionStarted, authorizer, events)
	preview.Action, preview.Document, preview.Event = result.Action, result.Document, result.Event
	return preview, err
}

// PreviewSpecHandoff derives execution and guided-session transitions without mutation.
func (s *Service) PreviewSpecHandoff(ctx context.Context, input SpecExecutionInput) (SpecExecutionResult, error) {
	result, err := s.PreviewSpecExecution(ctx, input)
	if err != nil {
		return SpecExecutionResult{}, err
	}
	state, err := s.repository.ReadGuidedSessions(ctx)
	if err != nil {
		return SpecExecutionResult{}, err
	}
	var record GuidedSessionRecord
	found := false
	for _, candidate := range state.Sessions {
		if candidate.Spec == string(input.ID) {
			record = candidate
			found = true
			break
		}
	}
	if !found {
		return SpecExecutionResult{}, fmt.Errorf("no guided session linked to spec %q", input.ID)
	}
	result.Recap = record.Summary
	if record.StageStatuses == nil {
		record.StageStatuses = map[string]string{}
	}
	if record.CurrentStage != "execution" || record.StageStatuses["spec"] != "done" || record.StageStatuses["execution"] != "in_progress" {
		result.Action = MutationUpdate
	}
	record.CurrentStage = "execution"
	record.StageStatuses["spec"] = "done"
	record.StageStatuses["execution"] = "in_progress"
	record.Summary = "Spec handoff complete. Continue the planning flow in the execution stage."
	record.NextAction = "Work through the execution slices one commit at a time until the spec is delivered."
	result.Session = &record
	return result, nil
}

// HandoffSpec atomically-compensates the confirmed spec/session transition on partial persistence failure.
func (s *Service) HandoffSpec(ctx context.Context, input SpecExecutionInput, authorizer Authorizer, events EventSink) (SpecExecutionResult, error) {
	preview, err := s.PreviewSpecHandoff(ctx, input)
	if err != nil {
		return SpecExecutionResult{}, err
	}
	if preview.Action == MutationUnchanged {
		return preview, nil
	}
	if !input.Confirmed {
		return SpecExecutionResult{}, ErrConfirmationRequired
	}
	if authorizer == nil {
		return SpecExecutionResult{}, fmt.Errorf("planning mutation requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionExecute); err != nil {
		return SpecExecutionResult{}, err
	}
	if events == nil {
		return SpecExecutionResult{}, ErrEventSinkRequired
	}
	originalSpec, err := s.repository.GetSpec(ctx, input.ID)
	if err != nil {
		return SpecExecutionResult{}, err
	}
	state, err := s.repository.ReadGuidedSessions(ctx)
	if err != nil {
		return SpecExecutionResult{}, err
	}
	record := *preview.Session
	now := s.now().UTC()
	timestamp := now.Format("2006-01-02T15:04:05Z07:00")
	record.UpdatedAt = timestamp
	state.LastActiveChain = record.ChainID
	state.LastUpdatedAt = timestamp
	state.Sessions[record.ChainID] = record
	updatedSpec := originalSpec
	specAction := MutationUnchanged
	if preview.Document.Artifact.Status != originalSpec.Artifact.Status {
		updatedSpec, specAction, err = s.repository.ReplaceSpec(ctx, preview.Document, now)
		if err != nil {
			return SpecExecutionResult{}, err
		}
	}
	updatedState, sessionAction, sessionErr := s.repository.ReplaceGuidedSessions(ctx, state)
	if sessionErr != nil {
		if specAction != MutationUnchanged {
			if rollbackErr := s.repository.RollbackSpecReplacement(ctx, updatedSpec, originalSpec); rollbackErr != nil {
				return SpecExecutionResult{}, errors.Join(sessionErr, fmt.Errorf("rollback spec handoff: %w", rollbackErr))
			}
		}
		return SpecExecutionResult{}, sessionErr
	}
	preview.Document = updatedSpec
	record = updatedState.Sessions[record.ChainID]
	preview.Session = &record
	preview.Action = specAction
	if preview.Action == MutationUnchanged {
		preview.Action = sessionAction
	}
	if preview.Action == MutationUnchanged {
		return preview, nil
	}
	event := Event{Name: EventSpecExecutionStarted, ModuleID: s.moduleID, Artifact: planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: input.ID}, Outcome: preview.Action, OccurredAt: now}
	if err := events.Publish(ctx, event); err != nil {
		return SpecExecutionResult{}, fmt.Errorf("publish planning event: %w", err)
	}
	preview.Event = &event
	return preview, nil
}

func (s *Service) specForMutation(ctx context.Context, id planning.ArtifactID) (SpecDocument, error) {
	if err := s.requireWritable(ctx); err != nil {
		return SpecDocument{}, err
	}
	if err := id.Validate(); err != nil {
		return SpecDocument{}, err
	}
	document, err := s.repository.GetSpec(ctx, id)
	if document.Metadata == nil {
		document.Metadata = map[string]any{}
	}
	return document, err
}

func (s *Service) applySpecDocument(ctx context.Context, document SpecDocument, confirmed bool, permission, eventName string, authorizer Authorizer, events EventSink) (SpecMutationResult, error) {
	if !confirmed {
		return SpecMutationResult{}, ErrConfirmationRequired
	}
	if authorizer == nil {
		return SpecMutationResult{}, fmt.Errorf("planning mutation requires an authorizer")
	}
	if err := authorizer.Require(ctx, permission); err != nil {
		return SpecMutationResult{}, err
	}
	if events == nil {
		return SpecMutationResult{}, ErrEventSinkRequired
	}
	now := s.now().UTC()
	updated, action, err := s.repository.ReplaceSpec(ctx, document, now)
	if err != nil {
		return SpecMutationResult{}, err
	}
	result := SpecMutationResult{Action: action, Document: updated}
	if action == MutationUnchanged {
		return result, nil
	}
	event := Event{Name: eventName, ModuleID: s.moduleID, Artifact: planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: updated.Artifact.ID}, Outcome: action, OccurredAt: now}
	if err := events.Publish(ctx, event); err != nil {
		return SpecMutationResult{}, fmt.Errorf("publish planning event: %w", err)
	}
	result.Event = &event
	return result, nil
}

func sameSpecDocument(a, b SpecDocument) bool {
	return a.Body == b.Body && reflect.DeepEqual(a.Metadata, b.Metadata) && reflect.DeepEqual(a.Artifact, b.Artifact)
}
func markdownItems(section string) []string {
	var out []string
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
func hasPlanningErrors(findings []planning.Finding) bool {
	for _, f := range findings {
		if f.Severity == planning.SeverityError {
			return true
		}
	}
	return false
}
func planningFindingMessages(findings []planning.Finding) string {
	var out []string
	for _, f := range findings {
		if f.Severity == planning.SeverityError {
			out = append(out, f.Message)
		}
	}
	return strings.Join(out, "; ")
}
func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func analyzeSpecDocument(document SpecDocument) []SpecAnalysisFinding {
	var out []SpecAnalysisFinding
	for _, f := range checkSpecDocument(SpecQueryDocument{ID: document.Artifact.ID, Title: document.Artifact.Title, Path: document.Path, Body: document.Body}) {
		category := "Recommended Revisions"
		if strings.Contains(f.Rule, "constraints") {
			category = "Missing Constraints"
		}
		if strings.Contains(f.Rule, "goals") || strings.Contains(f.Rule, "verification") {
			category = "Success Criteria Gaps"
		}
		out = append(out, SpecAnalysisFinding{Severity: f.Severity, Category: category, Message: f.Message, Recommendation: f.Suggestion})
	}
	if sectionLooksThin(extractMarkdownSection(document.Body, "Risks / Open Questions")) {
		out = append(out, SpecAnalysisFinding{Severity: "warn", Category: "Risk Gaps", Message: "Risks / Open Questions is present but too thin to pressure-test the spec.", Recommendation: "Expand ## Risks / Open Questions with concrete failure modes, ambiguity, and boundary risks."})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return indexOf(analysisCategories, out[i].Category) < indexOf(analysisCategories, out[j].Category)
	})
	return out
}
func renderAnalysis(findings []SpecAnalysisFinding) string {
	if len(findings) == 0 {
		return "### Missing Constraints\n\n### Success Criteria Gaps\n\n### Hidden Dependencies\n\n### Risk Gaps\n\n### What/Why vs How Leakage\n\n### Recommended Revisions\n\n- No findings."
	}
	var blocks []string
	for _, category := range analysisCategories {
		var lines []string
		for _, f := range findings {
			if f.Category == category {
				lines = append(lines, fmt.Sprintf("- [%s] %s", f.Severity, f.Message))
				if f.Recommendation != "" {
					lines = append(lines, "  fix: "+f.Recommendation)
				}
			}
		}
		blocks = append(blocks, "### "+category+"\n\n"+strings.Join(lines, "\n"))
	}
	return strings.Join(blocks, "\n\n")
}
func checklistSpecDocument(document SpecDocument, profile string) []SpecChecklistFinding {
	var out []SpecChecklistFinding
	if profile == "general" {
		for _, pair := range []struct {
			h     string
			block bool
		}{{"Goals", false}, {"Non-Goals", false}, {"Solution Shape", false}, {"Verification", true}} {
			if sectionLooksThin(extractMarkdownSection(document.Body, pair.h)) {
				sev := "warn"
				if pair.block {
					sev = "error"
				}
				out = append(out, SpecChecklistFinding{Severity: sev, Area: pair.h, Message: "The " + profile + " checklist expects concrete content under ## " + pair.h + ".", Recommendation: "Expand ## " + pair.h + " with execution-ready detail."})
			}
		}
	} else if profile == "ui-flow" {
		flows := strings.ToLower(extractMarkdownSection(document.Body, "Flows"))
		if sectionLooksThin(flows) {
			out = append(out, SpecChecklistFinding{Severity: "error", Area: "Flows", Message: "The UI flow checklist requires a concrete user journey under ## Flows.", Recommendation: "Describe the primary UI journey step by step under ## Flows."})
		}
		if !containsAnyText(flows, []string{"screen", "page", "modal", "form", "click", "select", "submit", "view"}) {
			out = append(out, SpecChecklistFinding{Severity: "warn", Area: "Flows", Message: "The UI flow checklist expects user-facing interaction detail in ## Flows.", Recommendation: "Call out user-visible screens, inputs, or transitions."})
		}
	} else if profile == "api-integration" {
		interfaces := strings.ToLower(extractMarkdownSection(document.Body, "Data / Interfaces"))
		if !containsAnyText(interfaces, []string{"api", "endpoint", "http", "webhook", "oauth", "external", "integration", "contract"}) {
			out = append(out, SpecChecklistFinding{Severity: "error", Area: "Data / Interfaces", Message: "The API integration checklist requires the external contract under ## Data / Interfaces.", Recommendation: "Document endpoint, payload, auth, or contract details."})
		}
		risks := strings.ToLower(extractMarkdownSection(document.Body, "Risks / Open Questions"))
		if !containsAnyText(risks, []string{"timeout", "retry", "failure", "auth", "ownership", "contract", "rate", "limit", "webhook"}) {
			out = append(out, SpecChecklistFinding{Severity: "warn", Area: "Risks / Open Questions", Message: "The API integration checklist expects dependency and failure-handling risks.", Recommendation: "Add external-contract, auth, timeout, retry, or ownership risks."})
		}
	} else if profile == "data-migration" {
		interfaces := strings.ToLower(extractMarkdownSection(document.Body, "Data / Interfaces"))
		if !containsAnyText(interfaces, []string{"migration", "schema", "table", "column", "backfill", "data", "database"}) {
			out = append(out, SpecChecklistFinding{Severity: "error", Area: "Data / Interfaces", Message: "The data migration checklist requires the schema or data-shape change under ## Data / Interfaces.", Recommendation: "Document migration, backfill, or data-shape changes."})
		}
		rollout := strings.ToLower(extractMarkdownSection(document.Body, "Rollout"))
		if !containsAnyText(rollout, []string{"rollback", "backfill", "dual", "compat", "reversible", "dry-run", "preview"}) {
			out = append(out, SpecChecklistFinding{Severity: "warn", Area: "Rollout", Message: "The data migration checklist expects rollback or compatibility safety.", Recommendation: "Describe rollback, preview, compatibility, or reversible rollout behavior."})
		}
	}
	return out
}
func containsAnyText(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
func renderChecklist(profile string, findings []SpecChecklistFinding) string {
	if len(findings) == 0 {
		return "profile: " + profile + "\nstatus: ok"
	}
	var lines []string
	for _, f := range findings {
		lines = append(lines, fmt.Sprintf("- [%s] %s: %s", f.Severity, f.Area, f.Message))
		if f.Recommendation != "" {
			lines = append(lines, "  fix: "+f.Recommendation)
		}
	}
	return strings.Join(lines, "\n")
}
func indexOf(values []string, value string) int {
	for i, v := range values {
		if v == value {
			return i
		}
	}
	return len(values)
}

func executionCandidates(body string) []planning.SliceCandidate {
	section := extractMarkdownSection(body, "Execution Plan")
	var out []planning.SliceCandidate
	var current *planning.SliceCandidate
	for _, raw := range strings.Split(section, "\n") {
		if strings.HasPrefix(raw, "- ") || strings.HasPrefix(raw, "* ") {
			title := executionCandidateTitle(raw)
			if strings.EqualFold(title, "Define execution slices when implementation begins") || strings.EqualFold(title, "Split approved spec into execution-ready stories") {
				current = nil
				continue
			}
			out = append(out, planning.SliceCandidate{Title: title})
			current = &out[len(out)-1]
			continue
		}
		if current == nil || !(strings.HasPrefix(raw, "  ") || strings.HasPrefix(raw, "\t")) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "- "))
		parts := strings.SplitN(value, ":", 2)
		if len(parts) != 2 {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(parts[0])) {
		case "description", "desc":
			current.Goal = strings.TrimSpace(parts[1])
		case "accept", "criteria", "criterion":
			if current.Goal == "" {
				current.Goal = strings.TrimSpace(parts[1])
			}
		case "verify", "verification":
			current.Verification = append(current.Verification, strings.TrimSpace(parts[1]))
		}
	}
	for index := range out {
		if strings.TrimSpace(out[index].Goal) == "" {
			out[index].Goal = fmt.Sprintf("Complete the %q slice described by the canonical spec.", out[index].Title)
		}
	}
	return out
}
func executionCandidateTitle(line string) string {
	line = strings.TrimSpace(line)
	for _, prefix := range []string{"- [ ] ", "- [x] ", "- ", "* "} {
		line = strings.TrimPrefix(line, prefix)
	}
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "[") {
		if end := strings.Index(line, "]("); end > 1 {
			return strings.TrimSpace(line[1:end])
		}
	}
	return line
}
func flowCandidates(body string) []planning.SliceCandidate {
	var out []planning.SliceCandidate
	for _, line := range markdownItems(extractMarkdownSection(body, "Flows")) {
		line = strings.TrimSpace(strings.TrimLeft(line, "0123456789. "))
		if line != "" {
			out = append(out, planning.SliceCandidate{Title: line})
		}
		if len(out) == 3 {
			break
		}
	}
	return out
}
