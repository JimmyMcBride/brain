package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"
)

type publicationRunner struct {
	t             *testing.T
	listed        []publicationIssue
	direct        map[int]publicationIssue
	relationships map[string][]publicationIssue
	calls         int
}

func (r *publicationRunner) Run(_ context.Context, _ string, args ...string) (RunResult, error) {
	r.calls++
	var value any
	switch {
	case len(args) > 1 && args[0] == "issue" && args[1] == "list":
		value = r.listed
	case len(args) == 4 && args[0] == "api" && args[1] == "--method" && args[2] == "GET":
		if links, ok := r.relationships[args[3]]; ok {
			value = links
		} else {
			found := false
			for number, issue := range r.direct {
				if args[3] == fmt.Sprintf("repos/owner/repo/issues/%d", number) {
					value = issue
					found = true
				}
			}
			for _, issue := range r.listed {
				if !found && args[3] == fmt.Sprintf("repos/owner/repo/issues/%d", issue.Number) {
					value = issue
					found = true
				}
			}
			if !found && strings.Contains(args[3], "?per_page=100") {
				value = []publicationIssue{}
			} else if !found {
				return RunResult{}, errors.New("mapped issue missing")
			}
		}
	default:
		r.t.Fatalf("unexpected/mutating command: %q", args)
	}
	raw, err := json.Marshal(value)
	return RunResult{Stdout: raw}, err
}

func publicationTestIssue(number int, title, body string) publicationIssue {
	var issue publicationIssue
	raw := fmt.Sprintf(`{"id":"I_%d","number":%d,"url":"https://github.com/owner/repo/issues/%d","title":%q,"body":%q,"state":"open","labels":[{"name":"plan:spec"},{"name":"plan:ready"}]}`, number, number, number, title, body)
	if err := json.Unmarshal([]byte(raw), &issue); err != nil {
		panic(err)
	}
	return issue
}

func publicationTestRequest() application.PublicationInspectRequest {
	return application.PublicationInspectRequest{Target: application.ExternalReference{Provider: "github", Kind: "repository", OpaqueID: "owner/repo"}, Source: &application.ExternalReference{Provider: "github", Kind: "discussion", URL: "https://github.com/owner/repo/discussions/49"}, Artifacts: []planning.ArtifactRef{{Kind: planning.ArtifactSpec, ID: "a"}}}
}

type publicationAllow struct{}

func (publicationAllow) Require(context.Context, string) error { return nil }

func TestPublicationRecoveryFeedsUnchangedPreviewWithoutMetadataWrites(t *testing.T) {
	body := "## Source\nhttps://github.com/owner/repo/discussions/49\n\n## Acceptance Criteria\nRetain exact criteria."
	issue := publicationTestIssue(10, "A", body)
	runner := &publicationRunner{t: t, listed: []publicationIssue{issue}}
	root := filepath.Join(t.TempDir(), "project with spaces")
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: root, Runner: runner})
	request := publicationTestRequest()
	snapshot, err := adapter.PublicationTarget().Inspect(context.Background(), request)
	if err != nil || len(snapshot.Artifacts) != 1 {
		t.Fatalf("%+v %v", snapshot, err)
	}
	if snapshot.Artifacts[0].Reference.OpaqueID != "I_10" || snapshot.Artifacts[0].Content != body || snapshot.Artifacts[0].Readiness != planning.ReadinessReady {
		t.Fatal(snapshot)
	}
	input := application.PublicationPreviewInput{Target: request.Target, Source: request.Source, Artifacts: snapshot.Artifacts}
	plan, err := application.New(nil, application.Options{}).PreviewPublication(context.Background(), adapter.PublicationTarget(), input, publicationAllow{})
	if err != nil || len(plan.Actions) != 1 || plan.Actions[0].Action != application.MutationUnchanged {
		t.Fatalf("%+v %v", plan, err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("inspection created workspace", err)
	}
}

func TestPublicationReadsRenamedUnlabelledMappedIssue(t *testing.T) {
	root := t.TempDir()
	store := newFileStateStore(root)
	state := normalizeGitHubState(githubState{Repo: "owner/repo"})
	state.Planning["a"] = githubPlanningRecord{Slug: "a", Kind: "spec", IssueNumber: 10, DiscussionURL: publicationTestRequest().Source.URL}
	if err := store.write(state); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	issue := publicationTestIssue(10, "Renamed", "new body")
	issue.Labels = nil
	issue.NodeID = issue.ID
	issue.ID = ""
	issue.HTMLURL = issue.URL
	issue.URL = "https://api.github.com/repos/owner/repo/issues/10"
	runner := &publicationRunner{t: t, listed: []publicationIssue{}, direct: map[int]publicationIssue{10: issue}}
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: root, Runner: runner})
	snapshot, err := adapter.PublicationTarget().Inspect(context.Background(), publicationTestRequest())
	if err != nil || len(snapshot.Artifacts) != 1 || snapshot.Artifacts[0].Title != "Renamed" {
		t.Fatalf("%+v %v", snapshot, err)
	}
	after, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("inspection rewrote metadata")
	}
	delete(runner.direct, 10)
	if _, err := adapter.PublicationTarget().Inspect(context.Background(), publicationTestRequest()); err == nil {
		t.Fatal("missing known object treated as create")
	}
}

