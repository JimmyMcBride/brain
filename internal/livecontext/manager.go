package livecontext

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"brain/internal/session"
	"brain/internal/structure"
)

type Manager struct{}

type Request struct {
	ProjectDir         string
	Task               string
	TaskSource         string
	Session            *session.ActiveSession
	StructuralSnapshot *structure.Snapshot
	Explain            bool
}

type Packet struct {
	Task         TaskInfo     `json:"task"`
	Session      SessionInfo  `json:"session"`
	Worktree     WorktreeInfo `json:"worktree"`
	NearbyTests  []NearbyTest `json:"nearby_tests"`
	Verification Verification `json:"verification"`
	PolicyHints  []PolicyHint `json:"policy_hints"`
	Ambiguities  []string     `json:"ambiguities"`
}

type TaskInfo struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

type SessionInfo struct {
	Active    bool   `json:"active"`
	ID        string `json:"id,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
}

type WorktreeInfo struct {
	GitAvailable      bool              `json:"git_available"`
	BaselineHead      string            `json:"baseline_head,omitempty"`
	CurrentHead       string            `json:"current_head,omitempty"`
	ChangedFiles      []ChangedFile     `json:"changed_files"`
	TouchedBoundaries []TouchedBoundary `json:"touched_boundaries"`
}

type ChangedFile struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Source string `json:"source"`
	Why    string `json:"why"`
}

type TouchedBoundary struct {
	Path  string `json:"path"`
	Label string `json:"label"`
	Role  string `json:"role"`
	Why   string `json:"why"`
}

type NearbyTest struct {
	Path     string `json:"path"`
	Relation string `json:"relation"`
	Why      string `json:"why"`
}

type Verification struct {
	RecentCommands []VerificationCommand `json:"recent_commands"`
	Profiles       []VerificationProfile `json:"profiles"`
}

type VerificationCommand struct {
	Command   string `json:"command"`
	ExitCode  int    `json:"exit_code"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at"`
}

type VerificationProfile struct {
	Name           string `json:"name"`
	Satisfied      bool   `json:"satisfied"`
	MatchedCommand string `json:"matched_command,omitempty"`
}

type PolicyHint struct {
	Source  string `json:"source"`
	Label   string `json:"label"`
	Excerpt string `json:"excerpt"`
	Why     string `json:"why"`
}

func New() *Manager {
	return &Manager{}
}

func (m *Manager) Collect(ctx context.Context, req Request) (*Packet, error) {
	task := strings.TrimSpace(req.Task)
	if task == "" {
		return nil, errors.New("live context requires a task")
	}
	taskSource := strings.TrimSpace(req.TaskSource)
	if taskSource == "" {
		taskSource = "flag"
	}

	currentHead := ""
	gitAvailable := gitAvailable(ctx, req.ProjectDir)
	if gitAvailable {
		currentHead = strings.TrimSpace(runGit(ctx, req.ProjectDir, "rev-parse", "HEAD"))
	}

	sessionInfo := SessionInfo{Active: req.Session != nil}
	baselineHead := ""
	if req.Session != nil {
		sessionInfo.ID = req.Session.ID
		sessionInfo.StartedAt = req.Session.StartedAt.UTC().Format("2006-01-02T15:04:05Z")
		baselineHead = strings.TrimSpace(req.Session.GitBaseline.Head)
	}
	changedFiles := deriveChangedFiles(ctx, req.ProjectDir, req.Session, currentHead, gitAvailable)
	touchedBoundaries, structureAvailable := deriveTouchedBoundaries(changedFiles, req.StructuralSnapshot)
	nearbyTests := deriveNearbyTests(req.ProjectDir, changedFiles, touchedBoundaries, req.StructuralSnapshot)

	packet := &Packet{
		Task: TaskInfo{
			Text:   task,
			Source: taskSource,
		},
		Session: sessionInfo,
		Worktree: WorktreeInfo{
			GitAvailable:      gitAvailable,
			BaselineHead:      baselineHead,
			CurrentHead:       currentHead,
			ChangedFiles:      changedFiles,
			TouchedBoundaries: touchedBoundaries,
		},
		NearbyTests: nearbyTests,
		Verification: Verification{
			RecentCommands: []VerificationCommand{},
			Profiles:       []VerificationProfile{},
		},
		PolicyHints: []PolicyHint{},
		Ambiguities: buildAmbiguities(req.Session, gitAvailable, structureAvailable, changedFiles, nearbyTests, nil),
	}
	return packet, nil
}

