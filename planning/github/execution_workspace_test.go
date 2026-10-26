package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"
)

type executionRunner struct {
	t       *testing.T
	project executionProject
	reads   int
	writes  int
}

func (r *executionRunner) Run(_ context.Context, _ string, args ...string) (RunResult, error) {
	r.reads++
	if len(args) == 4 && args[0] == "api" && args[1] == "--method" && args[2] == "GET" && strings.HasPrefix(args[3], "repos/owner/repo/issues/") {
		number := 1
		if strings.HasSuffix(args[3], "/2") {
			number = 2
		}
		raw, _ := json.Marshal(publicationTestIssue(number, fmt.Sprintf("Issue %d", number), "body"))
		return RunResult{Stdout: raw}, nil
	}
	query := publicationTestArgument(args, "query")
	if len(args) < 2 || args[0] != "api" || args[1] != "graphql" || !strings.Contains(query, "items(first:100)") || !strings.Contains(query, `status: fieldValueByName(name:"Status")`) {
		r.t.Fatalf("unexpected execution read: %q", args)
	}
	raw, _ := json.Marshal(map[string]any{"data": map[string]any{"node": r.project}})
	return RunResult{Stdout: raw}, nil
}

func (r *executionRunner) RunInput(_ context.Context, _ string, input []byte, args ...string) (RunResult, error) {
	r.writes++
	if strings.Join(args, " ") != "api --method POST graphql --input -" {
		r.t.Fatalf("unexpected execution write: %q", args)
	}
	var payload struct {
		Query     string            `json:"query"`
		Variables map[string]string `json:"variables"`
	}
	if err := json.Unmarshal(input, &payload); err != nil {
		r.t.Fatal(err)
	}
	var value any
	switch {
	case strings.Contains(payload.Query, "updateProjectV2ItemFieldValue"):
		for i := range r.project.Items.Nodes {
			if r.project.Items.Nodes[i].ID == payload.Variables["itemId"] {
				for _, option := range r.project.Fields.Nodes[0].Options {
					if option.ID == payload.Variables["optionId"] {
						r.project.Items.Nodes[i].Status.OptionID = option.ID
						r.project.Items.Nodes[i].Status.Name = option.Name
					}
				}
			}
		}
		value = map[string]any{"data": map[string]any{"updateProjectV2ItemFieldValue": map[string]any{"projectV2Item": map[string]string{"id": payload.Variables["itemId"]}}}}
	case strings.Contains(payload.Query, "addProjectV2ItemById"):
		if payload.Variables["contentId"] != "I_2" {
			r.t.Fatalf("attachment content id = %q, want resolved node id", payload.Variables["contentId"])
		}
		item := executionTestItem("PVTI_2", "I_2", 2, "Todo", "todo")
		r.project.Items.Nodes = append(r.project.Items.Nodes, item)
		value = map[string]any{"data": map[string]any{"addProjectV2ItemById": map[string]any{"item": map[string]string{"id": item.ID}}}}
	default:
		r.t.Fatalf("unexpected mutation: %s", payload.Query)
	}
	raw, _ := json.Marshal(value)
	return RunResult{Stdout: raw}, nil
}

func executionTestProject() executionProject {
	var project executionProject
	project.publicationProject = publicationTestProject(12, "Delivery", "owner/repo")
	project.Fields.Nodes = []executionFieldNode{{Typename: "ProjectV2SingleSelectField", ID: "PVTF_status", Name: "Status", DataType: "SINGLE_SELECT", Options: []executionOption{{ID: "todo", Name: "Todo"}, {ID: "progress", Name: "In Progress"}, {ID: "review", Name: "In Review"}, {ID: "done", Name: "Done"}}}}
	project.Items.Nodes = []executionItemNode{executionTestItem("PVTI_1", "I_1", 1, "Todo", "todo")}
	return project
}

