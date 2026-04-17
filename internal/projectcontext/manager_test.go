package projectcontext

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAgentInstructionFile(t *testing.T) {
	project := "/tmp/project"
	if got := filepath.ToSlash(agentInstructionFile(project, "codex")); got != "/tmp/project/.codex/AGENTS.md" {
		t.Fatalf("unexpected codex path: %s", got)
	}
	if got := filepath.ToSlash(agentInstructionFile(project, "claude")); got != "/tmp/project/.claude/CLAUDE.md" {
		t.Fatalf("unexpected claude path: %s", got)
	}
	if got := filepath.ToSlash(agentInstructionFile(project, "copilot")); got != "/tmp/project/.github/copilot-instructions.md" {
		t.Fatalf("unexpected copilot path: %s", got)
	}
}

func TestResolveAgentsRequiresExplicitRequest(t *testing.T) {
	manager := New(t.TempDir())
	got, err := manager.resolveAgents(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no implicit agent selection, got=%v", got)
	}
	got, err = manager.resolveAgents([]string{"codex", "openclaw"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"codex", "openclaw"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected resolved agents: got=%v want=%v", got, want)
	}
}

func TestResolveAgentsRejectsUnsupportedAgent(t *testing.T) {
	manager := New(t.TempDir())
	if _, err := manager.resolveAgents([]string{"codx"}); err == nil {
		t.Fatal("expected unsupported agent error")
	}
}

func TestDiscoverAgentIntegrationTargetsDefaultToExistingFiles(t *testing.T) {
	project := t.TempDir()
	mustWriteFile(t, filepath.Join(project, ".codex", "AGENTS.md"), "# Codex\n")
	mustWriteFile(t, filepath.Join(project, ".pi", "AGENTS.md"), "# Pi\n")

	targets, err := discoverAgentIntegrationTargets(project, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("unexpected target count: %d", len(targets))
	}
	agents := []string{targets[0].Agent, targets[1].Agent}
	if strings.Join(agents, ",") != "codex,pi" {
		t.Fatalf("unexpected targets: %+v", targets)
	}
}

func TestMergeDocumentPreservesLocalNotes(t *testing.T) {
	spec := fileSpec{
		Path:      "AGENTS.md",
		Kind:      "contract",
		Title:     "Project Agent Contract",
		BlockID:   "agents-contract",
		Body:      "fresh body",
		Style:     "markdown",
		LocalNote: true,
	}
	existing := "# Project Agent Contract\n\n<!-- brain:begin agents-contract -->\nstale\n<!-- brain:end agents-contract -->\n\n## Local Notes\n\nKeep this.\n"
	merged, preserved, action, err := mergeDocument(existing, spec, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !preserved {
		t.Fatal("expected local notes to be preserved")
	}
	if action != "updated" {
		t.Fatalf("unexpected action: %s", action)
	}
	if !strings.Contains(merged, "fresh body") || !strings.Contains(merged, "Keep this.") {
		t.Fatalf("merged document missing expected content:\n%s", merged)
	}
}

func TestMergeDocumentRequiresForceForUnmanagedFile(t *testing.T) {
	spec := fileSpec{
		Path:      "AGENTS.md",
		Kind:      "contract",
		Title:     "Project Agent Contract",
		BlockID:   "agents-contract",
		Body:      "fresh body",
		Style:     "markdown",
		LocalNote: true,
	}
	if _, _, _, err := mergeDocument("manual file", spec, false, false, false); err == nil {
		t.Fatal("expected unmanaged file error")
	}
}

func TestMergeDocumentForceAdoptsUnmanagedFile(t *testing.T) {
	spec := fileSpec{
		Path:      "AGENTS.md",
		Kind:      "contract",
		Title:     "Project Agent Contract",
		BlockID:   "agents-contract",
		Body:      "fresh body",
		Style:     "markdown",
		LocalNote: true,
	}
	merged, preserved, action, err := mergeDocument("manual file", spec, true, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !preserved || action != "adopted" {
		t.Fatalf("unexpected force adoption result: preserved=%t action=%s", preserved, action)
	}
	if !strings.Contains(merged, "manual file") || !strings.Contains(merged, "fresh body") {
		t.Fatalf("adopted document missing content:\n%s", merged)
	}
}

func TestMergeDocumentForceRefreshUnmanagedFileWithoutAdoptAction(t *testing.T) {
	spec := fileSpec{
		Path:      "AGENTS.md",
		Kind:      "contract",
		Title:     "Project Agent Contract",
		BlockID:   "agents-contract",
		Body:      "fresh body",
		Style:     "markdown",
		LocalNote: true,
	}
	_, preserved, action, err := mergeDocument("manual file", spec, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !preserved || action != "updated" {
		t.Fatalf("unexpected force refresh result: preserved=%t action=%s", preserved, action)
	}
}

func TestScanRepo(t *testing.T) {
	project := t.TempDir()
	mustWriteFile(t, filepath.Join(project, "go.mod"), "module example.com/test\n\ngo 1.26\n")
	mustWriteFile(t, filepath.Join(project, "README.md"), "# test\n")
	mustWriteFile(t, filepath.Join(project, "docs", "usage.md"), "# usage\n")
	mustWriteFile(t, filepath.Join(project, ".github", "workflows", "ci.yml"), "name: ci\n")
	mustWriteFile(t, filepath.Join(project, "internal", "app", "app.go"), "package app\n")
	mustWriteFile(t, filepath.Join(project, "internal", "app", "app_test.go"), "package app\n")

	snapshot := scanRepo(context.Background(), project)
	if snapshot.ProjectName != filepath.Base(project) {
		t.Fatalf("unexpected project name: %s", snapshot.ProjectName)
	}
	if snapshot.PrimaryRuntime != "go" {
		t.Fatalf("unexpected runtime: %s", snapshot.PrimaryRuntime)
	}
	if snapshot.GoModule != "example.com/test" {
		t.Fatalf("unexpected go module: %s", snapshot.GoModule)
	}
	if snapshot.TestFiles != 1 {
		t.Fatalf("unexpected test file count: %d", snapshot.TestFiles)
	}
	if !contains(snapshot.ManifestFiles, "go.mod") || !contains(snapshot.DocFiles, "README.md") || !contains(snapshot.CIFiles, ".github/workflows/ci.yml") {
		t.Fatalf("snapshot missing expected files: %+v", snapshot)
	}
}

func TestInstallDryRunDoesNotWrite(t *testing.T) {
	project := t.TempDir()
	manager := New(t.TempDir())
	results, err := manager.Install(context.Background(), Request{
		ProjectDir: project,
		Agents:     []string{"codex"},
		DryRun:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected planned results")
	}
	if _, err := os.Stat(filepath.Join(project, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run to avoid writes, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(project, ".codex", "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run to avoid agent writes, got err=%v", err)
	}
}

func TestBundleSpecsWritesGitIgnoreFirst(t *testing.T) {
	project := t.TempDir()
	snapshot := Snapshot{ProjectDir: project}

	specs := bundleSpecs(snapshot, "version: 1\n")
	if len(specs) == 0 {
		t.Fatal("expected managed specs")
	}
	if got := filepath.Base(specs[0].Path); got != ".gitignore" {
		t.Fatalf("expected .gitignore to be synced first, got=%s", got)
	}
}

func TestLoadReturnsDeterministicSourcesByLevel(t *testing.T) {
	project := t.TempDir()
	mustWriteFile(t, filepath.Join(project, "go.mod"), "module example.com/test\n\ngo 1.26\n")
	manager := New(t.TempDir())
	if _, err := manager.Install(context.Background(), Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	level0, err := manager.Load(LoadRequest{ProjectDir: project, Level: 0})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"AGENTS.md (summary)", ".brain/context/current-state.md"}; !reflect.DeepEqual(level0.Sources, want) {
		t.Fatalf("unexpected level 0 sources: got=%v want=%v", level0.Sources, want)
	}
	if strings.Contains(level0.Content, ".brain/context/overview.md") {
		t.Fatalf("expected level 0 to omit overview:\n%s", level0.Content)
	}

	level1, err := manager.Load(LoadRequest{ProjectDir: project, Level: 1})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"AGENTS.md (summary)", ".brain/context/current-state.md", ".brain/context/overview.md", ".brain/context/workflows.md"}; !reflect.DeepEqual(level1.Sources, want) {
		t.Fatalf("unexpected level 1 sources: got=%v want=%v", level1.Sources, want)
	}

	level2, err := manager.Load(LoadRequest{ProjectDir: project, Level: 2})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"AGENTS.md", ".brain/context/overview.md", ".brain/context/architecture.md", ".brain/context/standards.md", ".brain/context/workflows.md", ".brain/context/memory-policy.md", ".brain/context/current-state.md"}; !reflect.DeepEqual(level2.Sources, want) {
		t.Fatalf("unexpected level 2 sources: got=%v want=%v", level2.Sources, want)
	}
	if !strings.Contains(level2.Content, "## Source: .brain/context/architecture.md") || !strings.Contains(level2.Content, "## Source: .brain/context/memory-policy.md") {
		t.Fatalf("expected full static bundle in level 2:\n%s", level2.Content)
	}
}

func TestLoadRejectsUnsupportedLevel(t *testing.T) {
	manager := New(t.TempDir())
	if _, err := manager.Load(LoadRequest{ProjectDir: t.TempDir(), Level: 9}); err == nil {
		t.Fatal("expected unsupported level error")
	}
}

func TestAdoptMarksUnmanagedFilesAsAdopted(t *testing.T) {
	project := t.TempDir()
	mustWriteFile(t, filepath.Join(project, "AGENTS.md"), "manual contract\n")

	manager := New(t.TempDir())
	results, err := manager.Adopt(context.Background(), Request{
		ProjectDir: project,
	})
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, result := range results {
		if result.Path == "AGENTS.md" {
			found = true
			if result.Action != "adopted" || !result.PreservedUserContent {
				t.Fatalf("unexpected AGENTS.md adoption result: %+v", result)
			}
		}
	}
	if !found {
		t.Fatal("expected AGENTS.md result")
	}
}

func TestAdoptAppendsBrainSectionToExistingAgentFile(t *testing.T) {
	project := t.TempDir()
	mustWriteFile(t, filepath.Join(project, ".claude", "CLAUDE.md"), "# Existing Claude Notes\n")
	mustWriteFile(t, filepath.Join(project, ".pi", "AGENTS.md"), "# Existing Pi Notes\n")

	manager := New(t.TempDir())
	results, err := manager.Adopt(context.Background(), Request{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, result := range results {
		if result.Path == ".claude/CLAUDE.md" {
			found = true
			if result.Action != "adopted" || !result.PreservedUserContent {
				t.Fatalf("unexpected agent adoption result: %+v", result)
			}
		}
	}
	if !found {
		t.Fatal("expected claude integration result")
	}
	var foundPi bool
	for _, result := range results {
		if result.Path == ".pi/AGENTS.md" {
			foundPi = true
			if result.Action != "adopted" || !result.PreservedUserContent {
				t.Fatalf("unexpected pi adoption result: %+v", result)
			}
		}
	}
	if !foundPi {
		t.Fatal("expected pi integration result")
	}

	body, err := os.ReadFile(filepath.Join(project, ".claude", "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "# Existing Claude Notes") {
		t.Fatalf("expected existing content to remain:\n%s", text)
	}
	if !strings.Contains(text, "## Brain") || !strings.Contains(text, managedBegin("agent-integration-claude")) {
		t.Fatalf("expected Brain integration block:\n%s", text)
	}
	if strings.Contains(text, "canonical project contract") {
		t.Fatalf("unexpected canonical wording:\n%s", text)
	}
}

func TestAdoptWithExplicitAgentCreatesMissingAgentFile(t *testing.T) {
	project := t.TempDir()

	manager := New(t.TempDir())
	results, err := manager.Adopt(context.Background(), Request{
		ProjectDir: project,
		Agents:     []string{"codex"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, result := range results {
		if result.Path == ".codex/AGENTS.md" {
			found = true
			if result.Action != "created" {
				t.Fatalf("unexpected create result: %+v", result)
			}
		}
	}
	if !found {
		t.Fatal("expected created codex agent file")
	}

	body, err := os.ReadFile(filepath.Join(project, ".codex", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "canonical project contract") {
		t.Fatalf("unexpected canonical wording:\n%s", text)
	}
	if !strings.Contains(text, "## Brain") || !strings.Contains(text, managedBegin("agent-integration-codex")) {
		t.Fatalf("unexpected created agent file:\n%s", text)
	}
}

func TestRefreshUpdatesExistingManagedAgentBlockOnly(t *testing.T) {
	project := t.TempDir()
	existing := "# Existing Notes\n\n## Brain\n\n<!-- brain:begin agent-integration-codex -->\nstale\n<!-- brain:end agent-integration-codex -->\n"
	mustWriteFile(t, filepath.Join(project, ".codex", "AGENTS.md"), existing)

	manager := New(t.TempDir())
	results, err := manager.Refresh(context.Background(), Request{
		ProjectDir: project,
		Agents:     []string{"codex"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, result := range results {
		if result.Path == ".codex/AGENTS.md" {
			found = true
			if result.Action != "updated" || !result.PreservedUserContent {
				t.Fatalf("unexpected refresh result: %+v", result)
			}
		}
	}
	if !found {
		t.Fatal("expected codex refresh result")
	}

	body, err := os.ReadFile(filepath.Join(project, ".codex", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "# Existing Notes") {
		t.Fatalf("expected surrounding content to remain:\n%s", text)
	}
	if !strings.Contains(text, "Brain-managed project context") {
		t.Fatalf("expected refreshed block:\n%s", text)
	}
	if strings.Contains(text, "stale") {
		t.Fatalf("expected stale block content to be replaced:\n%s", text)
	}
}

func TestRefreshMigratesLegacyWrapperToAgentIntegration(t *testing.T) {
	project := t.TempDir()
	legacy := "# Codex Wrapper\n\n<!-- brain:begin agent-wrapper-codex -->\nThis `codex` wrapper delegates to the root project contract.\n<!-- brain:end agent-wrapper-codex -->\n"
	mustWriteFile(t, filepath.Join(project, ".codex", "AGENTS.md"), legacy)

	manager := New(t.TempDir())
	results, err := manager.Refresh(context.Background(), Request{
		ProjectDir: project,
		Agents:     []string{"codex"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, result := range results {
		if result.Path == ".codex/AGENTS.md" {
			found = true
			if result.Action != "updated" {
				t.Fatalf("unexpected legacy migration result: %+v", result)
			}
		}
	}
	if !found {
		t.Fatal("expected codex migration result")
	}

	body, err := os.ReadFile(filepath.Join(project, ".codex", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "canonical project contract") || strings.Contains(text, "agent-wrapper-codex") {
		t.Fatalf("expected legacy wrapper content to be removed:\n%s", text)
	}
	if !strings.Contains(text, managedBegin("agent-integration-codex")) {
		t.Fatalf("expected migrated integration block:\n%s", text)
	}
}

func TestRefreshSkipsUnmanagedAgentFileAndMissingTargets(t *testing.T) {
	project := t.TempDir()
	mustWriteFile(t, filepath.Join(project, ".openclaw", "AGENTS.md"), "# Manual OpenClaw\n")

	manager := New(t.TempDir())
	results, err := manager.Refresh(context.Background(), Request{
		ProjectDir: project,
		Agents:     []string{"codex", "openclaw"},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, result := range results {
		if result.Path == ".openclaw/AGENTS.md" || result.Path == ".codex/AGENTS.md" {
			t.Fatalf("expected unmanaged or missing agent files to be skipped, got %+v", result)
		}
	}

	body, err := os.ReadFile(filepath.Join(project, ".openclaw", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "# Manual OpenClaw\n" {
		t.Fatalf("expected unmanaged agent file to remain unchanged:\n%s", string(body))
	}
	if _, err := os.Stat(filepath.Join(project, ".codex", "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected missing codex agent file to remain missing, got err=%v", err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
