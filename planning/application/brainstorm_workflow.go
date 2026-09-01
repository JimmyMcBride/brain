package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/JimmyMcBride/brain/planning"
)

var brainstormSections = map[string]struct {
	heading string
	list    bool
}{
	"ideas":               {heading: "Ideas", list: true},
	"focus-question":      {heading: "Focus Question"},
	"desired-outcome":     {heading: "Desired Outcome"},
	"constraints":         {heading: "Constraints", list: true},
	"open-questions":      {heading: "Open Questions", list: true},
	"raw-notes":           {heading: "Raw Notes"},
	"supporting-material": {heading: "Supporting Material", list: true},
}

// PreviewBrainstormUpdate returns one append-only brainstorm mutation without writing.
func (s *Service) PreviewBrainstormUpdate(ctx context.Context, input BrainstormUpdateInput) (BrainstormMutationResult, error) {
	if err := s.requireWritable(ctx); err != nil {
		return BrainstormMutationResult{}, err
	}
	if err := input.ID.Validate(); err != nil {
		return BrainstormMutationResult{}, err
	}
	spec, ok := brainstormSections[strings.ToLower(strings.TrimSpace(input.Section))]
	if !ok {
		return BrainstormMutationResult{}, fmt.Errorf("unsupported brainstorm section %q", input.Section)
	}
	value := strings.TrimSpace(input.Body)
	if value == "" {
		return BrainstormMutationResult{}, fmt.Errorf("brainstorm entry is required")
	}
	if spec.list {
		value = normalizeBulletList(value)
	}
	document, err := s.repository.GetBrainstorm(ctx, input.ID)
	if err != nil {
		return BrainstormMutationResult{}, err
	}
	body, changed := appendUniqueSection(document.Body, spec.heading, value)
	action := MutationUpdate
	if !changed {
		action = MutationUnchanged
	}
	document.Body = body
	return BrainstormMutationResult{Action: action, Document: document}, nil
}

// UpdateBrainstorm applies one confirmed, authorized, audited brainstorm append.
func (s *Service) UpdateBrainstorm(ctx context.Context, input BrainstormUpdateInput, authorizer Authorizer, events EventSink) (BrainstormMutationResult, error) {
	preview, err := s.PreviewBrainstormUpdate(ctx, input)
	if err != nil || preview.Action == MutationUnchanged {
		return preview, err
	}
	return s.applyBrainstormBody(ctx, input.ID, preview.Document.Body, input.Confirmed, authorizer, events)
}

// PreviewBrainstormRefinement returns the merged structured refinement without writing.
func (s *Service) PreviewBrainstormRefinement(ctx context.Context, input BrainstormRefinementInput) (BrainstormMutationResult, error) {
	document, err := s.brainstormForMutation(ctx, input.ID)
	if err != nil {
		return BrainstormMutationResult{}, err
	}
	values := map[string]string{
		"Problem":                  input.Problem,
		"User / Value":             input.UserValue,
		"Appetite":                 input.Appetite,
		"Remaining Open Questions": normalizeBulletList(input.RemainingOpenQuestions),
		"Candidate Approaches":     normalizeBulletList(input.CandidateApproaches),
		"Decision Snapshot":        input.DecisionSnapshot,
	}
	if strings.TrimSpace(input.Constraints) != "" {
		document.Body = setMarkdownSection(document.Body, "Constraints", normalizeBulletList(input.Constraints))
	}
	refinement := extractMarkdownSection(document.Body, "Refinement")
	for _, heading := range []string{"Problem", "User / Value", "Appetite", "Remaining Open Questions", "Candidate Approaches", "Decision Snapshot"} {
		value := strings.TrimSpace(values[heading])
		if value == "" {
			value = extractMarkdownSection(refinement, heading)
		}
		refinement = setMarkdownSubsection(refinement, heading, value)
	}
	body := setMarkdownSection(document.Body, "Refinement", refinement)
	return brainstormPreview(document, body), nil
}

// RefineBrainstorm applies a confirmed, authorized, audited structured refinement.
func (s *Service) RefineBrainstorm(ctx context.Context, input BrainstormRefinementInput, authorizer Authorizer, events EventSink) (BrainstormMutationResult, error) {
	preview, err := s.PreviewBrainstormRefinement(ctx, input)
	if err != nil || preview.Action == MutationUnchanged {
		return preview, err
	}
	return s.applyBrainstormBody(ctx, input.ID, preview.Document.Body, input.Confirmed, authorizer, events)
}

