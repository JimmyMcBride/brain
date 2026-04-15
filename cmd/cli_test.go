package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"brain/internal/buildinfo"
	"brain/internal/config"
	"brain/internal/projectcontext"
	"brain/internal/skills"
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
	return e.runFromDir(t, e.moduleRoot, stdin, args...)
}

func (e *cliEnv) runFromDir(t *testing.T, dir, stdin string, args ...string) cliResult {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
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
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, root, "<ROOT>")
	s = strings.ReplaceAll(s, filepath.ToSlash(root), "<ROOT>")
	s = rfc3339Pattern.ReplaceAllString(s, "<TIME>")
	return s
}

func cliPath(parts ...string) string {
	return filepath.Join(parts...)
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

func TestCLIContextRefreshDoesNotCreateMissingAgentFiles(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	output := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "refresh", "--agent", "codex", "--agent", "openclaw"))
	if strings.Contains(output, ".codex/AGENTS.md") || strings.Contains(output, ".openclaw/AGENTS.md") {
		t.Fatalf("expected context refresh to skip missing agent files:\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".codex", "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected missing codex agent file to remain missing, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".openclaw", "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected missing openclaw agent file to remain missing, got err=%v", err)
	}
}

func TestCLIContextRefreshUpdatesExistingManagedAgentBlock(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	if err := os.MkdirAll(filepath.Join(env.project, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.project, ".claude", "CLAUDE.md"), []byte("## Brain\n\n<!-- brain:begin agent-integration-claude -->\nstale\n<!-- brain:end agent-integration-claude -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "refresh", "--agent", "claude"))
	if !strings.Contains(output, "updated   agent    .claude/CLAUDE.md") {
		t.Fatalf("unexpected context refresh output:\n%s", output)
	}

	body, err := os.ReadFile(filepath.Join(env.project, ".claude", "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "Brain-managed project context for `claude` lives under `.brain/`.") {
		t.Fatalf("expected refreshed claude integration:\n%s", text)
	}
	if strings.Contains(text, "canonical project contract") {
		t.Fatalf("unexpected canonical wording:\n%s", text)
	}
}

func TestCLIAdoptIntegratesExistingAgentFile(t *testing.T) {
	env := newCLIEnv(t)
	if err := os.MkdirAll(filepath.Join(env.project, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.project, ".codex", "AGENTS.md"), []byte("# Existing Codex Notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "adopt"))
	if !strings.Contains(output, "adopted   agent    .codex/AGENTS.md preserve-user") {
		t.Fatalf("unexpected adopt output:\n%s", output)
	}

	body, err := os.ReadFile(filepath.Join(env.project, ".codex", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "# Existing Codex Notes") || !strings.Contains(text, "## Brain") {
		t.Fatalf("expected existing codex file to gain Brain section:\n%s", text)
	}
	if strings.Contains(text, "canonical project contract") {
		t.Fatalf("unexpected canonical wording:\n%s", text)
	}
}

func TestCLIAdoptIntegratesExistingPiAgentFile(t *testing.T) {
	env := newCLIEnv(t)
	if err := os.MkdirAll(filepath.Join(env.project, ".pi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.project, ".pi", "AGENTS.md"), []byte("# Existing Pi Notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "adopt"))
	if !strings.Contains(output, "adopted   agent    .pi/AGENTS.md preserve-user") {
		t.Fatalf("unexpected pi adopt output:\n%s", output)
	}
}

func TestCLIAdoptWithAgentCreatesMissingAgentFile(t *testing.T) {
	env := newCLIEnv(t)

	output := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "adopt", "--agent", "codex"))
	if !strings.Contains(output, "created   agent    .codex/AGENTS.md") {
		t.Fatalf("unexpected adopt output:\n%s", output)
	}

	body, err := os.ReadFile(filepath.Join(env.project, ".codex", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "## Brain") || !strings.Contains(text, "<!-- brain:begin agent-integration-codex -->") {
		t.Fatalf("unexpected created codex integration file:\n%s", text)
	}
}

func TestCLIAdoptRejectsUnsupportedAgent(t *testing.T) {
	env := newCLIEnv(t)

	result := env.run(t, "", "--config", env.config, "--project", env.project, "adopt", "--agent", "codx")
	if result.err == nil || !strings.Contains(result.err.Error(), "unsupported agent") {
		t.Fatalf("expected unsupported agent error, got %+v", result)
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
	if !strings.Contains(searchOutput, "[") || !strings.Contains(searchOutput, "lex=") || !strings.Contains(searchOutput, "sem=") || !strings.Contains(searchOutput, "rec=") || !strings.Contains(searchOutput, "type=") || !strings.Contains(searchOutput, "ctx=") {
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

func TestCLISearchInjectIncludesContextBlockInHumanAndJSONModes(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "edit", "docs/project-overview.md", "-b", "# Project Overview\n\nInjectable context should cite the project overview note."))

	human := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "search", "--inject", "injectable context"))
	if !strings.Contains(human, "## Relevant Context") || !strings.Contains(human, "docs/project-overview.md") {
		t.Fatalf("expected injected context in human output:\n%s", human)
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "search", "--inject", "injectable context"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	results, ok := payload["results"].([]any)
	if !ok || len(results) == 0 {
		t.Fatalf("expected results in json payload: %#v", payload)
	}
	contextBlock, ok := payload["context_block"].(string)
	if !ok || !strings.Contains(contextBlock, "## Relevant Context") || !strings.Contains(contextBlock, "docs/project-overview.md") {
		t.Fatalf("expected context block in json payload: %#v", payload)
	}
}

func TestCLIDistillSessionCreatesProposal(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	initGitProject(t, env.project)

	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "tighten session distill"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "edit", ".brain/context/current-state.md", "-b", "# Current State\n\nSession context was updated.\n"))
	if err := os.WriteFile(filepath.Join(env.project, "main.go"), []byte("package main\nfunc main() { println(\"changed\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "version"))

	agentsBefore, err := os.ReadFile(filepath.Join(env.project, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}

	output := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "distill", "--session"))
	if !strings.Contains(output, "Created distill proposal .brain/resources/changes/tighten-session-distill-distill-proposal.md") {
		t.Fatalf("unexpected distill output:\n%s", output)
	}

	noteRaw, err := os.ReadFile(filepath.Join(env.project, ".brain", "resources", "changes", "tighten-session-distill-distill-proposal.md"))
	if err != nil {
		t.Fatal(err)
	}
	note := string(noteRaw)
	if !strings.Contains(note, "## Source Provenance") || !strings.Contains(note, "go version") || !strings.Contains(note, "main.go") {
		t.Fatalf("expected session-derived provenance in proposal:\n%s", note)
	}
	if !strings.Contains(note, "## Promotion Review") || !strings.Contains(note, "verification_recipe [promotable]") || !strings.Contains(note, "### .brain/resources/changes/tighten-session-distill.md") {
		t.Fatalf("expected promotion review and promotable target sections in proposal:\n%s", note)
	}

	agentsAfter, err := os.ReadFile(filepath.Join(env.project, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(agentsBefore) != string(agentsAfter) {
		t.Fatalf("expected distill not to modify AGENTS.md directly")
	}
}

func TestCLIDistillBrainstormModeMatchesLegacyWrapper(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "brainstorm", "start", "Auth Ideas"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "brainstorm", "idea", ".brain/brainstorms/auth-ideas.md", "-b", "Favor explicit durable memory over transcript storage."))

	topLevel := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "distill", "--brainstorm", ".brain/brainstorms/auth-ideas.md"))
	legacy := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "brainstorm", "distill", ".brain/brainstorms/auth-ideas.md"))
	if topLevel != legacy {
		t.Fatalf("expected equivalent command output\nnew:\n%s\nlegacy:\n%s", topLevel, legacy)
	}

	noteRaw, err := os.ReadFile(filepath.Join(env.project, ".brain", "resources", "changes", "auth-ideas-distill-proposal.md"))
	if err != nil {
		t.Fatal(err)
	}
	note := string(noteRaw)
	if !strings.Contains(note, "Mode: `brainstorm`") || !strings.Contains(note, "[[.brain/brainstorms/auth-ideas.md]]") {
		t.Fatalf("expected brainstorm provenance in proposal:\n%s", note)
	}
	if !strings.Contains(note, "### .brain/resources/decisions/auth-ideas.md") {
		t.Fatalf("expected proposed decision target in proposal:\n%s", note)
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
	if !strings.Contains(targets, "codex [global] brain "+cliPath("<ROOT>", "home", ".codex", "skills", "brain")) {
		t.Fatalf("missing global codex target:\n%s", targets)
	}
	if !strings.Contains(targets, "copilot [global] brain "+cliPath("<ROOT>", "home", ".copilot", "skills", "brain")) {
		t.Fatalf("missing global copilot target:\n%s", targets)
	}
	if !strings.Contains(targets, "copilot [local] brain "+cliPath("<ROOT>", "project", ".github", "skills", "brain")) {
		t.Fatalf("missing local copilot target:\n%s", targets)
	}
	if !strings.Contains(targets, "pi [global] brain "+cliPath("<ROOT>", "home", ".pi", "agent", "skills", "brain")) {
		t.Fatalf("missing global pi target:\n%s", targets)
	}
	if !strings.Contains(targets, "pi [local] brain "+cliPath("<ROOT>", "project", ".pi", "skills", "brain")) {
		t.Fatalf("missing local pi target:\n%s", targets)
	}
	if !strings.Contains(targets, "zed [local] brain "+cliPath("<ROOT>", "project", ".zed", "skills", "brain")) {
		t.Fatalf("missing local zed target:\n%s", targets)
	}
	requireOK(t, env.run(t, "", "skills", "install", "--scope", "local", "-a", "codex", "--project", env.project))
	if _, err := os.Stat(filepath.Join(env.project, ".codex", "skills", "brain", "SKILL.md")); err != nil {
		t.Fatalf("expected local skill install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".codex", "skills", "brain", ".brain-skill-manifest.json")); err != nil {
		t.Fatalf("expected local skill manifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".codex", "skills", "googleworkspace-cli")); !os.IsNotExist(err) {
		t.Fatalf("expected no non-brain skill installs, got err=%v", err)
	}

	help := requireOK(t, env.run(t, "", "skills", "install", "--help"))
	if strings.Contains(help, "--skill") || strings.Contains(help, "--skill-root") {
		t.Fatalf("expected help to omit removed flags:\n%s", help)
	}
	if strings.Contains(help, "--mode") {
		t.Fatalf("expected help to omit removed mode flag:\n%s", help)
	}
}

func TestCLISkillsCommandsWorkOutsideRepoCheckout(t *testing.T) {
	env := newCLIEnv(t)

	targets := requireOK(t, env.runFromDir(t, env.project, "", "skills", "targets", "--scope", "global", "-a", "codex"))
	if !strings.Contains(targets, "codex [global] brain "+cliPath("<ROOT>", "home", ".codex", "skills", "brain")) {
		t.Fatalf("missing global codex target from non-repo cwd:\n%s", targets)
	}

	requireOK(t, env.runFromDir(t, env.project, "", "skills", "install", "--scope", "global", "-a", "codex"))
	if _, err := os.Stat(filepath.Join(env.home, ".codex", "skills", "brain", "SKILL.md")); err != nil {
		t.Fatalf("expected global skill install from non-repo cwd: %v", err)
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

func TestCLIContextLoadLevels(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "edit", "docs/project-overview.md", "-b", "# Project Overview\n\nLayered context helps agents stay fast."))

	level0 := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "load", "--level", "0"))
	if !strings.Contains(level0, "## Source: AGENTS.md (summary)") || !strings.Contains(level0, "## Source: .brain/context/current-state.md") {
		t.Fatalf("unexpected level 0 output:\n%s", level0)
	}
	if strings.Contains(level0, ".brain/context/overview.md") {
		t.Fatalf("expected level 0 to omit overview:\n%s", level0)
	}

	level2JSON := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "load", "--level", "2"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(level2JSON), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, level2JSON)
	}
	if payload["level"].(float64) != 2 {
		t.Fatalf("expected level 2 payload: %#v", payload)
	}
	sources, ok := payload["sources"].([]any)
	if !ok || len(sources) != 7 {
		t.Fatalf("expected 7 static sources in level 2 payload: %#v", payload)
	}
	content, ok := payload["content"].(string)
	if !ok || !strings.Contains(content, "## Source: .brain/context/architecture.md") || !strings.Contains(content, "## Source: .brain/context/memory-policy.md") {
		t.Fatalf("unexpected level 2 content: %#v", payload)
	}
}

func TestCLIContextLoadLevelThreeUsesQueryOrActiveSession(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "edit", "docs/project-overview.md", "-b", "# Project Overview\n\nLayered context query should retrieve this overview."))

	missing := env.run(t, "", "--config", env.config, "--project", env.project, "context", "load", "--level", "3")
	if missing.err == nil || !strings.Contains(missing.err.Error(), "requires --query or an active session task") {
		t.Fatalf("expected missing-query error, got err=%v stdout=%s stderr=%s", missing.err, missing.stdout, missing.stderr)
	}

	initGitProject(t, env.project)
	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "layered context query"))
	level3 := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "load", "--level", "3"))
	if !strings.Contains(level3, "## Source: search:layered context query") || !strings.Contains(level3, "## Relevant Context") || !strings.Contains(level3, "docs/project-overview.md") {
		t.Fatalf("unexpected level 3 output:\n%s", level3)
	}
}

