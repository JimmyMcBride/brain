package contextassembly

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"brain/internal/notes"
	"brain/internal/projectcontext"
	"brain/internal/search"
)

type Manager struct {
	Context *projectcontext.Manager
}

type Request struct {
	ProjectDir    string
	Task          string
	TaskSource    string
	Limit         int
	Explain       bool
	SearchResults []search.Result
}

type Packet struct {
	Task          TaskInfo     `json:"task"`
	Summary       Summary      `json:"summary"`
	Selected      GroupedItems `json:"selected"`
	Ambiguities   []string     `json:"ambiguities"`
	OmittedNearby GroupedItems `json:"omitted_nearby"`
}

type TaskInfo struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

type Summary struct {
	Confidence    string      `json:"confidence"`
	SelectedCount int         `json:"selected_count"`
	GroupCounts   GroupCounts `json:"group_counts"`
}

type GroupCounts struct {
	DurableNotes     int `json:"durable_notes"`
	GeneratedContext int `json:"generated_context"`
	StructuralRepo   int `json:"structural_repo"`
	LiveWork         int `json:"live_work"`
	PolicyWorkflow   int `json:"policy_workflow"`
}

type GroupedItems struct {
	DurableNotes     []Item `json:"durable_notes"`
	GeneratedContext []Item `json:"generated_context"`
	StructuralRepo   []Item `json:"structural_repo"`
	LiveWork         []Item `json:"live_work"`
	PolicyWorkflow   []Item `json:"policy_workflow"`
}

type Item struct {
	Source          string           `json:"source"`
	Label           string           `json:"label"`
	Kind            string           `json:"kind"`
	Excerpt         string           `json:"excerpt"`
	Why             string           `json:"why"`
	Rank            int              `json:"rank,omitempty"`
	SelectionMethod string           `json:"selection_method,omitempty"`
	Diagnostics     *ItemDiagnostics `json:"diagnostics,omitempty"`
}

type ItemDiagnostics struct {
	SourceGroup string   `json:"source_group"`
	Notes       []string `json:"notes,omitempty"`
}

type candidate struct {
	group string
	item  Item
	score float64
	key   string
}

type staticSource struct {
	Path         string
	Label        string
	Group        string
	Kind         string
	BaseScore    float64
	DefaultWhy   string
	SectionTitle string
}

var taskTokenPattern = regexp.MustCompile(`[[:alnum:]_]+`)

var generatedContextSources = []staticSource{
	{Path: ".brain/context/current-state.md", Label: "Current State", Group: "generated_context", Kind: "generated_context", BaseScore: 0.040, DefaultWhy: "default current-state context for task assembly"},
	{Path: ".brain/context/overview.md", Label: "Overview", Group: "generated_context", Kind: "generated_context", BaseScore: 0.030, DefaultWhy: "default overview context for task assembly"},
	{Path: ".brain/context/architecture.md", Label: "Architecture", Group: "generated_context", Kind: "generated_context", BaseScore: 0.020, DefaultWhy: "default architecture context for task assembly"},
	{Path: ".brain/context/standards.md", Label: "Standards", Group: "generated_context", Kind: "generated_context", BaseScore: 0.010, DefaultWhy: "default standards context for task assembly"},
}

var policyWorkflowSources = []staticSource{
	{Path: "AGENTS.md", Label: "Required Workflow", Group: "policy_workflow", Kind: "policy", BaseScore: 0.040, DefaultWhy: "default required workflow guidance for task assembly", SectionTitle: "Required Workflow"},
	{Path: ".brain/context/workflows.md", Label: "Workflow Guidance", Group: "policy_workflow", Kind: "policy", BaseScore: 0.030, DefaultWhy: "default workflow guidance for task assembly"},
	{Path: ".brain/context/memory-policy.md", Label: "Memory Policy", Group: "policy_workflow", Kind: "policy", BaseScore: 0.020, DefaultWhy: "default memory guidance for task assembly"},
	{Path: ".brain/policy.yaml", Label: "Policy", Group: "policy_workflow", Kind: "policy", BaseScore: 0.010, DefaultWhy: "default policy contract for task assembly"},
}

