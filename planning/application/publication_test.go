package application

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/JimmyMcBride/brain/planning"
)

type publicationMemory struct {
	snapshot PublicationSnapshot
	reads    int
}

func (m *publicationMemory) Inspect(context.Context, PublicationInspectRequest) (PublicationSnapshot, error) {
	m.reads++
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
	input := PublicationPreviewInput{Target: target, Source: &ExternalReference{Provider: "test", Kind: "conversation", OpaqueID: "source", Revision: "first"}, Artifacts: []PublicationArtifact{
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
	if len(plan.Actions) != 6 {
		t.Fatalf("actions=%+v", plan.Actions)
	}
	for i, id := range []planning.ArtifactID{"initiative", "a", "b"} {
		if plan.Actions[i].Artifact.Artifact.ID != id || plan.Actions[i].Action != MutationCreate {
			t.Fatal(plan.Actions[i])
		}
	}
	if plan.Actions[2].Artifact.Content != "Preserve B criteria" || !reflect.DeepEqual(plan.Source, input.Source) {
		t.Fatal("lost canonical content/provenance")
	}
	slices.Reverse(input.Artifacts)
	reordered, err := service.PreviewPublication(context.Background(), target, input, policy)
	if err != nil || !reflect.DeepEqual(plan, reordered) {
		t.Fatal("input ordering changed plan", err)
	}
	for _, action := range plan.Actions {
		if action.Artifact != nil {
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
	if updated.Actions[1].Action != MutationUpdate {
		t.Fatal(updated.Actions)
	}
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
