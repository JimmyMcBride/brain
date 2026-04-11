package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brain/internal/backup"
	"brain/internal/history"
	"brain/internal/notes"
	"brain/internal/project"
	"brain/internal/templates"
	"brain/internal/workspace"
)

type testManagers struct {
	plan    *Manager
	project *project.Manager
	notes   *notes.Manager
	root    string
}

func setupTestManagers(t *testing.T) *testManagers {
	t.Helper()
	root := t.TempDir()
	stateDir := filepath.Join(root, ".brain", "state")
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	nm := notes.New(
		workspaceSvc,
		templates.New(),
		backup.New(filepath.Join(stateDir, "backups")),
		history.New(filepath.Join(stateDir, "history.jsonl")),
	)
	pm := project.New(nm, workspaceSvc)
	if _, err := pm.Init("epics"); err != nil {
		t.Fatal(err)
	}
	return &testManagers{
		plan:    New(nm, pm),
		project: pm,
		notes:   nm,
		root:    root,
	}
}

func TestCreateItemIncludesDescriptionCriteriaAndResources(t *testing.T) {
	mgrs := setupTestManagers(t)

	container, err := mgrs.plan.CreateContainer("Auth System")
	if err != nil {
		t.Fatal(err)
	}
	if container.Path != ".brain/planning/epics/auth-system.md" {
		t.Fatalf("unexpected container path: %s", container.Path)
	}

	item, err := mgrs.plan.CreateItem(
		"Login Flow",
		"Auth System",
		"Support email and password sign-in.",
		[]string{"Validate email format", "Hash passwords"},
		[]string{"[[.brain/brainstorms/auth-ideas.md]]"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if item.Path != ".brain/planning/stories/login-flow.md" {
		t.Fatalf("unexpected item path: %s", item.Path)
	}
	if got := item.Metadata["container"]; got != "auth-system" {
		t.Fatalf("unexpected container metadata: %v", got)
	}
	if !strings.Contains(item.Content, "Support email and password sign-in.") {
		t.Fatalf("expected description in content:\n%s", item.Content)
	}
	if !strings.Contains(item.Content, "- [ ] Validate email format") {
		t.Fatalf("expected criterion in content:\n%s", item.Content)
	}
	if !strings.Contains(item.Content, "- [[.brain/brainstorms/auth-ideas.md]]") {
		t.Fatalf("expected resource in content:\n%s", item.Content)
	}
}

func TestStatusAggregatesWorkByContainer(t *testing.T) {
	mgrs := setupTestManagers(t)
	if _, err := mgrs.plan.CreateContainer("Auth System"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgrs.plan.CreateItem("Login Flow", "Auth System", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := mgrs.plan.CreateItem("Signup Flow", "Auth System", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := mgrs.plan.UpdateItem("login-flow", ItemChanges{Status: "done"}); err != nil {
		t.Fatal(err)
	}

	status, err := mgrs.plan.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.TotalItems != 2 || status.DoneItems != 1 {
		t.Fatalf("unexpected totals: %+v", status)
	}
	if len(status.Containers) != 1 || status.Containers[0].DoneItems != 1 {
		t.Fatalf("unexpected containers: %+v", status.Containers)
	}
}

func TestPromoteCreatesItemsWithBrainstormResource(t *testing.T) {
	mgrs := setupTestManagers(t)
	if _, err := mgrs.notes.Create(notes.CreateInput{
		Title:    "Auth Ideas",
		NoteType: "brainstorm",
		Template: "brainstorm.md",
		Section:  ".brain",
		Subdir:   "brainstorms",
		Body: `# Brainstorm: Auth Ideas

## Ideas

- **10:00** Login flow
- **10:05** Signup flow
`,
	}); err != nil {
		t.Fatal(err)
	}
	items, err := mgrs.plan.Promote("auth-ideas")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	login, err := os.ReadFile(filepath.Join(mgrs.root, ".brain", "planning", "stories", "login-flow.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(login), "- [[.brain/brainstorms/auth-ideas.md]]") {
		t.Fatalf("expected brainstorm link in promoted item:\n%s", string(login))
	}
}

func TestExtractIdeas_Timestamped(t *testing.T) {
	content := "## Ideas\n\n- **10:00** build the thing\n- **10:05** test the thing\n"
	ideas := extractIdeas(content)
	if len(ideas) != 2 || ideas[0] != "build the thing" {
		t.Fatalf("unexpected ideas: %+v", ideas)
	}
}
