package projectcontext

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"brain/internal/buildinfo"
)

const (
	projectMigrationSchemaVersion = 1
	projectMigrationStateFile     = "project-migrations.json"
)

const (
	projectMigrationStateReady   = "ready"
	projectMigrationStateMissing = "missing"
	projectMigrationStateInvalid = "invalid"
)

const (
	projectMigrationPlanNotBrainProject = "not_brain_project"
	projectMigrationPlanPending         = "pending"
	projectMigrationPlanCurrent         = "current"
)

type ProjectMigrationDefinition struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type AppliedProjectMigration struct {
	ID           string `json:"id"`
	AppliedAt    string `json:"applied_at"`
	BrainVersion string `json:"brain_version,omitempty"`
	BrainCommit  string `json:"brain_commit,omitempty"`
}

type ProjectMigrationRun struct {
	RanAt        string   `json:"ran_at"`
	Status       string   `json:"status"`
	BrainVersion string   `json:"brain_version,omitempty"`
	BrainCommit  string   `json:"brain_commit,omitempty"`
	PlannedIDs   []string `json:"planned_ids,omitempty"`
	AppliedIDs   []string `json:"applied_ids,omitempty"`
	Error        string   `json:"error,omitempty"`
}

type ProjectMigrationState struct {
	SchemaVersion int                       `json:"schema_version"`
	Applied       []AppliedProjectMigration `json:"applied,omitempty"`
	LastRun       *ProjectMigrationRun      `json:"last_run,omitempty"`
}

type ProjectMigrationPlan struct {
	ProjectDir        string                       `json:"project_dir"`
	LedgerPath        string                       `json:"ledger_path"`
	UsesBrain         bool                         `json:"uses_brain"`
	Status            string                       `json:"status"`
	StateStatus       string                       `json:"state_status,omitempty"`
	NeedsStateWrite   bool                         `json:"needs_state_write,omitempty"`
	KnownMigrations   []ProjectMigrationDefinition `json:"known_migrations"`
	PendingMigrations []ProjectMigrationDefinition `json:"pending_migrations,omitempty"`
	AppliedMigrations []AppliedProjectMigration    `json:"applied_migrations,omitempty"`
}

type ProjectMigrationResult struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Action      string   `json:"action"`
	Results     []Result `json:"results,omitempty"`
}

type ApplyProjectMigrationsResult struct {
	ProjectDir          string                   `json:"project_dir"`
	LedgerPath          string                   `json:"ledger_path"`
	UsesBrain           bool                     `json:"uses_brain"`
	Status              string                   `json:"status"`
	AppliedMigrationIDs []string                 `json:"applied_migration_ids,omitempty"`
	Migrations          []ProjectMigrationResult `json:"migrations,omitempty"`
}

var knownProjectMigrationDefinitions = []ProjectMigrationDefinition{
	{
		ID:          "refresh-brain-managed-context-v1",
		Description: "Refresh Brain-managed generated context surfaces for the compiler-era workflow",
	},
	{
		ID:          "refresh-existing-agent-integrations-v1",
		Description: "Refresh or migrate existing Brain-managed agent integration blocks in local agent files",
	},
}

func KnownProjectMigrations() []ProjectMigrationDefinition {
	out := make([]ProjectMigrationDefinition, len(knownProjectMigrationDefinitions))
	copy(out, knownProjectMigrationDefinitions)
	return out
}

