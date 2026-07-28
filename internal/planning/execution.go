package planning

import (
	"fmt"
	"regexp"
	"strings"
)

const executionPlaceholder = "define execution slices when implementation begins"

var nonSlugCharacters = regexp.MustCompile(`[^a-z0-9]+`)

type SliceCandidate struct {
	ID           ArtifactID
	Title        string
	Goal         string
	Verification []string
}

type ExecutionInput struct {
	ID                  ArtifactID
	ExplicitCandidates  []SliceCandidate
	FlowCandidates      []SliceCandidate
	DefaultVerification []string
	Findings            []Finding
	Blockers            []ArtifactID
}

type SliceState string

const (
	SlicePending SliceState = "pending"
	SliceActive  SliceState = "active"
	SliceDone    SliceState = "done"
)

type Evidence struct {
	Kind      string
	Summary   string
	Reference string
}

type RuntimeSlice struct {
	ID           ArtifactID
	Title        string
	Goal         string
	Verification []string
	Position     int
	State        SliceState
	Evidence     []Evidence
}

type ExecutionState string

const (
	ExecutionActive   ExecutionState = "active"
	ExecutionComplete ExecutionState = "complete"
)

type ExecutionPlan struct {
	ID     ArtifactID
	SpecID ArtifactID
	Slices []RuntimeSlice
	Active int
	State  ExecutionState
}

// BeginExecution returns an implementing spec and a deterministic ephemeral
// execution plan. It performs no persistence.
func BeginExecution(spec Spec, input ExecutionInput) (Spec, ExecutionPlan, error) {
	if err := input.ID.Validate(); err != nil {
		return Spec{}, ExecutionPlan{}, domainError(ErrInvalidExecution, []ArtifactID{input.ID}, "execution identifier is invalid")
	}
	switch spec.Status {
	case SpecDraft:
		return Spec{}, ExecutionPlan{}, domainError(ErrApprovalRequired, []ArtifactID{spec.ID}, "approve the spec before execution")
	case SpecApproved:
	case SpecImplementing:
		if spec.ExecutionID == nil || *spec.ExecutionID != input.ID {
			return Spec{}, ExecutionPlan{}, domainError(ErrInvalidExecution, []ArtifactID{spec.ID, input.ID}, "spec is already running under a different execution")
		}
	case SpecDone:
		return Spec{}, ExecutionPlan{}, domainError(ErrInvalidTransition, []ArtifactID{spec.ID}, "completed spec must be reopened before execution")
	default:
		return Spec{}, ExecutionPlan{}, domainError(ErrInvalidTransition, []ArtifactID{spec.ID}, "spec status is invalid")
	}

	findings := append([]Finding(nil), validateSpecForExecution(spec)...)
	findings = append(findings, input.Findings...)
	readiness := EvaluateReadiness(specForReadiness(spec), findings, input.Blockers)
	if readiness.State != ReadinessReady {
		return Spec{}, ExecutionPlan{}, domainError(ErrNotReady, []ArtifactID{spec.ID}, readiness.Reasons...)
	}

	slices, err := deriveRuntimeSlices(spec, input)
	if err != nil {
		return Spec{}, ExecutionPlan{}, err
	}
	implementing := cloneSpec(spec)
	implementing.Status = SpecImplementing
	executionID := input.ID
	implementing.ExecutionID = &executionID

	plan := ExecutionPlan{
		ID:     input.ID,
		SpecID: spec.ID,
		Slices: slices,
		Active: 0,
		State:  ExecutionActive,
	}
	plan.Slices[0].State = SliceActive
	return implementing, plan, nil
}

// CompleteSlice advances exactly one active slice and retains supplied
// verification evidence as domain values.
func CompleteSlice(plan ExecutionPlan, sliceID ArtifactID, evidence []Evidence) (ExecutionPlan, error) {
	if plan.State != ExecutionActive || plan.Active < 0 || plan.Active >= len(plan.Slices) {
		return ExecutionPlan{}, domainError(ErrInvalidExecution, []ArtifactID{plan.ID}, "execution has no active slice")
	}
	active := plan.Slices[plan.Active]
	if active.ID != sliceID {
		return ExecutionPlan{}, domainError(ErrInvalidSlice, []ArtifactID{sliceID}, fmt.Sprintf("active slice is %s", active.ID))
	}
	if err := validateEvidence(evidence); err != nil {
		return ExecutionPlan{}, err
	}

	updated := cloneExecutionPlan(plan)
	updated.Slices[updated.Active].State = SliceDone
	updated.Slices[updated.Active].Evidence = append([]Evidence(nil), evidence...)
	updated.Active++
	if updated.Active == len(updated.Slices) {
		updated.State = ExecutionComplete
		return updated, nil
	}
	updated.Slices[updated.Active].State = SliceActive
	return updated, nil
}

// CompleteExecution returns a completed spec after its matching plan finishes.
func CompleteExecution(spec Spec, plan ExecutionPlan) (Spec, error) {
	if spec.Status != SpecImplementing || spec.ExecutionID == nil {
		return Spec{}, domainError(ErrInvalidTransition, []ArtifactID{spec.ID}, "spec is not implementing")
	}
	if *spec.ExecutionID != plan.ID || spec.ID != plan.SpecID {
		return Spec{}, domainError(ErrInvalidExecution, []ArtifactID{spec.ID, plan.ID}, "execution does not belong to spec")
	}
	if plan.State != ExecutionComplete || plan.Active != len(plan.Slices) {
		return Spec{}, domainError(ErrInvalidExecution, []ArtifactID{plan.ID}, "execution is not complete")
	}
	completed := cloneSpec(spec)
	completed.Status = SpecDone
	return completed, nil
}

