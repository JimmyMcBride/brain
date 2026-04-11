package cmd

import (
	"path/filepath"

	"brain/internal/config"
	"brain/internal/embeddings"
	"brain/internal/index"
	"brain/internal/projectcontext"
	"brain/internal/workspace"
)

type bootstrapResult struct {
	Config     *config.Config
	Global     config.Paths
	Project    config.Paths
	ProjectDir string
}

func bootstrapProject(flags *rootFlagsState, provider, model string) (*bootstrapResult, error) {
	cfg, globalPaths, err := config.LoadOrCreate(flags.configPath)
	if err != nil {
		return nil, err
	}
	if provider != "" {
		cfg.EmbeddingProvider = provider
	}
	if model != "" {
		cfg.EmbeddingModel = model
	}
	if err := config.Save(cfg, globalPaths.ConfigFile); err != nil {
		return nil, err
	}

	projectDir, err := filepath.Abs(flags.projectPath)
	if err != nil {
		return nil, err
	}
	projectPaths := config.ProjectPaths(globalPaths, projectDir)
	if err := config.EnsureProjectPaths(projectPaths); err != nil {
		return nil, err
	}

	projectWorkspace := workspace.New(projectDir)
	if err := projectWorkspace.Initialize(); err != nil {
		return nil, err
	}
	if _, err := embeddings.New(cfg); err != nil {
		return nil, err
	}
	store, err := index.New(projectPaths.DBFile)
	if err != nil {
		return nil, err
	}
	_ = store.Close()

	return &bootstrapResult{
		Config:     cfg,
		Global:     globalPaths,
		Project:    projectPaths,
		ProjectDir: projectDir,
	}, nil
}

func contextManager() *projectcontext.Manager {
	return projectcontext.New(userHomeDir())
}
