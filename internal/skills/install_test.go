package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveTargetsGlobalAndLocal(t *testing.T) {
	repoRoot := t.TempDir()
	for _, skill := range []string{"brain", "googleworkspace-cli"} {
		if err := os.MkdirAll(filepath.Join(repoRoot, "skills", skill), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repoRoot, "skills", skill, "SKILL.md"), []byte("skill"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	installer := NewInstaller("/home/tester")
	targets, err := installer.ResolveTargets(InstallRequest{
		Scope:      ScopeBoth,
		Agents:     []string{"codex", "zed"},
		ProjectDir: "/tmp/project",
		Mode:       ModeCopy,
		RepoRoot:   repoRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"/home/tester/.codex/skills/brain":               true,
		"/home/tester/.codex/skills/googleworkspace-cli": true,
		"/home/tester/.zed/skills/brain":                 true,
		"/home/tester/.zed/skills/googleworkspace-cli":   true,
		"/tmp/project/.codex/skills/brain":               true,
		"/tmp/project/.codex/skills/googleworkspace-cli": true,
		"/tmp/project/.zed/skills/brain":                 true,
		"/tmp/project/.zed/skills/googleworkspace-cli":   true,
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

func TestInstallFiltersSpecificSkills(t *testing.T) {
	repoRoot := t.TempDir()
	for _, skill := range []string{"brain", "googleworkspace-cli"} {
		if err := os.MkdirAll(filepath.Join(repoRoot, "skills", skill), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repoRoot, "skills", skill, "SKILL.md"), []byte("skill"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	home := t.TempDir()
	installer := NewInstaller(home)
	results, err := installer.Install(InstallRequest{
		Mode:     ModeCopy,
		Scope:    ScopeGlobal,
		Agents:   []string{"codex"},
		Skills:   []string{"googleworkspace-cli"},
		RepoRoot: repoRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Skill != "googleworkspace-cli" {
		t.Fatalf("unexpected install results: %+v", results)
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "skills", "googleworkspace-cli", "SKILL.md")); err != nil {
		t.Fatalf("expected googleworkspace-cli install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "skills", "brain")); !os.IsNotExist(err) {
		t.Fatalf("expected brain skill to be absent, got err=%v", err)
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

func TestInstallRejectsUnknownSkill(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "skills", "brain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "skills", "brain", "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	installer := NewInstaller(t.TempDir())
	_, err := installer.ResolveTargets(InstallRequest{
		Mode:     ModeCopy,
		Scope:    ScopeGlobal,
		Agents:   []string{"codex"},
		Skills:   []string{"missing"},
		RepoRoot: repoRoot,
	})
	if err == nil || !strings.Contains(err.Error(), "unknown skill(s): missing") {
		t.Fatalf("expected unknown skill error, got %v", err)
	}
}