func validateSpecForExecution(spec Spec) []Finding {
	candidate := cloneSpec(spec)
	if candidate.Status == SpecImplementing {
		candidate.Status = SpecApproved
		candidate.ExecutionID = nil
	}
	return ValidateSpec(candidate)
}

func specForReadiness(spec Spec) Spec {
	candidate := cloneSpec(spec)
	if candidate.Status == SpecImplementing {
		candidate.Status = SpecApproved
		candidate.ExecutionID = nil
	}
	return candidate
}

func deriveRuntimeSlices(spec Spec, input ExecutionInput) ([]RuntimeSlice, error) {
	defaultVerification := trimmedStrings(input.DefaultVerification)
	if len(defaultVerification) == 0 {
		defaultVerification = trimmedStrings(spec.Verification)
	}

	explicit, err := candidatesToSlices(input.ExplicitCandidates, defaultVerification, false, spec.Title, 0)
	if err != nil {
		return nil, err
	}
	if len(explicit) > 0 {
		return explicit, nil
	}

	flows, err := candidatesToSlices(input.FlowCandidates, defaultVerification, true, spec.Title, 3)
	if err != nil {
		return nil, err
	}
	if len(flows) > 0 {
		return flows, nil
	}
	return fallbackSlices(spec, defaultVerification)
}

func candidatesToSlices(candidates []SliceCandidate, defaults []string, flow bool, specTitle string, limit int) ([]RuntimeSlice, error) {
	var slices []RuntimeSlice
	seen := map[ArtifactID]struct{}{}
	for _, candidate := range candidates {
		title := strings.TrimSpace(candidate.Title)
		goal := strings.TrimSpace(candidate.Goal)
		if title == "" && goal == "" {
			continue
		}
		if strings.EqualFold(title, executionPlaceholder) {
			continue
		}
		if title == "" {
			return nil, domainError(ErrInvalidSlice, nil, "slice title is required")
		}
		if flow && goal == "" {
			goal = fmt.Sprintf("Implement and verify the %q flow for %s.", title, specTitle)
		}
		if goal == "" {
			return nil, domainError(ErrInvalidSlice, nil, fmt.Sprintf("slice %q requires a goal", title))
		}

		id := candidate.ID
		if id == "" {
			id = slugify(title)
		}
		if err := id.Validate(); err != nil {
			return nil, domainError(ErrInvalidSlice, []ArtifactID{id}, "slice identifier is invalid")
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, domainError(ErrInvalidSlice, []ArtifactID{id}, "slice identifier is duplicated")
		}
		seen[id] = struct{}{}

		verification := trimmedStrings(candidate.Verification)
		if len(verification) == 0 {
			verification = append([]string(nil), defaults...)
		}
		slices = append(slices, RuntimeSlice{
			ID:           id,
			Title:        title,
			Goal:         goal,
			Verification: verification,
			Position:     len(slices) + 1,
			State:        SlicePending,
		})
		if limit > 0 && len(slices) == limit {
			break
		}
	}
	return slices, nil
}

func fallbackSlices(spec Spec, verification []string) ([]RuntimeSlice, error) {
	baseTitle := strings.TrimSpace(strings.TrimSuffix(spec.Title, " Spec"))
	if baseTitle == "" {
		baseTitle = strings.TrimSpace(spec.Title)
	}
	candidates := []SliceCandidate{
		{Title: "Prepare " + baseTitle, Goal: fmt.Sprintf("Set up the interfaces, constraints, and prerequisites required for %s.", baseTitle)},
		{Title: "Implement " + baseTitle, Goal: fmt.Sprintf("Build the core behavior described by %s.", spec.Title)},
		{Title: "Verify " + baseTitle, Goal: fmt.Sprintf("Run the end-to-end checks for %s and clean up rollout edges.", spec.Title)},
	}
	return candidatesToSlices(candidates, verification, false, spec.Title, 0)
}

func slugify(value string) ArtifactID {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Trim(nonSlugCharacters.ReplaceAllString(value, "-"), "-")
	return ArtifactID(value)
}

func validateEvidence(evidence []Evidence) error {
	if len(evidence) == 0 {
		return domainError(ErrInvalidSlice, nil, "verification evidence is required")
	}
	for _, item := range evidence {
		if strings.TrimSpace(item.Summary) == "" && strings.TrimSpace(item.Reference) == "" {
			return domainError(ErrInvalidSlice, nil, "evidence requires a summary or reference")
		}
	}
	return nil
}

func cloneExecutionPlan(plan ExecutionPlan) ExecutionPlan {
	copy := plan
	copy.Slices = make([]RuntimeSlice, len(plan.Slices))
	for i, slice := range plan.Slices {
		copy.Slices[i] = slice
		copy.Slices[i].Verification = append([]string(nil), slice.Verification...)
		copy.Slices[i].Evidence = append([]Evidence(nil), slice.Evidence...)
	}
	return copy
}