func (m *Manager) ApplyProjectMigrations(ctx context.Context, projectDir string) (*ApplyProjectMigrationsResult, error) {
	plan, err := m.PlanProjectMigrations(projectDir)
	if err != nil {
		return nil, err
	}
	result := &ApplyProjectMigrationsResult{
		ProjectDir: plan.ProjectDir,
		LedgerPath: plan.LedgerPath,
		UsesBrain:  plan.UsesBrain,
		Status:     "not_brain_project",
	}
	if !plan.UsesBrain {
		return result, nil
	}
	if len(plan.PendingMigrations) == 0 {
		result.Status = "unchanged"
		return result, nil
	}

	state, _, err := loadProjectMigrationState(plan.ProjectDir)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	plannedIDs := make([]string, 0, len(plan.PendingMigrations))
	appliedIDs := make([]string, 0, len(plan.PendingMigrations))
	appliedSet := make(map[string]struct{}, len(state.Applied))
	for _, applied := range state.Applied {
		if id := strings.TrimSpace(applied.ID); id != "" {
			appliedSet[id] = struct{}{}
		}
	}

	for _, migration := range plan.PendingMigrations {
		plannedIDs = append(plannedIDs, migration.ID)
		migrationResult, err := m.applyProjectMigration(ctx, plan.ProjectDir, migration)
		if err != nil {
			state.LastRun = NewProjectMigrationRun("failed", plannedIDs, appliedIDs, now, err)
			if saveErr := m.SaveProjectMigrationState(plan.ProjectDir, state); saveErr != nil {
				return nil, fmt.Errorf("%w; additionally failed to write migration state: %v", err, saveErr)
			}
			return nil, err
		}
		result.Migrations = append(result.Migrations, migrationResult)
		if _, ok := appliedSet[migration.ID]; !ok {
			state.Applied = append(state.Applied, NewAppliedProjectMigration(migration.ID, now))
			appliedSet[migration.ID] = struct{}{}
		}
		appliedIDs = append(appliedIDs, migration.ID)
	}

	state.LastRun = NewProjectMigrationRun("applied", plannedIDs, appliedIDs, now, nil)
	if err := m.SaveProjectMigrationState(plan.ProjectDir, state); err != nil {
		return nil, err
	}

	result.Status = "applied"
	result.AppliedMigrationIDs = appliedIDs
	return result, nil
}

func (m *Manager) PlanProjectMigrations(projectDir string) (*ProjectMigrationPlan, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(projectDir); err != nil {
		return nil, fmt.Errorf("project dir %s: %w", projectDir, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("project dir is not a directory: %s", projectDir)
	}

	plan := &ProjectMigrationPlan{
		ProjectDir:        projectDir,
		LedgerPath:        projectMigrationLedgerPath(projectDir),
		Status:            projectMigrationPlanNotBrainProject,
		KnownMigrations:   KnownProjectMigrations(),
		PendingMigrations: nil,
	}
	if !usesBrainWorkspace(projectDir) {
		return plan, nil
	}

	plan.UsesBrain = true
	state, stateStatus, err := loadProjectMigrationState(projectDir)
	if err != nil {
		return nil, err
	}
	plan.StateStatus = stateStatus
	plan.AppliedMigrations = append([]AppliedProjectMigration(nil), state.Applied...)
	plan.NeedsStateWrite = stateStatus == projectMigrationStateMissing || stateStatus == projectMigrationStateInvalid

	applied := make(map[string]struct{}, len(state.Applied))
	for _, migration := range state.Applied {
		id := strings.TrimSpace(migration.ID)
		if id == "" {
			continue
		}
		applied[id] = struct{}{}
	}
	for _, migration := range plan.KnownMigrations {
		if _, ok := applied[migration.ID]; ok {
			continue
		}
		plan.PendingMigrations = append(plan.PendingMigrations, migration)
	}

	plan.Status = projectMigrationPlanCurrent
	if len(plan.PendingMigrations) != 0 {
		plan.Status = projectMigrationPlanPending
	}
	return plan, nil
}

func (m *Manager) SaveProjectMigrationState(projectDir string, state ProjectMigrationState) error {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return err
	}
	if !usesBrainWorkspace(projectDir) {
		return fmt.Errorf("project %s does not use Brain", projectDir)
	}
	state = normalizeProjectMigrationState(state)
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal project migration state: %w", err)
	}
	path := projectMigrationLedgerPath(projectDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create project migration state dir: %w", err)
	}
	if err := writeProjectMigrationFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write project migration state: %w", err)
	}
	return nil
}

