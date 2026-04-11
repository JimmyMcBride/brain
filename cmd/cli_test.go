package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var rfc3339Pattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)

type cliResult struct {
	stdout string
	stderr string
	err    error
}

type cliEnv struct {
	root       string
	moduleRoot string
	home       string
	config     string
	vault      string
	data       string
	project    string
	custom     string
}

func newCLIEnv(t *testing.T) *cliEnv {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Dir(filepath.Dir(file))
	root := t.TempDir()
	home := filepath.Join(root, "home")
	config := filepath.Join(root, "config.yaml")
	vault := filepath.Join(root, "vault")
	data := filepath.Join(root, "data")
	project := filepath.Join(root, "project")
	custom := filepath.Join(root, "custom-skills")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	return &cliEnv{
		root:       root,
		moduleRoot: moduleRoot,
		home:       home,
		config:     config,
		vault:      vault,
		data:       data,
		project:    project,
		custom:     custom,
	}
}

func (e *cliEnv) run(t *testing.T, stdin string, args ...string) cliResult {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(e.moduleRoot); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	cmd := newRootCommand(rootOptions{
		in:     strings.NewReader(stdin),
		out:    &stdout,
		errOut: &stderr,
	})
	cmd.SetArgs(args)
	err = cmd.Execute()
	return cliResult{
		stdout: normalizeCLIOutput(stdout.String(), e.root),
		stderr: normalizeCLIOutput(stderr.String(), e.root),
		err:    err,
	}
}

func normalizeCLIOutput(s, root string) string {
	s = strings.ReplaceAll(s, root, "<ROOT>")
	s = rfc3339Pattern.ReplaceAllString(s, "<TIME>")
	return filepath.ToSlash(s)
}

func requireOK(t *testing.T, result cliResult) string {
	t.Helper()
	if result.err != nil {
		t.Fatalf("unexpected error: %v\nstdout:\n%s\nstderr:\n%s", result.err, result.stdout, result.stderr)
	}
	return result.stdout
}

func TestCLIVaultLifecycle(t *testing.T) {
	env := newCLIEnv(t)

	requireOK(t, env.run(t, "", "--config", env.config, "init", "--vault", env.vault, "--data", env.data))

	if _, err := os.Stat(filepath.Join(env.vault, "Projects")); err != nil {
		t.Fatalf("missing Projects dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.data, "brain.sqlite3")); err != nil {
		t.Fatalf("missing sqlite db: %v", err)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "add", "Project Atlas", "-s", "Projects", "-t", "project"))
	requireOK(t, env.run(t, "", "--config", env.config, "add", "Signal Notes", "-s", "Resources", "-t", "resource", "-b", "# Signal Notes\n\nLexical retrieval works well locally."))

	findOutput := requireOK(t, env.run(t, "", "--config", env.config, "find", "signal"))
	if !strings.Contains(findOutput, "Resources/signal-notes.md [resource] Signal Notes") {
		t.Fatalf("find output missing note:\n%s", findOutput)
	}

	searchBefore := requireOK(t, env.run(t, "", "--config", env.config, "search", "signal"))
	if !strings.Contains(searchBefore, "No indexed content. Run `brain reindex` first.") {
		t.Fatalf("unexpected pre-index search output:\n%s", searchBefore)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "capture", "Quick Thought", "-b", "Semantic retrieval helps recall nearby concepts."))
	requireOK(t, env.run(t, "", "--config", env.config, "daily", "2026-04-09"))
	requireOK(t, env.run(t, "", "--config", env.config, "reindex"))

	searchAfter := requireOK(t, env.run(t, "", "--config", env.config, "search", "lexical retrieval"))
	if !strings.Contains(searchAfter, "Resources/signal-notes.md") {
		t.Fatalf("expected indexed search result:\n%s", searchAfter)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "edit", "Resources/signal-notes.md", "-m", "status=active", "-b", "# Signal Notes\n\nUpdated body."))
	requireOK(t, env.run(t, "", "--config", env.config, "move", "Resources/signal-notes.md", "Areas/Reference/"))

	readMoved := requireOK(t, env.run(t, "", "--config", env.config, "read", "Areas/Reference/signal-notes.md"))
	if !strings.Contains(readMoved, "Updated body.") {
		t.Fatalf("unexpected read output:\n%s", readMoved)
	}

	historyOutput := requireOK(t, env.run(t, "", "--config", env.config, "history", "-n", "3"))
	lines := strings.Split(strings.TrimSpace(historyOutput), "\n")
	if len(lines) == 0 || !strings.Contains(lines[0], "move") {
		t.Fatalf("expected newest history entry to be move:\n%s", historyOutput)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "undo"))
	if _, err := os.Stat(filepath.Join(env.vault, "Resources", "signal-notes.md")); err != nil {
		t.Fatalf("expected note restored after undo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.vault, "Areas", "Reference", "signal-notes.md")); !os.IsNotExist(err) {
		t.Fatalf("expected moved file removed after undo, got err=%v", err)
	}
}

