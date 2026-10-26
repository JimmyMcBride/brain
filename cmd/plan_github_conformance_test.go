package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	officialplanning "github.com/JimmyMcBride/brain/internal/official/planning"
	"github.com/JimmyMcBride/brain/internal/planning/conformance"
	planninggithub "github.com/JimmyMcBride/brain/planning/github"
)

type nativeConformanceState struct {
	Repository struct {
		NameWithOwner string `json:"name_with_owner"`
		URL           string `json:"url"`
		DefaultBranch string `json:"default_branch"`
		IssuesEnabled bool   `json:"issues_enabled"`
	} `json:"repository"`
	CurrentBranch       string                                 `json:"current_branch"`
	CurrentSHA          string                                 `json:"current_sha"`
	PlanningPullRequest *nativeConformancePullRequest          `json:"planning_pull_request"`
	Discussions         map[string]nativeConformanceDiscussion `json:"discussions"`
	Issues              map[string]nativeConformanceIssue      `json:"issues"`
	Projects            map[string]nativeConformanceProject    `json:"projects"`
}

type nativeConformancePullRequest struct {
	NodeID   string `json:"node_id"`
	Number   int    `json:"number"`
	URL      string `json:"url"`
	State    string `json:"state"`
	HeadRef  string `json:"head_ref"`
	BaseRef  string `json:"base_ref"`
	IsDraft  bool   `json:"is_draft"`
	IsMerged bool   `json:"is_merged"`
	HeadSHA  string `json:"head_sha"`
}

type nativeConformanceDiscussion struct {
	NodeID   string   `json:"node_id"`
	Number   int      `json:"number"`
	URL      string   `json:"url"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Revision string   `json:"revision"`
	Comments []string `json:"comments"`
}

type nativeConformanceIssue struct {
	NodeID string   `json:"node_id"`
	Number int      `json:"number"`
	URL    string   `json:"url"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	State  string   `json:"state"`
	Labels []string `json:"labels"`
}

type nativeConformanceField struct {
	NodeID  string            `json:"node_id"`
	Type    string            `json:"type"`
	Options map[string]string `json:"options"`
}

type nativeConformanceProjectItem struct {
	NodeID      string            `json:"node_id"`
	IssueNumber int               `json:"issue_number"`
	Values      map[string]string `json:"values"`
}

type nativeConformanceProject struct {
	Owner  string                                  `json:"owner"`
	Number int                                     `json:"number"`
	NodeID string                                  `json:"node_id"`
	URL    string                                  `json:"url"`
	Title  string                                  `json:"title"`
	Fields map[string]nativeConformanceField       `json:"fields"`
	Items  map[string]nativeConformanceProjectItem `json:"items"`
}

type nativeConformanceRunner struct {
	t      *testing.T
	state  nativeConformanceState
	calls  []string
	writes int
}

func loadNativeConformanceRunner(t *testing.T, moduleRoot, asset string) *nativeConformanceRunner {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(moduleRoot, "internal", "planning", "conformance", "testdata", asset))
	if err != nil {
		t.Fatal(err)
	}
	runner := &nativeConformanceRunner{t: t}
	if err := json.Unmarshal(raw, &runner.state); err != nil {
		t.Fatal(err)
	}
	return runner
}

func (r *nativeConformanceRunner) Run(ctx context.Context, _ string, args ...string) (planninggithub.RunResult, error) {
	if err := ctx.Err(); err != nil {
		return planninggithub.RunResult{}, err
	}
	r.calls = append(r.calls, strings.Join(args, " "))
	var value any
	switch {
	case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
		value = map[string]any{
			"nameWithOwner": r.state.Repository.NameWithOwner,
			"url":           r.state.Repository.URL,
			"defaultBranchRef": map[string]string{
				"name": r.state.Repository.DefaultBranch,
			},
			"hasIssuesEnabled": r.state.Repository.IssuesEnabled,
		}
	case len(args) >= 2 && args[0] == "pr" && args[1] == "list":
		value = r.pullRequests()
	case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
		value = r.issueListing(args)
	case len(args) >= 2 && args[0] == "api" && args[1] == "graphql":
		value = r.graphQL(args)
	case len(args) == 4 && args[0] == "api" && args[1] == "--method" && args[2] == "GET":
		var err error
		value, err = r.restGET(args[3])
		if err != nil {
			return planninggithub.RunResult{}, err
		}
	default:
		return planninggithub.RunResult{}, fmt.Errorf("unexpected conformance provider call: %q", args)
	}
	raw, err := json.Marshal(value)
	return planninggithub.RunResult{Stdout: raw}, err
}