func New(contextManager *projectcontext.Manager) *Manager {
	return &Manager{Context: contextManager}
}

func (m *Manager) Assemble(req Request) (*Packet, error) {
	task := strings.TrimSpace(req.Task)
	if task == "" {
		return nil, errors.New("task context assembly requires a task")
	}
	taskSource := strings.TrimSpace(req.TaskSource)
	if taskSource == "" {
		taskSource = "flag"
	}
	packet := &Packet{
		Task: TaskInfo{
			Text:   task,
			Source: taskSource,
		},
		Summary: Summary{
			Confidence:    "low",
			SelectedCount: 0,
			GroupCounts:   GroupCounts{},
		},
		Selected:      newGroupedItems(),
		Ambiguities:   []string{},
		OmittedNearby: newGroupedItems(),
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 8
	}
	selected := selectCandidates(task, req.ProjectDir, limit, req.SearchResults)
	packet.Selected = selected
	packet.Summary.SelectedCount = totalItems(selected)
	packet.Summary.GroupCounts = GroupCounts{
		DurableNotes:     len(selected.DurableNotes),
		GeneratedContext: len(selected.GeneratedContext),
		StructuralRepo:   len(selected.StructuralRepo),
		LiveWork:         len(selected.LiveWork),
		PolicyWorkflow:   len(selected.PolicyWorkflow),
	}
	return packet, nil
}

func RenderHuman(w io.Writer, packet *Packet, explain bool) error {
	if packet == nil {
		return errors.New("task context packet is required")
	}
	groupCount := countNonEmptyGroups(packet.Selected)
	if _, err := fmt.Fprintf(w, "## Task Context\n\n- Task: `%s`\n- Source: `%s`\n- Confidence: `%s`\n- Selected: %d item(s) across %d group(s)\n\n",
		packet.Task.Text,
		packet.Task.Source,
		packet.Summary.Confidence,
		packet.Summary.SelectedCount,
		groupCount,
	); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "## Selected Context\n\n"); err != nil {
		return err
	}
	if packet.Summary.SelectedCount == 0 {
		if _, err := io.WriteString(w, "- No selected context yet.\n"); err != nil {
			return err
		}
	} else {
		writeGroup := func(label string, items []Item) error {
			if len(items) == 0 {
				return nil
			}
			if _, err := fmt.Fprintf(w, "### %s\n\n", label); err != nil {
				return err
			}
			for _, item := range items {
				line := fmt.Sprintf("- %s (`%s`)", item.Label, item.Source)
				if strings.TrimSpace(item.Excerpt) != "" {
					line += ": " + item.Excerpt
				} else if strings.TrimSpace(item.Why) != "" {
					line += ": " + item.Why
				}
				if _, err := fmt.Fprintln(w, line); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
			return nil
		}
		for _, entry := range []struct {
			label string
			items []Item
		}{
			{label: "Durable Notes", items: packet.Selected.DurableNotes},
			{label: "Generated Context", items: packet.Selected.GeneratedContext},
			{label: "Structural Repo", items: packet.Selected.StructuralRepo},
			{label: "Live Work", items: packet.Selected.LiveWork},
			{label: "Policy Workflow", items: packet.Selected.PolicyWorkflow},
		} {
			if err := writeGroup(entry.label, entry.items); err != nil {
				return err
			}
		}
	}
	if len(packet.Ambiguities) > 0 {
		if _, err := io.WriteString(w, "\n## Ambiguities\n\n"); err != nil {
			return err
		}
		for _, ambiguity := range packet.Ambiguities {
			if _, err := fmt.Fprintf(w, "- %s\n", ambiguity); err != nil {
				return err
			}
		}
	}
	_ = explain
	return nil
}

