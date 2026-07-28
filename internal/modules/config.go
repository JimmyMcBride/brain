package modules

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const projectConfigSchemaVersion = 1

type FileConfigStore struct {
	path string
}

func NewFileConfigStore(projectRoot string) *FileConfigStore {
	return &FileConfigStore{path: filepath.Join(projectRoot, ".brain", "modules.yaml")}
}

func (s *FileConfigStore) Path() string {
	return s.path
}

func (s *FileConfigStore) Load() (ProjectConfig, error) {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyProjectConfig(), nil
	}
	if err != nil {
		return ProjectConfig{}, fmt.Errorf("read module config: %w", err)
	}
	config := ProjectConfig{}
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return ProjectConfig{}, fmt.Errorf("parse module config: %w", err)
	}
	if err := validateProjectConfig(config); err != nil {
		return ProjectConfig{}, err
	}
	return config, nil
}

func (s *FileConfigStore) Save(config ProjectConfig) error {
	if config.SchemaVersion == 0 {
		config.SchemaVersion = projectConfigSchemaVersion
	}
	if config.Modules == nil {
		config.Modules = map[string]ModuleConfig{}
	}
	if err := validateProjectConfig(config); err != nil {
		return err
	}
	raw, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal module config: %w", err)
	}
	if err := atomicWriteFile(s.path, raw, 0o644); err != nil {
		return fmt.Errorf("write module config: %w", err)
	}
	return nil
}

func emptyProjectConfig() ProjectConfig {
	return ProjectConfig{SchemaVersion: projectConfigSchemaVersion, Modules: map[string]ModuleConfig{}}
}

func validateProjectConfig(config ProjectConfig) error {
	if config.SchemaVersion != projectConfigSchemaVersion {
		return fmt.Errorf("unsupported module config schema version %d", config.SchemaVersion)
	}
	for id, entry := range config.Modules {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("module config contains an empty module ID")
		}
		if entry.ConfigVersion < 1 {
			return configFailure(id, fmt.Sprintf("config version must be at least 1, got %d", entry.ConfigVersion))
		}
		if err := rejectRawSecrets(entry.Config, nil); err != nil {
			return configFailure(id, err.Error())
		}
	}
	return nil
}

var secretConfigKeys = map[string]struct{}{
	"api_key":       {},
	"apikey":        {},
	"client_secret": {},
	"password":      {},
	"private_key":   {},
	"secret":        {},
	"token":         {},
}

func rejectRawSecrets(value any, path []string) error {
	switch typed := value.(type) {
	case Config:
		for key, child := range typed {
			if err := rejectConfigValue(key, child, path); err != nil {
				return err
			}
		}
	case map[string]any:
		for key, child := range typed {
			if err := rejectConfigValue(key, child, path); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := rejectRawSecrets(child, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func rejectConfigValue(key string, value any, path []string) error {
	nextPath := append(append([]string(nil), path...), key)
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	if _, secret := secretConfigKeys[normalized]; secret {
		reference, ok := value.(string)
		if !ok || !strings.HasPrefix(reference, "ref:") || strings.TrimSpace(strings.TrimPrefix(reference, "ref:")) == "" {
			return fmt.Errorf("raw secret value at %s is forbidden; use a ref: reference", strings.Join(nextPath, "."))
		}
	}
	return rejectRawSecrets(value, nextPath)
}
