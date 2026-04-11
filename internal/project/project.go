package project

import (
	"fmt"
	"os"
	"path/filepath"

	"brain/internal/notes"
	"brain/internal/workspace"

	"gopkg.in/yaml.v3"
)

type ProjectInfo struct {
	Name           string    `json:"name"`
	Dir            string    `json:"dir"`
	MetaPath       string    `json:"meta_path"`
	BrainDir       string    `json:"brain_dir"`
	BrainstormsDir string    `json:"brainstorms_dir"`
	PlanningDir    string    `json:"planning_dir"`
	ResourcesDir   string    `json:"resources_dir"`
	Paradigm       *Paradigm `json:"paradigm,omitempty"`
}

type projectFile struct {
	Name            string `yaml:"name"`
	Paradigm        string `yaml:"paradigm,omitempty"`
	ContainerType   string `yaml:"container_type,omitempty"`
	ContainerPlural string `yaml:"container_plural,omitempty"`
	ItemType        string `yaml:"item_type,omitempty"`
	ItemPlural      string `yaml:"item_plural,omitempty"`
}

type Manager struct {
	notes     *notes.Manager
	workspace *workspace.Service
}

func New(notesManager *notes.Manager, workspaceSvc *workspace.Service) *Manager {
	return &Manager{notes: notesManager, workspace: workspaceSvc}
}

func (m *Manager) Init(paradigmName string) (*ProjectInfo, error) {
	info, err := m.Resolve()
	if err != nil {
		return nil, err
	}
	if info.Paradigm != nil {
		return nil, fmt.Errorf("project management is already initialized")
	}
	p, err := LookupParadigm(paradigmName)
	if err != nil {
		return nil, err
	}

	for _, dir := range []string{
		info.BrainstormsDir,
		filepath.ToSlash(filepath.Join(info.PlanningDir, p.ContainerPlural)),
		filepath.ToSlash(filepath.Join(info.PlanningDir, p.ItemPlural)),
		info.ResourcesDir,
		filepath.ToSlash(filepath.Join(info.ResourcesDir, "captures")),
		filepath.ToSlash(filepath.Join(info.ResourcesDir, "changes")),
		filepath.ToSlash(filepath.Join(info.ResourcesDir, "references")),
	} {
		if err := os.MkdirAll(m.workspace.Abs(dir), 0o755); err != nil {
			return nil, err
		}
	}

	payload := projectFile{
		Name:            info.Name,
		Paradigm:        p.Name,
		ContainerType:   p.ContainerType,
		ContainerPlural: p.ContainerPlural,
		ItemType:        p.ItemType,
		ItemPlural:      p.ItemPlural,
	}
	raw, err := yaml.Marshal(&payload)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(m.workspace.Abs(info.MetaPath), raw, 0o644); err != nil {
		return nil, err
	}
	info.Paradigm = p
	return info, nil
}

func (m *Manager) Resolve() (*ProjectInfo, error) {
	if err := m.workspace.Validate(); err != nil {
		return nil, err
	}
	name := filepath.Base(m.workspace.Root)
	info := &ProjectInfo{
		Name:           name,
		Dir:            ".",
		MetaPath:       ".brain/project.yaml",
		BrainDir:       ".brain",
		BrainstormsDir: ".brain/brainstorms",
		PlanningDir:    ".brain/planning",
		ResourcesDir:   ".brain/resources",
	}

	raw, err := os.ReadFile(m.workspace.Abs(info.MetaPath))
	if err != nil {
		if os.IsNotExist(err) {
			return info, nil
		}
		return nil, err
	}
	var cfg projectFile
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse project config: %w", err)
	}
	if cfg.Name != "" {
		info.Name = cfg.Name
	}
	if cfg.Paradigm != "" {
		p, err := LookupParadigm(cfg.Paradigm)
		if err != nil {
			return nil, err
		}
		info.Paradigm = p
	}
	return info, nil
}

func (m *Manager) List() ([]ProjectInfo, error) {
	info, err := m.Resolve()
	if err != nil {
		return nil, err
	}
	return []ProjectInfo{*info}, nil
}
