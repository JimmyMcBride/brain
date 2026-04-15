package promotion

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type Category string

const (
	CategoryDecision           Category = "decision"
	CategoryInvariant          Category = "invariant"
	CategoryGotcha             Category = "gotcha"
	CategoryVerificationRecipe Category = "verification_recipe"
	CategoryBoundaryFact       Category = "boundary_fact"
	CategoryFollowUp           Category = "follow_up"
)

type Decision string

const (
	DecisionPromotable   Decision = "promotable"
	DecisionRejected     Decision = "rejected"
	DecisionInsufficient Decision = "insufficient"
)

type Support struct {
	PacketHashes       []string `json:"packet_hashes,omitempty"`
	ChangedFiles       []string `json:"changed_files,omitempty"`
	ChangedBoundaries  []string `json:"changed_boundaries,omitempty"`
	SuccessfulCommands []string `json:"successful_commands,omitempty"`
	FailedCommands     []string `json:"failed_commands,omitempty"`
	MissingCommands    []string `json:"missing_commands,omitempty"`
	DurableUpdates     []string `json:"durable_updates,omitempty"`
	Signals            []string `json:"signals,omitempty"`
}

type Candidate struct {
	ID              string   `json:"id"`
	Category        Category `json:"category"`
	Summary         string   `json:"summary"`
	SuggestedTarget string   `json:"suggested_target"`
	SuggestedBody   []string `json:"suggested_body,omitempty"`
	RequiresReview  bool     `json:"requires_review"`
	Support         Support  `json:"support"`
}

type Assessment struct {
	Candidate        Candidate `json:"candidate"`
	Decision         Decision  `json:"decision"`
	ReasonPromotable string    `json:"reason_promotable,omitempty"`
	ReasonRejected   string    `json:"reason_rejected,omitempty"`
	Diagnostics      []string  `json:"diagnostics,omitempty"`
}

type SessionSignals struct {
	Task                   string   `json:"task"`
	RepoChanged            bool     `json:"repo_changed"`
	ChangedFiles           []string `json:"changed_files,omitempty"`
	ChangedBoundaries      []string `json:"changed_boundaries,omitempty"`
	PacketHashes           []string `json:"packet_hashes,omitempty"`
	SuccessfulCommands     []string `json:"successful_commands,omitempty"`
	FailedCommands         []string `json:"failed_commands,omitempty"`
	MissingVerification    []string `json:"missing_verification,omitempty"`
	DurableUpdates         []string `json:"durable_updates,omitempty"`
	WorkflowSurfaceChanged bool     `json:"workflow_surface_changed"`
	DecisionLikeTask       bool     `json:"decision_like_task"`
}

func Categories() []Category {
	return []Category{
		CategoryDecision,
		CategoryInvariant,
		CategoryGotcha,
		CategoryVerificationRecipe,
		CategoryBoundaryFact,
		CategoryFollowUp,
	}
}

func NonPromotableDefaults() []string {
	return []string{
		"speculative_reasoning",
		"transient_scratch",
		"dead_end_experiment",
	}
}

func AssessSession(signals SessionSignals) []Assessment {
	candidates := BuildSessionCandidates(signals)
	assessments := make([]Assessment, 0, len(candidates))
	for _, candidate := range candidates {
		assessments = append(assessments, Assess(candidate))
	}
	sort.SliceStable(assessments, func(i, j int) bool {
		if assessments[i].Decision == assessments[j].Decision {
			return assessments[i].Candidate.ID < assessments[j].Candidate.ID
		}
		return decisionRank(assessments[i].Decision) < decisionRank(assessments[j].Decision)
	})
	return assessments
}