func TestCLIContextAssembleRequiresTaskOrActiveSession(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))

	missing := env.run(t, "", "--config", env.config, "--project", env.project, "context", "assemble")
	if missing.err == nil || !strings.Contains(missing.err.Error(), "requires --task or an active session task") {
		t.Fatalf("expected missing-task error, got err=%v stdout=%s stderr=%s", missing.err, missing.stdout, missing.stderr)
	}
}

func TestCLIContextAssembleResolvesTaskFromFlagAndSession(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))

	byFlag := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "assemble", "--task", "tighten auth flow"))
	if !strings.Contains(byFlag, "## Task Context") || !strings.Contains(byFlag, "- Task: `tighten auth flow`") || !strings.Contains(byFlag, "- Source: `flag`") {
		t.Fatalf("unexpected flag-based assemble output:\n%s", byFlag)
	}
	if !strings.Contains(byFlag, "## Selected Context") {
		t.Fatalf("expected selected context section:\n%s", byFlag)
	}

	initGitProject(t, env.project)
	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "session derived task"))
	bySession := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "assemble"))
	if !strings.Contains(bySession, "- Task: `session derived task`") || !strings.Contains(bySession, "- Source: `session`") {
		t.Fatalf("unexpected session-based assemble output:\n%s", bySession)
	}
}

