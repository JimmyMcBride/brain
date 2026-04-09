package content

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"brain/internal/notes"
	"brain/internal/search"
)

type Manager struct {
	notes  *notes.Manager
	search *search.Engine
}

func New(notesManager *notes.Manager, searchEngine *search.Engine) *Manager {
	return &Manager{notes: notesManager, search: searchEngine}
}

func (m *Manager) Seed(path string) (*notes.Note, error) {
	stage := "seed"
	return m.notes.Update(path, notes.UpdateInput{
		Metadata: map[string]any{
			"type":          "content_seed",
			"content_stage": stage,
		},
		Summary: "promoted note to content seed",
	})
}

func (m *Manager) Gather(ctx context.Context, seedPath string, limit int) (map[string]any, error) {
	seed, err := m.notes.Read(seedPath)
	if err != nil {
		return nil, err
	}
	query := strings.TrimSpace(seed.Title + " " + seed.Content)
	results, err := m.search.Search(ctx, query, limit+1)
	if err != nil {
		return nil, err
	}
	related := make([]search.Result, 0, len(results))
	for _, result := range results {
		if result.NotePath == seed.Path {
			continue
		}
		related = append(related, result)
		if len(related) >= limit {
			break
		}
	}
	return map[string]any{
		"seed":    seed,
		"related": related,
	}, nil
}

func (m *Manager) Outline(ctx context.Context, seedPath string, limit int) (*notes.Note, error) {
	gathered, err := m.Gather(ctx, seedPath, limit)
	if err != nil {
		return nil, err
	}
	seed := gathered["seed"].(*notes.Note)
	related := gathered["related"].([]search.Result)

	var builder strings.Builder
	builder.WriteString("# " + seed.Title + " Outline Package\n\n")
	builder.WriteString("Generated: " + time.Now().Format(time.RFC3339) + "\n\n")
	builder.WriteString("## Seed\n\n")
	builder.WriteString("- Source: [[" + seed.Path + "]]\n")
	builder.WriteString("- Type: " + seed.Type + "\n\n")
	builder.WriteString("## Proposed outline\n\n")
	builder.WriteString("1. Hook\n")
	builder.WriteString("2. Core insight\n")
	builder.WriteString("3. Supporting evidence\n")
	builder.WriteString("4. Practical takeaway\n\n")
	builder.WriteString("## Related notes\n\n")
	for _, result := range related {
		builder.WriteString("- [[" + result.NotePath + "]]")
		if result.Heading != "" {
			builder.WriteString(" -> " + result.Heading)
		}
		builder.WriteString("\n")
	}
	return m.notes.Create(notes.CreateInput{
		Title:    seed.Title + " Outline",
		NoteType: "content_outline",
		Template: "content_seed.md",
		Section:  "Resources",
		Subdir:   filepath.ToSlash(filepath.Join("Content", "Outlines")),
		Body:     builder.String(),
		Metadata: map[string]any{
			"source_note":    seed.Path,
			"content_stage":  "outline",
			"generated_from": "brain content outline",
		},
	})
}

func (m *Manager) Publish(path, channel, repurpose string) (*notes.Note, error) {
	meta := map[string]any{
		"content_stage": "published",
		"published":     true,
		"published_at":  time.Now().UTC().Format(time.RFC3339),
	}
	if channel != "" {
		meta["published_channel"] = channel
	}
	if repurpose != "" {
		meta["repurpose_target"] = repurpose
	}
	return m.notes.Update(path, notes.UpdateInput{
		Metadata: meta,
		Summary:  fmt.Sprintf("marked note as published%s", channelSuffix(channel)),
	})
}

func channelSuffix(channel string) string {
	if channel == "" {
		return ""
	}
	return " to " + channel
}
