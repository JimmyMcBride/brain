package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"
)

func TestWorkspaceStatusClassifiesCompatibilityWithoutMutation(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		state    application.WorkspaceState
		writable bool
	}{
		{name: "missing", state: application.WorkspaceMissing},
		{name: "compatible", fixture: "compatible", state: application.WorkspaceCompatible, writable: true},
		{name: "future", fixture: "future", state: application.WorkspaceFutureSchema},
		{name: "migration", fixture: "migration", state: application.WorkspaceMigrationRequired},
		{name: "legacy", fixture: "legacy", state: application.WorkspaceUnsupportedLegacy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.fixture != "" {
				copyFixture(t, tt.fixture, root)
			}
			before := snapshotFiles(t, root)
			status, err := New(root).Status(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if status.State != tt.state || status.Writable != tt.writable {
				t.Fatalf("unexpected status: %#v", status)
			}
			after := snapshotFiles(t, root)
			if strings.Join(before, "\n") != strings.Join(after, "\n") {
				t.Fatalf("status mutated workspace\nbefore=%v\nafter=%v", before, after)
			}
		})
	}
}

func TestAdapterReadsSchemaV3BrainstormsAndSpecs(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	adapter := New(root)

	brainstorms, err := adapter.ListBrainstorms(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(brainstorms) != 1 || brainstorms[0].Artifact.ID != "alpha" ||
		brainstorms[0].Artifact.Summary != "One compatible local brainstorm." {
		t.Fatalf("unexpected brainstorms: %#v", brainstorms)
	}
	specs, err := adapter.ListSpecs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 || specs[0].Artifact.ID != "alpha-spec" ||
		specs[0].Artifact.Status != planning.SpecApproved ||
		len(specs[0].Artifact.Verification) != 2 {
		t.Fatalf("unexpected specs: %#v", specs)
	}
	if findings := planning.ValidateBrainstorm(brainstorms[0].Artifact); hasErrorFindings(findings) {
		t.Fatalf("brainstorm did not map into valid domain value: %#v", findings)
	}
	if findings := planning.ValidateSpec(specs[0].Artifact); hasErrorFindings(findings) {
		t.Fatalf("spec did not map into valid domain value: %#v", findings)
	}
}

func TestAdapterReadsCheckDocumentsAndReplacesRoadmapAtomically(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	adapter := New(root)

	documents, err := adapter.QuerySpecs(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 1 || documents[0].Path != ".plan/specs/alpha-spec.md" || documents[0].Title != "Alpha Spec" {
		t.Fatalf("unexpected check documents: %#v", documents)
	}
	roadmap, err := adapter.ReadRoadmap(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if roadmap.Path != ".plan/ROADMAP.md" || !strings.Contains(roadmap.Body, "Deliver the fixture workflow") {
		t.Fatalf("unexpected roadmap: %#v", roadmap)
	}
	updated := "# Roadmap\n\n## Overview\n\nUpdated fixture roadmap.\n"
	roadmapPath := filepath.Join(root, ".plan", "ROADMAP.md")
	if err := os.Chmod(roadmapPath, 0o600); err != nil {
		t.Fatal(err)
	}
	roadmap, action, err := adapter.ReplaceRoadmap(context.Background(), updated)
	if err != nil {
		t.Fatal(err)
	}
	if action != application.MutationUpdate || roadmap.Body != updated {
		t.Fatalf("unexpected roadmap update: action=%s document=%#v", action, roadmap)
	}
	_, action, err = adapter.ReplaceRoadmap(context.Background(), updated)
	if err != nil {
		t.Fatal(err)
	}
	if action != application.MutationUnchanged {
		t.Fatalf("expected unchanged rerun, got %s", action)
	}
	info, err := os.Stat(roadmapPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("roadmap mode changed: got %o want 600", got)
		}
	}
	assertNoTemporaryFiles(t, filepath.Join(root, ".plan"))
}

func TestAdapterResolvesRelativeProjectRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "fixture")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	copyFixture(t, "compatible", root)
	oldWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(parent); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWorkingDirectory) })

	status, err := New("fixture").Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Project != "fixture" || status.State != application.WorkspaceCompatible {
		t.Fatalf("unexpected relative-root status: %#v", status)
	}
}

func TestAdapterConcurrentRoadmapRerunsUpdateOnce(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	adapter := New(root)
	const callers = 8
	actions := make(chan application.MutationAction, callers)
	errs := make(chan error, callers)
	var wait sync.WaitGroup
	for range callers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, action, err := adapter.ReplaceRoadmap(context.Background(), "# Concurrent Roadmap\n")
			actions <- action
			errs <- err
		}()
	}
	wait.Wait()
	close(actions)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	updates := 0
	unchanged := 0
	for action := range actions {
		switch action {
		case application.MutationUpdate:
			updates++
		case application.MutationUnchanged:
			unchanged++
		default:
			t.Fatalf("unexpected action %q", action)
		}
	}
	if updates != 1 || unchanged != callers-1 {
		t.Fatalf("expected one update and %d unchanged, got update=%d unchanged=%d", callers-1, updates, unchanged)
	}
}

