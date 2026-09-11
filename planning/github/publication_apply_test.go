package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"
)

type publicationWriteRunner struct {
	t                           *testing.T
	issues                      map[int]publicationIssue
	labels                      map[string]bool
	links                       map[string][]int
	calls, writes, next, failAt int
	lostResponse                bool
	payloads                    []map[string]any
	beforeRead                  func(string)
	beforeWrite                 func()
	failStderr                  string
}

func newPublicationWriteRunner(t *testing.T) *publicationWriteRunner {
	return &publicationWriteRunner{t: t, issues: map[int]publicationIssue{}, labels: map[string]bool{}, links: map[string][]int{}, next: 101}
}

func publicationREST(issue publicationIssue) map[string]any {
	raw, _ := json.Marshal(issue)
	var result map[string]any
	_ = json.Unmarshal(raw, &result)
	result["id"], result["node_id"], result["html_url"] = issue.Number+1000, issue.ID, issue.URL
	result["url"] = fmt.Sprintf("https://api.github.com/repos/owner/repo/issues/%d", issue.Number)
	return result
}

func (r *publicationWriteRunner) Run(_ context.Context, _ string, args ...string) (RunResult, error) {
	r.calls++
	var value any
	if len(args) > 1 && args[0] == "issue" && args[1] == "list" {
		listed := []publicationIssue{}
		for _, issue := range r.issues {
			if publicationHasLabel(issue, args[len(args)-1]) {
				listed = append(listed, issue)
			}
		}
		slices.SortFunc(listed, func(a, b publicationIssue) int { return a.Number - b.Number })
		value = listed
	} else if len(args) == 4 && args[0] == "api" && args[2] == "GET" {
		endpoint := args[3]
		if r.beforeRead != nil {
			r.beforeRead(endpoint)
		}
		switch {
		case strings.Contains(endpoint, "/labels?"):
			labels := []map[string]string{}
			for name := range r.labels {
				labels = append(labels, map[string]string{"name": name})
			}
			value = labels
		case strings.HasSuffix(endpoint, "?per_page=100"):
			linked := []map[string]any{}
			for _, number := range r.links[endpoint] {
				linked = append(linked, publicationREST(r.issues[number]))
			}
			value = linked
		default:
			parts := strings.Split(endpoint, "/")
			number, _ := strconv.Atoi(parts[len(parts)-1])
			issue, exists := r.issues[number]
			if !exists {
				return RunResult{}, errors.New("missing issue")
			}
			value = publicationREST(issue)
		}
	} else {
		r.t.Fatalf("unexpected argument-based mutation: %q", args)
	}
	raw, err := json.Marshal(value)
	return RunResult{Stdout: raw}, err
}

