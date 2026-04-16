package taskcontext

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"brain/internal/livecontext"
	"brain/internal/projectcontext"
)

const (
	defaultBudgetDiagnosticsReserve = 48
	budgetTaskReserveOverhead       = 18
	budgetSectionOverhead           = 8
	maxOmittedCandidates            = 3
)

var compileBudgetPresets = map[string]int{
	projectcontext.CompileBudgetPresetSmall:   650,
	projectcontext.CompileBudgetPresetDefault: 900,
	projectcontext.CompileBudgetPresetLarge:   1400,
}

type workingSetBudgetSelection struct {
	Selected projectcontext.CompiledWorkingSet
	Omitted  []projectcontext.BudgetOmittedItem
}

type workingSetCandidate struct {
	section       string
	title         string
	anchor        projectcontext.ContextAnchor
	reason        string
	estimatedCost int
	boundary      *projectcontext.CompiledBoundary
	file          *projectcontext.CompiledFile
	test          *projectcontext.CompiledTest
	note          *projectcontext.CompiledItem
}

func resolveCompileBudget(raw string) (projectcontext.CompileBudget, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return projectcontext.CompileBudget{
			Preset: projectcontext.CompileBudgetPresetDefault,
			Target: compileBudgetPresets[projectcontext.CompileBudgetPresetDefault],
		}, nil
	}
	if target, ok := compileBudgetPresets[value]; ok {
		return projectcontext.CompileBudget{Preset: value, Target: target}, nil
	}
	target, err := strconv.Atoi(value)
	if err != nil || target <= 0 {
		return projectcontext.CompileBudget{}, fmt.Errorf("invalid compile budget %q: expected one of %s or a positive integer token target", raw, compileBudgetPresetList())
	}
	return projectcontext.CompileBudget{Target: target}, nil
}

func compileBudgetPresetList() string {
	keys := make([]string, 0, len(compileBudgetPresets))
	for key := range compileBudgetPresets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

func estimateBaseContractReserve(items []projectcontext.CompiledItem) int {
	total := budgetSectionOverhead
	for _, item := range items {
		total += item.EstimatedTokens
		total += estimateProvenanceTokens("base_contract", item.ID, item.Anchor, item.Reason)
	}
	return total
}

func estimateVerificationReserve(items []projectcontext.VerificationHint) int {
	total := budgetSectionOverhead
	for _, item := range items {
		total += item.EstimatedTokens
	}
	return total
}

func estimateDiagnosticsReserve(task, taskSource string, packet *livecontext.Packet) int {
	total := budgetTaskReserveOverhead + projectcontext.EstimateTokens(task, summarizeTask(task), taskSource)
	if packet != nil && len(packet.Ambiguities) != 0 {
		total += budgetSectionOverhead
		for _, ambiguity := range packet.Ambiguities {
			total += projectcontext.EstimateTokens(ambiguity)
		}
	}
	return total + defaultBudgetDiagnosticsReserve
}

func buildPacketBudget(packet *projectcontext.CompiledPacket, budget projectcontext.CompileBudget, reserveBase, reserveVerification, reserveDiagnostics int, omitted []projectcontext.BudgetOmittedItem) projectcontext.PacketBudget {
	used := estimatePacketTokens(packet)
	remaining := budget.Target - used
	if remaining < 0 {
		remaining = 0
	}
	trimmed := append([]projectcontext.BudgetOmittedItem(nil), omitted...)
	if len(trimmed) > maxOmittedCandidates {
		trimmed = trimmed[:maxOmittedCandidates]
	}
	return projectcontext.PacketBudget{
		Preset:              budget.Preset,
		Target:              budget.Target,
		Used:                used,
		Remaining:           remaining,
		ReserveBaseContract: reserveBase,
		ReserveVerification: reserveVerification,
		ReserveDiagnostics:  reserveDiagnostics,
		OmittedDueToBudget:  len(omitted),
		OmittedCandidates:   trimmed,
		MandatoryOverTarget: reserveBase+reserveVerification+reserveDiagnostics > budget.Target,
	}
}

func estimatePacketTokens(packet *projectcontext.CompiledPacket) int {
	if packet == nil {
		return 0
	}
	total := budgetTaskReserveOverhead + projectcontext.EstimateTokens(packet.Task.Text, packet.Task.Summary, packet.Task.Source)
	if len(packet.BaseContract) != 0 {
		total += budgetSectionOverhead
		for _, item := range packet.BaseContract {
			total += item.EstimatedTokens
		}
	}
	total += estimateWorkingSetTokens(packet.WorkingSet)
	if len(packet.Verification) != 0 {
		total += budgetSectionOverhead
		for _, item := range packet.Verification {
			total += item.EstimatedTokens
		}
	}
	if len(packet.Ambiguities) != 0 {
		total += budgetSectionOverhead
		for _, ambiguity := range packet.Ambiguities {
			total += projectcontext.EstimateTokens(ambiguity)
		}
	}
	if len(packet.Provenance) != 0 {
		total += budgetSectionOverhead
		for _, entry := range packet.Provenance {
			total += estimateProvenanceTokens(entry.Section, entry.ItemID, entry.Anchor, entry.Reason)
		}
	}
	return total + defaultBudgetDiagnosticsReserve
}

func estimateWorkingSetTokens(set projectcontext.CompiledWorkingSet) int {
	total := budgetSectionOverhead
	for _, item := range set.Boundaries {
		total += item.EstimatedTokens
	}
	for _, item := range set.Files {
		total += item.EstimatedTokens
	}
	for _, item := range set.Tests {
		total += item.EstimatedTokens
	}
	for _, item := range set.Notes {
		total += item.EstimatedTokens
	}
	return total
}

func estimateSelectionOverheadReserve() int {
	return budgetSectionOverhead * 2
}

func estimateProvenanceTokens(section, itemID string, anchor projectcontext.ContextAnchor, reason string) int {
	return projectcontext.EstimateTokens(section, itemID, anchor.Path, anchor.Section, reason)
}

func selectBudgetedWorkingSet(remaining int, boundaries []projectcontext.CompiledBoundary, files []projectcontext.CompiledFile, tests []projectcontext.CompiledTest, notes []projectcontext.CompiledItem) workingSetBudgetSelection {
	queues := [][]workingSetCandidate{
		makeBoundaryCandidates(boundaries),
		makeFileCandidates(files),
		makeTestCandidates(tests),
		makeNoteCandidates(notes),
	}
	indexes := make([]int, len(queues))
	selected := projectcontext.CompiledWorkingSet{}
	omitted := []projectcontext.BudgetOmittedItem{}

	for {
		progress := false
		for i := range queues {
			for indexes[i] < len(queues[i]) {
				candidate := queues[i][indexes[i]]
				indexes[i]++
				if candidate.estimatedCost > remaining {
					omitted = append(omitted, candidate.omittedItem())
					continue
				}
				addWorkingSetCandidate(&selected, candidate)
				remaining -= candidate.estimatedCost
				progress = true
				break
			}
		}
		if !progress {
			break
		}
	}

	for i := range queues {
		for ; indexes[i] < len(queues[i]); indexes[i]++ {
			omitted = append(omitted, queues[i][indexes[i]].omittedItem())
		}
	}
	return workingSetBudgetSelection{
		Selected: selected,
		Omitted:  omitted,
	}
}

func makeBoundaryCandidates(items []projectcontext.CompiledBoundary) []workingSetCandidate {
	out := make([]workingSetCandidate, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, workingSetCandidate{
			section:       "working_set.boundaries",
			title:         item.Label,
			anchor:        projectcontext.ContextAnchor{Path: item.Path, Section: item.Label},
			reason:        item.Reason,
			estimatedCost: item.EstimatedTokens,
			boundary:      &item,
		})
	}
	return out
}

