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
	"runtime"
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

type PacketInclusionReason struct {
	ItemID  string `json:"item_id"`
	Section string `json:"section"`
	Reason  string `json:"reason"`
}

type PacketTelemetryEventType string

const (
	PacketTelemetryEventCompiled      PacketTelemetryEventType = "packet_compiled"
	PacketTelemetryEventExpanded      PacketTelemetryEventType = "item_expanded"
	PacketTelemetryEventVerification  PacketTelemetryEventType = "verification_recorded"
	PacketTelemetryEventDurableUpdate PacketTelemetryEventType = "durable_update_recorded"
	PacketTelemetryEventSessionClosed PacketTelemetryEventType = "session_closed"
	packetTelemetryVersion                                     = 1
	maxSessionPacketRecords                                    = 64
	maxSessionTelemetryEvents                                  = 256
)

type PacketTelemetryEvent struct {
	Type        PacketTelemetryEventType `json:"type"`
	Timestamp   time.Time                `json:"timestamp"`
	SessionID   string                   `json:"session_id"`
	PacketHash  string                   `json:"packet_hash,omitempty"`
	ItemID      string                   `json:"item_id,omitempty"`
	AnchorPath  string                   `json:"anchor_path,omitempty"`
	Command     string                   `json:"command,omitempty"`
	Success     *bool                    `json:"success,omitempty"`
	File        string                   `json:"file,omitempty"`
	Operation   string                   `json:"operation,omitempty"`
	CloseStatus string                   `json:"close_status,omitempty"`
	Metadata    map[string]any           `json:"metadata,omitempty"`
}

type PacketRecord struct {
	PacketHash       string                         `json:"packet_hash"`
	TaskText         string                         `json:"task_text"`
	TaskSummary      string                         `json:"task_summary"`
	TaskSource       string                         `json:"task_source"`
	CompiledAt       time.Time                      `json:"compiled_at"`
	IncludedItemIDs  []string                       `json:"included_item_ids"`
	IncludedAnchors  []projectcontext.ContextAnchor `json:"included_anchors"`
	InclusionReasons []PacketInclusionReason        `json:"inclusion_reasons"`
}

type ActiveSession struct {
	ID                string                 `json:"id"`
	Status            string                 `json:"status"`
	ProjectDir        string                 `json:"project_dir"`
	Task              string                 `json:"task"`
	PolicyPath        string                 `json:"policy_path"`
	OverridePath      string                 `json:"override_path,omitempty"`
	StartedAt         time.Time              `json:"started_at"`
	EndedAt           *time.Time             `json:"ended_at,omitempty"`
	GitBaseline       GitSnapshot            `json:"git_baseline"`
	HistoryBaseline   HistoryBaseline        `json:"history_baseline"`
	Checks            []Check                `json:"checks"`
	RequiredDocs      []string               `json:"required_docs"`
	SuggestedCommands []string               `json:"suggested_commands"`
	CommandRuns       []CommandRun           `json:"command_runs,omitempty"`
	PacketRecords     []PacketRecord         `json:"packet_records,omitempty"`
	TelemetryVersion  int                    `json:"telemetry_version,omitempty"`
	TelemetryEvents   []PacketTelemetryEvent `json:"telemetry_events,omitempty"`
	TerminalSummary   string                 `json:"terminal_summary,omitempty"`
	OverrideReason    string                 `json:"override_reason,omitempty"`
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
	OK                bool     `json:"ok"`
	Stage             string   `json:"stage"`
	SessionID         string   `json:"session_id,omitempty"`
	Task              string   `json:"task,omitempty"`
	RepoChanged       bool     `json:"repo_changed"`
	NotesChanged      bool     `json:"notes_changed"`
	MemorySatisfiedBy string   `json:"memory_satisfied_by,omitempty"`
	MissingCommands   []string `json:"missing_commands,omitempty"`
	Obligations       []string `json:"obligations,omitempty"`
	Remediation       []string `json:"remediation,omitempty"`
	Checks            []Check  `json:"checks,omitempty"`
}

