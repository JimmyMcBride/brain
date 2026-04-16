package taskcontext

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"brain/internal/livecontext"
	"brain/internal/projectcontext"
	"brain/internal/search"
	"brain/internal/structure"
)

type Manager struct {
	Context *projectcontext.Manager
}

type Request struct {
	ProjectDir     string
	Task           string
	TaskSource     string
	Budget         string
	SearchResults  []search.Result
	LivePacket     *livecontext.Packet
	BoundaryGraph  *structure.BoundaryGraph
	UtilitySignals map[string]ItemUtilitySignal
}

type ItemUtilitySignal struct {
	LikelyUtility               string
	IncludeCount                int
	ExpandCount                 int
	SuccessfulVerificationCount int
	DurableUpdateCount          int
	UnusedIncludeCount          int
	UtilityScore                int
	NoiseScore                  int
	Reasons                     []string
}

type noteCandidate struct {
	item              projectcontext.CompiledItem
	score             int
	baseScore         int
	primaryBoundary   string
	matchedBoundaries []string
	matchedFiles      []string
	utilityReason     string
}

func New(contextManager *projectcontext.Manager) *Manager {
	return &Manager{Context: contextManager}
}

func (m *Manager) Compile(req Request) (*projectcontext.CompiledPacket, error) {
	task := strings.TrimSpace(req.Task)
	if task == "" {
		return nil, errors.New("context compile requires a task")
	}
	taskSource := strings.TrimSpace(req.TaskSource)
	if taskSource == "" {
		taskSource = "flag"
	}

	baseContract, err := m.Context.BuildBaseContractItems(req.ProjectDir)
	if err != nil {
		return nil, err
	}
	sourceSummaries, err := m.Context.BuildSourceSummaryItems(req.ProjectDir)
	if err != nil {
		return nil, err
	}
	budget, err := resolveCompileBudget(strings.TrimSpace(req.Budget))
	if err != nil {
		return nil, err
	}
	baseContractItems := selectBaseContract(baseContract)
	verificationHints := selectVerificationHints(req.LivePacket)
	reserveBase := estimateBaseContractReserve(baseContractItems)
	reserveVerification := estimateVerificationReserve(verificationHints)
	reserveDiagnostics := estimateDiagnosticsReserve(task, taskSource, req.LivePacket)
	workingSet := selectBudgetedWorkingSet(
		max(0, budget.Target-reserveBase-reserveVerification-reserveDiagnostics-estimateSelectionOverheadReserve()),
		selectBoundaries(req.LivePacket, 0),
		selectFiles(req.LivePacket, 0),
		selectTests(req.LivePacket, 0),
		rankedWorkingSetNotes(req.ProjectDir, req.SearchResults, sourceSummaries, req.LivePacket, req.BoundaryGraph, req.UtilitySignals),
	)
	packet := &projectcontext.CompiledPacket{
		Task: projectcontext.CompiledTask{
			Text:    task,
			Summary: summarizeTask(task),
			Source:  taskSource,
		},
		BaseContract: baseContractItems,
		WorkingSet:   workingSet.Selected,
		Verification: verificationHints,
		Ambiguities:  buildAmbiguities(req.LivePacket, req.SearchResults, req.BoundaryGraph, workingSet.Selected.Notes),
	}
	packet.Provenance = buildProvenance(packet)
	packet.Budget = buildPacketBudget(packet, budget, reserveBase, reserveVerification, reserveDiagnostics, workingSet.Omitted)
	return packet, nil
}

