package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"brain/internal/projectcontext"
)

func TestPathMatchesAny(t *testing.T) {
	if !pathMatchesAny(".brain/resources/changes/project-change.md", []string{".brain/resources/**/project*.md"}) {
		t.Fatal("expected glob to match capture path")
	}
	if pathMatchesAny(".brain/resources/elsewhere/note.md", []string{"docs/project/**"}) {
		t.Fatal("did not expect glob to match unrelated path")
	}
}

func TestCommandProfileSatisfied(t *testing.T) {
	profile := projectcontext.VerificationProfile{
		Name:     "tests",
		Commands: []string{"go test ./..."},
	}
	runs := []CommandRun{
		{Command: "go test ./...", ExitCode: 0},
	}
	if !commandProfileSatisfied(profile, runs) {
		t.Fatal("expected profile to be satisfied")
	}
	runs[0].ExitCode = 1
	if commandProfileSatisfied(profile, runs) {
		t.Fatal("expected failed command run not to satisfy profile")
	}
}

func TestRunCommandConcurrentRecordsAll(t *testing.T) {
	project := makeSessionProject(t, sessionPolicyYAML(t, nil, false))
	manager := New(nil)
	if _, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "concurrency"}); err != nil {
		t.Fatalf("start session: %v", err)
	}

	const runs = 6
	var wg sync.WaitGroup
	errCh := make(chan error, runs)
	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			argv := helperCommand("sleep-ms", "75", fmt.Sprintf("run-%d", i))
			result, err := manager.RunCommand(context.Background(), RunRequest{
				ProjectDir:    project,
				Argv:          argv,
				CaptureOutput: true,
			}, nil, nil)
			if err != nil {
				errCh <- fmt.Errorf("run %d failed: %w", i, err)
				return
			}
			if !result.Recorded {
				errCh <- fmt.Errorf("run %d was not recorded", i)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}

	active, err := loadActiveSession(filepath.Join(project, ".brain", "session.json"))
	if err != nil {
		t.Fatalf("load active session: %v", err)
	}
	if got := len(active.CommandRuns); got != runs {
		t.Fatalf("expected %d command runs, got %d", runs, got)
	}
}

func TestValidateFinishSeesConcurrentRecordedCommands(t *testing.T) {
	testsCmd := helperCommand("sleep-ms", "75", "tests")
	buildCmd := helperCommand("sleep-ms", "75", "build")
	project := makeSessionProject(t, sessionPolicyYAML(t, []string{
		strings.Join(testsCmd, " "),
		strings.Join(buildCmd, " "),
	}, false))
	mustInitGitRepo(t, project)

	manager := New(nil)
	if _, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "verify"}); err != nil {
		t.Fatalf("start session: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "README.md"), []byte("# changed\n"), 0o644); err != nil {
		t.Fatalf("mark repo dirty: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for _, argv := range [][]string{testsCmd, buildCmd} {
		wg.Add(1)
		go func(argv []string) {
			defer wg.Done()
			result, err := manager.RunCommand(context.Background(), RunRequest{
				ProjectDir:    project,
				Argv:          argv,
				CaptureOutput: true,
			}, nil, nil)
			if err != nil {
				errCh <- err
				return
			}
			if !result.Recorded {
				errCh <- fmt.Errorf("command %q was not recorded", result.Command)
			}
		}(argv)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}

	result, err := manager.Validate(context.Background(), ValidateRequest{ProjectDir: project, Stage: "finish"})
	if err != nil {
		t.Fatalf("validate finish: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected finish validation ok, got obligations=%v remediation=%v missing=%v", result.Obligations, result.Remediation, result.MissingCommands)
	}
}

func TestRunCommandAfterAbortDoesNotRecreateActiveSession(t *testing.T) {
	project := makeSessionProject(t, sessionPolicyYAML(t, nil, false))
	manager := New(nil)
	if _, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "abort race"}); err != nil {
		t.Fatalf("start session: %v", err)
	}

	resultCh := make(chan *RunResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := manager.RunCommand(context.Background(), RunRequest{
			ProjectDir:    project,
			Argv:          helperCommand("sleep-ms", "200", "late"),
			CaptureOutput: true,
		}, nil, nil)
		resultCh <- result
		errCh <- err
	}()

	time.Sleep(50 * time.Millisecond)
	if _, err := manager.Abort(context.Background(), AbortRequest{ProjectDir: project, Reason: "test abort"}); err != nil {
		t.Fatalf("abort session: %v", err)
	}

	result := <-resultCh
	err := <-errCh
	if err == nil {
		t.Fatal("expected run command error after abort")
	}
	if result == nil {
		t.Fatal("expected run result even when recording fails")
	}
	if result.Recorded {
		t.Fatal("expected aborted command run to be unrecorded")
	}
	if _, statErr := os.Stat(filepath.Join(project, ".brain", "session.json")); !os.IsNotExist(statErr) {
		t.Fatalf("expected active session to stay removed, stat err=%v", statErr)
	}
}