func newGroupedItems() GroupedItems {
	return GroupedItems{
		DurableNotes:     []Item{},
		GeneratedContext: []Item{},
		StructuralRepo:   []Item{},
		LiveWork:         []Item{},
		PolicyWorkflow:   []Item{},
	}
}

func countNonEmptyGroups(groups GroupedItems) int {
	count := 0
	for _, items := range [][]Item{
		groups.DurableNotes,
		groups.GeneratedContext,
		groups.StructuralRepo,
		groups.LiveWork,
		groups.PolicyWorkflow,
	} {
		if len(items) > 0 {
			count++
		}
	}
	return count
}

func totalItems(groups GroupedItems) int {
	return len(groups.DurableNotes) + len(groups.GeneratedContext) + len(groups.StructuralRepo) + len(groups.LiveWork) + len(groups.PolicyWorkflow)
}

func selectCandidates(task, projectDir string, limit int, searchResults []search.Result) GroupedItems {
	selected := newGroupedItems()
	taskTokens := tokenize(task)
	grouped := map[string][]candidate{
		"durable_notes":     durableNoteCandidates(searchResults),
		"generated_context": staticCandidates(projectDir, taskTokens, generatedContextSources),
		"policy_workflow":   staticCandidates(projectDir, taskTokens, policyWorkflowSources),
	}
	groupCaps := map[string]int{
		"durable_notes":     3,
		"generated_context": 2,
		"policy_workflow":   2,
	}

	seen := map[string]struct{}{}
	for _, group := range []string{"durable_notes", "generated_context", "policy_workflow"} {
		candidates := grouped[group]
		selectedCount := 0
		for _, candidate := range candidates {
			if totalItems(selected) >= limit || selectedCount >= groupCaps[group] {
				break
			}
			if _, ok := seen[candidate.key]; ok {
				continue
			}
			addItem(&selected, group, candidate.item)
			seen[candidate.key] = struct{}{}
			selectedCount++
		}
	}

	if totalItems(selected) < limit {
		var remaining []candidate
		for _, group := range []string{"durable_notes", "generated_context", "policy_workflow"} {
			for _, candidate := range grouped[group] {
				if _, ok := seen[candidate.key]; ok {
					continue
				}
				remaining = append(remaining, candidate)
			}
		}
		sortCandidates(remaining)
		for _, candidate := range remaining {
			if totalItems(selected) >= limit {
				break
			}
			if _, ok := seen[candidate.key]; ok {
				continue
			}
			addItem(&selected, candidate.group, candidate.item)
			seen[candidate.key] = struct{}{}
		}
	}

	return selected
}

func durableNoteCandidates(results []search.Result) []candidate {
	candidates := make([]candidate, 0, len(results))
	for _, result := range results {
		if classifyResultGroup(result.NotePath) != "durable_notes" {
			continue
		}
		label := strings.TrimSpace(result.NoteTitle)
		if label == "" && strings.TrimSpace(result.Heading) != "" {
			label = strings.TrimSpace(result.Heading)
		}
		if label == "" {
			label = filepath.Base(result.NotePath)
		}
		excerpt := compactSnippet(result.Snippet)
		candidates = append(candidates, candidate{
			group: "durable_notes",
			score: result.Score,
			key:   result.NotePath + "#" + strings.TrimSpace(result.Heading),
			item: Item{
				Source:  result.NotePath,
				Label:   label,
				Kind:    "note",
				Excerpt: excerpt,
				Why:     "high search rank for the task",
			},
		})
	}
	sortCandidates(candidates)
	return candidates
}

func staticCandidates(projectDir string, taskTokens map[string]struct{}, specs []staticSource) []candidate {
	candidates := make([]candidate, 0, len(specs))
	for _, spec := range specs {
		content, err := loadStaticSource(projectDir, spec)
		if err != nil {
			continue
		}
		score := spec.BaseScore + overlapScore(taskTokens, strings.Join([]string{spec.Path, spec.Label, content}, " "))
		why := spec.DefaultWhy
		if overlap := overlapScore(taskTokens, strings.Join([]string{spec.Path, spec.Label, content}, " ")); overlap > 0 {
			why = "matched task terms in " + strings.ToLower(spec.Label)
		}
		candidates = append(candidates, candidate{
			group: spec.Group,
			score: score,
			key:   spec.Path,
			item: Item{
				Source:  spec.Path,
				Label:   spec.Label,
				Kind:    spec.Kind,
				Excerpt: compactSnippet(content),
				Why:     why,
			},
		})
	}
	sortCandidates(candidates)
	return candidates
}