func BuildSessionCandidates(signals SessionSignals) []Candidate {
	task := strings.TrimSpace(signals.Task)
	changeSlug := slugify(task)
	if changeSlug == "" {
		changeSlug = "session"
	}
	supportBase := Support{
		PacketHashes:       dedupeStrings(signals.PacketHashes),
		ChangedFiles:       dedupeStrings(signals.ChangedFiles),
		ChangedBoundaries:  dedupeStrings(signals.ChangedBoundaries),
		SuccessfulCommands: dedupeStrings(signals.SuccessfulCommands),
		FailedCommands:     dedupeStrings(signals.FailedCommands),
		MissingCommands:    dedupeStrings(signals.MissingVerification),
		DurableUpdates:     dedupeStrings(signals.DurableUpdates),
	}

	candidates := make([]Candidate, 0, 6)
	candidates = append(candidates, Candidate{
		ID:              "boundary-fact",
		Category:        CategoryBoundaryFact,
		Summary:         fmt.Sprintf("Record the durable outcome and touched boundaries from %q.", taskOrSession(task)),
		SuggestedTarget: ".brain/context/current-state.md",
		SuggestedBody:   buildBoundaryFactBody(task, supportBase.ChangedBoundaries, supportBase.ChangedFiles),
		RequiresReview:  true,
		Support:         supportBase,
	})
	candidates = append(candidates, Candidate{
		ID:              "verification-recipe",
		Category:        CategoryVerificationRecipe,
		Summary:         fmt.Sprintf("Capture the repeatable verification recipe that proved %q.", taskOrSession(task)),
		SuggestedTarget: filepath.ToSlash(filepath.Join(".brain/resources/changes", changeSlug+".md")),
		SuggestedBody:   buildVerificationRecipeBody(task, supportBase.SuccessfulCommands),
		RequiresReview:  true,
		Support:         supportBase,
	})
	candidates = append(candidates, Candidate{
		ID:              "decision",
		Category:        CategoryDecision,
		Summary:         fmt.Sprintf("Preserve the rationale if %q changed a technical or workflow decision.", taskOrSession(task)),
		SuggestedTarget: filepath.ToSlash(filepath.Join(".brain/resources/decisions", changeSlug+".md")),
		SuggestedBody:   buildDecisionBody(task),
		RequiresReview:  true,
		Support:         withSignal(supportBase, "decision_like_task", signals.DecisionLikeTask),
	})
	candidates = append(candidates, Candidate{
		ID:              "invariant",
		Category:        CategoryInvariant,
		Summary:         fmt.Sprintf("Promote any durable workflow or interface rule that %q changed.", taskOrSession(task)),
		SuggestedTarget: "AGENTS.md",
		SuggestedBody:   buildInvariantBody(task, supportBase.ChangedFiles),
		RequiresReview:  true,
		Support:         withSignal(supportBase, "workflow_surface_changed", signals.WorkflowSurfaceChanged),
	})
	candidates = append(candidates, Candidate{
		ID:              "gotcha",
		Category:        CategoryGotcha,
		Summary:         fmt.Sprintf("Capture any recurring trap or regression guard exposed while working on %q.", taskOrSession(task)),
		SuggestedTarget: ".brain/context/current-state.md",
		SuggestedBody:   buildGotchaBody(task, supportBase.FailedCommands),
		RequiresReview:  true,
		Support:         supportBase,
	})
	candidates = append(candidates, Candidate{
		ID:              "follow-up",
		Category:        CategoryFollowUp,
		Summary:         fmt.Sprintf("Record the unresolved follow-up required to fully close %q.", taskOrSession(task)),
		SuggestedTarget: ".brain/context/current-state.md",
		SuggestedBody:   buildFollowUpBody(task, supportBase.MissingCommands, supportBase.FailedCommands),
		RequiresReview:  true,
		Support:         supportBase,
	})
	return candidates
}

