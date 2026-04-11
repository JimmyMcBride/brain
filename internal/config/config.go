package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultOutputMode = "human"
	defaultEmbedder   = "localhash"
	defaultModel      = "hash-v1"
)

type Config struct {
	EmbeddingProvider string `yaml:"embedding_provider" json:"embedding_provider"`
	EmbeddingModel    string `yaml:"embedding_model" json:"embedding_model"`
	OutputMode        string `yaml:"output_mode" json:"output_mode"`
}

type Paths struct {
	ConfigFile      string `json:"config_file"`
	ConfigDir       string `json:"config_dir"`
	AppDataDir      string `json:"app_data_dir"`
	ProjectDir      string `json:"project_dir,omitempty"`
	BrainDir        string `json:"brain_dir,omitempty"`
	StateDir        string `json:"state_dir,omitempty"`
	BackupDir       string `json:"backup_dir,omitempty"`
	UpdateBackupDir string `json:"update_backup_dir"`
	LogFile         string `json:"log_file,omitempty"`
	DBFile          string `json:"db_file,omitempty"`
	IndexDir        string `json:"index_dir,omitempty"`
}

func LoadOrCreate(configPath string) (*Config, Paths, error) {
	paths, err := resolvePaths(configPath)
	if err != nil {
		return nil, Paths{}, err
	}
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		return nil, Paths{}, fmt.Errorf("create config dir: %w", err)
	}
	if err := os.MkdirAll(paths.AppDataDir, 0o755); err != nil {
		return nil, Paths{}, fmt.Errorf("create app data dir: %w", err)
	}
	if err := os.MkdirAll(paths.UpdateBackupDir, 0o755); err != nil {
		return nil, Paths{}, fmt.Errorf("create update backup dir: %w", err)
	}

	cfg := Default()
	if _, err := os.Stat(paths.ConfigFile); errors.Is(err, os.ErrNotExist) {
		if err := Save(cfg, paths.ConfigFile); err != nil {
			return nil, Paths{}, err
		}
	} else if err == nil {
		raw, readErr := os.ReadFile(paths.ConfigFile)
		if readErr != nil {
			return nil, Paths{}, fmt.Errorf("read config: %w", readErr)
		}
		if len(strings.TrimSpace(string(raw))) > 0 {
			if unmarshalErr := yaml.Unmarshal(raw, cfg); unmarshalErr != nil {
				return nil, Paths{}, fmt.Errorf("parse config: %w", unmarshalErr)
			}
		}
	} else {
		return nil, Paths{}, fmt.Errorf("stat config: %w", err)
	}

	applyEnvOverrides(cfg)
	cfg.normalize()
	if err := Save(cfg, paths.ConfigFile); err != nil {
		return nil, Paths{}, err
	}
	return cfg, paths, nil
}

func Save(cfg *Config, configFile string) error {
	if cfg == nil {
		return errors.New("nil config")
	}
	cfg.normalize()
	if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
		return fmt.Errorf("create config parent: %w", err)
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(configFile, out, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func Default() *Config {
	return &Config{
		EmbeddingProvider: defaultEmbedder,
		EmbeddingModel:    defaultModel,
		OutputMode:        defaultOutputMode,
	}
}

func ProjectPaths(global Paths, projectDir string) Paths {
	projectDir = filepath.Clean(expandHome(projectDir))
	brainDir := filepath.Join(projectDir, ".brain")
	stateDir := filepath.Join(brainDir, "state")
	return Paths{
		ConfigFile:      global.ConfigFile,
		ConfigDir:       global.ConfigDir,
		AppDataDir:      global.AppDataDir,
		ProjectDir:      projectDir,
		BrainDir:        brainDir,
		StateDir:        stateDir,
		BackupDir:       filepath.Join(stateDir, "backups"),
		UpdateBackupDir: global.UpdateBackupDir,
		LogFile:         filepath.Join(stateDir, "history.jsonl"),
		DBFile:          filepath.Join(stateDir, "brain.sqlite3"),
		IndexDir:        filepath.Join(stateDir, "index"),
	}
}

func EnsureProjectPaths(paths Paths) error {
	for _, dir := range []string{paths.BrainDir, paths.StateDir, paths.BackupDir, paths.IndexDir} {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}

func resolvePaths(configPath string) (Paths, error) {
	if configPath == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return Paths{}, fmt.Errorf("resolve user config dir: %w", err)
		}
		configPath = filepath.Join(dir, "brain", "config.yaml")
	}
	configPath = expandHome(configPath)
	home, _ := os.UserHomeDir()
	appDataDir := filepath.Join(userDataDirFor(runtime.GOOS, home), "brain")
	return Paths{
		ConfigFile:      configPath,
		ConfigDir:       filepath.Dir(configPath),
		AppDataDir:      appDataDir,
		UpdateBackupDir: filepath.Join(appDataDir, "updates", "backups"),
	}, nil
}

func userDataDirFor(goos, home string) string {
	if goos == "windows" {
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return expandHome(v)
		}
		return filepath.Join(home, "AppData", "Local")
	}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return expandHome(v)
	}
	return filepath.Join(home, ".local", "share")
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("BRAIN_EMBEDDING_PROVIDER"); v != "" {
		cfg.EmbeddingProvider = v
	}
	if v := os.Getenv("BRAIN_EMBEDDING_MODEL"); v != "" {
		cfg.EmbeddingModel = v
	}
	if v := os.Getenv("BRAIN_OUTPUT_MODE"); v != "" {
		cfg.OutputMode = v
	}
}

func (c *Config) normalize() {
	if c == nil {
		return
	}
	if c.EmbeddingProvider == "" {
		c.EmbeddingProvider = defaultEmbedder
	}
	if c.EmbeddingModel == "" {
		c.EmbeddingModel = defaultModel
	}
	if c.OutputMode == "" {
		c.OutputMode = defaultOutputMode
	}
}

func expandHome(p string) string {
	if p == "" || p[0] != '~' {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	return filepath.Join(home, strings.TrimPrefix(p, "~/"))
}
