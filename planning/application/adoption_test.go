package application

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/JimmyMcBride/brain/planning"
)

type adoptionMappingMemory struct {
	state        ExternalMappingState
	loads, saves int
	saveErr      error
}

func (m *adoptionMappingMemory) Load(context.Context) (ExternalMappingState, error) {
	m.loads++
	return m.state, nil
}

func (m *adoptionMappingMemory) Save(_ context.Context, state ExternalMappingState, expectedRevision string) (ExternalMappingState, error) {
	m.saves++
	if m.saveErr != nil {
		return ExternalMappingState{}, m.saveErr
	}
	if expectedRevision != m.state.Revision {
		return ExternalMappingState{}, &IntegrationError{Class: IntegrationRevisionConflict}
	}
	state.Revision = "next"
	m.state = state
	return state, nil
}

type adoptionTargetMemory struct {
	snapshot      PublicationSnapshot
	reads, writes int
	applyErr      error
}

func (m *adoptionTargetMemory) Inspect(_ context.Context, request PublicationInspectRequest) (PublicationSnapshot, error) {
	m.reads++
	if len(request.Candidates) != len(request.Artifacts) {
		return PublicationSnapshot{}, errors.New("candidate selectors missing")
	}
	return m.snapshot, nil
}

func (m *adoptionTargetMemory) Apply(_ context.Context, plan PublicationPlan) (PublicationResult, error) {
	m.writes++
	result := PublicationResult{SchemaVersion: IntegrationContractVersion}
	for _, action := range plan.Actions {
		evidence := PublicationActionEvidence{Action: action}
		if action.Artifact != nil {
			evidence.References = []ExternalReference{*action.Artifact.Reference}
		}
		result.Completed = append(result.Completed, evidence)
	}
	if m.applyErr != nil {
		result.Completed = result.Completed[:1]
		result.Failed = &PublicationActionEvidence{Action: plan.Actions[1]}
		return result, m.applyErr
	}
	var artifacts []PublicationArtifact
	var relationships []PublicationRelationship
	for _, action := range plan.Actions {
		if action.Artifact != nil {
			artifacts = append(artifacts, *action.Artifact)
		} else if action.Relationship != nil {
			relationships = append(relationships, *action.Relationship)
		}
	}
	m.snapshot.Artifacts = artifacts
	m.snapshot.Relationships = relationships
	return result, nil
}

func adoptionFixture(t *testing.T) (*Service, *adoptionTargetMemory, *adoptionMappingMemory, AdoptionPreviewInput, AdoptionPlan) {
	t.Helper()
	initiative := planning.ArtifactRef{Kind: planning.ArtifactInitiative, ID: "initiative"}
	spec := planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: "spec"}
	targetRef := ExternalReference{Provider: "test", Kind: "repository", OpaqueID: "owner/repo", URL: "https://example.test/owner/repo"}
	source := ExternalReference{Provider: "test", Kind: "discussion", OpaqueID: "source", DisplayID: "9", URL: "https://example.test/discussions/9"}
	initiativeRef := ExternalReference{Provider: "test", Kind: "work", OpaqueID: "node-1", DisplayID: "1", URL: "https://example.test/issues/1", Revision: "r1"}
	specRef := ExternalReference{Provider: "test", Kind: "work", OpaqueID: "node-2", DisplayID: "2", URL: "https://example.test/issues/2", Revision: "r2"}
	target := &adoptionTargetMemory{snapshot: PublicationSnapshot{SchemaVersion: IntegrationContractVersion, Target: targetRef, Artifacts: []PublicationArtifact{
		{Artifact: initiative, Title: "Initiative", Content: "same", Readiness: planning.ReadinessReady, Reference: &initiativeRef},
		{Artifact: spec, Title: "Spec", Content: "old", Readiness: planning.ReadinessReady, Reference: &specRef},
	}}}
	mappings := &adoptionMappingMemory{state: ExternalMappingState{SchemaVersion: IntegrationContractVersion, Revision: "base"}}
	input := AdoptionPreviewInput{Target: targetRef, Source: &source, Artifacts: []PublicationArtifact{
		{Artifact: initiative, Title: "Initiative", Content: "same", Readiness: planning.ReadinessReady},
		{Artifact: spec, Title: "Spec", Content: "new", Readiness: planning.ReadinessReady, Group: &initiative},
	}, Candidates: []ArtifactExternalReference{
		{Artifact: initiative, Reference: ExternalReference{Provider: "test", Kind: "work", OpaqueID: initiativeRef.URL, DisplayID: "1", URL: initiativeRef.URL}},
		{Artifact: spec, Reference: ExternalReference{Provider: "test", Kind: "work", OpaqueID: specRef.URL, DisplayID: "2", URL: specRef.URL}},
	}}
	service := New(nil, Options{ModuleID: "test", Now: func() time.Time { return time.Unix(1, 0) }})
	plan, err := service.PreviewAdoption(context.Background(), target, mappings, input, &collaborationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	target.reads, mappings.loads = 0, 0
	return service, target, mappings, input, plan
}