func (r *publicationWriteRunner) RunInput(_ context.Context, _ string, input []byte, args ...string) (RunResult, error) {
	if r.beforeWrite != nil {
		r.beforeWrite()
	}
	r.writes++
	if len(args) != 6 || args[0] != "api" || args[1] != "--method" || args[4] != "--input" || args[5] != "-" {
		r.t.Fatalf("unsafe mutation arguments: %q", args)
	}
	var payload map[string]any
	if err := json.Unmarshal(input, &payload); err != nil {
		r.t.Fatal(err)
	}
	r.payloads = append(r.payloads, payload)
	if r.writes == r.failAt && !r.lostResponse {
		stderr := r.failStderr
		if stderr == "" {
			stderr = "secret token and private content"
		}
		return RunResult{Stderr: []byte(stderr)}, errors.New("secret")
	}
	var response any
	endpoint := args[3]
	switch {
	case strings.HasSuffix(endpoint, "/labels"):
		name := payload["name"].(string)
		if r.labels[name] {
			r.t.Fatal("duplicate label create", name)
		}
		r.labels[name] = true
		response = payload
	case endpoint == "graphql":
		variables := payload["variables"].(map[string]any)
		var source, target publicationIssue
		for _, issue := range r.issues {
			if issue.ID == variables["source"] {
				source = issue
			}
			if issue.ID == variables["target"] {
				target = issue
			}
		}
		if source.ID == "" || target.ID == "" {
			r.t.Fatal("unresolved relationship")
		}
		mutation, field, suffix := "addSubIssue", "subIssue", "sub_issues"
		if strings.Contains(payload["query"].(string), "addBlockedBy") {
			mutation, field, suffix = "addBlockedBy", "blockingIssue", "dependencies/blocked_by"
		}
		key := fmt.Sprintf("repos/owner/repo/issues/%d/%s?per_page=100", source.Number, suffix)
		if slices.Contains(r.links[key], target.Number) {
			r.t.Fatal("duplicate relationship write")
		}
		r.links[key] = append(r.links[key], target.Number)
		response = map[string]any{"data": map[string]any{mutation: map[string]any{"issue": map[string]any{"id": source.ID, "number": source.Number}, field: map[string]any{"id": target.ID, "number": target.Number}}}}
	default:
		var issue publicationIssue
		if args[2] == "POST" {
			issue = publicationTestIssue(r.next, "", "")
			r.next++
		} else {
			parts := strings.Split(endpoint, "/")
			number, _ := strconv.Atoi(parts[len(parts)-1])
			issue = r.issues[number]
			if issue.ID == "" {
				r.t.Fatal("update target missing")
			}
		}
		issue.Title, issue.Body = payload["title"].(string), payload["body"].(string)
		issue.Labels = nil
		for _, label := range payload["labels"].([]any) {
			issue.Labels = append(issue.Labels, struct {
				Name string `json:"name"`
			}{label.(string)})
		}
		if state, exists := payload["state"]; exists {
			issue.State = state.(string)
		}
		r.issues[issue.Number] = issue
		response = publicationREST(issue)
	}
	if r.writes == r.failAt && r.lostResponse {
		return RunResult{}, errors.New("response lost")
	}
	raw, err := json.Marshal(response)
	return RunResult{Stdout: raw}, err
}

type publicationWriteEvents struct{ calls int }

func (e *publicationWriteEvents) Publish(context.Context, application.Event) error {
	e.calls++
	return nil
}

func publicationWriteFixture(t *testing.T) (*Adapter, *publicationWriteRunner, application.PublicationPreviewInput) {
	runner := newPublicationWriteRunner(t)
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: filepath.Join(t.TempDir(), "project with spaces"), Runner: runner})
	request := publicationTestRequest()
	input := application.PublicationPreviewInput{Target: request.Target, Source: request.Source, Artifacts: []application.PublicationArtifact{{Artifact: request.Artifacts[0], Title: "A", Content: "## Source\n" + request.Source.URL + "\n\n## Acceptance Criteria\n" + strings.Repeat("Keep full content: $(no shell) @file — criteria.\n", 1500), Readiness: planning.ReadinessReady}}}
	return adapter, runner, input
}

func runPublication(t *testing.T, adapter *Adapter, input application.PublicationPreviewInput, events *publicationWriteEvents) (application.PublicationApplyResult, error) {
	t.Helper()
	service := application.New(nil, application.Options{})
	plan, err := service.PreviewPublication(context.Background(), adapter.PublicationTarget(), input, publicationAllow{})
	if err != nil {
		return application.PublicationApplyResult{}, err
	}
	return service.ApplyPublication(context.Background(), adapter.PublicationTarget(), application.PublicationApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}, publicationAllow{}, events)
}

