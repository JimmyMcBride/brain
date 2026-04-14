package structure

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"brain/internal/index"
	"brain/internal/workspace"
)

type Manager struct {
	store     *index.Store
	workspace *workspace.Service
}

type Manifest struct {
	Signature string `json:"signature"`
	FileCount int    `json:"file_count"`
}

type State struct {
	IndexedAt          string `json:"indexed_at"`
	WorkspaceSignature string `json:"workspace_signature"`
	IndexedFileCount   int    `json:"indexed_file_count"`
	ItemCount          int    `json:"item_count"`
	BoundaryCount      int    `json:"boundary_count"`
	EntrypointCount    int    `json:"entrypoint_count"`
	ConfigSurfaceCount int    `json:"config_surface_count"`
	TestSurfaceCount   int    `json:"test_surface_count"`
}

type Status struct {
	State              string `json:"state"`
	Reason             string `json:"reason"`
	IndexedAt          string `json:"indexed_at,omitempty"`
	CurrentFileCount   int    `json:"current_file_count"`
	IndexedFileCount   int    `json:"indexed_file_count"`
	ItemCount          int    `json:"item_count"`
	BoundaryCount      int    `json:"boundary_count"`
	EntrypointCount    int    `json:"entrypoint_count"`
	ConfigSurfaceCount int    `json:"config_surface_count"`
	TestSurfaceCount   int    `json:"test_surface_count"`
}

type Item struct {
	ID       int64    `json:"id,omitempty"`
	Kind     string   `json:"kind"`
	Path     string   `json:"path"`
	Label    string   `json:"label"`
	Role     string   `json:"role"`
	Summary  string   `json:"summary"`
	Evidence []string `json:"evidence"`
}

var ignoredDirs = map[string]struct{}{
	".git":            {},
	".brain/state":    {},
	".brain/sessions": {},
	"node_modules":    {},
	"vendor":          {},
	".venv":           {},
	"venv":            {},
	"dist":            {},
	"build":           {},
	"coverage":        {},
	".next":           {},
}

func New(store *index.Store, workspaceSvc *workspace.Service) (*Manager, error) {
	if store == nil {
		return nil, fmt.Errorf("structure store is required")
	}
	manager := &Manager{store: store, workspace: workspaceSvc}
	if err := manager.InitSchema(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (m *Manager) InitSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS structure_state (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			indexed_at TEXT NOT NULL,
			workspace_signature TEXT NOT NULL,
			indexed_file_count INTEGER NOT NULL,
			item_count INTEGER NOT NULL,
			boundary_count INTEGER NOT NULL,
			entrypoint_count INTEGER NOT NULL,
			config_surface_count INTEGER NOT NULL,
			test_surface_count INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS structure_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL,
			path TEXT NOT NULL,
			label TEXT NOT NULL,
			role TEXT NOT NULL,
			summary TEXT NOT NULL,
			evidence_json TEXT NOT NULL
		);`,
	}
	for _, stmt := range stmts {
		if _, err := m.store.DB.Exec(stmt); err != nil {
			return fmt.Errorf("init structure schema: %w", err)
		}
	}
	return nil
}

func (m *Manager) BuildManifest() (Manifest, error) {
	if m.workspace == nil {
		return Manifest{}, fmt.Errorf("workspace service is required")
	}
	if err := m.workspace.Validate(); err != nil {
		return Manifest{}, err
	}
	hash := sha256.New()
	count := 0
	var entries []string
	err := filepath.WalkDir(m.workspace.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(m.workspace.Root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if shouldIgnoreDir(rel, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		entries = append(entries, fmt.Sprintf("%s\x00%d\x00%d", rel, info.Size(), info.ModTime().UTC().UnixNano()))
		count++
		return nil
	})
	if err != nil {
		return Manifest{}, fmt.Errorf("walk structural workspace: %w", err)
	}
	sort.Strings(entries)
	for _, entry := range entries {
		hash.Write([]byte(entry))
		hash.Write([]byte{'\n'})
	}
	return Manifest{
		Signature: hex.EncodeToString(hash.Sum(nil)),
		FileCount: count,
	}, nil
}

func (m *Manager) ReadState(ctx context.Context) (*State, error) {
	var state State
	err := m.store.DB.QueryRowContext(ctx, `
		SELECT indexed_at, workspace_signature, indexed_file_count, item_count, boundary_count, entrypoint_count, config_surface_count, test_surface_count
		FROM structure_state WHERE id = 1`,
	).Scan(
		&state.IndexedAt,
		&state.WorkspaceSignature,
		&state.IndexedFileCount,
		&state.ItemCount,
		&state.BoundaryCount,
		&state.EntrypointCount,
		&state.ConfigSurfaceCount,
		&state.TestSurfaceCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (m *Manager) WriteState(ctx context.Context, state State) error {
	_, err := m.store.DB.ExecContext(ctx, `DELETE FROM structure_state WHERE id = 1`)
	if err != nil {
		return fmt.Errorf("clear structure state: %w", err)
	}
	_, err = m.store.DB.ExecContext(ctx, `
		INSERT INTO structure_state(id, indexed_at, workspace_signature, indexed_file_count, item_count, boundary_count, entrypoint_count, config_surface_count, test_surface_count)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)`,
		state.IndexedAt,
		state.WorkspaceSignature,
		state.IndexedFileCount,
		state.ItemCount,
		state.BoundaryCount,
		state.EntrypointCount,
		state.ConfigSurfaceCount,
		state.TestSurfaceCount,
	)
	if err != nil {
		return fmt.Errorf("write structure state: %w", err)
	}
	return nil
}

func (m *Manager) ReplaceItems(ctx context.Context, items []Item) error {
	tx, err := m.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM structure_items`); err != nil {
		return fmt.Errorf("clear structure items: %w", err)
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO structure_items(kind, path, label, role, summary, evidence_json) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, item := range items {
		evidenceJSON, err := json.Marshal(item.Evidence)
		if err != nil {
			return fmt.Errorf("marshal structure evidence: %w", err)
		}
		if _, err := stmt.ExecContext(ctx, item.Kind, item.Path, item.Label, item.Role, item.Summary, string(evidenceJSON)); err != nil {
			return fmt.Errorf("insert structure item: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit structure items: %w", err)
	}
	return nil
}

func (m *Manager) Freshness(ctx context.Context) (*Status, error) {
	manifest, err := m.BuildManifest()
	if err != nil {
		return nil, err
	}
	state, err := m.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	status := &Status{
		State:            "missing",
		Reason:           "structure metadata missing",
		CurrentFileCount: manifest.FileCount,
	}
	if state == nil {
		return status, nil
	}
	status.IndexedAt = state.IndexedAt
	status.IndexedFileCount = state.IndexedFileCount
	status.ItemCount = state.ItemCount
	status.BoundaryCount = state.BoundaryCount
	status.EntrypointCount = state.EntrypointCount
	status.ConfigSurfaceCount = state.ConfigSurfaceCount
	status.TestSurfaceCount = state.TestSurfaceCount
	if state.WorkspaceSignature != manifest.Signature {
		status.State = "stale"
		status.Reason = "workspace signature changed"
		return status, nil
	}
	status.State = "fresh"
	status.Reason = "workspace matches"
	return status, nil
}

func shouldIgnoreDir(rel, name string) bool {
	if _, ok := ignoredDirs[rel]; ok {
		return true
	}
	if _, ok := ignoredDirs[name]; ok {
		return true
	}
	return strings.HasPrefix(name, ".git")
}

func NowUTCString() string {
	return time.Now().UTC().Format(time.RFC3339)
}
