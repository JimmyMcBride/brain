package planning

import (
	"reflect"
	"testing"
)

func TestArtifactIDValidation(t *testing.T) {
	t.Parallel()

	for _, id := range []ArtifactID{"planning", "planning-domain", "phase-2"} {
		if err := id.Validate(); err != nil {
			t.Fatalf("expected %q to be valid: %v", id, err)
		}
	}
	for _, id := range []ArtifactID{"", " Planning", "planning ", "Planning", "planning/domain", "planning_domain", "-planning"} {
		if err := id.Validate(); ErrorCodeOf(err) != ErrInvalidIdentifier {
			t.Errorf("expected %q to fail with %s, got %v", id, ErrInvalidIdentifier, err)
		}
	}
}

func TestOwnershipValidationExcludesLinear(t *testing.T) {
	t.Parallel()

	for _, mode := range []OwnershipMode{OwnershipLocal, OwnershipGitHub, OwnershipHybrid} {
		if err := mode.Validate(); err != nil {
			t.Fatalf("expected %q to be valid: %v", mode, err)
		}
	}
	for _, mode := range []OwnershipMode{"", "linear", "cloud", "github-hybrid"} {
		if err := mode.Validate(); ErrorCodeOf(err) != ErrInvalidOwnership {
			t.Errorf("expected %q to fail with %s, got %v", mode, ErrInvalidOwnership, err)
		}
	}
}

func TestArtifactValidation(t *testing.T) {
	t.Parallel()

	specRef := ArtifactRef{Kind: ArtifactSpec, ID: "domain-extraction"}
	cases := []struct {
		name     string
		findings []Finding
	}{
		{
			name: "brainstorm",
			findings: ValidateBrainstorm(Brainstorm{
				ID: "domain-brainstorm", Title: "Domain Brainstorm",
			}),
		},
		{
			name: "spec",
			findings: ValidateSpec(Spec{
				ID: "domain-extraction", Title: "Domain Extraction", Status: SpecDraft,
				Approval: Approval{State: ApprovalPending}, Verification: []string{"go test ./..."},
			}),
		},
		{
			name: "initiative",
			findings: ValidateInitiative(Initiative{
				ID: "planning", Title: "Planning", Specs: []ArtifactRef{specRef},
			}),
		},
		{
			name: "roadmap",
			findings: ValidateRoadmap(Roadmap{
				ID: "planning-roadmap", Title: "Planning Roadmap",
				Entries: []RoadmapEntry{{Ref: specRef}},
			}),
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if len(tc.findings) != 0 {
				t.Fatalf("expected valid artifact, got %#v", tc.findings)
			}
		})
	}
}

func TestArtifactValidationRejectsReferenceViolations(t *testing.T) {
	t.Parallel()

	spec := Spec{
		ID:           "domain-extraction",
		Title:        "Domain Extraction",
		Status:       SpecDraft,
		Approval:     Approval{State: ApprovalPending},
		Dependencies: []ArtifactID{"domain-extraction", "Bad ID", "dependency", "dependency"},
	}
	if findings := ValidateSpec(spec); len(findings) < 4 {
		t.Fatalf("expected multiple spec findings, got %#v", findings)
	}

	initiative := Initiative{
		ID: "planning", Title: "Planning",
		Specs: []ArtifactRef{
			{Kind: ArtifactBrainstorm, ID: "brainstorm"},
			{Kind: ArtifactSpec, ID: "domain"},
			{Kind: ArtifactSpec, ID: "domain"},
		},
	}
	if findings := ValidateInitiative(initiative); len(findings) != 2 {
		t.Fatalf("expected kind and duplicate findings, got %#v", findings)
	}

	roadmap := Roadmap{
		ID: "roadmap", Title: "Roadmap",
		Entries: []RoadmapEntry{{Ref: ArtifactRef{Kind: ArtifactBrainstorm, ID: "brainstorm"}}},
	}
	if findings := ValidateRoadmap(roadmap); len(findings) != 1 {
		t.Fatalf("expected invalid roadmap reference, got %#v", findings)
	}
}

func TestApproveAndReopenSpec(t *testing.T) {
	t.Parallel()

	original := Spec{
		ID:           "domain-extraction",
		Title:        "Domain Extraction",
		Status:       SpecDraft,
		Approval:     Approval{State: ApprovalPending},
		Dependencies: []ArtifactID{"module-framework"},
		Verification: []string{"go test ./..."},
	}
	approved, err := ApproveSpec(original, Approval{State: ApprovalApproved, Reason: "reviewed"})
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != SpecApproved || approved.Approval.State != ApprovalApproved {
		t.Fatalf("unexpected approved spec: %#v", approved)
	}
	if original.Status != SpecDraft || original.Approval.State != ApprovalPending {
		t.Fatalf("original was mutated: %#v", original)
	}

	approved.Dependencies[0] = "changed"
	if reflect.DeepEqual(original.Dependencies, approved.Dependencies) {
		t.Fatal("approved spec shares dependency storage with original")
	}

	reopened, err := ReopenSpec(approved)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Status != SpecDraft || reopened.Approval.State != ApprovalPending {
		t.Fatalf("unexpected reopened spec: %#v", reopened)
	}
}

func TestApproveSpecRejectsInvalidTransitionsAndContracts(t *testing.T) {
	t.Parallel()

	valid := Spec{
		ID: "domain", Title: "Domain", Status: SpecApproved,
		Approval: Approval{State: ApprovalApproved}, Verification: []string{"go test"},
	}
	if _, err := ApproveSpec(valid, Approval{State: ApprovalApproved}); ErrorCodeOf(err) != ErrInvalidTransition {
		t.Fatalf("expected invalid transition, got %v", err)
	}

	draft := Spec{ID: "domain", Title: "Domain", Status: SpecDraft, Approval: Approval{State: ApprovalPending}}
	if _, err := ApproveSpec(draft, Approval{State: ApprovalPending}); ErrorCodeOf(err) != ErrApprovalRequired {
		t.Fatalf("expected approval required, got %v", err)
	}
	if _, err := ApproveSpec(draft, Approval{State: ApprovalApproved}); ErrorCodeOf(err) != ErrInvalidArtifact {
		t.Fatalf("expected invalid artifact for missing verification, got %v", err)
	}
}