func TestCLIContextAssembleJSONReturnsStablePacketShape(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "assemble", "--task", "tighten auth flow"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	task, ok := payload["task"].(map[string]any)
	if !ok || task["text"] != "tighten auth flow" || task["source"] != "flag" {
		t.Fatalf("unexpected task payload: %#v", payload)
	}
	summary, ok := payload["summary"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected summary payload: %#v", payload)
	}
	confidence, ok := summary["confidence"].(string)
	if !ok || confidence == "" {
		t.Fatalf("expected confidence in summary payload: %#v", payload)
	}
	if _, ok := summary["selected_count"].(float64); !ok {
		t.Fatalf("expected selected_count in summary payload: %#v", payload)
	}
	selected, ok := payload["selected"].(map[string]any)
	if !ok {
		t.Fatalf("expected selected groups in payload: %#v", payload)
	}
	for _, key := range []string{"durable_notes", "generated_context", "structural_repo", "live_work", "policy_workflow"} {
		if _, ok := selected[key].([]any); !ok {
			t.Fatalf("expected %s group in payload: %#v", key, payload)
		}
	}
	omitted, ok := payload["omitted_nearby"].(map[string]any)
	if !ok {
		t.Fatalf("expected omitted groups in payload: %#v", payload)
	}
	for _, key := range []string{"durable_notes", "generated_context", "structural_repo", "live_work", "policy_workflow"} {
		if _, ok := omitted[key].([]any); !ok {
			t.Fatalf("expected omitted %s group in payload: %#v", key, payload)
		}
	}
}

func TestCLIContextAssembleSelectsFirstWaveSourceGroups(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))
	if err := os.MkdirAll(filepath.Join(env.project, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.project, "docs", "auth-flow.md"), []byte("# Auth Flow\n\nTighten the auth flow around bearer token refresh.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "assemble", "--task", "auth flow"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	summary := payload["summary"].(map[string]any)
	if summary["selected_count"].(float64) == 0 {
		t.Fatalf("expected selected sources in packet: %#v", payload)
	}
	groupCounts := summary["group_counts"].(map[string]any)
	if groupCounts["durable_notes"].(float64) == 0 || groupCounts["generated_context"].(float64) == 0 || groupCounts["policy_workflow"].(float64) == 0 {
		t.Fatalf("expected first-wave groups to be selected: %#v", payload)
	}
	if groupCounts["structural_repo"].(float64) != 0 {
		t.Fatalf("expected structural repo to remain empty in this packet: %#v", payload)
	}
}

func TestCLIContextAssembleIncludesStructuralRepoContext(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	for path, body := range map[string]string{
		"go.mod":                    "module example.com/test\n\ngo 1.26\n",
		"docs/search-overview.md":   "# Search Overview\n\nSearch context overview for task assembly.\n",
		"internal/search/search.go": "package search\n",
		"config/search.yaml":        "name: search\n",
	} {
		if err := os.MkdirAll(filepath.Join(env.project, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(env.project, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	human := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "assemble", "--task", "search config"))
	if !strings.Contains(human, "### Structural Repo") {
		t.Fatalf("expected structural repo section in assemble output:\n%s", human)
	}
	if !strings.Contains(human, "`config/`") && !strings.Contains(human, "`config/search.yaml`") && !strings.Contains(human, "`internal/search/`") {
		t.Fatalf("expected at least one structural path in assemble output:\n%s", human)
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "assemble", "--task", "search config"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	summary := payload["summary"].(map[string]any)
	groupCounts := summary["group_counts"].(map[string]any)
	if groupCounts["structural_repo"].(float64) == 0 {
		t.Fatalf("expected structural repo count in summary: %#v", payload)
	}
	selected := payload["selected"].(map[string]any)
	structural := selected["structural_repo"].([]any)
	if len(structural) == 0 {
		t.Fatalf("expected structural repo items in packet: %#v", payload)
	}
	first := structural[0].(map[string]any)
	if first["kind"] != "structural" || first["why"] == "" {
		t.Fatalf("expected structural packet item fields: %#v", payload)
	}
}

func TestCLIContextStructureStatusReportsFreshness(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	missing := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "structure", "status"))
	if !strings.Contains(missing, "state: missing (structure metadata missing)") {
		t.Fatalf("unexpected missing structure status:\n%s", missing)
	}

	statusJSON := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "structure", "status"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(statusJSON), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, statusJSON)
	}
	if payload["state"] != "missing" || payload["reason"] != "structure metadata missing" {
		t.Fatalf("unexpected structure status payload: %#v", payload)
	}
}

