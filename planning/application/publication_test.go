package application

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/planning"
)

type publicationMemory struct {
	snapshot PublicationSnapshot
	request  PublicationInspectRequest
	reads    int
}

func (m *publicationMemory) Inspect(_ context.Context, request PublicationInspectRequest) (PublicationSnapshot, error) {
	m.reads++
	m.request = request
	return m.snapshot, nil
}
func (m *publicationMemory) Apply(context.Context, PublicationPlan) (PublicationResult, error) {
	panic("preview must never apply")
}

func publicationFixture() (PublicationPreviewInput, *publicationMemory) {
	group := planning.ArtifactRef{Kind: planning.ArtifactInitiative, ID: "initiative"}
	a := planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: "a"}
	b := planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: "b"}
	target := ExternalReference{Provider: "test", Kind: "workspace", OpaqueID: "workspace"}
	input := PublicationPreviewInput{Target: target, Source: &ExternalReference{Provider: "test", Kind: "conversation", OpaqueID: "source", Revision: "first"}, Group: &PublicationGroup{Title: "Initiative"}, Artifacts: []PublicationArtifact{
		{Artifact: b, Title: "B", Content: "Preserve B criteria", Group: &group, Dependencies: []planning.ArtifactRef{a}},
		{Artifact: a, Title: "A", Content: "Preserve A criteria", Group: &group},
		{Artifact: group, Title: "Initiative"},
	}}
	return input, &publicationMemory{snapshot: PublicationSnapshot{SchemaVersion: 1, Target: target}}
}

func TestPublicationPreviewDeterministicOrderAndRerun(t *testing.T) {
	input, target := publicationFixture()
	service := New(nil, Options{})
	policy := &collaborationPolicy{}
	plan, err := service.PreviewPublication(context.Background(), target, input, policy)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 7 || plan.Actions[0].Kind != PublicationGroupAction || plan.Actions[0].Action != MutationCreate {
		t.Fatalf("actions=%+v", plan.Actions)
	}
	for i, id := range []planning.ArtifactID{"initiative", "a", "b"} {
		if plan.Actions[i+1].Artifact.Artifact.ID != id || plan.Actions[i+1].Action != MutationCreate {
			t.Fatal(plan.Actions[i+1])
		}
	}
	if plan.Actions[3].Artifact.Content != "Preserve B criteria" || !reflect.DeepEqual(plan.Source, input.Source) || !reflect.DeepEqual(target.request.Group, input.Group) {
		t.Fatal("lost canonical content/provenance")
	}
	slices.Reverse(input.Artifacts)
	reordered, err := service.PreviewPublication(context.Background(), target, input, policy)
	if err != nil || !reflect.DeepEqual(plan, reordered) {
		t.Fatal("input ordering changed plan", err)
	}
	for _, action := range plan.Actions {
		if action.Group != nil {
			group := *action.Group
			group.Reference = &ExternalReference{Provider: "test", Kind: "group", OpaqueID: "group", DisplayID: "1", URL: "https://example.test/groups/1"}
			target.snapshot.Group = &group
		} else if action.Artifact != nil {
			artifact := *action.Artifact
			artifact.Reference = &ExternalReference{Provider: "test", Kind: "work", OpaqueID: string(artifact.Artifact.ID)}
			target.snapshot.Artifacts = append(target.snapshot.Artifacts, artifact)
		} else {
			target.snapshot.Relationships = append(target.snapshot.Relationships, *action.Relationship)
		}
	}
	rerun, err := service.PreviewPublication(context.Background(), target, input, policy)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range rerun.Actions {
		if action.Artifact != nil && action.Action != MutationReuse {
			t.Fatal(action)
		}
		if action.Relationship != nil && action.Action != MutationUnchanged {
			t.Fatal(action)
		}
	}
	input.Artifacts = slices.Clone(target.snapshot.Artifacts)
	input.Group = target.snapshot.Group
	rerun, err = service.PreviewPublication(context.Background(), target, input, policy)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range rerun.Actions {
		if action.Action != MutationUnchanged {
			t.Fatal(action)
		}
	}
	input.Artifacts[1].Content = "Changed canonical criteria"
	updated, err := service.PreviewPublication(context.Background(), target, input, policy)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Actions[2].Action != MutationUpdate {
		t.Fatal(updated.Actions)
	}
}