func Assess(candidate Candidate) Assessment {
	assessment := Assessment{
		Candidate:   candidate,
		Decision:    DecisionInsufficient,
		Diagnostics: diagnosticsForCandidate(candidate),
	}
	switch candidate.Category {
	case CategoryBoundaryFact:
		if supportHasPath(candidate.Support.DurableUpdates, candidate.SuggestedTarget) {
			assessment.Decision = DecisionRejected
			assessment.ReasonRejected = "the suggested durable target already changed in this session"
			return assessment
		}
		if len(candidate.Support.ChangedFiles) == 0 {
			assessment.ReasonRejected = "no repo changes were observed"
			return assessment
		}
		if len(candidate.Support.PacketHashes) == 0 && len(candidate.Support.ChangedBoundaries) == 0 {
			assessment.Decision = DecisionPromotable
			assessment.ReasonPromotable = "repo changes touched durable files, but packet or boundary evidence was not recorded yet"
			return assessment
		}
		assessment.Decision = DecisionPromotable
		assessment.ReasonPromotable = "repo changes touched concrete files and boundaries that future sessions may need"
	case CategoryVerificationRecipe:
		if supportHasPath(candidate.Support.DurableUpdates, candidate.SuggestedTarget) {
			assessment.Decision = DecisionRejected
			assessment.ReasonRejected = "the suggested durable target already changed in this session"
			return assessment
		}
		if len(candidate.Support.SuccessfulCommands) == 0 {
			assessment.ReasonRejected = "no successful verification commands were recorded"
			return assessment
		}
		if len(candidate.Support.PacketHashes) == 0 {
			assessment.Decision = DecisionPromotable
			assessment.ReasonPromotable = "successful verification commands were recorded, but packet linkage was not captured"
			return assessment
		}
		assessment.Decision = DecisionPromotable
		assessment.ReasonPromotable = "successful verification commands were recorded against the packet-driven work"
	case CategoryDecision:
		if hasSignal(candidate.Support, "decision_like_task") && (len(candidate.Support.PacketHashes) != 0 || len(candidate.Support.ChangedFiles) != 0) {
			assessment.Decision = DecisionPromotable
			assessment.ReasonPromotable = "the task wording suggests a deliberate choice and the session changed repo state"
			return assessment
		}
		assessment.ReasonRejected = "the session does not show strong evidence that a durable decision changed"
	case CategoryInvariant:
		if hasSignal(candidate.Support, "workflow_surface_changed") && (len(candidate.Support.PacketHashes) != 0 || len(candidate.Support.ChangedFiles) != 0) {
			assessment.Decision = DecisionPromotable
			assessment.ReasonPromotable = "workflow or interface surfaces changed and may need an explicit durable rule"
			return assessment
		}
		assessment.ReasonRejected = "no workflow or contract surface changed strongly enough to justify a durable rule"
	case CategoryGotcha:
		if len(candidate.Support.FailedCommands) == 0 {
			assessment.ReasonRejected = "no failed verification or execution signal exposed a recurring trap"
			return assessment
		}
		assessment.Decision = DecisionPromotable
		assessment.ReasonPromotable = "the session recorded failed commands that may deserve a durable trap note"
	case CategoryFollowUp:
		if len(candidate.Support.MissingCommands) == 0 && len(candidate.Support.FailedCommands) == 0 {
			assessment.ReasonRejected = "no unresolved verification or execution follow-up remains"
			return assessment
		}
		assessment.Decision = DecisionPromotable
		assessment.ReasonPromotable = "the session still has unresolved verification or execution follow-up"
	default:
		assessment.Decision = DecisionRejected
		assessment.ReasonRejected = "unsupported promotion category"
	}
	return assessment
}

func Promotable(assessments []Assessment) []Assessment {
	out := make([]Assessment, 0, len(assessments))
	for _, assessment := range assessments {
		if assessment.Decision == DecisionPromotable {
			out = append(out, assessment)
		}
	}
	return out
}

func decisionRank(decision Decision) int {
	switch decision {
	case DecisionPromotable:
		return 0
	case DecisionInsufficient:
		return 1
	default:
		return 2
	}
}

func diagnosticsForCandidate(candidate Candidate) []string {
	diagnostics := []string{}
	if len(candidate.Support.PacketHashes) != 0 {
		diagnostics = append(diagnostics, fmt.Sprintf("linked to %d compiled packet(s)", len(candidate.Support.PacketHashes)))
	}
	if len(candidate.Support.ChangedBoundaries) != 0 {
		diagnostics = append(diagnostics, fmt.Sprintf("touches %d boundary/boundaries", len(candidate.Support.ChangedBoundaries)))
	}
	if len(candidate.Support.ChangedFiles) != 0 {
		diagnostics = append(diagnostics, fmt.Sprintf("touches %d changed file(s)", len(candidate.Support.ChangedFiles)))
	}
	if len(candidate.Support.SuccessfulCommands) != 0 {
		diagnostics = append(diagnostics, fmt.Sprintf("%d successful verification command(s) recorded", len(candidate.Support.SuccessfulCommands)))
	}
	if len(candidate.Support.FailedCommands) != 0 {
		diagnostics = append(diagnostics, fmt.Sprintf("%d failed command(s) recorded", len(candidate.Support.FailedCommands)))
	}
	if len(candidate.Support.MissingCommands) != 0 {
		diagnostics = append(diagnostics, fmt.Sprintf("%d verification profile(s) still missing", len(candidate.Support.MissingCommands)))
	}
	if len(candidate.Support.DurableUpdates) != 0 {
		diagnostics = append(diagnostics, fmt.Sprintf("%d durable update(s) already recorded", len(candidate.Support.DurableUpdates)))
	}
	if len(candidate.Support.Signals) != 0 {
		for _, signal := range candidate.Support.Signals {
			diagnostics = append(diagnostics, "signal: "+signal)
		}
	}
	return diagnostics
}

