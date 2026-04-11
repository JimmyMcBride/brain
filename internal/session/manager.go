package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"brain/internal/history"
	"brain/internal/projectcontext"
)

type Manager struct {
	History *history.Logger
}

type Check struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Details string `json:"details"`
}

type GitSnapshot struct {
	Available bool     `json:"available"`
	Head      string   `json:"head,omitempty"`
	Status    []string `json:"status,omitempty"`
}

type HistoryBaseline struct {
	LastID        string    `json:"last_id,omitempty"`
	LastTimestamp time.Time `json:"last_timestamp,omitempty"`
}

type CommandRun struct {
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Cwd       string    `json:"cwd"`
	Argv      []string  `json:"argv"`
	Command   string    `json:"command"`
	ExitCode  int       `json:"exit_code"`
}

type ActiveSession struct {
	ID                string          `json:"id"`
	Status            string          `json:"status"`
	ProjectDir        string          `json:"project_dir"`
	Task              string          `json:"task"`
	PolicyPath        string          `json:"policy_path"`
	OverridePath      string          `json:"override_path,omitempty"`
	StartedAt         time.Time       `json:"started_at"`
	EndedAt           *time.Time      `json:"ended_at,omitempty"`
	GitBaseline       GitSnapshot     `json:"git_baseline"`
	HistoryBaseline   HistoryBaseline `json:"history_baseline"`
	Checks            []Check         `json:"checks"`
	RequiredDocs      []string        `json:"required_docs"`
	SuggestedCommands []string        `json:"suggested_commands"`
	CommandRuns       []CommandRun    `json:"command_runs,omitempty"`
	TerminalSummary   string          `json:"terminal_summary,omitempty"`
	OverrideReason    string          `json:"override_reason,omitempty"`
}

type StartRequest struct {
	ProjectDir string
	Task       string
	ConfigPath string
}

type StartResult struct {
	Session           ActiveSession `json:"session"`
	RequiredDocs      []string      `json:"required_docs"`
	SuggestedCommands []string      `json:"suggested_commands"`
}

type ValidateRequest struct {
	ProjectDir string
	Stage      string
}

type FinishRequest struct {
	ProjectDir string
	Summary    string
	Force      bool
	Reason     string
}

type AbortRequest struct {
	ProjectDir string
	Reason     string
}

type RunRequest struct {
	ProjectDir    string
	Argv          []string
	CaptureOutput bool
}

type RunResult struct {
	SessionID string `json:"session_id"`
	Command   string `json:"command"`
	ExitCode  int    `json:"exit_code"`
	Stdout    string `json:"stdout,omitempty"`
	Stderr    string `json:"stderr,omitempty"`
	Recorded  bool   `json:"recorded"`
}

type ValidationResult struct {
	OK              bool     `json:"ok"`
	Stage           string   `json:"stage"`
	SessionID       string   `json:"session_id,omitempty"`
	Task            string   `json:"task,omitempty"`
	RepoChanged     bool     `json:"repo_changed"`
	NotesChanged    bool     `json:"notes_changed"`
	MissingCommands []string `json:"missing_commands,omitempty"`
	Obligations     []string `json:"obligations,omitempty"`
	Remediation     []string `json:"remediation,omitempty"`
	Checks          []Check  `json:"checks,omitempty"`
}

type FinishResult struct {
	Status     string           `json:"status"`
	SessionID  string           `json:"session_id,omitempty"`
	Forced     bool             `json:"forced"`
	Validation ValidationResult `json:"validation"`
	LedgerPath string           `json:"ledger_path,omitempty"`
}

func New(historyLog *history.Logger) *Manager {
	return &Manager{History: historyLog}
}

func (m *Manager) Start(ctx context.Context, req StartRequest) (*StartResult, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(req.ProjectDir))
	if err != nil {
		return nil, err
	}
	policy, policyPath, overridePath, err := projectcontext.LoadPolicy(projectDir)
	if err != nil {
		return nil, err
	}
	task := strings.TrimSpace(req.Task)
	if policy.Session.RequireTask && task == "" {
		return nil, errors.New("task is required; use --task")
	}
	activePath := filepath.Join(projectDir, filepath.FromSlash(policy.Session.ActiveFile))
	if policy.Session.SingleActive {
		if active, err := loadActiveSession(activePath); err == nil && active.Status == "active" {
			return nil, fmt.Errorf("active session %s already exists for task %q", active.ID, active.Task)
		}
	}

	checks, err := m.preflightChecks(ctx, projectDir, req.ConfigPath, policy)
	if err != nil {
		return nil, err
	}
	for _, check := range checks {
		if !check.OK {
			return nil, fmt.Errorf("preflight check failed: %s (%s)", check.Name, check.Details)
		}
	}

	gitBaseline := snapshotGit(ctx, projectDir)
	historyBaseline, err := m.historyBaseline()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	active := ActiveSession{
		ID:                fmt.Sprintf("%d", now.UnixNano()),
		Status:            "active",
		ProjectDir:        projectDir,
		Task:              task,
		PolicyPath:        filepath.ToSlash(policyPath),
		OverridePath:      filepath.ToSlash(overridePath),
		StartedAt:         now,
		GitBaseline:       gitBaseline,
		HistoryBaseline:   historyBaseline,
		Checks:            checks,
		RequiredDocs:      append([]string(nil), policy.Preflight.RequiredDocs...),
		SuggestedCommands: expandSuggestedCommands(policy.Preflight.SuggestedCommands, task),
	}
	if err := saveActiveSession(activePath, &active); err != nil {
		return nil, err
	}

	return &StartResult{
		Session:           active,
		RequiredDocs:      active.RequiredDocs,
		SuggestedCommands: active.SuggestedCommands,
	}, nil
}

