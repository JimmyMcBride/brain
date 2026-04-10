package notes

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"brain/internal/backup"
	"brain/internal/history"
	"brain/internal/templates"
	"brain/internal/vault"
)

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

type CreateInput struct {
	Title     string
	Filename  string
	NoteType  string
	Template  string
	Section   string
	Subdir    string
	Body      string
	Metadata  map[string]any
	Overwrite bool
}

type UpdateInput struct {
	Title    *string
	Body     *string
	Metadata map[string]any
	Summary  string
}

type Manager struct {
	vault     *vault.Service
	templates *templates.Manager
	backups   *backup.Manager
	history   *history.Logger
	editorRun func(editor, path string) error
}

func New(vaultSvc *vault.Service, tpl *templates.Manager, backups *backup.Manager, historyLog *history.Logger) *Manager {
	return &Manager{
		vault:     vaultSvc,
		templates: tpl,
		backups:   backups,
		history:   historyLog,
		editorRun: runEditor,
	}
}

func (m *Manager) Create(input CreateInput) (*Note, error) {
	if err := m.vault.Validate(); err != nil {
		return nil, err
	}
	if input.Title == "" {
		return nil, errors.New("title is required")
	}
	if input.Section == "" {
		input.Section = "Resources"
	}
	if input.Template == "" {
		input.Template = "resource.md"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	relDir := filepath.ToSlash(filepath.Join(input.Section, input.Subdir))
	filename := input.Filename
	if filename == "" {
		filename = slugify(input.Title)
	}
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}
	relPath := filepath.ToSlash(filepath.Join(relDir, filename))
	absPath := m.vault.Abs(relPath)
	if _, err := os.Stat(absPath); err == nil && !input.Overwrite {
		return nil, fmt.Errorf("note already exists: %s", relPath)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, fmt.Errorf("create note parent: %w", err)
	}

	body := strings.TrimSpace(input.Body)
	if body == "" {
		rendered, err := m.templates.Render(input.Template, map[string]any{
			"Title": input.Title,
			"Type":  input.NoteType,
			"Date":  time.Now().Format("2006-01-02"),
			"Now":   now,
		})
		if err != nil {
			return nil, err
		}
		body = rendered
	}
	meta := map[string]any{
		"title":   input.Title,
		"type":    input.NoteType,
		"created": now,
		"updated": now,
	}
	for k, v := range input.Metadata {
		meta[k] = v
	}
	raw, err := ComposeFrontmatter(meta, body)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(absPath, []byte(raw), 0o644); err != nil {
		return nil, fmt.Errorf("write note: %w", err)
	}
	if err := m.history.Append(history.Entry{
		Operation: "create",
		File:      relPath,
		Summary:   "created note",
		Metadata:  map[string]any{"title": input.Title, "type": input.NoteType},
	}); err != nil {
		return nil, err
	}
	return m.Read(relPath)
}

func (m *Manager) Read(path string) (*Note, error) {
	abs, rel, err := m.vault.ResolveMarkdown(path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read note: %w", err)
	}
	meta, body, err := ParseFrontmatter(string(raw))
	if err != nil {
		return nil, err
	}
	title := stringValue(meta["title"])
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(rel), ".md")
	}
	noteType := stringValue(meta["type"])
	if noteType == "" {
		noteType = inferTypeFromPath(rel)
	}
	return &Note{
		ID:       rel,
		Title:    title,
		Path:     rel,
		Type:     noteType,
		Metadata: meta,
		Content:  strings.TrimLeft(body, "\n"),
	}, nil
}

func (m *Manager) Update(path string, input UpdateInput) (*Note, error) {
	note, err := m.Read(path)
	if err != nil {
		return nil, err
	}
	abs := m.vault.Abs(note.Path)
	backupPath, err := m.backups.Create(abs)
	if err != nil {
		return nil, err
	}
	if input.Title != nil {
		note.Title = *input.Title
		note.Metadata["title"] = *input.Title
	}
	if input.Body != nil {
		note.Content = *input.Body
	}
	for k, v := range input.Metadata {
		note.Metadata[k] = v
	}
	note.Metadata["updated"] = time.Now().UTC().Format(time.RFC3339)
	raw, err := ComposeFrontmatter(note.Metadata, note.Content)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(abs, []byte(raw), 0o644); err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}
	summary := input.Summary
	if summary == "" {
		summary = "updated note"
	}
	if err := m.history.Append(history.Entry{
		Operation:  "update",
		File:       note.Path,
		BackupPath: backupPath,
		Summary:    summary,
	}); err != nil {
		return nil, err
	}
	return m.Read(note.Path)
}

