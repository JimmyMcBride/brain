package projectcontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlanProjectMigrationsSkipsNonBrainRepos(t *testing.T) {
	project := t.TempDir()
	manager := New(t.TempDir())

	plan, err := manager.PlanProjectMigrations(project)
	if err != nil {
		t.Fatal(err)
	}
	if plan.UsesBrain {
		t.Fatal("expected non-Brain repo to be skipped")
	}
	if plan.Status != projectMigrationPlanNotBrainProject {
		t.Fatalf("unexpected plan status: %s", plan.Status)
	}
	if _, err := os.Stat(plan.LedgerPath); !os.IsNotExist(err) {
		t.Fatalf("expected no ledger writes for non-Brain repo, err=%v", err)
	}
}

func TestPlanProjectMigrationsTreatsMissingLedgerAsRecoverable(t *testing.T) {
	project := newBrainProjectForMigrations(t)
	manager := New(t.TempDir())

	plan, err := manager.PlanProjectMigrations(project)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.UsesBrain {
		t.Fatal("expected Brain repo")
	}
	if plan.StateStatus != projectMigrationStateMissing {
		t.Fatalf("unexpected state status: %s", plan.StateStatus)
	}
	if !plan.NeedsStateWrite {
		t.Fatal("expected missing ledger to need rewrite")
	}
	if plan.Status != projectMigrationPlanPending {
		t.Fatalf("unexpected plan status: %s", plan.Status)
	}
	if len(plan.PendingMigrations) != len(KnownProjectMigrations()) {
		t.Fatalf("expected all known migrations to be pending, got=%d", len(plan.PendingMigrations))
	}
}

func TestPlanProjectMigrationsTreatsInvalidLedgerAsRecoverable(t *testing.T) {
	project := newBrainProjectForMigrations(t)
	ledgerPath := projectMigrationLedgerPath(project)
	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledgerPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}

	manager := New(t.TempDir())
	plan, err := manager.PlanProjectMigrations(project)
	if err != nil {
		t.Fatal(err)
	}
	if plan.StateStatus != projectMigrationStateInvalid {
		t.Fatalf("unexpected state status: %s", plan.StateStatus)
	}
	if !plan.NeedsStateWrite {
		t.Fatal("expected invalid ledger to need rewrite")
	}
	if plan.Status != projectMigrationPlanBroken {
		t.Fatalf("unexpected plan status: %s", plan.Status)
	}
	if !strings.Contains(plan.BrokenReason, "invalid project migration ledger") {
		t.Fatalf("unexpected broken reason: %s", plan.BrokenReason)
	}
	if len(plan.PendingMigrations) != len(KnownProjectMigrations()) {
		t.Fatalf("expected all migrations pending after invalid ledger, got=%d", len(plan.PendingMigrations))
	}
}

func TestPlanProjectMigrationsReportsBrokenAfterFailedRun(t *testing.T) {
	project := newBrainProjectForMigrations(t)
	manager := New(t.TempDir())
	state := ProjectMigrationState{
		LastRun: NewProjectMigrationRun(
			"failed",
			[]string{"refresh-brain-managed-context-v1"},
			nil,
			time.Unix(0, 0),
			context.DeadlineExceeded,
		),
	}
	if err := manager.SaveProjectMigrationState(project, state); err != nil {
		t.Fatal(err)
	}

	plan, err := manager.PlanProjectMigrations(project)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != projectMigrationPlanBroken {
		t.Fatalf("unexpected plan status: %s", plan.Status)
	}
	if !strings.Contains(plan.BrokenReason, "last migration run failed") {
		t.Fatalf("unexpected broken reason: %s", plan.BrokenReason)
	}
	if plan.LastRun == nil || plan.LastRun.Status != "failed" {
		t.Fatalf("expected failed last run to be preserved: %+v", plan.LastRun)
	}
}

