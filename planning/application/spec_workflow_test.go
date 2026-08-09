package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JimmyMcBride/brain/planning"
)

func TestSpecExecutionIsConfirmedAuditedAndIdempotent(t *testing.T) {
	repository := specWorkflowRepository()
	service := New(repository, Options{ModuleID: "planning", Now: func() time.Time { return time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC) }})
	events := &eventRecorder{}
	input := SpecExecutionInput{ID: "execution-ready", Confirmed: true}
	result, err := service.BeginSpecExecution(context.Background(), input, allowAuthorizer{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != MutationUpdate || result.Document.Artifact.Status != planning.SpecImplementing || len(result.Execution.Slices) != 2 || len(events.events) != 1 {
		t.Fatalf("unexpected execution result: %#v events=%d", result, len(events.events))
	}
	rerun, err := service.BeginSpecExecution(context.Background(), input, allowAuthorizer{}, events)
	if err != nil || rerun.Action != MutationUnchanged || rerun.Event != nil || len(events.events) != 1 {
		t.Fatalf("unexpected execution rerun: %#v err=%v events=%d", rerun, err, len(events.events))
	}
}

func TestSpecHandoffRollsBackSpecWhenSessionPersistenceFails(t *testing.T) {
	repository := specWorkflowRepository()
	repository.guidedWriteErr = errors.New("session write failed")
	service := New(repository, Options{ModuleID: "planning"})
	_, err := service.HandoffSpec(context.Background(), SpecExecutionInput{ID: "execution-ready", Confirmed: true}, allowAuthorizer{}, &eventRecorder{})
	if err == nil || !strings.Contains(err.Error(), "session write failed") {
		t.Fatalf("expected session failure, got %v", err)
	}
	document, getErr := repository.GetSpec(context.Background(), "execution-ready")
	if getErr != nil {
		t.Fatal(getErr)
	}
	if document.Artifact.Status != planning.SpecApproved {
		t.Fatalf("partial execution status remained after rollback: %#v", document.Artifact)
	}
}

func TestSpecHandoffUpdatesBothArtifactsAndRerunsUnchanged(t *testing.T) {
	repository := specWorkflowRepository()
	service := New(repository, Options{ModuleID: "planning"})
	events := &eventRecorder{}
	input := SpecExecutionInput{ID: "execution-ready", Confirmed: true}
	result, err := service.HandoffSpec(context.Background(), input, allowAuthorizer{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != MutationUpdate || result.Session == nil || result.Session.CurrentStage != "execution" || result.Document.Artifact.Status != planning.SpecImplementing || len(events.events) != 1 {
		t.Fatalf("unexpected handoff: %#v events=%d", result, len(events.events))
	}
	rerun, err := service.HandoffSpec(context.Background(), input, allowAuthorizer{}, events)
	if err != nil || rerun.Action != MutationUnchanged || rerun.Event != nil || len(events.events) != 1 {
		t.Fatalf("unexpected handoff rerun: %#v err=%v events=%d", rerun, err, len(events.events))
	}
}

func specWorkflowRepository() *fakeRepository {
	body := "# Execution Ready\n\n## Verification\n\n- go test ./...\n\n## Execution Plan\n\n- Capture contracts\n  - description: Freeze behavior.\n- Migrate workflow\n  - description: Share behavior.\n"
	document := SpecDocument{Artifact: planning.Spec{ID: "execution-ready", Title: "Execution Ready", Status: planning.SpecApproved, Approval: planning.Approval{State: planning.ApprovalApproved}, Verification: []string{"go test ./..."}}, Path: ".plan/specs/execution-ready.md", Body: body, Metadata: map[string]any{"slug": "execution-ready", "title": "Execution Ready", "type": "spec", "status": "approved"}}
	return &fakeRepository{status: WorkspaceStatus{State: WorkspaceCompatible, Writable: true}, specs: []SpecDocument{document}, guided: GuidedSessionState{SchemaVersion: 3, Sessions: map[string]GuidedSessionRecord{"brainstorm/execution-ready": {ChainID: "brainstorm/execution-ready", Brainstorm: "execution-ready", Spec: "execution-ready", CurrentStage: "spec", StageStatuses: map[string]string{"spec": "in_progress"}}}}}
}
