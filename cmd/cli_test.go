package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"brain/internal/buildinfo"
	"brain/internal/config"
	"brain/internal/update"
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
	project := filepath.Join(root, "project")
	custom := filepath.Join(root, "custom-skills")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", filepath.Join(root, "LocalAppData"))
	t.Setenv("APPDATA", filepath.Join(root, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	return &cliEnv{
		root:       root,
		moduleRoot: moduleRoot,
		home:       home,
		config:     config,
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

func TestCLIProjectLifecycle(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	for _, path := range []string{
		filepath.Join(env.project, "AGENTS.md"),
		filepath.Join(env.project, "docs", "project-overview.md"),
		filepath.Join(env.project, ".brain", "context", "overview.md"),
		filepath.Join(env.project, ".brain", "state", "brain.sqlite3"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}

	findOutput := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "find", "project-overview"))
	if !strings.Contains(findOutput, "docs/project-overview.md") {
		t.Fatalf("unexpected find output:\n%s", findOutput)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "edit", "docs/project-overview.md", "-b", "# Project Overview\n\nUpdated body."))
	readOutput := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "read", "docs/project-overview.md"))
	if !strings.Contains(readOutput, "Updated body.") {
		t.Fatalf("unexpected read output:\n%s", readOutput)
	}

	searchOutput := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "search", "updated body"))
	if !strings.Contains(searchOutput, "docs/project-overview.md") {
		t.Fatalf("unexpected search output:\n%s", searchOutput)
	}

	historyOutput := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "history", "-n", "3"))
	if !strings.Contains(historyOutput, "update") {
		t.Fatalf("unexpected history output:\n%s", historyOutput)
	}
}

func TestCLIAdoptExistingRepoPreservesManagedFiles(t *testing.T) {
	env := newCLIEnv(t)
	if err := os.WriteFile(filepath.Join(env.project, "README.md"), []byte("# Existing Readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.project, "AGENTS.md"), []byte("Manual contract\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(env.project, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.project, "docs", "project-overview.md"), []byte("Manual overview\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	adoptOutput := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "adopt"))
	if !strings.Contains(adoptOutput, "adopted") || !strings.Contains(adoptOutput, "AGENTS.md") {
		t.Fatalf("unexpected adopt output:\n%s", adoptOutput)
	}

	agentsRaw, err := os.ReadFile(filepath.Join(env.project, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	agents := string(agentsRaw)
	if !strings.Contains(agents, "<!-- brain:begin agents-contract -->") || !strings.Contains(agents, "Manual contract") {
		t.Fatalf("unexpected adopted AGENTS.md:\n%s", agents)
	}

	overviewRaw, err := os.ReadFile(filepath.Join(env.project, "docs", "project-overview.md"))
	if err != nil {
		t.Fatal(err)
	}
	overview := string(overviewRaw)
	if !strings.Contains(overview, "Manual overview") || !strings.Contains(overview, "## Local Notes") {
		t.Fatalf("unexpected adopted project overview:\n%s", overview)
	}

	readmeRaw, err := os.ReadFile(filepath.Join(env.project, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readmeRaw) != "# Existing Readme\n" {
		t.Fatalf("expected README to remain unchanged, got:\n%s", string(readmeRaw))
	}
}

func TestCLIAdoptDryRunDoesNotWrite(t *testing.T) {
	env := newCLIEnv(t)
	if err := os.WriteFile(filepath.Join(env.project, "AGENTS.md"), []byte("Manual contract\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "adopt", "--dry-run"))
	if !strings.Contains(output, "Adoption plan:") || !strings.Contains(output, "adopted") {
		t.Fatalf("unexpected dry-run output:\n%s", output)
	}

	agentsRaw, err := os.ReadFile(filepath.Join(env.project, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(agentsRaw) != "Manual contract\n" {
		t.Fatalf("expected dry-run to preserve unmanaged AGENTS.md, got:\n%s", string(agentsRaw))
	}
}

func TestCLIAdoptIsIdempotentOnManagedRepo(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	output := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "adopt"))
	if strings.Contains(output, "adopted") {
		t.Fatalf("expected already-managed repo not to be adopted again:\n%s", output)
	}
	if !strings.Contains(output, "unchanged") && !strings.Contains(output, "updated") {
		t.Fatalf("unexpected idempotent adopt output:\n%s", output)
	}
}

func TestCLISearchStatusAndExplain(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	missing := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "search", "status"))
	if !strings.Contains(missing, "state: missing") {
		t.Fatalf("expected missing index status before search:\n%s", missing)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "edit", "docs/project-overview.md", "-b", "# Project Overview\n\nRetrieval status should become observable."))
	searchOutput := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "search", "--explain", "retrieval observable"))
	if !strings.Contains(searchOutput, "[") || !strings.Contains(searchOutput, "lex=") || !strings.Contains(searchOutput, "sem=") {
		t.Fatalf("expected explain output:\n%s", searchOutput)
	}

	fresh := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "search", "status"))
	if !strings.Contains(fresh, "state: fresh") {
		t.Fatalf("expected fresh index status after search:\n%s", fresh)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "edit", "docs/project-overview.md", "-b", "# Project Overview\n\nRetrieval status should become stale after a managed edit."))
	stale := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "search", "status"))
	if !strings.Contains(stale, "state: stale") || !strings.Contains(stale, "workspace signature changed") {
		t.Fatalf("expected stale index status after managed markdown change:\n%s", stale)
	}
}

