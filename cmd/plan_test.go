package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brain/internal/modules"
	officialplanning "brain/internal/official/planning"
	"brain/internal/planning/application"
)

func TestCLIPlanningLocalWorkflowAndIdempotentAudit(t *testing.T) {
	env := newCLIEnv(t)
	fixture := filepath.Join(env.moduleRoot, "internal", "official", "planning", "local", "testdata", "compatible")
	if err := os.CopyFS(env.project, os.DirFS(fixture)); err != nil {
		t.Fatal(err)
	}

	result := runPlanningCLI(t, env, "", "--project", env.project, "plan", "status")
	if result.err == nil || !strings.Contains(result.err.Error(), "module_disabled") {
		t.Fatalf("expected disabled diagnostic, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".brain", "modules.yaml")); !os.IsNotExist(err) {
		t.Fatalf("disabled Planning command wrote module config: %v", err)
	}

	permissions := officialplanning.Registration().Descriptor.Permissions
	args := []string{"--project", env.project, "modules", "grant", officialplanning.ID}
	args = append(args, permissions...)
	requireOK(t, runPlanningCLI(t, env, "", args...))
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "modules", "enable", officialplanning.ID))

	status := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "status"))
	for _, expected := range []string{"state: compatible", "writable: true", "schema version: 3", "ownership: local"} {
		if !strings.Contains(status, expected) {
			t.Fatalf("status missing %q:\n%s", expected, status)
		}
	}
	brainstorms := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "list"))
	if !strings.Contains(brainstorms, "alpha\tAlpha Brainstorm") {
		t.Fatalf("unexpected brainstorm list:\n%s", brainstorms)
	}
	brainstorm := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "show", "alpha"))
	if !strings.Contains(brainstorm, "# Brainstorm: Alpha Brainstorm") {
		t.Fatalf("unexpected brainstorm show:\n%s", brainstorm)
	}
	specs := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "list"))
	if !strings.Contains(specs, "alpha-spec\tapproved\tAlpha Spec") {
		t.Fatalf("unexpected spec list:\n%s", specs)
	}
	spec := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "show", "alpha-spec"))
	if !strings.Contains(spec, "# Alpha Spec") {
		t.Fatalf("unexpected spec show:\n%s", spec)
	}

	preview := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "start", "CLI Flow"))
	if !strings.Contains(preview, "create\t.plan/brainstorms/cli-flow.md") ||
		!strings.Contains(preview, "Preview only") {
		t.Fatalf("unexpected preview:\n%s", preview)
	}
	createdPath := filepath.Join(env.project, ".plan", "brainstorms", "cli-flow.md")
	if _, err := os.Stat(createdPath); !os.IsNotExist(err) {
		t.Fatalf("preview wrote artifact: %v", err)
	}

	created := runPlanningCLI(t, env, "", "--json", "--project", env.project, "plan", "brainstorm", "start", "CLI Flow", "--confirm")
	output := requireOK(t, created)
	if created.stderr != "" {
		t.Fatalf("JSON command wrote stderr: %q", created.stderr)
	}
	var mutation planningMutationOutput
	if err := json.Unmarshal([]byte(output), &mutation); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, output)
	}
	if mutation.Action != application.MutationCreate || mutation.Event == nil ||
		mutation.Event.Name != application.EventBrainstormCreated {
		t.Fatalf("unexpected mutation: %#v", mutation)
	}
	if _, err := os.Stat(createdPath); err != nil {
		t.Fatal(err)
	}

	unchanged := requireOK(t, runPlanningCLI(t, env, "", "--json", "--project", env.project, "plan", "brainstorm", "start", "CLI Flow", "--confirm"))
	mutation = planningMutationOutput{}
	if err := json.Unmarshal([]byte(unchanged), &mutation); err != nil {
		t.Fatal(err)
	}
	if mutation.Action != application.MutationUnchanged || mutation.Event != nil {
		t.Fatalf("unexpected idempotent mutation: %#v", mutation)
	}
	history, err := os.ReadFile(filepath.Join(env.project, ".brain", "state", "history.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(history), application.EventBrainstormCreated); count != 2 {
		t.Fatalf("expected one event record with operation and ID references, got count=%d:\n%s", count, history)
	}

	before, err := os.ReadFile(createdPath)
	if err != nil {
		t.Fatal(err)
	}
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "modules", "disable", officialplanning.ID))
	result = runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "start", "Blocked", "--confirm")
	if result.err == nil || !strings.Contains(result.err.Error(), "module_disabled") {
		t.Fatalf("expected disabled command failure, got %#v", result)
	}
	after, err := os.ReadFile(createdPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("disabled command changed existing Planning artifact")
	}
	if _, err := os.Stat(filepath.Join(env.project, ".plan", "brainstorms", "blocked.md")); !os.IsNotExist(err) {
		t.Fatalf("disabled command created artifact: %v", err)
	}
}

func TestCLIPlanningFutureSchemaIsReadOnly(t *testing.T) {
	env := newCLIEnv(t)
	fixture := filepath.Join(env.moduleRoot, "internal", "official", "planning", "local", "testdata", "future")
	if err := os.CopyFS(env.project, os.DirFS(fixture)); err != nil {
		t.Fatal(err)
	}
	permissions := officialplanning.Registration().Descriptor.Permissions
	args := []string{"--project", env.project, "modules", "grant", officialplanning.ID}
	args = append(args, permissions...)
	requireOK(t, runPlanningCLI(t, env, "", args...))
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "modules", "enable", officialplanning.ID))
	status := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "status"))
	if !strings.Contains(status, "state: future_schema") || !strings.Contains(status, "writable: false") {
		t.Fatalf("unexpected future status:\n%s", status)
	}
	result := runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "start", "Blocked", "--confirm")
	if result.err == nil || !strings.Contains(result.err.Error(), application.ErrWorkspaceNotWritable.Error()) {
		t.Fatalf("expected future-schema write refusal, got %#v", result)
	}
}

func runPlanningCLI(t *testing.T, env *cliEnv, stdin string, args ...string) cliResult {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(env.moduleRoot); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	command := newRootCommand(rootOptions{
		in:                  strings.NewReader(stdin),
		out:                 &stdout,
		errOut:              &stderr,
		moduleRegistrations: []modules.Registration{officialplanning.Registration()},
	})
	command.SetArgs(args)
	err = command.Execute()
	return cliResult{
		stdout: normalizeCLIOutput(stdout.String(), env.root),
		stderr: normalizeCLIOutput(stderr.String(), env.root),
		err:    err,
	}
}
