package projectcontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brain/internal/skills"
)

func TestWrapperFile(t *testing.T) {
	project := "/tmp/project"
	if got := filepath.ToSlash(wrapperFile(project, "codex")); got != "/tmp/project/.codex/AGENTS.md" {
		t.Fatalf("unexpected codex wrapper path: %s", got)
	}
	if got := filepath.ToSlash(wrapperFile(project, "claude")); got != "/tmp/project/.claude/CLAUDE.md" {
		t.Fatalf("unexpected claude wrapper path: %s", got)
	}
}

func TestResolveAgentsFromInstalledSkills(t *testing.T) {
	home := t.TempDir()
	for _, agent := range []string{"codex", "openclaw"} {
		path := filepath.Join(skills.GlobalSkillRoot(home, agent), "brain")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("skill"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	manager := New(home)
	got := manager.resolveAgents(nil)
	want := []string{"codex", "openclaw"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected resolved agents: got=%v want=%v", got, want)
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
	merged, preserved, action, err := mergeDocument(existing, spec, false, false)
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
	if _, _, _, err := mergeDocument("manual file", spec, false, false); err == nil {
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
	merged, preserved, action, err := mergeDocument("manual file", spec, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !preserved || action != "updated" {
		t.Fatalf("unexpected force adoption result: preserved=%t action=%s", preserved, action)
	}
	if !strings.Contains(merged, "manual file") || !strings.Contains(merged, "fresh body") {
		t.Fatalf("adopted document missing content:\n%s", merged)
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
