package projectcontext

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ContextItemKind string

const (
	ContextItemKindBaseContract       ContextItemKind = "base_contract"
	ContextItemKindDurableNote        ContextItemKind = "durable_note"
	ContextItemKindGeneratedContext   ContextItemKind = "generated_context"
	ContextItemKindWorkflowRule       ContextItemKind = "workflow_rule"
	ContextItemKindVerificationRecipe ContextItemKind = "verification_recipe"
)

type ContextAnchor struct {
	Path    string `json:"path"`
	Section string `json:"section,omitempty"`
}

type ContextItem struct {
	ID              string          `json:"id"`
	Kind            ContextItemKind `json:"kind"`
	Title           string          `json:"title"`
	Summary         string          `json:"summary"`
	Anchor          ContextAnchor   `json:"anchor"`
	Boundaries      []string        `json:"boundaries,omitempty"`
	Files           []string        `json:"files,omitempty"`
	SourceHash      string          `json:"source_hash"`
	ExpansionCost   int             `json:"expansion_cost"`
	EstimatedTokens int             `json:"estimated_tokens"`
}

func (m *Manager) BuildBaseContractItems(projectDir string) ([]ContextItem, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return nil, err
	}

	agents, err := readProjectContextFile(projectDir, "AGENTS.md")
	if err != nil {
		return nil, err
	}
	workflows, err := readProjectContextFile(projectDir, ".brain/context/workflows.md")
	if err != nil {
		return nil, err
	}
	memory, err := readProjectContextFile(projectDir, ".brain/context/memory-policy.md")
	if err != nil {
		return nil, err
	}
	architecture, err := readProjectContextFile(projectDir, ".brain/context/architecture.md")
	if err != nil {
		return nil, err
	}
	policy, _, _, err := LoadPolicy(projectDir)
	if err != nil {
		return nil, err
	}
	policyBody, err := readProjectContextFile(projectDir, ".brain/policy.yaml")
	if err != nil {
		return nil, err
	}

	items := []ContextItem{
		newContextItem(
			"base_boot_summary",
			ContextItemKindBaseContract,
			"Boot Summary",
			"AGENTS.md",
			"Project Agent Contract",
			agents,
			summarizeBootContract(agents),
		),
		newContextItem(
			"base_workflow_contract",
			ContextItemKindBaseContract,
			"Workflow Contract",
			"AGENTS.md",
			"Required Workflow",
			extractMarkdownSection(agents, "Required Workflow"),
			summarizeWorkflowContract(agents, workflows),
		),
		newContextItem(
			"base_memory_update_rules",
			ContextItemKindBaseContract,
			"Memory Update Rules",
			".brain/context/memory-policy.md",
			"Memory Policy",
			memory,
			summarizeMemoryRules(memory),
		),
		newContextItem(
			"base_architecture_summary",
			ContextItemKindBaseContract,
			"Architecture Summary",
			".brain/context/architecture.md",
			"Architecture Notes",
			extractMarkdownSection(architecture, "Architecture Notes"),
			summarizeArchitectureNotes(architecture),
		),
		newContextItem(
			"base_verification_summary",
			ContextItemKindBaseContract,
			"Verification Summary",
			".brain/policy.yaml",
			"closeout.verification_profiles",
			policyBody,
			summarizeVerificationPolicy(policy),
		),
	}
	return items, nil
}

