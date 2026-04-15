package structure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type BoundaryRecord struct {
	ID                 string   `json:"id"`
	Label              string   `json:"label"`
	Role               string   `json:"role"`
	RootPath           string   `json:"root_path"`
	Files              []string `json:"files"`
	OwnedTests         []string `json:"owned_tests"`
	AdjacentBoundaries []string `json:"adjacent_boundaries"`
	Responsibilities   []string `json:"responsibilities"`
}

type BoundaryGraph struct {
	Boundaries []BoundaryRecord `json:"boundaries"`

	byID           map[string]*BoundaryRecord
	fileToBoundary map[string]string
}

func (m *Manager) BoundaryGraph(ctx context.Context) (*BoundaryGraph, error) {
	snapshot, err := m.Snapshot(ctx, "")
	if err != nil {
		return nil, err
	}
	return buildBoundaryGraph(m.workspace.Root, snapshot)
}

func (g *BoundaryGraph) BoundaryByID(id string) *BoundaryRecord {
	if g == nil {
		return nil
	}
	g.ensureIndexes()
	return g.byID[strings.TrimSpace(id)]
}

func (g *BoundaryGraph) BoundaryForFile(path string) *BoundaryRecord {
	if g == nil {
		return nil
	}
	g.ensureIndexes()
	path = filepath.ToSlash(strings.TrimSpace(path))
	if id, ok := g.fileToBoundary[path]; ok {
		return g.byID[id]
	}
	return g.boundaryForPath(path)
}

func buildBoundaryGraph(root string, snapshot *Snapshot) (*BoundaryGraph, error) {
	if snapshot == nil {
		return &BoundaryGraph{
			Boundaries:     []BoundaryRecord{},
			byID:           map[string]*BoundaryRecord{},
			fileToBoundary: map[string]string{},
		}, nil
	}

	graph := &BoundaryGraph{
		Boundaries:     make([]BoundaryRecord, 0, len(snapshot.Boundaries)),
		byID:           map[string]*BoundaryRecord{},
		fileToBoundary: map[string]string{},
	}

	for _, item := range snapshot.Boundaries {
		record := BoundaryRecord{
			ID:                 boundaryID(item.Path),
			Label:              item.Label,
			Role:               item.Role,
			RootPath:           normalizeBoundaryRoot(item.Path),
			Files:              []string{},
			OwnedTests:         []string{},
			AdjacentBoundaries: []string{},
			Responsibilities:   initialResponsibilities(item, snapshot),
		}
		graph.Boundaries = append(graph.Boundaries, record)
	}
	sort.Slice(graph.Boundaries, func(i, j int) bool {
		return graph.Boundaries[i].RootPath < graph.Boundaries[j].RootPath
	})
	for i := range graph.Boundaries {
		record := &graph.Boundaries[i]
		graph.byID[record.ID] = record
	}

	files, err := workspaceFiles(root)
	if err != nil {
		return nil, err
	}
	for _, path := range files {
		record := graph.boundaryForPath(path)
		if record == nil {
			continue
		}
		record.Files = append(record.Files, path)
		graph.fileToBoundary[path] = record.ID
		if isTestPath(path) {
			record.OwnedTests = append(record.OwnedTests, path)
		}
	}

	for _, item := range snapshot.TestSurfaces {
		for _, path := range expandTestSurface(root, item.Path) {
			record := graph.boundaryForPath(path)
			if record == nil {
				continue
			}
			record.OwnedTests = append(record.OwnedTests, path)
		}
	}

	assignAdjacency(graph.Boundaries)
	for i := range graph.Boundaries {
		record := &graph.Boundaries[i]
		record.Files = dedupeSortedStrings(record.Files)
		record.OwnedTests = dedupeSortedStrings(record.OwnedTests)
		record.AdjacentBoundaries = dedupeSortedStrings(record.AdjacentBoundaries)
		record.Responsibilities = dedupeSortedStrings(record.Responsibilities)
	}
	return graph, nil
}

func (g *BoundaryGraph) ensureIndexes() {
	if g.byID == nil {
		g.byID = map[string]*BoundaryRecord{}
		for i := range g.Boundaries {
			record := &g.Boundaries[i]
			g.byID[record.ID] = record
		}
	}
	if g.fileToBoundary == nil {
		g.fileToBoundary = map[string]string{}
		for i := range g.Boundaries {
			record := &g.Boundaries[i]
			for _, file := range record.Files {
				g.fileToBoundary[filepath.ToSlash(strings.TrimSpace(file))] = record.ID
			}
		}
	}
}

