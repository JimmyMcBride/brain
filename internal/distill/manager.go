package distill

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"brain/internal/history"
	"brain/internal/notes"
	"brain/internal/project"
	"brain/internal/search"
	"brain/internal/session"
)

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

type Manager struct {
	notes   *notes.Manager
	search  *search.Engine
	project *project.Manager
	history *history.Logger
	session *session.Manager
}

func New(notesManager *notes.Manager, searchEngine *search.Engine, projectManager *project.Manager, historyLog *history.Logger, sessionManager *session.Manager) *Manager {
	return &Manager{
		notes:   notesManager,
		search:  searchEngine,
		project: projectManager,
		history: historyLog,
		session: sessionManager,
	}
}

func (m *Manager) FromSession(ctx context.Context, limit int) (*notes.Note, error) {
	projectDir := m.notes.WorkspaceAbs(".")
	active, err := m.session.Active(projectDir)
	if err != nil {
		return nil, err
	}
	if active == nil || active.Status != "active" {
		return nil, errors.New("distill --session requires an active session")
	}
	if limit <= 0 {
		limit = 6
	}

	changedFiles, diffSummary := gitDiffSummary(ctx, projectDir, active.GitBaseline.Head)
	recentNotes, err := m.historyAfterBaseline(active.HistoryBaseline, limit)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(active.Task)
	if title == "" {
		title = "Session"
	}
	title += " Distill Proposal"
	changeSlug := slugify(active.Task)
	if changeSlug == "" {
		changeSlug = "session"
	}
	decisionSlug := changeSlug
	body := renderSessionProposal(sessionProposal{
		Title:       title,
		SessionID:   active.ID,
		Task:        active.Task,
		GitHead:     active.GitBaseline.Head,
		Commands:    active.CommandRuns,
		Changed:     changedFiles,
		DiffSummary: diffSummary,
		RecentNotes: recentNotes,
		Targets: []proposalTarget{
			{
				Path:   "AGENTS.md",
				Reason: "capture new workflow guidance, decisions, or constraints that should become part of the repo contract",
				Body: []string{
					fmt.Sprintf("- Add the durable guidance learned while working on %q.", active.Task),
					"- Keep the note concise and operational so future agents can reuse it.",
				},
			},
			{
				Path:   ".brain/context/current-state.md",
				Reason: "record what changed, what remains active, and any follow-up state that should persist beyond the session",
				Body: []string{
					fmt.Sprintf("- Summarize the current state of %q.", active.Task),
					"- Note the highest-signal changed files and any remaining follow-up.",
				},
			},
			{
				Path:   filepath.ToSlash(filepath.Join(".brain/resources/changes", changeSlug+".md")),
				Reason: "capture the implementation outcome, changed files, and verification trail as a durable change note",
				Body: []string{
					fmt.Sprintf("# %s", strings.TrimSpace(active.Task)),
					"",
					"## Outcome",
					"- Summarize the shipped behavior from this session.",
					"",
					"## Verification",
					"- Copy the passing session commands after review.",
					"",
					"## Changed Files",
					"- List the durable files that materially changed.",
				},
			},
			{
				Path:   filepath.ToSlash(filepath.Join(".brain/resources/decisions", decisionSlug+".md")),
				Reason: "preserve rationale if this session changed a technical or workflow decision instead of only changing code",
				Body: []string{
					fmt.Sprintf("# Why we chose %s", strings.TrimSpace(active.Task)),
					"",
					"## Context",
					"",
					"## Options Considered",
					"",
					"## Decision",
					"",
					"## Tradeoffs",
				},
			},
		},
	})

	return m.createProposal(title, map[string]any{
		"type":              "distill_proposal",
		"distill_scope":     "session",
		"source_session_id": active.ID,
		"source_task":       active.Task,
		"proposed_targets": proposalPaths([]proposalTarget{
			{Path: "AGENTS.md"},
			{Path: ".brain/context/current-state.md"},
			{Path: filepath.ToSlash(filepath.Join(".brain/resources/changes", changeSlug+".md"))},
			{Path: filepath.ToSlash(filepath.Join(".brain/resources/decisions", decisionSlug+".md"))},
		}),
	}, body)
}

