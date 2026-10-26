package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/JimmyMcBride/brain/planning"
)

type reconciliationRepositorySource struct {
	evidence RepositoryEvidence
	reads    int
}

func (s *reconciliationRepositorySource) Current(context.Context) (RepositoryEvidence, error) {
	s.reads++
	return s.evidence, nil
}

type reconciliationMappings struct {
	state ExternalMappingState
	loads int
	saves int
	fail  error
}

func (m *reconciliationMappings) Load(context.Context) (ExternalMappingState, error) {
	m.loads++
	return m.state, nil
}

func (m *reconciliationMappings) Save(_ context.Context, state ExternalMappingState, expected string) (ExternalMappingState, error) {
	m.saves++
	if expected != m.state.Revision {
		return m.state, reconciliationConflict("stale test mapping")
	}
	if m.fail != nil {
		return m.state, m.fail
	}
	state.Revision = "mapping-2"
	m.state = state
	return state, nil
}

type reconciliationTarget struct {
	snapshot PublicationSnapshot
	reads    int
	writes   int
	fail     error
}

func (t *reconciliationTarget) Inspect(context.Context, PublicationInspectRequest) (PublicationSnapshot, error) {
	t.reads++
	return t.snapshot, nil
}

func (t *reconciliationTarget) Apply(_ context.Context, plan PublicationPlan) (PublicationResult, error) {
	t.writes++
	result := PublicationResult{SchemaVersion: IntegrationContractVersion}
	for _, action := range plan.Actions {
		evidence := PublicationActionEvidence{Action: action}
		if action.Artifact != nil {
			ref := *action.Artifact.Reference
			evidence.References = []ExternalReference{ref}
			artifact := *action.Artifact
			found := false
			for i := range t.snapshot.Artifacts {
				if t.snapshot.Artifacts[i].Artifact == artifact.Artifact {
					t.snapshot.Artifacts[i] = artifact
					found = true
				}
			}
			if !found {
				t.snapshot.Artifacts = append(t.snapshot.Artifacts, artifact)
			}
		}
		result.Completed = append(result.Completed, evidence)
		if t.fail != nil {
			return result, t.fail
		}
	}
	return result, nil
}

func reconciliationFixture() (*Service, *reconciliationTarget, *reconciliationRepositorySource, *reconciliationMappings, ReconciliationPreviewInput) {
	repo := ExternalReference{Provider: "test", Kind: "repository", OpaqueID: "owner/repo", DisplayID: "owner/repo", URL: "https://example.test/owner/repo"}
	artifact := planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: "reconcile"}
	issue := ExternalReference{Provider: "test", Kind: "issue", OpaqueID: "issue-1", DisplayID: "1", URL: "https://example.test/owner/repo/issues/1", Revision: "issue-1"}
	issueSelector := issue
	issueSelector.OpaqueID = issue.URL
	issueSelector.Revision = ""
	input := ReconciliationPreviewInput{Publication: PublicationPreviewInput{Target: repo, Artifacts: []PublicationArtifact{{Artifact: artifact, Title: "Reconcile", Content: "canonical", Readiness: planning.ReadinessReady}}}}
	target := &reconciliationTarget{snapshot: PublicationSnapshot{SchemaVersion: IntegrationContractVersion, Target: repo, Artifacts: []PublicationArtifact{{Artifact: artifact, Title: "Reconcile", Content: "stale", Readiness: planning.ReadinessNeedsRefinement, Reference: &issue}}}}
	repository := &reconciliationRepositorySource{evidence: RepositoryEvidence{SchemaVersion: IntegrationContractVersion, Repository: repo, DefaultRef: "develop", CurrentRef: "codex/reconcile", Commit: "0123456789abcdef0123456789abcdef01234567"}}
	mappings := &reconciliationMappings{state: ExternalMappingState{SchemaVersion: IntegrationContractVersion, Revision: "mapping-1", ArtifactReferences: []ArtifactExternalReference{{Artifact: artifact, Reference: issueSelector}}}}
	service := New(nil, Options{ModuleID: "test", Now: func() time.Time { return time.Unix(5, 0) }})
	return service, target, repository, mappings, input
}