func TestCLIProjectPlanningWorkflow(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	initOutput := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "plan", "init"))
	if !strings.Contains(initOutput, "Initialized epic-only planning") {
		t.Fatalf("unexpected plan init output:\n%s", initOutput)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "brainstorm", "start", "Auth Ideas"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "plan", "epic", "promote", "auth-ideas"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "plan", "spec", "status", "auth-ideas", "--set", "approved"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "plan", "story", "create", "auth-ideas", "Login Flow", "-b", "Support email and password login.", "--criteria", "Validate email format", "--resource", "[[.brain/brainstorms/auth-ideas.md]]"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "plan", "story", "update", "login-flow", "--status", "done", "--criteria", "Hash passwords"))

	storyPath := filepath.Join(env.project, ".brain", "planning", "stories", "login-flow.md")
	storyRaw, err := os.ReadFile(storyPath)
	if err != nil {
		t.Fatal(err)
	}
	story := string(storyRaw)
	if !strings.Contains(story, "Support email and password login.") || !strings.Contains(story, "- [ ] Hash passwords") {
		t.Fatalf("unexpected story contents:\n%s", story)
	}
	if !strings.Contains(story, "spec: auth-ideas") {
		t.Fatalf("expected story to reference canonical spec:\n%s", story)
	}

	statusOutput := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "plan", "status"))
	if !strings.Contains(statusOutput, "epic_spec_v1") || !strings.Contains(statusOutput, "Stories: 1 total, 1 done, 0 in progress, 0 blocked, 0 remaining") || !strings.Contains(statusOutput, "Epic Auth Ideas [approved]: 1/1 stories done") {
		t.Fatalf("unexpected status output:\n%s", statusOutput)
	}
}

