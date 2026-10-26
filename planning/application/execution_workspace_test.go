package application

import (
	"context"
	"errors"
	"testing"

	"github.com/JimmyMcBride/brain/planning"
)

type executionMemory struct {
	snapshot                       ExecutionWorkspaceSnapshot
	reads, attaches, statuses      int
	attachErr, statusErr           error
	attachEvidence, statusEvidence ExecutionWorkItem
}

func (m *executionMemory) Inspect(context.Context, ExternalReference) (ExecutionWorkspaceSnapshot, error) {
	m.reads++
	return m.snapshot, nil
}

func (m *executionMemory) Attach(_ context.Context, request ExecutionAttachRequest) (ExecutionWorkItem, error) {
	if m.attachErr != nil {
		return m.attachEvidence, m.attachErr
	}
	m.attaches++
	item := ExecutionWorkItem{Artifact: request.Artifact, PublishedWork: request.PublishedWork, Reference: ExternalReference{Provider: "test", Kind: "project_item", OpaqueID: "item-1", DisplayID: "#1", URL: request.PublishedWork.URL, Revision: "item-1"}, Status: ExecutionTodo}
	m.snapshot.Items = append(m.snapshot.Items, item)
	m.snapshot.Workspace.Revision = "workspace-2"
	return item, nil
}

func (m *executionMemory) ApplyStatus(_ context.Context, request ExecutionStatusRequest) (ExecutionWorkItem, error) {
	if m.statusErr != nil {
		return m.statusEvidence, m.statusErr
	}
	m.statuses++
	for i := range m.snapshot.Items {
		if sameExternalLocation(m.snapshot.Items[i].Reference, request.WorkItem) {
			m.snapshot.Items[i].Status = request.Status
			m.snapshot.Items[i].Reference.Revision = "item-2"
			m.snapshot.Workspace.Revision = "workspace-2"
			return m.snapshot.Items[i], nil
		}
	}
	return ExecutionWorkItem{}, errors.New("missing item")
}

func executionFixture() (*Service, *executionMemory, planning.ArtifactRef, ExternalReference) {
	workspace := ExternalReference{Provider: "test", Kind: "project", OpaqueID: "project-1", DisplayID: "1", URL: "https://example.test/projects/1", Revision: "workspace-1"}
	artifact := planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: "work"}
	issue := ExternalReference{Provider: "test", Kind: "issue", OpaqueID: "issue-1", DisplayID: "1", URL: "https://example.test/issues/1"}
	memory := &executionMemory{snapshot: ExecutionWorkspaceSnapshot{SchemaVersion: IntegrationContractVersion, Workspace: workspace, ConfigurationRevision: "configuration-1", Fields: []ExecutionField{{Role: ExecutionFieldRoleStatus, SupportedStatuses: []ExecutionStatus{ExecutionTodo, ExecutionInProgress, ExecutionInReview, ExecutionDone}}}}}
	return New(nil, Options{ModuleID: "test"}), memory, artifact, issue
}