func TestPublicationPreviewPlansSharedGroupAndWorkspaceDecision(t *testing.T) {
	initiative := planning.ArtifactRef{Kind: planning.ArtifactInitiative, ID: "initiative"}
	targetRef := ExternalReference{Provider: "test", Kind: "repository", OpaqueID: "repo"}
	input := PublicationPreviewInput{
		Target:    targetRef,
		Group:     &PublicationGroup{Title: "Release milestone"},
		Workspace: &PublicationWorkspaceDecision{Choice: PublicationWorkspaceCreate, Title: "Delivery board", Reason: "Five specs need coordinated execution."},
		Artifacts: []PublicationArtifact{{Artifact: initiative, Title: "Initiative"}},
	}
	for _, id := range []planning.ArtifactID{"e", "d", "c", "b", "a"} {
		group := initiative
		input.Artifacts = append(input.Artifacts, PublicationArtifact{Artifact: planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: id}, Title: strings.ToUpper(string(id)), Group: &group})
	}
	target := &publicationMemory{snapshot: PublicationSnapshot{SchemaVersion: IntegrationContractVersion, Target: targetRef}}
	service := New(nil, Options{})
	plan, err := service.PreviewPublication(context.Background(), target, input, &collaborationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Actions[0].Kind != PublicationGroupAction || plan.Actions[0].Action != MutationCreate || plan.Actions[len(plan.Actions)-1].Kind != PublicationWorkspaceAction || plan.Actions[len(plan.Actions)-1].Action != MutationCreate {
		t.Fatalf("coordination actions=%+v", plan.Actions)
	}
	if !reflect.DeepEqual(target.request.Group, input.Group) || !reflect.DeepEqual(target.request.Workspace, input.Workspace) {
		t.Fatalf("inspect request=%+v", target.request)
	}
	slices.Reverse(input.Artifacts)
	reordered, err := service.PreviewPublication(context.Background(), target, input, &collaborationPolicy{})
	if err != nil || !reflect.DeepEqual(plan, reordered) {
		t.Fatalf("ordering changed plan: %+v %v", reordered, err)
	}
	groupRef := ExternalReference{Provider: "test", Kind: "group", OpaqueID: "milestone-3", DisplayID: "3", URL: "https://example.test/milestones/3"}
	workspaceRef := ExternalReference{Provider: "test", Kind: "workspace", OpaqueID: "project-7", DisplayID: "7", URL: "https://example.test/projects/7"}
	target.snapshot.Group = &PublicationGroup{Title: input.Group.Title, Reference: &groupRef}
	target.snapshot.Workspace = &PublicationWorkspaceDecision{Choice: PublicationWorkspaceCreate, Title: input.Workspace.Title, Reason: input.Workspace.Reason, Reference: &workspaceRef}
	rerun, err := service.PreviewPublication(context.Background(), target, input, &collaborationPolicy{})
	if err != nil || rerun.Actions[0].Action != MutationReuse || rerun.Actions[len(rerun.Actions)-1].Action != MutationReuse {
		t.Fatalf("rerun=%+v err=%v", rerun, err)
	}
	input.Group.Reference = &groupRef
	input.Workspace = &PublicationWorkspaceDecision{Choice: PublicationWorkspaceConnect, Reason: "Use the reviewed board.", Reference: &workspaceRef}
	connected, err := service.PreviewPublication(context.Background(), target, input, &collaborationPolicy{})
	if err != nil || connected.Actions[0].Action != MutationUnchanged || connected.Actions[len(connected.Actions)-1].Action != MutationReuse {
		t.Fatalf("connected=%+v err=%v", connected, err)
	}
}