func TestAdapterValidationErrorsIncludeFindingMessages(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	adapter := New(root)

	_, _, err := adapter.CreateBrainstorm(context.Background(), planning.Brainstorm{}, time.Now())
	if err == nil {
		t.Fatal("expected invalid brainstorm artifact error")
	} else if got, want := err.Error(), "invalid brainstorm artifact: artifact identifier must be a canonical lowercase slug; title is required"; got != want {
		t.Fatalf("unexpected create validation error:\ngot:  %s\nwant: %s", got, want)
	}

	brainstormPath := filepath.Join(root, ".plan", "brainstorms", "alpha.md")
	brainstormRaw, err := os.ReadFile(brainstormPath)
	if err != nil {
		t.Fatal(err)
	}
	brainstormRaw = []byte(strings.Replace(string(brainstormRaw), "title: Alpha Brainstorm", `title: ""`, 1))
	if err := os.WriteFile(brainstormPath, brainstormRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.GetBrainstorm(context.Background(), "alpha"); err == nil {
		t.Fatal("expected invalid brainstorm error")
	} else if got, want := err.Error(), "invalid brainstorm .plan/brainstorms/alpha.md: title is required"; got != want {
		t.Fatalf("unexpected brainstorm validation error:\ngot:  %s\nwant: %s", got, want)
	}

	specPath := filepath.Join(root, ".plan", "specs", "alpha-spec.md")
	specRaw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	specRaw = []byte(strings.Replace(string(specRaw), "status: approved", "status: invalid", 1))
	if err := os.WriteFile(specPath, specRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.GetSpec(context.Background(), "alpha-spec"); err == nil {
		t.Fatal("expected invalid spec error")
	} else if got, want := err.Error(), "invalid spec .plan/specs/alpha-spec.md: unknown spec status"; got != want {
		t.Fatalf("unexpected spec validation error:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestAdapterCreatesPlanCompatibleBrainstormAtomicallyAndIdempotently(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	workspacePath := filepath.Join(root, ".plan", ".meta", "workspace.json")
	workspaceBefore, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	adapter := New(root)
	artifact := planning.Brainstorm{ID: "new-flow", Title: "New Flow"}
	createdAt := time.Date(2026, 7, 29, 7, 15, 0, 0, time.UTC)

	first, action, err := adapter.CreateBrainstorm(context.Background(), artifact, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if action != application.MutationCreate || first.Path != ".plan/brainstorms/new-flow.md" {
		t.Fatalf("unexpected create: action=%s document=%#v", action, first)
	}
	raw, err := os.ReadFile(filepath.Join(root, first.Path))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, required := range []string{
		"type: brainstorm",
		"slug: new-flow",
		"title: New Flow",
		"# Brainstorm: New Flow",
		"Started: 2026-07-29T07:15:00Z",
		"## Refinement",
		"## Challenge",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("created brainstorm missing %q:\n%s", required, content)
		}
	}
	second, action, err := adapter.CreateBrainstorm(context.Background(), artifact, createdAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if action != application.MutationUnchanged || second.Path != first.Path {
		t.Fatalf("unexpected idempotent create: action=%s document=%#v", action, second)
	}
	rawAfter, err := os.ReadFile(filepath.Join(root, first.Path))
	if err != nil {
		t.Fatal(err)
	}
	if string(rawAfter) != content {
		t.Fatal("idempotent rerun changed brainstorm content")
	}
	workspaceAfter, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(workspaceAfter) != string(workspaceBefore) {
		t.Fatal("brainstorm write changed workspace metadata")
	}
	assertNoTemporaryFiles(t, filepath.Join(root, ".plan", "brainstorms"))
}

func TestAdapterRollsBackOnlyTheExactBrainstormCreation(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	adapter := New(root)
	createdAt := time.Date(2026, 8, 9, 8, 0, 0, 0, time.UTC)
	artifact := planning.Brainstorm{ID: "rollback-flow", Title: "Rollback Flow"}
	if _, action, err := adapter.CreateBrainstorm(context.Background(), artifact, createdAt); err != nil || action != application.MutationCreate {
		t.Fatalf("unexpected create: action=%s err=%v", action, err)
	}
	if err := adapter.RollbackBrainstormCreation(context.Background(), artifact, createdAt.Add(time.Hour)); !errors.Is(err, application.ErrArtifactConflict) {
		t.Fatalf("expected guarded rollback conflict, got %v", err)
	}
	path := filepath.Join(root, ".plan", "brainstorms", "rollback-flow.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("guarded rollback removed the wrong creation: %v", err)
	}
	if err := adapter.RollbackBrainstormCreation(context.Background(), artifact, createdAt); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("exact rollback left brainstorm: %v", err)
	}
	if err := adapter.RollbackBrainstormCreation(context.Background(), artifact, createdAt); err != nil {
		t.Fatalf("rollback rerun was not idempotent: %v", err)
	}
}

func TestAdapterAtomicWriteFailureLeavesNoArtifact(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	writeFailure := errors.New("write failed")
	adapter := newWithWriter(root, func(string, []byte, os.FileMode) error {
		return writeFailure
	})
	_, _, err := adapter.CreateBrainstorm(
		context.Background(),
		planning.Brainstorm{ID: "failed", Title: "Failed"},
		time.Now(),
	)
	if !errors.Is(err, writeFailure) {
		t.Fatalf("expected write failure, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".plan", "brainstorms", "failed.md")); !os.IsNotExist(err) {
		t.Fatalf("failed write left artifact: %v", err)
	}
}

func TestAdapterRoadmapWriteFailurePreservesExistingFile(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	path := filepath.Join(root, ".plan", "ROADMAP.md")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeFailure := errors.New("write failed")
	adapter := newWithWriter(root, func(string, []byte, os.FileMode) error {
		return writeFailure
	})
	_, _, err = adapter.ReplaceRoadmap(context.Background(), "# Failed Roadmap\n")
	if !errors.Is(err, writeFailure) {
		t.Fatalf("expected write failure, got %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("failed roadmap write changed existing content")
	}
	assertNoTemporaryFiles(t, filepath.Join(root, ".plan"))
}

func TestAdapterConcurrentRerunsCreateOnce(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	adapter := New(root)
	artifact := planning.Brainstorm{ID: "concurrent", Title: "Concurrent"}
	const callers = 8
	actions := make(chan application.MutationAction, callers)
	errs := make(chan error, callers)
	var wait sync.WaitGroup
	for range callers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, action, err := adapter.CreateBrainstorm(context.Background(), artifact, time.Now())
			actions <- action
			errs <- err
		}()
	}
	wait.Wait()
	close(actions)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	created := 0
	unchanged := 0
	for action := range actions {
		switch action {
		case application.MutationCreate:
			created++
		case application.MutationUnchanged:
			unchanged++
		default:
			t.Fatalf("unexpected action %q", action)
		}
	}
	if created != 1 || unchanged != callers-1 {
		t.Fatalf("expected one create and %d unchanged, got create=%d unchanged=%d", callers-1, created, unchanged)
	}
	assertNoTemporaryFiles(t, filepath.Join(root, ".plan", "brainstorms"))
}

func TestCreationLockTimeoutIdentifiesLockPath(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "brainstorm.lock")
	if err := os.WriteFile(lockPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := acquireMutationLock(context.Background(), lockPath)
	if err == nil || !strings.Contains(err.Error(), lockPath) {
		t.Fatalf("expected timeout to identify lock path %q, got %v", lockPath, err)
	}
}

func TestAdapterReplacesBrainstormAndGuidedSessionsIdempotently(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	adapter := New(root)
	ctx := context.Background()
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

	brainstorm, err := adapter.GetBrainstorm(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	body := brainstorm.Body + "\n## Ideas\n\n- Preserve the local contract.\n"
	updated, action, err := adapter.ReplaceBrainstorm(ctx, "alpha", body, now)
	if err != nil {
		t.Fatal(err)
	}
	if action != application.MutationUpdate || updated.Body != body || updated.Metadata["created_at"] != brainstorm.Metadata["created_at"] {
		t.Fatalf("unexpected brainstorm update: action=%s document=%#v", action, updated)
	}
	_, action, err = adapter.ReplaceBrainstorm(ctx, "alpha", body, now.Add(time.Hour))
	if err != nil || action != application.MutationUnchanged {
		t.Fatalf("unexpected brainstorm rerun: action=%s err=%v", action, err)
	}

	state := application.GuidedSessionState{
		SchemaVersion:   3,
		LastActiveChain: "brainstorm/alpha",
		LastUpdatedAt:   now.Format(time.RFC3339),
		Sessions: map[string]application.GuidedSessionRecord{
			"brainstorm/alpha": {
				ChainID:       "brainstorm/alpha",
				Brainstorm:    "alpha",
				CurrentStage:  "brainstorm",
				StageStatuses: map[string]string{"brainstorm": "in_progress"},
			},
		},
	}
	_, action, err = adapter.ReplaceGuidedSessions(ctx, state)
	if err != nil || action != application.MutationCreate {
		t.Fatalf("unexpected guided-session create: action=%s err=%v", action, err)
	}
	got, err := adapter.ReadGuidedSessions(ctx)
	if err != nil || got.LastActiveChain != state.LastActiveChain {
		t.Fatalf("unexpected guided-session read: %#v err=%v", got, err)
	}
	_, action, err = adapter.ReplaceGuidedSessions(ctx, state)
	if err != nil || action != application.MutationUnchanged {
		t.Fatalf("unexpected guided-session rerun: action=%s err=%v", action, err)
	}
	assertNoTemporaryFiles(t, filepath.Join(root, ".plan"))
}

func TestAdapterWritesDirectPromotionSpecsWithoutLegacyHierarchy(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	adapter := New(root)
	ctx := context.Background()
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	write := application.PromotionSpecWrite{
		Artifact: planning.Spec{
			ID: "local-promotion", Title: "Local Promotion", Status: planning.SpecDraft,
			Approval:     planning.Approval{State: planning.ApprovalPending},
			Verification: []string{"Run tests.", "Verify CLI output."},
			Sources:      []planning.SourceReference{{URI: ".plan/brainstorms/alpha.md", Purpose: "promotion_source"}},
		},
		Body:     "# Local Promotion\n\n## Verification\n\n- Run tests.\n- Verify CLI output.\n",
		Metadata: map[string]any{"source_brainstorm": ".plan/brainstorms/alpha.md"},
	}

	documents, action, err := adapter.WritePromotionSpecs(ctx, []application.PromotionSpecWrite{write}, now)
	if err != nil {
		t.Fatal(err)
	}
	if action != application.MutationCreate || len(documents) != 1 || documents[0].Artifact.ID != "local-promotion" {
		t.Fatalf("unexpected promotion write: action=%s documents=%#v", action, documents)
	}
	for _, legacy := range []string{"epics", "stories"} {
		if _, err := os.Stat(filepath.Join(root, ".plan", legacy)); !os.IsNotExist(err) {
			t.Fatalf("direct promotion created legacy %s hierarchy: %v", legacy, err)
		}
	}
	_, action, err = adapter.WritePromotionSpecs(ctx, []application.PromotionSpecWrite{write}, now.Add(time.Hour))
	if err != nil || action != application.MutationUnchanged {
		t.Fatalf("unexpected promotion rerun: action=%s err=%v", action, err)
	}
	assertNoTemporaryFiles(t, filepath.Join(root, ".plan"))
}

func TestAdapterPromotionWriteFailureRollsBackCreatedSpecs(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	writeFailure := errors.New("second write failed")
	writesSeen := 0
	adapter := newWithWriter(root, func(path string, raw []byte, mode os.FileMode) error {
		writesSeen++
		if writesSeen == 2 {
			return writeFailure
		}
		return atomicWriteFile(path, raw, mode)
	})
	makeWrite := func(id, title string) application.PromotionSpecWrite {
		return application.PromotionSpecWrite{
			Artifact: planning.Spec{
				ID: planning.ArtifactID(id), Title: title, Status: planning.SpecDraft,
				Approval:     planning.Approval{State: planning.ApprovalPending},
				Verification: []string{"Run tests.", "Verify CLI output."},
			},
			Body:     "# " + title + "\n\n## Verification\n\n- Run tests.\n- Verify CLI output.\n",
			Metadata: map[string]any{"source_brainstorm": ".plan/brainstorms/alpha.md"},
		}
	}
	_, _, err := adapter.WritePromotionSpecs(context.Background(), []application.PromotionSpecWrite{
		makeWrite("first-promoted", "First Promoted"),
		makeWrite("second-promoted", "Second Promoted"),
	}, time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC))
	if !errors.Is(err, writeFailure) {
		t.Fatalf("expected write failure, got %v", err)
	}
	for _, id := range []string{"first-promoted", "second-promoted"} {
		if _, err := os.Stat(filepath.Join(root, ".plan", "specs", id+".md")); !os.IsNotExist(err) {
			t.Fatalf("failed promotion left %s: %v", id, err)
		}
	}
	assertNoTemporaryFiles(t, filepath.Join(root, ".plan"))
}

func TestAdapterRejectsWritesForUnsupportedWorkspace(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "future", root)
	_, _, err := New(root).CreateBrainstorm(
		context.Background(),
		planning.Brainstorm{ID: "blocked", Title: "Blocked"},
		time.Now(),
	)
	if !errors.Is(err, application.ErrWorkspaceNotWritable) {
		t.Fatalf("expected writable gate, got %v", err)
	}
}

func copyFixture(t *testing.T, name, root string) {
	t.Helper()
	source := filepath.Join("testdata", name)
	if err := os.CopyFS(root, os.DirFS(source)); err != nil {
		t.Fatal(err)
	}
}

func snapshotFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel)+"\x00"+string(raw))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func assertNoTemporaryFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Fatalf("temporary file left behind: %s", entry.Name())
		}
	}
}