func TestAdoptionUpdatesExplicitUnmanagedIssueThenPersistsMapping(t *testing.T) {
	runner := newPublicationWriteRunner(t)
	issue := publicationTestIssue(42, "A", "unmanaged body")
	issue.Labels = nil
	runner.issues[issue.Number] = issue
	root := filepath.Join(t.TempDir(), "project with spaces")
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: root, Runner: runner})
	artifact := planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: "a"}
	source := application.ExternalReference{Provider: "github", Kind: "discussion", OpaqueID: "https://github.com/owner/repo/discussions/49", DisplayID: "49", URL: "https://github.com/owner/repo/discussions/49"}
	issueURL := "https://github.com/owner/repo/issues/42"
	intent := application.AdoptionPreviewInput{
		Target:    application.ExternalReference{Provider: "github", Kind: "repository", OpaqueID: "owner/repo", URL: "https://github.com/owner/repo"},
		Source:    &source,
		Artifacts: []application.PublicationArtifact{{Artifact: artifact, Title: "A", Content: "## Source\n" + source.URL + "\n\nCanonical body.", Readiness: planning.ReadinessReady}},
		Candidates: []application.ArtifactExternalReference{{
			Artifact:  artifact,
			Reference: application.ExternalReference{Provider: "github", Kind: "issue", OpaqueID: issueURL, DisplayID: "42", URL: issueURL},
		}},
	}
	service := application.New(nil, application.Options{ModuleID: "planning"})
	plan, err := service.PreviewAdoption(context.Background(), adapter.PublicationTarget(), adapter.ExternalMappingRepository(), intent, publicationAllow{})
	if err != nil || len(plan.Publication.Actions) != 1 || plan.Publication.Actions[0].Action != application.MutationUpdate {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	events := &publicationWriteEvents{}
	result, err := service.ApplyAdoption(context.Background(), adapter.PublicationTarget(), adapter.ExternalMappingRepository(), application.AdoptionApplyInput{Intent: intent, ExpectedPlan: plan, Confirmed: true}, publicationAllow{}, events)
	if err != nil || runner.issues[42].Body != intent.Artifacts[0].Content || len(result.Mappings.ArtifactReferences) != 1 || result.Event == nil || events.calls != 1 {
		t.Fatalf("result=%+v issue=%+v events=%d err=%v", result, runner.issues[42], events.calls, err)
	}
	writes := runner.writes
	plan, err = service.PreviewAdoption(context.Background(), adapter.PublicationTarget(), adapter.ExternalMappingRepository(), intent, publicationAllow{})
	if err != nil || plan.Publication.Actions[0].Action != application.MutationReuse {
		t.Fatalf("rerun plan=%+v err=%v", plan, err)
	}
	result, err = service.ApplyAdoption(context.Background(), adapter.PublicationTarget(), adapter.ExternalMappingRepository(), application.AdoptionApplyInput{Intent: intent, ExpectedPlan: plan, Confirmed: true}, publicationAllow{}, events)
	if err != nil || runner.writes != writes || result.Event != nil || events.calls != 1 {
		t.Fatalf("rerun result=%+v writes=%d events=%d err=%v", result, runner.writes, events.calls, err)
	}
}

func TestPublicationCreateUpdateAndRerunPreserveContent(t *testing.T) {
	adapter, runner, input := publicationWriteFixture(t)
	events := &publicationWriteEvents{}
	result, err := runPublication(t, adapter, input, events)
	if err != nil || len(runner.issues) != 1 || runner.writes != 3 || events.calls != 1 || result.Evidence.Completed[0].References[2].Kind != "issue" {
		t.Fatalf("%+v %v", result, err)
	}
	if runner.issues[101].Body != input.Artifacts[0].Content {
		t.Fatal("lost structured brief")
	}
	if _, err := os.Stat(adapter.projectRoot); !os.IsNotExist(err) {
		t.Fatal("publication wrote local metadata", err)
	}
	writes := runner.writes
	result, err = runPublication(t, adapter, input, events)
	if err != nil || runner.writes != writes || events.calls != 1 || result.Evidence.Completed[0].Action.Action != application.MutationReuse {
		t.Fatalf("rerun: %+v %v", result, err)
	}
	issue := runner.issues[101]
	issue.Labels = append(issue.Labels, struct {
		Name string `json:"name"`
	}{"user-owned"})
	runner.issues[101] = issue
	input.Artifacts[0].Content += "\nUpdated verification."
	input.Artifacts[0].Readiness = planning.ReadinessBlocked
	result, err = runPublication(t, adapter, input, events)
	if err != nil || len(runner.issues) != 1 || !publicationHasLabel(runner.issues[101], "user-owned") || publicationHasLabel(runner.issues[101], "plan:ready") || !publicationHasLabel(runner.issues[101], "plan:blocked") {
		t.Fatalf("update: %+v %v", result, err)
	}
	writes = runner.writes
	if _, err := runPublication(t, adapter, input, events); err != nil || runner.writes != writes {
		t.Fatal("updated rerun mutated", err)
	}
}