func TestPublicationPreviewRejectsInvalidCoordinationBeforeProviderRead(t *testing.T) {
	validRef := ExternalReference{Provider: "test", Kind: "workspace", OpaqueID: "one", DisplayID: "1", URL: "https://example.test/workspaces/1"}
	tests := []struct {
		name   string
		mutate func(*PublicationPreviewInput)
	}{
		{"missing large-workspace decision", func(i *PublicationPreviewInput) { i.Workspace = nil }},
		{"invalid choice", func(i *PublicationPreviewInput) { i.Workspace.Choice = "later" }},
		{"create with reference", func(i *PublicationPreviewInput) { i.Workspace.Reference = &validRef }},
		{"connect without reference", func(i *PublicationPreviewInput) {
			i.Workspace.Choice, i.Workspace.Reference = PublicationWorkspaceConnect, nil
		}},
		{"skip with title", func(i *PublicationPreviewInput) { i.Workspace.Choice = PublicationWorkspaceSkip }},
		{"whitespace group", func(i *PublicationPreviewInput) { i.Group.Title = " Milestone" }},
		{"foreign group", func(i *PublicationPreviewInput) {
			i.Group.Reference = &ExternalReference{Provider: "other", Kind: "group", OpaqueID: "one", DisplayID: "1", URL: "https://example.test/groups/1"}
		}},
		{"foreign workspace", func(i *PublicationPreviewInput) {
			i.Workspace = &PublicationWorkspaceDecision{Choice: PublicationWorkspaceConnect, Reference: &ExternalReference{Provider: "other", Kind: "workspace", OpaqueID: "one", DisplayID: "1", URL: "https://example.test/workspaces/1"}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, target := fiveSpecPublicationFixture()
			test.mutate(&input)
			if _, err := New(nil, Options{}).PreviewPublication(context.Background(), target, input, &collaborationPolicy{}); err == nil || target.reads != 0 {
				t.Fatalf("invalid coordination reached provider: reads=%d err=%v", target.reads, err)
			}
		})
	}
	input, target := publicationFixture()
	input.Workspace = &PublicationWorkspaceDecision{Choice: PublicationWorkspaceSkip}
	if _, err := New(nil, Options{}).PreviewPublication(context.Background(), target, input, &collaborationPolicy{}); err != nil {
		t.Fatal("small multi-spec publication should allow an explicit skip", err)
	}
}

func TestPublicationPreviewRejectsUnrequestedCoordination(t *testing.T) {
	input, target := publicationFixture()
	target.snapshot.Group = &PublicationGroup{Title: "Other", Reference: &ExternalReference{Provider: "test", Kind: "group", OpaqueID: "one", DisplayID: "1", URL: "https://example.test/groups/1"}}
	input.Group = nil
	if _, err := New(nil, Options{}).PreviewPublication(context.Background(), target, input, &collaborationPolicy{}); err == nil {
		t.Fatal("accepted unrequested provider group")
	}
	input, target = publicationFixture()
	target.snapshot.Workspace = &PublicationWorkspaceDecision{Choice: PublicationWorkspaceConnect, Reference: &ExternalReference{Provider: "test", Kind: "workspace", OpaqueID: "one", DisplayID: "1", URL: "https://example.test/workspaces/1"}}
	if _, err := New(nil, Options{}).PreviewPublication(context.Background(), target, input, &collaborationPolicy{}); err == nil {
		t.Fatal("accepted unrequested provider workspace")
	}
}

func fiveSpecPublicationFixture() (PublicationPreviewInput, *publicationMemory) {
	initiative := planning.ArtifactRef{Kind: planning.ArtifactInitiative, ID: "initiative"}
	targetRef := ExternalReference{Provider: "test", Kind: "repository", OpaqueID: "repo"}
	input := PublicationPreviewInput{Target: targetRef, Group: &PublicationGroup{Title: "Milestone"}, Workspace: &PublicationWorkspaceDecision{Choice: PublicationWorkspaceCreate, Title: "Workspace"}, Artifacts: []PublicationArtifact{{Artifact: initiative, Title: "Initiative"}}}
	for _, id := range []planning.ArtifactID{"a", "b", "c", "d", "e"} {
		group := initiative
		input.Artifacts = append(input.Artifacts, PublicationArtifact{Artifact: planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: id}, Title: strings.ToUpper(string(id)), Group: &group})
	}
	return input, &publicationMemory{snapshot: PublicationSnapshot{SchemaVersion: IntegrationContractVersion, Target: targetRef}}
}

