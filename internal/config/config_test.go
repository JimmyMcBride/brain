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
	if cfg.DataPath != filepath.Join(root, "xdg-data", "brain") {
		t.Fatalf("unexpected data path: %s", cfg.DataPath)
	}
	if cfg.VaultPath != filepath.Join(root, "home", "Documents", "brain") {
		t.Fatalf("unexpected vault path: %s", cfg.VaultPath)
	}
	if _, err := os.Stat(paths.ConfigFile); err != nil {
		t.Fatalf("expected config file: %v", err)
	}
}

func TestLoadOrCreateAppliesEnvOverrides(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")
	if err := os.WriteFile(configPath, []byte("vault_path: /tmp/vault\noutput_mode: human\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BRAIN_VAULT_PATH", filepath.Join(root, "override-vault"))
	t.Setenv("BRAIN_DATA_PATH", filepath.Join(root, "override-data"))
	t.Setenv("BRAIN_EMBEDDING_PROVIDER", "none")
	t.Setenv("BRAIN_EMBEDDING_MODEL", "none")
	t.Setenv("BRAIN_OUTPUT_MODE", "json")

	cfg, paths, err := LoadOrCreate(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.VaultPath != filepath.Join(root, "override-vault") {
		t.Fatalf("unexpected vault path: %s", cfg.VaultPath)
	}
	if cfg.DataPath != filepath.Join(root, "override-data") {
		t.Fatalf("unexpected data path: %s", cfg.DataPath)
	}
	if cfg.EmbeddingProvider != "none" || cfg.OutputMode != "json" {
		t.Fatalf("unexpected config overrides: %+v", cfg)
	}
	if paths.DBFile != filepath.Join(root, "override-data", "brain.sqlite3") {
		t.Fatalf("unexpected db file: %s", paths.DBFile)
	}
}