func TestReconciliationPreviewAndApplyLifecycle(t *testing.T) {
	service, target, repository, mappings, input := reconciliationFixture()
	policy := &collaborationPolicy{}
	plan, err := service.PreviewReconciliation(context.Background(), target, repository, mappings, nil, input, policy)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Repository.Commit != repository.evidence.Commit || plan.ExpectedMappingRevision != "mapping-1" || len(plan.Publication.Actions) != 1 || plan.Publication.Actions[0].Action != MutationUpdate || plan.Publication.Actions[0].Artifact.Reference == nil {
		t.Fatalf("plan=%+v", plan)
	}
	if target.writes != 0 || mappings.saves != 0 || !reflect.DeepEqual(policy.calls, []string{PermissionRead}) {
		t.Fatalf("preview mutated state or skipped authorization: policy=%v writes=%d saves=%d", policy.calls, target.writes, mappings.saves)
	}
	events := &collaborationEvents{}
	result, err := service.ApplyReconciliation(context.Background(), target, repository, mappings, nil, ReconciliationApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}, policy, events)
	if err != nil {
		t.Fatal(err)
	}
	if target.writes != 1 || mappings.saves != 1 || events.calls != 1 || result.Event == nil || result.Event.Name != EventIntegrationReconciled || result.Mappings.Revision != "mapping-2" || result.Evidence.Repository.Commit != repository.evidence.Commit || !result.Evidence.CompletedAt.Equal(time.Unix(5, 0)) {
		t.Fatalf("result=%+v writes=%d saves=%d events=%d", result, target.writes, mappings.saves, events.calls)
	}
	if result.Mappings.LastReconciliation == nil || result.Mappings.LastReconciliation.Repository.Commit != repository.evidence.Commit {
		t.Fatalf("mapping evidence=%+v", result.Mappings.LastReconciliation)
	}

	plan, err = service.PreviewReconciliation(context.Background(), target, repository, mappings, nil, input, &collaborationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	result, err = service.ApplyReconciliation(context.Background(), target, repository, mappings, nil, ReconciliationApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}, &collaborationPolicy{}, events)
	if err != nil || target.writes != 1 || mappings.saves != 1 || events.calls != 1 || result.Event != nil {
		t.Fatalf("rerun result=%+v writes=%d saves=%d events=%d err=%v", result, target.writes, mappings.saves, events.calls, err)
	}
}

func TestReconciliationRepairsExecutionWorkspaceInReviewedSteps(t *testing.T) {
	service, target, repository, mappings, input := reconciliationFixture()
	artifact := input.Publication.Artifacts[0].Artifact
	issue := *target.snapshot.Artifacts[0].Reference
	workspaceRef := ExternalReference{Provider: "test", Kind: "project", OpaqueID: "project-1", DisplayID: "1", URL: "https://example.test/projects/1", Revision: "workspace-1"}
	mappings.state.WorkspaceReferences = []ExternalReference{workspaceRef}
	execution := &executionMemory{snapshot: ExecutionWorkspaceSnapshot{SchemaVersion: IntegrationContractVersion, Workspace: workspaceRef, Fields: []ExecutionField{{Role: ExecutionFieldRoleStatus, SupportedStatuses: []ExecutionStatus{ExecutionTodo, ExecutionInProgress, ExecutionInReview, ExecutionDone}}}}}
	input.Execution = []ExecutionIntent{{Artifact: artifact, PublishedWork: issue, Status: ExecutionInProgress}}

	plan, err := service.PreviewReconciliation(context.Background(), target, repository, mappings, execution, input, &collaborationPolicy{})
	if err != nil || len(plan.Attachments) != 1 || len(plan.Execution) != 0 {
		t.Fatalf("attach plan=%+v err=%v", plan, err)
	}
	result, err := service.ApplyReconciliation(context.Background(), target, repository, mappings, execution, ReconciliationApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}, &collaborationPolicy{}, &collaborationEvents{})
	if err != nil || execution.attaches != 1 || len(result.Execution) != 1 {
		t.Fatalf("attach result=%+v err=%v", result, err)
	}

	plan, err = service.PreviewReconciliation(context.Background(), target, repository, mappings, execution, input, &collaborationPolicy{})
	if err != nil || len(plan.Attachments) != 0 || len(plan.Execution) != 1 {
		t.Fatalf("status plan=%+v err=%v", plan, err)
	}
	result, err = service.ApplyReconciliation(context.Background(), target, repository, mappings, execution, ReconciliationApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}, &collaborationPolicy{}, &collaborationEvents{})
	if err != nil || execution.statuses != 1 || result.Execution[0].Status != ExecutionInProgress {
		t.Fatalf("status result=%+v err=%v", result, err)
	}

	plan, err = service.PreviewReconciliation(context.Background(), target, repository, mappings, execution, input, &collaborationPolicy{})
	if err != nil || len(plan.Attachments) != 0 || len(plan.Execution) != 0 {
		t.Fatalf("rerun plan=%+v err=%v", plan, err)
	}
}

