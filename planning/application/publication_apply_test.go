package application

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type publicationApplyMemory struct {
	*publicationMemory
	writes int
	apply  func(context.Context, PublicationPlan) (PublicationResult, error)
}

func (m *publicationApplyMemory) Apply(ctx context.Context, plan PublicationPlan) (PublicationResult, error) {
	m.writes++
	return m.apply(ctx, plan)
}

func confirmedPublicationFixture(t *testing.T) (*Service, *publicationApplyMemory, PublicationApplyInput) {
	t.Helper()
	intent, memory := publicationFixture()
	service := New(nil, Options{ModuleID: "test", Now: func() time.Time { return time.Unix(1, 0) }})
	plan, err := service.PreviewPublication(context.Background(), memory, intent, &collaborationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	memory.reads = 0
	target := &publicationApplyMemory{publicationMemory: memory}
	target.apply = func(_ context.Context, plan PublicationPlan) (PublicationResult, error) {
		result := PublicationResult{SchemaVersion: IntegrationContractVersion}
		for _, action := range plan.Actions {
			evidence := PublicationActionEvidence{Action: action}
			if action.Group != nil {
				group := *action.Group
				ref := ExternalReference{Provider: "test", Kind: "group", OpaqueID: "group", DisplayID: "1", URL: "https://example.test/groups/1"}
				group.Reference = &ref
				target.snapshot.Group = &group
				evidence.References = []ExternalReference{ref}
			} else if action.Artifact != nil {
				artifact := *action.Artifact
				ref := ExternalReference{Provider: "test", Kind: "work", OpaqueID: string(artifact.Artifact.ID), Revision: "first"}
				artifact.Reference = &ref
				target.snapshot.Artifacts = append(target.snapshot.Artifacts, artifact)
				evidence.References = []ExternalReference{ref}
			} else if action.Relationship != nil {
				target.snapshot.Relationships = append(target.snapshot.Relationships, *action.Relationship)
			} else if action.Workspace != nil && action.Workspace.Choice != PublicationWorkspaceSkip {
				workspace := *action.Workspace
				ref := ExternalReference{Provider: "test", Kind: "workspace", OpaqueID: "workspace-1", DisplayID: "1", URL: "https://example.test/workspaces/1"}
				workspace.Reference = &ref
				target.snapshot.Workspace = &workspace
				evidence.References = []ExternalReference{ref}
			}
			result.Completed = append(result.Completed, evidence)
		}
		return result, nil
	}
	return service, target, PublicationApplyInput{Intent: intent, ExpectedPlan: plan, Confirmed: true}
}

func TestPublicationApplyLifecycleAndNoOpRerun(t *testing.T) {
	service, target, input := confirmedPublicationFixture(t)
	// Reviewed plans can travel through a JSON CLI without false conflicts.
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip PublicationApplyInput
	if err := json.Unmarshal(raw, &roundTrip); err != nil {
		t.Fatal(err)
	}
	policy, events := &collaborationPolicy{}, &collaborationEvents{}
	result, err := service.ApplyPublication(context.Background(), target, roundTrip, policy, events)
	if err != nil || target.writes != 1 || target.reads != 1 || events.calls != 1 || len(result.Evidence.Completed) != 7 {
		t.Fatalf("result=%+v reads=%d writes=%d events=%d err=%v", result, target.reads, target.writes, events.calls, err)
	}
	if !reflect.DeepEqual(policy.calls, []string{PermissionPublish, PermissionRead}) || result.Event.Name != EventPublicationApplied || result.Event.Source.OpaqueID != input.Intent.Target.OpaqueID || result.Event.ModuleID != "test" || !result.Event.OccurredAt.Equal(time.Unix(1, 0)) {
		t.Fatalf("policy=%v event=%+v", policy.calls, result.Event)
	}
	// An old create plan must not be silently converted into reuse after a write.
	if _, err := service.ApplyPublication(context.Background(), target, input, policy, events); err == nil || target.writes != 1 {
		t.Fatal("stale create plan reapplied", err)
	}
	for _, mapped := range []bool{false, true} {
		if mapped {
			input.Intent.Artifacts = target.snapshot.Artifacts
		}
		input.ExpectedPlan, err = service.PreviewPublication(context.Background(), target, input.Intent, policy)
		if err != nil {
			t.Fatal(err)
		}
		result, err = service.ApplyPublication(context.Background(), target, input, policy, events)
		if err != nil || target.writes != 1 || events.calls != 1 || result.Event != nil || len(result.Evidence.Completed) != 7 {
			t.Fatalf("no-op rerun mapped=%v result=%+v err=%v", mapped, result, err)
		}
		if len(result.Evidence.Completed[0].References) != 1 {
			t.Fatal("lost reusable identity")
		}
	}
}

func TestPublicationApplyNoOpRetainsCoordinationEvidence(t *testing.T) {
	input, memory := fiveSpecPublicationFixture()
	service := New(nil, Options{})
	initial, err := service.PreviewPublication(context.Background(), memory, input, &collaborationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range initial.Actions {
		switch {
		case action.Group != nil:
			group := *action.Group
			ref := ExternalReference{Provider: "test", Kind: "group", OpaqueID: "group-1", DisplayID: "1", URL: "https://example.test/groups/1"}
			group.Reference = &ref
			memory.snapshot.Group = &group
			input.Group = &group
		case action.Artifact != nil:
			artifact := *action.Artifact
			ref := ExternalReference{Provider: "test", Kind: "work", OpaqueID: string(artifact.Artifact.ID)}
			artifact.Reference = &ref
			memory.snapshot.Artifacts = append(memory.snapshot.Artifacts, artifact)
		case action.Relationship != nil:
			memory.snapshot.Relationships = append(memory.snapshot.Relationships, *action.Relationship)
		case action.Workspace != nil:
			ref := ExternalReference{Provider: "test", Kind: "workspace", OpaqueID: "workspace-1", DisplayID: "1", URL: "https://example.test/workspaces/1"}
			memory.snapshot.Workspace = &PublicationWorkspaceDecision{Choice: PublicationWorkspaceCreate, Title: action.Workspace.Title, Reference: &ref}
			input.Workspace = &PublicationWorkspaceDecision{Choice: PublicationWorkspaceConnect, Reason: "Use reviewed workspace.", Reference: &ref}
		}
	}
	input.Artifacts = memory.snapshot.Artifacts
	plan, err := service.PreviewPublication(context.Background(), memory, input, &collaborationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	target := &publicationApplyMemory{publicationMemory: memory, apply: func(context.Context, PublicationPlan) (PublicationResult, error) {
		t.Fatal("no-op coordination plan reached provider apply")
		return PublicationResult{}, nil
	}}
	result, err := service.ApplyPublication(context.Background(), target, PublicationApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}, &collaborationPolicy{}, &collaborationEvents{})
	if err != nil || target.writes != 0 || result.Event != nil {
		t.Fatalf("result=%+v writes=%d err=%v", result, target.writes, err)
	}
	if got := result.Evidence.Completed[0].References; len(got) != 1 || got[0].Kind != "group" {
		t.Fatalf("group evidence=%+v", got)
	}
	if got := result.Evidence.Completed[len(result.Evidence.Completed)-1].References; len(got) != 1 || got[0].Kind != "workspace" {
		t.Fatalf("workspace evidence=%+v", got)
	}
}

func TestPublicationApplyGatesBeforeProviderWork(t *testing.T) {
	for _, test := range []string{"confirmation", "publish", "read", "authorizer", "events", "preview", "target"} {
		t.Run(test, func(t *testing.T) {
			service, target, input := confirmedPublicationFixture(t)
			var policy Authorizer = &collaborationPolicy{}
			var events EventSink = &collaborationEvents{}
			var port PublicationTarget = target
			switch test {
			case "confirmation":
				input.Confirmed = false
			case "publish":
				policy = &collaborationPolicy{deny: PermissionPublish}
			case "read":
				policy = &collaborationPolicy{deny: PermissionRead}
			case "authorizer":
				policy = nil
			case "events":
				events = nil
			case "preview":
				input.ExpectedPlan = PublicationPlan{}
			case "target":
				port = nil
			}
			if _, err := service.ApplyPublication(context.Background(), port, input, policy, events); err == nil || target.reads != 0 || target.writes != 0 {
				t.Fatalf("gate failed reads=%d writes=%d err=%v", target.reads, target.writes, err)
			}
		})
	}
}

func TestPublicationApplyRejectsChangedIntentAndProviderRevision(t *testing.T) {
	for _, change := range []string{"content", "source", "revision", "actions"} {
		t.Run(change, func(t *testing.T) {
			service, target, input := confirmedPublicationFixture(t)
			if change == "revision" {
				_, err := service.ApplyPublication(context.Background(), target, input, &collaborationPolicy{}, &collaborationEvents{})
				if err != nil {
					t.Fatal(err)
				}
				input.ExpectedPlan, err = service.PreviewPublication(context.Background(), target, input.Intent, &collaborationPolicy{})
				if err != nil {
					t.Fatal(err)
				}
				target.snapshot.Artifacts[0].Reference.Revision = "changed"
			} else if change == "content" {
				input.Intent.Artifacts[0].Content = "unreviewed"
			} else if change == "source" {
				input.Intent.Source = &ExternalReference{Provider: "test", Kind: "conversation", OpaqueID: "other"}
			} else {
				input.ExpectedPlan.Actions = input.ExpectedPlan.Actions[1:]
			}
			writes := target.writes
			events := &collaborationEvents{}
			_, err := service.ApplyPublication(context.Background(), target, input, &collaborationPolicy{}, events)
			var integration *IntegrationError
			if !errors.As(err, &integration) || integration.Class != IntegrationRevisionConflict || target.writes != writes || events.calls != 0 {
				t.Fatalf("accepted changed plan: %v", err)
			}
		})
	}
}

func TestPublicationApplyPartialAndAuditFailuresKeepEvidence(t *testing.T) {
	for _, failAudit := range []bool{false, true} {
		service, target, input := confirmedPublicationFixture(t)
		providerErr := errors.New("provider failed")
		target.apply = func(_ context.Context, plan PublicationPlan) (PublicationResult, error) {
			return PublicationResult{SchemaVersion: 1, Completed: []PublicationActionEvidence{{Action: plan.Actions[0], References: []ExternalReference{{Provider: "test", Kind: "work", OpaqueID: "created"}}}}, Failed: &PublicationActionEvidence{Action: plan.Actions[1]}, ManualFallbackAllowed: true, ManualFallbackReason: "provider-approved recovery"}, providerErr
		}
		events := &collaborationEvents{fail: failAudit}
		result, err := service.ApplyPublication(context.Background(), target, input, &collaborationPolicy{}, events)
		if !errors.Is(err, providerErr) || len(result.Evidence.Completed) != 1 || result.Evidence.Failed == nil || !result.Evidence.ManualFallbackAllowed || result.Evidence.ManualFallbackReason != "provider-approved recovery" || events.calls != 1 || result.Event == nil {
			t.Fatalf("lost partial evidence: %+v %v", result, err)
		}
		if failAudit {
			var integration *IntegrationError
			if !errors.As(err, &integration) || integration.Class != IntegrationPartialFailure || !strings.Contains(err.Error(), "audit persistence failed") {
				t.Fatal(err)
			}
		}
	}
}

func TestPublicationApplyRejectsFalseSuccessAndAuditsCompletedWrites(t *testing.T) {
	for _, failure := range []string{"empty", "partial", "wrong-order", "failed", "identity", "audit"} {
		t.Run(failure, func(t *testing.T) {
			service, target, input := confirmedPublicationFixture(t)
			apply := target.apply
			target.apply = func(ctx context.Context, plan PublicationPlan) (PublicationResult, error) {
				result, err := apply(ctx, plan)
				switch failure {
				case "empty":
					result = PublicationResult{}
				case "partial":
					result.Completed = result.Completed[:1]
				case "wrong-order":
					result.Completed[0].Action = plan.Actions[1]
				case "failed":
					result.Failed = &PublicationActionEvidence{Action: plan.Actions[0]}
				case "identity":
					result.Completed[0].References = nil
				}
				return result, err
			}
			events := &collaborationEvents{fail: failure == "audit"}
			result, err := service.ApplyPublication(context.Background(), target, input, &collaborationPolicy{}, events)
			var integration *IntegrationError
			if !errors.As(err, &integration) || integration.Class != IntegrationPartialFailure {
				t.Fatal("accepted false success", err)
			}
			if failure != "empty" && (events.calls != 1 || result.Event == nil) {
				t.Fatal("lost completed mutation audit")
			}
		})
	}
}

type publicationContextEvents struct{ calls int }

func (e *publicationContextEvents) Publish(ctx context.Context, _ Event) error {
	e.calls++
	return ctx.Err()
}

func TestPublicationApplyCancellationAndPreMutationFailure(t *testing.T) {
	for _, completed := range []bool{false, true} {
		service, target, input := confirmedPublicationFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		target.apply = func(_ context.Context, plan PublicationPlan) (PublicationResult, error) {
			cancel()
			result := PublicationResult{SchemaVersion: 1}
			if completed {
				result.Completed = []PublicationActionEvidence{{Action: plan.Actions[0], References: []ExternalReference{{Provider: "test", Kind: "work", OpaqueID: "created"}}}}
			}
			return result, context.Canceled
		}
		events := &publicationContextEvents{}
		_, err := service.ApplyPublication(ctx, target, input, &collaborationPolicy{}, events)
		if !errors.Is(err, context.Canceled) || (completed && events.calls != 1) || (!completed && events.calls != 0) {
			t.Fatal(err, events.calls)
		}
		if _, err := service.ApplyPublication(ctx, target, input, &collaborationPolicy{}, events); !errors.Is(err, context.Canceled) || target.writes != 1 {
			t.Fatal("cancelled request reached apply", err)
		}
	}
}

func TestPublicationApplyAuditsWriteWithinFailedAction(t *testing.T) {
	service, target, input := confirmedPublicationFixture(t)
	target.apply = func(_ context.Context, plan PublicationPlan) (PublicationResult, error) {
		return PublicationResult{SchemaVersion: 1, Failed: &PublicationActionEvidence{Action: plan.Actions[0], References: []ExternalReference{{Provider: "test", Kind: "work", OpaqueID: "created"}}}}, errors.New("follow-up write failed")
	}
	events := &collaborationEvents{}
	result, err := service.ApplyPublication(context.Background(), target, input, &collaborationPolicy{}, events)
	var integration *IntegrationError
	if !errors.As(err, &integration) || integration.Class != IntegrationPartialFailure || events.calls != 1 || result.Event == nil || result.Evidence.Failed == nil {
		t.Fatalf("lost failed-action mutation: %+v %v", result, err)
	}
}