func TestCLISkillsCommands(t *testing.T) {
	env := newCLIEnv(t)
	targets := requireOK(t, env.run(t, "", "skills", "targets", "--scope", "both", "-a", "codex", "-a", "copilot", "-a", "pi.dev", "-a", "zed", "--project", env.project))
	if !strings.Contains(targets, "codex [global] brain <ROOT>/home/.codex/skills/brain") {
		t.Fatalf("missing global codex target:\n%s", targets)
	}
	if !strings.Contains(targets, "copilot [global] brain <ROOT>/home/.copilot/skills/brain") {
		t.Fatalf("missing global copilot target:\n%s", targets)
	}
	if !strings.Contains(targets, "copilot [local] brain <ROOT>/project/.github/skills/brain") {
		t.Fatalf("missing local copilot target:\n%s", targets)
	}
	if !strings.Contains(targets, "pi [global] brain <ROOT>/home/.pi/agent/skills/brain") {
		t.Fatalf("missing global pi target:\n%s", targets)
	}
	if !strings.Contains(targets, "pi [local] brain <ROOT>/project/.pi/skills/brain") {
		t.Fatalf("missing local pi target:\n%s", targets)
	}
	if !strings.Contains(targets, "zed [local] brain <ROOT>/project/.zed/skills/brain") {
		t.Fatalf("missing local zed target:\n%s", targets)
	}
	requireOK(t, env.run(t, "", "skills", "install", "--scope", "local", "-a", "codex", "--project", env.project, "--mode", "copy"))
	if _, err := os.Stat(filepath.Join(env.project, ".codex", "skills", "brain", "SKILL.md")); err != nil {
		t.Fatalf("expected local skill install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".codex", "skills", "googleworkspace-cli")); !os.IsNotExist(err) {
		t.Fatalf("expected no non-brain skill installs, got err=%v", err)
	}

	help := requireOK(t, env.run(t, "", "skills", "install", "--help"))
	if strings.Contains(help, "--skill") || strings.Contains(help, "--skill-root") {
		t.Fatalf("expected help to omit removed flags:\n%s", help)
	}
}

func TestCLIEditNormalizesFullNoteInput(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	payload := "---\ntitle: Manual\ntype: resource\n---\n# Body\n"
	requireOK(t, env.run(t, payload, "--config", env.config, "--project", env.project, "edit", ".brain/context/current-state.md", "--stdin"))

	raw, err := os.ReadFile(filepath.Join(env.project, ".brain", "context", "current-state.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if strings.Count(content, "---\n") != 2 {
		t.Fatalf("expected exactly one frontmatter block:\n%s", content)
	}
	if strings.Contains(content, "\n---\n---\n") {
		t.Fatalf("unexpected nested frontmatter:\n%s", content)
	}
	if !strings.Contains(content, "# Body") {
		t.Fatalf("missing updated body:\n%s", content)
	}
}

func TestCLIDoctorDetectsBrokenNotes(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	broken := "---\nupdated: now\n---\n---\ntitle: Broken\n---\n# Body\n"
	if err := os.WriteFile(filepath.Join(env.project, ".brain", "context", "current-state.md"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	doctor := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "doctor"))
	if !strings.Contains(doctor, "note_integrity: fail") {
		t.Fatalf("expected doctor to report note integrity failure:\n%s", doctor)
	}
}

func TestCLIDoctorReportsIndexFreshness(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	before := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "doctor"))
	if !strings.Contains(before, "index_freshness: fail (missing") {
		t.Fatalf("expected missing freshness in doctor output:\n%s", before)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "search", "project overview"))
	after := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "doctor"))
	if !strings.Contains(after, "index_freshness: ok (fresh") {
		t.Fatalf("expected fresh index status in doctor output:\n%s", after)
	}
}

func TestCLIContextCommands(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project, "--agent", "codex"))

	for _, path := range []string{
		filepath.Join(env.project, "AGENTS.md"),
		filepath.Join(env.project, ".brain", "context", "overview.md"),
		filepath.Join(env.project, "docs", "project-overview.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected context file %s: %v", path, err)
		}
	}
}

func TestCLISessionWorkflow(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	initGitProject(t, env.project)

	startOutput := requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "tighten session enforcement"))
	if !strings.Contains(startOutput, "Started session") {
		t.Fatalf("unexpected session start output:\n%s", startOutput)
	}

	if err := os.WriteFile(filepath.Join(env.project, "main.go"), []byte("package main\nfunc main() { println(\"x\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	validateBlocked := env.run(t, "", "--config", env.config, "session", "validate", "--project", env.project, "--stage", "finish")
	if validateBlocked.err == nil || !strings.Contains(validateBlocked.stdout, "durable note update required for repo changes") {
		t.Fatalf("expected finish validation to block before note update:\nstdout=%s\nstderr=%s", validateBlocked.stdout, validateBlocked.stderr)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "edit", "AGENTS.md", "-b", "# Project Agent Contract\n\nRecorded durable note for project changes.\n"))
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "test", "./..."))
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "build", "./..."))

	finishOutput := requireOK(t, env.run(t, "", "--config", env.config, "session", "finish", "--project", env.project, "--summary", "session complete"))
	if !strings.Contains(finishOutput, "finished") || !strings.Contains(finishOutput, ".brain/sessions/") {
		t.Fatalf("unexpected finish output:\n%s", finishOutput)
	}
}

