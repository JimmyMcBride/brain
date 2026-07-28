package modules

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const grantSchemaVersion = 1

type grantFile struct {
	SchemaVersion int      `json:"schema_version"`
	Modules       GrantSet `json:"modules"`
}

type FileGrantStore struct {
	path string
}

func NewFileGrantStore(projectRoot string) *FileGrantStore {
	return &FileGrantStore{path: filepath.Join(projectRoot, ".brain", "state", "module-grants.json")}
}

func (s *FileGrantStore) Path() string {
	return s.path
}

func (s *FileGrantStore) Load() (GrantSet, error) {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return GrantSet{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read module grants: %w", err)
	}
	file := grantFile{}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse module grants: %w", err)
	}
	if file.SchemaVersion != grantSchemaVersion {
		return nil, fmt.Errorf("unsupported module grants schema version %d", file.SchemaVersion)
	}
	if file.Modules == nil {
		file.Modules = GrantSet{}
	}
	for id, permissions := range file.Modules {
		file.Modules[id] = setStrings(stringSet(permissions))
	}
	return file.Modules, nil
}

func (s *FileGrantStore) Save(grants GrantSet) error {
	normalized := GrantSet{}
	for id, permissions := range grants {
		values := setStrings(stringSet(permissions))
		if len(values) != 0 {
			normalized[id] = values
		}
	}
	raw, err := json.MarshalIndent(grantFile{
		SchemaVersion: grantSchemaVersion,
		Modules:       normalized,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal module grants: %w", err)
	}
	raw = append(raw, '\n')
	if err := atomicWriteFile(s.path, raw, 0o600); err != nil {
		return fmt.Errorf("write module grants: %w", err)
	}
	return nil
}
