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
	"time"

	"brain/internal/history"
	"brain/internal/notes"
	"brain/internal/promotion"
	"brain/internal/session"
)

type Manager struct {
	notes   *notes.Manager
	history *history.Logger
	session *session.Manager
}

var proposalSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

type SessionProposalPreview struct {
	Path     string         `json:"path"`
	Title    string         `json:"title"`
	Type     string         `json:"type"`
	Metadata map[string]any `json:"metadata"`
	Content  string         `json:"content"`
}

func New(notesManager *notes.Manager, historyLog *history.Logger, sessionManager *session.Manager) *Manager {
	return &Manager{
		notes:   notesManager,
		history: historyLog,
		session: sessionManager,
	}
}

func (m *Manager) FromSession(ctx context.Context, limit int) (*notes.Note, error) {
	preview, err := m.PreviewFromSession(ctx, limit)
	if err != nil {
		return nil, err
	}
	return m.createProposal(preview.Title, preview.Metadata, preview.Content)
}

func (m *Manager) PreviewFromSession(ctx context.Context, limit int) (*SessionProposalPreview, error) {
	projectDir := m.notes.WorkspaceAbs(".")
	if limit <= 0 {
		limit = 6
	}
	review, err := m.session.ReviewActiveSessionPromotions(ctx, projectDir, limit)
	if err != nil {
		return nil, err
	}
	if review.Session == nil || review.Session.Status != "active" {
		return nil, errors.New("distill --session requires an active session")
	}
	active := review.Session

	title := strings.TrimSpace(active.Task)
	if title == "" {
		title = "Session"
	}
	title += " Distill Proposal"
	body := renderSessionProposal(sessionProposal{
		Title:       title,
		SessionID:   active.ID,
		Task:        active.Task,
		GitHead:     active.GitBaseline.Head,
		Commands:    active.CommandRuns,
		Changed:     review.ChangedFiles,
		DiffSummary: review.DiffSummary,
		RecentNotes: review.RecentNotes,
		Assessments: review.Assessments,
	})

	metadata := map[string]any{
		"type":                 "distill_proposal",
		"distill_scope":        "session",
		"source_session_id":    active.ID,
		"source_task":          active.Task,
		"proposed_targets":     promotableProposalTargets(review.Assessments),
		"promotion_categories": promotableCategories(review.Assessments),
	}

	return &SessionProposalPreview{
		Path:     proposalPath(title),
		Title:    title,
		Type:     "distill_proposal",
		Metadata: metadata,
		Content:  body,
	}, nil
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
	Assessments []promotion.Assessment
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
	appendPromotionReview(&b, proposal.Assessments)
	appendTargets(&b, promotableTargetEntries(proposal.Assessments))
	return b.String()
}

func appendTargets(b *strings.Builder, targets []proposalTarget) {
	b.WriteString("\n## Proposed Updates\n")
	if len(targets) == 0 {
		b.WriteString("\n- No promotable updates were detected yet. If durable knowledge changed, write the notes manually after review.\n")
		return
	}
	for _, target := range targets {
		b.WriteString("\n### " + target.Path + "\n\n")
		b.WriteString("Reason: " + target.Reason + "\n\n")
		b.WriteString("Suggested update:\n\n")
		b.WriteString("```md\n")
		b.WriteString(strings.TrimRight(strings.Join(target.Body, "\n"), "\n"))
		b.WriteString("\n```\n")
	}
}

func appendPromotionReview(b *strings.Builder, assessments []promotion.Assessment) {
	b.WriteString("\n## Promotion Review\n")
	if len(assessments) == 0 {
		b.WriteString("\n- No promotion candidates were generated for this session.\n")
		return
	}
	for _, assessment := range assessments {
		b.WriteString("\n### " + string(assessment.Candidate.Category) + " [" + string(assessment.Decision) + "]\n\n")
		b.WriteString("Summary: " + assessment.Candidate.Summary + "\n\n")
		b.WriteString("Target: `" + assessment.Candidate.SuggestedTarget + "`\n\n")
		switch assessment.Decision {
		case promotion.DecisionPromotable:
			b.WriteString("Why promotable: " + assessment.ReasonPromotable + "\n\n")
		case promotion.DecisionRejected, promotion.DecisionInsufficient:
			if strings.TrimSpace(assessment.ReasonRejected) != "" {
				b.WriteString("Why not promoted: " + assessment.ReasonRejected + "\n\n")
			}
		}
		if len(assessment.Diagnostics) != 0 {
			b.WriteString("Diagnostics:\n")
			for _, diagnostic := range assessment.Diagnostics {
				b.WriteString("- " + diagnostic + "\n")
			}
			b.WriteString("\n")
		}
	}
}

func promotableTargetEntries(assessments []promotion.Assessment) []proposalTarget {
	targets := []proposalTarget{}
	seen := map[string]struct{}{}
	for _, assessment := range assessments {
		if assessment.Decision != promotion.DecisionPromotable {
			continue
		}
		key := string(assessment.Candidate.Category) + "::" + assessment.Candidate.SuggestedTarget
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		reason := assessment.ReasonPromotable
		if strings.TrimSpace(reason) == "" {
			reason = assessment.Candidate.Summary
		}
		targets = append(targets, proposalTarget{
			Path:   assessment.Candidate.SuggestedTarget,
			Reason: fmt.Sprintf("%s [%s]", reason, assessment.Candidate.Category),
			Body:   append([]string(nil), assessment.Candidate.SuggestedBody...),
		})
	}
	sort.SliceStable(targets, func(i, j int) bool {
		if targets[i].Path == targets[j].Path {
			return targets[i].Reason < targets[j].Reason
		}
		return targets[i].Path < targets[j].Path
	})
	return targets
}

func promotableProposalTargets(assessments []promotion.Assessment) []string {
	targets := promotableTargetEntries(assessments)
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		out = append(out, target.Path)
	}
	return out
}

func promotableCategories(assessments []promotion.Assessment) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, assessment := range assessments {
		if assessment.Decision != promotion.DecisionPromotable {
			continue
		}
		key := string(assessment.Candidate.Category)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func proposalPath(title string) string {
	slug := strings.ToLower(strings.TrimSpace(title))
	slug = proposalSlugPattern.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = time.Now().UTC().Format("20060102-150405")
	}
	return filepath.ToSlash(filepath.Join(".brain", "resources", "changes", slug+".md"))
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

func isDurableNotePath(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return false
	}
	return path == "AGENTS.md" || strings.HasPrefix(path, "docs/") || strings.HasPrefix(path, ".brain/")
}