func TestPublicationPartialAndLostResponseRecoverWithoutDuplicates(t *testing.T) {
	for _, lost := range []bool{false, true} {
		adapter, runner, input := publicationWriteFixture(t)
		runner.failAt, runner.lostResponse = 3, lost
		events := &publicationWriteEvents{}
		result, err := runPublication(t, adapter, input, events)
		var integration *application.IntegrationError
		if !errors.As(err, &integration) || integration.Class != application.IntegrationPartialFailure || result.Evidence.Failed == nil || result.Evidence.ManualFallbackAllowed || events.calls != 1 || strings.Contains(err.Error(), "secret") {
			t.Fatalf("failure: %+v %v", result, err)
		}
		runner.failAt = 0
		if _, err := runPublication(t, adapter, input, events); err != nil || len(runner.issues) != 1 {
			t.Fatal("recovery duplicated issue", err)
		}
		if lost && runner.writes != 3 {
			t.Fatal("lost response repeated writes")
		}
	}
}

func TestPublicationRelationshipsAndPartialRerun(t *testing.T) {
	adapter, runner, input := publicationWriteFixture(t)
	group := planning.ArtifactRef{Kind: planning.ArtifactInitiative, ID: "group"}
	input.Artifacts[0].Group = &group
	input.Artifacts = append(input.Artifacts, application.PublicationArtifact{Artifact: group, Title: "Group", Content: input.Source.URL, Readiness: planning.ReadinessReady}, application.PublicationArtifact{Artifact: planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: "b"}, Title: "B", Content: input.Source.URL, Readiness: planning.ReadinessBlocked, Group: &group, Dependencies: []planning.ArtifactRef{input.Artifacts[0].Artifact}})
	runner.failAt = 8 // first relationship, after four labels and three issues
	events := &publicationWriteEvents{}
	result, err := runPublication(t, adapter, input, events)
	if err == nil || len(result.Evidence.Completed) != 3 || len(runner.issues) != 3 || result.Evidence.Failed.Action.Kind != application.PublicationRelationshipAction {
		t.Fatalf("partial: %+v %v", result, err)
	}
	runner.failAt = 0
	if _, err := runPublication(t, adapter, input, events); err != nil {
		t.Fatal("relationship recovery", err)
	}
	if len(runner.issues) != 3 || len(runner.links) != 2 {
		t.Fatal("lost grouping/dependencies", runner.links)
	}
	writes := runner.writes
	if _, err := runPublication(t, adapter, input, events); err != nil || runner.writes != writes {
		t.Fatal("complete rerun mutated", err)
	}
}