func executionTestItem(itemID, issueID string, number int, status, optionID string) executionItemNode {
	var item executionItemNode
	item.ID = itemID
	item.Content = &struct {
		Typename   string `json:"__typename"`
		ID         string `json:"id"`
		Number     int    `json:"number"`
		URL        string `json:"url"`
		Repository struct {
			NameWithOwner string `json:"nameWithOwner"`
		} `json:"repository"`
	}{Typename: "Issue", ID: issueID, Number: number, URL: fmt.Sprintf("https://github.com/owner/repo/issues/%d", number)}
	item.Content.Repository.NameWithOwner = "owner/repo"
	item.Status = &struct {
		Typename string `json:"__typename"`
		OptionID string `json:"optionId"`
		Name     string `json:"name"`
		Field    *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"field"`
	}{Typename: "ProjectV2ItemFieldSingleSelectValue", OptionID: optionID, Name: status, Field: &struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: "PVTF_status", Name: "Status"}}
	return item
}

func executionTestAdapter(t *testing.T) (*Adapter, *executionRunner) {
	root := t.TempDir()
	store := newFileStateStore(root)
	state := normalizeGitHubState(githubState{Repo: "owner/repo", RepoURL: "https://github.com/owner/repo"})
	state.Planning["one"] = githubPlanningRecord{Slug: "one", Kind: "spec", IssueNumber: 1, IssueURL: "https://github.com/owner/repo/issues/1"}
	state.Planning["two"] = githubPlanningRecord{Slug: "two", Kind: "spec", IssueNumber: 2, IssueURL: "https://github.com/owner/repo/issues/2"}
	state.ProjectDecisions["delivery"] = githubProjectDecision{Slug: "delivery", ProjectOwner: "owner", ProjectNumber: 12, ProjectID: "PVT_12", ProjectURL: "https://github.com/users/owner/projects/12", FieldIDs: map[string]string{"Status": "PVTF_status"}}
	if err := store.write(state); err != nil {
		t.Fatal(err)
	}
	runner := &executionRunner{t: t, project: executionTestProject()}
	return New(Config{Enabled: true}, Options{ProjectRoot: root, Runner: runner}), runner
}

func TestExecutionWorkspaceInspectsAndUpdatesStatusIdempotently(t *testing.T) {
	adapter, runner := executionTestAdapter(t)
	ref := publicationProjectReference(runner.project.publicationProject)
	snapshot, err := adapter.ExecutionWorkspace().Inspect(context.Background(), ref)
	if err != nil || len(snapshot.Fields) != 1 || len(snapshot.Fields[0].SupportedStatuses) != 4 || len(snapshot.Items) != 1 || snapshot.Items[0].Status != application.ExecutionTodo {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	request := application.ExecutionStatusRequest{Workspace: snapshot.Workspace, WorkItem: snapshot.Items[0].Reference, Status: application.ExecutionInReview}
	item, err := adapter.ExecutionWorkspace().ApplyStatus(context.Background(), request)
	if err != nil || item.Status != application.ExecutionInReview || runner.writes != 1 {
		t.Fatalf("item=%+v writes=%d err=%v", item, runner.writes, err)
	}
	fresh, err := adapter.ExecutionWorkspace().Inspect(context.Background(), snapshot.Workspace)
	if err != nil {
		t.Fatal(err)
	}
	request.Workspace, request.WorkItem = fresh.Workspace, fresh.Items[0].Reference
	if _, err = adapter.ExecutionWorkspace().ApplyStatus(context.Background(), request); err != nil || runner.writes != 1 {
		t.Fatalf("idempotent writes=%d err=%v", runner.writes, err)
	}
}

func TestExecutionWorkspaceAcceptsConsistentRepeatedProjectDecisions(t *testing.T) {
	adapter, runner := executionTestAdapter(t)
	store := adapter.state.(*fileStateStore)
	state, err := store.read()
	if err != nil {
		t.Fatal(err)
	}
	state.ProjectDecisions["delivery-copy"] = githubProjectDecision{Slug: "delivery-copy", ProjectOwner: "owner", ProjectNumber: 12, ProjectID: "PVT_12", ProjectURL: "https://github.com/users/owner/projects/12", FieldIDs: map[string]string{"Status": "PVTF_status"}}
	if err := store.write(state); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.ExecutionWorkspace().Inspect(context.Background(), publicationProjectReference(runner.project.publicationProject)); err != nil {
		t.Fatal(err)
	}
}

func TestExecutionWorkspaceAttachesAndRejectsStaleRevision(t *testing.T) {
	adapter, runner := executionTestAdapter(t)
	ref := publicationProjectReference(runner.project.publicationProject)
	snapshot, err := adapter.ExecutionWorkspace().Inspect(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	request := application.ExecutionAttachRequest{Workspace: snapshot.Workspace, Artifact: planningRef("two"), PublishedWork: application.ExternalReference{Provider: providerName, Kind: "issue", OpaqueID: "https://github.com/owner/repo/issues/2", DisplayID: "2", URL: "https://github.com/owner/repo/issues/2"}}
	stale := request
	stale.Workspace.Revision = "stale"
	if _, err = adapter.ExecutionWorkspace().Attach(context.Background(), stale); !isProviderClass(err, application.IntegrationRevisionConflict) {
		t.Fatalf("stale error=%v", err)
	}
	mismatch := request
	mismatch.Artifact = planningRef("one")
	if _, err = adapter.ExecutionWorkspace().Attach(context.Background(), mismatch); !isProviderClass(err, application.IntegrationManualRemediationRequired) || runner.writes != 0 {
		t.Fatalf("mapping error=%v writes=%d", err, runner.writes)
	}
	item, err := adapter.ExecutionWorkspace().Attach(context.Background(), request)
	if err != nil || item.Artifact.ID != "two" || runner.writes != 1 {
		t.Fatalf("item=%+v writes=%d err=%v", item, runner.writes, err)
	}
}

func planningRef(id string) planning.ArtifactRef {
	return planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: planning.ArtifactID(id)}
}

func isProviderClass(err error, class application.IntegrationErrorClass) bool {
	var integration *application.IntegrationError
	return errors.As(err, &integration) && integration.Class == class
}
