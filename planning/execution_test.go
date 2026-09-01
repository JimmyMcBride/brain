package planning

import (
	"reflect"
	"testing"
)

func TestBeginExecutionRequiresApprovalAndReadiness(t *testing.T) {
	t.Parallel()

	input := ExecutionInput{ID: "run-domain"}
	if _, _, err := BeginExecution(testSpec("domain", SpecDraft), input); ErrorCodeOf(err) != ErrApprovalRequired {
		t.Fatalf("expected approval required, got %v", err)
	}
	if _, _, err := BeginExecution(testSpec("domain", SpecApproved), ExecutionInput{
		ID: "run-domain", Blockers: []ArtifactID{"framework"},
	}); ErrorCodeOf(err) != ErrNotReady {
		t.Fatalf("expected not ready, got %v", err)
	}
}

func TestBeginExecutionPrefersExplicitCandidates(t *testing.T) {
	t.Parallel()

	spec := testSpec("domain", SpecApproved)
	implementing, plan, err := BeginExecution(spec, ExecutionInput{
		ID: "run-domain",
		ExplicitCandidates: []SliceCandidate{
			{Title: executionPlaceholder},
			{Title: "Domain Values", Goal: "Add domain values."},
			{Title: "Readiness", Goal: "Add readiness.", Verification: []string{"go test ./internal/planning"}},
		},
		FlowCandidates: []SliceCandidate{{Title: "Ignored Flow"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if implementing.Status != SpecImplementing || implementing.ExecutionID == nil || *implementing.ExecutionID != "run-domain" {
		t.Fatalf("unexpected implementing spec: %#v", implementing)
	}
	if spec.Status != SpecApproved || spec.ExecutionID != nil {
		t.Fatalf("original spec was mutated: %#v", spec)
	}
	wantIDs := []ArtifactID{"domain-values", "readiness"}
	if got := sliceIDs(plan.Slices); !reflect.DeepEqual(got, wantIDs) {
		t.Fatalf("slice IDs = %v, want %v", got, wantIDs)
	}
	if plan.Slices[0].State != SliceActive || plan.Slices[1].State != SlicePending {
		t.Fatalf("unexpected initial states: %#v", plan.Slices)
	}
	if !reflect.DeepEqual(plan.Slices[0].Verification, spec.Verification) {
		t.Fatalf("default verification = %v, want %v", plan.Slices[0].Verification, spec.Verification)
	}
}

func TestBeginExecutionUsesAtMostThreeFlowCandidates(t *testing.T) {
	t.Parallel()

	_, plan, err := BeginExecution(testSpec("domain", SpecApproved), ExecutionInput{
		ID: "run-domain",
		FlowCandidates: []SliceCandidate{
			{Title: "Validate"},
			{Title: "Order"},
			{Title: "Execute"},
			{Title: "Validate"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []ArtifactID{"validate", "order", "execute"}
	if got := sliceIDs(plan.Slices); !reflect.DeepEqual(got, want) {
		t.Fatalf("flow slices = %v, want %v", got, want)
	}
	for _, slice := range plan.Slices {
		if slice.Goal == "" {
			t.Fatalf("flow slice has no goal: %#v", slice)
		}
	}
}

func TestBeginExecutionFallsBackDeterministically(t *testing.T) {
	t.Parallel()

	spec := testSpec("domain", SpecApproved)
	spec.Title = "Planning Domain Spec"
	_, plan, err := BeginExecution(spec, ExecutionInput{
		ID:                 "run-domain",
		ExplicitCandidates: []SliceCandidate{{Title: executionPlaceholder}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []ArtifactID{"prepare-planning-domain", "implement-planning-domain", "verify-planning-domain"}
	if got := sliceIDs(plan.Slices); !reflect.DeepEqual(got, want) {
		t.Fatalf("fallback slices = %v, want %v", got, want)
	}
}

func TestBeginExecutionRejectsInvalidSlices(t *testing.T) {
	t.Parallel()

	cases := []ExecutionInput{
		{ID: "run", ExplicitCandidates: []SliceCandidate{{Title: "Missing Goal"}}},
		{ID: "run", ExplicitCandidates: []SliceCandidate{{Title: "Same", Goal: "one"}, {Title: "Same!", Goal: "two"}}},
		{ID: "run", ExplicitCandidates: []SliceCandidate{{Title: "", Goal: "goal"}}},
	}
	for _, input := range cases {
		if _, _, err := BeginExecution(testSpec("domain", SpecApproved), input); ErrorCodeOf(err) != ErrInvalidSlice {
			t.Errorf("expected invalid slice for %#v, got %v", input, err)
		}
	}
}

func TestBeginExecutionIsIdempotentForSameIdentity(t *testing.T) {
	t.Parallel()

	spec := testSpec("domain", SpecImplementing)
	_, first, err := BeginExecution(spec, ExecutionInput{ID: *spec.ExecutionID})
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := BeginExecution(spec, ExecutionInput{ID: *spec.ExecutionID})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same execution input produced different plans:\n%#v\n%#v", first, second)
	}
	if _, _, err := BeginExecution(spec, ExecutionInput{ID: "different"}); ErrorCodeOf(err) != ErrInvalidExecution {
		t.Fatalf("expected invalid execution, got %v", err)
	}
}

func TestCompleteSliceAdvancesAndRetainsEvidence(t *testing.T) {
	t.Parallel()

	spec, plan, err := BeginExecution(testSpec("domain", SpecApproved), ExecutionInput{
		ID: "run-domain",
		ExplicitCandidates: []SliceCandidate{
			{Title: "Values", Goal: "Add values."},
			{Title: "Execution", Goal: "Add execution."},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteSlice(plan, "execution", []Evidence{{Summary: "too early"}}); ErrorCodeOf(err) != ErrInvalidSlice {
		t.Fatalf("expected active-slice error, got %v", err)
	}
	if _, err := CompleteSlice(plan, "values", nil); ErrorCodeOf(err) != ErrInvalidSlice {
		t.Fatalf("expected evidence error, got %v", err)
	}

	evidence := []Evidence{{Kind: "test", Summary: "focused tests pass", Reference: "go test ./internal/planning"}}
	advanced, err := CompleteSlice(plan, "values", evidence)
	if err != nil {
		t.Fatal(err)
	}
	if advanced.Active != 1 || advanced.Slices[0].State != SliceDone || advanced.Slices[1].State != SliceActive {
		t.Fatalf("unexpected advanced plan: %#v", advanced)
	}
	if !reflect.DeepEqual(advanced.Slices[0].Evidence, evidence) {
		t.Fatalf("evidence = %#v, want %#v", advanced.Slices[0].Evidence, evidence)
	}
	if plan.Active != 0 || plan.Slices[0].State != SliceActive || len(plan.Slices[0].Evidence) != 0 {
		t.Fatalf("original plan was mutated: %#v", plan)
	}

	completedPlan, err := CompleteSlice(advanced, "execution", []Evidence{{Summary: "all tests pass"}})
	if err != nil {
		t.Fatal(err)
	}
	if completedPlan.State != ExecutionComplete || completedPlan.Active != len(completedPlan.Slices) {
		t.Fatalf("unexpected completed plan: %#v", completedPlan)
	}
	completedSpec, err := CompleteExecution(spec, completedPlan)
	if err != nil {
		t.Fatal(err)
	}
	if completedSpec.Status != SpecDone {
		t.Fatalf("completed spec status = %q", completedSpec.Status)
	}

	reopened, err := ReopenSpec(completedSpec)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.ExecutionID != nil {
		t.Fatalf("reopened spec retained execution ID: %#v", reopened)
	}
}

func sliceIDs(slices []RuntimeSlice) []ArtifactID {
	ids := make([]ArtifactID, len(slices))
	for i, slice := range slices {
		ids[i] = slice.ID
	}
	return ids
}
