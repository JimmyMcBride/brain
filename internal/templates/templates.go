package templates

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

var defaults = map[string]string{
	"project.md": `# {{ .Title }}

## Outcome

## Next actions

## Notes
`,
	"area.md": `# {{ .Title }}

## Standard

## Current focus

## Notes
`,
	"resource.md": `# {{ .Title }}

## Summary

## References

## Notes
`,
	"capture.md": `# {{ .Title }}

Captured: {{ .Now }}

## Context

## Raw notes
`,
	"lesson.md": `# {{ .Title }}

## What happened

## Lesson

## Apply next time
`,
	"content_seed.md": `# {{ .Title }}

## Core idea

## Why it matters

## Supporting notes
`,
	"daily.md": `# Daily {{ .Date }}

## Focus

## Notes

## Wins
`,
	"brainstorm.md": `# Brainstorm: {{ .Title }}

Started: {{ .Now }}

## Focus Question

What are we exploring?

## Ideas

## Related

## Raw Notes
`,
	"project_meta.md": `# {{ .Title }}

## Outcome

## Current Status

## Notes
`,
	"epic.md": `# {{ .Title }}

Created: {{ .Now }}

## Summary

## Why It Matters

## Spec

## Sources

## Progress

## Notes
`,
	"spec.md": `# {{ .Title }}

Created: {{ .Now }}

## Why

## Problem

## Goals

## Non-Goals

## Requirements

## UX / Flows

## Data / Interfaces

## Risks / Open Questions

## Rollout

## Story Breakdown

## Resources

## Notes
`,
	"story.md": `# {{ .Title }}

Created: {{ .Now }}

## Description

## Acceptance Criteria

## Resources

## Notes
`,
}

type Manager struct {
	searchDirs []string
}

func New(extraDirs ...string) *Manager {
	dirs := make([]string, 0, 4+len(extraDirs))
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, "templates"))
	}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Join(filepath.Dir(exe), "templates"))
	}
	dirs = append(dirs, extraDirs...)
	return &Manager{searchDirs: dedupe(dirs)}
}

func (m *Manager) Available() []string {
	set := map[string]struct{}{}
	for name := range defaults {
		set[name] = struct{}{}
	}
	for _, dir := range m.searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
				set[entry.Name()] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (m *Manager) Load(name string) (string, error) {
	if name == "" {
		return "", errors.New("template name is required")
	}
	for _, dir := range m.searchDirs {
		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path)
		if err == nil {
			return string(raw), nil
		}
	}
	if raw, ok := defaults[name]; ok {
		return raw, nil
	}
	return "", fmt.Errorf("template not found: %s", name)
}

func (m *Manager) Render(name string, data map[string]any) (string, error) {
	raw, err := m.Load(name)
	if err != nil {
		return "", err
	}
	values := map[string]any{
		"Now":  time.Now().Format(time.RFC3339),
		"Date": time.Now().Format("2006-01-02"),
	}
	for k, v := range data {
		values[k] = v
	}

	tpl, err := template.New(name).Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, values); err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return strings.TrimRight(buf.String(), "\n") + "\n", nil
}

func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = filepath.Clean(item)
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