func TestCLIContentWorkflow(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "init", "--vault", env.vault, "--data", env.data))
	requireOK(t, env.run(t, "", "--config", env.config, "add", "Content Seed", "-s", "Projects", "-t", "project", "-b", "# Content Seed\n\nAgents need explicit tool contracts for content workflows."))
	requireOK(t, env.run(t, "", "--config", env.config, "add", "Agent Notes", "-s", "Resources", "-t", "resource", "-b", "# Agent Notes\n\nContent seed workflows need explicit tool contracts for agents."))
	requireOK(t, env.run(t, "", "--config", env.config, "reindex"))

	requireOK(t, env.run(t, "", "--config", env.config, "content", "seed", "Projects/content-seed.md"))
	gatherOutput := requireOK(t, env.run(t, "", "--config", env.config, "content", "gather", "Projects/content-seed.md", "-n", "3"))
	if !strings.Contains(gatherOutput, "Resources/agent-notes.md") {
		t.Fatalf("expected related note in gather output:\n%s", gatherOutput)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "content", "outline", "Projects/content-seed.md", "-n", "3"))
	if _, err := os.Stat(filepath.Join(env.vault, "Resources", "Content", "Outlines", "content-seed-outline.md")); err != nil {
		t.Fatalf("expected outline note: %v", err)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "content", "publish", "Projects/content-seed.md", "--channel", "blog"))
	published, err := os.ReadFile(filepath.Join(env.vault, "Projects", "content-seed.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(published), "published: true") {
		t.Fatalf("expected publish metadata in note:\n%s", string(published))
	}
}

func TestCLISkillsCommands(t *testing.T) {
	env := newCLIEnv(t)
	targets := requireOK(t, env.run(t, "", "skills", "targets", "--scope", "both", "-a", "codex", "-a", "zed", "--project", env.project, "--skill-root", env.custom))
	if !strings.Contains(targets, "codex [global] <ROOT>/home/.codex/skills/brain") {
		t.Fatalf("missing global codex target:\n%s", targets)
	}
	if !strings.Contains(targets, "zed [local] <ROOT>/project/.zed/skills/brain") {
		t.Fatalf("missing local zed target:\n%s", targets)
	}
	if !strings.Contains(targets, "custom [custom] <ROOT>/custom-skills/brain") {
		t.Fatalf("missing custom target:\n%s", targets)
	}

	requireOK(t, env.run(t, "", "skills", "install", "--scope", "local", "-a", "codex", "--project", env.project, "--mode", "copy"))
	if _, err := os.Stat(filepath.Join(env.project, ".codex", "skills", "brain", "SKILL.md")); err != nil {
		t.Fatalf("expected local skill install: %v", err)
	}

	requireOK(t, env.run(t, "", "skills", "install", "--scope", "global", "-a", "codex", "--mode", "copy"))
	if _, err := os.Stat(filepath.Join(env.home, ".codex", "skills", "brain", "SKILL.md")); err != nil {
		t.Fatalf("expected global skill install: %v", err)
	}
}

