package promotion

import "testing"

func TestCategoriesAndNonPromotableDefaults(t *testing.T) {
	if len(Categories()) != 6 {
		t.Fatalf("expected six first-wave categories, got %d", len(Categories()))
	}
	if len(NonPromotableDefaults()) != 3 {
		t.Fatalf("expected three non-promotable defaults, got %d", len(NonPromotableDefaults()))
	}
}

func TestAssessSessionPromotableAndRejectedCandidates(t *testing.T) {
	assessments := AssessSession(SessionSignals{
		Task:                   "replace context loading flow",
		RepoChanged:            true,
		ChangedFiles:           []string{"internal/session/manager.go", "skills/brain/SKILL.md"},
		ChangedBoundaries:      []string{"internal/session/", "skills/"},
		PacketHashes:           []string{"packet-1"},
		SuccessfulCommands:     []string{"go test ./...", "go build ./..."},
		MissingVerification:    []string{"integration-tests"},
		WorkflowSurfaceChanged: true,
		DecisionLikeTask:       true,
	})

	got := promotableCategories(assessments)
	if len(got) < 4 {
		t.Fatalf("expected multiple promotable categories, got %#v", got)
	}
	if !containsCategory(got, CategoryBoundaryFact) {
		t.Fatalf("expected boundary fact to be promotable, got %#v", got)
	}
	if !containsCategory(got, CategoryVerificationRecipe) {
		t.Fatalf("expected verification recipe to be promotable, got %#v", got)
	}
	if !containsCategory(got, CategoryDecision) {
		t.Fatalf("expected decision to be promotable, got %#v", got)
	}
	if !containsCategory(got, CategoryInvariant) {
		t.Fatalf("expected invariant to be promotable, got %#v", got)
	}
}

func TestAssessSessionRejectsWhenEvidenceIsMissing(t *testing.T) {
	assessments := AssessSession(SessionSignals{
		Task:         "touch a file",
		RepoChanged:  true,
		ChangedFiles: []string{"main.go"},
	})

	if containsPromotable(assessments, CategoryVerificationRecipe) {
		t.Fatalf("did not expect verification recipe to be promotable: %#v", assessments)
	}
	if containsPromotable(assessments, CategoryDecision) {
		t.Fatalf("did not expect decision to be promotable: %#v", assessments)
	}
	if containsPromotable(assessments, CategoryInvariant) {
		t.Fatalf("did not expect invariant to be promotable: %#v", assessments)
	}
	if !containsDecision(assessments, CategoryBoundaryFact, DecisionPromotable) {
		t.Fatalf("expected boundary fact to remain reviewably promotable from repo changes alone: %#v", assessments)
	}
}

func TestAssessSessionRejectsVerificationRecipeWithoutMeaningfulRepoChange(t *testing.T) {
	assessments := AssessSession(SessionSignals{
		Task:               "merge pr and sync develop",
		SuccessfulCommands: []string{"go test ./...", "go build ./..."},
	})

	if containsPromotable(assessments, CategoryVerificationRecipe) {
		t.Fatalf("did not expect verification recipe to be promotable without meaningful repo change: %#v", assessments)
	}
}

func TestAssessSessionRejectsAlreadyCapturedDurableUpdates(t *testing.T) {
	assessments := AssessSession(SessionSignals{
		Task:               "tighten compiler docs",
		RepoChanged:        true,
		ChangedFiles:       []string{"docs/usage.md"},
		ChangedBoundaries:  []string{"docs/"},
		PacketHashes:       []string{"packet-1"},
		SuccessfulCommands: []string{"go test ./..."},
		DurableUpdates:     []string{"docs/usage.md"},
	})

	if !containsPromotable(assessments, CategoryBoundaryFact) {
		t.Fatalf("expected boundary fact to remain promotable when a different durable note changed: %#v", assessments)
	}
	if !containsPromotable(assessments, CategoryVerificationRecipe) {
		t.Fatalf("expected verification recipe to remain promotable when its own target is still untouched: %#v", assessments)
	}
}

func promotableCategories(assessments []Assessment) []Category {
	out := []Category{}
	for _, assessment := range assessments {
		if assessment.Decision == DecisionPromotable {
			out = append(out, assessment.Candidate.Category)
		}
	}
	return out
}

func containsCategory(items []Category, target Category) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func containsPromotable(assessments []Assessment, category Category) bool {
	return containsDecision(assessments, category, DecisionPromotable)
}

func containsDecision(assessments []Assessment, category Category, decision Decision) bool {
	for _, assessment := range assessments {
		if assessment.Candidate.Category == category && assessment.Decision == decision {
			return true
		}
	}
	return false
}
