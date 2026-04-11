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
	if _, err := pm.Init(); err != nil {
		t.Fatal(err)
	}
	return &testManagers{
		plan:    New(nm, pm),
		project: pm,
		notes:   nm,
		root:    root,
	}
}

func TestCreateEpicCreatesDraftSpec(t *testing.T) {
	mgrs := setupTestManagers(t)

	bundle, err := mgrs.plan.CreateEpic("Auth System", "")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Epic.Path != ".brain/planning/epics/auth-system.md" {
		t.Fatalf("unexpected epic path: %s", bundle.Epic.Path)
	}
	if bundle.Spec.Path != ".brain/planning/specs/auth-system.md" {
		t.Fatalf("unexpected spec path: %s", bundle.Spec.Path)
	}
	if got := bundle.Epic.Metadata["spec"]; got != "auth-system" {
		t.Fatalf("unexpected spec metadata on epic: %v", got)
	}
	if got := bundle.Spec.Metadata["status"]; got != "draft" {
		t.Fatalf("unexpected spec status: %v", got)
	}
}

func TestCreateStoryRequiresApprovedSpec(t *testing.T) {
	mgrs := setupTestManagers(t)
	if _, err := mgrs.plan.CreateEpic("Auth System", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := mgrs.plan.CreateStory("auth-system", "Login Flow", "", nil, nil); err == nil {
		t.Fatal("expected story creation to fail for draft spec")
	}
	if _, err := mgrs.plan.SetSpecStatus("auth-system", "approved"); err != nil {
		t.Fatal(err)
	}
	story, err := mgrs.plan.CreateStory(
		"auth-system",
		"Login Flow",
		"Support email and password sign-in.",
		[]string{"Validate email format", "Hash passwords"},
		[]string{"[[docs/project-overview.md]]"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if story.Path != ".brain/planning/stories/login-flow.md" {
		t.Fatalf("unexpected story path: %s", story.Path)
	}
	if got := story.Metadata["epic"]; got != "auth-system" {
		t.Fatalf("unexpected epic metadata: %v", got)
	}
	if got := story.Metadata["spec"]; got != "auth-system" {
		t.Fatalf("unexpected spec metadata: %v", got)
	}
	if !strings.Contains(story.Content, "- [ ] Validate email format") {
		t.Fatalf("expected criterion in story:\n%s", story.Content)
	}
	if !strings.Contains(story.Content, "- [[.brain/planning/specs/auth-system.md]]") {
		t.Fatalf("expected canonical spec link in story resources:\n%s", story.Content)
	}
}

func TestPromoteBrainstormCreatesEpicAndSeededSpec(t *testing.T) {
	mgrs := setupTestManagers(t)
	if _, err := mgrs.notes.Create(notes.CreateInput{
		Title:    "Auth Ideas",
		NoteType: "brainstorm",
		Template: "brainstorm.md",
		Section:  ".brain",
		Subdir:   "brainstorms",
		Body: `# Brainstorm: Auth Ideas

## Focus Question

How should auth work?

## Ideas

- **10:00** Email and password sign-in
- **10:05** Invite-based onboarding
`,
	}); err != nil {
		t.Fatal(err)
	}

	bundle, err := mgrs.plan.PromoteBrainstorm("auth-ideas")
	if err != nil {
		t.Fatal(err)
	}
	if got := bundle.Epic.Metadata["source_brainstorm"]; got != ".brain/brainstorms/auth-ideas.md" {
		t.Fatalf("unexpected brainstorm metadata on epic: %v", got)
	}
	if !strings.Contains(bundle.Spec.Content, "How should auth work?") {
		t.Fatalf("expected brainstorm focus question in spec:\n%s", bundle.Spec.Content)
	}
	if !strings.Contains(bundle.Spec.Content, "- Email and password sign-in") {
		t.Fatalf("expected brainstorm ideas in spec goals:\n%s", bundle.Spec.Content)
	}
	if !strings.Contains(bundle.Spec.Content, "- [[.brain/brainstorms/auth-ideas.md]]") {
		t.Fatalf("expected brainstorm resource link in spec:\n%s", bundle.Spec.Content)
	}
}

func TestLegacyEpicStoriesGetSpecMetadataBackfilled(t *testing.T) {
	mgrs := setupTestManagers(t)
	if err := os.WriteFile(filepath.Join(mgrs.root, ".brain", "project.yaml"), []byte("name: test\nplanning_model: epic_spec_v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := mgrs.notes.Create(notes.CreateInput{
		Title:    "Auth System",
		NoteType: "epic",
		Template: "epic.md",
		Section:  ".brain",
		Subdir:   "planning/epics",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgrs.notes.Create(notes.CreateInput{
		Title:    "Login Flow",
		NoteType: "story",
		Template: "story.md",
		Section:  ".brain",
		Subdir:   "planning/stories",
		Metadata: map[string]any{
			"container": "auth-system",
			"status":    "done",
		},
	}); err != nil {
		t.Fatal(err)
	}
	status, err := mgrs.plan.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Epics) != 1 {
		t.Fatalf("expected 1 epic after migration, got %d", len(status.Epics))
	}
	story, err := mgrs.notes.Read(".brain/planning/stories/login-flow.md")
	if err != nil {
		t.Fatal(err)
	}
	if story.Metadata["epic"] != "auth-system" || story.Metadata["spec"] != "auth-system" {
		t.Fatalf("expected story metadata to be backfilled, got %+v", story.Metadata)
	}
	if _, err := os.Stat(filepath.Join(mgrs.root, ".brain", "planning", "specs", "auth-system.md")); err != nil {
		t.Fatalf("expected canonical spec to be created: %v", err)
	}
}

func TestStatusAggregatesEpicAndStoryCounts(t *testing.T) {
	mgrs := setupTestManagers(t)
	if _, err := mgrs.plan.CreateEpic("Auth System", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := mgrs.plan.SetSpecStatus("auth-system", "approved"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgrs.plan.CreateStory("auth-system", "Login Flow", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := mgrs.plan.CreateStory("auth-system", "Signup Flow", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := mgrs.plan.UpdateStory("login-flow", StoryChanges{Status: "done"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgrs.plan.UpdateStory("signup-flow", StoryChanges{Status: "blocked"}); err != nil {
		t.Fatal(err)
	}

	status, err := mgrs.plan.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.TotalStories != 2 || status.DoneStories != 1 || status.BlockedStories != 1 {
		t.Fatalf("unexpected status totals: %+v", status)
	}
	if len(status.Epics) != 1 || status.Epics[0].SpecStatus != "approved" {
		t.Fatalf("unexpected epic summary: %+v", status.Epics)
	}
}