func TestCLIContextStructureRebuildsAndSupportsPathFilter(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	for path, body := range map[string]string{
		"go.mod":                         "module example.com/test\n\ngo 1.26\n",
		"cmd/brain/main.go":              "package main\nfunc main() {}\n",
		"internal/search/search.go":      "package search\n",
		"internal/search/search_test.go": "package search\n",
		".github/workflows/ci.yml":       "name: ci\n",
		"config/app.yaml":                "name: app\n",
	} {
		if err := os.MkdirAll(filepath.Join(env.project, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(env.project, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	human := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "structure"))
	if !strings.Contains(human, "## Repository Shape") || !strings.Contains(human, "## Boundaries") || !strings.Contains(human, "## Entrypoints") || !strings.Contains(human, "## Config Surfaces") || !strings.Contains(human, "## Test Surfaces") {
		t.Fatalf("unexpected structure output:\n%s", human)
	}
	if !strings.Contains(human, "`internal/search/`") || !strings.Contains(human, "`cmd/brain/main.go`") {
		t.Fatalf("expected structural items in output:\n%s", human)
	}

	filtered := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "structure", "--path", "internal/search"))
	if !strings.Contains(filtered, "`internal/search/`") {
		t.Fatalf("expected filtered structural boundary:\n%s", filtered)
	}
	if strings.Contains(filtered, "`cmd/brain/main.go`") {
		t.Fatalf("expected path filter to exclude unrelated entrypoints:\n%s", filtered)
	}
}

func TestCLIContextLiveRequiresTaskOrActiveSession(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))

	missing := env.run(t, "", "--config", env.config, "--project", env.project, "context", "live")
	if missing.err == nil || !strings.Contains(missing.err.Error(), "requires --task or an active session task") {
		t.Fatalf("expected missing-task error, got err=%v stdout=%s stderr=%s", missing.err, missing.stdout, missing.stderr)
	}
}

func TestCLIContextLiveResolvesTaskFromFlagAndSession(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))

	byFlag := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "live", "--task", "tighten auth flow"))
	if !strings.Contains(byFlag, "## Task") || !strings.Contains(byFlag, "- Task: `tighten auth flow`") || !strings.Contains(byFlag, "- Source: `flag`") {
		t.Fatalf("unexpected flag-based live output:\n%s", byFlag)
	}

	initGitProject(t, env.project)
	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "session live task"))
	bySession := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "live"))
	if !strings.Contains(bySession, "- Task: `session live task`") || !strings.Contains(bySession, "- Source: `session`") || !strings.Contains(bySession, "## Session") {
		t.Fatalf("unexpected session-based live output:\n%s", bySession)
	}
}

func TestCLIContextLiveJSONReturnsStablePacketShape(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "live", "--task", "tighten auth flow"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	task, ok := payload["task"].(map[string]any)
	if !ok || task["text"] != "tighten auth flow" || task["source"] != "flag" {
		t.Fatalf("unexpected task payload: %#v", payload)
	}
	sessionPayload, ok := payload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected session payload: %#v", payload)
	}
	if _, ok := sessionPayload["active"].(bool); !ok {
		t.Fatalf("expected active boolean in session payload: %#v", payload)
	}
	worktree, ok := payload["worktree"].(map[string]any)
	if !ok {
		t.Fatalf("expected worktree payload: %#v", payload)
	}
	for _, key := range []string{"changed_files", "touched_boundaries"} {
		if _, ok := worktree[key].([]any); !ok {
			t.Fatalf("expected %s array in worktree payload: %#v", key, payload)
		}
	}
	if _, ok := payload["nearby_tests"].([]any); !ok {
		t.Fatalf("expected nearby_tests array: %#v", payload)
	}
	verification, ok := payload["verification"].(map[string]any)
	if !ok {
		t.Fatalf("expected verification payload: %#v", payload)
	}
	for _, key := range []string{"recent_commands", "profiles", "recipes"} {
		if _, ok := verification[key].([]any); !ok {
			t.Fatalf("expected %s array in verification payload: %#v", key, payload)
		}
	}
	if _, ok := payload["policy_hints"].([]any); !ok {
		t.Fatalf("expected policy_hints array: %#v", payload)
	}
	if _, ok := payload["ambiguities"].([]any); !ok {
		t.Fatalf("expected ambiguities array: %#v", payload)
	}
}

