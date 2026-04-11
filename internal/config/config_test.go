package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateUsesXDGDefaults(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))

	cfg, paths, err := LoadOrCreate("")
	if err != nil {
		t.Fatal(err)
	}
	if paths.ConfigFile != filepath.Join(root, "xdg-config", "brain", "config.yaml") {
		t.Fatalf("unexpected config path: %s", paths.ConfigFile)
	}
	if paths.AppDataDir != filepath.Join(root, "xdg-data", "brain") {
		t.Fatalf("unexpected app data dir: %s", paths.AppDataDir)
	}
	if cfg.EmbeddingProvider != "localhash" || cfg.EmbeddingModel != "hash-v1" || cfg.OutputMode != "human" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if _, err := os.Stat(paths.ConfigFile); err != nil {
		t.Fatalf("expected config file: %v", err)
	}
	if _, err := os.Stat(paths.UpdateBackupDir); err != nil {
		t.Fatalf("expected update backup dir: %v", err)
	}
}

func TestLoadOrCreateAppliesEnvOverrides(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")
	if err := os.WriteFile(configPath, []byte("output_mode: human\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BRAIN_EMBEDDING_PROVIDER", "none")
	t.Setenv("BRAIN_EMBEDDING_MODEL", "none")
	t.Setenv("BRAIN_OUTPUT_MODE", "json")
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))

	cfg, paths, err := LoadOrCreate(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EmbeddingProvider != "none" || cfg.EmbeddingModel != "none" || cfg.OutputMode != "json" {
		t.Fatalf("unexpected config overrides: %+v", cfg)
	}
	if paths.UpdateBackupDir != filepath.Join(root, "xdg-data", "brain", "updates", "backups") {
		t.Fatalf("unexpected update backup dir: %s", paths.UpdateBackupDir)
	}
}

func TestProjectPathsUsesProjectLocalState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	global := Paths{
		ConfigFile:      filepath.Join(root, "config.yaml"),
		ConfigDir:       root,
		AppDataDir:      filepath.Join(root, "xdg-data", "brain"),
		UpdateBackupDir: filepath.Join(root, "xdg-data", "brain", "updates", "backups"),
	}
	projectRoot := filepath.Join(root, "repo")
	paths := ProjectPaths(global, projectRoot)
	if paths.BrainDir != filepath.Join(projectRoot, ".brain") {
		t.Fatalf("unexpected brain dir: %s", paths.BrainDir)
	}
	if paths.StateDir != filepath.Join(projectRoot, ".brain", "state") {
		t.Fatalf("unexpected state dir: %s", paths.StateDir)
	}
	if paths.DBFile != filepath.Join(projectRoot, ".brain", "state", "brain.sqlite3") {
		t.Fatalf("unexpected db file: %s", paths.DBFile)
	}
	if paths.UpdateBackupDir != global.UpdateBackupDir {
		t.Fatalf("unexpected update backup dir: %s", paths.UpdateBackupDir)
	}
}

func TestUserDataDirForWindowsUsesLocalAppData(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LOCALAPPDATA", filepath.Join(root, "LocalAppData"))

	got := userDataDirFor("windows", filepath.Join(root, "home"))
	want := filepath.Join(root, "LocalAppData")
	if got != want {
		t.Fatalf("unexpected windows app data dir: %s", got)
	}
}

func TestUserDataDirForWindowsFallsBackToUserProfile(t *testing.T) {
	root := t.TempDir()
	got := userDataDirFor("windows", filepath.Join(root, "home"))
	want := filepath.Join(root, "home", "AppData", "Local")
	if got != want {
		t.Fatalf("unexpected windows fallback app data dir: %s", got)
	}
}