func TestPublicationRejectsUnsafePlansBeforeMutation(t *testing.T) {
	for _, change := range []string{"source", "source-url", "slug", "readiness", "order", "version"} {
		t.Run(change, func(t *testing.T) {
			adapter, runner, input := publicationWriteFixture(t)
			plan, err := application.New(nil, application.Options{}).PreviewPublication(context.Background(), adapter.PublicationTarget(), input, publicationAllow{})
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "source":
				plan.Source = nil
			case "source-url":
				plan.Source.URL += "/"
				plan.Actions[0].Artifact.Content = plan.Source.URL
			case "slug":
				plan.Actions[0].Artifact.Title = "Unrecoverable"
			case "readiness":
				plan.Actions[0].Artifact.Readiness = planning.ReadinessClarifying
			case "order":
				plan.Actions = append(plan.Actions, plan.Actions[0])
			case "version":
				plan.SchemaVersion = 2
			}
			if _, err := adapter.PublicationTarget().Apply(context.Background(), plan); err == nil || runner.writes != 0 {
				t.Fatal("unsafe plan mutated", err)
			}
		})
	}
}

func TestPublicationRevisionNormalizesTransportShape(t *testing.T) {
	issue := publicationTestIssue(101, "A", "body")
	rest := publicationREST(issue)
	rest["state"] = "OPEN"
	raw, _ := json.Marshal(rest)
	var decoded publicationIssue
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := validatePublicationIssue(&decoded, "owner/repo"); err != nil {
		t.Fatal(err)
	}
	slices.Reverse(decoded.Labels)
	if !reflect.DeepEqual(publicationIssueReference(issue), publicationIssueReference(decoded)) {
		t.Fatal("REST and CLI revisions disagree")
	}
}

func TestPublicationUpdateGuardsLastReadAndKeepsMetadata(t *testing.T) {
	adapter, runner, input := publicationWriteFixture(t)
	events := &publicationWriteEvents{}
	if _, err := runPublication(t, adapter, input, events); err != nil {
		t.Fatal(err)
	}
	store := newFileStateStore(adapter.projectRoot)
	state := normalizeGitHubState(githubState{Repo: "owner/repo"})
	state.Planning["a"] = githubPlanningRecord{Slug: "a", Kind: "spec", IssueNumber: 101}
	if err := store.write(state); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	input.Artifacts[0].Title = "Renamed mapped issue"
	input.Artifacts[0].Content = "New canonical body without a discussion link"
	writes := runner.writes
	runner.beforeRead = func(endpoint string) {
		if endpoint == "repos/owner/repo/issues/101" {
			issue := runner.issues[101]
			issue.Body = "Concurrent edit"
			runner.issues[101] = issue
		}
	}
	if _, err := runPublication(t, adapter, input, events); err == nil || runner.writes != writes {
		t.Fatal("overwrote concurrent edit", err)
	}
	runner.beforeRead = nil
	if _, err := runPublication(t, adapter, input, events); err != nil {
		t.Fatal("mapped update failed", err)
	}
	after, err := os.ReadFile(store.path)
	if err != nil || string(before) != string(after) {
		t.Fatal("publication rewrote mappings", err)
	}
	writes = runner.writes
	if _, err := runPublication(t, adapter, input, events); err != nil || runner.writes != writes {
		t.Fatal("renamed mapping was not reusable", err)
	}
}

func TestPublicationRejectsForeignOrOwnerlessMappings(t *testing.T) {
	for _, repo := range []string{"another/repo", ""} {
		t.Run(repo, func(t *testing.T) {
			adapter, runner, input := publicationWriteFixture(t)
			events := &publicationWriteEvents{}
			if _, err := runPublication(t, adapter, input, events); err != nil {
				t.Fatal(err)
			}
			state := normalizeGitHubState(githubState{Repo: repo})
			state.Planning["a"] = githubPlanningRecord{Slug: "a", Kind: "spec", IssueNumber: 101}
			if err := newFileStateStore(adapter.projectRoot).write(state); err != nil {
				t.Fatal(err)
			}
			input.Artifacts[0].Title = "Renamed mapped issue"
			input.Artifacts[0].Content = "No recoverable source link"
			writes := runner.writes
			_, err := runPublication(t, adapter, input, events)
			var integration *application.IntegrationError
			if !errors.As(err, &integration) || integration.Class != application.IntegrationAmbiguousIdentity || runner.writes != writes {
				t.Fatalf("trusted invalid metadata: %v", err)
			}
		})
	}
}