func RenderHuman(w io.Writer, packet *projectcontext.CompiledPacket) error {
	if packet == nil {
		return errors.New("compiled packet is required")
	}
	if _, err := fmt.Fprintf(w, "## Compiled Context Packet\n\n- Task: `%s`\n- Summary: %s\n- Source: `%s`\n\n", packet.Task.Text, packet.Task.Summary, packet.Task.Source); err != nil {
		return err
	}
	if err := renderBudget(w, packet.Budget); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "## Base Contract\n\n"); err != nil {
		return err
	}
	for _, item := range packet.BaseContract {
		if _, err := fmt.Fprintf(w, "- %s (`%s`): %s\n  Reason: %s\n", item.Title, anchorLabel(item.Anchor), item.Summary, item.Reason); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, "\n## Working Set\n\n"); err != nil {
		return err
	}
	if err := renderBoundaries(w, packet.WorkingSet.Boundaries); err != nil {
		return err
	}
	if err := renderFiles(w, packet.WorkingSet.Files); err != nil {
		return err
	}
	if err := renderTests(w, packet.WorkingSet.Tests); err != nil {
		return err
	}
	if err := renderNotes(w, packet.WorkingSet.Notes); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n## Verification Hints\n\n"); err != nil {
		return err
	}
	if len(packet.Verification) == 0 {
		if _, err := io.WriteString(w, "- None yet.\n"); err != nil {
			return err
		}
	} else {
		for _, hint := range packet.Verification {
			if hint.Command != "" {
				if _, err := fmt.Fprintf(w, "- %s [%s] (`%s`): `%s`\n  %s\n  Reason: %s\n", hint.Label, hint.Strength, hint.Source, hint.Command, hint.Summary, hint.Reason); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "- %s (`%s`): %s\n  Reason: %s\n", hint.Label, hint.Source, hint.Summary, hint.Reason); err != nil {
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
	if _, err := io.WriteString(w, "\n## Provenance\n\n"); err != nil {
		return err
	}
	for _, entry := range packet.Provenance {
		if _, err := fmt.Fprintf(w, "- `%s` [%s]: %s\n", entry.ItemID, anchorLabel(entry.Anchor), entry.Reason); err != nil {
			return err
		}
	}
	return nil
}

func selectBaseContract(items []projectcontext.ContextItem) []projectcontext.CompiledItem {
	selected := make([]projectcontext.CompiledItem, 0, len(items))
	for _, item := range items {
		selected = append(selected, projectcontext.CompiledItem{
			ContextItem: item,
			Reason:      "always included as part of the base contract",
		})
	}
	return selected
}

func selectFiles(packet *livecontext.Packet, limit int) []projectcontext.CompiledFile {
	if packet == nil {
		return []projectcontext.CompiledFile{}
	}
	files := make([]projectcontext.CompiledFile, 0, min(limit, len(packet.Worktree.ChangedFiles)))
	for _, file := range trimChangedFiles(packet.Worktree.ChangedFiles, limit) {
		files = append(files, projectcontext.CompiledFile{
			Path:            file.Path,
			Status:          file.Status,
			Source:          file.Source,
			Reason:          file.Why,
			EstimatedTokens: projectcontext.EstimateTokens(file.Path, file.Status, file.Source, file.Why),
		})
	}
	return files
}

func selectBoundaries(packet *livecontext.Packet, limit int) []projectcontext.CompiledBoundary {
	if packet == nil {
		return []projectcontext.CompiledBoundary{}
	}
	boundaries := make([]projectcontext.CompiledBoundary, 0, min(limit, len(packet.Worktree.TouchedBoundaries)))
	for _, boundary := range trimBoundaries(packet.Worktree.TouchedBoundaries, limit) {
		boundaries = append(boundaries, projectcontext.CompiledBoundary{
			Path:               boundary.Path,
			Label:              boundary.Label,
			Role:               boundary.Role,
			Reason:             boundary.Why,
			AdjacentBoundaries: append([]string(nil), boundary.AdjacentBoundaries...),
			Responsibilities:   append([]string(nil), boundary.Responsibilities...),
			EstimatedTokens:    projectcontext.EstimateTokens(boundary.Path, boundary.Label, boundary.Role, boundary.Why, strings.Join(boundary.AdjacentBoundaries, " "), strings.Join(boundary.Responsibilities, " ")),
		})
	}
	return boundaries
}

func selectTests(packet *livecontext.Packet, limit int) []projectcontext.CompiledTest {
	if packet == nil {
		return []projectcontext.CompiledTest{}
	}
	tests := make([]projectcontext.CompiledTest, 0, min(limit, len(packet.NearbyTests)))
	for _, test := range trimTests(packet.NearbyTests, limit) {
		tests = append(tests, projectcontext.CompiledTest{
			Path:            test.Path,
			Relation:        test.Relation,
			Reason:          test.Why,
			EstimatedTokens: projectcontext.EstimateTokens(test.Path, test.Relation, test.Why),
		})
	}
	return tests
}

func selectWorkingSetNotes(projectDir string, results []search.Result, generated []projectcontext.ContextItem, packet *livecontext.Packet, graph *structure.BoundaryGraph, utility map[string]ItemUtilitySignal, limit int) []projectcontext.CompiledItem {
	ranked := rankedWorkingSetNotes(projectDir, results, generated, packet, graph, utility)
	if limit <= 0 {
		return ranked
	}
	if len(ranked) > limit {
		return append([]projectcontext.CompiledItem(nil), ranked[:limit]...)
	}
	return ranked
}

func rankedWorkingSetNotes(projectDir string, results []search.Result, generated []projectcontext.ContextItem, packet *livecontext.Packet, graph *structure.BoundaryGraph, utility map[string]ItemUtilitySignal) []projectcontext.CompiledItem {
	if graph == nil || packet == nil || len(packet.Worktree.TouchedBoundaries) == 0 {
		return lexicalDurableNotes(results, utility, 0)
	}
	touchedOrder := orderedTouchedBoundaries(packet, graph)
	candidates := buildNoteCandidates(projectDir, results, generated, packet, graph, utility)
	if len(candidates) == 0 {
		return lexicalDurableNotes(results, utility, 0)
	}

	grouped := map[string][]noteCandidate{}
	for _, candidate := range candidates {
		if candidate.primaryBoundary != "" {
			grouped[candidate.primaryBoundary] = append(grouped[candidate.primaryBoundary], candidate)
		}
	}
	for key := range grouped {
		sort.Slice(grouped[key], func(i, j int) bool { return noteCandidateLess(grouped[key][i], grouped[key][j]) })
	}

	selected := []projectcontext.CompiledItem{}
	seen := map[string]struct{}{}
	for _, boundaryID := range touchedOrder {
		group := grouped[boundaryID]
		if len(group) == 0 {
			continue
		}
		candidate := group[0]
		if _, ok := seen[candidate.item.ID]; ok {
			continue
		}
		selected = append(selected, candidate.item)
		seen[candidate.item.ID] = struct{}{}
	}

	remaining := []noteCandidate{}
	for _, candidate := range candidates {
		if _, ok := seen[candidate.item.ID]; ok {
			continue
		}
		remaining = append(remaining, candidate)
	}
	sort.Slice(remaining, func(i, j int) bool { return noteCandidateLess(remaining[i], remaining[j]) })
	for _, candidate := range remaining {
		selected = append(selected, candidate.item)
		seen[candidate.item.ID] = struct{}{}
	}
	for _, item := range lexicalDurableNotes(results, utility, 0) {
		if _, ok := seen[item.ID]; ok {
			continue
		}
		selected = append(selected, item)
		seen[item.ID] = struct{}{}
	}
	return selected
}

func lexicalDurableNotes(results []search.Result, utility map[string]ItemUtilitySignal, limit int) []projectcontext.CompiledItem {
	candidates := make([]noteCandidate, 0, len(results))
	seen := map[string]struct{}{}
	for _, result := range results {
		if !isDurableNotePath(result.NotePath) {
			continue
		}
		item := baseDurableNoteItem(result)
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		baseScore := int(result.Score * 100)
		scored, utilityReason := applyUtilityAdjustment(baseScore, item.ID, utility)
		reason := noteReason(result)
		if utilityReason != "" {
			reason += "; " + utilityReason
		}
		candidates = append(candidates, noteCandidate{
			item: projectcontext.CompiledItem{
				ContextItem: item,
				Reason:      reason,
			},
			score:         scored,
			baseScore:     baseScore,
			utilityReason: utilityReason,
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return noteCandidateLess(candidates[i], candidates[j]) })
	selected := make([]projectcontext.CompiledItem, 0, min(limit, len(candidates)))
	if limit <= 0 {
		selected = make([]projectcontext.CompiledItem, 0, len(candidates))
	}
	for _, candidate := range candidates {
		selected = append(selected, candidate.item)
		if limit > 0 && len(selected) == limit {
			break
		}
	}
	return selected
}

func buildNoteCandidates(projectDir string, results []search.Result, generated []projectcontext.ContextItem, packet *livecontext.Packet, graph *structure.BoundaryGraph, utility map[string]ItemUtilitySignal) []noteCandidate {
	touched := touchedBoundarySet(packet)
	adjacent := adjacentBoundarySet(packet)
	pressure := changedBoundaryPressure(packet, graph)
	changedFiles := changedFileSet(packet)
	candidates := []noteCandidate{}
	seen := map[string]struct{}{}

	for _, result := range results {
		if !isDurableNotePath(result.NotePath) {
			continue
		}
		item := baseDurableNoteItem(result)
		text := strings.Join([]string{result.NotePath, result.NoteTitle, result.Heading, result.Snippet}, "\n")
		linked, matchedFiles := linkedBoundariesForText(projectDir, text, graph, changedFiles)
		direct := intersectBoundaryIDs(linked, touched)
		adj := diffBoundaryIDs(intersectBoundaryIDs(linked, adjacent), direct)
		allMatched := dedupeStrings(append(append([]string{}, direct...), adj...))
		baseScore := int(result.Score * 100)
		score, primary := scoreBoundaryCandidate(baseScore, direct, adj, pressure)
		if score == 0 {
			continue
		}
		scored, utilityReason := applyUtilityAdjustment(score, item.ID, utility)
		item.Boundaries = append([]string(nil), allMatched...)
		item.Files = dedupeStrings(append(item.Files, matchedFiles...))
		reason := boundaryCandidateReason("durable note", direct, adj, matchedFiles, pressure, primary)
		if utilityReason != "" {
			reason += "; " + utilityReason
		}
		candidate := noteCandidate{
			item: projectcontext.CompiledItem{
				ContextItem: item,
				Reason:      reason,
			},
			score:             scored,
			baseScore:         score,
			primaryBoundary:   primary,
			matchedBoundaries: append([]string(nil), allMatched...),
			matchedFiles:      matchedFiles,
			utilityReason:     utilityReason,
		}
		if _, ok := seen[candidate.item.ID]; ok {
			continue
		}
		seen[candidate.item.ID] = struct{}{}
		candidates = append(candidates, candidate)
	}

	for _, item := range generated {
		if item.Kind != projectcontext.ContextItemKindGeneratedContext && item.Kind != projectcontext.ContextItemKindWorkflowRule {
			continue
		}
		text := strings.Join([]string{item.Title, item.Summary, readAnchorContent(projectDir, item.Anchor.Path)}, "\n")
		linked, matchedFiles := linkedBoundariesForText(projectDir, text, graph, changedFiles)
		direct := intersectBoundaryIDs(linked, touched)
		adj := diffBoundaryIDs(intersectBoundaryIDs(linked, adjacent), direct)
		allMatched := dedupeStrings(append(append([]string{}, direct...), adj...))
		score, primary := scoreBoundaryCandidate(35, direct, adj, pressure)
		if score == 0 {
			continue
		}
		copyItem := item
		copyItem.Boundaries = append([]string(nil), allMatched...)
		copyItem.Files = dedupeStrings(append(copyItem.Files, matchedFiles...))
		scored, utilityReason := applyUtilityAdjustment(score, copyItem.ID, utility)
		reason := boundaryCandidateReason("generated context", direct, adj, matchedFiles, pressure, primary)
		if utilityReason != "" {
			reason += "; " + utilityReason
		}
		candidate := noteCandidate{
			item: projectcontext.CompiledItem{
				ContextItem: copyItem,
				Reason:      reason,
			},
			score:             scored,
			baseScore:         score,
			primaryBoundary:   primary,
			matchedBoundaries: append([]string(nil), allMatched...),
			matchedFiles:      matchedFiles,
			utilityReason:     utilityReason,
		}
		if _, ok := seen[candidate.item.ID]; ok {
			continue
		}
		seen[candidate.item.ID] = struct{}{}
		candidates = append(candidates, candidate)
	}

	sort.Slice(candidates, func(i, j int) bool { return noteCandidateLess(candidates[i], candidates[j]) })
	return candidates
}

func selectVerificationHints(packet *livecontext.Packet) []projectcontext.VerificationHint {
	if packet == nil {
		return []projectcontext.VerificationHint{}
	}
	hints := make([]projectcontext.VerificationHint, 0, len(packet.Verification.Recipes)+len(packet.PolicyHints))
	selectedRecipes := selectRecipeHints(packet, 4)
	for _, recipe := range selectedRecipes {
		hints = append(hints, projectcontext.VerificationHint{
			ID:              "recipe:" + shortHash(recipe.Source+recipe.Command),
			Label:           recipe.Label,
			Command:         recipe.Command,
			Summary:         recipe.Reason,
			Source:          recipe.Source,
			Strength:        recipe.Strength,
			Reason:          recipe.Reason,
			EstimatedTokens: projectcontext.EstimateTokens(recipe.Label, recipe.Command, recipe.Source, recipe.Strength, recipe.Reason),
		})
	}
	for _, hint := range packet.PolicyHints {
		if !strings.Contains(strings.ToLower(hint.Label), "verification") {
			continue
		}
		hints = append(hints, projectcontext.VerificationHint{
			ID:              "policy:" + shortHash(hint.Source+hint.Label+hint.Excerpt),
			Label:           hint.Label,
			Summary:         clampSummary(hint.Excerpt, 28),
			Source:          hint.Source,
			Strength:        "suggested",
			Reason:          hint.Why,
			EstimatedTokens: projectcontext.EstimateTokens(hint.Label, clampSummary(hint.Excerpt, 28), hint.Source, hint.Why),
		})
	}
	sort.Slice(hints, func(i, j int) bool {
		if hints[i].Strength == hints[j].Strength {
			return hints[i].ID < hints[j].ID
		}
		return hints[i].Strength == "strong"
	})
	return hints
}

func selectRecipeHints(packet *livecontext.Packet, limit int) []livecontext.VerificationRecipe {
	if packet == nil || limit <= 0 {
		return []livecontext.VerificationRecipe{}
	}
	recipes := append([]livecontext.VerificationRecipe(nil), packet.Verification.Recipes...)
	sort.Slice(recipes, func(i, j int) bool {
		return verificationRecipeLess(recipes[i], recipes[j], packet)
	})
	if len(recipes) > limit {
		recipes = recipes[:limit]
	}
	return recipes
}

func verificationRecipeLess(a, b livecontext.VerificationRecipe, packet *livecontext.Packet) bool {
	scoreA := verificationRecipeScore(a, packet)
	scoreB := verificationRecipeScore(b, packet)
	if scoreA == scoreB {
		if a.Source == b.Source {
			return a.Command < b.Command
		}
		return a.Source < b.Source
	}
	return scoreA > scoreB
}

func verificationRecipeScore(recipe livecontext.VerificationRecipe, packet *livecontext.Packet) int {
	command := strings.ToLower(strings.TrimSpace(recipe.Command))
	score := 0
	if recipe.Strength == "strong" {
		score += 100
	} else {
		score += 50
	}
	if strings.Contains(command, "test") && len(packet.NearbyTests) > 0 {
		score += 25
	}
	if strings.Contains(command, "build") && len(packet.Worktree.ChangedFiles) > 0 {
		score += 10
	}
	for _, boundary := range packet.Worktree.TouchedBoundaries {
		root := strings.ToLower(strings.TrimSuffix(boundary.Path, "/"))
		if root != "" && strings.Contains(command, root) {
			score += 35
		}
	}
	for _, test := range packet.NearbyTests {
		dir := strings.ToLower(filepath.ToSlash(filepath.Dir(test.Path)))
		if dir != "" && dir != "." && strings.Contains(command, dir) {
			score += 45
		}
	}
	return score
}

func buildAmbiguities(packet *livecontext.Packet, results []search.Result, graph *structure.BoundaryGraph, selectedNotes []projectcontext.CompiledItem) []string {
	ambiguities := []string{}
	if packet == nil {
		return []string{"live work context was unavailable during compilation"}
	}
	ambiguities = append(ambiguities, packet.Ambiguities...)
	if graph == nil {
		ambiguities = append(ambiguities, "boundary graph was unavailable, so note selection fell back to lexical ranking only")
	}
	if len(packet.Worktree.ChangedFiles) == 0 && !containsAmbiguityFragment(ambiguities, "no changed files") {
		ambiguities = append(ambiguities, "no changed files were detected, so file and boundary selection relied on current repo state rather than an active diff")
	}
	if len(packet.NearbyTests) == 0 && !containsAmbiguityFragment(ambiguities, "no nearby tests") {
		ambiguities = append(ambiguities, "no nearby tests were detected for the current task")
	}
	if len(selectedNotes) == 0 && len(lexicalDurableNotes(results, nil, 1)) == 0 {
		ambiguities = append(ambiguities, "no durable note summaries ranked highly enough to enter the first working set")
	}
	if len(selectedNotes) > 0 {
		boundaryLinked := false
		for _, item := range selectedNotes {
			if len(item.Boundaries) > 0 {
				boundaryLinked = true
				break
			}
		}
		if !boundaryLinked && graph != nil && len(packet.Worktree.TouchedBoundaries) > 0 {
			ambiguities = append(ambiguities, "selected notes did not map cleanly onto the touched boundaries, so lexical note ranking filled the remaining slots")
		}
	}
	sort.Strings(ambiguities)
	return dedupeStrings(ambiguities)
}

func containsAmbiguityFragment(ambiguities []string, fragment string) bool {
	fragment = strings.ToLower(strings.TrimSpace(fragment))
	for _, ambiguity := range ambiguities {
		if strings.Contains(strings.ToLower(ambiguity), fragment) {
			return true
		}
	}
	return false
}

func buildProvenance(packet *projectcontext.CompiledPacket) []projectcontext.PacketProvenance {
	provenance := make([]projectcontext.PacketProvenance, 0, len(packet.BaseContract)+len(packet.WorkingSet.Notes))
	for _, item := range packet.BaseContract {
		provenance = append(provenance, projectcontext.PacketProvenance{
			ItemID:  item.ID,
			Section: "base_contract",
			Anchor:  item.Anchor,
			Reason:  item.Reason,
		})
	}
	for _, item := range packet.WorkingSet.Notes {
		provenance = append(provenance, projectcontext.PacketProvenance{
			ItemID:  item.ID,
			Section: "working_set.notes",
			Anchor:  item.Anchor,
			Reason:  item.Reason,
		})
	}
	sort.Slice(provenance, func(i, j int) bool {
		if provenance[i].Section == provenance[j].Section {
			return provenance[i].ItemID < provenance[j].ItemID
		}
		return provenance[i].Section < provenance[j].Section
	})
	return provenance
}

func renderBudget(w io.Writer, budget projectcontext.PacketBudget) error {
	if _, err := io.WriteString(w, "## Budget\n\n"); err != nil {
		return err
	}
	line := fmt.Sprintf(
		"- Target: %d tokens\n- Used: %d tokens\n- Remaining: %d tokens\n- Reserve base contract: %d tokens\n- Reserve verification: %d tokens\n- Reserve diagnostics: %d tokens\n- Omitted due to budget: %d\n",
		budget.Target,
		budget.Used,
		budget.Remaining,
		budget.ReserveBaseContract,
		budget.ReserveVerification,
		budget.ReserveDiagnostics,
		budget.OmittedDueToBudget,
	)
	if budget.Preset != "" {
		line = fmt.Sprintf("- Preset: `%s`\n%s", budget.Preset, line)
	}
	if _, err := io.WriteString(w, line); err != nil {
		return err
	}
	if budget.MandatoryOverTarget {
		if _, err := io.WriteString(w, "- Mandatory sections exceeded the target budget before optional working-set selection.\n"); err != nil {
			return err
		}
	}
	if len(budget.OmittedCandidates) != 0 {
		if _, err := io.WriteString(w, "- Top omitted candidates:\n"); err != nil {
			return err
		}
		for _, item := range budget.OmittedCandidates {
			if _, err := fmt.Fprintf(w, "  - %s [%s] (`%s`, %d tokens): %s\n", item.Title, item.Section, anchorLabel(item.Anchor), item.EstimatedTokens, item.Reason); err != nil {
				return err
			}
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func renderBoundaries(w io.Writer, items []projectcontext.CompiledBoundary) error {
	if _, err := io.WriteString(w, "### Boundaries\n\n"); err != nil {
		return err
	}
	if len(items) == 0 {
		_, err := io.WriteString(w, "- None yet.\n\n")
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "- `%s` [%s]: %s\n", item.Path, item.Role, item.Reason); err != nil {
			return err
		}
		if len(item.Responsibilities) > 0 {
			if _, err := fmt.Fprintf(w, "  Responsibilities: %s\n", strings.Join(item.Responsibilities, "; ")); err != nil {
				return err
			}
		}
		if len(item.AdjacentBoundaries) > 0 {
			if _, err := fmt.Fprintf(w, "  Adjacent: %s\n", strings.Join(item.AdjacentBoundaries, ", ")); err != nil {
				return err
			}
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func renderFiles(w io.Writer, items []projectcontext.CompiledFile) error {
	if _, err := io.WriteString(w, "### Files\n\n"); err != nil {
		return err
	}
	if len(items) == 0 {
		_, err := io.WriteString(w, "- None yet.\n\n")
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "- `%s` [%s, %s]: %s\n", item.Path, item.Status, item.Source, item.Reason); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func renderTests(w io.Writer, items []projectcontext.CompiledTest) error {
	if _, err := io.WriteString(w, "### Tests\n\n"); err != nil {
		return err
	}
	if len(items) == 0 {
		_, err := io.WriteString(w, "- None yet.\n\n")
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "- `%s` [%s]: %s\n", item.Path, item.Relation, item.Reason); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func renderNotes(w io.Writer, items []projectcontext.CompiledItem) error {
	if _, err := io.WriteString(w, "### Notes\n\n"); err != nil {
		return err
	}
	if len(items) == 0 {
		_, err := io.WriteString(w, "- None yet.\n\n")
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "- %s (`%s`): %s\n  Reason: %s\n", item.Title, anchorLabel(item.Anchor), item.Summary, item.Reason); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func anchorLabel(anchor projectcontext.ContextAnchor) string {
	if anchor.Section == "" {
		return anchor.Path
	}
	return anchor.Path + "#" + anchor.Section
}

func summarizeTask(task string) string {
	return clampSummary(strings.TrimSpace(task), 18)
}

func noteReason(result search.Result) string {
	reason := "ranked highly in local durable-note search for the task"
	if result.NoteType != "" {
		reason = "matched task terms in " + result.NoteType + " note search results"
	}
	if strings.TrimSpace(result.Heading) != "" {
		reason += " with a matching note section"
	}
	return reason
}

func baseDurableNoteItem(result search.Result) projectcontext.ContextItem {
	raw := strings.TrimSpace(strings.Join([]string{result.NoteTitle, result.Heading, result.Snippet}, "\n"))
	if raw == "" {
		raw = strings.TrimSpace(result.NotePath)
	}
	return projectcontext.ContextItem{
		ID:      "durable_note:" + shortHash(result.NotePath+"#"+noteAnchorSection(result)),
		Kind:    projectcontext.ContextItemKindDurableNote,
		Title:   durableNoteTitle(result),
		Summary: clampSummary(result.Snippet, 30),
		Anchor: projectcontext.ContextAnchor{
			Path:    filepath.ToSlash(strings.TrimSpace(result.NotePath)),
			Section: noteAnchorSection(result),
		},
		SourceHash:      shortHash(raw),
		ExpansionCost:   len(strings.Fields(raw)),
		EstimatedTokens: projectcontext.EstimateTokens(result.NoteTitle, result.Heading, clampSummary(result.Snippet, 30), result.NotePath),
	}
}

func durableNoteTitle(result search.Result) string {
	if strings.TrimSpace(result.NoteTitle) != "" {
		return result.NoteTitle
	}
	base := filepath.Base(result.NotePath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func noteAnchorSection(result search.Result) string {
	if strings.TrimSpace(result.Heading) != "" {
		return strings.TrimSpace(result.Heading)
	}
	if strings.TrimSpace(result.NoteTitle) != "" {
		return strings.TrimSpace(result.NoteTitle)
	}
	return ""
}

func orderedTouchedBoundaries(packet *livecontext.Packet, graph *structure.BoundaryGraph) []string {
	if packet == nil {
		return []string{}
	}
	out := make([]string, 0, len(packet.Worktree.TouchedBoundaries))
	seen := map[string]struct{}{}
	for _, boundary := range packet.Worktree.TouchedBoundaries {
		id := strings.TrimSuffix(strings.TrimSpace(boundary.Path), "/")
		if graph != nil {
			if record := graph.BoundaryByID(id); record != nil {
				id = record.ID
			}
		}
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func touchedBoundarySet(packet *livecontext.Packet) []string {
	return orderedTouchedBoundaries(packet, nil)
}

func adjacentBoundarySet(packet *livecontext.Packet) []string {
	if packet == nil {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, boundary := range packet.Worktree.TouchedBoundaries {
		for _, adjacent := range boundary.AdjacentBoundaries {
			adjacent = strings.TrimSpace(adjacent)
			if adjacent == "" {
				continue
			}
			if _, ok := seen[adjacent]; ok {
				continue
			}
			seen[adjacent] = struct{}{}
			out = append(out, adjacent)
		}
	}
	sort.Strings(out)
	return out
}

func changedBoundaryPressure(packet *livecontext.Packet, graph *structure.BoundaryGraph) map[string]int {
	pressure := map[string]int{}
	if packet == nil {
		return pressure
	}
	for _, file := range packet.Worktree.ChangedFiles {
		if graph == nil {
			continue
		}
		record := graph.BoundaryForFile(file.Path)
		if record == nil {
			continue
		}
		pressure[record.ID]++
	}
	if len(pressure) == 0 {
		for _, boundary := range packet.Worktree.TouchedBoundaries {
			id := strings.TrimSuffix(strings.TrimSpace(boundary.Path), "/")
			if id != "" {
				pressure[id]++
			}
		}
	}
	return pressure
}

func changedFileSet(packet *livecontext.Packet) []string {
	if packet == nil {
		return []string{}
	}
	out := make([]string, 0, len(packet.Worktree.ChangedFiles))
	for _, file := range packet.Worktree.ChangedFiles {
		if trimmed := filepath.ToSlash(strings.TrimSpace(file.Path)); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	sort.Strings(out)
	return dedupeStrings(out)
}

func linkedBoundariesForText(projectDir, text string, graph *structure.BoundaryGraph, changedFiles []string) ([]string, []string) {
	_ = projectDir
	if graph == nil {
		return []string{}, []string{}
	}
	lower := strings.ToLower(filepath.ToSlash(strings.TrimSpace(text)))
	if lower == "" {
		return []string{}, []string{}
	}
	boundaries := []string{}
	matchedFiles := []string{}
	for _, file := range changedFiles {
		if strings.Contains(lower, strings.ToLower(file)) {
			matchedFiles = append(matchedFiles, file)
			if record := graph.BoundaryForFile(file); record != nil {
				boundaries = append(boundaries, record.ID)
			}
		}
	}
	for _, boundary := range graph.Boundaries {
		terms := []string{
			strings.ToLower(boundary.ID),
			strings.ToLower(strings.TrimSuffix(boundary.RootPath, "/")),
			strings.ToLower(boundary.Label),
		}
		for _, responsibility := range boundary.Responsibilities {
			terms = append(terms, strings.ToLower(strings.TrimSpace(responsibility)))
		}
		for _, term := range terms {
			if term == "" {
				continue
			}
			if strings.Contains(lower, term) {
				boundaries = append(boundaries, boundary.ID)
				break
			}
		}
	}
	return dedupeStrings(boundaries), dedupeStrings(matchedFiles)
}

func intersectBoundaryIDs(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	for _, item := range b {
		seen[item] = struct{}{}
	}
	out := []string{}
	for _, item := range a {
		if _, ok := seen[item]; ok {
			out = append(out, item)
		}
	}
	return dedupeStrings(out)
}

func diffBoundaryIDs(a, b []string) []string {
	if len(a) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	for _, item := range b {
		seen[item] = struct{}{}
	}
	out := []string{}
	for _, item := range a {
		if _, ok := seen[item]; ok {
			continue
		}
		out = append(out, item)
	}
	return dedupeStrings(out)
}

func scoreBoundaryCandidate(base int, direct, adjacent []string, pressure map[string]int) (int, string) {
	if len(direct) == 0 && len(adjacent) == 0 {
		return 0, ""
	}
	score := base
	primary := ""
	bestPressure := -1
	for _, item := range direct {
		score += 45 + pressure[item]*10
		if pressure[item] > bestPressure {
			primary = item
			bestPressure = pressure[item]
		}
	}
	for _, item := range adjacent {
		score += 20 + pressure[item]*5
		if pressure[item] > bestPressure {
			primary = item
			bestPressure = pressure[item]
		}
	}
	if primary == "" {
		if len(direct) > 0 {
			primary = direct[0]
		} else if len(adjacent) > 0 {
			primary = adjacent[0]
		}
	}
	return score, primary
}

func boundaryCandidateReason(kind string, direct, adjacent, matchedFiles []string, pressure map[string]int, primary string) string {
	parts := []string{kind + " matched task-relevant repo context"}
	if len(direct) > 0 {
		parts = append(parts, "touched boundaries "+strings.Join(direct, ", "))
	}
	if len(adjacent) > 0 {
		parts = append(parts, "adjacent boundaries "+strings.Join(adjacent, ", "))
	}
	if len(matchedFiles) > 0 {
		parts = append(parts, "changed files "+strings.Join(matchedFiles, ", "))
	}
	if primary != "" && pressure[primary] > 1 {
		parts = append(parts, fmt.Sprintf("multiple changed files cluster under %s", primary))
	}
	return strings.Join(parts, "; ")
}

func applyUtilityAdjustment(score int, itemID string, utility map[string]ItemUtilitySignal) (int, string) {
	if len(utility) == 0 {
		return score, ""
	}
	signal, ok := utility[itemID]
	if !ok {
		return score, ""
	}
	switch signal.LikelyUtility {
	case "likely_signal":
		bonus := 18
		if signal.ExpandCount >= 3 || signal.DurableUpdateCount >= 2 {
			bonus = 24
		}
		return score + bonus, utilityAdjustmentReason("boosted", signal)
	case "likely_noise":
		penalty := 14
		if signal.IncludeCount >= 5 && signal.UnusedIncludeCount >= 4 {
			penalty = 20
		}
		return score - penalty, utilityAdjustmentReason("suppressed", signal)
	default:
		return score, ""
	}
}

func utilityAdjustmentReason(action string, signal ItemUtilitySignal) string {
	evidence := []string{}
	if signal.ExpandCount > 0 {
		evidence = append(evidence, fmt.Sprintf("%d recorded expansion(s)", signal.ExpandCount))
	}
	if signal.SuccessfulVerificationCount > 0 {
		evidence = append(evidence, fmt.Sprintf("%d successful verification link(s)", signal.SuccessfulVerificationCount))
	}
	if signal.DurableUpdateCount > 0 {
		evidence = append(evidence, fmt.Sprintf("%d durable update link(s)", signal.DurableUpdateCount))
	}
	if signal.LikelyUtility == "likely_noise" && signal.UnusedIncludeCount > 0 {
		evidence = append(evidence, fmt.Sprintf("%d low-impact include(s)", signal.UnusedIncludeCount))
	}
	if len(evidence) == 0 {
		return ""
	}
	return fmt.Sprintf("%s by local utility signal after %s", action, strings.Join(evidence, ", "))
}

func noteCandidateLess(a, b noteCandidate) bool {
	if a.score == b.score {
		if a.baseScore == b.baseScore {
			if a.item.Title == b.item.Title {
				return a.item.ID < b.item.ID
			}
			return a.item.Title < b.item.Title
		}
		return a.baseScore > b.baseScore
	}
	return a.score > b.score
}

func readAnchorContent(projectDir, anchorPath string) string {
	projectDir = strings.TrimSpace(projectDir)
	anchorPath = filepath.FromSlash(strings.TrimSpace(anchorPath))
	if projectDir == "" || anchorPath == "" {
		return ""
	}
	body, err := os.ReadFile(filepath.Join(projectDir, anchorPath))
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(string(body), "\r\n", "\n")
}

func isDurableNotePath(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return false
	}
	if path == "AGENTS.md" || path == ".brain/policy.yaml" || strings.HasPrefix(path, ".brain/context/") {
		return false
	}
	return strings.HasPrefix(path, "docs/") || strings.HasPrefix(path, ".brain/")
}

func shortHash(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:8])
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

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
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

func trimChangedFiles(items []livecontext.ChangedFile, limit int) []livecontext.ChangedFile {
	if limit > 0 && len(items) > limit {
		return append([]livecontext.ChangedFile(nil), items[:limit]...)
	}
	return append([]livecontext.ChangedFile(nil), items...)
}

func trimBoundaries(items []livecontext.TouchedBoundary, limit int) []livecontext.TouchedBoundary {
	if limit > 0 && len(items) > limit {
		return append([]livecontext.TouchedBoundary(nil), items[:limit]...)
	}
	return append([]livecontext.TouchedBoundary(nil), items...)
}

func trimTests(items []livecontext.NearbyTest, limit int) []livecontext.NearbyTest {
	if limit > 0 && len(items) > limit {
		return append([]livecontext.NearbyTest(nil), items[:limit]...)
	}
	return append([]livecontext.NearbyTest(nil), items...)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