func (m *Manager) FromBrainstorm(ctx context.Context, path string, limit int) (*notes.Note, error) {
	note, err := m.notes.Read(path)
	if err != nil {
		return nil, err
	}
	if note.Type != "brainstorm" {
		return nil, fmt.Errorf("note %s is type %q, not brainstorm", path, note.Type)
	}
	if limit <= 0 {
		limit = 6
	}

	query := strings.TrimSpace(note.Title + " " + note.Content)
	results, err := m.search.Search(ctx, query, limit+1)
	if err != nil {
		return nil, err
	}
	related := make([]search.Result, 0, len(results))
	for _, result := range results {
		if result.NotePath == note.Path {
			continue
		}
		related = append(related, result)
		if len(related) >= limit {
			break
		}
	}

	sourceSlug := strings.TrimSuffix(filepath.Base(note.Path), filepath.Ext(note.Path))
	title := strings.TrimSpace(note.Title)
	if title == "" {
		title = sourceSlug
	}
	title += " Distill Proposal"
	body := renderBrainstormProposal(brainstormProposal{
		Title:      title,
		SourcePath: note.Path,
		KeyIdeas:   extractSection(note.Content, "Ideas"),
		Related:    related,
		Targets: []proposalTarget{
			{
				Path:   "AGENTS.md",
				Reason: "carry forward any durable workflow or policy guidance that emerged from the brainstorm",
				Body: []string{
					fmt.Sprintf("- Promote the durable guidance from %q only if it should become repo contract.", note.Title),
				},
			},
			{
				Path:   ".brain/context/current-state.md",
				Reason: "capture the active problem framing if the brainstorm changes what the project is currently pursuing",
				Body: []string{
					fmt.Sprintf("- Note how %q changes the current focus or next actions.", note.Title),
				},
			},
			{
				Path:   filepath.ToSlash(filepath.Join(".brain/resources/changes", sourceSlug+".md")),
				Reason: "turn the brainstorm into a durable change note once the ideas collapse into an execution path",
				Body: []string{
					fmt.Sprintf("# %s", note.Title),
					"",
					"## Summary",
					"- Distill the strongest ideas that should survive beyond the brainstorm.",
					"",
					"## References",
					fmt.Sprintf("- [[%s]]", note.Path),
				},
			},
			{
				Path:   filepath.ToSlash(filepath.Join(".brain/resources/decisions", sourceSlug+".md")),
				Reason: "preserve the why if the brainstorm already surfaces a clear choice or tradeoff",
				Body: []string{
					fmt.Sprintf("# Why we chose %s", note.Title),
					"",
					"## Context",
					"",
					"## Options Considered",
					"",
					"## Decision",
					"",
					"## Tradeoffs",
				},
			},
		},
	})

	return m.createProposal(title, map[string]any{
		"type":              "distill_proposal",
		"distill_scope":     "brainstorm",
		"source_brainstorm": note.Path,
		"proposed_targets": proposalPaths([]proposalTarget{
			{Path: "AGENTS.md"},
			{Path: ".brain/context/current-state.md"},
			{Path: filepath.ToSlash(filepath.Join(".brain/resources/changes", sourceSlug+".md"))},
			{Path: filepath.ToSlash(filepath.Join(".brain/resources/decisions", sourceSlug+".md"))},
		}),
	}, body)
}

type sessionProposal struct {
	Title       string
	SessionID   string
	Task        string
	GitHead     string
	Commands    []session.CommandRun
	Changed     []string
	DiffSummary string
	RecentNotes []history.Entry
	Targets     []proposalTarget
}

type brainstormProposal struct {
	Title      string
	SourcePath string
	KeyIdeas   string
	Related    []search.Result
	Targets    []proposalTarget
}

type proposalTarget struct {
	Path   string
	Reason string
	Body   []string
}

func renderSessionProposal(proposal sessionProposal) string {
	var b strings.Builder
	b.WriteString("# " + proposal.Title + "\n\n")
	b.WriteString("## Source Provenance\n\n")
	b.WriteString("- Mode: `session`\n")
	b.WriteString("- Session: `" + proposal.SessionID + "`\n")
	if strings.TrimSpace(proposal.Task) != "" {
		b.WriteString("- Task: " + proposal.Task + "\n")
	}
	if strings.TrimSpace(proposal.GitHead) != "" {
		b.WriteString("- Git baseline: `" + proposal.GitHead + "`\n")
	}
	b.WriteString("\n### Commands Run\n\n")
	if len(proposal.Commands) == 0 {
		b.WriteString("- No recorded session commands yet.\n")
	} else {
		for _, command := range proposal.Commands {
			b.WriteString(fmt.Sprintf("- `%s` (exit %d)\n", command.Command, command.ExitCode))
		}
	}
	b.WriteString("\n### Git Diff\n\n")
	if len(proposal.Changed) == 0 && strings.TrimSpace(proposal.DiffSummary) == "" {
		b.WriteString("- No changed files detected from the session baseline.\n")
	} else {
		for _, path := range proposal.Changed {
			b.WriteString("- `" + path + "`\n")
		}
		if strings.TrimSpace(proposal.DiffSummary) != "" {
			b.WriteString("\n```text\n" + strings.TrimSpace(proposal.DiffSummary) + "\n```\n")
		}
	}
	b.WriteString("\n### Recent Durable Notes\n\n")
	if len(proposal.RecentNotes) == 0 {
		b.WriteString("- No durable note edits were recorded after the session baseline.\n")
	} else {
		for _, entry := range proposal.RecentNotes {
			b.WriteString(fmt.Sprintf("- `%s` (%s: %s)\n", entry.File, entry.Operation, entry.Summary))
		}
	}
	appendTargets(&b, proposal.Targets)
	return b.String()
}

