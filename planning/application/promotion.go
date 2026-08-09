package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/JimmyMcBride/brain/planning"
)

// AssessLocalBrainstorm returns a deterministic local maturity decision.
func (s *Service) AssessLocalBrainstorm(ctx context.Context, id planning.ArtifactID) (CollaborationAssessment, error) {
	if err := s.requireReadable(ctx); err != nil {
		return CollaborationAssessment{}, err
	}
	document, err := s.repository.GetBrainstorm(ctx, id)
	if err != nil {
		return CollaborationAssessment{}, err
	}
	decision := assessBrainstorm(document)
	return CollaborationAssessment{
		SchemaVersion: 1,
		Kind:          "maturity_assessment",
		GeneratedAt:   s.now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		Source:        localPromotionSource(document),
		Ownership:     localOwnership(),
		Decision:      decision,
	}, nil
}

// PreviewLocalPromotionRepair returns an explicit multi-spec source repair without writing.
func (s *Service) PreviewLocalPromotionRepair(ctx context.Context, input LocalPromotionRepairInput) (LocalPromotionRepairResult, error) {
	result, _, _, err := s.previewLocalPromotionRepair(ctx, input)
	return result, err
}

// RepairLocalPromotionSource applies an explicit multi-spec split to a local brainstorm.
func (s *Service) RepairLocalPromotionSource(ctx context.Context, input LocalPromotionRepairInput, authorizer Authorizer, events EventSink) (LocalPromotionRepairResult, error) {
	result, document, body, err := s.previewLocalPromotionRepair(ctx, input)
	if err != nil || result.Action == MutationUnchanged {
		return result, err
	}
	mutation, err := s.applyBrainstormBody(ctx, input.BrainstormID, body, input.Confirmed, authorizer, events)
	if err != nil {
		return LocalPromotionRepairResult{}, err
	}
	result.Source = localPromotionSource(document)
	result.Action = mutation.Action
	result.Event = mutation.Event
	return result, nil
}

func (s *Service) previewLocalPromotionRepair(ctx context.Context, input LocalPromotionRepairInput) (LocalPromotionRepairResult, BrainstormDocument, string, error) {
	if len(input.Specs) < 2 {
		return LocalPromotionRepairResult{}, BrainstormDocument{}, "", fmt.Errorf("repair requires at least two spec titles")
	}
	document, err := s.brainstormForMutation(ctx, input.BrainstormID)
	if err != nil {
		return LocalPromotionRepairResult{}, BrainstormDocument{}, "", err
	}
	seen := map[string]struct{}{}
	var specs []string
	for _, spec := range input.Specs {
		spec = strings.TrimSpace(spec)
		key := strings.ToLower(spec)
		if spec == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		specs = append(specs, spec)
	}
	if len(specs) < 2 {
		return LocalPromotionRepairResult{}, BrainstormDocument{}, "", fmt.Errorf("repair requires at least two distinct spec titles")
	}
	lines := make([]string, 0, len(specs))
	for _, spec := range specs {
		lines = append(lines, "- "+spec)
	}
	body := setMarkdownSection(document.Body, "Specs", strings.Join(lines, "\n"))
	result := LocalPromotionRepairResult{
		SchemaVersion: 1, Kind: "source_repair", GeneratedAt: s.now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		Source: localPromotionSource(document), Specs: specs, UpdatedPath: document.Path,
		NextCommands: []string{"brain plan brainstorm assess " + string(input.BrainstormID) + " --json", "brain plan brainstorm promote " + string(input.BrainstormID) + " --json"},
		Action:       MutationUpdate,
	}
	if body == document.Body {
		result.Action = MutationUnchanged
		return result, document, body, nil
	}
	return result, document, body, nil
}