func loadStaticSource(projectDir string, spec staticSource) (string, error) {
	raw, err := os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(spec.Path)))
	if err != nil {
		return "", err
	}
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if spec.Path == ".brain/policy.yaml" {
		return strings.TrimSpace(content), nil
	}
	meta, body, err := notes.ParseFrontmatter(content)
	if err == nil && len(meta) >= 0 {
		content = body
	}
	content = strings.TrimSpace(content)
	if spec.SectionTitle != "" {
		if section := extractMarkdownSection(content, spec.SectionTitle); section != "" {
			return section, nil
		}
	}
	return content, nil
}

func classifyResultGroup(path string) string {
	switch {
	case path == "AGENTS.md":
		return ""
	case isGeneratedContextPath(path):
		return ""
	case path == ".brain/context/workflows.md", path == ".brain/context/memory-policy.md", path == ".brain/policy.yaml":
		return ""
	case strings.HasPrefix(path, "docs/"):
		return "durable_notes"
	case strings.HasPrefix(path, ".brain/") && !strings.HasPrefix(path, ".brain/context/"):
		return "durable_notes"
	default:
		return ""
	}
}

func isGeneratedContextPath(path string) bool {
	switch path {
	case ".brain/context/current-state.md", ".brain/context/overview.md", ".brain/context/architecture.md", ".brain/context/standards.md":
		return true
	default:
		return false
	}
}

func addItem(groups *GroupedItems, group string, item Item) {
	switch group {
	case "durable_notes":
		groups.DurableNotes = append(groups.DurableNotes, item)
	case "generated_context":
		groups.GeneratedContext = append(groups.GeneratedContext, item)
	case "policy_workflow":
		groups.PolicyWorkflow = append(groups.PolicyWorkflow, item)
	case "structural_repo":
		groups.StructuralRepo = append(groups.StructuralRepo, item)
	case "live_work":
		groups.LiveWork = append(groups.LiveWork, item)
	}
}

func sortCandidates(candidates []candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			if candidates[i].group == candidates[j].group {
				return candidates[i].item.Source < candidates[j].item.Source
			}
			return candidates[i].group < candidates[j].group
		}
		return candidates[i].score > candidates[j].score
	})
}

func tokenize(text string) map[string]struct{} {
	matches := taskTokenPattern.FindAllString(strings.ToLower(text), -1)
	if len(matches) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		out[match] = struct{}{}
	}
	return out
}

func overlapScore(taskTokens map[string]struct{}, text string) float64 {
	if len(taskTokens) == 0 {
		return 0
	}
	candidateTokens := tokenize(text)
	if len(candidateTokens) == 0 {
		return 0
	}
	matches := 0
	for token := range taskTokens {
		if _, ok := candidateTokens[token]; ok {
			matches++
		}
	}
	if matches == 0 {
		return 0
	}
	return float64(matches) / float64(len(taskTokens))
}

func compactSnippet(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	lines := strings.Split(content, "\n")
	var paragraph []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "<!--") {
			continue
		}
		paragraph = append(paragraph, line)
		if len(strings.Join(paragraph, " ")) >= 160 {
			break
		}
	}
	if len(paragraph) == 0 {
		paragraph = append(paragraph, lines[0])
	}
	snippet := strings.Join(paragraph, " ")
	snippet = strings.TrimSpace(snippet)
	if len(snippet) > 180 {
		snippet = strings.TrimSpace(snippet[:177]) + "..."
	}
	return snippet
}

func extractMarkdownSection(content, heading string) string {
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
