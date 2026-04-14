package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"brain/internal/backup"
	"brain/internal/brainstorm"
	"brain/internal/config"
	"brain/internal/distill"
	"brain/internal/embeddings"
	"brain/internal/history"
	"brain/internal/index"
	"brain/internal/livecontext"
	"brain/internal/notes"
	"brain/internal/output"
	"brain/internal/plan"
	"brain/internal/project"
	"brain/internal/projectcontext"
	"brain/internal/search"
	"brain/internal/session"
	"brain/internal/skills"
	"brain/internal/structure"
	"brain/internal/templates"
	"brain/internal/workspace"
)

type App struct {
	Config     *config.Config
	Paths      config.Paths
	Workspace  *workspace.Service
	Templates  *templates.Manager
	Notes      *notes.Manager
	Backups    *backup.Manager
	History    *history.Logger
	Undoer     *history.Undoer
	Index      *index.Store
	Embedder   embeddings.Provider
	Search     *search.Engine
	Project    *project.Manager
	Brainstorm *brainstorm.Manager
	Distill    *distill.Manager
	Plan       *plan.Manager
	Skills     *skills.Installer
	Context    *projectcontext.Manager
	Structure  *structure.Manager
	Live       *livecontext.Manager
	Session    *session.Manager
	Output     *output.Printer
}

type Options struct {
	Stdout io.Writer
	Stderr io.Writer
}

func New(configPath, projectPath string, jsonOutput bool, opts Options) (*App, error) {
	cfg, globalPaths, err := config.LoadOrCreate(configPath)
	if err != nil {
		return nil, err
	}
	if jsonOutput {
		cfg.OutputMode = "json"
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	projectDir, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}
	paths := config.ProjectPaths(globalPaths, projectDir)
	if err := config.EnsureProjectPaths(paths); err != nil {
		return nil, err
	}

	workspaceSvc := workspace.New(projectDir)
	tpl := templates.New(filepathIfExists(workspaceSvc.Root, "templates"))
	backups := backup.New(paths.BackupDir)
	historyLog := history.New(paths.LogFile)
	embedder, err := embeddings.New(cfg)
	if err != nil {
		return nil, err
	}
	store, err := index.New(paths.DBFile)
	if err != nil {
		return nil, err
	}
	searchEngine := search.New(store, embedder)
	notesManager := notes.New(workspaceSvc, tpl, backups, historyLog)
	projectManager := project.New(notesManager, workspaceSvc)
	sessionManager := session.New(historyLog)
	brainstormManager := brainstorm.New(notesManager, searchEngine, projectManager)
	distillManager := distill.New(notesManager, searchEngine, projectManager, historyLog, sessionManager)
	planManager := plan.New(notesManager, projectManager)
	userHome, _ := os.UserHomeDir()
	structureManager, err := structure.New(store, workspaceSvc)
	if err != nil {
		return nil, err
	}
	liveContextManager := livecontext.New(historyLog)

	return &App{
		Config:     cfg,
		Paths:      paths,
		Workspace:  workspaceSvc,
		Templates:  tpl,
		Notes:      notesManager,
		Backups:    backups,
		History:    historyLog,
		Undoer:     history.NewUndoer(historyLog, backups, workspaceSvc),
		Index:      store,
		Embedder:   embedder,
		Search:     searchEngine,
		Project:    projectManager,
		Brainstorm: brainstormManager,
		Distill:    distillManager,
		Plan:       planManager,
		Skills:     skills.NewInstaller(userHome),
		Context:    projectcontext.New(userHome),
		Structure:  structureManager,
		Live:       liveContextManager,
		Session:    sessionManager,
		Output:     output.New(cfg.OutputMode, opts.Stdout),
	}, nil
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}
	if a.Index != nil {
		return a.Index.Close()
	}
	return nil
}

func filepathIfExists(base, child string) string {
	if base == "" {
		return ""
	}
	path := base + string(os.PathSeparator) + child
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func (a *App) EnsureWorkspace() error {
	if err := a.Workspace.Validate(); err != nil {
		return fmt.Errorf("%w; run `brain init --project %s` first", err, a.Paths.ProjectDir)
	}
	return nil
}

func (a *App) SyncIndex(ctx context.Context) error {
	if a == nil || a.Index == nil || a.Workspace == nil {
		return nil
	}
	_, err := a.EnsureFreshIndex(ctx)
	return err
}

func (a *App) IndexStatus(ctx context.Context) (*index.FreshnessStatus, error) {
	if a == nil || a.Index == nil || a.Workspace == nil {
		return nil, nil
	}
	return a.Index.Freshness(ctx, a.Workspace, a.Embedder)
}

func (a *App) EnsureFreshIndex(ctx context.Context) (*index.FreshnessStatus, error) {
	if a == nil || a.Index == nil || a.Workspace == nil {
		return nil, nil
	}
	status, err := a.IndexStatus(ctx)
	if err != nil {
		return nil, err
	}
	if status != nil && status.State == "fresh" {
		return status, nil
	}
	if _, err := a.Index.Reindex(ctx, a.Workspace, a.Embedder); err != nil {
		return nil, err
	}
	return a.IndexStatus(ctx)
}