func TestPlanProjectMigrationsUsesAppliedIDsInsteadOfVersionComparison(t *testing.T) {
	project := newBrainProjectForMigrations(t)
	manager := New(t.TempDir())
	state := ProjectMigrationState{
		Applied: []AppliedProjectMigration{
			{
				ID:           "refresh-brain-managed-context-v1",
				AppliedAt:    "2026-04-15T00:00:00Z",
				BrainVersion: "v9.9.9",
				BrainCommit:  "abc123",
			},
		},
		LastRun: NewProjectMigrationRun("applied", []string{"refresh-brain-managed-context-v1"}, []string{"refresh-brain-managed-context-v1"}, time.Unix(0, 0), nil),
	}
	if err := manager.SaveProjectMigrationState(project, state); err != nil {
		t.Fatal(err)
	}

	plan, err := manager.PlanProjectMigrations(project)
	if err != nil {
		t.Fatal(err)
	}
	if plan.StateStatus != projectMigrationStateReady {
		t.Fatalf("unexpected state status: %s", plan.StateStatus)
	}
	if len(plan.AppliedMigrations) != 1 {
		t.Fatalf("unexpected applied migrations: %+v", plan.AppliedMigrations)
	}
	if len(plan.PendingMigrations) != 1 {
		t.Fatalf("expected one pending migration, got=%d", len(plan.PendingMigrations))
	}
	if plan.PendingMigrations[0].ID != "refresh-existing-agent-integrations-v1" {
		t.Fatalf("unexpected pending migration: %+v", plan.PendingMigrations)
	}
}

func TestSaveProjectMigrationStateWritesLedger(t *testing.T) {
	project := newBrainProjectForMigrations(t)
	manager := New(t.TempDir())
	state := ProjectMigrationState{
		Applied: []AppliedProjectMigration{
			NewAppliedProjectMigration("refresh-brain-managed-context-v1", time.Date(2026, 4, 15, 5, 0, 0, 0, time.UTC)),
		},
		LastRun: NewProjectMigrationRun(
			"applied",
			[]string{"refresh-brain-managed-context-v1", "refresh-existing-agent-integrations-v1"},
			[]string{"refresh-brain-managed-context-v1"},
			time.Date(2026, 4, 15, 5, 0, 0, 0, time.UTC),
			nil,
		),
	}
	if err := manager.SaveProjectMigrationState(project, state); err != nil {
		t.Fatal(err)
	}

	plan, err := manager.PlanProjectMigrations(project)
	if err != nil {
		t.Fatal(err)
	}
	if plan.StateStatus != projectMigrationStateReady {
		t.Fatalf("unexpected state status: %s", plan.StateStatus)
	}
	if len(plan.AppliedMigrations) != 1 {
		t.Fatalf("unexpected applied migrations: %+v", plan.AppliedMigrations)
	}
	if plan.AppliedMigrations[0].ID != "refresh-brain-managed-context-v1" {
		t.Fatalf("unexpected applied migration: %+v", plan.AppliedMigrations[0])
	}
	if len(plan.PendingMigrations) != 1 {
		t.Fatalf("expected one pending migration, got=%d", len(plan.PendingMigrations))
	}
	if plan.PendingMigrations[0].ID != "refresh-existing-agent-integrations-v1" {
		t.Fatalf("unexpected pending migration: %+v", plan.PendingMigrations)
	}
}

func TestSaveProjectMigrationStateRejectsNonBrainRepos(t *testing.T) {
	manager := New(t.TempDir())
	project := t.TempDir()
	if err := manager.SaveProjectMigrationState(project, ProjectMigrationState{}); err == nil {
		t.Fatal("expected non-Brain repo write to fail")
	}
}