func TestCLIContextLiveIncludesChangedFilesTouchedBoundariesAndNearbyTests(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	for path, body := range map[string]string{
		"internal/search/search.go":      "package search\n",
		"internal/search/search_test.go": "package search\n",
		"config/search.yaml":             "name: search\n",
	} {
		if err := os.MkdirAll(filepath.Join(env.project, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(env.project, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	initGitProject(t, env.project)
	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "search config"))
	if err := os.WriteFile(filepath.Join(env.project, "internal", "search", "search.go"), []byte("package search\n\nfunc Search() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	human := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "live"))
	if !strings.Contains(human, "## Changed Files") || !strings.Contains(human, "`internal/search/search.go`") {
		t.Fatalf("expected changed files in live output:\n%s", human)
	}
	if !strings.Contains(human, "## Touched Boundaries") || !strings.Contains(human, "`internal/search/`") {
		t.Fatalf("expected touched boundary in live output:\n%s", human)
	}
	if !strings.Contains(human, "## Nearby Tests") || !strings.Contains(human, "`internal/search/search_test.go`") {
		t.Fatalf("expected nearby test in live output:\n%s", human)
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "live"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	worktree := payload["worktree"].(map[string]any)
	if len(worktree["changed_files"].([]any)) == 0 || len(worktree["touched_boundaries"].([]any)) == 0 {
		t.Fatalf("expected changed files and touched boundaries in payload: %#v", payload)
	}
	if len(payload["nearby_tests"].([]any)) == 0 {
		t.Fatalf("expected nearby tests in payload: %#v", payload)
	}
}

func TestCLIContextLiveReportsVerificationProfilesAndPolicyHints(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	for path, body := range map[string]string{
		"internal/search/search.go":      "package search\n",
		"internal/search/search_test.go": "package search\n",
	} {
		if err := os.MkdirAll(filepath.Join(env.project, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(env.project, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	override := "closeout:\n  verification_profiles:\n    - name: tests\n      commands:\n        - go test ./...\n    - name: build\n      commands:\n        - go build ./...\n"
	if err := os.WriteFile(filepath.Join(env.project, ".brain", "policy.override.yaml"), []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	initGitProject(t, env.project)
	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "search config"))
	if err := os.WriteFile(filepath.Join(env.project, "internal", "search", "search.go"), []byte("package search\n\nfunc Search() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "test", "./..."))

	human := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "live"))
	if !strings.Contains(human, "## Verification") || !strings.Contains(human, "go test ./...") || !strings.Contains(human, "recipe `tests`") {
		t.Fatalf("expected verification output in live context:\n%s", human)
	}
	if !strings.Contains(human, "## Policy Hints") || !strings.Contains(human, "Verification workflow") || !strings.Contains(human, "Durable memory update") {
		t.Fatalf("expected policy hints in live context:\n%s", human)
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "live"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	verification := payload["verification"].(map[string]any)
	if len(verification["recent_commands"].([]any)) == 0 || len(verification["profiles"].([]any)) == 0 || len(verification["recipes"].([]any)) == 0 {
		t.Fatalf("expected verification details in payload: %#v", payload)
	}
	if len(payload["policy_hints"].([]any)) == 0 {
		t.Fatalf("expected policy hints in payload: %#v", payload)
	}
}

func TestCLIContextAssembleIncludesLiveWorkContext(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	for path, body := range map[string]string{
		"docs/search-overview.md":        "# Search Overview\n\nSearch context overview for task assembly.\n",
		"internal/search/search.go":      "package search\n",
		"internal/search/search_test.go": "package search\n",
	} {
		if err := os.MkdirAll(filepath.Join(env.project, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(env.project, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	initGitProject(t, env.project)
	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "search config"))
	if err := os.WriteFile(filepath.Join(env.project, "internal", "search", "search.go"), []byte("package search\n\nfunc Search() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	human := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "assemble"))
	if !strings.Contains(human, "### Live Work") {
		t.Fatalf("expected live work section in assemble output:\n%s", human)
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "assemble"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	summary := payload["summary"].(map[string]any)
	groupCounts := summary["group_counts"].(map[string]any)
	if groupCounts["live_work"].(float64) == 0 {
		t.Fatalf("expected live work count in summary: %#v", payload)
	}
}

func TestCLIContextAssembleExplainReportsRationaleAndAmbiguities(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "context", "install", "--project", env.project))
	if err := os.MkdirAll(filepath.Join(env.project, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.project, "docs", "workflow-guide.md"), []byte("# Workflow Guide\n\nTask workflow guide for auth flow changes.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.project, "docs", "workflow-extra.md"), []byte("# Workflow Extra\n\nNearby workflow notes for auth flow changes.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	human := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "assemble", "--task", "auth flow workflow", "--explain"))
	if !strings.Contains(human, "## Why This Was Selected") || !strings.Contains(human, "## Omitted Nearby Context") || !strings.Contains(human, "## Missing Or Unused Source Groups") {
		t.Fatalf("expected explain sections in human output:\n%s", human)
	}
	if !strings.Contains(human, "## Ambiguities") {
		t.Fatalf("expected ambiguity section in explain output:\n%s", human)
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "assemble", "--task", "auth flow workflow", "--explain"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	summary := payload["summary"].(map[string]any)
	if summary["confidence"] != "medium" && summary["confidence"] != "low" {
		t.Fatalf("expected explain run to compute a non-empty confidence bucket: %#v", payload)
	}
	ambiguities := payload["ambiguities"].([]any)
	if len(ambiguities) == 0 {
		t.Fatalf("expected explain run to report ambiguities: %#v", payload)
	}
	selected := payload["selected"].(map[string]any)
	durable := selected["durable_notes"].([]any)
	if len(durable) == 0 {
		t.Fatalf("expected durable notes in explain packet: %#v", payload)
	}
	first := durable[0].(map[string]any)
	if first["selection_method"] == "" || first["rank"].(float64) == 0 {
		t.Fatalf("expected explain metadata on selected item: %#v", payload)
	}
}

func TestCLIContextCompileWithExplicitTask(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	if err := os.MkdirAll(filepath.Join(env.project, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.project, "docs", "context-compiler.md"), []byte("# Context Compiler\n\nKeep compiled context packets compact and deterministic.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	human := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "compile", "--task", "context compiler deterministic packet"))
	for _, section := range []string{"## Compiled Context Packet", "## Base Contract", "## Working Set", "## Verification Hints", "## Provenance"} {
		if !strings.Contains(human, section) {
			t.Fatalf("expected section %q in compile output:\n%s", section, human)
		}
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "compile", "--task", "context compiler deterministic packet"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	task := payload["task"].(map[string]any)
	if task["text"] != "context compiler deterministic packet" || task["source"] != "flag" {
		t.Fatalf("unexpected compile task payload: %#v", payload)
	}
	baseContract := payload["base_contract"].([]any)
	if len(baseContract) != 5 {
		t.Fatalf("expected five base contract items: %#v", payload)
	}
	workingSet := payload["working_set"].(map[string]any)
	if _, ok := workingSet["notes"]; !ok {
		t.Fatalf("expected working_set.notes in compile payload: %#v", payload)
	}
	if _, ok := payload["provenance"]; !ok {
		t.Fatalf("expected provenance in compile payload: %#v", payload)
	}
	if _, ok := payload["verification"].([]any); !ok {
		t.Fatalf("expected compiled verification array in payload: %#v", payload)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".brain", "session.json")); !os.IsNotExist(err) {
		t.Fatalf("expected explicit no-session compile not to create an active session file, err=%v", err)
	}
}

func TestCLIContextCompileIncludesVerificationRecipes(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	for path, body := range map[string]string{
		"go.mod":                         "module example.com/test\n\ngo 1.26\n",
		"internal/search/search.go":      "package search\n",
		"internal/search/search_test.go": "package search\n",
	} {
		if err := os.MkdirAll(filepath.Join(env.project, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(env.project, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	override := "closeout:\n  verification_profiles:\n    - name: tests\n      commands:\n        - go test ./...\n    - name: build\n      commands:\n        - go build ./...\n"
	if err := os.WriteFile(filepath.Join(env.project, ".brain", "policy.override.yaml"), []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	initGitProject(t, env.project)
	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "compile verification packet"))
	if err := os.WriteFile(filepath.Join(env.project, "internal", "search", "search.go"), []byte("package search\n\nfunc Search() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "test", "./..."))

	human := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "compile"))
	if !strings.Contains(human, "go test ./...") || !strings.Contains(human, "strong") {
		t.Fatalf("expected verification recipes in compile output:\n%s", human)
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "compile"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	verification := payload["verification"].([]any)
	if len(verification) == 0 {
		t.Fatalf("expected compiled verification hints in payload: %#v", payload)
	}
	first := verification[0].(map[string]any)
	if first["command"] == "" || first["strength"] == "" {
		t.Fatalf("expected command and strength in verification hint: %#v", payload)
	}
}

func TestCLIContextCompileUsesActiveSessionTask(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	initGitProject(t, env.project)

	startOutput := requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "session-backed context compile"))
	if !strings.Contains(startOutput, "Started session") {
		t.Fatalf("unexpected session start output:\n%s", startOutput)
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "compile"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("parse json output: %v\n%s", err, jsonOut)
	}
	task := payload["task"].(map[string]any)
	if task["text"] != "session-backed context compile" || task["source"] != "session" {
		t.Fatalf("expected session task resolution in compile payload: %#v", payload)
	}

	raw, err := os.ReadFile(filepath.Join(env.project, ".brain", "session.json"))
	if err != nil {
		t.Fatalf("read active session: %v", err)
	}
	var sessionPayload map[string]any
	if err := json.Unmarshal(raw, &sessionPayload); err != nil {
		t.Fatalf("parse active session: %v\n%s", err, raw)
	}
	packetRecords := sessionPayload["packet_records"].([]any)
	if len(packetRecords) != 1 {
		t.Fatalf("expected one packet record after session-backed compile: %#v", sessionPayload)
	}
}

func TestCLIContextExplainShowsRecordedPacketOutcomes(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "brainstorm", "start", "Compiler Telemetry Signal"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "brainstorm", "idea", ".brain/brainstorms/compiler-telemetry-signal.md", "-b", "Compiler telemetry signal note for explain output."))
	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "compiler telemetry signal"))

	compiledJSON := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "compile"))
	var compiled map[string]any
	if err := json.Unmarshal([]byte(compiledJSON), &compiled); err != nil {
		t.Fatalf("parse compile payload: %v\n%s", err, compiledJSON)
	}
	workingSet := compiled["working_set"].(map[string]any)
	notes := workingSet["notes"].([]any)
	if len(notes) == 0 {
		t.Fatalf("expected compile packet to include at least one note: %#v", compiled)
	}
	firstNote := notes[0].(map[string]any)
	anchor := firstNote["anchor"].(map[string]any)
	notePath := anchor["path"].(string)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "read", notePath))

	human := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "explain", "--last"))
	for _, section := range []string{"## Packet", "## Included Items", "## Expanded Later", "## Downstream Outcomes"} {
		if !strings.Contains(human, section) {
			t.Fatalf("expected section %q in explain output:\n%s", section, human)
		}
	}
	if !strings.Contains(human, notePath) {
		t.Fatalf("expected explain output to reference expanded note path %q:\n%s", notePath, human)
	}

	explainJSON := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "explain", "--last"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(explainJSON), &payload); err != nil {
		t.Fatalf("parse explain payload: %v\n%s", err, explainJSON)
	}
	packet := payload["packet"].(map[string]any)
	if packet["packet_hash"] == "" {
		t.Fatalf("expected packet hash in explain payload: %#v", payload)
	}
	if len(payload["included_items"].([]any)) == 0 {
		t.Fatalf("expected included items in explain payload: %#v", payload)
	}
	if len(payload["expanded_later"].([]any)) == 0 {
		t.Fatalf("expected expanded item telemetry in explain payload: %#v", payload)
	}
}

func TestCLIContextStatsSummarizesSignalAndVerificationLinks(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "brainstorm", "start", "Compiler Telemetry Signal"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "brainstorm", "idea", ".brain/brainstorms/compiler-telemetry-signal.md", "-b", "Compiler telemetry signal note for stats output."))
	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "compiler telemetry signal"))

	compiledJSON := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "compile"))
	var compiled map[string]any
	if err := json.Unmarshal([]byte(compiledJSON), &compiled); err != nil {
		t.Fatalf("parse first compile payload: %v\n%s", err, compiledJSON)
	}
	workingSet := compiled["working_set"].(map[string]any)
	notes := workingSet["notes"].([]any)
	if len(notes) == 0 {
		t.Fatalf("expected compile packet to include at least one note: %#v", compiled)
	}
	firstNote := notes[0].(map[string]any)
	anchor := firstNote["anchor"].(map[string]any)
	notePath := anchor["path"].(string)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "read", notePath))
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "version"))

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "compile"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "read", notePath))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "compile"))

	human := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "stats", "--limit", "3"))
	for _, section := range []string{"## Context Stats", "## Top Signal", "## Frequently Expanded", "## Common Verification Links"} {
		if !strings.Contains(human, section) {
			t.Fatalf("expected section %q in stats output:\n%s", section, human)
		}
	}
	if !strings.Contains(human, "likely_utility=likely_signal") {
		t.Fatalf("expected likely signal wording in stats output:\n%s", human)
	}
	if !strings.Contains(human, "go version") {
		t.Fatalf("expected recorded verification command in stats output:\n%s", human)
	}
	explainHuman := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "explain", "--last"))
	if !strings.Contains(explainHuman, "boosted by local utility signal") {
		t.Fatalf("expected explain output to surface utility-aware selection reasons:\n%s", explainHuman)
	}

	statsJSON := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "context", "stats", "--limit", "3"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(statsJSON), &payload); err != nil {
		t.Fatalf("parse stats payload: %v\n%s", err, statsJSON)
	}
	if len(payload["top_signal"].([]any)) == 0 {
		t.Fatalf("expected top signal items in stats payload: %#v", payload)
	}
	if len(payload["frequently_expanded"].([]any)) == 0 {
		t.Fatalf("expected frequently expanded items in stats payload: %#v", payload)
	}
	if len(payload["common_verification_links"].([]any)) == 0 {
		t.Fatalf("expected verification links in stats payload: %#v", payload)
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

	finishBlocked := env.run(t, "", "--config", env.config, "session", "finish", "--project", env.project, "--summary", "premature closeout")
	if finishBlocked.err == nil || !strings.Contains(finishBlocked.stdout, "durable note update required for repo changes") || !strings.Contains(finishBlocked.stdout, "brain distill --session") {
		t.Fatalf("expected finish to block and suggest distill:\nstdout=%s\nstderr=%s", finishBlocked.stdout, finishBlocked.stderr)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "distill", "--session"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "edit", "AGENTS.md", "-b", "# Project Agent Contract\n\nRecorded durable note for project changes.\n"))
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "test", "./..."))
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "build", "./..."))

	finishOutput := requireOK(t, env.run(t, "", "--config", env.config, "session", "finish", "--project", env.project, "--summary", "session complete"))
	if !strings.Contains(finishOutput, "finished") || !strings.Contains(finishOutput, ".brain/sessions/") {
		t.Fatalf("unexpected finish output:\n%s", finishOutput)
	}
}