func TestCLISessionPublishWorkflowUsesCommittedDurableNotes(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	initGitProject(t, env.project)

	if err := os.WriteFile(filepath.Join(env.project, "main.go"), []byte("package main\nfunc main() { println(\"release\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.project, "AGENTS.md"), []byte("# Project Agent Contract\n\nCommitted durable note before publish.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	startOutput := requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "publish main release"))
	if !strings.Contains(startOutput, "Started session") {
		t.Fatalf("unexpected session start output:\n%s", startOutput)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "test", "./..."))
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "build", "./..."))
	runGitCommand(t, env.project, "add", ".")
	runGitCommand(t, env.project, "commit", "-q", "-m", "publish changes")

	validateOutput := requireOK(t, env.run(t, "", "--config", env.config, "session", "validate", "--project", env.project, "--stage", "finish"))
	if !strings.Contains(validateOutput, "Memory: git_committed_notes") {
		t.Fatalf("expected git-backed memory satisfaction in validate output:\n%s", validateOutput)
	}

	finishOutput := requireOK(t, env.run(t, "", "--config", env.config, "session", "finish", "--project", env.project, "--summary", "publish complete"))
	if !strings.Contains(finishOutput, "finished") || !strings.Contains(finishOutput, "Memory: git_committed_notes") {
		t.Fatalf("unexpected finish output:\n%s", finishOutput)
	}
}

func TestCLIVersionCommand(t *testing.T) {
	env := newCLIEnv(t)
	restore := setCLICommandBuildInfo("v1.2.3", "abc123", "2026-04-10T00:00:00Z")
	defer restore()

	human := requireOK(t, env.run(t, "", "--config", env.config, "version"))
	if !strings.Contains(human, "version: v1.2.3") || !strings.Contains(human, "commit:  abc123") {
		t.Fatalf("unexpected version output:\n%s", human)
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--json", "version"))
	if !strings.Contains(jsonOut, "\"version\": \"v1.2.3\"") {
		t.Fatalf("unexpected version json:\n%s", jsonOut)
	}
}

func TestCLIUpdateCommand(t *testing.T) {
	env := newCLIEnv(t)
	restoreUpdater := newUpdater
	restoreBuild := setCLICommandBuildInfo("v0.1.0", "abc123", "2026-04-10T00:00:00Z")
	defer func() {
		newUpdater = restoreUpdater
		restoreBuild()
	}()

	newUpdater = func(cfg *config.Config, paths config.Paths) updater {
		return stubUpdater{result: update.Result{
			CurrentVersion: "v0.1.0",
			LatestVersion:  "v0.2.0",
			ReleaseTag:     "v0.2.0",
			ReleaseURL:     "https://example.com/releases/v0.2.0",
			Status:         "update_available",
			Message:        "v0.1.0 -> v0.2.0",
		}}
	}
	checkOnly := requireOK(t, env.run(t, "", "--config", env.config, "update", "--check"))
	if !strings.Contains(checkOnly, "update: v0.1.0 -> v0.2.0") {
		t.Fatalf("unexpected check output:\n%s", checkOnly)
	}
}

type stubUpdater struct {
	result update.Result
	err    error
}

func (s stubUpdater) Update(context.Context, update.Request) (update.Result, error) {
	return s.result, s.err
}

func setCLICommandBuildInfo(version, commit, date string) func() {
	oldVersion := buildinfo.Version
	oldCommit := buildinfo.Commit
	oldDate := buildinfo.Date
	buildinfo.Version = version
	buildinfo.Commit = commit
	buildinfo.Date = date
	return func() {
		buildinfo.Version = oldVersion
		buildinfo.Commit = oldCommit
		buildinfo.Date = oldDate
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