func (g *BoundaryGraph) boundaryForPath(path string) *BoundaryRecord {
	path = filepath.ToSlash(strings.TrimSpace(path))
	var best *BoundaryRecord
	for i := range g.Boundaries {
		record := &g.Boundaries[i]
		if strings.HasPrefix(path, record.RootPath) {
			if best == nil || len(record.RootPath) > len(best.RootPath) {
				best = record
			}
		}
	}
	return best
}

func workspaceFiles(root string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
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
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk workspace files: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func expandTestSurface(root, path string) []string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return nil
	}
	if isTestPath(path) {
		return []string{path}
	}
	if !strings.HasSuffix(path, "/") {
		return nil
	}
	abs := filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(path, "/")))
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil
	}
	out := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(strings.TrimSuffix(path, "/"), entry.Name()))
		if isTestPath(rel) {
			out = append(out, rel)
		}
	}
	sort.Strings(out)
	return out
}

func assignAdjacency(boundaries []BoundaryRecord) {
	for i := range boundaries {
		for j := range boundaries {
			if i == j {
				continue
			}
			if areAdjacentBoundaries(boundaries[i], boundaries[j]) {
				boundaries[i].AdjacentBoundaries = append(boundaries[i].AdjacentBoundaries, boundaries[j].ID)
			}
		}
	}
}

func areAdjacentBoundaries(a, b BoundaryRecord) bool {
	aRoot := strings.TrimSuffix(a.RootPath, "/")
	bRoot := strings.TrimSuffix(b.RootPath, "/")
	if aRoot == "" || bRoot == "" || aRoot == bRoot {
		return false
	}
	if strings.HasPrefix(aRoot+"/", bRoot+"/") || strings.HasPrefix(bRoot+"/", aRoot+"/") {
		return true
	}
	return filepath.ToSlash(filepath.Dir(aRoot)) == filepath.ToSlash(filepath.Dir(bRoot))
}

func boundaryID(path string) string {
	return strings.TrimSuffix(filepath.ToSlash(strings.TrimSpace(path)), "/")
}

func normalizeBoundaryRoot(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return ""
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

func initialResponsibilities(item Item, snapshot *Snapshot) []string {
	responsibilities := []string{}
	summary := strings.TrimSpace(item.Summary)
	if summary != "" {
		switch summary {
		case "Nested structural boundary", "Primary library boundary", "Application boundary", "Configuration boundary", "Test boundary", "Documentation boundary", "Brain-managed workspace boundary", "Continuous integration boundary", "Scripts boundary":
		default:
			responsibilities = append(responsibilities, summary)
		}
	}
	if len(responsibilities) == 0 {
		responsibilities = append(responsibilities, defaultResponsibility(item))
	}

	entrypoints := 0
	configs := 0
	for _, entry := range snapshot.Entrypoints {
		if strings.HasPrefix(entry.Path, normalizeBoundaryRoot(item.Path)) {
			entrypoints++
		}
	}
	for _, entry := range snapshot.ConfigSurfaces {
		if strings.HasPrefix(entry.Path, normalizeBoundaryRoot(item.Path)) {
			configs++
		}
	}
	if entrypoints > 0 {
		responsibilities = append(responsibilities, fmt.Sprintf("contains %d entrypoint surface(s)", entrypoints))
	}
	if configs > 0 {
		responsibilities = append(responsibilities, fmt.Sprintf("contains %d config surface(s)", configs))
	}
	return responsibilities
}

func defaultResponsibility(item Item) string {
	switch item.Role {
	case "app":
		return "application entry or command boundary"
	case "library":
		return "library and implementation boundary"
	case "config":
		return "configuration boundary"
	case "tests":
		return "test boundary"
	case "docs":
		return "documentation boundary"
	case "brain":
		return "Brain-managed workspace boundary"
	case "scripts":
		return "script and automation boundary"
	case "ci":
		return "continuous integration boundary"
	default:
		return "structural repo boundary"
	}
}

func dedupeSortedStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	sort.Strings(items)
	out := make([]string, 0, len(items))
	prev := ""
	for _, item := range items {
		item = strings.TrimSpace(filepath.ToSlash(item))
		if item == "" || item == prev {
			continue
		}
		out = append(out, item)
		prev = item
	}
	return out
}

func isTestPath(path string) bool {
	return testFilePattern.MatchString(filepath.ToSlash(strings.TrimSpace(path)))
}