func (r *nativeConformanceRunner) RunInput(ctx context.Context, root string, input []byte, args ...string) (planninggithub.RunResult, error) {
	r.writes++
	return planninggithub.RunResult{}, fmt.Errorf("unexpected conformance provider mutation: %q (%d bytes)", args, len(input))
}

func (r *nativeConformanceRunner) pullRequests() []map[string]any {
	pr := r.state.PlanningPullRequest
	if pr == nil {
		return nil
	}
	owner, repo, _ := strings.Cut(r.state.Repository.NameWithOwner, "/")
	mergedAt := ""
	if pr.IsMerged {
		mergedAt = "2026-01-01T00:00:00Z"
	}
	return []map[string]any{{
		"id": pr.NodeID, "number": pr.Number, "url": pr.URL, "state": pr.State, "isDraft": pr.IsDraft,
		"mergedAt": mergedAt, "headRefName": pr.HeadRef, "baseRefName": pr.BaseRef, "headRefOid": pr.HeadSHA,
		"headRepository": map[string]string{"name": repo}, "headRepositoryOwner": map[string]string{"login": owner},
	}}
}

func (r *nativeConformanceRunner) issueListing(args []string) []map[string]any {
	wanted := ""
	for i := range args {
		if args[i] == "--label" && i+1 < len(args) {
			wanted = args[i+1]
		}
	}
	result := make([]map[string]any, 0)
	for _, issue := range r.state.Issues {
		if wanted != "" && !containsString(issue.Labels, wanted) {
			continue
		}
		result = append(result, r.issueJSON(issue))
	}
	sort.Slice(result, func(i, j int) bool { return result[i]["number"].(int) < result[j]["number"].(int) })
	return result
}

