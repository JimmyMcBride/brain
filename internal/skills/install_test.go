package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveTargetsGlobalAndLocal(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "skills", "brain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "skills", "brain", "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	installer := NewInstaller("/home/tester")
	targets, err := installer.ResolveTargets(InstallRequest{
		Scope:      ScopeBoth,
		Agents:     []string{"codex", "copilot", "pi", "zed"},
		ProjectDir: "/tmp/project",
		Mode:       ModeCopy,
		RepoRoot:   repoRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		filepath.Join("/home/tester", ".codex", "skills", "brain"):       true,
		filepath.Join("/home/tester", ".copilot", "skills", "brain"):     true,
		filepath.Join("/home/tester", ".pi", "agent", "skills", "brain"): true,
		filepath.Join("/home/tester", ".zed", "skills", "brain"):         true,
		filepath.Join("/tmp/project", ".codex", "skills", "brain"):       true,
		filepath.Join("/tmp/project", ".github", "skills", "brain"):      true,
		filepath.Join("/tmp/project", ".pi", "skills", "brain"):          true,
		filepath.Join("/tmp/project", ".zed", "skills", "brain"):         true,
	}
	if len(targets) != len(want) {
		t.Fatalf("expected %d targets, got %d", len(want), len(targets))
	}
	for _, target := range targets {
		if !want[target.Path] {
			t.Fatalf("unexpected target: %+v", target)
		}
	}
}

func TestNormalizeAgentsSupportsAliases(t *testing.T) {
	got := normalizeAgents([]string{"copilot", "github-copilot", "pi.dev", "pi", "  "})
	want := []string{"copilot", "pi"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected normalized agents: got=%v want=%v", got, want)
	}
}

func TestInstallCopiesSkillBundle(t *testing.T) {
	repoRoot := t.TempDir()
	source := filepath.Join(repoRoot, "skills", "brain", "agents")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "skills", "brain", "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "openai.yaml"), []byte("name: brain"), 0o644); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	installer := NewInstaller(home)
	results, err := installer.Install(InstallRequest{
		Mode:     ModeCopy,
		Scope:    ScopeGlobal,
		Agents:   []string{"codex"},
		RepoRoot: repoRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	skillFile := filepath.Join(home, ".codex", "skills", "brain", "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		t.Fatalf("expected skill file: %v", err)
	}
	metaFile := filepath.Join(home, ".codex", "skills", "brain", "agents", "openai.yaml")
	if _, err := os.Stat(metaFile); err != nil {
		t.Fatalf("expected metadata file: %v", err)
	}
}

func TestResolveTargetsRequiresBrainSkillSource(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}

	installer := NewInstaller(t.TempDir())
	_, err := installer.ResolveTargets(InstallRequest{
		Mode:     ModeCopy,
		Scope:    ScopeGlobal,
		Agents:   []string{"codex"},
		RepoRoot: repoRoot,
	})
	if err == nil || !strings.Contains(err.Error(), "skill source") {
		t.Fatalf("expected missing brain skill source error, got %v", err)
	}
}

func TestInstallForcesCopyForOpenClaw(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "skills", "brain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "skills", "brain", "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	installer := NewInstaller(home)
	results, err := installer.Install(InstallRequest{
		Mode:     ModeSymlink,
		Scope:    ScopeGlobal,
		Agents:   []string{"openclaw"},
		RepoRoot: repoRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Method != string(ModeCopy) {
		t.Fatalf("expected forced copy result, got %+v", results)
	}
	info, err := os.Lstat(filepath.Join(home, ".openclaw", "skills", "brain"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("expected OpenClaw install to be a directory copy, not a symlink")
	}
}