func (m *Manager) EditInEditor(path, editor string) (*Note, error) {
	note, err := m.Read(path)
	if err != nil {
		return nil, err
	}
	abs := m.vault.Abs(note.Path)
	before, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	backupPath, err := m.backups.Create(abs)
	if err != nil {
		return nil, err
	}
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	if err := m.editorRun(editor, abs); err != nil {
		return nil, fmt.Errorf("run editor: %w", err)
	}
	after, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	if string(before) == string(after) {
		return note, nil
	}
	if err := m.history.Append(history.Entry{
		Operation:  "update",
		File:       note.Path,
		BackupPath: backupPath,
		Summary:    "edited note in editor",
	}); err != nil {
		return nil, err
	}
	return m.Read(note.Path)
}

func (m *Manager) Rename(path, newTitle string) (string, string, error) {
	note, err := m.Read(path)
	if err != nil {
		return "", "", err
	}
	oldAbs := m.vault.Abs(note.Path)
	backupPath, err := m.backups.Create(oldAbs)
	if err != nil {
		return "", "", err
	}
	newRel := filepath.ToSlash(filepath.Join(filepath.Dir(note.Path), slugify(newTitle)+".md"))
	newAbs := m.vault.Abs(newRel)
	if err := os.MkdirAll(filepath.Dir(newAbs), 0o755); err != nil {
		return "", "", err
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return "", "", fmt.Errorf("rename note: %w", err)
	}
	updatedTitle := newTitle
	if _, err := m.Update(newRel, UpdateInput{
		Title:   &updatedTitle,
		Summary: "updated note title after rename",
	}); err != nil {
		return "", "", err
	}
	if err := m.history.Append(history.Entry{
		Operation:  "rename",
		File:       note.Path,
		Target:     newRel,
		BackupPath: backupPath,
		Summary:    "renamed note",
	}); err != nil {
		return "", "", err
	}
	return note.Path, newRel, nil
}

func (m *Manager) Move(path, destination string) (string, string, error) {
	note, err := m.Read(path)
	if err != nil {
		return "", "", err
	}
	oldAbs := m.vault.Abs(note.Path)
	backupPath, err := m.backups.Create(oldAbs)
	if err != nil {
		return "", "", err
	}
	destRel := destination
	if strings.HasSuffix(destination, "/") || filepath.Ext(destination) == "" {
		destRel = filepath.ToSlash(filepath.Join(destination, filepath.Base(note.Path)))
	}
	if !strings.HasSuffix(destRel, ".md") {
		destRel += ".md"
	}
	destAbs := m.vault.Abs(destRel)
	if err := os.MkdirAll(filepath.Dir(destAbs), 0o755); err != nil {
		return "", "", err
	}
	if err := os.Rename(oldAbs, destAbs); err != nil {
		return "", "", fmt.Errorf("move note: %w", err)
	}
	if err := m.history.Append(history.Entry{
		Operation:  "move",
		File:       note.Path,
		Target:     filepath.ToSlash(destRel),
		BackupPath: backupPath,
		Summary:    "moved note",
	}); err != nil {
		return "", "", err
	}
	return note.Path, filepath.ToSlash(destRel), nil
}

func (m *Manager) Find(query, noteType, pathFilter string, limit int) ([]map[string]any, error) {
	files, err := m.vault.WalkMarkdownFiles()
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	noteType = strings.ToLower(strings.TrimSpace(noteType))
	pathFilter = strings.ToLower(strings.TrimSpace(pathFilter))

	results := make([]map[string]any, 0, len(files))
	for _, file := range files {
		rel, err := m.vault.Rel(file)
		if err != nil {
			return nil, err
		}
		note, err := m.Read(rel)
		if err != nil {
			return nil, err
		}
		if noteType != "" && strings.ToLower(note.Type) != noteType {
			continue
		}
		if pathFilter != "" && !strings.Contains(strings.ToLower(note.Path), pathFilter) {
			continue
		}
		if query != "" && !matchesFindQuery(note, query) {
			continue
		}
		results = append(results, map[string]any{
			"path":  note.Path,
			"title": note.Title,
			"type":  note.Type,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i]["path"].(string) < results[j]["path"].(string)
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func slugify(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	title = slugPattern.ReplaceAllString(title, "-")
	title = strings.Trim(title, "-")
	if title == "" {
		return time.Now().Format("20060102-150405")
	}
	return title
}

func inferTypeFromPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) == 0 {
		return "note"
	}
	switch parts[0] {
	case "Projects":
		return "project"
	case "Areas":
		return "area"
	case "Resources":
		return "resource"
	case "Archives":
		return "archive"
	default:
		return "note"
	}
}

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func runEditor(editor, path string) error {
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func matchesFindQuery(note *Note, query string) bool {
	if strings.Contains(strings.ToLower(note.Path), query) ||
		strings.Contains(strings.ToLower(note.Title), query) ||
		strings.Contains(strings.ToLower(note.Type), query) ||
		strings.Contains(strings.ToLower(note.Content), query) {
		return true
	}
	for key, value := range note.Metadata {
		if strings.Contains(strings.ToLower(key), query) || strings.Contains(strings.ToLower(fmt.Sprint(value)), query) {
			return true
		}
	}
	return false
}