func TestPublicationResolvesExplicitAdoptionCandidate(t *testing.T) {
	issue := publicationTestIssue(10, "Unmanaged title", "unmanaged body")
	issue.Labels = nil
	runner := &publicationRunner{t: t, listed: []publicationIssue{}, direct: map[int]publicationIssue{10: issue}}
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: t.TempDir(), Runner: runner})
	request := publicationTestRequest()
	request.Candidates = []application.ArtifactExternalReference{{
		Artifact: request.Artifacts[0],
		Reference: application.ExternalReference{
			Provider: "github", OpaqueID: issue.URL, Kind: "issue", DisplayID: "10", URL: issue.URL,
		},
	}}
	snapshot, err := adapter.PublicationTarget().Inspect(context.Background(), request)
	if err != nil || len(snapshot.Artifacts) != 1 || snapshot.Artifacts[0].Reference.OpaqueID != "I_10" || snapshot.Artifacts[0].Title != "Unmanaged title" {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	request.Candidates[0].Reference.OpaqueID = "different-node"
	if _, err := adapter.PublicationTarget().Inspect(context.Background(), request); err == nil {
		t.Fatal("accepted candidate with conflicting opaque identity")
	}
}

func TestPublicationRecoveryRejectsAmbiguityAndIgnoresTitleOnlyMatch(t *testing.T) {
	runner := &publicationRunner{t: t, listed: []publicationIssue{publicationTestIssue(10, "A", "same title only"), publicationTestIssue(11, "A", "https://github.com/owner/repo/discussions/490")}}
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: t.TempDir(), Runner: runner})
	snapshot, err := adapter.PublicationTarget().Inspect(context.Background(), publicationTestRequest())
	if err != nil || len(snapshot.Artifacts) != 0 {
		t.Fatalf("title/prefix guessed: %+v %v", snapshot, err)
	}
	for i := range runner.listed {
		runner.listed[i].Body = publicationTestRequest().Source.URL
	}
	_, err = adapter.PublicationTarget().Inspect(context.Background(), publicationTestRequest())
	var integration *application.IntegrationError
	if !errors.As(err, &integration) || integration.Class != application.IntegrationAmbiguousIdentity {
		t.Fatal(err)
	}
}

func TestPublicationRelationshipsAndIncompleteScope(t *testing.T) {
	a := publicationTestIssue(10, "A", publicationTestRequest().Source.URL)
	b := publicationTestIssue(11, "B", publicationTestRequest().Source.URL)
	runner := &publicationRunner{t: t, listed: []publicationIssue{b, a}, relationships: map[string][]publicationIssue{"repos/owner/repo/issues/11/dependencies/blocked_by?per_page=100": {a}}}
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: t.TempDir(), Runner: runner})
	request := publicationTestRequest()
	request.Artifacts = append(request.Artifacts, planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: "b"})
	snapshot, err := adapter.PublicationTarget().Inspect(context.Background(), request)
	if err != nil || len(snapshot.Relationships) != 1 || len(snapshot.Artifacts[1].Dependencies) != 1 || snapshot.Artifacts[1].Dependencies[0].ID != "a" {
		t.Fatalf("%+v %v", snapshot, err)
	}
	request.Artifacts = request.Artifacts[1:]
	_, err = adapter.PublicationTarget().Inspect(context.Background(), request)
	var integration *application.IntegrationError
	if !errors.As(err, &integration) || integration.Class != application.IntegrationManualRemediationRequired {
		t.Fatal(err)
	}
}

func TestPublicationListingLimitsAndRepositoryMismatch(t *testing.T) {
	runner := &publicationRunner{t: t, listed: make([]publicationIssue, 1000)}
	root := t.TempDir()
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: root, Runner: runner})
	if _, err := adapter.PublicationTarget().Inspect(context.Background(), publicationTestRequest()); err == nil {
		t.Fatal("accepted truncated listing")
	}
	if err := newFileStateStore(root).write(githubState{Repo: "another/repo"}); err != nil {
		t.Fatal(err)
	}
	calls := runner.calls
	if _, err := adapter.PublicationTarget().Inspect(context.Background(), publicationTestRequest()); err == nil || runner.calls != calls {
		t.Fatal("mismatched repository reached provider")
	}
}

func TestPublicationIssueDecodesRESTNumericID(t *testing.T) {
	var issue publicationIssue
	if err := json.Unmarshal([]byte(`{"id":12345,"node_id":"I_node","number":10,"html_url":"https://github.com/owner/repo/issues/10"}`), &issue); err != nil {
		t.Fatal(err)
	}
	if err := validatePublicationIssue(&issue, "owner/repo"); err != nil {
		t.Fatal(err)
	}
	if issue.ID != "I_node" {
		t.Fatalf("identity=%q", issue.ID)
	}
}

func TestPublicationIssueNormalizesNodeIdentity(t *testing.T) {
	for _, raw := range []string{`{"id":"I_node"}`, `{"id":123,"node_id":"I_node"}`} {
		var issue publicationIssue
		if err := json.Unmarshal([]byte(raw), &issue); err != nil {
			t.Fatal(err)
		}
		if issue.ID != "I_node" || issue.NodeID != "I_node" {
			t.Fatalf("inconsistent identity: %+v", issue)
		}
	}
	var issue publicationIssue
	if err := json.Unmarshal([]byte(`{"id":"I_a","node_id":"I_b"}`), &issue); err == nil {
		t.Fatal("accepted conflicting node identities")
	}
}

func TestPublicationSourceLinkBoundaries(t *testing.T) {
	link := "https://github.com/owner/repo/discussions/49"
	for _, body := range []string{link, "(" + link + ")", "<" + link + ">", "\n" + link + "\t", link + "0 then " + link} {
		if !publicationSourceLink(body, link) {
			t.Errorf("missed source in %q", body)
		}
	}
	for _, body := range []string{"", "prefix" + link, link + "0", link + "/suffix", link + "?query"} {
		if publicationSourceLink(body, link) {
			t.Errorf("accepted false source in %q", body)
		}
	}
	if publicationSourceLink("body", "") {
		t.Fatal("accepted empty source")
	}
}
