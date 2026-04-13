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
	ProjectDir       string
	Task             string
	TaskSource       string
	HasActiveSession bool
	Limit            int
	Explain          bool
	SearchResults    []search.Result
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
	group  string
	item   Item
	score  float64
	key    string
	method string
	notes  []string
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

type candidateGroups struct {
	DurableNotes     []candidate
	GeneratedContext []candidate
	StructuralRepo   []candidate
	LiveWork         []candidate
	PolicyWorkflow   []candidate
}

type selectionPlan struct {
	Selected candidateGroups
	Omitted  candidateGroups
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

	limit := req.Limit
	if limit <= 0 {
		limit = 8
	}

	plan := selectCandidates(task, req.ProjectDir, limit, req.SearchResults)
	selected := plan.Selected.items(req.Explain)
	omitted := plan.Omitted.items(req.Explain)
	ambiguities := buildAmbiguities(req, plan)

	return &Packet{
		Task: TaskInfo{
			Text:   task,
			Source: taskSource,
		},
		Summary: Summary{
			Confidence:    computeConfidence(selected, ambiguities),
			SelectedCount: totalItems(selected),
			GroupCounts: GroupCounts{
				DurableNotes:     len(selected.DurableNotes),
				GeneratedContext: len(selected.GeneratedContext),
				StructuralRepo:   len(selected.StructuralRepo),
				LiveWork:         len(selected.LiveWork),
				PolicyWorkflow:   len(selected.PolicyWorkflow),
			},
		},
		Selected:      selected,
		Ambiguities:   ambiguities,
		OmittedNearby: omitted,
	}, nil
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
			if err := renderSelectedGroup(w, entry.label, entry.items); err != nil {
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

	if explain {
		if err := renderExplainSections(w, packet); err != nil {
			return err
		}
	}
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

func newCandidateGroups() candidateGroups {
	return candidateGroups{
		DurableNotes:     []candidate{},
		GeneratedContext: []candidate{},
		StructuralRepo:   []candidate{},
		LiveWork:         []candidate{},
		PolicyWorkflow:   []candidate{},
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

func selectCandidates(task, projectDir string, limit int, searchResults []search.Result) selectionPlan {
	plan := selectionPlan{
		Selected: newCandidateGroups(),
		Omitted:  newCandidateGroups(),
	}
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
	selectedCount := 0
	for _, group := range []string{"durable_notes", "generated_context", "policy_workflow"} {
		groupSelected := 0
		for _, entry := range grouped[group] {
			if _, ok := seen[entry.key]; ok {
				continue
			}
			if selectedCount < limit && groupSelected < groupCaps[group] {
				addCandidate(&plan.Selected, group, entry)
				seen[entry.key] = struct{}{}
				selectedCount++
				groupSelected++
				continue
			}
			addCandidate(&plan.Omitted, group, entry)
		}
	}

	if selectedCount < limit {
		var remaining []candidate
		for _, group := range []string{"durable_notes", "generated_context", "policy_workflow"} {
			for _, entry := range grouped[group] {
				if _, ok := seen[entry.key]; ok {
					continue
				}
				remaining = append(remaining, entry)
			}
		}
		sortCandidates(remaining)
		for _, entry := range remaining {
			if _, ok := seen[entry.key]; ok {
				continue
			}
			if selectedCount < limit {
				addCandidate(&plan.Selected, entry.group, entry)
				seen[entry.key] = struct{}{}
				selectedCount++
				continue
			}
			addCandidate(&plan.Omitted, entry.group, entry)
		}
	}

	plan.Omitted.limitPerGroup(2)
	return plan
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
		candidates = append(candidates, candidate{
			group:  "durable_notes",
			score:  result.Score,
			key:    result.NotePath + "#" + strings.TrimSpace(result.Heading),
			method: "search",
			notes:  []string{"matched task terms", "high search rank"},
			item: Item{
				Source:  result.NotePath,
				Label:   label,
				Kind:    "note",
				Excerpt: compactSnippet(result.Snippet),
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
		overlap := overlapScore(taskTokens, strings.Join([]string{spec.Path, spec.Label, content}, " "))
		why := spec.DefaultWhy
		notes := []string{"deterministic context source"}
		if overlap > 0 {
			why = "matched task terms in " + strings.ToLower(spec.Label)
			notes = []string{"matched task terms", "deterministic context source"}
		}
		candidates = append(candidates, candidate{
			group:  spec.Group,
			score:  spec.BaseScore + overlap,
			key:    spec.Path,
			method: "deterministic",
			notes:  notes,
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
	_, body, err := notes.ParseFrontmatter(content)
	if err == nil {
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

func addCandidate(groups *candidateGroups, group string, entry candidate) {
	switch group {
	case "durable_notes":
		groups.DurableNotes = append(groups.DurableNotes, entry)
	case "generated_context":
		groups.GeneratedContext = append(groups.GeneratedContext, entry)
	case "policy_workflow":
		groups.PolicyWorkflow = append(groups.PolicyWorkflow, entry)
	case "structural_repo":
		groups.StructuralRepo = append(groups.StructuralRepo, entry)
	case "live_work":
		groups.LiveWork = append(groups.LiveWork, entry)
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

func (groups candidateGroups) items(explain bool) GroupedItems {
	out := newGroupedItems()
	out.DurableNotes = candidatesToItems("durable_notes", groups.DurableNotes, explain)
	out.GeneratedContext = candidatesToItems("generated_context", groups.GeneratedContext, explain)
	out.StructuralRepo = candidatesToItems("structural_repo", groups.StructuralRepo, explain)
	out.LiveWork = candidatesToItems("live_work", groups.LiveWork, explain)
	out.PolicyWorkflow = candidatesToItems("policy_workflow", groups.PolicyWorkflow, explain)
	return out
}

func (groups *candidateGroups) limitPerGroup(limit int) {
	if limit <= 0 {
		return
	}
	for _, items := range []*[]candidate{
		&groups.DurableNotes,
		&groups.GeneratedContext,
		&groups.StructuralRepo,
		&groups.LiveWork,
		&groups.PolicyWorkflow,
	} {
		if len(*items) > limit {
			*items = (*items)[:limit]
		}
	}
}

func candidatesToItems(group string, candidates []candidate, explain bool) []Item {
	items := make([]Item, 0, len(candidates))
	for i, entry := range candidates {
		item := entry.item
		if explain {
			item.Rank = i + 1
			item.SelectionMethod = entry.method
			item.Diagnostics = &ItemDiagnostics{
				SourceGroup: group,
				Notes:       append([]string(nil), entry.notes...),
			}
		}
		items = append(items, item)
	}
	return items
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
	snippet := strings.TrimSpace(strings.Join(paragraph, " "))
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

func buildAmbiguities(req Request, plan selectionPlan) []string {
	ambiguities := []string{}
	selected := plan.Selected.items(false)
	if !req.HasActiveSession && req.TaskSource == "flag" {
		ambiguities = append(ambiguities, "using explicit task text without an active session")
	}
	if countNonEmptyGroups(selected) <= 1 {
		ambiguities = append(ambiguities, "only one source group provided useful context")
	}
	if hasCompetitiveDurableNearby(plan) {
		ambiguities = append(ambiguities, "multiple nearby durable notes compete for the same role")
	}
	return ambiguities
}

func hasCompetitiveDurableNearby(plan selectionPlan) bool {
	if len(plan.Selected.DurableNotes) == 0 || len(plan.Omitted.DurableNotes) == 0 {
		return false
	}
	lastSelected := plan.Selected.DurableNotes[len(plan.Selected.DurableNotes)-1]
	firstOmitted := plan.Omitted.DurableNotes[0]
	return (lastSelected.score - firstOmitted.score) <= 0.10
}

func computeConfidence(selected GroupedItems, ambiguities []string) string {
	groupCount := countNonEmptyGroups(selected)
	ambiguityCount := len(ambiguities)
	switch {
	case groupCount >= 3 && ambiguityCount == 0:
		return "high"
	case groupCount <= 1 || ambiguityCount >= 2:
		return "low"
	case groupCount == 2 || ambiguityCount == 1:
		return "medium"
	default:
		return "low"
	}
}

func renderSelectedGroup(w io.Writer, label string, items []Item) error {
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

func renderExplainSections(w io.Writer, packet *Packet) error {
	if _, err := io.WriteString(w, "\n## Why This Was Selected\n\n"); err != nil {
		return err
	}
	for _, entry := range []struct {
		label string
		items []Item
	}{
		{label: "Durable Notes", items: packet.Selected.DurableNotes},
		{label: "Generated Context", items: packet.Selected.GeneratedContext},
		{label: "Policy Workflow", items: packet.Selected.PolicyWorkflow},
		{label: "Structural Repo", items: packet.Selected.StructuralRepo},
		{label: "Live Work", items: packet.Selected.LiveWork},
	} {
		if err := renderExplainGroup(w, entry.label, entry.items); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(w, "\n## Omitted Nearby Context\n\n"); err != nil {
		return err
	}
	if totalItems(packet.OmittedNearby) == 0 {
		if _, err := io.WriteString(w, "- No omitted nearby context.\n"); err != nil {
			return err
		}
	} else {
		for _, entry := range []struct {
			label string
			items []Item
		}{
			{label: "Durable Notes", items: packet.OmittedNearby.DurableNotes},
			{label: "Generated Context", items: packet.OmittedNearby.GeneratedContext},
			{label: "Policy Workflow", items: packet.OmittedNearby.PolicyWorkflow},
			{label: "Structural Repo", items: packet.OmittedNearby.StructuralRepo},
			{label: "Live Work", items: packet.OmittedNearby.LiveWork},
		} {
			if err := renderExplainGroup(w, entry.label, entry.items); err != nil {
				return err
			}
		}
	}

	if _, err := io.WriteString(w, "\n## Missing Or Unused Source Groups\n\n"); err != nil {
		return err
	}
	for _, entry := range []struct {
		label string
		count int
	}{
		{label: "Durable Notes", count: len(packet.Selected.DurableNotes)},
		{label: "Generated Context", count: len(packet.Selected.GeneratedContext)},
		{label: "Policy Workflow", count: len(packet.Selected.PolicyWorkflow)},
		{label: "Structural Repo", count: len(packet.Selected.StructuralRepo)},
		{label: "Live Work", count: len(packet.Selected.LiveWork)},
	} {
		if entry.count == 0 {
			if _, err := fmt.Fprintf(w, "- %s\n", entry.label); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderExplainGroup(w io.Writer, label string, items []Item) error {
	if len(items) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(w, "### %s\n\n", label); err != nil {
		return err
	}
	for _, item := range items {
		line := fmt.Sprintf("- %s (`%s`): %s", item.Label, item.Source, item.Why)
		if item.Diagnostics != nil && len(item.Diagnostics.Notes) > 0 {
			line += " [" + strings.Join(item.Diagnostics.Notes, ", ") + "]"
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