func (r *nativeConformanceRunner) graphQL(args []string) any {
	query := conformanceArgument(args, "query")
	switch {
	case strings.Contains(query, "discussion(number"):
		number := conformanceArgument(args, "number")
		discussion, ok := r.state.Discussions[number]
		if !ok {
			return map[string]any{"data": map[string]any{"repository": map[string]any{"discussion": nil}}}
		}
		comments := make([]map[string]string, 0, len(discussion.Comments))
		for _, comment := range discussion.Comments {
			comments = append(comments, map[string]string{"body": comment})
		}
		item := map[string]any{
			"id": discussion.NodeID, "number": discussion.Number, "url": discussion.URL, "title": discussion.Title,
			"body": discussion.Body, "updatedAt": "2026-01-01T00:00:00Z",
			"comments": map[string]any{"nodes": comments, "pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""}},
		}
		return map[string]any{"data": map[string]any{"repository": map[string]any{"discussion": item}}}
	case strings.Contains(query, "items(first:100)"):
		project := r.projectByID(conformanceArgument(args, "id"), true)
		return map[string]any{"data": map[string]any{"node": project}}
	case strings.Contains(query, "node(id:$id)"):
		project := r.projectByID(conformanceArgument(args, "id"), false)
		return map[string]any{"data": map[string]any{"node": project}}
	case strings.Contains(query, "projectsV2(first:100)"):
		projects := make([]any, 0, len(r.state.Projects))
		for _, project := range r.state.Projects {
			projects = append(projects, r.projectJSON(project, false))
		}
		owner, _, _ := strings.Cut(r.state.Repository.NameWithOwner, "/")
		return map[string]any{"data": map[string]any{"repositoryOwner": map[string]any{"login": owner, "projects": map[string]any{"nodes": projects, "pageInfo": map[string]bool{"hasNextPage": false}}}}}
	default:
		r.t.Fatalf("unexpected conformance GraphQL query: %s", query)
		return nil
	}
}

func (r *nativeConformanceRunner) restGET(endpoint string) (any, error) {
	if strings.HasSuffix(endpoint, "/milestones?state=all&per_page=100") || strings.Contains(endpoint, "/sub_issues?") || strings.Contains(endpoint, "/dependencies/blocked_by?") {
		return []any{}, nil
	}
	parts := strings.Split(endpoint, "/")
	if len(parts) >= 5 && parts[len(parts)-2] == "issues" {
		issue, ok := r.state.Issues[parts[len(parts)-1]]
		if !ok {
			return nil, errors.New("fixture issue is missing")
		}
		return r.issueJSON(issue), nil
	}
	return nil, fmt.Errorf("unexpected conformance GET %q", endpoint)
}

func (r *nativeConformanceRunner) issueJSON(issue nativeConformanceIssue) map[string]any {
	labels := make([]map[string]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		labels = append(labels, map[string]string{"name": label})
	}
	body := issue.Body
	if body == "" {
		body = "# " + issue.Title + "\n"
	}
	return map[string]any{"id": issue.NodeID, "node_id": issue.NodeID, "number": issue.Number, "url": issue.URL, "html_url": issue.URL, "title": issue.Title, "body": body, "state": issue.State, "labels": labels}
}

func (r *nativeConformanceRunner) projectByID(id string, execution bool) any {
	for _, project := range r.state.Projects {
		if project.NodeID == id {
			return r.projectJSON(project, execution)
		}
	}
	return nil
}

func (r *nativeConformanceRunner) projectJSON(project nativeConformanceProject, execution bool) map[string]any {
	base := map[string]any{
		"id": project.NodeID, "number": project.Number, "url": project.URL, "title": project.Title,
		"owner":        map[string]string{"__typename": "User", "login": project.Owner},
		"repositories": map[string]any{"nodes": []map[string]string{{"nameWithOwner": r.state.Repository.NameWithOwner}}, "pageInfo": map[string]bool{"hasNextPage": false}},
	}
	if !execution {
		return base
	}
	fields := make([]map[string]any, 0, len(project.Fields))
	for name, field := range project.Fields {
		options := make([]map[string]string, 0, len(field.Options))
		for optionName, id := range field.Options {
			options = append(options, map[string]string{"id": id, "name": optionName})
		}
		sort.Slice(options, func(i, j int) bool { return options[i]["name"] < options[j]["name"] })
		typename := "ProjectV2Field"
		if field.Type == "SINGLE_SELECT" {
			typename = "ProjectV2SingleSelectField"
		}
		fields = append(fields, map[string]any{"__typename": typename, "id": field.NodeID, "name": name, "dataType": field.Type, "options": options})
	}
	items := make([]map[string]any, 0, len(project.Items))
	for _, item := range project.Items {
		issue, ok := r.state.Issues[strconv.Itoa(item.IssueNumber)]
		if !ok {
			continue
		}
		entry := map[string]any{
			"id":      item.NodeID,
			"content": map[string]any{"__typename": "Issue", "id": issue.NodeID, "number": issue.Number, "url": issue.URL, "repository": map[string]string{"nameWithOwner": r.state.Repository.NameWithOwner}},
		}
		if status := item.Values["Status"]; status != "" {
			field := project.Fields["Status"]
			entry["status"] = map[string]any{"__typename": "ProjectV2ItemFieldSingleSelectValue", "optionId": field.Options[status], "name": status, "field": map[string]string{"id": field.NodeID, "name": "Status"}}
		}
		items = append(items, entry)
	}
	base["fields"] = map[string]any{"nodes": fields, "pageInfo": map[string]bool{"hasNextPage": false}}
	base["items"] = map[string]any{"nodes": items, "pageInfo": map[string]bool{"hasNextPage": false}}
	return base
}

type nativeConformanceGitRunner struct{ state nativeConformanceState }

func (r nativeConformanceGitRunner) Run(_ context.Context, _ string, args ...string) (planninggithub.RunResult, error) {
	switch strings.Join(args, " ") {
	case "symbolic-ref --quiet --short HEAD":
		return planninggithub.RunResult{Stdout: []byte(r.state.CurrentBranch + "\n")}, nil
	case "rev-parse --verify HEAD":
		return planninggithub.RunResult{Stdout: []byte(r.state.CurrentSHA + "\n")}, nil
	default:
		return planninggithub.RunResult{}, fmt.Errorf("unexpected conformance git call: %q", args)
	}
}

func TestNativeGitHubPlanningReplaysEveryMigratedManifestFamily(t *testing.T) {
	manifest, err := conformance.LoadGitHub()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		family     string
		caseID     string
		confirmed  bool
		wantError  bool
		contains   string
		alsoCovers []string
	}{
		{family: "discuss.assess.remote", caseID: "discuss.assess.remote.ready-multi", contains: "ready_multi_spec"},
		{family: "discuss.repair.remote", caseID: "discuss.repair.remote.confirmed", contains: `"applied": false`},
		{family: "discuss.promote.remote", caseID: "discuss.promote.remote.preview", contains: "\"plan\""},
		{family: "github.enable", caseID: "github.enable", confirmed: true, contains: "github_backend: github"},
		{family: "github.adopt", caseID: "github.adopt.ordered-issues", contains: `"schema_version": 1`},
		{family: "github.reconcile", caseID: "github.reconcile.planning-pr", contains: "Preview only", alsoCovers: []string{"repository.evidence"}},
		{family: "github.project.status", caseID: "github.project.status.in-progress", contains: "Preview only"},
		{family: "check.github", caseID: "check.github.project-drift", wantError: true, contains: "not attached to the configured GitHub Project"},
		{family: "guide.show.remote", caseID: "guide.show.remote.initiative-draft", contains: "\"kind\": \"guide_packet\""},
	}
	covered := map[string]bool{}
	for _, test := range tests {
		t.Run(test.family, func(t *testing.T) {
			testCase := nativeConformanceCase(t, manifest, test.caseID)
			fixture := nativeConformanceFixture(t, manifest, testCase.Fixture)
			env := newCLIEnv(t)
			fixtureRoot := filepath.Join(env.moduleRoot, "internal", "planning", "conformance", "testdata", fixture.Workspace)
			if err := os.CopyFS(env.project, os.DirFS(fixtureRoot)); err != nil {
				t.Fatal(err)
			}
			if test.family == "github.reconcile" {
				prepareNativeReconciliationFixture(t, env.project)
			}
			runner := loadNativeConformanceRunner(t, env.moduleRoot, fixture.ProviderState)
			registration := officialplanning.RegistrationWithOptions(officialplanning.Options{
				GitHubRunner: runner, GitRunner: nativeConformanceGitRunner{state: runner.state},
				Now: func() time.Time { return time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC) },
			})
			grant := []string{"--project", env.project, "modules", "grant", officialplanning.ID}
			grant = append(grant, registration.Descriptor.Permissions...)
			requireOK(t, runPlanningCLIWithRegistration(t, env, registration, grant...))
			requireOK(t, runPlanningCLIWithRegistration(t, env, registration, "--project", env.project, "modules", "enable", officialplanning.ID))

			if test.family != "github.enable" {
				requireOK(t, runPlanningCLIWithRegistration(t, env, registration, "--project", env.project, "plan", "github", "enable", "--confirm"))
				runner.calls = nil
				runner.writes = 0
			}
			args, err := manifest.NativeArgs(testCase, test.confirmed)
			if err != nil {
				t.Fatal(err)
			}
			args = append([]string{"--project", env.project}, args...)
			result := runPlanningCLIWithRegistration(t, env, registration, args...)
			combined := result.stdout + result.stderr
			if test.wantError != (result.err != nil) {
				t.Fatalf("unexpected result: stdout=%q stderr=%q error=%v", result.stdout, result.stderr, result.err)
			}
			if !strings.Contains(combined, test.contains) {
				t.Fatalf("output does not contain %q:\n%s\nerror=%v", test.contains, combined, result.err)
			}
			if strings.Contains(strings.ToLower(combined), "secret-token") {
				t.Fatalf("provider secret reached output: %s", combined)
			}
			if test.family != "github.enable" && runner.writes != 0 {
				t.Fatalf("preview/read replay performed %d provider writes", runner.writes)
			}
			if len(runner.calls) == 0 {
				t.Fatal("native replay did not reach the injected provider")
			}
			covered[test.family] = true
			for _, family := range test.alsoCovers {
				covered[family] = true
			}
		})
	}
	for _, family := range manifest.Families {
		if family.Retention == "migrate" && !covered[family.ID] {
			t.Fatalf("migrated native family %s was not replayed", family.ID)
		}
		if family.Retention == "compatibility-only" && len(family.Native) != 0 {
			t.Fatalf("compatibility-only family %s leaked into Brain", family.ID)
		}
	}
}