func TestPublicationRequestPreservesCancellationAndAuthClasses(t *testing.T) {
	for _, failure := range []string{"canceled", "unauthenticated", "unauthorized", "uncertain"} {
		t.Run(failure, func(t *testing.T) {
			runner := newPublicationWriteRunner(t)
			runner.failAt = 1
			ctx := context.Background()
			if failure == "canceled" {
				cancelCtx, cancel := context.WithCancel(ctx)
				ctx = cancelCtx
				runner.beforeWrite = cancel
			} else if failure == "unauthenticated" {
				runner.failStderr = "gh auth login"
			} else if failure == "unauthorized" {
				runner.failStderr = "HTTP 403"
			}
			adapter := New(Config{Enabled: true}, Options{ProjectRoot: t.TempDir(), Runner: runner})
			_, err := adapter.publicationRequest(ctx, "POST", "repos/owner/repo/issues", map[string]string{"title": "A"})
			if failure == "canceled" {
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				return
			}
			var integration *application.IntegrationError
			if !errors.As(err, &integration) {
				t.Fatal(err)
			}
			want := application.IntegrationPartialFailure
			if failure == "unauthenticated" {
				want = application.IntegrationUnauthenticated
			} else if failure == "unauthorized" {
				want = application.IntegrationUnauthorized
			}
			if integration.Class != want {
				t.Fatalf("class=%s want=%s", integration.Class, want)
			}
		})
	}
}

func TestPublicationRefusesLossOfUnmappedRecoveryIdentity(t *testing.T) {
	adapter, runner, input := publicationWriteFixture(t)
	events := &publicationWriteEvents{}
	if _, err := runPublication(t, adapter, input, events); err != nil {
		t.Fatal(err)
	}
	input.Artifacts[0].Title = "Different slug"
	writes := runner.writes
	if _, err := runPublication(t, adapter, input, events); err == nil || runner.writes != writes {
		t.Fatal("lost recovery identity", err)
	}
}

func TestPublicationRejectsIncompleteLabelListingAndMissingInputRunner(t *testing.T) {
	adapter, runner, input := publicationWriteFixture(t)
	for i := range 100 {
		runner.labels[fmt.Sprintf("label-%d", i)] = true
	}
	if _, err := runPublication(t, adapter, input, &publicationWriteEvents{}); err == nil || runner.writes != 0 {
		t.Fatal("truncated labels allowed writes", err)
	}
	adapter.runner = &publicationRunner{t: t, listed: []publicationIssue{}}
	_, err := runPublication(t, adapter, input, &publicationWriteEvents{})
	var integration *application.IntegrationError
	if !errors.As(err, &integration) || integration.Class != application.IntegrationUnsupportedCapability {
		t.Fatal(err)
	}
}

func TestPublicationRejectsCoordinationActionsBeforeProviderWrites(t *testing.T) {
	for _, kind := range []string{"group", "workspace"} {
		t.Run(kind, func(t *testing.T) {
			adapter, runner, input := publicationWriteFixture(t)
			if kind == "group" {
				input.Group = &application.PublicationGroup{Title: "Milestone"}
			} else {
				input.Workspace = &application.PublicationWorkspaceDecision{Choice: application.PublicationWorkspaceSkip}
			}
			plan, err := application.New(nil, application.Options{}).PreviewPublication(context.Background(), adapter.PublicationTarget(), input, publicationAllow{})
			if err != nil {
				t.Fatal(err)
			}
			_, err = adapter.PublicationTarget().Apply(context.Background(), plan)
			var integration *application.IntegrationError
			if !errors.As(err, &integration) || integration.Class != application.IntegrationUnsupportedCapability || runner.writes != 0 {
				t.Fatalf("coordination action reached provider: writes=%d err=%v", runner.writes, err)
			}
		})
	}
}