func defaultProjectDir(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return "."
	}
	return dir
}

func (m *Manager) Validate(ctx context.Context, req ValidateRequest) (*ValidationResult, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(req.ProjectDir))
	if err != nil {
		return nil, err
	}
	policy, _, _, err := projectcontext.LoadPolicy(projectDir)
	if err != nil {
		return nil, err
	}
	active, err := loadActiveSession(filepath.Join(projectDir, filepath.FromSlash(policy.Session.ActiveFile)))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Stage) == "finish" {
		result, err := m.evaluateFinish(ctx, policy, active)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	result := &ValidationResult{
		OK:        true,
		Stage:     "active",
		SessionID: active.ID,
		Task:      active.Task,
		Checks:    append([]Check(nil), active.Checks...),
	}
	for _, doc := range policy.Preflight.RequiredDocs {
		if _, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(doc))); err != nil {
			result.OK = false
			result.Obligations = append(result.Obligations, fmt.Sprintf("missing required doc %s", doc))
			result.Remediation = append(result.Remediation, fmt.Sprintf("run `brain context refresh --project %s` or restore %s", projectDir, doc))
		}
	}
	return result, nil
}

func (m *Manager) Finish(ctx context.Context, req FinishRequest) (*FinishResult, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(req.ProjectDir))
	if err != nil {
		return nil, err
	}
	policy, _, _, err := projectcontext.LoadPolicy(projectDir)
	if err != nil {
		return nil, err
	}
	activePath := filepath.Join(projectDir, filepath.FromSlash(policy.Session.ActiveFile))
	active, err := loadActiveSession(activePath)
	if err != nil {
		return nil, err
	}

	validation, err := m.evaluateFinish(ctx, policy, active)
	if err != nil {
		return nil, err
	}
	if !validation.OK && !req.Force {
		return &FinishResult{
			Status:     "blocked",
			SessionID:  active.ID,
			Validation: *validation,
		}, nil
	}
	if req.Force && strings.TrimSpace(req.Reason) == "" {
		return nil, errors.New("force finish requires --reason")
	}

	now := time.Now().UTC()
	if validation.OK {
		active.Status = "finished"
	} else {
		active.Status = "forced_finished"
	}
	active.EndedAt = &now
	active.TerminalSummary = strings.TrimSpace(req.Summary)
	active.OverrideReason = strings.TrimSpace(req.Reason)
	ledgerPath, err := writeLedger(projectDir, policy, active)
	if err != nil {
		return nil, err
	}
	if err := os.Remove(activePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return &FinishResult{
		Status:     active.Status,
		SessionID:  active.ID,
		Forced:     !validation.OK,
		Validation: *validation,
		LedgerPath: filepath.ToSlash(ledgerPath),
	}, nil
}

func (m *Manager) Abort(ctx context.Context, req AbortRequest) (*FinishResult, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(req.ProjectDir))
	if err != nil {
		return nil, err
	}
	policy, _, _, err := projectcontext.LoadPolicy(projectDir)
	if err != nil {
		return nil, err
	}
	activePath := filepath.Join(projectDir, filepath.FromSlash(policy.Session.ActiveFile))
	active, err := loadActiveSession(activePath)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	active.Status = "aborted"
	active.EndedAt = &now
	active.OverrideReason = strings.TrimSpace(req.Reason)
	ledgerPath, err := writeLedger(projectDir, policy, active)
	if err != nil {
		return nil, err
	}
	if err := os.Remove(activePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return &FinishResult{
		Status:     "aborted",
		SessionID:  active.ID,
		Validation: ValidationResult{OK: true, Stage: "abort", SessionID: active.ID, Task: active.Task},
		LedgerPath: filepath.ToSlash(ledgerPath),
	}, nil
}