func buildBoundaryFactBody(task string, boundaries, files []string) []string {
	body := []string{
		fmt.Sprintf("- Summarize the durable outcome from %q.", taskOrSession(task)),
	}
	if len(boundaries) != 0 {
		body = append(body, fmt.Sprintf("- Note the touched boundaries: %s.", joinBackticked(boundaries)))
	}
	if len(files) != 0 {
		body = append(body, fmt.Sprintf("- Mention the highest-signal changed files: %s.", joinBackticked(trimList(files, 6))))
	}
	return body
}

func buildVerificationRecipeBody(task string, commands []string) []string {
	body := []string{
		fmt.Sprintf("## Verification for %s", taskOrSession(task)),
		"",
		"- Capture only the commands that proved the work after review.",
	}
	for _, command := range trimList(commands, 5) {
		body = append(body, fmt.Sprintf("- `%s`", command))
	}
	return body
}

func buildDecisionBody(task string) []string {
	return []string{
		fmt.Sprintf("# Why we chose %s", taskOrSession(task)),
		"",
		"## Context",
		"",
		"## Options Considered",
		"",
		"## Decision",
		"",
		"## Tradeoffs",
	}
}

func buildInvariantBody(task string, files []string) []string {
	body := []string{
		fmt.Sprintf("- If %q changed a reusable workflow or interface rule, record it here as an operational invariant.", taskOrSession(task)),
	}
	if len(files) != 0 {
		body = append(body, fmt.Sprintf("- Review the changed surfaces first: %s.", joinBackticked(trimList(files, 6))))
	}
	return body
}

func buildGotchaBody(task string, failed []string) []string {
	body := []string{
		fmt.Sprintf("- Capture the recurring trap exposed while working on %q only if it will matter again.", taskOrSession(task)),
	}
	for _, command := range trimList(failed, 4) {
		body = append(body, fmt.Sprintf("- Failed command to inspect: `%s`", command))
	}
	return body
}

func buildFollowUpBody(task string, missing, failed []string) []string {
	body := []string{
		fmt.Sprintf("- Record the unresolved follow-up for %q only if it should survive this session.", taskOrSession(task)),
	}
	for _, name := range trimList(missing, 4) {
		body = append(body, fmt.Sprintf("- Missing verification profile: `%s`", name))
	}
	for _, command := range trimList(failed, 4) {
		body = append(body, fmt.Sprintf("- Failed command still needing follow-up: `%s`", command))
	}
	return body
}

func withSignal(support Support, signal string, enabled bool) Support {
	if !enabled {
		return support
	}
	support.Signals = append(dedupeStrings(support.Signals), signal)
	return support
}

func hasSignal(support Support, signal string) bool {
	for _, current := range support.Signals {
		if current == signal {
			return true
		}
	}
	return false
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "_", "-", ".", "-", ":", "-")
	value = replacer.Replace(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func taskOrSession(task string) string {
	task = strings.TrimSpace(task)
	if task == "" {
		return "this session"
	}
	return task
}

func joinBackticked(items []string) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, "`"+item+"`")
	}
	return strings.Join(parts, ", ")
}

func trimList(items []string, limit int) []string {
	items = dedupeStrings(items)
	if len(items) <= limit {
		return items
	}
	return append([]string(nil), items[:limit]...)
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

func supportHasPath(paths []string, target string) bool {
	target = filepath.ToSlash(strings.TrimSpace(target))
	for _, path := range paths {
		if filepath.ToSlash(strings.TrimSpace(path)) == target {
			return true
		}
	}
	return false
}