func TestApplyProjectMigrationsRefreshesManagedDocsAndLegacyAgentFiles(t *testing.T) {
	project := newInstalledBrainProject(t)
	manager := New(t.TempDir())

	staleAgents := "# Project Agent Contract\n\n<!-- brain:begin agents-contract -->\nstale\n<!-- brain:end agents-contract -->\n\n## Local Notes\n\nkeep me\n"
	if err := os.WriteFile(filepath.Join(project, "AGENTS.md"), []byte(staleAgents), 0o644); err != nil {
		t.Fatal(err)
	}
	legacyCodex := "# Codex Wrapper\n\n<!-- brain:begin agent-wrapper-codex -->\nstale wrapper\n<!-- brain:end agent-wrapper-codex -->\n\nCustom codex note.\n"
	if err := os.MkdirAll(filepath.Join(project, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".codex", "AGENTS.md"), []byte(legacyCodex), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := manager.ApplyProjectMigrations(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "applied" {
		t.Fatalf("unexpected result status: %s", result.Status)
	}
	if strings.Join(result.AppliedMigrationIDs, ",") != "refresh-brain-managed-context-v1,refresh-existing-agent-integrations-v1" {
		t.Fatalf("unexpected applied ids: %+v", result.AppliedMigrationIDs)
	}

	agentsBody, err := os.ReadFile(filepath.Join(project, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agentsBody), "brain context compile") || !strings.Contains(string(agentsBody), "keep me") {
		t.Fatalf("managed context was not refreshed correctly:\n%s", string(agentsBody))
	}

	codexBody, err := os.ReadFile(filepath.Join(project, ".codex", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(codexBody), "agent-integration-codex") {
		t.Fatalf("expected migrated agent integration block:\n%s", string(codexBody))
	}
	if strings.Contains(string(codexBody), "agent-wrapper-codex") {
		t.Fatalf("expected legacy wrapper block to be removed:\n%s", string(codexBody))
	}
	if !strings.Contains(string(codexBody), "Custom codex note.") {
		t.Fatalf("expected legacy wrapper user content to be preserved:\n%s", string(codexBody))
	}

	plan, err := manager.PlanProjectMigrations(project)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != projectMigrationPlanCurrent {
		t.Fatalf("expected project to be current after apply, got=%s", plan.Status)
	}
}

func TestApplyProjectMigrationsLeavesUnmanagedAgentFilesAloneAndBecomesNoop(t *testing.T) {
	project := newInstalledBrainProject(t)
	manager := New(t.TempDir())
	manualClaude := "# Team Claude Notes\n\nKeep this exactly.\n"
	if err := os.MkdirAll(filepath.Join(project, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	claudePath := filepath.Join(project, ".claude", "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte(manualClaude), 0o644); err != nil {
		t.Fatal(err)
	}

	first, err := manager.ApplyProjectMigrations(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != "applied" {
		t.Fatalf("unexpected first result status: %s", first.Status)
	}
	body, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != manualClaude {
		t.Fatalf("expected unmanaged agent file to stay unchanged:\n%s", string(body))
	}

	second, err := manager.ApplyProjectMigrations(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != "unchanged" {
		t.Fatalf("expected second apply to be a no-op, got=%s", second.Status)
	}
	if len(second.AppliedMigrationIDs) != 0 {
		t.Fatalf("expected no applied ids on noop rerun, got=%+v", second.AppliedMigrationIDs)
	}
}

func TestApplyProjectMigrationsSkipsUnmanagedBrainOwnedDocs(t *testing.T) {
	project := newInstalledBrainProject(t)
	manager := New(t.TempDir())
	manualAgents := "# Project Agent Contract\n\nCommitted durable note before publish.\n"
	if err := os.WriteFile(filepath.Join(project, "AGENTS.md"), []byte(manualAgents), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := manager.ApplyProjectMigrations(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "applied" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	body, err := os.ReadFile(filepath.Join(project, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != manualAgents {
		t.Fatalf("expected unmanaged AGENTS.md to stay untouched:\n%s", string(body))
	}
}

func TestApplyProjectMigrationsSkipsNonBrainRepos(t *testing.T) {
	manager := New(t.TempDir())
	result, err := manager.ApplyProjectMigrations(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "not_brain_project" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
}

func TestInstallInitializesProjectMigrationsAsCurrent(t *testing.T) {
	project := newBrainProjectForMigrations(t)
	manager := New(t.TempDir())

	if _, err := manager.Install(context.Background(), Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	plan, err := manager.PlanProjectMigrations(project)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != projectMigrationPlanCurrent {
		t.Fatalf("expected current plan after install, got=%s", plan.Status)
	}
	if len(plan.PendingMigrations) != 0 {
		t.Fatalf("expected no pending migrations after install, got=%d", len(plan.PendingMigrations))
	}
	if len(plan.AppliedMigrations) != len(KnownProjectMigrations()) {
		t.Fatalf("expected all known migrations applied after install, got=%d", len(plan.AppliedMigrations))
	}
}

func TestAdoptInitializesProjectMigrationsAsCurrent(t *testing.T) {
	project := newBrainProjectForMigrations(t)
	if err := os.WriteFile(filepath.Join(project, "AGENTS.md"), []byte("Manual contract\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager := New(t.TempDir())

	if _, err := manager.Adopt(context.Background(), Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	plan, err := manager.PlanProjectMigrations(project)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != projectMigrationPlanCurrent {
		t.Fatalf("expected current plan after adopt, got=%s", plan.Status)
	}
	if len(plan.PendingMigrations) != 0 {
		t.Fatalf("expected no pending migrations after adopt, got=%d", len(plan.PendingMigrations))
	}
}

func newBrainProjectForMigrations(t *testing.T) string {
	t.Helper()
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".brain", "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".brain", "policy.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return project
}

func newInstalledBrainProject(t *testing.T) string {
	t.Helper()
	project := newBrainProjectForMigrations(t)
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/demo\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager := New(t.TempDir())
	if _, err := manager.Install(context.Background(), Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(projectMigrationLedgerPath(project)); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return project
}
