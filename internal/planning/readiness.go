package planning

import (
	"fmt"
	"slices"
	"strings"
)

type ReadinessState string

const (
	ReadinessClarifying      ReadinessState = "clarifying"
	ReadinessReady           ReadinessState = "ready"
	ReadinessBlocked         ReadinessState = "blocked"
	ReadinessNeedsRefinement ReadinessState = "needs_refinement"
	ReadinessDone            ReadinessState = "done"
)

type Readiness struct {
	State   ReadinessState
	Reasons []string
}

// EvaluateReadiness applies the documented precedence to structured inputs.
func EvaluateReadiness(spec Spec, findings []Finding, blockers []ArtifactID) Readiness {
	if spec.Status == SpecDone {
		return Readiness{State: ReadinessDone}
	}

	blockers = uniqueSortedIDs(blockers)
	var blockedReasons []string
	for _, blocker := range blockers {
		blockedReasons = append(blockedReasons, fmt.Sprintf("blocked by %s", blocker))
	}
	for _, finding := range findings {
		if finding.Code == ErrDependencyMissing || finding.Code == ErrDependencyCycle {
			blockedReasons = append(blockedReasons, finding.Message)
		}
	}
	if len(blockedReasons) > 0 {
		slices.Sort(blockedReasons)
		return Readiness{State: ReadinessBlocked, Reasons: uniqueStrings(blockedReasons)}
	}

	var refinementReasons []string
	for _, finding := range findings {
		if finding.Severity == SeverityError {
			refinementReasons = append(refinementReasons, finding.Message)
		}
	}
	if len(refinementReasons) > 0 {
		slices.Sort(refinementReasons)
		return Readiness{State: ReadinessNeedsRefinement, Reasons: uniqueStrings(refinementReasons)}
	}

	questions := trimmedStrings(spec.UnresolvedQuestions)
	if len(questions) > 0 {
		slices.Sort(questions)
		return Readiness{State: ReadinessClarifying, Reasons: questions}
	}
	if spec.Approval.State != ApprovalApproved || (spec.Status != SpecApproved && spec.Status != SpecImplementing) {
		return Readiness{State: ReadinessNeedsRefinement, Reasons: []string{"spec is not approved for execution"}}
	}
	return Readiness{State: ReadinessReady}
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
