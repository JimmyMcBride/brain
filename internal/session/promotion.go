package session

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"brain/internal/history"
	"brain/internal/projectcontext"
	"brain/internal/promotion"
)

type PromotionSuggestion struct {
	Category               string   `json:"category"`
	Summary                string   `json:"summary"`
	SuggestedTarget        string   `json:"suggested_target"`
	SupportingPacketHashes []string `json:"supporting_packet_hashes,omitempty"`
	SupportingVerification []string `json:"supporting_verification,omitempty"`
	ReasonSuggested        string   `json:"reason_suggested"`
}

type PromotionReview struct {
	Session      *ActiveSession         `json:"session,omitempty"`
	ChangedFiles []string               `json:"changed_files,omitempty"`
	DiffSummary  string                 `json:"diff_summary,omitempty"`
	RecentNotes  []history.Entry        `json:"recent_notes,omitempty"`
	Assessments  []promotion.Assessment `json:"assessments,omitempty"`
}

func (m *Manager) ReviewActiveSessionPromotions(ctx context.Context, projectDir string, limit int) (*PromotionReview, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return nil, err
	}
	policy, _, _, err := projectcontext.LoadPolicy(projectDir)
	if err != nil {
		return nil, err
	}
	active, err := loadActiveSessionIfExists(filepath.Join(projectDir, filepath.FromSlash(policy.Session.ActiveFile)))
	if err != nil {
		return nil, err
	}
	if active == nil || active.Status != "active" {
		return nil, fmt.Errorf("promotion review requires an active session")
	}
	return m.buildPromotionReview(ctx, policy, active, nil, limit)
}

func (m *Manager) buildPromotionReview(ctx context.Context, policy *projectcontext.Policy, active *ActiveSession, missingCommands []string, limit int) (*PromotionReview, error) {
	if active == nil {
		return nil, fmt.Errorf("active session is required")
	}
	if limit <= 0 {
		limit = 6
	}
	entries, err := m.historyAfterBaseline(active.HistoryBaseline)
	if err != nil {
		return nil, err
	}
	durableUpdates, _ := filterHistoryEntries(entries, policy.Closeout.AcceptableHistoryOperations, policy.Project.Memory.AcceptedNoteGlobs)
	recentNotes := recentDurableEntries(entries, policy.Project.Memory.AcceptedNoteGlobs, limit)
	changedFiles, diffSummary := changedFilesSinceBaseline(ctx, active.ProjectDir, active.GitBaseline.Head)
	packetHashes, boundaries := packetSupportFromSession(active)
	successfulCommands, failedCommands := commandSupport(active.CommandRuns)
	repoChanged := len(changedFiles) != 0

	signals := promotion.SessionSignals{
		Task:                   active.Task,
		RepoChanged:            repoChanged,
		ChangedFiles:           changedFiles,
		ChangedBoundaries:      boundaries,
		PacketHashes:           packetHashes,
		SuccessfulCommands:     successfulCommands,
		FailedCommands:         failedCommands,
		MissingVerification:    append([]string(nil), missingCommands...),
		DurableUpdates:         durableUpdatePaths(durableUpdates),
		WorkflowSurfaceChanged: workflowSurfaceChanged(changedFiles),
		DecisionLikeTask:       decisionLikeTask(active.Task),
	}

	return &PromotionReview{
		Session:      active,
		ChangedFiles: changedFiles,
		DiffSummary:  diffSummary,
		RecentNotes:  recentNotes,
		Assessments:  promotion.AssessSession(signals),
	}, nil
}

func promotionSuggestionsFromReview(review *PromotionReview) []PromotionSuggestion {
	if review == nil {
		return nil
	}
	assessments := promotion.Promotable(review.Assessments)
	if len(assessments) == 0 {
		return nil
	}
	suggestions := make([]PromotionSuggestion, 0, len(assessments))
	seen := map[string]struct{}{}
	for _, assessment := range assessments {
		key := string(assessment.Candidate.Category) + "::" + assessment.Candidate.SuggestedTarget
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		suggestions = append(suggestions, PromotionSuggestion{
			Category:               string(assessment.Candidate.Category),
			Summary:                assessment.Candidate.Summary,
			SuggestedTarget:        assessment.Candidate.SuggestedTarget,
			SupportingPacketHashes: append([]string(nil), assessment.Candidate.Support.PacketHashes...),
			SupportingVerification: append([]string(nil), assessment.Candidate.Support.SuccessfulCommands...),
			ReasonSuggested:        suggestionReason(assessment),
		})
	}
	sort.SliceStable(suggestions, func(i, j int) bool {
		ri := suggestionRank(suggestions[i].Category)
		rj := suggestionRank(suggestions[j].Category)
		if ri == rj {
			return suggestions[i].SuggestedTarget < suggestions[j].SuggestedTarget
		}
		return ri < rj
	})
	if len(suggestions) > 3 {
		suggestions = append([]PromotionSuggestion(nil), suggestions[:3]...)
	}
	return suggestions
}

func filterPromotionSuggestionsForValidation(result *ValidationResult, suggestions []PromotionSuggestion) []PromotionSuggestion {
	if len(suggestions) == 0 || result == nil {
		return suggestions
	}
	filtered := make([]PromotionSuggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		if strings.TrimSpace(result.MemorySatisfiedBy) != "" {
			if suggestion.Category == string(promotion.CategoryFollowUp) || suggestion.Category == string(promotion.CategoryGotcha) {
				filtered = append(filtered, suggestion)
			}
			continue
		}
		if len(suggestion.SupportingPacketHashes) != 0 || suggestion.Category == string(promotion.CategoryFollowUp) || suggestion.Category == string(promotion.CategoryGotcha) {
			filtered = append(filtered, suggestion)
		}
	}
	return filtered
}