func prepareNativeReconciliationFixture(t *testing.T, projectRoot string) {
	t.Helper()
	specDir := filepath.Join(projectRoot, ".plan", "specs")
	if err := os.Remove(filepath.Join(specDir, "github-adapter.md")); err != nil {
		t.Fatal(err)
	}
	content := `---
created_at: "2026-05-17T00:00:00Z"
project: plan
slug: repository-evidence
status: approved
title: Repository evidence
type: spec
updated_at: "2026-05-17T00:00:00Z"
---

# Repository evidence

## Problem

Repository evidence must stay synchronized with the planning issue.

## Goals

- Preserve the reviewed repository and pull-request identity.

## Non-Goals

- Change planning ownership.

## Constraints

- Reconciliation remains preview-first.

## Verification

- Replay the pinned fake-provider transcript.
`
	if err := os.WriteFile(filepath.Join(specDir, "repository-evidence.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func nativeConformanceCase(t *testing.T, manifest conformance.GitHubManifest, id string) conformance.GitHubCase {
	t.Helper()
	for _, testCase := range manifest.Cases {
		if testCase.ID == id {
			return testCase
		}
	}
	t.Fatalf("conformance case %s is missing", id)
	return conformance.GitHubCase{}
}

func nativeConformanceFixture(t *testing.T, manifest conformance.GitHubManifest, id string) conformance.GitHubFixture {
	t.Helper()
	for _, fixture := range manifest.Fixtures {
		if fixture.ID == id {
			return fixture
		}
	}
	t.Fatalf("conformance fixture %s is missing", id)
	return conformance.GitHubFixture{}
}

func conformanceArgument(args []string, key string) string {
	prefix := key + "="
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
	return ""
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
