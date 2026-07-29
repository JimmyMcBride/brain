package planning

import (
	"reflect"
	"testing"
)

func TestOrderSpecsIsDeterministic(t *testing.T) {
	t.Parallel()

	specs := []Spec{
		testSpec("deploy", SpecApproved, "build", "test"),
		testSpec("test", SpecDone, "build"),
		testSpec("build", SpecDone),
		testSpec("docs", SpecApproved),
	}
	want := []ArtifactID{"build", "docs", "test", "deploy"}
	for _, input := range [][]Spec{
		specs,
		{specs[3], specs[1], specs[0], specs[2]},
		{specs[0], specs[2], specs[3], specs[1]},
	} {
		got, err := OrderSpecs(input)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("OrderSpecs() = %v, want %v", got, want)
		}
	}
}

func TestOrderSpecsReportsMissingDependencies(t *testing.T) {
	t.Parallel()

	_, err := OrderSpecs([]Spec{
		testSpec("deploy", SpecApproved, "missing-b", "missing-a"),
		testSpec("build", SpecDone),
	})
	if ErrorCodeOf(err) != ErrDependencyMissing {
		t.Fatalf("expected dependency missing, got %v", err)
	}
	domain := err.(*DomainError)
	want := []ArtifactID{"missing-a", "missing-b"}
	if !reflect.DeepEqual(domain.Artifacts, want) {
		t.Fatalf("missing artifacts = %v, want %v", domain.Artifacts, want)
	}
}

func TestOrderSpecsIdentifiesCycleMembersOnly(t *testing.T) {
	t.Parallel()

	_, err := OrderSpecs([]Spec{
		testSpec("a", SpecApproved, "b"),
		testSpec("b", SpecApproved, "a"),
		testSpec("downstream", SpecApproved, "a"),
	})
	if ErrorCodeOf(err) != ErrDependencyCycle {
		t.Fatalf("expected dependency cycle, got %v", err)
	}
	domain := err.(*DomainError)
	want := []ArtifactID{"a", "b"}
	if !reflect.DeepEqual(domain.Artifacts, want) {
		t.Fatalf("cycle artifacts = %v, want %v", domain.Artifacts, want)
	}
}

func TestEvaluateReadinessPrecedence(t *testing.T) {
	t.Parallel()

	errorFinding := Finding{Code: ErrInvalidArtifact, Severity: SeverityError, Message: "invalid"}
	dependencyFinding := Finding{Code: ErrDependencyCycle, Severity: SeverityError, Message: "cycle"}
	cases := []struct {
		name     string
		spec     Spec
		findings []Finding
		blockers []ArtifactID
		want     ReadinessState
	}{
		{name: "done wins", spec: testSpec("spec", SpecDone), findings: []Finding{dependencyFinding}, blockers: []ArtifactID{"other"}, want: ReadinessDone},
		{name: "blocked before refinement", spec: testSpec("spec", SpecApproved), findings: []Finding{errorFinding}, blockers: []ArtifactID{"other"}, want: ReadinessBlocked},
		{name: "dependency finding blocks", spec: testSpec("spec", SpecApproved), findings: []Finding{dependencyFinding}, want: ReadinessBlocked},
		{name: "error needs refinement", spec: testSpec("spec", SpecApproved), findings: []Finding{errorFinding}, want: ReadinessNeedsRefinement},
		{name: "approval before question", spec: withQuestions(testSpec("spec", SpecDraft), "Which adapter?"), want: ReadinessNeedsRefinement},
		{name: "question clarifies", spec: withQuestions(testSpec("spec", SpecApproved), "Which adapter?"), want: ReadinessClarifying},
		{name: "approved is ready", spec: testSpec("spec", SpecApproved), want: ReadinessReady},
		{name: "draft needs refinement", spec: testSpec("spec", SpecDraft), want: ReadinessNeedsRefinement},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := EvaluateReadiness(tc.spec, tc.findings, tc.blockers).State; got != tc.want {
				t.Fatalf("EvaluateReadiness() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEvaluateReadinessDeduplicatesQuestions(t *testing.T) {
	t.Parallel()

	readiness := EvaluateReadiness(
		withQuestions(testSpec("spec", SpecApproved), "Beta?", "Alpha?", "Beta?", " "),
		nil,
		nil,
	)
	want := Readiness{State: ReadinessClarifying, Reasons: []string{"Alpha?", "Beta?"}}
	if !reflect.DeepEqual(readiness, want) {
		t.Fatalf("EvaluateReadiness() = %#v, want %#v", readiness, want)
	}
}

func TestBuildQueueIsStableAndInfersCurrent(t *testing.T) {
	t.Parallel()

	specs := []Spec{
		testSpec("done", SpecDone),
		testSpec("current", SpecImplementing, "done"),
		testSpec("next", SpecApproved, "current"),
		testSpec("independent", SpecApproved),
		testSpec("draft", SpecDraft),
	}
	want := Queue{Entries: []QueueEntry{
		{SpecID: "done", State: QueueDone},
		{SpecID: "current", State: QueueCurrent},
		{SpecID: "draft", State: QueueBlocked, Reasons: []string{"spec is not approved for execution"}},
		{SpecID: "independent", State: QueueReady},
		{SpecID: "next", State: QueueBlocked, Reasons: []string{"blocked by current"}},
	}}
	for _, input := range [][]Spec{
		specs,
		{specs[4], specs[2], specs[0], specs[3], specs[1]},
	} {
		got, err := BuildQueue(input, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("BuildQueue() = %#v, want %#v", got, want)
		}
	}
}

func TestBuildQueueRejectsMultipleCurrentSpecs(t *testing.T) {
	t.Parallel()

	_, err := BuildQueue([]Spec{
		testSpec("one", SpecImplementing),
		testSpec("two", SpecImplementing),
	}, nil)
	if ErrorCodeOf(err) != ErrInvalidExecution {
		t.Fatalf("expected invalid execution, got %v", err)
	}
}

func TestBuildQueueSortsCurrentBlockers(t *testing.T) {
	t.Parallel()

	build := func(dependencies []ArtifactID) error {
		current := testSpec("current", SpecImplementing, dependencies...)
		_, err := BuildQueue([]Spec{
			current,
			testSpec("alpha", SpecApproved),
			testSpec("beta", SpecApproved),
		}, nil)
		return err
	}

	first := build([]ArtifactID{"beta", "alpha", "beta"})
	second := build([]ArtifactID{"alpha", "beta"})
	if ErrorCodeOf(first) != ErrInvalidExecution || ErrorCodeOf(second) != ErrInvalidExecution {
		t.Fatalf("expected invalid execution errors, got %v and %v", first, second)
	}
	if first.Error() != second.Error() {
		t.Fatalf("blocker errors differ:\n%s\n%s", first, second)
	}
}

func testSpec(id ArtifactID, status SpecStatus, dependencies ...ArtifactID) Spec {
	approval := Approval{State: ApprovalApproved}
	var executionID *ArtifactID
	if status == SpecDraft {
		approval.State = ApprovalPending
	}
	if status == SpecImplementing {
		value := ArtifactID("execution-" + string(id))
		executionID = &value
	}
	return Spec{
		ID:           id,
		Title:        string(id),
		Status:       status,
		Approval:     approval,
		ExecutionID:  executionID,
		Dependencies: append([]ArtifactID(nil), dependencies...),
		Verification: []string{"go test ./..."},
	}
}

func withQuestions(spec Spec, questions ...string) Spec {
	spec.UnresolvedQuestions = append([]string(nil), questions...)
	return spec
}
