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
	"regexp"
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

type Snapshot struct {
	Summary        Summary `json:"summary"`
	Boundaries     []Item  `json:"boundaries"`
	Entrypoints    []Item  `json:"entrypoints"`
	ConfigSurfaces []Item  `json:"config_surfaces"`
	TestSurfaces   []Item  `json:"test_surfaces"`
}

type Summary struct {
	Runtime            string `json:"runtime"`
	ItemCount          int    `json:"item_count"`
	BoundaryCount      int    `json:"boundary_count"`
	EntrypointCount    int    `json:"entrypoint_count"`
	ConfigSurfaceCount int    `json:"config_surface_count"`
	TestSurfaceCount   int    `json:"test_surface_count"`
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

var commonBoundaryRoots = map[string]struct{}{
	"cmd":      {},
	"internal": {},
	"pkg":      {},
	"src":      {},
	"app":      {},
	"services": {},
	"lib":      {},
	"scripts":  {},
	"config":   {},
	"test":     {},
	"tests":    {},
}

var manifestFiles = map[string]string{
	"go.mod":         "go",
	"package.json":   "node",
	"Cargo.toml":     "rust",
	"pyproject.toml": "python",
}

var rootConfigNames = map[string]struct{}{
	"go.mod":         {},
	"package.json":   {},
	"Cargo.toml":     {},
	"pyproject.toml": {},
	"Makefile":       {},
}

var testFilePattern = regexp.MustCompile(`(_test\.go|\.test\.[^/]+|\.spec\.[^/]+)$`)

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

func (m *Manager) ReadItems(ctx context.Context, pathPrefix string) ([]Item, error) {
	query := `SELECT id, kind, path, label, role, summary, evidence_json FROM structure_items`
	args := []any{}
	if prefix := strings.Trim(strings.TrimSpace(pathPrefix), "/"); prefix != "" {
		prefix = filepath.ToSlash(prefix)
		query += ` WHERE path = ? OR path LIKE ?`
		args = append(args, prefix, prefix+"/%")
	}
	query += ` ORDER BY kind, path`
	rows, err := m.store.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		var evidenceJSON string
		if err := rows.Scan(&item.ID, &item.Kind, &item.Path, &item.Label, &item.Role, &item.Summary, &evidenceJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(evidenceJSON), &item.Evidence); err != nil {
			return nil, fmt.Errorf("decode structure evidence: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
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

func (m *Manager) Rebuild(ctx context.Context) (*Snapshot, error) {
	items, runtime, err := m.scan()
	if err != nil {
		return nil, err
	}
	if err := m.ReplaceItems(ctx, items); err != nil {
		return nil, err
	}
	manifest, err := m.BuildManifest()
	if err != nil {
		return nil, err
	}
	snapshot := snapshotFromItems(items, runtime)
	if err := m.WriteState(ctx, State{
		IndexedAt:          NowUTCString(),
		WorkspaceSignature: manifest.Signature,
		IndexedFileCount:   manifest.FileCount,
		ItemCount:          snapshot.Summary.ItemCount,
		BoundaryCount:      snapshot.Summary.BoundaryCount,
		EntrypointCount:    snapshot.Summary.EntrypointCount,
		ConfigSurfaceCount: snapshot.Summary.ConfigSurfaceCount,
		TestSurfaceCount:   snapshot.Summary.TestSurfaceCount,
	}); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (m *Manager) Snapshot(ctx context.Context, pathPrefix string) (*Snapshot, error) {
	status, err := m.Freshness(ctx)
	if err != nil {
		return nil, err
	}
	if status.State != "fresh" {
		if _, err := m.Rebuild(ctx); err != nil {
			return nil, err
		}
	}
	items, err := m.ReadItems(ctx, pathPrefix)
	if err != nil {
		return nil, err
	}
	runtime := detectRuntimeFromItems(items)
	return snapshotFromItems(items, runtime), nil
}

func (m *Manager) scan() ([]Item, string, error) {
	if m.workspace == nil {
		return nil, "", fmt.Errorf("workspace service is required")
	}
	if err := m.workspace.Validate(); err != nil {
		return nil, "", err
	}

	itemsByKey := map[string]Item{}
	runtime := "unknown"
	testDirs := map[string]struct{}{}

	add := func(item Item) {
		if item.Path == "" {
			return
		}
		item.Path = filepath.ToSlash(strings.TrimPrefix(item.Path, "./"))
		key := item.Kind + ":" + item.Path
		if existing, ok := itemsByKey[key]; ok {
			if len(existing.Evidence) < len(item.Evidence) {
				itemsByKey[key] = item
			}
			return
		}
		itemsByKey[key] = item
	}

	entries, err := os.ReadDir(m.workspace.Root)
	if err != nil {
		return nil, "", err
	}
	for _, entry := range entries {
		name := entry.Name()
		rel := filepath.ToSlash(name)
		if entry.IsDir() {
			if shouldIgnoreDir(rel, name) {
				continue
			}
			add(rootBoundaryItem(rel))
			if _, ok := commonBoundaryRoots[name]; ok {
				children, err := os.ReadDir(filepath.Join(m.workspace.Root, name))
				if err == nil {
					for _, child := range children {
						if !child.IsDir() {
							continue
						}
						childRel := filepath.ToSlash(filepath.Join(name, child.Name()))
						add(boundaryChildItem(childRel))
					}
				}
			}
			if name == "config" {
				add(Item{Kind: "config_surface", Path: "config/", Label: "config", Role: "config", Summary: "Configuration subtree", Evidence: []string{"matched config directory"}})
			}
			if name == "test" || name == "tests" || name == "spec" {
				testDirs[rel] = struct{}{}
				add(Item{Kind: "test_surface", Path: rel + "/", Label: name, Role: "tests", Summary: "Top-level test surface", Evidence: []string{"matched test directory"}})
			}
			continue
		}
		if detected, ok := manifestFiles[name]; ok && runtime == "unknown" {
			runtime = detected
		}
		if strings.HasPrefix(name, "main.") {
			add(Item{Kind: "entrypoint", Path: rel, Label: name, Role: "app", Summary: "Root entrypoint file", Evidence: []string{"matched root main file"}})
		}
		if _, ok := rootConfigNames[name]; ok {
			add(Item{Kind: "entrypoint", Path: rel, Label: name, Role: roleForPath(rel), Summary: "Runtime manifest or bootstrap surface", Evidence: []string{"matched runtime manifest"}})
			add(Item{Kind: "config_surface", Path: rel, Label: name, Role: "config", Summary: "Root configuration surface", Evidence: []string{"matched root config manifest"}})
			continue
		}
		if strings.HasPrefix(name, ".env") || isRootConfigExt(name) {
			add(Item{Kind: "config_surface", Path: rel, Label: name, Role: "config", Summary: "Root configuration surface", Evidence: []string{"matched root config file"}})
		}
		if testFilePattern.MatchString(name) {
			dir := filepath.ToSlash(filepath.Dir(rel))
			if dir == "." {
				dir = ""
			}
			testDirs[dir] = struct{}{}
		}
	}

	err = filepath.WalkDir(m.workspace.Root, func(path string, d os.DirEntry, err error) error {
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
		base := filepath.Base(rel)
		dir := filepath.ToSlash(filepath.Dir(rel))
		if strings.HasPrefix(rel, ".github/workflows/") {
			add(Item{Kind: "config_surface", Path: rel, Label: base, Role: "ci", Summary: "CI workflow definition", Evidence: []string{"matched GitHub workflows path"}})
		}
		if strings.HasPrefix(rel, "cmd/") && strings.HasSuffix(rel, "/main.go") {
			add(Item{Kind: "entrypoint", Path: rel, Label: strings.TrimSuffix(strings.TrimPrefix(rel, "cmd/"), "/main.go"), Role: "app", Summary: "Command entrypoint", Evidence: []string{"matched cmd/*/main.*"}})
		}
		if (strings.HasPrefix(rel, "app/") || strings.HasPrefix(rel, "src/")) && strings.HasPrefix(base, "main.") {
			add(Item{Kind: "entrypoint", Path: rel, Label: base, Role: "app", Summary: "Application bootstrap file", Evidence: []string{"matched app/src main file"}})
		}
		if testFilePattern.MatchString(base) {
			if dir == "." {
				dir = ""
			}
			testDirs[dir] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("scan structural repo context: %w", err)
	}

	for dir := range testDirs {
		if dir == "" {
			add(Item{Kind: "test_surface", Path: filepath.ToSlash("tests"), Label: "tests", Role: "tests", Summary: "Root-level test surface", Evidence: []string{"matched root test files"}})
			continue
		}
		path := dir
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		add(Item{Kind: "test_surface", Path: path, Label: filepath.Base(dir), Role: "tests", Summary: "Test surface near changed or grouped code", Evidence: []string{"matched test files in directory"}})
	}

	items := make([]Item, 0, len(itemsByKey))
	for _, item := range itemsByKey {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Kind == items[j].Kind {
			return items[i].Path < items[j].Path
		}
		return items[i].Kind < items[j].Kind
	})
	return items, runtime, nil
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

func rootBoundaryItem(rel string) Item {
	role := roleForPath(rel)
	summary := fmt.Sprintf("%s area", role)
	switch role {
	case "library":
		summary = "Primary library boundary"
	case "app":
		summary = "Application boundary"
	case "config":
		summary = "Configuration boundary"
	case "tests":
		summary = "Test boundary"
	case "docs":
		summary = "Documentation boundary"
	case "brain":
		summary = "Brain-managed workspace boundary"
	case "ci":
		summary = "Continuous integration boundary"
	case "scripts":
		summary = "Scripts boundary"
	}
	return Item{
		Kind:     "boundary",
		Path:     rel + "/",
		Label:    rel,
		Role:     role,
		Summary:  summary,
		Evidence: []string{"important root directory"},
	}
}

func boundaryChildItem(rel string) Item {
	return Item{
		Kind:     "boundary",
		Path:     rel + "/",
		Label:    rel,
		Role:     roleForPath(rel),
		Summary:  "Nested structural boundary",
		Evidence: []string{"matched common source root child"},
	}
}

func roleForPath(rel string) string {
	rel = filepath.ToSlash(strings.TrimSuffix(rel, "/"))
	switch {
	case rel == "cmd" || strings.HasPrefix(rel, "cmd/"), rel == "app" || strings.HasPrefix(rel, "app/"):
		return "app"
	case rel == "internal" || strings.HasPrefix(rel, "internal/"), rel == "pkg" || strings.HasPrefix(rel, "pkg/"), rel == "lib" || strings.HasPrefix(rel, "lib/"), rel == "src" || strings.HasPrefix(rel, "src/"), rel == "services" || strings.HasPrefix(rel, "services/"):
		return "library"
	case rel == "config" || strings.HasPrefix(rel, "config/"):
		return "config"
	case rel == "test" || strings.HasPrefix(rel, "test/"), rel == "tests" || strings.HasPrefix(rel, "tests/"), rel == "spec" || strings.HasPrefix(rel, "spec/"):
		return "tests"
	case rel == "scripts" || strings.HasPrefix(rel, "scripts/"):
		return "scripts"
	case rel == "docs" || strings.HasPrefix(rel, "docs/"):
		return "docs"
	case rel == ".brain" || strings.HasPrefix(rel, ".brain/"):
		return "brain"
	case rel == ".github/workflows" || strings.HasPrefix(rel, ".github/workflows/"):
		return "ci"
	default:
		return "unknown"
	}
}

func isRootConfigExt(name string) bool {
	switch filepath.Ext(name) {
	case ".yaml", ".yml", ".json", ".toml":
		return true
	default:
		return false
	}
}

func snapshotFromItems(items []Item, runtime string) *Snapshot {
	snapshot := &Snapshot{
		Summary:        Summary{Runtime: runtime},
		Boundaries:     []Item{},
		Entrypoints:    []Item{},
		ConfigSurfaces: []Item{},
		TestSurfaces:   []Item{},
	}
	for _, item := range items {
		switch item.Kind {
		case "boundary":
			snapshot.Boundaries = append(snapshot.Boundaries, item)
		case "entrypoint":
			snapshot.Entrypoints = append(snapshot.Entrypoints, item)
		case "config_surface":
			snapshot.ConfigSurfaces = append(snapshot.ConfigSurfaces, item)
		case "test_surface":
			snapshot.TestSurfaces = append(snapshot.TestSurfaces, item)
		}
	}
	snapshot.Summary.ItemCount = len(items)
	snapshot.Summary.BoundaryCount = len(snapshot.Boundaries)
	snapshot.Summary.EntrypointCount = len(snapshot.Entrypoints)
	snapshot.Summary.ConfigSurfaceCount = len(snapshot.ConfigSurfaces)
	snapshot.Summary.TestSurfaceCount = len(snapshot.TestSurfaces)
	return snapshot
}

func detectRuntimeFromItems(items []Item) string {
	for _, item := range items {
		switch item.Path {
		case "go.mod":
			return "go"
		case "package.json":
			return "node"
		case "Cargo.toml":
			return "rust"
		case "pyproject.toml":
			return "python"
		}
	}
	return "unknown"
}
