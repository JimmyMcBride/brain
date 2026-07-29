package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"brain/internal/planning"
	"brain/internal/planning/application"
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
	specRaw = []byte(strings.Replace(
		string(specRaw),
		"- Run focused adapter tests.\n- Preserve stable ordering.\n",
		"",
		1,
	))
	if err := os.WriteFile(specPath, specRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.GetSpec(context.Background(), "alpha-spec"); err == nil {
		t.Fatal("expected invalid spec error")
	} else if got, want := err.Error(), "invalid spec .plan/specs/alpha-spec.md: verification is required"; got != want {
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

func TestAdapterAtomicWriteFailureLeavesNoArtifact(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "compatible", root)
	writeFailure := errors.New("write failed")
	adapter := NewWithWriter(root, func(string, []byte, os.FileMode) error {
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

	_, err := acquireCreationLock(context.Background(), lockPath)
	if err == nil || !strings.Contains(err.Error(), lockPath) {
		t.Fatalf("expected timeout to identify lock path %q, got %v", lockPath, err)
	}
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