func TestSessionCommandHelper(t *testing.T) {
	idx := -1
	for i, arg := range os.Args {
		if arg == "--" {
			idx = i
			break
		}
	}
	if idx < 0 || idx+1 >= len(os.Args) {
		return
	}
	switch os.Args[idx+1] {
	case "sleep-ms":
		if idx+2 >= len(os.Args) {
			fmt.Fprintln(os.Stderr, "missing duration")
			os.Exit(2)
		}
		duration, err := time.ParseDuration(os.Args[idx+2] + "ms")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		time.Sleep(duration)
		os.Exit(0)
	default:
		return
	}
}

func makeSessionProject(t *testing.T, policy string) string {
	t.Helper()
	project := t.TempDir()
	for _, rel := range []string{
		".brain/state",
		".brain/context",
		".brain/resources",
		".brain/sessions",
		"docs",
	} {
		if err := os.MkdirAll(filepath.Join(project, rel), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
	}
	files := map[string]string{
		"AGENTS.md":                       "# contract\n",
		".brain/context/overview.md":      "# overview\n",
		".brain/context/workflows.md":     "# workflows\n",
		".brain/context/memory-policy.md": "# memory policy\n",
		"README.md":                       "# project\n",
		".brain/policy.yaml":              policy,
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(project, rel), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return project
}

func sessionPolicyYAML(t *testing.T, verificationCommands []string, requireMemoryUpdate bool) string {
	t.Helper()
	commands := []string{
		"go test ./...",
		"go build ./...",
	}
	if len(verificationCommands) != 0 {
		commands = verificationCommands
	}
	lines := make([]string, 0, len(commands)*2)
	names := []string{"tests", "build", "verify-3", "verify-4", "verify-5", "verify-6"}
	for i, command := range commands {
		name := names[i]
		lines = append(lines, fmt.Sprintf("    - name: %s", name))
		lines = append(lines, fmt.Sprintf("      commands:\n        - %q", command))
	}
	return strings.TrimSpace(fmt.Sprintf(`
version: 1
project:
  name: brain
  slug: brain
  runtime: go
  memory:
    accepted_note_globs:
      - AGENTS.md
      - docs/**
      - .brain/context/**
      - .brain/planning/**
      - .brain/brainstorms/**
      - .brain/resources/**
session:
  require_task: true
  single_active: true
  active_file: .brain/session.json
  ledger_dir: .brain/sessions
preflight:
  require_brain_doctor: true
  required_docs:
    - AGENTS.md
    - .brain/context/overview.md
    - .brain/context/workflows.md
    - .brain/context/memory-policy.md
  suggested_commands:
    - brain find brain
closeout:
  acceptable_history_operations:
    - update
  require_memory_update_on_repo_change: %t
  verification_profiles:
%s
`, requireMemoryUpdate, strings.Join(lines, "\n")))
}

func helperCommand(args ...string) []string {
	base := []string{"env", "GO_WANT_SESSION_HELPER=1", os.Args[0], "-test.run=TestSessionCommandHelper", "--"}
	return append(base, args...)
}

func mustInitGitRepo(t *testing.T, dir string) {
	t.Helper()
	commands := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "brain-tests@example.com"},
		{"git", "config", "user.name", "Brain Tests"},
		{"git", "add", "."},
		{"git", "commit", "-m", "initial"},
	}
	for _, argv := range commands {
		cmd := exec.Command(argv[0], argv[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %v\n%s", strings.Join(argv, " "), err, out)
		}
	}
}