func TestCLIContextCommands(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "context", "install", "--project", env.project, "--agent", "codex"))

	for _, path := range []string{
		filepath.Join(env.project, "AGENTS.md"),
		filepath.Join(env.project, ".brain", "context", "overview.md"),
		filepath.Join(env.project, ".brain", "policy.yaml"),
		filepath.Join(env.project, ".codex", "AGENTS.md"),
		filepath.Join(env.project, ".gitignore"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected context file %s: %v", path, err)
		}
	}

	overviewPath := filepath.Join(env.project, ".brain", "context", "overview.md")
	overviewData, err := os.ReadFile(overviewPath)
	if err != nil {
		t.Fatal(err)
	}
	overviewData = append(overviewData, []byte("\nProject note: keep this.\n")...)
	if err := os.WriteFile(overviewPath, overviewData, 0o644); err != nil {
		t.Fatal(err)
	}

	refreshOutput := requireOK(t, env.run(t, "", "context", "refresh", "--project", env.project, "--agent", "codex"))
	if !strings.Contains(refreshOutput, "updated   context  .brain/context/overview.md preserve-user") {
		t.Fatalf("unexpected refresh output:\n%s", refreshOutput)
	}

	refreshed, err := os.ReadFile(overviewPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(refreshed), "Project note: keep this.") {
		t.Fatalf("expected preserved local note:\n%s", string(refreshed))
	}

	dryRun := requireOK(t, env.run(t, "", "context", "refresh", "--project", env.project, "--agent", "codex", "--dry-run"))
	if !strings.Contains(dryRun, "unchanged") {
		t.Fatalf("expected unchanged dry-run output:\n%s", dryRun)
	}
}

func TestCLISessionWorkflow(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "init", "--vault", env.vault, "--data", env.data))
	initGitProject(t, env.project)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))

	startOutput := requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "tighten session enforcement"))
	if !strings.Contains(startOutput, "Started session") || !strings.Contains(startOutput, "brain search \"project tighten session enforcement\"") {
		t.Fatalf("unexpected session start output:\n%s", startOutput)
	}

	secondStart := env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "duplicate")
	if secondStart.err == nil {
		t.Fatalf("expected second start to fail:\nstdout=%s\nstderr=%s", secondStart.stdout, secondStart.stderr)
	}

	if err := os.WriteFile(filepath.Join(env.project, "main.go"), []byte("package main\nfunc main() { println(\"x\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	validateBlocked := env.run(t, "", "--config", env.config, "session", "validate", "--project", env.project, "--stage", "finish")
	if validateBlocked.err == nil || !strings.Contains(validateBlocked.stdout, "durable note update required for repo changes") {
		t.Fatalf("expected finish validation to block before note update:\nstdout=%s\nstderr=%s", validateBlocked.stdout, validateBlocked.stderr)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "capture", "Project: session enforcement note", "--body", "Recorded durable note for project changes."))
	validateNeedsReindex := env.run(t, "", "--config", env.config, "session", "validate", "--project", env.project, "--stage", "finish")
	if validateNeedsReindex.err == nil || !strings.Contains(validateNeedsReindex.stdout, "reindex required after note updates") {
		t.Fatalf("expected reindex obligation:\nstdout=%s\nstderr=%s", validateNeedsReindex.stdout, validateNeedsReindex.stderr)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "reindex"))
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "test", "./..."))
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "build", "./..."))

	finishOutput := requireOK(t, env.run(t, "", "--config", env.config, "session", "finish", "--project", env.project, "--summary", "session complete"))
	if !strings.Contains(finishOutput, "finished") || !strings.Contains(finishOutput, ".brain/sessions/") {
		t.Fatalf("unexpected finish output:\n%s", finishOutput)
	}

	validateNoSession := env.run(t, "", "--config", env.config, "session", "validate", "--project", env.project)
	if validateNoSession.err == nil {
		t.Fatalf("expected validate to fail after finish")
	}
}

func TestCLISessionAbort(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "init", "--vault", env.vault, "--data", env.data))
	initGitProject(t, env.project)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))
	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "abort flow"))

	abortOutput := requireOK(t, env.run(t, "", "--config", env.config, "session", "abort", "--project", env.project, "--reason", "stopped work"))
	if !strings.Contains(abortOutput, "aborted") {
		t.Fatalf("unexpected abort output:\n%s", abortOutput)
	}

	if _, err := os.Stat(filepath.Join(env.project, ".brain", "session.json")); !os.IsNotExist(err) {
		t.Fatalf("expected active session cleared, got err=%v", err)
	}
}

func initGitProject(t *testing.T, project string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/demo\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCommand(t, project, "init", "-q")
	runGitCommand(t, project, "config", "user.email", "tester@example.com")
	runGitCommand(t, project, "config", "user.name", "tester")
	runGitCommand(t, project, "add", ".")
	runGitCommand(t, project, "commit", "-q", "-m", "init")
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}