func TestCLISessionFinishSurfacesPromotionSuggestions(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))
	initGitProject(t, env.project)

	requireOK(t, env.run(t, "", "--config", env.config, "session", "start", "--project", env.project, "--task", "replace context loading flow"))
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "context", "compile"))
	if err := os.WriteFile(filepath.Join(env.project, "main.go"), []byte("package main\nfunc main() { println(\"x\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	requireOK(t, env.run(t, "", "--config", env.config, "session", "run", "--project", env.project, "--", "go", "version"))

	finishBlocked := env.run(t, "", "--config", env.config, "session", "finish", "--project", env.project, "--summary", "premature closeout")
	if finishBlocked.err == nil {
		t.Fatalf("expected finish to block, got stdout=%s stderr=%s", finishBlocked.stdout, finishBlocked.stderr)
	}
	for _, needle := range []string{"Promote: boundary_fact", "Support: packets", "Support: verification"} {
		if !strings.Contains(finishBlocked.stdout, needle) {
			t.Fatalf("expected %q in blocked finish output:\nstdout=%s\nstderr=%s", needle, finishBlocked.stdout, finishBlocked.stderr)
		}
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
	restoreSkillRunner := skillInstallRunner
	restoreMigrationRunner := projectMigrationRunner
	restoreBuild := setCLICommandBuildInfo("v0.1.0", "abc123", "2026-04-10T00:00:00Z")
	defer func() {
		newUpdater = restoreUpdater
		skillInstallRunner = restoreSkillRunner
		projectMigrationRunner = restoreMigrationRunner
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
	skillInstallRunner = func(binaryPath, configPath, projectPath string, scope skills.Scope, agents []string) ([]skills.InstallResult, error) {
		t.Fatal("unexpected skill refresh during update --check")
		return nil, nil
	}
	projectMigrationRunner = func(binaryPath, configPath, projectPath string) (*projectcontext.ApplyProjectMigrationsResult, error) {
		t.Fatal("unexpected project migration during update --check")
		return nil, nil
	}
	checkOnly := requireOK(t, env.run(t, "", "--config", env.config, "update", "--check"))
	if !strings.Contains(checkOnly, "update: v0.1.0 -> v0.2.0") {
		t.Fatalf("unexpected check output:\n%s", checkOnly)
	}
}

func TestCLIUpdateRefreshesInstalledSkills(t *testing.T) {
	env := newCLIEnv(t)
	restoreUpdater := newUpdater
	restoreSkillRunner := skillInstallRunner
	restoreMigrationRunner := projectMigrationRunner
	restoreBuild := setCLICommandBuildInfo("v0.1.0", "abc123", "2026-04-10T00:00:00Z")
	defer func() {
		newUpdater = restoreUpdater
		skillInstallRunner = restoreSkillRunner
		projectMigrationRunner = restoreMigrationRunner
		restoreBuild()
	}()

	if err := os.MkdirAll(filepath.Join(env.home, ".codex", "skills", "brain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.home, ".codex", "skills", "brain", "SKILL.md"), []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(env.project, ".github", "skills", "brain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.project, ".github", "skills", "brain", "SKILL.md"), []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}

	newUpdater = func(cfg *config.Config, paths config.Paths) updater {
		return stubUpdater{result: update.Result{
			CurrentVersion: "v0.1.0",
			LatestVersion:  "v0.1.0",
			Status:         "up_to_date",
			Message:        "already up to date (v0.1.0)",
			CurrentPath:    filepath.Join(env.root, "bin", "brain"),
		}}
	}

	var calls []string
	var migrationCalls []string
	skillInstallRunner = func(binaryPath, configPath, projectPath string, scope skills.Scope, agents []string) ([]skills.InstallResult, error) {
		calls = append(calls, string(scope)+":"+strings.Join(agents, ","))
		results := make([]skills.InstallResult, 0, len(agents))
		for _, agent := range agents {
			root := filepath.Join(env.home, "."+agent, "skills")
			if scope == skills.ScopeLocal {
				root = filepath.Join(projectPath, "."+agent, "skills")
				if agent == "copilot" {
					root = filepath.Join(projectPath, ".github", "skills")
				}
			}
			results = append(results, skills.InstallResult{
				Agent:  agent,
				Skill:  "brain",
				Scope:  string(scope),
				Root:   root,
				Path:   filepath.Join(root, "brain"),
				Method: "copy",
			})
		}
		return results, nil
	}
	projectMigrationRunner = func(binaryPath, configPath, projectPath string) (*projectcontext.ApplyProjectMigrationsResult, error) {
		migrationCalls = append(migrationCalls, binaryPath+":"+projectPath)
		return &projectcontext.ApplyProjectMigrationsResult{
			UsesBrain:           true,
			ProjectDir:          projectPath,
			Status:              "applied",
			AppliedMigrationIDs: []string{"refresh-brain-managed-context-v1", "refresh-existing-agent-integrations-v1"},
		}, nil
	}

	output := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "update"))
	if !strings.Contains(output, "skills:  refreshed") {
		t.Fatalf("expected skill refresh output:\n%s", output)
	}
	if !strings.Contains(output, "project migrations: applied") || !strings.Contains(output, "migration: refresh-brain-managed-context-v1") {
		t.Fatalf("expected project migration output:\n%s", output)
	}
	if len(calls) != 2 || calls[0] != "global:codex" || calls[1] != "local:copilot" {
		t.Fatalf("unexpected refresh calls: %v", calls)
	}
	if len(migrationCalls) != 1 || migrationCalls[0] != filepath.Join(env.root, "bin", "brain")+":"+env.project {
		t.Fatalf("unexpected migration calls: %v", migrationCalls)
	}
}

func TestCLIUpdateJSONIncludesProjectMigrationStatus(t *testing.T) {
	env := newCLIEnv(t)
	restoreUpdater := newUpdater
	restoreSkillRunner := skillInstallRunner
	restoreMigrationRunner := projectMigrationRunner
	restoreBuild := setCLICommandBuildInfo("v0.1.0", "abc123", "2026-04-10T00:00:00Z")
	defer func() {
		newUpdater = restoreUpdater
		skillInstallRunner = restoreSkillRunner
		projectMigrationRunner = restoreMigrationRunner
		restoreBuild()
	}()

	newUpdater = func(cfg *config.Config, paths config.Paths) updater {
		return stubUpdater{result: update.Result{
			CurrentVersion: "v0.1.0",
			LatestVersion:  "v0.1.0",
			Status:         "up_to_date",
			Message:        "already up to date (v0.1.0)",
			CurrentPath:    filepath.Join(env.root, "bin", "brain"),
		}}
	}
	skillInstallRunner = func(binaryPath, configPath, projectPath string, scope skills.Scope, agents []string) ([]skills.InstallResult, error) {
		return nil, nil
	}
	projectMigrationRunner = func(binaryPath, configPath, projectPath string) (*projectcontext.ApplyProjectMigrationsResult, error) {
		return &projectcontext.ApplyProjectMigrationsResult{
			UsesBrain:           true,
			ProjectDir:          projectPath,
			Status:              "applied",
			AppliedMigrationIDs: []string{"refresh-brain-managed-context-v1"},
		}, nil
	}

	jsonOut := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "update"))
	if !strings.Contains(jsonOut, "\"project_migration_status\": \"applied\"") || !strings.Contains(jsonOut, "\"applied_project_migrations\": [") {
		t.Fatalf("expected project migration fields in json output:\n%s", jsonOut)
	}
}