func (m *Manager) RunCommand(ctx context.Context, req RunRequest, stdout, stderr io.Writer) (*RunResult, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(req.ProjectDir))
	if err != nil {
		return nil, err
	}
	if len(req.Argv) == 0 {
		return nil, errors.New("session run requires a command after --")
	}
	policy, _, _, err := projectcontext.LoadPolicy(projectDir)
	if err != nil {
		return nil, err
	}
	activePath := filepath.Join(projectDir, filepath.FromSlash(policy.Session.ActiveFile))
	active, err := loadActiveSession(activePath)
	if err != nil {
		return nil, err
	}

	var outBuf, errBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, req.Argv[0], req.Argv[1:]...)
	cmd.Dir = projectDir
	started := time.Now().UTC()
	if req.CaptureOutput {
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
	} else {
		if stdout != nil {
			cmd.Stdout = io.MultiWriter(stdout, &outBuf)
		} else {
			cmd.Stdout = &outBuf
		}
		if stderr != nil {
			cmd.Stderr = io.MultiWriter(stderr, &errBuf)
		} else {
			cmd.Stderr = &errBuf
		}
	}
	exitCode := 0
	runErr := cmd.Run()
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	ended := time.Now().UTC()

	record := CommandRun{
		StartedAt: started,
		EndedAt:   ended,
		Cwd:       filepath.ToSlash(projectDir),
		Argv:      append([]string(nil), req.Argv...),
		Command:   strings.Join(req.Argv, " "),
		ExitCode:  exitCode,
	}
	active.CommandRuns = append(active.CommandRuns, record)
	if err := saveActiveSession(activePath, active); err != nil {
		return nil, err
	}

	result := &RunResult{
		SessionID: active.ID,
		Command:   record.Command,
		ExitCode:  exitCode,
		Recorded:  true,
	}
	if req.CaptureOutput {
		result.Stdout = outBuf.String()
		result.Stderr = errBuf.String()
	}
	if runErr != nil {
		return result, runErr
	}
	return result, nil
}

func (m *Manager) preflightChecks(ctx context.Context, projectDir, configPath string, policy *projectcontext.Policy) ([]Check, error) {
	var checks []Check
	if policy.Preflight.RequireBrainDoctor {
		projectRoot, err := filepath.Abs(defaultProjectDir(projectDir))
		if err != nil {
			checks = append(checks, Check{Name: "brain_doctor", OK: false, Details: err.Error()})
			return checks, nil
		}
		for _, rel := range []string{".brain", ".brain/state"} {
			if _, err := os.Stat(filepath.Join(projectRoot, rel)); err != nil {
				checks = append(checks, Check{Name: "brain_doctor", OK: false, Details: fmt.Sprintf("missing %s", rel)})
				goto docs
			}
		}
		checks = append(checks, Check{Name: "brain_doctor", OK: true, Details: "project-local brain workspace present"})
	}
docs:
	for _, doc := range policy.Preflight.RequiredDocs {
		if _, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(doc))); err != nil {
			checks = append(checks, Check{Name: "required_doc", OK: false, Details: doc})
		} else {
			checks = append(checks, Check{Name: "required_doc", OK: true, Details: doc})
		}
	}
	return checks, nil
}

func (m *Manager) evaluateFinish(ctx context.Context, policy *projectcontext.Policy, active *ActiveSession) (*ValidationResult, error) {
	result := &ValidationResult{
		OK:        true,
		Stage:     "finish",
		SessionID: active.ID,
		Task:      active.Task,
	}
	currentGit := snapshotGit(ctx, active.ProjectDir)
	result.RepoChanged = repoChanged(active.GitBaseline, currentGit)

	entries, err := m.historyAfterBaseline(active.HistoryBaseline)
	if err != nil {
		return nil, err
	}
	qualifyingNotes, _ := filterHistoryEntries(entries, policy.Closeout.AcceptableHistoryOperations, policy.Project.Memory.AcceptedNoteGlobs)
	result.NotesChanged = len(qualifyingNotes) != 0

	if result.RepoChanged && policy.Closeout.RequireMemoryUpdateOnRepoChange && !result.NotesChanged {
		result.OK = false
		result.Obligations = append(result.Obligations, "durable note update required for repo changes")
		result.Remediation = append(result.Remediation, fmt.Sprintf("run `brain edit AGENTS.md ...` or update docs/.brain notes for %s", policy.Project.Name))
	}
	if result.RepoChanged {
		for _, profile := range policy.Closeout.VerificationProfiles {
			if !commandProfileSatisfied(profile, active.CommandRuns) {
				result.OK = false
				result.MissingCommands = append(result.MissingCommands, profile.Name)
				if len(profile.Commands) != 0 {
					result.Remediation = append(result.Remediation, fmt.Sprintf("run `brain session run -- %s`", profile.Commands[0]))
				}
			}
		}
	}
	return result, nil
}

