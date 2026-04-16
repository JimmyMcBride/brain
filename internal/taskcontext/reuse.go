package taskcontext

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"brain/internal/livecontext"
	"brain/internal/projectcontext"
	"brain/internal/search"
)

func (m *Manager) BuildFingerprintInputs(req Request) (projectcontext.PacketFingerprintInputs, error) {
	task := strings.TrimSpace(req.Task)
	if task == "" {
		return projectcontext.PacketFingerprintInputs{}, errors.New("context compile requires a task")
	}
	taskSource := strings.TrimSpace(req.TaskSource)
	if taskSource == "" {
		taskSource = "flag"
	}
	budget, err := resolveCompileBudget(strings.TrimSpace(req.Budget))
	if err != nil {
		return projectcontext.PacketFingerprintInputs{}, err
	}
	sourceSummaries, err := m.Context.BuildSourceSummaryItems(req.ProjectDir)
	if err != nil {
		return projectcontext.PacketFingerprintInputs{}, err
	}
	verificationHints := selectVerificationHints(req.LivePacket)
	inputs := projectcontext.PacketFingerprintInputs{
		TaskText:                 task,
		TaskSummary:              summarizeTask(task),
		TaskSource:               taskSource,
		BudgetPreset:             budget.Preset,
		BudgetTarget:             budget.Target,
		ChangedFiles:             fingerprintChangedFiles(req.LivePacket),
		TouchedBoundaries:        fingerprintTouchedBoundaries(req.LivePacket),
		DurableSearchSignals:     fingerprintDurableSearchSignals(req.SearchResults),
		SourceSummaryHashes:      fingerprintSourceSummaryHashes(sourceSummaries),
		VerificationRequirements: fingerprintVerificationHints(verificationHints),
	}
	return inputs, nil
}

