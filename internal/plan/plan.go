package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"brain/internal/notes"
	"brain/internal/project"
)

var validSpecStatuses = map[string]struct{}{
	"draft":        {},
	"approved":     {},
	"implementing": {},
	"done":         {},
}

var validStoryStatuses = map[string]struct{}{
	"todo":        {},
	"in_progress": {},
	"blocked":     {},
	"done":        {},
}

type EpicInfo struct {
	Path             string `json:"path"`
	Title            string `json:"title"`
	Spec             string `json:"spec"`
	SpecStatus       string `json:"spec_status"`
	SourceBrainstorm string `json:"source_brainstorm,omitempty"`
	TotalStories     int    `json:"total_stories"`
	DoneStories      int    `json:"done_stories"`
}

type StoryInfo struct {
	Path   string `json:"path"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Epic   string `json:"epic"`
	Spec   string `json:"spec"`
}

type ProjectStatus struct {
	Project           string     `json:"project"`
	PlanningModel     string     `json:"planning_model"`
	Epics             []EpicInfo `json:"epics"`
	TotalStories      int        `json:"total_stories"`
	DoneStories       int        `json:"done_stories"`
	BlockedStories    int        `json:"blocked_stories"`
	InProgressStories int        `json:"in_progress_stories"`
}

type StoryChanges struct {
	Status       string
	AddCriteria  []string
	AddResources []string
}

type EpicBundle struct {
	Epic *notes.Note `json:"epic"`
	Spec *notes.Note `json:"spec"`
}

type Manager struct {
	notes   *notes.Manager
	project *project.Manager
}

func New(notesManager *notes.Manager, projectManager *project.Manager) *Manager {
	return &Manager{notes: notesManager, project: projectManager}
}

func (m *Manager) CreateEpic(title, sourceBrainstorm string) (*EpicBundle, error) {
	info, err := m.requirePlanning()
	if err != nil {
		return nil, err
	}
	epicSlug := slugify(title)
	epic, err := m.notes.Create(notes.CreateInput{
		Title:    title,
		NoteType: "epic",
		Template: "epic.md",
		Section:  ".brain",
		Subdir:   "planning/epics",
		Metadata: map[string]any{
			"project": info.Name,
			"spec":    epicSlug,
		},
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sourceBrainstorm) != "" {
		epic, err = m.notes.Update(epic.Path, notes.UpdateInput{
			Metadata: map[string]any{"source_brainstorm": sourceBrainstorm},
			Summary:  "linked epic source brainstorm",
		})
		if err != nil {
			return nil, err
		}
	}
	spec, err := m.createSpecForEpic(info, epic, sourceBrainstorm)
	if err != nil {
		return nil, err
	}
	return &EpicBundle{Epic: epic, Spec: spec}, nil
}

func (m *Manager) ListEpics() ([]EpicInfo, error) {
	info, err := m.requirePlanning()
	if err != nil {
		return nil, err
	}
	epicNotes, err := m.readDir(info.EpicsDir)
	if err != nil {
		return nil, err
	}
	stories, err := m.listStoriesInDir(info.StoriesDir)
	if err != nil {
		return nil, err
	}
	var epics []EpicInfo
	for _, note := range epicNotes {
		slug := slugFromPath(note.Path)
		specSlug := stringValue(note.Metadata["spec"])
		if specSlug == "" {
			specSlug = slug
		}
		specStatus := "draft"
		if spec, err := m.notes.Read(m.specPath(specSlug)); err == nil {
			if s := stringValue(spec.Metadata["status"]); s != "" {
				specStatus = s
			}
		}
		info := EpicInfo{
			Path:             note.Path,
			Title:            note.Title,
			Spec:             specSlug,
			SpecStatus:       specStatus,
			SourceBrainstorm: stringValue(note.Metadata["source_brainstorm"]),
		}
		for _, story := range stories {
			if story.Epic != slug {
				continue
			}
			info.TotalStories++
			if story.Status == "done" {
				info.DoneStories++
			}
		}
		epics = append(epics, info)
	}
	return epics, nil
}

func (m *Manager) ReadEpic(epicSlug string) (*notes.Note, error) {
	if _, err := m.requirePlanning(); err != nil {
		return nil, err
	}
	return m.notes.Read(m.epicPath(epicSlug))
}

func (m *Manager) ReadSpec(epicSlug string) (*notes.Note, error) {
	if _, err := m.requirePlanning(); err != nil {
		return nil, err
	}
	return m.readSpecForEpic(epicSlug)
}

func (m *Manager) UpdateSpec(epicSlug string, input notes.UpdateInput) (*notes.Note, error) {
	if _, err := m.requirePlanning(); err != nil {
		return nil, err
	}
	if status := stringValue(input.Metadata["status"]); status != "" {
		if !isValidSpecStatus(status) {
			return nil, fmt.Errorf("invalid spec status %q (expected: draft, approved, implementing, done)", status)
		}
	}
	spec, err := m.readSpecForEpic(epicSlug)
	if err != nil {
		return nil, err
	}
	if input.Summary == "" {
		input.Summary = "updated spec"
	}
	return m.notes.Update(spec.Path, input)
}

func (m *Manager) SetSpecStatus(epicSlug, status string) (*notes.Note, error) {
	if !isValidSpecStatus(status) {
		return nil, fmt.Errorf("invalid spec status %q (expected: draft, approved, implementing, done)", status)
	}
	return m.UpdateSpec(epicSlug, notes.UpdateInput{
		Metadata: map[string]any{"status": status},
		Summary:  "updated spec status",
	})
}

func (m *Manager) CreateStory(epicSlug, title, description string, criteria, resources []string) (*notes.Note, error) {
	info, err := m.requirePlanning()
	if err != nil {
		return nil, err
	}
	epic, err := m.ReadEpic(epicSlug)
	if err != nil {
		return nil, err
	}
	spec, err := m.readSpecForEpic(epicSlug)
	if err != nil {
		return nil, err
	}
	specStatus := stringValue(spec.Metadata["status"])
	if specStatus == "" {
		specStatus = "draft"
	}
	if specStatus != "approved" {
		return nil, fmt.Errorf("spec %s is %q; approve the spec before creating stories", spec.Path, specStatus)
	}
	note, err := m.notes.Create(notes.CreateInput{
		Title:    title,
		NoteType: "story",
		Template: "story.md",
		Section:  ".brain",
		Subdir:   "planning/stories",
		Metadata: map[string]any{
			"project": info.Name,
			"epic":    slugFromPath(epic.Path),
			"spec":    slugFromPath(spec.Path),
			"status":  "todo",
		},
	})
	if err != nil {
		return nil, err
	}
	if description == "" && len(criteria) == 0 && len(resources) == 0 {
		return note, nil
	}
	return m.applyStoryUpdates(note.Path, StoryChanges{
		AddCriteria:  criteria,
		AddResources: append([]string{fmt.Sprintf("[[%s]]", spec.Path)}, resources...),
	}, description)
}

func (m *Manager) UpdateStory(storySlug string, changes StoryChanges) (*notes.Note, error) {
	info, err := m.requirePlanning()
	if err != nil {
		return nil, err
	}
	storyPath := filepath.ToSlash(filepath.Join(info.StoriesDir, storySlug+".md"))
	meta := map[string]any{}
	if changes.Status != "" {
		if !isValidStoryStatus(changes.Status) {
			return nil, fmt.Errorf("invalid story status %q (expected: todo, in_progress, blocked, done)", changes.Status)
		}
		meta["status"] = changes.Status
	}
	if len(changes.AddCriteria) == 0 && len(changes.AddResources) == 0 {
		return m.notes.Update(storyPath, notes.UpdateInput{Metadata: meta, Summary: "updated story"})
	}
	note, err := m.applyStoryUpdates(storyPath, changes, "")
	if err != nil {
		return nil, err
	}
	if len(meta) == 0 {
		return note, nil
	}
	return m.notes.Update(storyPath, notes.UpdateInput{Metadata: meta, Summary: "updated story"})
}

func (m *Manager) ListStories(filterEpic, filterStatus string) ([]StoryInfo, error) {
	info, err := m.requirePlanning()
	if err != nil {
		return nil, err
	}
	stories, err := m.listStoriesInDir(info.StoriesDir)
	if err != nil {
		return nil, err
	}
	filterEpic = slugify(filterEpic)
	var filtered []StoryInfo
	for _, story := range stories {
		if filterEpic != "" && story.Epic != filterEpic {
			continue
		}
		if filterStatus != "" && story.Status != filterStatus {
			continue
		}
		filtered = append(filtered, story)
	}
	return filtered, nil
}

func (m *Manager) Status() (*ProjectStatus, error) {
	info, err := m.requirePlanning()
	if err != nil {
		return nil, err
	}
	epics, err := m.ListEpics()
	if err != nil {
		return nil, err
	}
	stories, err := m.listStoriesInDir(info.StoriesDir)
	if err != nil {
		return nil, err
	}
	status := &ProjectStatus{
		Project:       info.Name,
		PlanningModel: info.PlanningModel,
		Epics:         epics,
		TotalStories:  len(stories),
	}
	for _, story := range stories {
		switch story.Status {
		case "done":
			status.DoneStories++
		case "blocked":
			status.BlockedStories++
		case "in_progress":
			status.InProgressStories++
		}
	}
	return status, nil
}

func (m *Manager) PromoteBrainstorm(brainstormSlug string) (*EpicBundle, error) {
	info, err := m.requirePlanning()
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
	bundle, err := m.CreateEpic(trimBrainstormTitle(bs.Title), bs.Path)
	if err != nil {
		return nil, err
	}
	spec, err := m.seedSpecFromBrainstorm(bundle.Spec, bs)
	if err != nil {
		return nil, err
	}
	bundle.Spec = spec
	return bundle, nil
}

func (m *Manager) requirePlanning() (*project.ProjectInfo, error) {
	info, err := m.project.EnsurePlanningLayout()
	if err != nil {
		return nil, err
	}
	if err := m.ensureEpicSpecMigration(info); err != nil {
		return nil, err
	}
	return info, nil
}

func (m *Manager) createSpecForEpic(info *project.ProjectInfo, epic *notes.Note, sourceBrainstorm string) (*notes.Note, error) {
	specSlug := stringValue(epic.Metadata["spec"])
	if specSlug == "" {
		specSlug = slugFromPath(epic.Path)
	}
	spec, err := m.notes.Create(notes.CreateInput{
		Title:    epic.Title + " Spec",
		Filename: specSlug,
		NoteType: "spec",
		Template: "spec.md",
		Section:  ".brain",
		Subdir:   "planning/specs",
		Metadata: map[string]any{
			"project": info.Name,
			"epic":    slugFromPath(epic.Path),
			"status":  "draft",
		},
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sourceBrainstorm) == "" {
		return spec, nil
	}
	content := notes.AppendUnderHeading(spec.Content, "Resources", fmt.Sprintf("- [[%s]]", sourceBrainstorm))
	return m.notes.Update(spec.Path, notes.UpdateInput{
		Body:    &content,
		Summary: "linked source brainstorm",
	})
}

func (m *Manager) readSpecForEpic(epicSlug string) (*notes.Note, error) {
	epic, err := m.ReadEpic(epicSlug)
	if err != nil {
		return nil, err
	}
	specSlug := stringValue(epic.Metadata["spec"])
	if specSlug == "" {
		specSlug = slugFromPath(epic.Path)
	}
	return m.notes.Read(m.specPath(specSlug))
}

func (m *Manager) seedSpecFromBrainstorm(spec *notes.Note, brainstorm *notes.Note) (*notes.Note, error) {
	content := spec.Content
	focus := extractSection(brainstorm.Content, "Focus Question")
	if strings.TrimSpace(focus) != "" && !strings.Contains(content, strings.TrimSpace(focus)) {
		content = notes.AppendUnderHeading(content, "Why", strings.TrimSpace(focus))
		content = notes.AppendUnderHeading(content, "Problem", strings.TrimSpace(focus))
	}
	ideas := extractIdeas(brainstorm.Content)
	for _, idea := range ideas {
		content = notes.AppendUnderHeading(content, "Goals", fmt.Sprintf("- %s", idea))
	}
	content = notes.AppendUnderHeading(content, "Resources", fmt.Sprintf("- [[%s]]", brainstorm.Path))
	content = notes.AppendUnderHeading(content, "Story Breakdown", "- [ ] Break approved spec into execution stories")
	return m.notes.Update(spec.Path, notes.UpdateInput{
		Body:    &content,
		Summary: "seeded spec from brainstorm",
	})
}

func (m *Manager) ensureEpicSpecMigration(info *project.ProjectInfo) error {
	epics, err := m.readDir(info.EpicsDir)
	if err != nil {
		return err
	}
	for _, epic := range epics {
		epicSlug := slugFromPath(epic.Path)
		specSlug := stringValue(epic.Metadata["spec"])
		if specSlug == "" {
			specSlug = epicSlug
			if _, err := m.notes.Update(epic.Path, notes.UpdateInput{
				Metadata: map[string]any{"spec": specSlug},
				Summary:  "linked canonical spec",
			}); err != nil {
				return err
			}
		}
		specPath := m.specPath(specSlug)
		spec, err := m.notes.Read(specPath)
		if err != nil {
			if !strings.Contains(err.Error(), "no such file") {
				return err
			}
			spec, err = m.createSpecForEpic(info, epic, stringValue(epic.Metadata["source_brainstorm"]))
			if err != nil {
				return err
			}
		}
		meta := map[string]any{}
		if stringValue(spec.Metadata["epic"]) == "" {
			meta["epic"] = epicSlug
		}
		if stringValue(spec.Metadata["status"]) == "" {
			meta["status"] = "draft"
		}
		if len(meta) > 0 {
			spec, err = m.notes.Update(spec.Path, notes.UpdateInput{
				Metadata: meta,
				Summary:  "normalized spec metadata",
			})
			if err != nil {
				return err
			}
		}
		epicContent := epic.Content
		specLink := fmt.Sprintf("- [[%s]]", spec.Path)
		if !strings.Contains(epicContent, specLink) {
			epicContent = notes.AppendUnderHeading(epicContent, "Spec", specLink)
		}
		if source := stringValue(epic.Metadata["source_brainstorm"]); source != "" {
			sourceLink := fmt.Sprintf("- [[%s]]", source)
			if !strings.Contains(epicContent, sourceLink) {
				epicContent = notes.AppendUnderHeading(epicContent, "Sources", sourceLink)
			}
		}
		if epicContent != epic.Content {
			if _, err := m.notes.Update(epic.Path, notes.UpdateInput{
				Body:    &epicContent,
				Summary: "normalized epic structure",
			}); err != nil {
				return err
			}
		}
	}
	stories, err := m.readDir(info.StoriesDir)
	if err != nil {
		return err
	}
	for _, story := range stories {
		meta := map[string]any{}
		epicSlug := stringValue(story.Metadata["epic"])
		if epicSlug == "" {
			epicSlug = stringValue(story.Metadata["container"])
			if epicSlug != "" {
				meta["epic"] = epicSlug
			}
		}
		if epicSlug != "" && stringValue(story.Metadata["spec"]) == "" {
			meta["spec"] = epicSlug
		}
		content := story.Content
		if epicSlug != "" {
			specLink := fmt.Sprintf("- [[%s]]", m.specPath(epicSlug))
			if !strings.Contains(content, specLink) {
				content = notes.AppendUnderHeading(content, "Resources", specLink)
			}
		}
		if len(meta) == 0 && content == story.Content {
			continue
		}
		input := notes.UpdateInput{
			Summary: "migrated story to epic-spec model",
		}
		if len(meta) > 0 {
			input.Metadata = meta
		}
		if content != story.Content {
			input.Body = &content
		}
		if _, err := m.notes.Update(story.Path, input); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) applyStoryUpdates(storyPath string, changes StoryChanges, description string) (*notes.Note, error) {
	note, err := m.notes.Read(storyPath)
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
	return m.notes.Update(storyPath, notes.UpdateInput{Body: &content, Summary: "updated story"})
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

func (m *Manager) listStoriesInDir(relDir string) ([]StoryInfo, error) {
	noteFiles, err := m.readDir(relDir)
	if err != nil {
		return nil, err
	}
	var stories []StoryInfo
	for _, note := range noteFiles {
		status := stringValue(note.Metadata["status"])
		if status == "" {
			status = "todo"
		}
		epic := stringValue(note.Metadata["epic"])
		if epic == "" {
			epic = stringValue(note.Metadata["container"])
		}
		spec := stringValue(note.Metadata["spec"])
		if spec == "" {
			spec = epic
		}
		stories = append(stories, StoryInfo{
			Path:   note.Path,
			Title:  note.Title,
			Status: status,
			Epic:   epic,
			Spec:   spec,
		})
	}
	return stories, nil
}

func (m *Manager) epicPath(epicSlug string) string {
	return filepath.ToSlash(filepath.Join(".brain/planning/epics", slugify(epicSlug)+".md"))
}

func (m *Manager) specPath(specSlug string) string {
	return filepath.ToSlash(filepath.Join(".brain/planning/specs", slugify(specSlug)+".md"))
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

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func isValidSpecStatus(status string) bool {
	_, ok := validSpecStatuses[status]
	return ok
}

func isValidStoryStatus(status string) bool {
	_, ok := validStoryStatuses[status]
	return ok
}

func trimBrainstormTitle(title string) string {
	title = strings.TrimSpace(title)
	return strings.TrimPrefix(title, "Brainstorm: ")
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
			trimmed := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(trimmed, "## "):
				inIdeas = strings.EqualFold(strings.TrimPrefix(trimmed, "## "), "Ideas")
			case inIdeas && strings.HasPrefix(trimmed, "- "):
				idea := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				if idea != "" {
					ideas = append(ideas, idea)
				}
			}
		}
	}
	return ideas
}

func extractSection(content, heading string) string {
	lines := strings.Split(content, "\n")
	var out []string
	inSection := false
	target := strings.ToLower(strings.TrimSpace(heading))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			current := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
			if inSection {
				break
			}
			inSection = current == target
			continue
		}
		if inSection {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
