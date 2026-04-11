package brainstorm

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"brain/internal/notes"
	"brain/internal/project"
	"brain/internal/search"
)

// Manager handles brainstorm session lifecycle.
type Manager struct {
	notes   *notes.Manager
	search  *search.Engine
	project *project.Manager
}

// New creates a brainstorm Manager.
func New(notesManager *notes.Manager, searchEngine *search.Engine, projectManager *project.Manager) *Manager {
	return &Manager{notes: notesManager, search: searchEngine, project: projectManager}
}

// Start creates a new brainstorm note in the current project.
func (m *Manager) Start(topic string) (*notes.Note, error) {
	info, err := m.project.Resolve()
	if err != nil {
		return nil, err
	}

	return m.notes.Create(notes.CreateInput{
		Title:    topic,
		NoteType: "brainstorm",
		Template: "brainstorm.md",
		Section:  ".brain",
		Subdir:   "brainstorms",
		Metadata: map[string]any{
			"brainstorm_status": "active",
			"idea_count":        0,
			"project":           info.Name,
		},
	})
}

// Idea appends a timestamped idea to a brainstorm note under the ## Ideas heading.
func (m *Manager) Idea(path string, body string) (*notes.Note, error) {
	note, err := m.notes.Read(path)
	if err != nil {
		return nil, err
	}
	if note.Type != "brainstorm" {
		return nil, fmt.Errorf("note %s is type %q, not brainstorm", path, note.Type)
	}

	stamp := time.Now().Format("15:04")
	entry := fmt.Sprintf("- **%s** %s", stamp, strings.TrimSpace(body))
	newBody := notes.AppendUnderHeading(note.Content, "Ideas", entry)

	count := 0
	if v, ok := note.Metadata["idea_count"]; ok {
		switch c := v.(type) {
		case int:
			count = c
		case float64:
			count = int(c)
		}
	}
	count++

	return m.notes.Update(path, notes.UpdateInput{
		Body: &newBody,
		Metadata: map[string]any{
			"idea_count": count,
		},
		Summary: "added brainstorm idea",
	})
}

// Gather finds project notes related to the brainstorm topic.
func (m *Manager) Gather(ctx context.Context, path string, limit int) (map[string]any, error) {
	note, err := m.notes.Read(path)
	if err != nil {
		return nil, err
	}
	query := strings.TrimSpace(note.Title + " " + note.Content)
	results, err := m.search.Search(ctx, query, limit+1)
	if err != nil {
		return nil, err
	}
	related := make([]search.Result, 0, len(results))
	for _, r := range results {
		if r.NotePath == note.Path {
			continue
		}
		related = append(related, r)
		if len(related) >= limit {
			break
		}
	}
	return map[string]any{
		"brainstorm": note,
		"related":    related,
	}, nil
}

// Distill creates a summary note from a brainstorm, gathering related project context.
// The distillation is stored alongside the brainstorm in the project's brainstorms directory.
func (m *Manager) Distill(ctx context.Context, path string, limit int) (*notes.Note, error) {
	note, err := m.notes.Read(path)
	if err != nil {
		return nil, err
	}
	if note.Type != "brainstorm" {
		return nil, fmt.Errorf("note %s is type %q, not brainstorm", path, note.Type)
	}

	// Gather related notes.
	gathered, err := m.Gather(ctx, path, limit)
	if err != nil {
		return nil, err
	}
	related := gathered["related"].([]search.Result)

	// Extract ideas section.
	ideas := extractSection(note.Content, "Ideas")

	var b strings.Builder
	b.WriteString("## Source\n\n")
	b.WriteString("- [[" + note.Path + "]]\n\n")
	b.WriteString("## Key Ideas\n\n")
	if ideas != "" {
		b.WriteString(ideas + "\n\n")
	}
	b.WriteString("## Themes\n\n")
	b.WriteString("## Related Notes\n\n")
	for _, r := range related {
		b.WriteString("- [[" + r.NotePath + "]]")
		if r.Heading != "" {
			b.WriteString(" -> " + r.Heading)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## Action Items\n")

	parentDir := filepath.Dir(note.Path)

	return m.notes.Create(notes.CreateInput{
		Title:    note.Title + " Distillation",
		NoteType: "brainstorm_distill",
		Template: "brainstorm.md",
		Section:  "",
		Subdir:   parentDir,
		Body:     b.String(),
		Metadata: map[string]any{
			"source_brainstorm": note.Path,
			"distilled_from":    "brain brainstorm distill",
		},
	})
}

// extractSection pulls content between a heading and the next heading of same or higher level.
func extractSection(content, heading string) string {
	lines := strings.Split(content, "\n")
	inSection := false
	sectionLevel := 0
	var out []string

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			level := 0
			for _, ch := range line {
				if ch == '#' {
					level++
				} else {
					break
				}
			}
			title := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if strings.EqualFold(title, heading) {
				inSection = true
				sectionLevel = level
				continue
			}
			if inSection && level <= sectionLevel {
				break
			}
		}
		if inSection {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
