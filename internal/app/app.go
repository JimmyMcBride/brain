package app

import (
	"fmt"
	"os"

	"brain/internal/backup"
	"brain/internal/config"
	"brain/internal/content"
	"brain/internal/embeddings"
	"brain/internal/history"
	"brain/internal/index"
	"brain/internal/notes"
	"brain/internal/output"
	"brain/internal/search"
	"brain/internal/skills"
	"brain/internal/templates"
	"brain/internal/vault"
)

type App struct {
	Config    *config.Config
	Paths     config.Paths
	Vault     *vault.Service
	Templates *templates.Manager
	Notes     *notes.Manager
	Backups   *backup.Manager
	History   *history.Logger
	Undoer    *history.Undoer
	Index     *index.Store
	Embedder  embeddings.Provider
	Search    *search.Engine
	Content   *content.Manager
	Skills    *skills.Installer
	Output    *output.Printer
}

func New(configPath string, jsonOutput bool) (*App, error) {
	cfg, paths, err := config.LoadOrCreate(configPath)
	if err != nil {
		return nil, err
	}
	if jsonOutput {
		cfg.OutputMode = "json"
	}

	vaultSvc := vault.New(cfg)
	tpl := templates.New(filepathIfExists(vaultSvc.Root, "templates"))
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
	notesManager := notes.New(vaultSvc, tpl, backups, historyLog)
	contentManager := content.New(notesManager, searchEngine)
	userHome, _ := os.UserHomeDir()

	return &App{
		Config:    cfg,
		Paths:     paths,
		Vault:     vaultSvc,
		Templates: tpl,
		Notes:     notesManager,
		Backups:   backups,
		History:   historyLog,
		Undoer:    history.NewUndoer(historyLog, backups, vaultSvc),
		Index:     store,
		Embedder:  embedder,
		Search:    searchEngine,
		Content:   contentManager,
		Skills:    skills.NewInstaller(userHome),
		Output:    output.New(cfg.OutputMode, os.Stdout),
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

func (a *App) EnsureVault() error {
	if err := a.Vault.Validate(); err != nil {
		return fmt.Errorf("%w; run `brain init` first", err)
	}
	return nil
}