func TestExecutionAttachPreviewApplyAndRerun(t *testing.T) {
	service, workspace, artifact, issue := executionFixture()
	policy, events := &collaborationPolicy{}, &collaborationEvents{}
	request := ExecutionAttachRequest{Workspace: workspace.snapshot.Workspace, Artifact: artifact, PublishedWork: issue}
	plan, err := service.PreviewExecutionAttach(context.Background(), workspace, request, policy)
	if err != nil || plan.Action != MutationCreate || workspace.attaches != 0 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	if _, err = service.ApplyExecutionAttach(context.Background(), workspace, ExecutionAttachApplyInput{ExpectedPlan: plan}, policy, events); !errors.Is(err, ErrConfirmationRequired) {
		t.Fatal(err)
	}
	result, err := service.ApplyExecutionAttach(context.Background(), workspace, ExecutionAttachApplyInput{ExpectedPlan: plan, Confirmed: true}, policy, events)
	if err != nil || result.Action != MutationCreate || workspace.attaches != 1 || events.calls != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	plan, err = service.PreviewExecutionAttach(context.Background(), workspace, request, &collaborationPolicy{})
	if err != nil || plan.Action != MutationUnchanged {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	result, err = service.ApplyExecutionAttach(context.Background(), workspace, ExecutionAttachApplyInput{ExpectedPlan: plan, Confirmed: true}, &collaborationPolicy{}, events)
	if err != nil || result.Action != MutationUnchanged || workspace.attaches != 1 || events.calls != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestExecutionStatusTransitionsAndStalePlan(t *testing.T) {
	service, workspace, artifact, issue := executionFixture()
	item, _ := workspace.Attach(context.Background(), ExecutionAttachRequest{Workspace: workspace.snapshot.Workspace, Artifact: artifact, PublishedWork: issue})
	policy, events := &collaborationPolicy{}, &collaborationEvents{}
	request := ExecutionStatusRequest{Workspace: workspace.snapshot.Workspace, WorkItem: item.Reference, Status: ExecutionInReview}
	plan, err := service.PreviewExecutionStatus(context.Background(), workspace, request, policy)
	if err != nil || plan.Action != MutationUpdate {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	stale := plan
	workspace.snapshot.Items[0].Status = ExecutionInProgress
	if _, err = service.ApplyExecutionStatus(context.Background(), workspace, ExecutionStatusApplyInput{ExpectedPlan: stale, Confirmed: true}, policy, events); !isIntegrationClass(err, IntegrationRevisionConflict) {
		t.Fatalf("stale error=%v", err)
	}
	request.Workspace = workspace.snapshot.Workspace
	request.WorkItem = workspace.snapshot.Items[0].Reference
	plan, err = service.PreviewExecutionStatus(context.Background(), workspace, request, policy)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ApplyExecutionStatus(context.Background(), workspace, ExecutionStatusApplyInput{ExpectedPlan: plan, Confirmed: true}, policy, events)
	if err != nil || result.Item.Status != ExecutionInReview || workspace.statuses != 1 || events.calls != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	request.Workspace = workspace.snapshot.Workspace
	request.WorkItem = workspace.snapshot.Items[0].Reference
	plan, err = service.PreviewExecutionStatus(context.Background(), workspace, request, policy)
	if err != nil || plan.Action != MutationUnchanged {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestExecutionStatusRequiresSupportedUnambiguousField(t *testing.T) {
	service, workspace, artifact, issue := executionFixture()
	item, _ := workspace.Attach(context.Background(), ExecutionAttachRequest{Workspace: workspace.snapshot.Workspace, Artifact: artifact, PublishedWork: issue})
	workspace.snapshot.Fields[0].SupportedStatuses = []ExecutionStatus{ExecutionTodo, ExecutionDone}
	_, err := service.PreviewExecutionStatus(context.Background(), workspace, ExecutionStatusRequest{Workspace: workspace.snapshot.Workspace, WorkItem: item.Reference, Status: ExecutionInReview}, &collaborationPolicy{})
	if !isIntegrationClass(err, IntegrationManualRemediationRequired) {
		t.Fatalf("error=%v", err)
	}
}

func TestExecutionStatusCanInitializeUnsetField(t *testing.T) {
	service, workspace, artifact, issue := executionFixture()
	item, _ := workspace.Attach(context.Background(), ExecutionAttachRequest{Workspace: workspace.snapshot.Workspace, Artifact: artifact, PublishedWork: issue})
	workspace.snapshot.Items[0].Status = ""
	request := ExecutionStatusRequest{Workspace: workspace.snapshot.Workspace, WorkItem: item.Reference, Status: ExecutionInProgress}
	plan, err := service.PreviewExecutionStatus(context.Background(), workspace, request, &collaborationPolicy{})
	if err != nil || plan.Action != MutationUpdate {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestExecutionPartialVerificationStillAuditsConfirmedWrites(t *testing.T) {
	service, workspace, artifact, issue := executionFixture()
	policy, events := &collaborationPolicy{}, &collaborationEvents{}
	partial := &IntegrationError{Class: IntegrationPartialFailure, Operation: "test", Message: "verification failed"}
	attachPlan, err := service.PreviewExecutionAttach(context.Background(), workspace, ExecutionAttachRequest{Workspace: workspace.snapshot.Workspace, Artifact: artifact, PublishedWork: issue}, policy)
	if err != nil {
		t.Fatal(err)
	}
	workspace.attachErr = partial
	workspace.attachEvidence = ExecutionWorkItem{Artifact: artifact, PublishedWork: issue, Reference: ExternalReference{Provider: "test", Kind: "project_item", OpaqueID: "item-1", DisplayID: "#1", URL: issue.URL}}
	result, err := service.ApplyExecutionAttach(context.Background(), workspace, ExecutionAttachApplyInput{ExpectedPlan: attachPlan, Confirmed: true}, policy, events)
	if !isIntegrationClass(err, IntegrationPartialFailure) || result.Event == nil || events.calls != 1 {
		t.Fatalf("attach result=%+v events=%d err=%v", result, events.calls, err)
	}

	workspace.attachErr = nil
	workspace.snapshot.Items = []ExecutionWorkItem{{Artifact: artifact, PublishedWork: issue, Reference: ExternalReference{Provider: "test", Kind: "project_item", OpaqueID: "item-1", DisplayID: "#1", URL: issue.URL, Revision: "item-1"}, Status: ExecutionTodo}}
	statusPlan, err := service.PreviewExecutionStatus(context.Background(), workspace, ExecutionStatusRequest{Workspace: workspace.snapshot.Workspace, WorkItem: workspace.snapshot.Items[0].Reference, Status: ExecutionDone}, policy)
	if err != nil {
		t.Fatal(err)
	}
	workspace.statusErr = partial
	workspace.statusEvidence = workspace.snapshot.Items[0]
	workspace.statusEvidence.Status = ExecutionDone
	workspace.statusEvidence.Reference.Revision = ""
	result, err = service.ApplyExecutionStatus(context.Background(), workspace, ExecutionStatusApplyInput{ExpectedPlan: statusPlan, Confirmed: true}, policy, events)
	if !isIntegrationClass(err, IntegrationPartialFailure) || result.Event == nil || events.calls != 2 {
		t.Fatalf("status result=%+v events=%d err=%v", result, events.calls, err)
	}
}

func isIntegrationClass(err error, class IntegrationErrorClass) bool {
	var integration *IntegrationError
	return errors.As(err, &integration) && integration.Class == class
}
