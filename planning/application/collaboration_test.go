package application

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type collaborationMemory struct {
	snapshot      CollaborationSourceSnapshot
	reads, writes int
	conflict      bool
}

func (m *collaborationMemory) Read(context.Context, ExternalReference) (CollaborationSourceSnapshot, error) {
	m.reads++
	return m.snapshot, nil
}
func (m *collaborationMemory) Repair(_ context.Context, r CollaborationRepairRequest) (CollaborationRepairEvidence, error) {
	if m.conflict || r.ExpectedRevision != m.snapshot.Source.Revision {
		return CollaborationRepairEvidence{}, &IntegrationError{Class: IntegrationRevisionConflict}
	}
	changed := m.snapshot.Content != r.Content
	if changed {
		m.writes++
		m.snapshot.Content = r.Content
		m.snapshot.Source.Revision = "next"
	}
	return CollaborationRepairEvidence{Source: m.snapshot.Source, Changed: changed}, nil
}

type collaborationPolicy struct {
	deny  string
	calls []string
}

func (p *collaborationPolicy) Require(_ context.Context, permission string) error {
	p.calls = append(p.calls, permission)
	if permission == p.deny {
		return errors.New("denied")
	}
	return nil
}

type collaborationEvents struct {
	calls int
	fail  bool
}

func (e *collaborationEvents) Publish(context.Context, Event) error {
	e.calls++
	if e.fail {
		return errors.New("audit failed")
	}
	return nil
}

const matureCollaboration = "## Problem\nBound planning work.\n\n## Goals\nClear specs.\n\n## Constraints\nRetain source.\n\n## Non-Goals\nNo cloud.\n\n## Proposed Shape\nShared services.\n\n## Spec Split\n- Read source\n- Repair source\n\nRepair source depends on Read source.\n"

func newCollaborationMemory() *collaborationMemory {
	return &collaborationMemory{snapshot: CollaborationSourceSnapshot{SchemaVersion: 1, Source: ExternalReference{Provider: "test", Kind: "conversation", OpaqueID: "source", Revision: "first"}, Title: "Planning", Content: matureCollaboration}}
}

func TestCollaborationAssessmentAndRepairLifecycle(t *testing.T) {
	ctx := context.Background()
	source := newCollaborationMemory()
	service := New(nil, Options{})
	policy := &collaborationPolicy{}
	events := &collaborationEvents{}
	assessment, err := service.AssessCollaboration(ctx, source, source.snapshot.Source, policy)
	if err != nil || assessment.Decision.State != "ready_multi_spec" || len(assessment.Decision.DependencyGuess) != 2 {
		t.Fatalf("%+v %v", assessment, err)
	}
	input := CollaborationRepairInput{Source: source.snapshot.Source, Specs: []string{" New A ", "New B", "new a"}}
	preview, err := service.PreviewCollaborationRepair(ctx, source, input, policy)
	if err != nil || source.writes != 0 || len(preview.Specs) != 2 || !strings.Contains(preview.Request.Content, "## Specs") {
		t.Fatalf("%+v %v", preview, err)
	}
	input.ExpectedRevision = preview.Request.ExpectedRevision
	if _, err = service.RepairCollaboration(ctx, source, input, policy, events); !errors.Is(err, ErrConfirmationRequired) {
		t.Fatal(err)
	}
	input.Confirmed = true
	policy.deny = PermissionCollaboration
	if _, err = service.RepairCollaboration(ctx, source, input, policy, events); err == nil || source.writes != 0 {
		t.Fatal("permission bypass")
	}
	policy.deny = ""
	result, err := service.RepairCollaboration(ctx, source, input, policy, events)
	if err != nil || !result.Applied || source.writes != 1 || events.calls != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	assessment, err = service.AssessCollaboration(ctx, source, source.snapshot.Source, policy)
	if err != nil || assessment.Decision.SuggestedTitles["specs"].([]string)[0] != "New A" {
		t.Fatalf("repair ignored: %+v %v", assessment, err)
	}
	input.ExpectedRevision = source.snapshot.Source.Revision
	result, err = service.RepairCollaboration(ctx, source, input, policy, events)
	if err != nil || result.Action != MutationUnchanged || source.writes != 1 || events.calls != 1 {
		t.Fatalf("rerun %+v %v", result, err)
	}
}

func TestCollaborationConflictAndAuditFailure(t *testing.T) {
	for _, auditFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "conflict", true: "audit"}[auditFailure], func(t *testing.T) {
			source := newCollaborationMemory()
			source.conflict = !auditFailure
			events := &collaborationEvents{fail: auditFailure}
			result, err := New(nil, Options{}).RepairCollaboration(context.Background(), source, CollaborationRepairInput{Source: source.snapshot.Source, Specs: []string{"A", "B"}, Confirmed: true, ExpectedRevision: "first"}, &collaborationPolicy{}, events)
			var integration *IntegrationError
			if !errors.As(err, &integration) {
				t.Fatal(err)
			}
			if auditFailure {
				if !result.Applied || result.Evidence == nil || integration.Class != IntegrationPartialFailure {
					t.Fatalf("lost evidence: %+v %v", result, err)
				}
			} else if source.writes != 0 || events.calls != 0 || integration.Class != IntegrationRevisionConflict {
				t.Fatal(err)
			}
		})
	}
}

func TestCollaborationReadDenialAndContributions(t *testing.T) {
	source := newCollaborationMemory()
	service := New(nil, Options{})
	if _, err := service.AssessCollaboration(context.Background(), source, source.snapshot.Source, &collaborationPolicy{deny: PermissionRead}); err == nil || source.reads != 0 {
		t.Fatal("read denial performed provider work")
	}
	source.snapshot.Content = "## Problem\nBound work."
	if decision := assessCollaboration(source.snapshot); decision.State != "not_ready" {
		t.Fatal(decision)
	}
	source.snapshot.Contributions = []CollaborationContribution{{Content: matureCollaboration}}
	if decision := assessCollaboration(source.snapshot); decision.State != "ready_multi_spec" {
		t.Fatal(decision)
	}
	source.snapshot.Content = strings.Split(matureCollaboration, "## Spec Split")[0] + "\nPlease split into multiple specs."
	source.snapshot.Contributions = nil
	if decision := assessCollaboration(source.snapshot); decision.State != "needs_source_repair" {
		t.Fatal(decision)
	}
}
