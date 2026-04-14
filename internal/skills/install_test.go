package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveTargetsGlobalAndLocal(t *testing.T) {
	installer := NewInstaller("/home/tester")
	targets, err := installer.ResolveTargets(InstallRequest{
		Scope:      ScopeBoth,
		Agents:     []string{"codex", "copilot", "pi", "zed"},
		ProjectDir: "/tmp/project",
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

func TestInstallCopiesSkillBundleAndWritesManifest(t *testing.T) {
	bundleHash := registerTestBundle(t)

	home := t.TempDir()
	installer := NewInstaller(home)
	results, err := installer.Install(InstallRequest{
		Scope:  ScopeGlobal,
		Agents: []string{"codex"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Method != "copy" {
		t.Fatalf("expected copy install method, got %+v", results[0])
	}

	skillDir := filepath.Join(home, ".codex", "skills", "brain")
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		t.Fatalf("expected skill file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(skillDir, "agents", "openai.yaml")); err != nil {
		t.Fatalf("expected metadata file: %v", err)
	}
	manifest, err := readManifest(skillDir)
	if err != nil {
		t.Fatalf("expected manifest: %v", err)
	}
	if manifest.BundleHash != bundleHash {
		t.Fatalf("unexpected bundle hash: %+v", manifest)
	}
	if manifest.Agent != "codex" || manifest.Scope != string(ScopeGlobal) {
		t.Fatalf("unexpected manifest routing: %+v", manifest)
	}
}

func TestInspectMarksLegacyInstallWithoutManifestForRepair(t *testing.T) {
	registerTestBundle(t)

	home := t.TempDir()
	skillDir := filepath.Join(home, ".codex", "skills", "brain")
	if err := os.MkdirAll(filepath.Join(skillDir, "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "agents", "openai.yaml"), []byte("name: brain"), 0o644); err != nil {
		t.Fatal(err)
	}

	installer := NewInstaller(home)
	statuses, err := installer.Inspect(InstallRequest{
		Scope:  ScopeGlobal,
		Agents: []string{"codex"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || !statuses[0].Installed || !statuses[0].NeedsRepair || statuses[0].Reason != "legacy_install" {
		t.Fatalf("expected legacy install repair state, got %+v", statuses)
	}
}

func TestInstallReplacesLegacySymlink(t *testing.T) {
	registerTestBundle(t)

	home := t.TempDir()
	legacySource := filepath.Join(t.TempDir(), "legacy-brain")
	if err := os.MkdirAll(legacySource, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacySource, "SKILL.md"), []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(home, ".openclaw", "skills", "brain")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(legacySource, target); err != nil {
		t.Fatal(err)
	}

	installer := NewInstaller(home)
	if _, err := installer.Install(InstallRequest{
		Scope:  ScopeGlobal,
		Agents: []string{"openclaw"},
	}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("expected legacy symlink to be replaced with a copied directory")
	}
	if _, err := readManifest(target); err != nil {
		t.Fatalf("expected manifest after replacement: %v", err)
	}
}

func registerTestBundle(t *testing.T) string {
	t.Helper()

	bundleDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(bundleDir, "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "agents", "openai.yaml"), []byte("name: brain"), 0o644); err != nil {
		t.Fatal(err)
	}

	RegisterBundle(os.DirFS(bundleDir))
	t.Cleanup(func() {
		RegisterBundle(nil)
	})

	bundle, err := loadBundle()
	if err != nil {
		t.Fatal(err)
	}
	return bundle.Hash
}
