package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JimmyMcBride/brain/planning"
)

func TestCreateBrainstormRequiresConfirmationPermissionAndEventSink(t *testing.T) {
	repository := newFakeRepository()
	service := New(repository, Options{
		ModuleID: "official.planning",
		Now:      func() time.Time { return time.Date(2026, 7, 29, 7, 0, 0, 0, time.UTC) },
	})
	ctx := context.Background()

	preview, err := service.PreviewBrainstorm(ctx, "Local Workflow")
	if err != nil {
		t.Fatal(err)
	}
	if preview.Action != MutationCreate || preview.Document.Path != ".plan/brainstorms/local-workflow.md" {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	if _, err := service.CreateBrainstorm(ctx, CreateBrainstormInput{Title: "Local Workflow"}, allowAuthorizer{}, &eventRecorder{}); !errors.Is(err, ErrConfirmationRequired) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
	denied := errors.New("denied")
	if _, err := service.CreateBrainstorm(ctx, CreateBrainstormInput{Title: "Local Workflow", Confirmed: true}, errorAuthorizer{err: denied}, &eventRecorder{}); !errors.Is(err, denied) {
		t.Fatalf("expected permission error, got %v", err)
	}
	if _, err := service.CreateBrainstorm(ctx, CreateBrainstormInput{Title: "Local Workflow", Confirmed: true}, allowAuthorizer{}, nil); !errors.Is(err, ErrEventSinkRequired) {
		t.Fatalf("expected event sink error, got %v", err)
	}
	if repository.creates != 0 {
		t.Fatalf("repository mutated before all gates: %d", repository.creates)
	}
}

func TestCreateBrainstormEmitsOnceAndRerunIsIdempotent(t *testing.T) {
	repository := newFakeRepository()
	now := time.Date(2026, 7, 29, 7, 0, 0, 0, time.UTC)
	service := New(repository, Options{ModuleID: "official.planning", Now: func() time.Time { return now }})
	events := &eventRecorder{}
	input := CreateBrainstormInput{Title: "Local Workflow", Confirmed: true}

	first, err := service.CreateBrainstorm(context.Background(), input, allowAuthorizer{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if first.Action != MutationCreate || first.Event == nil || first.Event.Name != EventBrainstormCreated {
		t.Fatalf("unexpected create result: %#v", first)
	}
	second, err := service.CreateBrainstorm(context.Background(), input, allowAuthorizer{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if second.Action != MutationUnchanged || second.Event != nil {
		t.Fatalf("unexpected idempotent result: %#v", second)
	}
	if repository.creates != 1 || len(events.events) != 1 {
		t.Fatalf("expected one write and event, got writes=%d events=%d", repository.creates, len(events.events))
	}
}

func TestServiceRefusesIncompatibleWorkspace(t *testing.T) {
	repository := newFakeRepository()
	repository.status = WorkspaceStatus{
		State:    WorkspaceFutureSchema,
		Writable: false,
		Message:  "future schema",
	}
	service := New(repository, Options{})
	if _, err := service.ListSpecs(context.Background()); !errors.Is(err, ErrWorkspaceNotReadable) {
		t.Fatalf("expected readable workspace error, got %v", err)
	}
	if _, err := service.PreviewBrainstorm(context.Background(), "No Write"); !errors.Is(err, ErrWorkspaceNotWritable) {
		t.Fatalf("expected writable workspace error, got %v", err)
	}
}

type fakeRepository struct {
	status      WorkspaceStatus
	brainstorms map[planning.ArtifactID]BrainstormDocument
	creates     int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		status: WorkspaceStatus{
			State:     WorkspaceCompatible,
			Ownership: planning.OwnershipLocal,
			Writable:  true,
			Message:   "compatible",
		},
		brainstorms: map[planning.ArtifactID]BrainstormDocument{},
	}
}

func (r *fakeRepository) Status(context.Context) (WorkspaceStatus, error) {
	return r.status, nil
}

func (r *fakeRepository) ListBrainstorms(context.Context) ([]BrainstormDocument, error) {
	out := make([]BrainstormDocument, 0, len(r.brainstorms))
	for _, document := range r.brainstorms {
		out = append(out, document)
	}
	return out, nil
}

func (r *fakeRepository) GetBrainstorm(_ context.Context, id planning.ArtifactID) (BrainstormDocument, error) {
	document, ok := r.brainstorms[id]
	if !ok {
		return BrainstormDocument{}, errors.New("not found")
	}
	return document, nil
}

func (r *fakeRepository) FindBrainstorm(_ context.Context, id planning.ArtifactID) (BrainstormDocument, bool, error) {
	document, ok := r.brainstorms[id]
	return document, ok, nil
}

func (r *fakeRepository) CreateBrainstorm(_ context.Context, artifact planning.Brainstorm, _ time.Time) (BrainstormDocument, MutationAction, error) {
	if document, exists := r.brainstorms[artifact.ID]; exists {
		return document, MutationUnchanged, nil
	}
	r.creates++
	document := BrainstormDocument{Artifact: artifact, Path: ".plan/brainstorms/" + string(artifact.ID) + ".md"}
	r.brainstorms[artifact.ID] = document
	return document, MutationCreate, nil
}

func (r *fakeRepository) ListSpecs(context.Context) ([]SpecDocument, error) {
	return nil, nil
}

func (r *fakeRepository) GetSpec(context.Context, planning.ArtifactID) (SpecDocument, error) {
	return SpecDocument{}, errors.New("not found")
}

type allowAuthorizer struct{}

func (allowAuthorizer) Require(context.Context, string) error { return nil }

type errorAuthorizer struct{ err error }

func (a errorAuthorizer) Require(context.Context, string) error { return a.err }

type eventRecorder struct{ events []Event }

func (r *eventRecorder) Publish(_ context.Context, event Event) error {
	r.events = append(r.events, event)
	return nil
}