// PreviewLocalPromotion returns direct local spec actions without mutation.
func (s *Service) PreviewLocalPromotion(ctx context.Context, id planning.ArtifactID) (LocalPromotionDraft, error) {
	assessment, err := s.AssessLocalBrainstorm(ctx, id)
	if err != nil {
		return LocalPromotionDraft{}, err
	}
	document, err := s.repository.GetBrainstorm(ctx, id)
	if err != nil {
		return LocalPromotionDraft{}, err
	}
	draft := LocalPromotionDraft{
		SchemaVersion: 1, Kind: "promotion_draft", GeneratedAt: s.now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		Source: assessment.Source, Ownership: assessment.Ownership, Assessment: assessment.Decision,
		ProposedSpecs: []PromotionSpecDraft{},
		AgentPolicy: map[string]any{
			"allowed_mutations":   []string{"brain plan brainstorm promote --confirm"},
			"forbidden_mutations": []string{"gh issue create", "gh label create", "gh milestone create"},
			"manual_fallback":     "only after the Planning host reports an explicit fallback",
		},
		ManualFallbackAllowed: false, ConfirmationRequired: true,
	}
	if assessment.Decision.State != "ready_single_spec" && assessment.Decision.State != "ready_multi_spec" {
		return draft, nil
	}
	draft.PromotionDecision = assessment.Decision.RecommendedPath
	draft.WhyThisPath = assessment.Decision.Reason
	titles, _ := assessment.Decision.SuggestedTitles["specs"].([]string)
	slugs := make(map[string]string, len(titles))
	for _, title := range titles {
		specDraft := buildPromotionSpecDraft(document, title)
		if err := planning.ArtifactID(specDraft.Slug).Validate(); err != nil {
			return LocalPromotionDraft{}, fmt.Errorf("invalid promoted spec title %q: %w", title, err)
		}
		if prior, exists := slugs[specDraft.Slug]; exists {
			return LocalPromotionDraft{}, fmt.Errorf("promoted spec titles %q and %q resolve to the same slug %q", prior, title, specDraft.Slug)
		}
		slugs[specDraft.Slug] = title
		existing, found, err := s.repository.FindSpec(ctx, planning.ArtifactID(specDraft.Slug))
		if err != nil {
			return LocalPromotionDraft{}, err
		}
		if found {
			source := strings.TrimSpace(stringMetadata(existing.Metadata, "source_brainstorm"))
			switch {
			case source != "" && source != document.Path:
				specDraft.Action = MutationReuse
			case existing.Body == canonicalSpecBody(specDraft):
				specDraft.Action = MutationUnchanged
			default:
				specDraft.Action = MutationUpdate
			}
		}
		draft.ProposedSpecs = append(draft.ProposedSpecs, specDraft)
	}
	return draft, nil
}