func (m *Manager) BuildSourceSummaryItems(projectDir string) ([]ContextItem, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return nil, err
	}

	files := []struct {
		id      string
		kind    ContextItemKind
		title   string
		path    string
		section string
		build   func(string) string
		raw     func(string) string
	}{
		{
			id:      "source_agents_summary",
			kind:    ContextItemKindGeneratedContext,
			title:   "Project Contract Summary",
			path:    "AGENTS.md",
			section: "Project Agent Contract",
			build:   summarizeBootContract,
			raw:     func(content string) string { return content },
		},
		{
			id:      "source_overview_summary",
			kind:    ContextItemKindGeneratedContext,
			title:   "Overview Summary",
			path:    ".brain/context/overview.md",
			section: "Overview",
			build:   summarizeOverview,
			raw:     func(content string) string { return content },
		},
		{
			id:      "source_architecture_summary",
			kind:    ContextItemKindGeneratedContext,
			title:   "Architecture Notes Summary",
			path:    ".brain/context/architecture.md",
			section: "Architecture Notes",
			build:   summarizeArchitectureNotes,
			raw:     func(content string) string { return extractMarkdownSection(content, "Architecture Notes") },
		},
		{
			id:      "source_workflows_summary",
			kind:    ContextItemKindWorkflowRule,
			title:   "Workflow Guidance Summary",
			path:    ".brain/context/workflows.md",
			section: "Workflows",
			build:   summarizeWorkflowGuide,
			raw:     func(content string) string { return content },
		},
		{
			id:      "source_memory_policy_summary",
			kind:    ContextItemKindWorkflowRule,
			title:   "Memory Policy Summary",
			path:    ".brain/context/memory-policy.md",
			section: "Memory Policy",
			build:   summarizeMemoryRules,
			raw:     func(content string) string { return content },
		},
		{
			id:      "source_current_state_summary",
			kind:    ContextItemKindGeneratedContext,
			title:   "Current State Summary",
			path:    ".brain/context/current-state.md",
			section: "Repository",
			build:   summarizeCurrentState,
			raw:     func(content string) string { return extractMarkdownSection(content, "Repository") },
		},
		{
			id:      "source_policy_summary",
			kind:    ContextItemKindVerificationRecipe,
			title:   "Policy Summary",
			path:    ".brain/policy.yaml",
			section: "policy",
			build:   nil,
			raw:     func(content string) string { return content },
		},
	}

	items := make([]ContextItem, 0, len(files))
	for _, spec := range files {
		rawFile, err := readProjectContextFile(projectDir, spec.path)
		if err != nil {
			return nil, err
		}
		raw := spec.raw(rawFile)
		summary := ""
		if spec.path == ".brain/policy.yaml" {
			policy, _, _, err := LoadPolicy(projectDir)
			if err != nil {
				return nil, err
			}
			summary = summarizePolicy(policy)
		} else {
			summary = spec.build(rawFile)
		}
		items = append(items, newContextItem(spec.id, spec.kind, spec.title, spec.path, spec.section, raw, summary))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func readProjectContextFile(projectDir, rel string) (string, error) {
	body, err := os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(rel)))
	if err != nil {
		return "", fmt.Errorf("read context source %s: %w", rel, err)
	}
	return strings.ReplaceAll(string(body), "\r\n", "\n"), nil
}

func newContextItem(id string, kind ContextItemKind, title, path, section, raw, summary string) ContextItem {
	raw = strings.TrimSpace(stripLocalNotes(strings.ReplaceAll(raw, "\r\n", "\n")))
	summary = strings.TrimSpace(strings.ReplaceAll(summary, "\r\n", "\n"))
	if raw == "" {
		raw = summary
	}
	if summary == "" {
		summary = raw
	}
	return ContextItem{
		ID:      id,
		Kind:    kind,
		Title:   title,
		Summary: summary,
		Anchor: ContextAnchor{
			Path:    path,
			Section: section,
		},
		SourceHash:      hashContextSource(raw),
		ExpansionCost:   len(strings.Fields(raw)),
		EstimatedTokens: EstimateTokens(title, path, section, summary),
	}
}