// PreviewBrainstormChallenge returns the merged structured challenge without writing.
func (s *Service) PreviewBrainstormChallenge(ctx context.Context, input BrainstormChallengeInput) (BrainstormMutationResult, error) {
	document, err := s.brainstormForMutation(ctx, input.ID)
	if err != nil {
		return BrainstormMutationResult{}, err
	}
	values := map[string]string{
		"Rabbit Holes":           normalizeBulletList(input.RabbitHoles),
		"No-Gos":                 normalizeBulletList(input.NoGos),
		"Assumptions":            normalizeBulletList(input.Assumptions),
		"Likely Overengineering": input.LikelyOverengineering,
		"Simpler Alternative":    input.SimplerAlternative,
	}
	challenge := extractMarkdownSection(document.Body, "Challenge")
	for _, heading := range []string{"Rabbit Holes", "No-Gos", "Assumptions", "Likely Overengineering", "Simpler Alternative"} {
		value := strings.TrimSpace(values[heading])
		if value == "" {
			value = extractMarkdownSection(challenge, heading)
		}
		challenge = setMarkdownSubsection(challenge, heading, value)
	}
	body := setMarkdownSection(document.Body, "Challenge", challenge)
	return brainstormPreview(document, body), nil
}

// ChallengeBrainstorm applies a confirmed, authorized, audited challenge pass.
func (s *Service) ChallengeBrainstorm(ctx context.Context, input BrainstormChallengeInput, authorizer Authorizer, events EventSink) (BrainstormMutationResult, error) {
	preview, err := s.PreviewBrainstormChallenge(ctx, input)
	if err != nil || preview.Action == MutationUnchanged {
		return preview, err
	}
	return s.applyBrainstormBody(ctx, input.ID, preview.Document.Body, input.Confirmed, authorizer, events)
}

func (s *Service) brainstormForMutation(ctx context.Context, id planning.ArtifactID) (BrainstormDocument, error) {
	if err := s.requireWritable(ctx); err != nil {
		return BrainstormDocument{}, err
	}
	if err := id.Validate(); err != nil {
		return BrainstormDocument{}, err
	}
	return s.repository.GetBrainstorm(ctx, id)
}

func (s *Service) applyBrainstormBody(ctx context.Context, id planning.ArtifactID, body string, confirmed bool, authorizer Authorizer, events EventSink) (BrainstormMutationResult, error) {
	if !confirmed {
		return BrainstormMutationResult{}, ErrConfirmationRequired
	}
	if authorizer == nil {
		return BrainstormMutationResult{}, fmt.Errorf("planning mutation requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionBrainstorm); err != nil {
		return BrainstormMutationResult{}, err
	}
	if events == nil {
		return BrainstormMutationResult{}, ErrEventSinkRequired
	}
	now := s.now().UTC()
	document, action, err := s.repository.ReplaceBrainstorm(ctx, id, body, now)
	if err != nil {
		return BrainstormMutationResult{}, err
	}
	result := BrainstormMutationResult{Action: action, Document: document}
	if action == MutationUnchanged {
		return result, nil
	}
	event := Event{Name: EventBrainstormUpdated, ModuleID: s.moduleID, Artifact: planning.ArtifactRef{Kind: planning.ArtifactBrainstorm, ID: id}, Outcome: action, OccurredAt: now}
	if err := events.Publish(ctx, event); err != nil {
		return BrainstormMutationResult{}, fmt.Errorf("publish planning event: %w", err)
	}
	result.Event = &event
	return result, nil
}

func brainstormPreview(document BrainstormDocument, body string) BrainstormMutationResult {
	action := MutationUpdate
	if document.Body == body {
		action = MutationUnchanged
	}
	document.Body = body
	return BrainstormMutationResult{Action: action, Document: document}
}

func normalizeBulletList(value string) string {
	var items []string
	for _, line := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if line != "" {
			items = append(items, "- "+line)
		}
	}
	return strings.Join(items, "\n")
}

func appendUniqueSection(content, heading, value string) (string, bool) {
	current := extractMarkdownSection(content, heading)
	for _, block := range strings.Split(current, "\n") {
		if strings.TrimSpace(block) == strings.TrimSpace(value) {
			return content, false
		}
	}
	if strings.Contains(current, value) {
		return content, false
	}
	next := strings.TrimSpace(current)
	if next != "" {
		next += "\n"
	}
	next += value
	return setMarkdownSection(content, heading, next), true
}

func setMarkdownSection(content, heading, value string) string {
	return setMarkdownHeading(content, "## "+heading, value)
}

func setMarkdownSubsection(content, heading, value string) string {
	return setMarkdownHeading(content, "### "+heading, value)
}

func setMarkdownHeading(content, marker, value string) string {
	content = strings.TrimRight(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	lines := strings.Split(content, "\n")
	targetLevel := len(marker) - len(strings.TrimLeft(marker, "#"))
	start := -1
	end := len(lines)
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if start < 0 && strings.EqualFold(trimmed, marker) {
			start = index
			continue
		}
		if start >= 0 && strings.HasPrefix(trimmed, "#") {
			level := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
			if level <= targetLevel {
				end = index
				break
			}
		}
	}
	block := []string{marker, ""}
	if strings.TrimSpace(value) != "" {
		block = append(block, strings.TrimSpace(value))
	}
	if start < 0 {
		return content + "\n\n" + strings.Join(block, "\n") + "\n"
	}
	updated := append([]string(nil), lines[:start]...)
	updated = append(updated, block...)
	if end < len(lines) {
		updated = append(updated, "")
		updated = append(updated, lines[end:]...)
	}
	return strings.TrimRight(strings.Join(updated, "\n"), "\n") + "\n"
}