// PromoteLocalBrainstorm applies confirmed direct local spec actions without an epic intermediary.
func (s *Service) PromoteLocalBrainstorm(ctx context.Context, input LocalPromotionInput, authorizer Authorizer, events EventSink) (LocalPromotionResult, error) {
	draft, err := s.PreviewLocalPromotion(ctx, input.BrainstormID)
	if err != nil {
		return LocalPromotionResult{}, err
	}
	if draft.Assessment.State != "ready_single_spec" && draft.Assessment.State != "ready_multi_spec" {
		return LocalPromotionResult{}, fmt.Errorf("brainstorm is not ready for promotion: %s", strings.Join(draft.Assessment.Gaps, ", "))
	}
	var writes []PromotionSpecWrite
	var existing []SpecDocument
	brainstormPath, ok := draft.Source["brainstorm_path"].(string)
	if !ok || strings.TrimSpace(brainstormPath) == "" {
		return LocalPromotionResult{}, fmt.Errorf("promotion draft is missing brainstorm source path")
	}
	for _, specDraft := range draft.ProposedSpecs {
		switch specDraft.Action {
		case MutationReuse, MutationUnchanged:
			document, err := s.repository.GetSpec(ctx, planning.ArtifactID(specDraft.Slug))
			if err != nil {
				return LocalPromotionResult{}, err
			}
			existing = append(existing, document)
		default:
			artifact := planning.Spec{
				ID: planning.ArtifactID(specDraft.Slug), Title: specDraft.Title, Status: planning.SpecDraft,
				Approval:     planning.Approval{State: planning.ApprovalPending},
				Verification: []string{"Add automated coverage for the promoted behavior.", "Verify the CLI output and local artifact contract for this slice."},
				Sources:      []planning.SourceReference{{URI: brainstormPath, Purpose: "promotion_source"}},
			}
			writes = append(writes, PromotionSpecWrite{
				Artifact: artifact, Body: canonicalSpecBody(specDraft),
				Metadata: map[string]any{"source_brainstorm": brainstormPath},
			})
		}
	}
	if len(writes) == 0 {
		return LocalPromotionResult{Draft: draft, Specs: existing, Action: MutationUnchanged}, nil
	}
	if !input.Confirmed {
		return LocalPromotionResult{}, ErrConfirmationRequired
	}
	if authorizer == nil {
		return LocalPromotionResult{}, fmt.Errorf("planning mutation requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionBrainstorm); err != nil {
		return LocalPromotionResult{}, err
	}
	if events == nil {
		return LocalPromotionResult{}, ErrEventSinkRequired
	}
	now := s.now().UTC()
	documents, action, err := s.repository.WritePromotionSpecs(ctx, writes, now)
	if err != nil {
		return LocalPromotionResult{}, err
	}
	documents = append(existing, documents...)
	sortSpecDocuments(documents)
	if err := s.linkGuidedPromotion(ctx, input.BrainstormID, documents, now.Format("2006-01-02T15:04:05Z07:00")); err != nil {
		return LocalPromotionResult{}, err
	}
	result := LocalPromotionResult{Draft: draft, Specs: documents, Action: action}
	if action == MutationUnchanged {
		return result, nil
	}
	event := Event{Name: EventBrainstormPromoted, ModuleID: s.moduleID, Artifact: planning.ArtifactRef{Kind: planning.ArtifactBrainstorm, ID: input.BrainstormID}, Outcome: action, OccurredAt: now}
	if err := events.Publish(ctx, event); err != nil {
		return LocalPromotionResult{}, fmt.Errorf("publish planning event: %w", err)
	}
	result.Event = &event
	return result, nil
}

func assessBrainstorm(document BrainstormDocument) MaturityDecision {
	refinement := extractMarkdownSection(document.Body, "Refinement")
	challenge := extractMarkdownSection(document.Body, "Challenge")
	values := []struct {
		value    string
		strength string
		gap      string
	}{
		{firstNonEmpty(extractMarkdownSection(refinement, "Problem"), extractMarkdownSection(document.Body, "Focus Question")), "clear problem", "missing concrete problem statement"},
		{firstNonEmpty(extractMarkdownSection(refinement, "User / Value"), extractMarkdownSection(document.Body, "Desired Outcome"), extractMarkdownSection(document.Body, "Vision")), "clear goals or user value", "missing goals or user value"},
		{extractMarkdownSection(document.Body, "Constraints"), "clear constraints", "missing constraints"},
		{extractMarkdownSection(challenge, "No-Gos"), "non-goals are called out", "missing non-goals"},
		{firstNonEmpty(extractMarkdownSection(refinement, "Decision Snapshot"), extractMarkdownSection(refinement, "Candidate Approaches"), extractMarkdownSection(challenge, "Simpler Alternative")), "solution shape is visible", "missing initial solution shape"},
	}
	decision := MaturityDecision{State: "not_ready", Confidence: "low", SourceMode: "local", Reason: "The source still has planning gaps that would force promotion to guess.", SuggestedTitles: map[string]any{}}
	for _, value := range values {
		if strings.TrimSpace(value.value) == "" {
			decision.Gaps = append(decision.Gaps, value.gap)
		} else {
			decision.Strengths = append(decision.Strengths, value.strength)
		}
	}
	if len(decision.Gaps) > 0 {
		return decision
	}
	titles := promotionTitles(document)
	decision.Confidence = "high"
	decision.DependencyGuess = make([]map[string]any, 0, len(titles))
	for _, title := range titles {
		decision.DependencyGuess = append(decision.DependencyGuess, map[string]any{"spec": title, "blocked_by": []string{}})
	}
	if len(titles) == 1 {
		decision.State = "ready_single_spec"
		decision.RecommendedPath = "single_spec"
		decision.Reason = "The source is clear enough to promote directly into one bounded spec."
		decision.SuggestedTitles["specs"] = titles
		return decision
	}
	decision.State = "ready_multi_spec"
	decision.RecommendedPath = "multi_spec"
	decision.Reason = fmt.Sprintf("The source is clear enough to promote directly into %d bounded specs.", len(titles))
	decision.SuggestedTitles["initiative"] = document.Artifact.Title
	decision.SuggestedTitles["specs"] = titles
	return decision
}

func buildPromotionSpecDraft(document BrainstormDocument, title string) PromotionSpecDraft {
	refinement := extractMarkdownSection(document.Body, "Refinement")
	challenge := extractMarkdownSection(document.Body, "Challenge")
	problem := firstNonEmpty(extractMarkdownSection(refinement, "Problem"), extractMarkdownSection(document.Body, "Focus Question"))
	goals := firstNonEmpty(extractMarkdownSection(refinement, "User / Value"), extractMarkdownSection(document.Body, "Desired Outcome"), extractMarkdownSection(document.Body, "Vision"))
	nonGoals := firstNonEmpty(extractMarkdownSection(challenge, "No-Gos"), "- No scope outside this promoted spec.")
	constraints := extractMarkdownSection(document.Body, "Constraints")
	shape := firstNonEmpty(extractMarkdownSection(refinement, "Decision Snapshot"), extractMarkdownSection(refinement, "Candidate Approaches"), extractMarkdownSection(challenge, "Simpler Alternative"))
	body := strings.Join([]string{
		"## Spec", problem, "", "## Problem", problem, "", "## Goals", goals, "", "## Non-Goals", nonGoals, "",
		"## Constraints", constraints, "", "## Proposed Shape", shape, "", "## Verification",
		"- Add automated coverage for the promoted behavior.", "- Verify the CLI output and local artifact contract for this slice.", "",
		"## Dependencies", "- blocked by: none", "", "## Readiness", "- status: ready", "", "## Source", "- " + document.Path,
	}, "\n")
	return PromotionSpecDraft{
		Kind: "spec", Title: title, Body: body, Slug: slugify(title), Action: MutationCreate, Readiness: "ready",
		Labels: []string{"enhancement", "plan:spec", "plan:ready"}, SourceLinks: []string{document.Path}, ReadyByDefault: true,
	}
}

func canonicalSpecBody(draft PromotionSpecDraft) string {
	return strings.Join([]string{
		"# " + draft.Title, "", "## Why", extractMarkdownSection(draft.Body, "Goals"), "", "## Problem", extractMarkdownSection(draft.Body, "Problem"), "",
		"## Goals", extractMarkdownSection(draft.Body, "Goals"), "", "## Non-Goals", extractMarkdownSection(draft.Body, "Non-Goals"), "",
		"## Constraints", extractMarkdownSection(draft.Body, "Constraints"), "", "## Solution Shape", extractMarkdownSection(draft.Body, "Proposed Shape"), "",
		"## Verification", "- Add automated coverage for the promoted behavior.", "- Verify the CLI output and local artifact contract for this slice.", "",
		"## Execution Plan", "- [ ] Derive runtime slices when implementation begins.", "", "## Resources", "- [Source Brainstorm](../brainstorms/" + strings.TrimSuffix(strings.TrimPrefix(draft.SourceLinks[0], ".plan/brainstorms/"), ".md") + ".md)",
	}, "\n") + "\n"
}

func localPromotionSource(document BrainstormDocument) map[string]any {
	return map[string]any{
		"mode": "local", "entry_mode": "local_promotion", "brainstorm_slug": string(document.Artifact.ID),
		"brainstorm_path": document.Path, "source_links": []string{document.Path}, "canonical_source": "local_brainstorm",
	}
}

func promotionTitles(document BrainstormDocument) []string {
	var titles []string
	seen := map[string]struct{}{}
	for _, line := range strings.Split(extractMarkdownSection(document.Body, "Specs"), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		key := strings.ToLower(line)
		if line == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		titles = append(titles, line)
	}
	if len(titles) == 0 {
		titles = []string{document.Artifact.Title}
	}
	return titles
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func stringMetadata(metadata map[string]any, key string) string {
	value, _ := metadata[key].(string)
	return value
}

func sortSpecDocuments(documents []SpecDocument) {
	for index := 1; index < len(documents); index++ {
		for cursor := index; cursor > 0 && documents[cursor].Artifact.ID < documents[cursor-1].Artifact.ID; cursor-- {
			documents[cursor], documents[cursor-1] = documents[cursor-1], documents[cursor]
		}
	}
}

func (s *Service) linkGuidedPromotion(ctx context.Context, brainstormID planning.ArtifactID, specs []SpecDocument, timestamp string) error {
	state, err := s.repository.ReadGuidedSessions(ctx)
	if err != nil {
		return err
	}
	chainID := "brainstorm/" + string(brainstormID)
	record, exists := state.Sessions[chainID]
	if !exists {
		return nil
	}
	if len(specs) == 1 {
		record.Spec = string(specs[0].Artifact.ID)
	} else {
		record.Spec = ""
	}
	record.CurrentStage = "spec"
	if record.StageStatuses == nil {
		record.StageStatuses = map[string]string{}
	}
	record.StageStatuses["brainstorm"] = "done"
	record.StageStatuses["spec"] = "in_progress"
	record.Summary = "Brainstorm promoted directly into canonical local spec work."
	record.NextAction = "Review and approve the promoted spec before execution."
	record.UpdatedAt = timestamp
	state.LastActiveChain = chainID
	state.LastUpdatedAt = timestamp
	state.Sessions[chainID] = record
	_, _, err = s.repository.ReplaceGuidedSessions(ctx, state)
	return err
}
