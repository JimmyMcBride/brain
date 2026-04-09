package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultOutputMode = "human"
	defaultEmbedder   = "localhash"
	defaultModel      = "hash-v1"
)

type Config struct {
	VaultPath         string `yaml:"vault_path" json:"vault_path"`
	DataPath          string `yaml:"data_path" json:"data_path"`
	EmbeddingProvider string `yaml:"embedding_provider" json:"embedding_provider"`
	EmbeddingModel    string `yaml:"embedding_model" json:"embedding_model"`
	OutputMode        string `yaml:"output_mode" json:"output_mode"`
}

type Paths struct {
	ConfigFile string `json:"config_file"`
	ConfigDir  string `json:"config_dir"`
	DataDir    string `json:"data_dir"`
	BackupDir  string `json:"backup_dir"`
	LogFile    string `json:"log_file"`
	DBFile     string `json:"db_file"`
	IndexDir   string `json:"index_dir"`
}

func LoadOrCreate(configPath string) (*Config, Paths, error) {
	paths, err := resolvePaths(configPath)
	if err != nil {
		return nil, Paths{}, err
	}

	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		return nil, Paths{}, fmt.Errorf("create config dir: %w", err)
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

	paths = BuildPaths(cfg, paths.ConfigFile)
	if err := ensureDataPaths(paths); err != nil {
		return nil, Paths{}, err
	}
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
	home, _ := os.UserHomeDir()
	vault := filepath.Join(home, "Documents", "brain")
	data := filepath.Join(userDataDir(home), "brain")
	return &Config{
		VaultPath:         vault,
		DataPath:          data,
		EmbeddingProvider: defaultEmbedder,
		EmbeddingModel:    defaultModel,
		OutputMode:        defaultOutputMode,
	}
}

func userDataDir(home string) string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return expandHome(v)
	}
	return filepath.Join(home, ".local", "share")
}

func BuildPaths(cfg *Config, configFile string) Paths {
	dataDir := filepath.Clean(cfg.DataPath)
	return Paths{
		ConfigFile: configFile,
		ConfigDir:  filepath.Dir(configFile),
		DataDir:    dataDir,
		BackupDir:  filepath.Join(dataDir, "backups"),
		LogFile:    filepath.Join(dataDir, "history.jsonl"),
		DBFile:     filepath.Join(dataDir, "brain.sqlite3"),
		IndexDir:   filepath.Join(dataDir, "index"),
	}
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
	return Paths{
		ConfigFile: configPath,
		ConfigDir:  filepath.Dir(configPath),
	}, nil
}

func ensureDataPaths(paths Paths) error {
	for _, dir := range []string{paths.DataDir, paths.BackupDir, paths.IndexDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("BRAIN_VAULT_PATH"); v != "" {
		cfg.VaultPath = v
	}
	if v := os.Getenv("BRAIN_DATA_PATH"); v != "" {
		cfg.DataPath = v
	}
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
	if c.VaultPath == "" {
		c.VaultPath = Default().VaultPath
	}
	if c.DataPath == "" {
		c.DataPath = Default().DataPath
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
	c.VaultPath = filepath.Clean(expandHome(c.VaultPath))
	c.DataPath = filepath.Clean(expandHome(c.DataPath))
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