func TestCLILocalSkillPreflightRepairsLegacyInstall(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	skillDir := filepath.Join(env.project, ".codex", "skills", "brain")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "find", "overview"))
	if _, err := os.Stat(filepath.Join(skillDir, ".brain-skill-manifest.json")); err != nil {
		t.Fatalf("expected local skill repair manifest: %v", err)
	}
}

func TestCLIUpdateFailsWhenSkillRefreshIsIncomplete(t *testing.T) {
	env := newCLIEnv(t)
	restoreUpdater := newUpdater
	restoreSkillRunner := skillInstallRunner
	restoreMigrationRunner := projectMigrationRunner
	restoreBuild := setCLICommandBuildInfo("v0.1.0", "abc123", "2026-04-10T00:00:00Z")
	defer func() {
		newUpdater = restoreUpdater
		skillInstallRunner = restoreSkillRunner
		projectMigrationRunner = restoreMigrationRunner
		restoreBuild()
	}()

	if err := os.MkdirAll(filepath.Join(env.home, ".codex", "skills", "brain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.home, ".codex", "skills", "brain", "SKILL.md"), []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}

	newUpdater = func(cfg *config.Config, paths config.Paths) updater {
		return stubUpdater{result: update.Result{
			CurrentVersion: "v0.1.0",
			LatestVersion:  "v0.2.0",
			Status:         "updated",
			Message:        "v0.1.0 -> v0.2.0",
			Updated:        true,
			CurrentPath:    filepath.Join(env.root, "old-brain"),
			InstalledPath:  filepath.Join(env.root, "new-brain"),
		}}
	}
	skillInstallRunner = func(binaryPath, configPath, projectPath string, scope skills.Scope, agents []string) ([]skills.InstallResult, error) {
		return nil, fmt.Errorf("boom")
	}
	projectMigrationRunner = func(binaryPath, configPath, projectPath string) (*projectcontext.ApplyProjectMigrationsResult, error) {
		t.Fatal("unexpected project migration when skill refresh failed")
		return nil, nil
	}

	result := env.run(t, "", "--config", env.config, "update")
	if result.err == nil || !strings.Contains(result.err.Error(), "binary updated, skill refresh incomplete") {
		t.Fatalf("expected incomplete skill refresh error, got %+v", result)
	}
	if !strings.Contains(result.stdout, "skills:  failed") {
		t.Fatalf("expected printed failed skill refresh status:\n%s", result.stdout)
	}
}

func TestCLILocalProjectPreflightAppliesPendingMigrations(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	staleAgents := "# Project Agent Contract\n\n<!-- brain:begin agents-contract -->\nstale\n<!-- brain:end agents-contract -->\n\n## Local Notes\n\nkeep me\n"
	if err := os.WriteFile(filepath.Join(env.project, "AGENTS.md"), []byte(staleAgents), 0o644); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(env.project, ".brain", "state", "project-migrations.json")
	if err := os.Remove(ledgerPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "find", "overview"))
	if _, err := os.Stat(ledgerPath); err != nil {
		t.Fatalf("expected migration ledger to be written during preflight: %v", err)
	}
	agentsBody, err := os.ReadFile(filepath.Join(env.project, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agentsBody), "brain context compile") || !strings.Contains(string(agentsBody), "keep me") {
		t.Fatalf("expected preflight migration to refresh AGENTS.md:\n%s", string(agentsBody))
	}
}

func TestCLILocalProjectPreflightReportsMigrationRemediation(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	agentsPath := filepath.Join(env.project, "AGENTS.md")
	if err := os.Remove(agentsPath); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(agentsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(env.project, ".brain", "state", "project-migrations.json")
	if err := os.Remove(ledgerPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	result := env.run(t, "", "--config", env.config, "--project", env.project, "find", "overview")
	if result.err == nil || !strings.Contains(result.err.Error(), "project migrations blocked") {
		t.Fatalf("expected project migration failure with remediation, got err=%v stdout=%s", result.err, result.stdout)
	}
	for _, snippet := range []string{
		"brain doctor --project .",
		"brain context refresh --project .",
		"brain adopt --project .",
	} {
		if !strings.Contains(result.err.Error(), snippet) {
			t.Fatalf("expected remediation snippet %q in error: %v", snippet, result.err)
		}
	}
}

func TestCLISkipCommandsDoNotRunMigrationPreflight(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	ledgerPath := filepath.Join(env.project, ".brain", "state", "project-migrations.json")
	if err := os.Remove(ledgerPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "doctor"))
	if _, err := os.Stat(ledgerPath); !os.IsNotExist(err) {
		t.Fatalf("expected doctor to skip migration preflight, got err=%v", err)
	}
}

func TestCLIDoctorReportsProjectMigrationStatus(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "init"))

	pending := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "doctor"))
	if !strings.Contains(pending, "project_migrations: fail (pending") {
		t.Fatalf("expected pending project migrations in doctor output:\n%s", pending)
	}

	requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "find", "overview"))
	current := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "doctor"))
	if !strings.Contains(current, "project_migrations: ok (current)") {
		t.Fatalf("expected current project migrations in doctor output:\n%s", current)
	}

	ledgerPath := filepath.Join(env.project, ".brain", "state", "project-migrations.json")
	if err := os.WriteFile(ledgerPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "doctor"))
	if !strings.Contains(broken, "project_migrations: fail (broken") {
		t.Fatalf("expected broken project migrations in doctor output:\n%s", broken)
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
