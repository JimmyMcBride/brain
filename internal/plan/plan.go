package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"brain/internal/notes"
	"brain/internal/project"
)

type ContainerInfo struct {
	Path       string `json:"path"`
	Title      string `json:"title"`
	TotalItems int    `json:"total_items"`
	DoneItems  int    `json:"done_items"`
}

type ItemInfo struct {
	Path      string `json:"path"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Container string `json:"container,omitempty"`
}

type ProjectStatus struct {
	Project         string          `json:"project"`
	ParadigmName    string          `json:"paradigm"`
	ContainerType   string          `json:"container_type"`
	ContainerPlural string          `json:"container_plural"`
	ItemType        string          `json:"item_type"`
	ItemPlural      string          `json:"item_plural"`
	Containers      []ContainerInfo `json:"containers"`
	TotalItems      int             `json:"total_items"`
	DoneItems       int             `json:"done_items"`
}

type ItemChanges struct {
	Status       string
	AddCriteria  []string
	AddResources []string
}

type Manager struct {
	notes   *notes.Manager
	project *project.Manager
}

func New(notesManager *notes.Manager, projectManager *project.Manager) *Manager {
	return &Manager{notes: notesManager, project: projectManager}
}

func (m *Manager) CreateContainer(title string) (*notes.Note, error) {
	info, err := m.requireParadigm()
	if err != nil {
		return nil, err
	}
	return m.notes.Create(notes.CreateInput{
		Title:    title,
		NoteType: info.Paradigm.ContainerType,
		Template: "container.md",
		Section:  ".brain",
		Subdir:   filepath.ToSlash(filepath.Join("planning", info.Paradigm.ContainerPlural)),
		Metadata: map[string]any{
			"project": info.Name,
		},
	})
}

func (m *Manager) ListContainers() ([]ContainerInfo, error) {
	info, err := m.requireParadigm()
	if err != nil {
		return nil, err
	}
	containerDir := filepath.ToSlash(filepath.Join(info.PlanningDir, info.Paradigm.ContainerPlural))
	items, err := m.listItemsInDir(filepath.ToSlash(filepath.Join(info.PlanningDir, info.Paradigm.ItemPlural)))
	if err != nil {
		return nil, err
	}
	noteFiles, err := m.readDir(containerDir)
	if err != nil {
		return nil, err
	}
	var containers []ContainerInfo
	for _, note := range noteFiles {
		ci := ContainerInfo{Path: note.Path, Title: note.Title}
		slug := slugFromPath(note.Path)
		for _, item := range items {
			if item.Container == slug {
				ci.TotalItems++
				if item.Status == "done" {
					ci.DoneItems++
				}
			}
		}
		containers = append(containers, ci)
	}
	return containers, nil
}

func (m *Manager) CreateItem(title, containerRef, description string, criteria, resources []string) (*notes.Note, error) {
	info, err := m.requireParadigm()
	if err != nil {
		return nil, err
	}
	meta := map[string]any{
		"status":  "todo",
		"project": info.Name,
	}
	if containerRef != "" {
		meta["container"] = slugify(containerRef)
	}
	note, err := m.notes.Create(notes.CreateInput{
		Title:    title,
		NoteType: info.Paradigm.ItemType,
		Template: "work_item.md",
		Section:  ".brain",
		Subdir:   filepath.ToSlash(filepath.Join("planning", info.Paradigm.ItemPlural)),
		Metadata: meta,
	})
	if err != nil {
		return nil, err
	}
	if description == "" && len(criteria) == 0 && len(resources) == 0 {
		return note, nil
	}
	return m.applyItemUpdates(note.Path, ItemChanges{
		AddCriteria:  criteria,
		AddResources: resources,
	}, description)
}

func (m *Manager) UpdateItem(itemSlug string, changes ItemChanges) (*notes.Note, error) {
	info, err := m.requireParadigm()
	if err != nil {
		return nil, err
	}
	itemPath := filepath.ToSlash(filepath.Join(info.PlanningDir, info.Paradigm.ItemPlural, itemSlug+".md"))
	meta := map[string]any{}
	if changes.Status != "" {
		if !isValidStatus(changes.Status) {
			return nil, fmt.Errorf("invalid status %q (expected: todo, in_progress, done)", changes.Status)
		}
		meta["status"] = changes.Status
	}
	if len(changes.AddCriteria) == 0 && len(changes.AddResources) == 0 {
		return m.notes.Update(itemPath, notes.UpdateInput{Metadata: meta, Summary: "updated work item"})
	}
	note, err := m.applyItemUpdates(itemPath, changes, "")
	if err != nil {
		return nil, err
	}
	if len(meta) == 0 {
		return note, nil
	}
	return m.notes.Update(itemPath, notes.UpdateInput{Metadata: meta, Summary: "updated work item"})
}

func (m *Manager) ListItems(filterContainer, filterStatus string) ([]ItemInfo, error) {
	info, err := m.requireParadigm()
	if err != nil {
		return nil, err
	}
	items, err := m.listItemsInDir(filepath.ToSlash(filepath.Join(info.PlanningDir, info.Paradigm.ItemPlural)))
	if err != nil {
		return nil, err
	}
	filterContainer = slugify(filterContainer)
	var filtered []ItemInfo
	for _, item := range items {
		if filterContainer != "" && item.Container != filterContainer {
			continue
		}
		if filterStatus != "" && item.Status != filterStatus {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered, nil
}

func (m *Manager) Status() (*ProjectStatus, error) {
	info, err := m.requireParadigm()
	if err != nil {
		return nil, err
	}
	containers, err := m.ListContainers()
	if err != nil {
		return nil, err
	}
	items, err := m.listItemsInDir(filepath.ToSlash(filepath.Join(info.PlanningDir, info.Paradigm.ItemPlural)))
	if err != nil {
		return nil, err
	}
	total := len(items)
	done := 0
	for _, item := range items {
		if item.Status == "done" {
			done++
		}
	}
	return &ProjectStatus{
		Project:         info.Name,
		ParadigmName:    info.Paradigm.Name,
		ContainerType:   info.Paradigm.ContainerType,
		ContainerPlural: info.Paradigm.ContainerPlural,
		ItemType:        info.Paradigm.ItemType,
		ItemPlural:      info.Paradigm.ItemPlural,
		Containers:      containers,
		TotalItems:      total,
		DoneItems:       done,
	}, nil
}

func (m *Manager) Promote(brainstormSlug string) ([]ItemInfo, error) {
	info, err := m.requireParadigm()
	if err != nil {
		return nil, err
	}
	bsPath := filepath.ToSlash(filepath.Join(info.BrainstormsDir, brainstormSlug+".md"))
	bs, err := m.notes.Read(bsPath)
	if err != nil {
		return nil, err
	}
	if bs.Type != "brainstorm" {
		return nil, fmt.Errorf("note %s is type %q, not brainstorm", bsPath, bs.Type)
	}
	ideas := extractIdeas(bs.Content)
	if len(ideas) == 0 {
		return nil, fmt.Errorf("no ideas found in brainstorm %s", bsPath)
	}
	var created []ItemInfo
	for _, idea := range ideas {
		note, err := m.CreateItem(idea, "", "", nil, []string{fmt.Sprintf("[[%s]]", bs.Path)})
		if err != nil {
			return nil, err
		}
		created = append(created, ItemInfo{Path: note.Path, Title: note.Title, Status: "todo"})
	}
	return created, nil
}

func (m *Manager) requireParadigm() (*project.ProjectInfo, error) {
	info, err := m.project.Resolve()
	if err != nil {
		return nil, err
	}
	if info.Paradigm == nil {
		return nil, fmt.Errorf("project management is not initialized; run `brain plan init --paradigm <paradigm>`")
	}
	return info, nil
}

func (m *Manager) applyItemUpdates(itemPath string, changes ItemChanges, description string) (*notes.Note, error) {
	note, err := m.notes.Read(itemPath)
	if err != nil {
		return nil, err
	}
	content := note.Content
	if strings.TrimSpace(description) != "" {
		content = notes.AppendUnderHeading(content, "Description", strings.TrimSpace(description))
	}
	for _, c := range changes.AddCriteria {
		trimmed := strings.TrimSpace(c)
		if trimmed == "" {
			continue
		}
		content = notes.AppendUnderHeading(content, "Acceptance Criteria", fmt.Sprintf("- [ ] %s", trimmed))
	}
	for _, resource := range changes.AddResources {
		trimmed := strings.TrimSpace(resource)
		if trimmed == "" {
			continue
		}
		entry := trimmed
		if !strings.HasPrefix(entry, "- ") {
			entry = "- " + entry
		}
		content = notes.AppendUnderHeading(content, "Resources", entry)
	}
	return m.notes.Update(itemPath, notes.UpdateInput{Body: &content, Summary: "updated work item"})
}

func (m *Manager) readDir(relDir string) ([]*notes.Note, error) {
	abs := m.notes.WorkspaceAbs(relDir)
	entries, err := os.ReadDir(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []*notes.Note
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		notePath := filepath.ToSlash(filepath.Join(relDir, entry.Name()))
		note, err := m.notes.Read(notePath)
		if err == nil {
			result = append(result, note)
		}
	}
	return result, nil
}

func (m *Manager) listItemsInDir(relDir string) ([]ItemInfo, error) {
	noteFiles, err := m.readDir(relDir)
	if err != nil {
		return nil, err
	}
	var items []ItemInfo
	for _, note := range noteFiles {
		status := "todo"
		if s, ok := note.Metadata["status"].(string); ok && s != "" {
			status = s
		}
		container := ""
		if c, ok := note.Metadata["container"].(string); ok {
			container = c
		}
		items = append(items, ItemInfo{
			Path:      note.Path,
			Title:     note.Title,
			Status:    status,
			Container: container,
		})
	}
	return items, nil
}

func slugFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".md")
}

func slugify(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, name)
	name = strings.Trim(name, "-")
	if name == "" {
		return "item"
	}
	return name
}

func isValidStatus(status string) bool {
	switch status {
	case "todo", "in_progress", "done":
		return true
	default:
		return false
	}
}

func extractIdeas(content string) []string {
	var ideas []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- **") {
			continue
		}
		rest := strings.TrimPrefix(trimmed, "- **")
		idx := strings.Index(rest, "** ")
		if idx < 0 {
			continue
		}
		idea := strings.TrimSpace(rest[idx+3:])
		if idea != "" {
			ideas = append(ideas, idea)
		}
	}
	if len(ideas) == 0 {
		inIdeas := false
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "# ") {
				title := strings.TrimSpace(strings.TrimLeft(line, "#"))
				if strings.EqualFold(title, "Ideas") {
					inIdeas = true
					continue
				}
				if inIdeas {
					break
				}
			}
			if inIdeas {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "- ") {
					ideas = append(ideas, strings.TrimPrefix(trimmed, "- "))
				}
			}
		}
	}
	return ideas
}