func TestPublicationPreviewRejectsAmbiguityAndInvalidGraph(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*PublicationPreviewInput, *publicationMemory)
		beforeRead bool
	}{
		{"duplicate desired", func(i *PublicationPreviewInput, _ *publicationMemory) {
			i.Artifacts = append(i.Artifacts, i.Artifacts[0])
		}, true},
		{"cycle", func(i *PublicationPreviewInput, _ *publicationMemory) {
			i.Artifacts[1].Dependencies = []planning.ArtifactRef{i.Artifacts[0].Artifact}
		}, true},
		{"missing dependency", func(i *PublicationPreviewInput, _ *publicationMemory) {
			i.Artifacts[0].Dependencies = []planning.ArtifactRef{{Kind: planning.ArtifactSpec, ID: "missing"}}
		}, true},
		{"known mapping missing", func(i *PublicationPreviewInput, _ *publicationMemory) {
			i.Artifacts[0].Reference = &ExternalReference{Provider: "test", Kind: "work", OpaqueID: "known"}
		}, false},
		{"duplicate candidates", func(i *PublicationPreviewInput, m *publicationMemory) {
			a := i.Artifacts[0]
			a.Reference = &ExternalReference{Provider: "test", Kind: "work", OpaqueID: "known"}
			m.snapshot.Artifacts = []PublicationArtifact{a, a}
		}, false},
		{"target mismatch", func(_ *PublicationPreviewInput, m *publicationMemory) { m.snapshot.Target.OpaqueID = "other" }, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, target := publicationFixture()
			test.mutate(&input, target)
			_, err := New(nil, Options{}).PreviewPublication(context.Background(), target, input, &collaborationPolicy{})
			if err == nil {
				t.Fatal("accepted invalid plan")
			}
			if test.beforeRead && target.reads != 0 {
				t.Fatal("invalid intent triggered provider work")
			}
		})
	}
}

func TestPublicationPreviewDoesNotGuessByTitleOrReadWhenDenied(t *testing.T) {
	input, target := publicationFixture()
	_, err := New(nil, Options{}).PreviewPublication(context.Background(), target, input, &collaborationPolicy{deny: PermissionRead})
	if err == nil || target.reads != 0 {
		t.Fatal("read permission bypass")
	}
	other := input.Artifacts[0]
	other.Artifact.ID = "unrelated"
	other.Reference = &ExternalReference{Provider: "test", Kind: "work", OpaqueID: "other"}
	target.snapshot.Artifacts = []PublicationArtifact{other}
	plan, err := New(nil, Options{}).PreviewPublication(context.Background(), target, input, &collaborationPolicy{})
	if err != nil || plan.Actions[2].Action != MutationCreate {
		t.Fatalf("title guessed: %+v %v", plan, err)
	}
	other.Artifact = input.Artifacts[1].Artifact
	target.snapshot.Artifacts = append(target.snapshot.Artifacts, other)
	_, err = New(nil, Options{}).PreviewPublication(context.Background(), target, input, &collaborationPolicy{})
	var integration *IntegrationError
	if !errors.As(err, &integration) || integration.Class != IntegrationAmbiguousIdentity {
		t.Fatal(err)
	}
}

func TestPublicationIdentityComponentsCannotCollide(t *testing.T) {
	input, target := publicationFixture()
	a, b := input.Artifacts[0], input.Artifacts[1]
	a.Reference = &ExternalReference{Provider: "test", Kind: "work\x00item", OpaqueID: "one"}
	b.Reference = &ExternalReference{Provider: "test", Kind: "work", OpaqueID: "item\x00one"}
	target.snapshot.Artifacts = []PublicationArtifact{a, b}
	plan, err := New(nil, Options{}).PreviewPublication(context.Background(), target, input, &collaborationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Actions[2].Artifact.Reference.OpaqueID != b.Reference.OpaqueID || plan.Actions[3].Artifact.Reference.OpaqueID != a.Reference.OpaqueID {
		t.Fatal("distinct identities were conflated")
	}
}