func hashContextSource(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func summarizeBootContract(content string) string {
	summary := firstParagraph(content)
	if summary == "" {
		summary = "Use Brain-managed repo context before substantial work."
	}
	return clampSummary(summary, 36)
}

func summarizeWorkflowContract(agents, workflows string) string {
	steps := markdownListItems(extractMarkdownSection(agents, "Required Workflow"))
	if len(steps) == 0 {
		steps = markdownListItems(extractMarkdownSection(workflows, "Ticket Loop"))
	}
	return summarizeSteps(steps, 6, "Start or validate a Brain session, read the linked context, use Brain for durable notes, run required verification through `brain session run`, and finish the session.")
}

func summarizeWorkflowGuide(content string) string {
	parts := []string{}
	for _, heading := range []string{"Startup", "During Work", "Ticket Loop"} {
		steps := markdownListItems(extractMarkdownSection(content, heading))
		if len(steps) == 0 {
			continue
		}
		parts = append(parts, summarizeSteps(steps, 3, ""))
	}
	if len(parts) == 0 {
		return clampSummary(firstParagraph(content), 40)
	}
	return clampSummary(strings.Join(parts, " "), 48)
}

func summarizeMemoryRules(content string) string {
	required := markdownListItems(extractMarkdownSection(content, "Capture Required"))
	avoid := markdownListItems(extractMarkdownSection(content, "Do Not Capture"))
	var parts []string
	if len(required) > 0 {
		parts = append(parts, "Capture "+joinFragments(required, 4))
	}
	if len(avoid) > 0 {
		parts = append(parts, "Avoid "+joinFragments(avoid, 3))
	}
	if len(parts) == 0 {
		return clampSummary(firstParagraph(content), 40)
	}
	return clampSummary(strings.Join(parts, ". ")+".", 44)
}

func summarizeArchitectureNotes(content string) string {
	notes := markdownListItems(extractMarkdownSection(content, "Architecture Notes"))
	if len(notes) == 0 {
		return clampSummary(firstParagraph(content), 40)
	}
	return clampSummary(strings.Join(notes, " "), 42)
}

func summarizeOverview(content string) string {
	manifestItems := markdownListItems(extractMarkdownSection(content, "Repo Map"))
	if len(manifestItems) == 0 {
		return clampSummary(firstParagraph(content), 36)
	}
	return clampSummary(firstParagraph(content)+" Repo map: "+joinFragments(manifestItems, 6)+".", 42)
}

func summarizeCurrentState(content string) string {
	lines := markdownListItems(extractMarkdownSection(content, "Repository"))
	if len(lines) == 0 {
		return clampSummary(firstParagraph(content), 40)
	}
	return clampSummary("Repository snapshot: "+joinFragments(lines, 5)+".", 42)
}

func summarizeVerificationPolicy(policy *Policy) string {
	if policy == nil || len(policy.Closeout.VerificationProfiles) == 0 {
		return "No verification profiles are configured yet."
	}
	commands := make([]string, 0, len(policy.Closeout.VerificationProfiles))
	for _, profile := range policy.Closeout.VerificationProfiles {
		if len(profile.Commands) == 0 {
			continue
		}
		commands = append(commands, profile.Commands[0])
	}
	if len(commands) == 0 {
		return "Verification profiles exist but do not declare commands yet."
	}
	return clampSummary("Run "+strings.Join(commands, " and ")+" through `brain session run -- <command>` before session finish.", 42)
}

func summarizePolicy(policy *Policy) string {
	if policy == nil {
		return "No Brain policy is configured."
	}
	requiredDocs := append([]string(nil), policy.Preflight.RequiredDocs...)
	sort.Strings(requiredDocs)
	return clampSummary(
		fmt.Sprintf(
			"Preflight requires %d doc(s); closeout verification runs %d profile(s); durable notes count under accepted memory globs.",
			len(requiredDocs),
			len(policy.Closeout.VerificationProfiles),
		),
		30,
	)
}

func markdownListItems(content string) []string {
	lines := strings.Split(content, "\n")
	items := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "- "):
			items = append(items, cleanSummaryFragment(strings.TrimPrefix(line, "- ")))
		case isNumberedListItem(line):
			if idx := strings.Index(line, ". "); idx >= 0 {
				items = append(items, cleanSummaryFragment(line[idx+2:]))
			}
		}
	}
	return items
}

func summarizeSteps(steps []string, max int, fallback string) string {
	if len(steps) == 0 {
		return clampSummary(fallback, 42)
	}
	return clampSummary(strings.Join(trimList(steps, max), " "), 46)
}

func joinFragments(items []string, max int) string {
	items = trimList(items, max)
	return strings.Join(items, ", ")
}

func trimList(items []string, max int) []string {
	if max > 0 && len(items) > max {
		items = items[:max]
	}
	return items
}

func cleanSummaryFragment(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".")
	return s
}

func clampSummary(summary string, maxWords int) string {
	words := strings.Fields(strings.TrimSpace(summary))
	if len(words) == 0 {
		return ""
	}
	if maxWords > 0 && len(words) > maxWords {
		words = append(words[:maxWords], "...")
	}
	return strings.Join(words, " ")
}

func isNumberedListItem(line string) bool {
	for i := 0; i < len(line); i++ {
		if line[i] < '0' || line[i] > '9' {
			return i > 0 && strings.HasPrefix(line[i:], ". ")
		}
	}
	return false
}
