package projectcontext

import (
	"os"
	"path/filepath"
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
	if plan.Status != projectMigrationPlanPending {
		t.Fatalf("unexpected plan status: %s", plan.Status)
	}
	if len(plan.PendingMigrations) != len(KnownProjectMigrations()) {
		t.Fatalf("expected all migrations pending after invalid ledger, got=%d", len(plan.PendingMigrations))
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