func makeFileCandidates(items []projectcontext.CompiledFile) []workingSetCandidate {
	out := make([]workingSetCandidate, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, workingSetCandidate{
			section:       "working_set.files",
			title:         item.Path,
			anchor:        projectcontext.ContextAnchor{Path: item.Path},
			reason:        item.Reason,
			estimatedCost: item.EstimatedTokens,
			file:          &item,
		})
	}
	return out
}

func makeTestCandidates(items []projectcontext.CompiledTest) []workingSetCandidate {
	out := make([]workingSetCandidate, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, workingSetCandidate{
			section:       "working_set.tests",
			title:         item.Path,
			anchor:        projectcontext.ContextAnchor{Path: item.Path},
			reason:        item.Reason,
			estimatedCost: item.EstimatedTokens,
			test:          &item,
		})
	}
	return out
}

func makeNoteCandidates(items []projectcontext.CompiledItem) []workingSetCandidate {
	out := make([]workingSetCandidate, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, workingSetCandidate{
			section:       "working_set.notes",
			title:         item.Title,
			anchor:        item.Anchor,
			reason:        item.Reason,
			estimatedCost: item.EstimatedTokens + estimateProvenanceTokens("working_set.notes", item.ID, item.Anchor, item.Reason),
			note:          &item,
		})
	}
	return out
}

func (c workingSetCandidate) omittedItem() projectcontext.BudgetOmittedItem {
	return projectcontext.BudgetOmittedItem{
		ID:              omittedItemID(c),
		Section:         c.section,
		Title:           c.title,
		Anchor:          c.anchor,
		EstimatedTokens: c.estimatedCost,
		Reason:          c.reason,
	}
}

func omittedItemID(c workingSetCandidate) string {
	switch {
	case c.note != nil:
		return c.note.ID
	case c.boundary != nil:
		return "boundary:" + c.boundary.Path
	case c.file != nil:
		return "file:" + c.file.Path
	case c.test != nil:
		return "test:" + c.test.Path
	default:
		return c.title
	}
}

func addWorkingSetCandidate(set *projectcontext.CompiledWorkingSet, candidate workingSetCandidate) {
	switch {
	case candidate.boundary != nil:
		set.Boundaries = append(set.Boundaries, *candidate.boundary)
	case candidate.file != nil:
		set.Files = append(set.Files, *candidate.file)
	case candidate.test != nil:
		set.Tests = append(set.Tests, *candidate.test)
	case candidate.note != nil:
		set.Notes = append(set.Notes, *candidate.note)
	}
}
