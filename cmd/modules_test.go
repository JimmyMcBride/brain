package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/internal/app"
	"github.com/JimmyMcBride/brain/internal/modules"
	"github.com/JimmyMcBride/brain/internal/modules/testmodule"
)

func TestCLIModulesEmptyListCreatesNoModuleFiles(t *testing.T) {
	env := newCLIEnv(t)
	output := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "modules", "list"))
	if output != "No modules registered.\n" {
		t.Fatalf("unexpected output: %q", output)
	}
	assertNoModuleFiles(t, env.project)

	output = requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "--json", "modules", "list"))
	if output != "[]\n" {
		t.Fatalf("unexpected JSON output: %q", output)
	}
	assertNoModuleFiles(t, env.project)
}

func TestCLIModulesLifecycleHumanAndJSON(t *testing.T) {
	env := newCLIEnv(t)
	counters := &testmodule.Counters{}
	registration := testmodule.Registration(counters, testmodule.Options{})

	output := requireOK(t, runWithModules(t, env, []modules.Registration{registration}, "", "modules", "list"))
	if output != testmodule.ID+"\tavailable\n" {
		t.Fatalf("unexpected list output: %q", output)
	}
	assertNoModuleFiles(t, env.project)

	output = requireOK(t, runWithModules(t, env, []modules.Registration{registration}, "", "modules", "disable", testmodule.ID))
	if output != testmodule.ID+"\tdisabled\n" {
		t.Fatalf("unexpected disable output: %q", output)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".brain", "modules.yaml")); err != nil {
		t.Fatalf("expected module config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".brain", "state", "module-grants.json")); !os.IsNotExist(err) {
		t.Fatalf("grant file should not exist before grant, got %v", err)
	}

	result := runWithModules(t, env, []modules.Registration{registration}, "", "modules", "enable", testmodule.ID)
	if result.err == nil || !strings.Contains(result.err.Error(), "brain modules grant "+testmodule.ID+" test.read") {
		t.Fatalf("expected exact grant command, got %v", result.err)
	}

	output = requireOK(t, runWithModules(t, env, []modules.Registration{registration}, "", "modules", "grant", testmodule.ID, "test.read"))
	if output != testmodule.ID+"\tdisabled\n" {
		t.Fatalf("unexpected grant output: %q", output)
	}
	output = requireOK(t, runWithModules(t, env, []modules.Registration{registration}, "", "modules", "enable", testmodule.ID))
	if output != testmodule.ID+"\tenabled\thealthy\n" {
		t.Fatalf("unexpected enable output: %q", output)
	}
	output = requireOK(t, runWithModules(t, env, []modules.Registration{registration}, "", "modules", "show", testmodule.ID))
	wantDetails := "id: dev.brain.test\n" +
		"name: Test Module\n" +
		"version: 1.0.0\n" +
		"state: enabled\n" +
		"desired enabled: true\n" +
		"config version: 1\n" +
		"capabilities: test.inspect\n" +
		"permissions: test.read\n" +
		"grants: test.read\n" +
		"health: healthy\n"
	if output != wantDetails {
		t.Fatalf("unexpected show output:\n%s", output)
	}
	output = requireOK(t, runWithModules(t, env, []modules.Registration{registration}, "", "modules", "health"))
	if output != testmodule.ID+"\tenabled\thealthy\n" {
		t.Fatalf("unexpected health output: %q", output)
	}

	jsonOutput := requireOK(t, runWithModules(t, env, []modules.Registration{registration}, "", "--json", "modules", "show", testmodule.ID))
	var report modules.Report
	if err := json.Unmarshal([]byte(jsonOutput), &report); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, jsonOutput)
	}
	if report.State != modules.StateEnabled || report.Health == nil || report.Health.Status != modules.HealthHealthy {
		t.Fatalf("unexpected JSON report: %#v", report)
	}

	output = requireOK(t, runWithModules(t, env, []modules.Registration{registration}, "", "modules", "revoke", testmodule.ID, "test.read"))
	if output != testmodule.ID+"\tblocked\tpermission_missing\n" {
		t.Fatalf("unexpected revoke output: %q", output)
	}
	output = requireOK(t, runWithModules(t, env, []modules.Registration{registration}, "", "modules", "disable", testmodule.ID))
	if output != testmodule.ID+"\tdisabled\n" {
		t.Fatalf("unexpected final disable output: %q", output)
	}
}

func TestCLIModulesUnknownConfigCanDisableAndRevoke(t *testing.T) {
	env := newCLIEnv(t)
	writeCLIFile(t, filepath.Join(env.project, ".brain", "modules.yaml"), "schema_version: 1\nmodules:\n  dev.brain.missing:\n    enabled: true\n    config_version: 1\n    config:\n      keep: value\n")
	writeCLIFile(t, filepath.Join(env.project, ".brain", "state", "module-grants.json"), "{\n  \"schema_version\": 1,\n  \"modules\": {\n    \"dev.brain.missing\": [\"old.permission\"]\n  }\n}\n")

	output := requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "modules", "disable", "dev.brain.missing"))
	if output != "dev.brain.missing\tunavailable\tmodule_unavailable\n" {
		t.Fatalf("unexpected disable output: %q", output)
	}
	output = requireOK(t, env.run(t, "", "--config", env.config, "--project", env.project, "modules", "revoke", "dev.brain.missing", "old.permission"))
	if output != "dev.brain.missing\tunavailable\tmodule_unavailable\n" {
		t.Fatalf("unexpected revoke output: %q", output)
	}
	config := readProjectFile(t, env.project, ".brain/modules.yaml")
	if !strings.Contains(config, "keep: value") || !strings.Contains(config, "enabled: false") {
		t.Fatalf("unknown config not preserved:\n%s", config)
	}
	grants := readProjectFile(t, env.project, ".brain/state/module-grants.json")
	if strings.Contains(grants, "old.permission") {
		t.Fatalf("stale grant not revoked:\n%s", grants)
	}
}

func TestCLIModulesRejectsInvalidRegistration(t *testing.T) {
	env := newCLIEnv(t)
	registration := testmodule.Registration(&testmodule.Counters{}, testmodule.Options{})
	registration.Descriptor.Version = "invalid"
	result := runWithModules(t, env, []modules.Registration{registration}, "", "modules", "list")
	if result.err == nil || !strings.Contains(result.err.Error(), "semantic version") {
		t.Fatalf("expected registration validation error, got %v", result.err)
	}
}

func runWithModules(t *testing.T, env *cliEnv, registrations []modules.Registration, stdin string, args ...string) cliResult {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command := newRootCommand(rootOptions{
		in:     strings.NewReader(stdin),
		out:    &stdout,
		errOut: &stderr,
		appLoad: func(configPath, projectPath string, jsonOutput bool, out, errOut io.Writer) (*app.App, error) {
			return app.New(configPath, projectPath, jsonOutput, app.Options{
				Stdout:              out,
				Stderr:              errOut,
				ModuleRegistrations: registrations,
			})
		},
	})
	command.SetArgs(append([]string{"--config", env.config, "--project", env.project}, args...))
	err := command.Execute()
	return cliResult{
		stdout: normalizeCLIOutput(stdout.String(), env.root),
		stderr: normalizeCLIOutput(stderr.String(), env.root),
		err:    err,
	}
}

func assertNoModuleFiles(t *testing.T, project string) {
	t.Helper()
	for _, rel := range []string{".brain/modules.yaml", ".brain/state/module-grants.json"} {
		if _, err := os.Stat(filepath.Join(project, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Fatalf("expected %s not to exist, got %v", rel, err)
		}
	}
}
