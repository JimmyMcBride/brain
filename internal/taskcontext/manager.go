package taskcontext

import (
	"crypto/sha256"
	"encoding/hex"
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

type Manager struct {
	Context *projectcontext.Manager
}

type Request struct {
	ProjectDir    string
	Task          string
	TaskSource    string
	SearchResults []search.Result
	LivePacket    *livecontext.Packet
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

	packet := &projectcontext.CompiledPacket{
		Task: projectcontext.CompiledTask{
			Text:    task,
			Summary: summarizeTask(task),
			Source:  taskSource,
		},
		BaseContract: selectBaseContract(baseContract),
		WorkingSet: projectcontext.CompiledWorkingSet{
			Boundaries: selectBoundaries(req.LivePacket, 4),
			Files:      selectFiles(req.LivePacket, 6),
			Tests:      selectTests(req.LivePacket, 4),
			Notes:      selectDurableNotes(req.SearchResults, 4),
		},
		Verification: selectVerificationHints(req.LivePacket),
		Ambiguities:  buildAmbiguities(req.LivePacket, req.SearchResults),
	}
	packet.Provenance = buildProvenance(packet)
	return packet, nil
}

func RenderHuman(w io.Writer, packet *projectcontext.CompiledPacket) error {
	if packet == nil {
		return errors.New("compiled packet is required")
	}
	if _, err := fmt.Fprintf(w, "## Compiled Context Packet\n\n- Task: `%s`\n- Summary: %s\n- Source: `%s`\n\n", packet.Task.Text, packet.Task.Summary, packet.Task.Source); err != nil {
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
			Path:   file.Path,
			Status: file.Status,
			Source: file.Source,
			Reason: file.Why,
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
			Path:   boundary.Path,
			Label:  boundary.Label,
			Role:   boundary.Role,
			Reason: boundary.Why,
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
			Path:     test.Path,
			Relation: test.Relation,
			Reason:   test.Why,
		})
	}
	return tests
}

func selectDurableNotes(results []search.Result, limit int) []projectcontext.CompiledItem {
	selected := make([]projectcontext.CompiledItem, 0, limit)
	seen := map[string]struct{}{}
	for _, result := range results {
		if !isDurableNotePath(result.NotePath) {
			continue
		}
		key := result.NotePath + "#" + result.Heading
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		item := projectcontext.ContextItem{
			ID:    "note:" + shortHash(key),
			Kind:  projectcontext.ContextItemKindDurableNote,
			Title: durableNoteTitle(result),
			Summary: clampSummary(
				strings.TrimSpace(result.Snippet),
				34,
			),
			Anchor: projectcontext.ContextAnchor{
				Path:    filepath.ToSlash(result.NotePath),
				Section: noteAnchorSection(result),
			},
			Files:         []string{filepath.ToSlash(result.NotePath)},
			SourceHash:    shortHash(result.NotePath + result.Heading + result.Snippet),
			ExpansionCost: len(strings.Fields(strings.TrimSpace(result.Snippet))),
		}
		selected = append(selected, projectcontext.CompiledItem{
			ContextItem: item,
			Reason:      noteReason(result),
		})
		if len(selected) == limit {
			break
		}
	}
	return selected
}

func selectVerificationHints(packet *livecontext.Packet) []projectcontext.VerificationHint {
	if packet == nil {
		return []projectcontext.VerificationHint{}
	}
	hints := make([]projectcontext.VerificationHint, 0, len(packet.Verification.Profiles)+len(packet.PolicyHints))
	for _, profile := range packet.Verification.Profiles {
		summary := "Verification profile is not satisfied yet."
		reason := "required verification profile is still missing"
		if profile.Satisfied {
			summary = "Verification profile is already satisfied."
			reason = "recent recorded verification already satisfies this profile"
		}
		if profile.MatchedCommand != "" {
			summary = fmt.Sprintf("%s Matched `%s`.", summary, profile.MatchedCommand)
		}
		hints = append(hints, projectcontext.VerificationHint{
			ID:      "profile:" + profile.Name,
			Label:   profile.Name,
			Summary: summary,
			Source:  ".brain/policy.yaml",
			Reason:  reason,
		})
	}
	for _, hint := range packet.PolicyHints {
		hints = append(hints, projectcontext.VerificationHint{
			ID:      "policy:" + shortHash(hint.Source+hint.Label+hint.Excerpt),
			Label:   hint.Label,
			Summary: clampSummary(hint.Excerpt, 28),
			Source:  hint.Source,
			Reason:  hint.Why,
		})
	}
	sort.Slice(hints, func(i, j int) bool {
		return hints[i].ID < hints[j].ID
	})
	return hints
}

func buildAmbiguities(packet *livecontext.Packet, results []search.Result) []string {
	ambiguities := []string{}
	if packet == nil {
		return []string{"live work context was unavailable during compilation"}
	}
	ambiguities = append(ambiguities, packet.Ambiguities...)
	if len(packet.Worktree.ChangedFiles) == 0 {
		ambiguities = append(ambiguities, "no changed files were detected, so file and boundary selection relied on current repo state rather than an active diff")
	}
	if len(packet.NearbyTests) == 0 {
		ambiguities = append(ambiguities, "no nearby tests were detected for the current task")
	}
	if len(selectDurableNotes(results, 1)) == 0 {
		ambiguities = append(ambiguities, "no durable note summaries ranked highly enough to enter the first working set")
	}
	sort.Strings(ambiguities)
	return dedupeStrings(ambiguities)
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