func TestReconciliationGatesAndStaleEvidence(t *testing.T) {
	for _, test := range []string{"confirmation", "publish", "read", "events", "plan", "repository", "mapping"} {
		t.Run(test, func(t *testing.T) {
			service, target, repository, mappings, input := reconciliationFixture()
			plan, err := service.PreviewReconciliation(context.Background(), target, repository, mappings, nil, input, &collaborationPolicy{})
			if err != nil {
				t.Fatal(err)
			}
			apply := ReconciliationApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}
			policy := &collaborationPolicy{}
			var events EventSink = &collaborationEvents{}
			switch test {
			case "confirmation":
				apply.Confirmed = false
			case "publish":
				policy.deny = PermissionPublish
			case "read":
				policy.deny = PermissionRead
			case "events":
				events = nil
			case "plan":
				apply.ExpectedPlan.Repository.Commit = "changed"
			case "repository":
				repository.evidence.Commit = "abcdefabcdefabcdefabcdefabcdefabcdefabcd"
			case "mapping":
				mappings.state.Revision = "changed"
			}
			_, err = service.ApplyReconciliation(context.Background(), target, repository, mappings, nil, apply, policy, events)
			if err == nil || target.writes != 0 || mappings.saves != 0 {
				t.Fatalf("gate=%s writes=%d saves=%d err=%v", test, target.writes, mappings.saves, err)
			}
		})
	}
}

func TestReconciliationRequiresMappedCanonicalArtifacts(t *testing.T) {
	service, target, repository, mappings, input := reconciliationFixture()
	mappings.state.ArtifactReferences = nil
	_, err := service.PreviewReconciliation(context.Background(), target, repository, mappings, nil, input, &collaborationPolicy{})
	var integration *IntegrationError
	if !errors.As(err, &integration) || integration.Class != IntegrationManualRemediationRequired || !integration.ManualFallbackAllowed || target.reads != 0 {
		t.Fatalf("error=%v reads=%d", err, target.reads)
	}
}

func TestReconciliationPartialAndAuditFailuresRetainEvidence(t *testing.T) {
	for _, fail := range []string{"provider", "mapping", "audit"} {
		t.Run(fail, func(t *testing.T) {
			service, target, repository, mappings, input := reconciliationFixture()
			plan, err := service.PreviewReconciliation(context.Background(), target, repository, mappings, nil, input, &collaborationPolicy{})
			if err != nil {
				t.Fatal(err)
			}
			providerErr := errors.New("provider failed")
			if fail == "provider" {
				target.fail = providerErr
			}
			if fail == "mapping" {
				mappings.fail = errors.New("mapping failed")
			}
			events := &collaborationEvents{fail: fail == "audit"}
			result, err := service.ApplyReconciliation(context.Background(), target, repository, mappings, nil, ReconciliationApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}, &collaborationPolicy{}, events)
			var integration *IntegrationError
			if err == nil || !errors.As(err, &integration) || integration.Class != IntegrationPartialFailure || len(result.Publication.Completed) != 1 || result.Event == nil || events.calls != 1 {
				t.Fatalf("result=%+v events=%d err=%v", result, events.calls, err)
			}
			if fail == "provider" && !errors.Is(err, providerErr) {
				t.Fatal(err)
			}
		})
	}
}
