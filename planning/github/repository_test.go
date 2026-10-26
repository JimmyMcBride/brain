package github

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JimmyMcBride/brain/planning/application"
)

type repositoryRunnerStep struct {
	want []string
	out  string
	err  error
}

type repositoryRunner struct {
	root  string
	steps []repositoryRunnerStep
	calls [][]string
}

func (r *repositoryRunner) Run(_ context.Context, root string, args ...string) (RunResult, error) {
	r.root = root
	r.calls = append(r.calls, append([]string(nil), args...))
	if len(r.steps) == 0 {
		return RunResult{}, errors.New("unexpected runner call")
	}
	step := r.steps[0]
	r.steps = r.steps[1:]
	if !reflect.DeepEqual(args, step.want) {
		return RunResult{}, errors.New("unexpected runner arguments")
	}
	return RunResult{Stdout: []byte(step.out)}, step.err
}

func TestRepositoryEvidenceReturnsCanonicalCheckedOutContext(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project with spaces")
	sha := "0123456789abcdef0123456789abcdef01234567"
	provider := &repositoryRunner{steps: []repositoryRunnerStep{
		{want: []string{"repo", "view", "--json", "nameWithOwner,url,defaultBranchRef"}, out: `{"nameWithOwner":"JimmyMcBride/brain","url":"https://github.com/JimmyMcBride/brain","defaultBranchRef":{"name":"develop"}}`},
		{want: []string{"pr", "list", "--repo", "JimmyMcBride/brain", "--head", "codex/reconcile", "--state", "all", "--limit", "100", "--json", "id,number,url,state,isDraft,mergedAt,headRefName,baseRefName,headRefOid,headRepository,headRepositoryOwner"}, out: `[{"id":"PR_node","number":81,"url":"https://github.com/JimmyMcBride/brain/pull/81","state":"OPEN","isDraft":false,"mergedAt":"","headRefName":"codex/reconcile","baseRefName":"develop","headRefOid":"` + sha + `","headRepository":{"name":"brain"},"headRepositoryOwner":{"login":"JimmyMcBride"}}]`},
	}}
	git := &repositoryRunner{steps: []repositoryRunnerStep{
		{want: []string{"symbolic-ref", "--quiet", "--short", "HEAD"}, out: "codex/reconcile\n"},
		{want: []string{"rev-parse", "--verify", "HEAD"}, out: sha + "\n"},
	}}
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: root, Runner: provider, GitRunner: git})
	evidence, err := adapter.RepositoryEvidenceSource().Current(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Repository.OpaqueID != "JimmyMcBride/brain" || evidence.Repository.DisplayID != "JimmyMcBride/brain" || evidence.DefaultRef != "develop" || evidence.CurrentRef != "codex/reconcile" || evidence.Commit != sha {
		t.Fatalf("evidence=%+v", evidence)
	}
	if evidence.PlanningChangeRequest == nil || evidence.PlanningChangeRequest.Reference.OpaqueID != "PR_node" || evidence.PlanningChangeRequest.Reference.DisplayID != "#81" || evidence.PlanningChangeRequest.State != application.ChangeRequestOpen || evidence.PlanningChangeRequest.HeadRef != "codex/reconcile" || evidence.PlanningChangeRequest.BaseRef != "develop" {
		t.Fatalf("change request=%+v", evidence.PlanningChangeRequest)
	}
	if provider.root != root || git.root != root || len(provider.steps) != 0 || len(git.steps) != 0 {
		t.Fatalf("runner roots provider=%q git=%q steps=%d/%d", provider.root, git.root, len(provider.steps), len(git.steps))
	}
}

func TestRepositoryEvidenceAllowsNoPlanningPullRequest(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef01234567"
	provider := &repositoryRunner{steps: []repositoryRunnerStep{
		{want: []string{"repo", "view", "--json", "nameWithOwner,url,defaultBranchRef"}, out: `{"nameWithOwner":"JimmyMcBride/brain","url":"https://github.com/JimmyMcBride/brain","defaultBranchRef":{"name":"develop"}}`},
		{want: []string{"pr", "list", "--repo", "JimmyMcBride/brain", "--head", "develop", "--state", "all", "--limit", "100", "--json", "id,number,url,state,isDraft,mergedAt,headRefName,baseRefName,headRefOid,headRepository,headRepositoryOwner"}, out: `[]`},
	}}
	git := &repositoryRunner{steps: []repositoryRunnerStep{
		{want: []string{"symbolic-ref", "--quiet", "--short", "HEAD"}, out: "develop\n"},
		{want: []string{"rev-parse", "--verify", "HEAD"}, out: sha + "\n"},
	}}
	evidence, err := New(Config{Enabled: true}, Options{Runner: provider, GitRunner: git}).RepositoryEvidenceSource().Current(context.Background())
	if err != nil || evidence.PlanningChangeRequest != nil {
		t.Fatalf("evidence=%+v err=%v", evidence, err)
	}
}

