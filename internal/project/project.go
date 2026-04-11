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
	Name               string `json:"name"`
	Dir                string `json:"dir"`
	MetaPath           string `json:"meta_path"`
	BrainDir           string `json:"brain_dir"`
	BrainstormsDir     string `json:"brainstorms_dir"`
	PlanningDir        string `json:"planning_dir"`
	EpicsDir           string `json:"epics_dir"`
	SpecsDir           string `json:"specs_dir"`
	StoriesDir         string `json:"stories_dir"`
	ResourcesDir       string `json:"resources_dir"`
	PlanningInitialized bool   `json:"planning_initialized"`
	PlanningModel      string `json:"planning_model,omitempty"`
}

type projectFile struct {
	Name          string `yaml:"name"`
	PlanningModel string `yaml:"planning_model,omitempty"`
}

type Manager struct {
	notes     *notes.Manager
	workspace *workspace.Service
}

func New(notesManager *notes.Manager, workspaceSvc *workspace.Service) *Manager {
	return &Manager{notes: notesManager, workspace: workspaceSvc}
}

func (m *Manager) Init() (*ProjectInfo, error) {
	info, err := m.Resolve()
	if err != nil {
		return nil, err
	}
	if info.PlanningInitialized {
		return nil, fmt.Errorf("project management is already initialized")
	}
	if err := m.ensureLayout(info); err != nil {
		return nil, err
	}
	payload := projectFile{
		Name:          info.Name,
		PlanningModel: "epic_spec_v1",
	}
	raw, err := yaml.Marshal(&payload)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(m.workspace.Abs(info.MetaPath), raw, 0o644); err != nil {
		return nil, err
	}
	info.PlanningInitialized = true
	info.PlanningModel = payload.PlanningModel
	return info, nil
}

func (m *Manager) Resolve() (*ProjectInfo, error) {
	if err := m.workspace.Validate(); err != nil {
		return nil, err
	}
	name := filepath.Base(m.workspace.Root)
	info := &ProjectInfo{
		Name:               name,
		Dir:                ".",
		MetaPath:           ".brain/project.yaml",
		BrainDir:           ".brain",
		BrainstormsDir:     ".brain/brainstorms",
		PlanningDir:        ".brain/planning",
		EpicsDir:           ".brain/planning/epics",
		SpecsDir:           ".brain/planning/specs",
		StoriesDir:         ".brain/planning/stories",
		ResourcesDir:       ".brain/resources",
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
	switch {
	case cfg.PlanningModel == "epic_spec_v1":
		info.PlanningInitialized = true
		info.PlanningModel = cfg.PlanningModel
	case cfg.PlanningModel != "":
		return nil, fmt.Errorf("unsupported planning model %q", cfg.PlanningModel)
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

func (m *Manager) EnsurePlanningLayout() (*ProjectInfo, error) {
	info, err := m.Resolve()
	if err != nil {
		return nil, err
	}
	if !info.PlanningInitialized {
		return nil, fmt.Errorf("project planning is not initialized; run `brain plan init`")
	}
	if err := m.ensureLayout(info); err != nil {
		return nil, err
	}
	return info, nil
}

func (m *Manager) ensureLayout(info *ProjectInfo) error {
	for _, dir := range []string{
		info.BrainstormsDir,
		info.EpicsDir,
		info.SpecsDir,
		info.StoriesDir,
		info.ResourcesDir,
		filepath.ToSlash(filepath.Join(info.ResourcesDir, "captures")),
		filepath.ToSlash(filepath.Join(info.ResourcesDir, "changes")),
		filepath.ToSlash(filepath.Join(info.ResourcesDir, "references")),
	} {
		if err := os.MkdirAll(m.workspace.Abs(dir), 0o755); err != nil {
			return err
		}
	}
	return nil
}