func TestAdoptionPreviewApplyAndNoOpRerun(t *testing.T) {
	service, target, mappings, input, plan := adoptionFixture(t)
	if len(plan.Publication.Actions) != 3 || plan.Publication.Actions[0].Action != MutationReuse || plan.Publication.Actions[1].Action != MutationUpdate || plan.Publication.Actions[2].Action != MutationCreate || len(plan.Mappings.ArtifactReferences) != 2 || len(plan.Mappings.SourceReferences) != 1 {
		t.Fatalf("plan=%+v", plan)
	}
	raw, err := json.Marshal(plan)
	if err != nil || json.Unmarshal(raw, &plan) != nil {
		t.Fatal("adoption plan is not JSON round-trippable", err)
	}
	policy, events := &collaborationPolicy{}, &collaborationEvents{}
	result, err := service.ApplyAdoption(context.Background(), target, mappings, AdoptionApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}, policy, events)
	if err != nil || target.reads != 1 || target.writes != 1 || mappings.loads != 1 || mappings.saves != 1 || events.calls != 1 || result.Event == nil || result.Event.Name != EventIntegrationAdopted || result.Mappings.Revision != "next" || len(result.Actions) != 3 {
		t.Fatalf("result=%+v reads=%d writes=%d loads=%d saves=%d events=%d err=%v", result, target.reads, target.writes, mappings.loads, mappings.saves, events.calls, err)
	}
	if !reflect.DeepEqual(policy.calls, []string{PermissionPublish, PermissionRead}) {
		t.Fatal(policy.calls)
	}
	plan, err = service.PreviewAdoption(context.Background(), target, mappings, input, &collaborationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	writes, saves, eventCalls := target.writes, mappings.saves, events.calls
	result, err = service.ApplyAdoption(context.Background(), target, mappings, AdoptionApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}, &collaborationPolicy{}, events)
	if err != nil || target.writes != writes || mappings.saves != saves+1 || events.calls != eventCalls || result.Event != nil || result.Mappings.Revision != "next" {
		t.Fatalf("no-op result=%+v err=%v", result, err)
	}
}

func TestAdoptionGatesAndStalePreview(t *testing.T) {
	for _, gate := range []string{"confirmation", "publish", "read", "authorizer", "events", "target", "mappings"} {
		t.Run(gate, func(t *testing.T) {
			service, target, mappings, input, plan := adoptionFixture(t)
			apply := AdoptionApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}
			var policy Authorizer = &collaborationPolicy{}
			var events EventSink = &collaborationEvents{}
			var targetPort PublicationTarget = target
			var mappingPort ExternalMappingRepository = mappings
			switch gate {
			case "confirmation":
				apply.Confirmed = false
			case "publish":
				policy = &collaborationPolicy{deny: PermissionPublish}
			case "read":
				policy = &collaborationPolicy{deny: PermissionRead}
			case "authorizer":
				policy = nil
			case "events":
				events = nil
			case "target":
				targetPort = nil
			case "mappings":
				mappingPort = nil
			}
			if _, err := service.ApplyAdoption(context.Background(), targetPort, mappingPort, apply, policy, events); err == nil || target.reads != 0 || target.writes != 0 || mappings.loads != 0 || mappings.saves != 0 {
				t.Fatalf("gate=%s reads=%d writes=%d loads=%d saves=%d err=%v", gate, target.reads, target.writes, mappings.loads, mappings.saves, err)
			}
		})
	}
	service, target, mappings, input, plan := adoptionFixture(t)
	mappings.state.Revision = "changed"
	if _, err := service.ApplyAdoption(context.Background(), target, mappings, AdoptionApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}, &collaborationPolicy{}, &collaborationEvents{}); err == nil || target.writes != 0 || mappings.saves != 0 {
		t.Fatal("stale mapping preview reached mutation", err)
	}
}

func TestAdoptionRejectsMissingOrAmbiguousCandidatesBeforeMutation(t *testing.T) {
	service, target, mappings, input, _ := adoptionFixture(t)
	input.Candidates = input.Candidates[:1]
	if _, err := service.PreviewAdoption(context.Background(), target, mappings, input, &collaborationPolicy{}); err == nil || target.reads != 0 {
		t.Fatal("missing candidate reached provider", err)
	}
	_, target, mappings, input, _ = adoptionFixture(t)
	input.Candidates[1].Reference = input.Candidates[0].Reference
	if _, err := service.PreviewAdoption(context.Background(), target, mappings, input, &collaborationPolicy{}); err == nil || target.reads != 0 {
		t.Fatal("duplicate candidate reached provider", err)
	}
	_, target, mappings, input, _ = adoptionFixture(t)
	target.snapshot.Artifacts = target.snapshot.Artifacts[:1]
	if _, err := service.PreviewAdoption(context.Background(), target, mappings, input, &collaborationPolicy{}); err == nil {
		t.Fatal("missing provider object became create")
	}
}

func TestAdoptionRetainsEvidenceWhenMappingOrAuditFails(t *testing.T) {
	for _, auditFailure := range []bool{false, true} {
		service, target, mappings, input, plan := adoptionFixture(t)
		mappingErr := errors.New("mapping failed")
		mappings.saveErr = mappingErr
		events := &collaborationEvents{fail: auditFailure}
		result, err := service.ApplyAdoption(context.Background(), target, mappings, AdoptionApplyInput{Intent: input, ExpectedPlan: plan, Confirmed: true}, &collaborationPolicy{}, events)
		var integration *IntegrationError
		if !errors.Is(err, mappingErr) || !errors.As(err, &integration) || integration.Class != IntegrationPartialFailure || len(result.Actions) != 3 || result.Event == nil || events.calls != 1 || result.Mappings.Revision != "base" {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	}
}