func RenderCompileResponseHuman(w io.Writer, response *projectcontext.CompileResponse) error {
	if response == nil {
		return errors.New("compile response is required")
	}
	if _, err := fmt.Fprintf(
		w,
		"## Compiled Context Packet\n\n- Hash: `%s`\n- Task: `%s`\n- Summary: %s\n- Source: `%s`\n- Cache status: `%s`\n- Fingerprint: `%s`\n- Full packet included: %t\n\n",
		response.PacketHash,
		response.Task.Text,
		response.Task.Summary,
		response.Task.Source,
		response.CacheStatus,
		response.Fingerprint,
		response.FullPacketIncluded,
	); err != nil {
		return err
	}
	if err := renderBudget(w, response.Budget); err != nil {
		return err
	}
	if err := renderCompileLineage(w, response); err != nil {
		return err
	}
	if !response.FullPacketIncluded {
		return nil
	}
	packet := response.ToCompiledPacket()
	if packet == nil {
		return errors.New("full packet was requested but packet sections are unavailable")
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

func renderCompileLineage(w io.Writer, response *projectcontext.CompileResponse) error {
	if _, err := io.WriteString(w, "## Lineage\n\n"); err != nil {
		return err
	}
	if response.ReusedFrom != "" {
		if _, err := fmt.Fprintf(w, "- Reused from: `%s`\n", response.ReusedFrom); err != nil {
			return err
		}
	}
	if response.DeltaFrom != "" {
		if _, err := fmt.Fprintf(w, "- Delta from: `%s`\n", response.DeltaFrom); err != nil {
			return err
		}
	}
	if response.FallbackReason != "" {
		if _, err := fmt.Fprintf(w, "- Full-packet fallback: %s\n", response.FallbackReason); err != nil {
			return err
		}
	}
	for _, reason := range response.InvalidationReasons {
		if _, err := fmt.Fprintf(w, "- Invalidation: %s\n", reason); err != nil {
			return err
		}
	}
	if len(response.ChangedSections) != 0 {
		if _, err := fmt.Fprintf(w, "- Changed sections: %s\n", strings.Join(response.ChangedSections, ", ")); err != nil {
			return err
		}
	}
	if len(response.ChangedItemIDs) != 0 {
		if _, err := fmt.Fprintf(w, "- Changed item ids: %s\n", strings.Join(response.ChangedItemIDs, ", ")); err != nil {
			return err
		}
	}
	if !response.FullPacketIncluded {
		if _, err := io.WriteString(w, "- Unchanged packet sections were not re-emitted. Use `brain context explain --last` to inspect the full recorded packet or `brain context compile --fresh` to force a standalone full packet.\n"); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func PacketDiff(previous, current *projectcontext.CompiledPacket) ([]string, []string) {
	if previous == nil || current == nil {
		return nil, nil
	}
	changedSections := []string{}
	changedItems := map[string]struct{}{}

	if previous.Task != current.Task {
		changedSections = append(changedSections, "task")
	}
	if budgetSignature(previous.Budget) != budgetSignature(current.Budget) {
		changedSections = append(changedSections, "budget")
	}
	if ids := diffCompiledItems(previous.BaseContract, current.BaseContract); len(ids) != 0 {
		changedSections = append(changedSections, "base_contract")
		for _, id := range ids {
			changedItems[id] = struct{}{}
		}
	}
	if ids := diffCompiledBoundaries(previous.WorkingSet.Boundaries, current.WorkingSet.Boundaries); len(ids) != 0 {
		changedSections = append(changedSections, "working_set.boundaries")
		for _, id := range ids {
			changedItems[id] = struct{}{}
		}
	}
	if ids := diffCompiledFiles(previous.WorkingSet.Files, current.WorkingSet.Files); len(ids) != 0 {
		changedSections = append(changedSections, "working_set.files")
		for _, id := range ids {
			changedItems[id] = struct{}{}
		}
	}
	if ids := diffCompiledTests(previous.WorkingSet.Tests, current.WorkingSet.Tests); len(ids) != 0 {
		changedSections = append(changedSections, "working_set.tests")
		for _, id := range ids {
			changedItems[id] = struct{}{}
		}
	}
	if ids := diffCompiledItems(previous.WorkingSet.Notes, current.WorkingSet.Notes); len(ids) != 0 {
		changedSections = append(changedSections, "working_set.notes")
		for _, id := range ids {
			changedItems[id] = struct{}{}
		}
	}
	if ids := diffVerificationHints(previous.Verification, current.Verification); len(ids) != 0 {
		changedSections = append(changedSections, "verification")
		for _, id := range ids {
			changedItems[id] = struct{}{}
		}
	}
	if !equalStringSlices(previous.Ambiguities, current.Ambiguities) {
		changedSections = append(changedSections, "ambiguities")
	}
	if ids := diffProvenance(previous.Provenance, current.Provenance); len(ids) != 0 {
		changedSections = append(changedSections, "provenance")
		for _, id := range ids {
			changedItems[id] = struct{}{}
		}
	}

	return sortedCopy(changedSections), mapKeys(changedItems)
}

func fingerprintChangedFiles(packet *livecontext.Packet) []string {
	if packet == nil {
		return nil
	}
	values := make([]string, 0, len(packet.Worktree.ChangedFiles))
	for _, file := range packet.Worktree.ChangedFiles {
		values = append(values, strings.Join([]string{
			filepath.ToSlash(strings.TrimSpace(file.Path)),
			strings.TrimSpace(file.Status),
			strings.TrimSpace(file.Source),
		}, "|"))
	}
	return sortedCopy(values)
}

func fingerprintTouchedBoundaries(packet *livecontext.Packet) []string {
	if packet == nil {
		return nil
	}
	values := make([]string, 0, len(packet.Worktree.TouchedBoundaries))
	for _, boundary := range packet.Worktree.TouchedBoundaries {
		values = append(values, strings.Join([]string{
			filepath.ToSlash(strings.TrimSpace(boundary.Path)),
			strings.TrimSpace(boundary.Label),
			strings.TrimSpace(boundary.Role),
		}, "|"))
	}
	return sortedCopy(values)
}

func fingerprintDurableSearchSignals(results []search.Result) []string {
	values := []string{}
	for _, result := range results {
		if !isDurableNotePath(result.NotePath) {
			continue
		}
		snippetHash := shortHash(strings.TrimSpace(result.Snippet))
		values = append(values, strings.Join([]string{
			filepath.ToSlash(strings.TrimSpace(result.NotePath)),
			strings.TrimSpace(result.Heading),
			strings.TrimSpace(result.ModifiedAt),
			snippetHash,
		}, "|"))
	}
	return sortedCopy(values)
}

func fingerprintSourceSummaryHashes(items []projectcontext.ContextItem) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, item.ID+"|"+item.SourceHash)
	}
	return sortedCopy(values)
}

func fingerprintVerificationHints(hints []projectcontext.VerificationHint) []string {
	values := make([]string, 0, len(hints))
	for _, hint := range hints {
		values = append(values, strings.Join([]string{
			strings.TrimSpace(hint.ID),
			strings.TrimSpace(hint.Label),
			strings.TrimSpace(hint.Command),
			strings.TrimSpace(hint.Source),
			strings.TrimSpace(hint.Strength),
			strings.TrimSpace(hint.Summary),
		}, "|"))
	}
	return sortedCopy(values)
}

func diffCompiledItems(previous, current []projectcontext.CompiledItem) []string {
	return diffNamed(
		len(previous),
		len(current),
		func(i int) (string, string) {
			item := previous[i]
			return item.ID, strings.Join([]string{
				item.Title,
				item.Summary,
				item.Anchor.Path,
				item.Anchor.Section,
				item.Reason,
				strings.Join(sortedCopy(item.Boundaries), ","),
				strings.Join(sortedCopy(item.Files), ","),
				item.SourceHash,
			}, "|")
		},
		func(i int) (string, string) {
			item := current[i]
			return item.ID, strings.Join([]string{
				item.Title,
				item.Summary,
				item.Anchor.Path,
				item.Anchor.Section,
				item.Reason,
				strings.Join(sortedCopy(item.Boundaries), ","),
				strings.Join(sortedCopy(item.Files), ","),
				item.SourceHash,
			}, "|")
		},
	)
}

func diffCompiledBoundaries(previous, current []projectcontext.CompiledBoundary) []string {
	return diffNamed(
		len(previous),
		len(current),
		func(i int) (string, string) {
			item := previous[i]
			return "boundary:" + item.Path, strings.Join([]string{
				item.Path,
				item.Label,
				item.Role,
				item.Reason,
				strings.Join(sortedCopy(item.AdjacentBoundaries), ","),
				strings.Join(sortedCopy(item.Responsibilities), ","),
			}, "|")
		},
		func(i int) (string, string) {
			item := current[i]
			return "boundary:" + item.Path, strings.Join([]string{
				item.Path,
				item.Label,
				item.Role,
				item.Reason,
				strings.Join(sortedCopy(item.AdjacentBoundaries), ","),
				strings.Join(sortedCopy(item.Responsibilities), ","),
			}, "|")
		},
	)
}

func diffCompiledFiles(previous, current []projectcontext.CompiledFile) []string {
	return diffNamed(
		len(previous),
		len(current),
		func(i int) (string, string) {
			item := previous[i]
			return "file:" + item.Path, strings.Join([]string{item.Path, item.Status, item.Source, item.Reason}, "|")
		},
		func(i int) (string, string) {
			item := current[i]
			return "file:" + item.Path, strings.Join([]string{item.Path, item.Status, item.Source, item.Reason}, "|")
		},
	)
}

func diffCompiledTests(previous, current []projectcontext.CompiledTest) []string {
	return diffNamed(
		len(previous),
		len(current),
		func(i int) (string, string) {
			item := previous[i]
			return "test:" + item.Path, strings.Join([]string{item.Path, item.Relation, item.Reason}, "|")
		},
		func(i int) (string, string) {
			item := current[i]
			return "test:" + item.Path, strings.Join([]string{item.Path, item.Relation, item.Reason}, "|")
		},
	)
}

func diffVerificationHints(previous, current []projectcontext.VerificationHint) []string {
	return diffNamed(
		len(previous),
		len(current),
		func(i int) (string, string) {
			item := previous[i]
			return item.ID, strings.Join([]string{item.Label, item.Command, item.Summary, item.Source, item.Strength, item.Reason}, "|")
		},
		func(i int) (string, string) {
			item := current[i]
			return item.ID, strings.Join([]string{item.Label, item.Command, item.Summary, item.Source, item.Strength, item.Reason}, "|")
		},
	)
}

func diffProvenance(previous, current []projectcontext.PacketProvenance) []string {
	return diffNamed(
		len(previous),
		len(current),
		func(i int) (string, string) {
			item := previous[i]
			return item.ItemID, strings.Join([]string{item.Section, item.Anchor.Path, item.Anchor.Section, item.Reason}, "|")
		},
		func(i int) (string, string) {
			item := current[i]
			return item.ItemID, strings.Join([]string{item.Section, item.Anchor.Path, item.Anchor.Section, item.Reason}, "|")
		},
	)
}

func diffNamed(previousCount, currentCount int, previous func(int) (string, string), current func(int) (string, string)) []string {
	prevMap := map[string]string{}
	currMap := map[string]string{}
	for i := 0; i < previousCount; i++ {
		id, signature := previous(i)
		prevMap[id] = signature
	}
	for i := 0; i < currentCount; i++ {
		id, signature := current(i)
		currMap[id] = signature
	}
	changed := map[string]struct{}{}
	for id, signature := range prevMap {
		if currSig, ok := currMap[id]; !ok || currSig != signature {
			changed[id] = struct{}{}
		}
	}
	for id, signature := range currMap {
		if prevSig, ok := prevMap[id]; !ok || prevSig != signature {
			changed[id] = struct{}{}
		}
	}
	return mapKeys(changed)
}

func mapKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func sortedCopy(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func budgetSignature(budget projectcontext.PacketBudget) string {
	omitted := make([]string, 0, len(budget.OmittedCandidates))
	for _, item := range budget.OmittedCandidates {
		omitted = append(omitted, strings.Join([]string{
			item.Section,
			item.ID,
			item.Title,
			item.Anchor.Path,
			item.Anchor.Section,
			fmt.Sprintf("%d", item.EstimatedTokens),
			item.Reason,
		}, "|"))
	}
	return strings.Join([]string{
		budget.Preset,
		fmt.Sprintf("%d", budget.Target),
		fmt.Sprintf("%d", budget.Used),
		fmt.Sprintf("%d", budget.Remaining),
		fmt.Sprintf("%d", budget.ReserveBaseContract),
		fmt.Sprintf("%d", budget.ReserveVerification),
		fmt.Sprintf("%d", budget.ReserveDiagnostics),
		fmt.Sprintf("%d", budget.OmittedDueToBudget),
		fmt.Sprintf("%t", budget.MandatoryOverTarget),
		strings.Join(sortedCopy(omitted), ","),
	}, "|")
}