func suggestionReason(assessment promotion.Assessment) string {
	if strings.TrimSpace(assessment.ReasonPromotable) != "" {
		return assessment.ReasonPromotable
	}
	return "packet-backed session evidence suggests this durable follow-through is worth review"
}

func suggestionRank(category string) int {
	switch category {
	case string(promotion.CategoryFollowUp):
		return 0
	case string(promotion.CategoryBoundaryFact):
		return 1
	case string(promotion.CategoryVerificationRecipe):
		return 2
	case string(promotion.CategoryDecision):
		return 3
	case string(promotion.CategoryInvariant):
		return 4
	case string(promotion.CategoryGotcha):
		return 5
	default:
		return 6
	}
}

func changedFilesSinceBaseline(ctx context.Context, projectDir, baseHead string) ([]string, string) {
	changedSet := map[string]struct{}{}
	if strings.TrimSpace(baseHead) != "" {
		paths, err := gitChangedPathsBetween(ctx, projectDir, baseHead, "HEAD")
		if err == nil {
			for _, path := range paths {
				changedSet[filepath.ToSlash(path)] = struct{}{}
			}
		}
	}
	current := snapshotGit(ctx, projectDir)
	for _, line := range current.Status {
		path := gitStatusPath(line)
		if path == "" {
			continue
		}
		changedSet[path] = struct{}{}
	}
	changed := make([]string, 0, len(changedSet))
	for path := range changedSet {
		changed = append(changed, path)
	}
	sort.Strings(changed)
	diffSummary := strings.TrimSpace(runGit(ctx, projectDir, "diff", "--stat", "--"))
	if strings.TrimSpace(baseHead) != "" {
		diffSummary = strings.TrimSpace(runGit(ctx, projectDir, "diff", "--stat", baseHead, "--"))
	}
	return changed, diffSummary
}

func packetSupportFromSession(active *ActiveSession) ([]string, []string) {
	if active == nil {
		return nil, nil
	}
	packetHashes := make([]string, 0, len(active.PacketRecords))
	boundaries := []string{}
	for _, record := range active.PacketRecords {
		if strings.TrimSpace(record.PacketHash) != "" {
			packetHashes = append(packetHashes, record.PacketHash)
		}
		for idx, itemID := range record.IncludedItemIDs {
			if !strings.HasPrefix(itemID, "boundary:") {
				continue
			}
			if idx < len(record.IncludedAnchors) && strings.TrimSpace(record.IncludedAnchors[idx].Path) != "" {
				boundaries = append(boundaries, record.IncludedAnchors[idx].Path)
				continue
			}
			boundaries = append(boundaries, strings.TrimPrefix(itemID, "boundary:"))
		}
	}
	sort.Strings(packetHashes)
	sort.Strings(boundaries)
	return dedupeStrings(packetHashes), dedupeStrings(boundaries)
}

func commandSupport(runs []CommandRun) ([]string, []string) {
	successful := []string{}
	failed := []string{}
	for _, run := range runs {
		if strings.TrimSpace(run.Command) == "" {
			continue
		}
		if run.ExitCode == 0 {
			successful = append(successful, run.Command)
		} else {
			failed = append(failed, run.Command)
		}
	}
	sort.Strings(successful)
	sort.Strings(failed)
	return dedupeStrings(successful), dedupeStrings(failed)
}

func durableUpdatePaths(entries []history.Entry) []string {
	paths := []string{}
	for _, entry := range entries {
		if strings.TrimSpace(entry.File) != "" {
			paths = append(paths, entry.File)
		}
		if strings.TrimSpace(entry.Target) != "" {
			paths = append(paths, entry.Target)
		}
	}
	sort.Strings(paths)
	return dedupeStrings(paths)
}

func recentDurableEntries(entries []history.Entry, globs []string, limit int) []history.Entry {
	recent := make([]history.Entry, 0, len(entries))
	for _, entry := range entries {
		if pathMatchesAny(entry.File, globs) || pathMatchesAny(entry.Target, globs) {
			recent = append(recent, entry)
		}
	}
	if limit > 0 && len(recent) > limit {
		recent = recent[len(recent)-limit:]
	}
	return recent
}

func workflowSurfaceChanged(files []string) bool {
	for _, file := range files {
		file = filepath.ToSlash(strings.TrimSpace(file))
		switch {
		case file == "AGENTS.md":
			return true
		case file == ".brain/policy.yaml":
			return true
		case file == "skills/brain/SKILL.md":
			return true
		case strings.HasPrefix(file, "docs/"):
			return true
		case strings.HasPrefix(file, "cmd/"):
			return true
		case strings.HasPrefix(file, "internal/session/"):
			return true
		case strings.HasPrefix(file, "internal/projectcontext/"):
			return true
		}
	}
	return false
}

func decisionLikeTask(task string) bool {
	task = strings.ToLower(strings.TrimSpace(task))
	for _, token := range []string{"adopt", "choose", "deprecate", "migrate", "remove", "rename", "replace", "switch"} {
		if strings.Contains(task, token) {
			return true
		}
	}
	return false
}

func dedupeStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = filepath.ToSlash(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