type FinishResult struct {
	Status     string           `json:"status"`
	SessionID  string           `json:"session_id,omitempty"`
	Forced     bool             `json:"forced"`
	Validation ValidationResult `json:"validation"`
	LedgerPath string           `json:"ledger_path,omitempty"`
}

const (
	sessionLockRetryDelay = 20 * time.Millisecond
	sessionLockTimeout    = 2 * time.Second
	sessionLockStaleAfter = 30 * time.Second
)

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
		TelemetryVersion:  packetTelemetryVersion,
	}
	if err := withSessionLock(activePath, func() error {
		if policy.Session.SingleActive {
			existing, err := loadActiveSessionIfExists(activePath)
			if err != nil {
				return err
			}
			if existing != nil && existing.Status == "active" {
				return fmt.Errorf("active session %s already exists for task %q", existing.ID, existing.Task)
			}
		}
		return saveActiveSession(activePath, &active)
	}); err != nil {
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

func (m *Manager) Active(projectDir string) (*ActiveSession, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return nil, err
	}
	policy, _, _, err := projectcontext.LoadPolicy(projectDir)
	if err != nil {
		return nil, err
	}
	return loadActiveSessionIfExists(filepath.Join(projectDir, filepath.FromSlash(policy.Session.ActiveFile)))
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
	var result *FinishResult
	if err := withSessionLock(activePath, func() error {
		active, err := loadActiveSession(activePath)
		if err != nil {
			return err
		}

		validation, err := m.evaluateFinish(ctx, policy, active)
		if err != nil {
			return err
		}
		if !validation.OK && !req.Force {
			result = &FinishResult{
				Status:     "blocked",
				SessionID:  active.ID,
				Validation: *validation,
			}
			return nil
		}
		if req.Force && strings.TrimSpace(req.Reason) == "" {
			return errors.New("force finish requires --reason")
		}
		entries, err := m.historyAfterBaseline(active.HistoryBaseline)
		if err != nil {
			return err
		}
		appendDurableUpdateTelemetry(active, entries, policy.Closeout.AcceptableHistoryOperations, policy.Project.Memory.AcceptedNoteGlobs)

		now := time.Now().UTC()
		if validation.OK {
			active.Status = "finished"
		} else {
			active.Status = "forced_finished"
		}
		active.EndedAt = &now
		active.TerminalSummary = strings.TrimSpace(req.Summary)
		active.OverrideReason = strings.TrimSpace(req.Reason)
		appendSessionClosedTelemetry(active, now, active.Status, validation)
		ledgerPath, err := writeLedger(projectDir, policy, active)
		if err != nil {
			return err
		}
		if err := removeActiveSession(activePath); err != nil {
			return err
		}
		result = &FinishResult{
			Status:     active.Status,
			SessionID:  active.ID,
			Forced:     !validation.OK,
			Validation: *validation,
			LedgerPath: filepath.ToSlash(ledgerPath),
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
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
	var result *FinishResult
	if err := withSessionLock(activePath, func() error {
		active, err := loadActiveSession(activePath)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		active.Status = "aborted"
		active.EndedAt = &now
		active.OverrideReason = strings.TrimSpace(req.Reason)
		appendSessionClosedTelemetry(active, now, active.Status, &ValidationResult{OK: true, Stage: "abort", SessionID: active.ID, Task: active.Task})
		ledgerPath, err := writeLedger(projectDir, policy, active)
		if err != nil {
			return err
		}
		if err := removeActiveSession(activePath); err != nil {
			return err
		}
		result = &FinishResult{
			Status:     "aborted",
			SessionID:  active.ID,
			Validation: ValidationResult{OK: true, Stage: "abort", SessionID: active.ID, Task: active.Task},
			LedgerPath: filepath.ToSlash(ledgerPath),
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
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
	active, err := loadActiveSessionForRecording(activePath)
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

	result := &RunResult{
		SessionID: active.ID,
		Command:   record.Command,
		ExitCode:  exitCode,
	}
	if req.CaptureOutput {
		result.Stdout = outBuf.String()
		result.Stderr = errBuf.String()
	}
	if err := appendCommandRun(activePath, active.ID, record); err != nil {
		result.Recorded = false
		return result, err
	}
	result.Recorded = true
	if runErr != nil {
		return result, runErr
	}
	return result, nil
}

func (m *Manager) RecordCompiledPacket(projectDir, sessionID string, packet *projectcontext.CompiledPacket) error {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return err
	}
	if packet == nil {
		return errors.New("compiled packet is required")
	}
	policy, _, _, err := projectcontext.LoadPolicy(projectDir)
	if err != nil {
		return err
	}
	activePath := filepath.Join(projectDir, filepath.FromSlash(policy.Session.ActiveFile))
	record := packetRecordFromCompiledPacket(packet)
	return appendPacketRecord(activePath, sessionID, record)
}

func (m *Manager) RecordPacketExpansion(projectDir, path string) error {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return err
	}
	policy, _, _, err := projectcontext.LoadPolicy(projectDir)
	if err != nil {
		return err
	}
	activePath := filepath.Join(projectDir, filepath.FromSlash(policy.Session.ActiveFile))
	return appendExpansionEvent(activePath, filepath.ToSlash(strings.TrimSpace(path)))
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
	if result.NotesChanged {
		result.MemorySatisfiedBy = "history"
	}

	committedNotes, err := committedDurableNotes(ctx, active.ProjectDir, active.GitBaseline, currentGit, policy.Project.Memory.AcceptedNoteGlobs)
	if err != nil {
		return nil, err
	}
	if result.MemorySatisfiedBy == "" && len(committedNotes) != 0 {
		result.MemorySatisfiedBy = "git_committed_notes"
	}

	if result.RepoChanged && policy.Closeout.RequireMemoryUpdateOnRepoChange && result.MemorySatisfiedBy == "" {
		result.OK = false
		result.Obligations = append(result.Obligations, "durable note update required for repo changes")
		result.Remediation = append(result.Remediation, "run `brain distill --session` to generate a session-scoped memory proposal")
		result.Remediation = append(result.Remediation, fmt.Sprintf("review the proposal, apply the durable note updates for %s, then retry `brain session finish`", policy.Project.Name))
	} else if result.MemorySatisfiedBy == "git_committed_notes" {
		result.Remediation = append(result.Remediation, "durable notes were already committed in the session commit range")
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

func loadActiveSessionIfExists(path string) (*ActiveSession, error) {
	active, err := loadActiveSession(path)
	if err == nil {
		return active, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return nil, err
}

func saveActiveSession(path string, active *ActiveSession) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := jsonMarshal(active)
	if err != nil {
		return err
	}
	return writeFileAtomically(path, raw, 0o644)
}

func removeActiveSession(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func loadActiveSessionForRecording(path string) (*ActiveSession, error) {
	var active *ActiveSession
	if err := withSessionLock(path, func() error {
		current, err := loadActiveSession(path)
		if err != nil {
			return err
		}
		active = current
		return nil
	}); err != nil {
		return nil, err
	}
	return active, nil
}

func appendCommandRun(path, sessionID string, record CommandRun) error {
	return withSessionLock(path, func() error {
		active, err := loadActiveSession(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("session %s ended before command %q could be recorded", sessionID, record.Command)
			}
			return err
		}
		if active.ID != sessionID || active.Status != "active" {
			return fmt.Errorf("session %s is no longer active; command %q was not recorded", sessionID, record.Command)
		}
		active.CommandRuns = append(active.CommandRuns, record)
		appendTelemetryEvent(active, PacketTelemetryEvent{
			Type:       PacketTelemetryEventVerification,
			Timestamp:  record.EndedAt.UTC(),
			SessionID:  active.ID,
			PacketHash: latestPacketHashBefore(active, record.EndedAt),
			Command:    record.Command,
			Success:    boolPtr(record.ExitCode == 0),
		})
		return saveActiveSession(path, active)
	})
}

func appendPacketRecord(path, sessionID string, record PacketRecord) error {
	return withSessionLock(path, func() error {
		active, err := loadActiveSession(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("session %s ended before packet %q could be recorded", sessionID, record.PacketHash)
			}
			return err
		}
		if active.ID != sessionID || active.Status != "active" {
			return fmt.Errorf("session %s is no longer active; packet %q was not recorded", sessionID, record.PacketHash)
		}
		active.PacketRecords = append(active.PacketRecords, record)
		trimPacketRecords(active)
		appendTelemetryEvent(active, PacketTelemetryEvent{
			Type:       PacketTelemetryEventCompiled,
			Timestamp:  record.CompiledAt.UTC(),
			SessionID:  active.ID,
			PacketHash: record.PacketHash,
			Metadata: map[string]any{
				"included_item_ids": append([]string(nil), record.IncludedItemIDs...),
			},
		})
		return saveActiveSession(path, active)
	})
}

func appendExpansionEvent(path, anchorPath string) error {
	if strings.TrimSpace(anchorPath) == "" {
		return nil
	}
	return withSessionLock(path, func() error {
		active, err := loadActiveSession(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if active.Status != "active" {
			return nil
		}
		packetHash, itemID := latestPacketExpansionMatch(active, anchorPath)
		if packetHash == "" || itemID == "" {
			return nil
		}
		appendTelemetryEvent(active, PacketTelemetryEvent{
			Type:       PacketTelemetryEventExpanded,
			Timestamp:  time.Now().UTC(),
			SessionID:  active.ID,
			PacketHash: packetHash,
			ItemID:     itemID,
			AnchorPath: anchorPath,
		})
		return saveActiveSession(path, active)
	})
}

func packetRecordFromCompiledPacket(packet *projectcontext.CompiledPacket) PacketRecord {
	record := PacketRecord{
		PacketHash:       packet.Hash(),
		TaskText:         packet.Task.Text,
		TaskSummary:      packet.Task.Summary,
		TaskSource:       packet.Task.Source,
		CompiledAt:       time.Now().UTC(),
		IncludedItemIDs:  []string{},
		IncludedAnchors:  []projectcontext.ContextAnchor{},
		InclusionReasons: []PacketInclusionReason{},
	}

	appendItem := func(section string, item projectcontext.CompiledItem) {
		record.IncludedItemIDs = append(record.IncludedItemIDs, item.ID)
		record.IncludedAnchors = append(record.IncludedAnchors, item.Anchor)
		record.InclusionReasons = append(record.InclusionReasons, PacketInclusionReason{
			ItemID:  item.ID,
			Section: section,
			Reason:  item.Reason,
		})
	}

	for _, item := range packet.BaseContract {
		appendItem("base_contract", item)
	}
	for _, item := range packet.WorkingSet.Notes {
		appendItem("working_set.notes", item)
	}
	for _, boundary := range packet.WorkingSet.Boundaries {
		record.IncludedItemIDs = append(record.IncludedItemIDs, "boundary:"+boundary.Path)
		record.IncludedAnchors = append(record.IncludedAnchors, projectcontext.ContextAnchor{
			Path:    boundary.Path,
			Section: boundary.Label,
		})
		record.InclusionReasons = append(record.InclusionReasons, PacketInclusionReason{
			ItemID:  "boundary:" + boundary.Path,
			Section: "working_set.boundaries",
			Reason:  boundary.Reason,
		})
	}
	for _, file := range packet.WorkingSet.Files {
		record.IncludedItemIDs = append(record.IncludedItemIDs, "file:"+file.Path)
		record.IncludedAnchors = append(record.IncludedAnchors, projectcontext.ContextAnchor{Path: file.Path})
		record.InclusionReasons = append(record.InclusionReasons, PacketInclusionReason{
			ItemID:  "file:" + file.Path,
			Section: "working_set.files",
			Reason:  file.Reason,
		})
	}
	for _, test := range packet.WorkingSet.Tests {
		record.IncludedItemIDs = append(record.IncludedItemIDs, "test:"+test.Path)
		record.IncludedAnchors = append(record.IncludedAnchors, projectcontext.ContextAnchor{Path: test.Path})
		record.InclusionReasons = append(record.InclusionReasons, PacketInclusionReason{
			ItemID:  "test:" + test.Path,
			Section: "working_set.tests",
			Reason:  test.Reason,
		})
	}
	for _, verification := range packet.Verification {
		record.IncludedItemIDs = append(record.IncludedItemIDs, verification.ID)
		record.IncludedAnchors = append(record.IncludedAnchors, projectcontext.ContextAnchor{
			Path:    verification.Source,
			Section: verification.Label,
		})
		record.InclusionReasons = append(record.InclusionReasons, PacketInclusionReason{
			ItemID:  verification.ID,
			Section: "verification",
			Reason:  verification.Reason,
		})
	}
	return record
}

func appendDurableUpdateTelemetry(active *ActiveSession, entries []history.Entry, acceptedOps, acceptedGlobs []string) {
	if active == nil {
		return
	}
	qualifying, _ := filterHistoryEntries(entries, acceptedOps, acceptedGlobs)
	for _, entry := range qualifying {
		appendTelemetryEvent(active, PacketTelemetryEvent{
			Type:       PacketTelemetryEventDurableUpdate,
			Timestamp:  entry.Timestamp.UTC(),
			SessionID:  active.ID,
			PacketHash: latestPacketHashBefore(active, entry.Timestamp),
			File:       filepath.ToSlash(entry.File),
			Operation:  entry.Operation,
		})
	}
}

func appendSessionClosedTelemetry(active *ActiveSession, ts time.Time, status string, validation *ValidationResult) {
	if active == nil {
		return
	}
	metadata := map[string]any{}
	if validation != nil {
		metadata["ok"] = validation.OK
		if len(validation.MissingCommands) != 0 {
			metadata["missing_commands"] = append([]string(nil), validation.MissingCommands...)
		}
		if validation.MemorySatisfiedBy != "" {
			metadata["memory_satisfied_by"] = validation.MemorySatisfiedBy
		}
	}
	appendTelemetryEvent(active, PacketTelemetryEvent{
		Type:        PacketTelemetryEventSessionClosed,
		Timestamp:   ts.UTC(),
		SessionID:   active.ID,
		PacketHash:  latestPacketHashBefore(active, ts),
		CloseStatus: status,
		Success:     boolPtr(validation == nil || validation.OK),
		Metadata:    metadata,
	})
}

func appendTelemetryEvent(active *ActiveSession, event PacketTelemetryEvent) {
	if active == nil {
		return
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if strings.TrimSpace(event.SessionID) == "" {
		event.SessionID = active.ID
	}
	active.TelemetryVersion = packetTelemetryVersion
	active.TelemetryEvents = append(active.TelemetryEvents, event)
	trimTelemetryEvents(active)
}

func trimPacketRecords(active *ActiveSession) {
	if active == nil || len(active.PacketRecords) <= maxSessionPacketRecords {
		return
	}
	active.PacketRecords = append([]PacketRecord(nil), active.PacketRecords[len(active.PacketRecords)-maxSessionPacketRecords:]...)
}

func trimTelemetryEvents(active *ActiveSession) {
	if active == nil || len(active.TelemetryEvents) <= maxSessionTelemetryEvents {
		return
	}
	active.TelemetryEvents = append([]PacketTelemetryEvent(nil), active.TelemetryEvents[len(active.TelemetryEvents)-maxSessionTelemetryEvents:]...)
}

func latestPacketHashBefore(active *ActiveSession, ts time.Time) string {
	if active == nil {
		return ""
	}
	var match string
	for _, record := range active.PacketRecords {
		if record.CompiledAt.After(ts) {
			continue
		}
		match = record.PacketHash
	}
	if match != "" {
		return match
	}
	if len(active.PacketRecords) == 0 {
		return ""
	}
	return active.PacketRecords[len(active.PacketRecords)-1].PacketHash
}

func latestPacketExpansionMatch(active *ActiveSession, anchorPath string) (string, string) {
	if active == nil {
		return "", ""
	}
	anchorPath = filepath.ToSlash(strings.TrimSpace(anchorPath))
	for i := len(active.PacketRecords) - 1; i >= 0; i-- {
		record := active.PacketRecords[i]
		for j, anchor := range record.IncludedAnchors {
			if filepath.ToSlash(strings.TrimSpace(anchor.Path)) != anchorPath {
				continue
			}
			if j < len(record.IncludedItemIDs) {
				return record.PacketHash, record.IncludedItemIDs[j]
			}
			return record.PacketHash, ""
		}
	}
	return "", ""
}

func boolPtr(v bool) *bool {
	return &v
}

func withSessionLock(activePath string, fn func() error) error {
	lockPath := activePath + ".lock"
	deadline := time.Now().Add(sessionLockTimeout)
	for {
		if err := os.Mkdir(lockPath, 0o700); err == nil {
			defer func() {
				_ = os.Remove(lockPath)
			}()
			return fn()
		} else if !sessionLockBusy(lockPath, err) {
			return fmt.Errorf("acquire session lock: %w", err)
		}

		if stale, staleErr := sessionLockIsStale(lockPath); staleErr == nil && stale {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("session state is busy at %s; retry", filepath.ToSlash(activePath))
		}
		time.Sleep(sessionLockRetryDelay)
	}
}

func sessionLockBusy(lockPath string, err error) bool {
	if errors.Is(err, os.ErrExist) {
		return true
	}
	// On Windows, racing Mkdir calls against an existing directory lock can
	// surface as a permission error instead of EEXIST. Treat permission
	// errors there as retryable contention; the lock dir can appear and
	// disappear between the failed Mkdir and a follow-up Stat.
	if !errors.Is(err, os.ErrPermission) {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	info, statErr := os.Stat(lockPath)
	return statErr == nil && info.IsDir()
}

func sessionLockIsStale(lockPath string) (bool, error) {
	info, err := os.Stat(lockPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return time.Since(info.ModTime()) > sessionLockStaleAfter, nil
}

func writeFileAtomically(path string, raw []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, ".brain-session-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()
	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return err
	}
	if _, err := tempFile.Write(raw); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return nil
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
	status = filterVolatileGitStatusLines(status)
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

func worktreeClean(snapshot GitSnapshot) bool {
	if !snapshot.Available {
		return false
	}
	return len(snapshot.Status) == 0
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

func filterVolatileGitStatusLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if isVolatileGitStatusLine(line) {
			continue
		}
		out = append(out, line)
	}
	return out
}

func isVolatileGitStatusLine(line string) bool {
	path := gitStatusPath(line)
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

func gitStatusPath(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if strings.Contains(line, " -> ") {
		parts := strings.Split(line, " -> ")
		return filepath.ToSlash(strings.TrimSpace(parts[len(parts)-1]))
	}
	if idx := strings.IndexByte(line, ' '); idx >= 0 {
		return filepath.ToSlash(strings.TrimSpace(line[idx+1:]))
	}
	return filepath.ToSlash(line)
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

func committedDurableNotes(ctx context.Context, projectDir string, base, current GitSnapshot, globs []string) ([]string, error) {
	if !base.Available || !current.Available {
		return nil, nil
	}
	if base.Head == "" || current.Head == "" || base.Head == current.Head {
		return nil, nil
	}
	if !worktreeClean(current) {
		return nil, nil
	}
	paths, err := gitChangedPathsBetween(ctx, projectDir, base.Head, current.Head)
	if err != nil {
		return nil, err
	}
	var matched []string
	for _, path := range paths {
		if pathMatchesAny(path, globs) {
			matched = append(matched, path)
		}
	}
	return matched, nil
}

func gitChangedPathsBetween(ctx context.Context, dir, baseHead, currentHead string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", baseHead, currentHead)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff changed paths: %w", err)
	}
	lines := splitNonEmpty(string(out))
	sort.Strings(lines)
	return lines, nil
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