func RenderHuman(w io.Writer, packet *Packet, explain bool) error {
	if packet == nil {
		return errors.New("live context packet is required")
	}
	if _, err := fmt.Fprintf(w, "## Task\n\n- Task: `%s`\n- Source: `%s`\n\n", packet.Task.Text, packet.Task.Source); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "## Session\n\n"); err != nil {
		return err
	}
	if packet.Session.Active {
		if _, err := fmt.Fprintf(w, "- Active: yes\n- Session: `%s`\n- Started: `%s`\n\n", packet.Session.ID, packet.Session.StartedAt); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(w, "- Active: no\n\n"); err != nil {
			return err
		}
	}
	if err := renderChangedFiles(w, packet.Worktree); err != nil {
		return err
	}
	if err := renderTouchedBoundaries(w, packet.Worktree.TouchedBoundaries); err != nil {
		return err
	}
	if err := renderNearbyTests(w, packet.NearbyTests); err != nil {
		return err
	}
	if err := renderVerification(w, packet.Verification); err != nil {
		return err
	}
	if len(packet.PolicyHints) > 0 {
		if _, err := io.WriteString(w, "## Policy Hints\n\n"); err != nil {
			return err
		}
		for _, hint := range packet.PolicyHints {
			if _, err := fmt.Fprintf(w, "- %s (`%s`): %s\n", hint.Label, hint.Source, hint.Excerpt); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}
	if len(packet.Ambiguities) > 0 {
		if _, err := io.WriteString(w, "## Ambiguities\n\n"); err != nil {
			return err
		}
		for _, ambiguity := range packet.Ambiguities {
			if _, err := fmt.Fprintf(w, "- %s\n", ambiguity); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}
	if explain {
		if _, err := io.WriteString(w, "## Why These Signals Matter\n\n"); err != nil {
			return err
		}
		for _, line := range explainLines(packet) {
			if _, err := fmt.Fprintf(w, "- %s\n", line); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, "\n## Missing Live Signals\n\n"); err != nil {
			return err
		}
		for _, line := range missingSignals(packet) {
			if _, err := fmt.Fprintf(w, "- %s\n", line); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}
	return nil
}

func renderChangedFiles(w io.Writer, worktree WorktreeInfo) error {
	if _, err := io.WriteString(w, "## Changed Files\n\n"); err != nil {
		return err
	}
	if len(worktree.ChangedFiles) == 0 {
		status := "no changed files detected yet"
		if !worktree.GitAvailable {
			status = "git unavailable"
		}
		_, err := fmt.Fprintf(w, "- %s\n\n", status)
		return err
	}
	for _, file := range worktree.ChangedFiles {
		if _, err := fmt.Fprintf(w, "- `%s` [%s, %s]: %s\n", file.Path, file.Status, file.Source, file.Why); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func renderTouchedBoundaries(w io.Writer, boundaries []TouchedBoundary) error {
	if _, err := io.WriteString(w, "## Touched Boundaries\n\n"); err != nil {
		return err
	}
	if len(boundaries) == 0 {
		_, err := io.WriteString(w, "- None yet.\n\n")
		return err
	}
	for _, boundary := range boundaries {
		if _, err := fmt.Fprintf(w, "- `%s` [%s]: %s\n", boundary.Path, boundary.Role, boundary.Why); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func renderNearbyTests(w io.Writer, tests []NearbyTest) error {
	if _, err := io.WriteString(w, "## Nearby Tests\n\n"); err != nil {
		return err
	}
	if len(tests) == 0 {
		_, err := io.WriteString(w, "- None yet.\n\n")
		return err
	}
	limit := len(tests)
	if limit > 5 {
		limit = 5
	}
	for _, test := range tests[:limit] {
		if _, err := fmt.Fprintf(w, "- `%s` [%s]: %s\n", test.Path, test.Relation, test.Why); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func renderVerification(w io.Writer, verification Verification) error {
	if _, err := io.WriteString(w, "## Verification\n\n"); err != nil {
		return err
	}
	if len(verification.RecentCommands) == 0 && len(verification.Profiles) == 0 {
		_, err := io.WriteString(w, "- No recorded verification yet.\n\n")
		return err
	}
	for _, command := range verification.RecentCommands {
		if _, err := fmt.Fprintf(w, "- `%s` (exit %d)\n", command.Command, command.ExitCode); err != nil {
			return err
		}
	}
	for _, profile := range verification.Profiles {
		status := "missing"
		if profile.Satisfied {
			status = "satisfied"
		}
		if _, err := fmt.Fprintf(w, "- profile `%s`: %s\n", profile.Name, status); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func explainLines(packet *Packet) []string {
	lines := []string{
		"task text defines the current live-work scope",
	}
	if packet.Session.Active {
		lines = append(lines, "active session metadata anchors the live-work view to the current repo workflow")
	} else {
		lines = append(lines, "explicit task fallback is in use because there is no active session")
	}
	if packet.Worktree.GitAvailable {
		lines = append(lines, "git availability allows live-work signals to compare current repo state")
	}
	return lines
}

func missingSignals(packet *Packet) []string {
	var out []string
	if len(packet.Worktree.ChangedFiles) == 0 {
		out = append(out, "changed-file signals are not populated yet")
	}
	if len(packet.Worktree.TouchedBoundaries) == 0 {
		out = append(out, "touched structural boundaries are not populated yet")
	}
	if len(packet.NearbyTests) == 0 {
		out = append(out, "nearby test signals are not populated yet")
	}
	if len(packet.Verification.RecentCommands) == 0 && len(packet.Verification.Profiles) == 0 {
		out = append(out, "verification signals are not populated yet")
	}
	if len(packet.PolicyHints) == 0 {
		out = append(out, "policy hints are not populated yet")
	}
	return out
}

var testFilePattern = regexp.MustCompile(`(_test\.go|\.test\.[^/]+|\.spec\.[^/]+)$`)

func gitAvailable(ctx context.Context, dir string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func runGit(ctx context.Context, dir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func normalizePath(path string) string {
	return filepath.ToSlash(strings.TrimSpace(path))
}

func deriveChangedFiles(ctx context.Context, projectDir string, active *session.ActiveSession, currentHead string, gitAvailable bool) []ChangedFile {
	if !gitAvailable {
		return []ChangedFile{}
	}
	entries := map[string]ChangedFile{}
	if active != nil && strings.TrimSpace(active.GitBaseline.Head) != "" && strings.TrimSpace(active.GitBaseline.Head) != strings.TrimSpace(currentHead) {
		for _, file := range gitDiffFiles(ctx, projectDir, active.GitBaseline.Head, currentHead) {
			existing := entries[file.Path]
			existing.Path = file.Path
			existing.Status = preferChangedStatus(existing.Status, file.Status)
			existing.Source = mergeSource(existing.Source, file.Source)
			existing.Why = file.Why
			entries[file.Path] = existing
		}
	}
	for _, file := range gitStatusFiles(ctx, projectDir) {
		existing := entries[file.Path]
		existing.Path = file.Path
		existing.Status = preferChangedStatus(existing.Status, file.Status)
		existing.Source = mergeSource(existing.Source, file.Source)
		existing.Why = file.Why
		entries[file.Path] = existing
	}
	out := make([]ChangedFile, 0, len(entries))
	for _, file := range entries {
		if strings.TrimSpace(file.Status) == "" {
			file.Status = "unknown"
		}
		out = append(out, file)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func gitDiffFiles(ctx context.Context, projectDir, baseHead, currentHead string) []ChangedFile {
	if strings.TrimSpace(baseHead) == "" || strings.TrimSpace(currentHead) == "" {
		return nil
	}
	out := runGit(ctx, projectDir, "diff", "--name-status", baseHead, currentHead)
	lines := splitNonEmpty(out)
	files := make([]ChangedFile, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		status := mapGitNameStatus(fields[0])
		path := normalizePath(fields[len(fields)-1])
		if isVolatilePath(path) {
			continue
		}
		files = append(files, ChangedFile{
			Path:   path,
			Status: status,
			Source: "commit_range",
			Why:    "changed since session baseline",
		})
	}
	return files
}

func gitStatusFiles(ctx context.Context, projectDir string) []ChangedFile {
	out := runGit(ctx, projectDir, "status", "--porcelain=v1")
	lines := splitRawLines(out)
	files := make([]ChangedFile, 0, len(lines))
	for _, line := range lines {
		path := gitStatusPath(line)
		if path == "" || isVolatilePath(path) {
			continue
		}
		files = append(files, ChangedFile{
			Path:   path,
			Status: mapGitPorcelainStatus(line),
			Source: "worktree",
			Why:    "present in current worktree changes",
		})
	}
	return files
}

func deriveTouchedBoundaries(changedFiles []ChangedFile, snapshot *structure.Snapshot) ([]TouchedBoundary, bool) {
	if snapshot == nil {
		return []TouchedBoundary{}, false
	}
	byPath := map[string]TouchedBoundary{}
	for _, file := range changedFiles {
		if file.Path == "" {
			continue
		}
		var best *structure.Item
		for i := range snapshot.Boundaries {
			item := snapshot.Boundaries[i]
			if strings.HasPrefix(file.Path, item.Path) {
				if best == nil || len(item.Path) > len(best.Path) {
					copy := item
					best = &copy
				}
			}
		}
		if best == nil {
			continue
		}
		byPath[best.Path] = TouchedBoundary{
			Path:  best.Path,
			Label: best.Label,
			Role:  best.Role,
			Why:   "contains changed files",
		}
	}
	out := make([]TouchedBoundary, 0, len(byPath))
	for _, boundary := range byPath {
		out = append(out, boundary)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, true
}

func deriveNearbyTests(projectDir string, changedFiles []ChangedFile, boundaries []TouchedBoundary, snapshot *structure.Snapshot) []NearbyTest {
	testsByPath := map[string]NearbyTest{}
	for _, file := range changedFiles {
		if isTestPath(file.Path) {
			testsByPath[file.Path] = NearbyTest{
				Path:     file.Path,
				Relation: "direct_test_change",
				Why:      "test file changed directly",
			}
		}
	}
	if len(testsByPath) == 0 {
		for _, file := range changedFiles {
			dir := filepath.Dir(filepath.FromSlash(file.Path))
			if dir == "." || dir == "" {
				continue
			}
			entries, err := os.ReadDir(filepath.Join(projectDir, dir))
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				rel := normalizePath(filepath.Join(dir, entry.Name()))
				if !isTestPath(rel) {
					continue
				}
				testsByPath[rel] = NearbyTest{
					Path:     rel,
					Relation: "same_dir",
					Why:      "test surface near changed code",
				}
			}
		}
	}
	if len(testsByPath) == 0 && snapshot != nil {
		for _, boundary := range boundaries {
			for _, testSurface := range snapshot.TestSurfaces {
				if strings.HasPrefix(testSurface.Path, boundary.Path) {
					testsByPath[testSurface.Path] = NearbyTest{
						Path:     testSurface.Path,
						Relation: "same_boundary",
						Why:      "test surface near changed code",
					}
				}
			}
		}
	}
	out := make([]NearbyTest, 0, len(testsByPath))
	for _, test := range testsByPath {
		out = append(out, test)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func splitNonEmpty(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func splitRawLines(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

func mapGitNameStatus(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "unknown"
	}
	switch code[0] {
	case 'M':
		return "modified"
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	default:
		return "unknown"
	}
}

func mapGitPorcelainStatus(line string) string {
	line = strings.TrimSpace(line)
	if len(line) < 2 {
		return "unknown"
	}
	code := strings.TrimSpace(line[:2])
	if code == "" {
		return "unknown"
	}
	if strings.Contains(code, "R") {
		return "renamed"
	}
	if strings.Contains(code, "A") {
		return "added"
	}
	if strings.Contains(code, "D") {
		return "deleted"
	}
	if strings.Contains(code, "M") {
		return "modified"
	}
	return "unknown"
}

func gitStatusPath(line string) string {
	if strings.TrimSpace(line) == "" {
		return ""
	}
	if strings.Contains(line, " -> ") {
		parts := strings.Split(line, " -> ")
		return normalizePath(parts[len(parts)-1])
	}
	if len(line) <= 3 {
		return ""
	}
	return normalizePath(strings.TrimSpace(line[3:]))
}

func isVolatilePath(path string) bool {
	path = normalizePath(path)
	switch {
	case path == ".brain/session.json":
		return true
	case strings.HasPrefix(path, ".brain/sessions/"):
		return true
	case strings.HasPrefix(path, ".brain/state/"):
		return true
	default:
		return false
	}
}

func isTestPath(path string) bool {
	return testFilePattern.MatchString(normalizePath(path))
}

func mergeSource(existing, next string) string {
	if existing == "" {
		return next
	}
	if existing == next {
		return existing
	}
	return "both"
}

func preferChangedStatus(existing, next string) string {
	if existing == "" || existing == "unknown" {
		return next
	}
	if next == "" || next == "unknown" {
		return existing
	}
	return next
}

func buildAmbiguities(active *session.ActiveSession, gitAvailable, structureAvailable bool, changedFiles []ChangedFile, nearbyTests []NearbyTest, verification []VerificationCommand) []string {
	var out []string
	if active == nil {
		out = append(out, "using explicit task text without an active session")
	}
	if !gitAvailable {
		out = append(out, "git is unavailable, so live work signals are limited")
	}
	if len(changedFiles) == 0 {
		out = append(out, "no changed files detected yet")
	}
	if !structureAvailable {
		out = append(out, "structural context is unavailable, so touched boundaries could not be computed")
	}
	if len(nearbyTests) == 0 {
		out = append(out, "no nearby tests detected yet")
	}
	if len(verification) == 0 {
		out = append(out, "no recorded verification commands yet")
	}
	return out
}