func (m *Manager) historyBaseline() (HistoryBaseline, error) {
	if m.History == nil {
		return HistoryBaseline{}, nil
	}
	entries, err := m.History.All()
	if err != nil {
		return HistoryBaseline{}, err
	}
	if len(entries) == 0 {
		return HistoryBaseline{}, nil
	}
	last := entries[len(entries)-1]
	return HistoryBaseline{
		LastID:        last.ID,
		LastTimestamp: last.Timestamp,
	}, nil
}

func (m *Manager) historyAfterBaseline(baseline HistoryBaseline) ([]history.Entry, error) {
	if m.History == nil {
		return nil, nil
	}
	entries, err := m.History.All()
	if err != nil {
		return nil, err
	}
	if baseline.LastID == "" {
		return entries, nil
	}
	for i, entry := range entries {
		if entry.ID == baseline.LastID {
			return entries[i+1:], nil
		}
	}
	var out []history.Entry
	for _, entry := range entries {
		if entry.Timestamp.After(baseline.LastTimestamp) {
			out = append(out, entry)
		}
	}
	return out, nil
}

func loadActiveSession(path string) (*ActiveSession, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load active session: %w", err)
	}
	var active ActiveSession
	if err := jsonUnmarshal(raw, &active); err != nil {
		return nil, fmt.Errorf("parse active session: %w", err)
	}
	return &active, nil
}

func saveActiveSession(path string, active *ActiveSession) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := jsonMarshal(active)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func writeLedger(projectDir string, policy *projectcontext.Policy, active *ActiveSession) (string, error) {
	ledgerDir := filepath.Join(projectDir, filepath.FromSlash(policy.Session.LedgerDir))
	if err := os.MkdirAll(ledgerDir, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s-%s.json", active.StartedAt.UTC().Format("20060102T150405Z"), active.ID)
	path := filepath.Join(ledgerDir, filename)
	raw, err := jsonMarshal(active)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func expandSuggestedCommands(commands []string, task string) []string {
	out := make([]string, 0, len(commands))
	for _, command := range commands {
		out = append(out, strings.ReplaceAll(command, "{task}", task))
	}
	return out
}

func snapshotGit(ctx context.Context, projectDir string) GitSnapshot {
	if !gitAvailable(ctx, projectDir) {
		return GitSnapshot{}
	}
	head := strings.TrimSpace(runGit(ctx, projectDir, "rev-parse", "HEAD"))
	status := splitNonEmpty(strings.TrimSpace(runGit(ctx, projectDir, "status", "--porcelain=v1")))
	sort.Strings(status)
	return GitSnapshot{
		Available: true,
		Head:      head,
		Status:    status,
	}
}

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

func repoChanged(base, current GitSnapshot) bool {
	if !base.Available || !current.Available {
		return false
	}
	if base.Head != current.Head {
		return true
	}
	if len(base.Status) != len(current.Status) {
		return true
	}
	for i := range base.Status {
		if base.Status[i] != current.Status[i] {
			return true
		}
	}
	return false
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

func filterHistoryEntries(entries []history.Entry, acceptableOps, globs []string) ([]history.Entry, []int) {
	allowed := map[string]struct{}{}
	for _, op := range acceptableOps {
		allowed[op] = struct{}{}
	}
	var out []history.Entry
	var indexes []int
	for i, entry := range entries {
		if _, ok := allowed[entry.Operation]; !ok {
			continue
		}
		if pathMatchesAny(entry.File, globs) || pathMatchesAny(entry.Target, globs) {
			out = append(out, entry)
			indexes = append(indexes, i)
		}
	}
	return out, indexes
}

func commandProfileSatisfied(profile projectcontext.VerificationProfile, runs []CommandRun) bool {
	for _, run := range runs {
		if run.ExitCode != 0 {
			continue
		}
		for _, command := range profile.Commands {
			if run.Command == command {
				return true
			}
		}
	}
	return false
}

func pathMatchesAny(path string, globs []string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	for _, glob := range globs {
		if globMatch(glob, path) {
			return true
		}
	}
	return false
}

func globMatch(glob, path string) bool {
	pattern := regexpQuote(glob)
	pattern = strings.ReplaceAll(pattern, `\*\*`, `.*`)
	pattern = strings.ReplaceAll(pattern, `\*`, `[^/]*`)
	pattern = "^" + pattern + "$"
	matched, _ := regexpMatchString(pattern, filepath.ToSlash(path))
	return matched
}

var regexpMatchString = func(pattern, s string) (bool, error) {
	return regexp.MatchString(pattern, s)
}

var regexpQuote = func(s string) string {
	return regexp.QuoteMeta(s)
}

var jsonMarshal = func(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

var jsonUnmarshal = func(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