func renderBrainstormProposal(proposal brainstormProposal) string {
	var b strings.Builder
	b.WriteString("# " + proposal.Title + "\n\n")
	b.WriteString("## Source Provenance\n\n")
	b.WriteString("- Mode: `brainstorm`\n")
	b.WriteString("- Source: `[[" + proposal.SourcePath + "]]`\n")
	b.WriteString("\n### Key Ideas\n\n")
	if strings.TrimSpace(proposal.KeyIdeas) == "" {
		b.WriteString("- No brainstorm ideas were captured yet.\n")
	} else {
		b.WriteString(strings.TrimSpace(proposal.KeyIdeas) + "\n")
	}
	b.WriteString("\n\n### Related Notes\n\n")
	if len(proposal.Related) == 0 {
		b.WriteString("- No related notes found.\n")
	} else {
		for _, result := range proposal.Related {
			line := "- `[[" + result.NotePath + "]]`"
			if result.Heading != "" {
				line += " -> `" + result.Heading + "`"
			}
			if strings.TrimSpace(result.Snippet) != "" {
				line += ": " + strings.TrimSpace(result.Snippet)
			}
			b.WriteString(line + "\n")
		}
	}
	appendTargets(&b, proposal.Targets)
	return b.String()
}

func appendTargets(b *strings.Builder, targets []proposalTarget) {
	b.WriteString("\n## Proposed Updates\n")
	for _, target := range targets {
		b.WriteString("\n### " + target.Path + "\n\n")
		b.WriteString("Reason: " + target.Reason + "\n\n")
		b.WriteString("Suggested update:\n\n")
		b.WriteString("```md\n")
		b.WriteString(strings.TrimRight(strings.Join(target.Body, "\n"), "\n"))
		b.WriteString("\n```\n")
	}
}

func (m *Manager) createProposal(title string, metadata map[string]any, body string) (*notes.Note, error) {
	return m.notes.Create(notes.CreateInput{
		Title:     title,
		NoteType:  "distill_proposal",
		Section:   ".brain",
		Subdir:    "resources/changes",
		Body:      body,
		Metadata:  metadata,
		Overwrite: true,
	})
}

func (m *Manager) historyAfterBaseline(baseline session.HistoryBaseline, limit int) ([]history.Entry, error) {
	if m.history == nil {
		return nil, nil
	}
	entries, err := m.history.All()
	if err != nil {
		return nil, err
	}
	var recent []history.Entry
	for _, entry := range entries {
		if baseline.LastID != "" {
			if entry.ID == baseline.LastID {
				recent = recent[:0]
				continue
			}
		}
		if baseline.LastID == "" && !baseline.LastTimestamp.IsZero() && !entry.Timestamp.After(baseline.LastTimestamp) {
			continue
		}
		if isDurableNotePath(entry.File) {
			recent = append(recent, entry)
		}
	}
	if limit > 0 && len(recent) > limit {
		recent = recent[len(recent)-limit:]
	}
	return recent, nil
}

func gitDiffSummary(ctx context.Context, projectDir, baseHead string) ([]string, string) {
	changedSet := map[string]struct{}{}
	if strings.TrimSpace(baseHead) != "" {
		for _, path := range runGitLines(ctx, "-C", projectDir, "diff", "--name-only", baseHead, "--") {
			changedSet[path] = struct{}{}
		}
	}
	for _, path := range normalizeStatusLines(runGitLines(ctx, "-C", projectDir, "status", "--short")) {
		changedSet[path] = struct{}{}
	}
	changed := make([]string, 0, len(changedSet))
	for path := range changedSet {
		if path == "" {
			continue
		}
		changed = append(changed, path)
	}
	sort.Strings(changed)
	diffSummary := strings.TrimSpace(runGitText(ctx, "-C", projectDir, "diff", "--stat", baseHead, "--"))
	if baseHead == "" {
		diffSummary = strings.TrimSpace(runGitText(ctx, "-C", projectDir, "diff", "--stat", "--"))
	}
	return changed, diffSummary
}

func runGitLines(ctx context.Context, args ...string) []string {
	text := runGitText(ctx, args...)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, filepath.ToSlash(line))
	}
	return out
}

func runGitText(ctx context.Context, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func normalizeStatusLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		out = append(out, filepath.ToSlash(fields[len(fields)-1]))
	}
	return out
}

func proposalPaths(targets []proposalTarget) []string {
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		out = append(out, target.Path)
	}
	return out
}

func isDurableNotePath(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return false
	}
	return path == "AGENTS.md" || strings.HasPrefix(path, "docs/") || strings.HasPrefix(path, ".brain/")
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugPattern.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

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