func NewAppliedProjectMigration(id string, appliedAt time.Time) AppliedProjectMigration {
	info := buildinfo.Current()
	return AppliedProjectMigration{
		ID:           strings.TrimSpace(id),
		AppliedAt:    appliedAt.UTC().Format(time.RFC3339),
		BrainVersion: info.Version,
		BrainCommit:  info.Commit,
	}
}

func NewProjectMigrationRun(status string, plannedIDs, appliedIDs []string, runAt time.Time, err error) *ProjectMigrationRun {
	info := buildinfo.Current()
	run := &ProjectMigrationRun{
		RanAt:        runAt.UTC().Format(time.RFC3339),
		Status:       strings.TrimSpace(status),
		BrainVersion: info.Version,
		BrainCommit:  info.Commit,
		PlannedIDs:   compactStrings(plannedIDs),
		AppliedIDs:   compactStrings(appliedIDs),
	}
	if err != nil {
		run.Error = err.Error()
	}
	return run
}

func (m *Manager) applyProjectMigration(ctx context.Context, projectDir string, migration ProjectMigrationDefinition) (ProjectMigrationResult, error) {
	result := ProjectMigrationResult{
		ID:          migration.ID,
		Description: migration.Description,
		Action:      "unchanged",
	}
	var (
		results []Result
		err     error
	)
	switch migration.ID {
	case "refresh-brain-managed-context-v1":
		results, err = m.syncManagedContext(ctx, projectDir, false, false, false)
	case "refresh-existing-agent-integrations-v1":
		results, err = m.syncAgentIntegrations(projectDir, nil, false, false)
	default:
		return result, fmt.Errorf("unknown project migration %q", migration.ID)
	}
	if err != nil {
		return result, fmt.Errorf("apply project migration %s: %w", migration.ID, err)
	}
	result.Results = results
	if migrationChanged(results) {
		result.Action = "updated"
	}
	return result, nil
}

func projectMigrationLedgerPath(projectDir string) string {
	return filepath.Join(projectDir, ".brain", "state", projectMigrationStateFile)
}

func usesBrainWorkspace(projectDir string) bool {
	for _, rel := range []string{
		".brain/policy.yaml",
		".brain/project.yaml",
		".brain/state/brain.sqlite3",
	} {
		if _, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(rel))); err == nil {
			return true
		}
	}
	return false
}

func loadProjectMigrationState(projectDir string) (ProjectMigrationState, string, error) {
	path := projectMigrationLedgerPath(projectDir)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultProjectMigrationState(), projectMigrationStateMissing, nil
		}
		return ProjectMigrationState{}, "", fmt.Errorf("read project migration state: %w", err)
	}

	var state ProjectMigrationState
	if err := json.Unmarshal(raw, &state); err != nil {
		return defaultProjectMigrationState(), projectMigrationStateInvalid, nil
	}
	if state.SchemaVersion != 0 && state.SchemaVersion != projectMigrationSchemaVersion {
		return defaultProjectMigrationState(), projectMigrationStateInvalid, nil
	}
	return normalizeProjectMigrationState(state), projectMigrationStateReady, nil
}

func defaultProjectMigrationState() ProjectMigrationState {
	return ProjectMigrationState{
		SchemaVersion: projectMigrationSchemaVersion,
		Applied:       []AppliedProjectMigration{},
	}
}

func normalizeProjectMigrationState(state ProjectMigrationState) ProjectMigrationState {
	if state.SchemaVersion == 0 {
		state.SchemaVersion = projectMigrationSchemaVersion
	}
	if state.Applied == nil {
		state.Applied = []AppliedProjectMigration{}
	}
	if state.LastRun != nil {
		state.LastRun.PlannedIDs = compactStrings(state.LastRun.PlannedIDs)
		state.LastRun.AppliedIDs = compactStrings(state.LastRun.AppliedIDs)
	}
	return state
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func migrationChanged(results []Result) bool {
	for _, result := range results {
		if result.Action != "unchanged" {
			return true
		}
	}
	return false
}

func writeProjectMigrationFile(path string, raw []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".brain-project-migrations-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