func TestRepositoryEvidenceClassifiesClosedAndMergedPullRequests(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef01234567"
	for _, test := range []struct {
		name     string
		state    string
		mergedAt string
		want     application.ChangeRequestState
	}{
		{name: "closed", state: "CLOSED", want: application.ChangeRequestClosed},
		{name: "merged", state: "MERGED", mergedAt: "2026-10-26T00:00:00Z", want: application.ChangeRequestMerged},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := &repositoryRunner{steps: []repositoryRunnerStep{
				{want: []string{"repo", "view", "--json", "nameWithOwner,url,defaultBranchRef"}, out: `{"nameWithOwner":"JimmyMcBride/brain","url":"https://github.com/JimmyMcBride/brain","defaultBranchRef":{"name":"develop"}}`},
				{want: []string{"pr", "list", "--repo", "JimmyMcBride/brain", "--head", "codex/reconcile", "--state", "all", "--limit", "100", "--json", "id,number,url,state,isDraft,mergedAt,headRefName,baseRefName,headRefOid,headRepository,headRepositoryOwner"}, out: `[{"id":"PR_node","number":81,"url":"https://github.com/JimmyMcBride/brain/pull/81","state":"` + test.state + `","mergedAt":"` + test.mergedAt + `","headRefName":"codex/reconcile","baseRefName":"develop","headRefOid":"` + sha + `","headRepository":{"name":"brain"},"headRepositoryOwner":{"login":"JimmyMcBride"}}]`},
			}}
			git := &repositoryRunner{steps: []repositoryRunnerStep{
				{want: []string{"symbolic-ref", "--quiet", "--short", "HEAD"}, out: "codex/reconcile\n"},
				{want: []string{"rev-parse", "--verify", "HEAD"}, out: sha + "\n"},
			}}
			evidence, err := New(Config{Enabled: true}, Options{Runner: provider, GitRunner: git}).RepositoryEvidenceSource().Current(context.Background())
			if err != nil || evidence.PlanningChangeRequest == nil || evidence.PlanningChangeRequest.State != test.want {
				t.Fatalf("evidence=%+v err=%v", evidence, err)
			}
		})
	}
}

func TestRepositoryEvidenceFailsClosedOnDetachedOrForeignPullRequest(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef01234567"
	for _, test := range []struct {
		name     string
		provider *repositoryRunner
		git      *repositoryRunner
		class    application.IntegrationErrorClass
	}{
		{
			name:     "detached",
			provider: &repositoryRunner{steps: []repositoryRunnerStep{{want: []string{"repo", "view", "--json", "nameWithOwner,url,defaultBranchRef"}, out: `{"nameWithOwner":"JimmyMcBride/brain","url":"https://github.com/JimmyMcBride/brain","defaultBranchRef":{"name":"develop"}}`}}},
			git:      &repositoryRunner{steps: []repositoryRunnerStep{{want: []string{"symbolic-ref", "--quiet", "--short", "HEAD"}, err: errors.New("detached")}}},
			class:    application.IntegrationAmbiguousIdentity,
		},
		{
			name: "foreign",
			provider: &repositoryRunner{steps: []repositoryRunnerStep{
				{want: []string{"repo", "view", "--json", "nameWithOwner,url,defaultBranchRef"}, out: `{"nameWithOwner":"JimmyMcBride/brain","url":"https://github.com/JimmyMcBride/brain","defaultBranchRef":{"name":"develop"}}`},
				{want: []string{"pr", "list", "--repo", "JimmyMcBride/brain", "--head", "codex/reconcile", "--state", "all", "--limit", "100", "--json", "id,number,url,state,isDraft,mergedAt,headRefName,baseRefName,headRefOid,headRepository,headRepositoryOwner"}, out: `[{"id":"PR_node","number":81,"url":"https://github.com/JimmyMcBride/brain/pull/81","state":"OPEN","headRefName":"codex/reconcile","baseRefName":"develop","headRefOid":"` + sha + `","headRepository":{"name":"brain"},"headRepositoryOwner":{"login":"fork-owner"}}]`},
			}},
			git: &repositoryRunner{steps: []repositoryRunnerStep{
				{want: []string{"symbolic-ref", "--quiet", "--short", "HEAD"}, out: "codex/reconcile\n"},
				{want: []string{"rev-parse", "--verify", "HEAD"}, out: sha + "\n"},
			}},
			class: application.IntegrationAmbiguousIdentity,
		},
		{
			name: "stale head",
			provider: &repositoryRunner{steps: []repositoryRunnerStep{
				{want: []string{"repo", "view", "--json", "nameWithOwner,url,defaultBranchRef"}, out: `{"nameWithOwner":"JimmyMcBride/brain","url":"https://github.com/JimmyMcBride/brain","defaultBranchRef":{"name":"develop"}}`},
				{want: []string{"pr", "list", "--repo", "JimmyMcBride/brain", "--head", "codex/reconcile", "--state", "all", "--limit", "100", "--json", "id,number,url,state,isDraft,mergedAt,headRefName,baseRefName,headRefOid,headRepository,headRepositoryOwner"}, out: `[{"id":"PR_node","number":81,"url":"https://github.com/JimmyMcBride/brain/pull/81","state":"OPEN","headRefName":"codex/reconcile","baseRefName":"develop","headRefOid":"abcdefabcdefabcdefabcdefabcdefabcdefabcd","headRepository":{"name":"brain"},"headRepositoryOwner":{"login":"JimmyMcBride"}}]`},
			}},
			git: &repositoryRunner{steps: []repositoryRunnerStep{
				{want: []string{"symbolic-ref", "--quiet", "--short", "HEAD"}, out: "codex/reconcile\n"},
				{want: []string{"rev-parse", "--verify", "HEAD"}, out: sha + "\n"},
			}},
			class: application.IntegrationRevisionConflict,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(Config{Enabled: true}, Options{Runner: test.provider, GitRunner: test.git}).RepositoryEvidenceSource().Current(context.Background())
			var integration *application.IntegrationError
			if !errors.As(err, &integration) || integration.Class != test.class {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
