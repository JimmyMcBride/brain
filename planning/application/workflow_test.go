package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JimmyMcBride/brain/planning"
)

func TestBrainstormUpdateRequiresConfirmationAuthorizationAndAudit(t *testing.T) {
	repository := newFakeRepository()
	id := planning.ArtifactID("local-promotion")
	repository.brainstorms[id] = BrainstormDocument{
		Artifact: planning.Brainstorm{ID: id, Title: "Local Promotion"},
		Path:     ".plan/brainstorms/local-promotion.md",
		Body:     "# Brainstorm: Local Promotion\n\n## Ideas\n",
	}
	service := New(repository, Options{ModuleID: "test", Now: func() time.Time { return time.Unix(1, 0) }})
	input := BrainstormUpdateInput{ID: id, Section: "ideas", Body: "Keep compatibility stable."}
	preview, err := service.PreviewBrainstormUpdate(context.Background(), input)
	if err != nil || preview.Action != MutationUpdate {
		t.Fatalf("unexpected preview: %#v err=%v", preview, err)
	}
	if _, err := service.UpdateBrainstorm(context.Background(), input, allowAuthorizer{}, &eventRecorder{}); !errors.Is(err, ErrConfirmationRequired) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
	denied := errors.New("denied")
	input.Confirmed = true
	if _, err := service.UpdateBrainstorm(context.Background(), input, errorAuthorizer{err: denied}, &eventRecorder{}); !errors.Is(err, denied) {
		t.Fatalf("expected authorization error, got %v", err)
	}
	events := &eventRecorder{}
	result, err := service.UpdateBrainstorm(context.Background(), input, allowAuthorizer{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != MutationUpdate || result.Event == nil || result.Event.Name != EventBrainstormUpdated || len(events.events) != 1 {
		t.Fatalf("unexpected update: %#v events=%d", result, len(events.events))
	}
	rerun, err := service.UpdateBrainstorm(context.Background(), input, allowAuthorizer{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if rerun.Action != MutationUnchanged || rerun.Event != nil || len(events.events) != 1 {
		t.Fatalf("unexpected rerun: %#v events=%d", rerun, len(events.events))
	}
}

func TestGuidedSessionMutationsPreserveReviewFlow(t *testing.T) {
	repository := newFakeRepository()
	repository.guided = GuidedSessionState{
		SchemaVersion:   3,
		LastActiveChain: "brainstorm/first",
		Sessions: map[string]GuidedSessionRecord{
			"brainstorm/first":  {ChainID: "brainstorm/first", Brainstorm: "first", CurrentStage: "brainstorm", StageStatuses: map[string]string{"brainstorm": "in_progress"}},
			"brainstorm/second": {ChainID: "brainstorm/second", Brainstorm: "second", CurrentStage: "spec", StageStatuses: map[string]string{"brainstorm": "done", "spec": "done", "execution": "in_progress"}},
		},
	}
	service := New(repository, Options{ModuleID: "test", Now: func() time.Time { return time.Unix(2, 0) }})
	events := &eventRecorder{}
	switched, err := service.SwitchGuidedSession(context.Background(), GuidedSessionMutationInput{ChainID: "second", Confirmed: true}, allowAuthorizer{}, events)
	if err != nil || switched.Session.ChainID != "brainstorm/second" {
		t.Fatalf("unexpected switch: %#v err=%v", switched, err)
	}
	reopened, err := service.ReopenGuidedSession(context.Background(), GuidedSessionMutationInput{ChainID: "second", Stage: "brainstorm", Confirmed: true}, allowAuthorizer{}, events)
	if err != nil || len(reopened.Impacted) != 2 || reopened.Session.StageStatuses["spec"] != "needs_review" {
		t.Fatalf("unexpected reopen: %#v err=%v", reopened, err)
	}
	reopenRerun, err := service.ReopenGuidedSession(context.Background(), GuidedSessionMutationInput{ChainID: "second", Stage: "brainstorm", Confirmed: true}, allowAuthorizer{}, events)
	if err != nil || reopenRerun.Action != MutationUnchanged || reopenRerun.Event != nil || len(events.events) != 2 {
		t.Fatalf("unexpected reopen rerun: %#v events=%d err=%v", reopenRerun, len(events.events), err)
	}
	reviewed, err := service.ReviewGuidedSession(context.Background(), GuidedSessionMutationInput{ChainID: "second", Confirmed: true}, allowAuthorizer{}, events)
	if err != nil || len(reviewed.Impacted) != 2 || reviewed.Session.StageStatuses["execution"] != "reviewed" {
		t.Fatalf("unexpected review: %#v err=%v", reviewed, err)
	}
}

func TestLocalPromotionRepairPreviewsThenAppliesIdempotently(t *testing.T) {
	repository := newFakeRepository()
	id := planning.ArtifactID("local-promotion")
	repository.brainstorms[id] = matureBrainstorm(id)
	service := New(repository, Options{ModuleID: "test", Now: func() time.Time { return time.Unix(5, 0) }})
	input := LocalPromotionRepairInput{BrainstormID: id, Specs: []string{"Storage Contract", "CLI Contract"}}

	preview, err := service.PreviewLocalPromotionRepair(context.Background(), input)
	if err != nil || preview.Action != MutationUpdate || len(preview.Specs) != 2 {
		t.Fatalf("unexpected repair preview: %#v err=%v", preview, err)
	}
	if strings.Contains(repository.brainstorms[id].Body, "## Specs") {
		t.Fatal("repair preview mutated brainstorm")
	}
	events := &eventRecorder{}
	input.Confirmed = true
	result, err := service.RepairLocalPromotionSource(context.Background(), input, allowAuthorizer{}, events)
	if err != nil || result.Action != MutationUpdate || result.Event == nil || len(events.events) != 1 {
		t.Fatalf("unexpected repair: %#v events=%d err=%v", result, len(events.events), err)
	}
	rerun, err := service.RepairLocalPromotionSource(context.Background(), input, allowAuthorizer{}, events)
	if err != nil || rerun.Action != MutationUnchanged || rerun.Event != nil || len(events.events) != 1 {
		t.Fatalf("unexpected repair rerun: %#v events=%d err=%v", rerun, len(events.events), err)
	}
}

func TestLocalPromotionDraftTargetsSpecWithoutEpic(t *testing.T) {
	repository := newFakeRepository()
	id := planning.ArtifactID("local-promotion")
	repository.brainstorms[id] = BrainstormDocument{
		Artifact: planning.Brainstorm{ID: id, Title: "Local Promotion"},
		Path:     ".plan/brainstorms/local-promotion.md",
		Body: `# Brainstorm: Local Promotion

## Constraints

- Stay local.

## Refinement

### Problem

Promotion lacks a shared contract.

### User / Value

Agents get one canonical spec.

### Decision Snapshot

Promote directly.

## Challenge

### No-Gos

- No epic intermediate.
`,
	}
	service := New(repository, Options{ModuleID: "test", Now: func() time.Time { return time.Unix(3, 0) }})
	draft, err := service.PreviewLocalPromotion(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if draft.PromotionDecision != "single_spec" || len(draft.ProposedSpecs) != 1 || draft.ProposedSpecs[0].Slug != "local-promotion" {
		t.Fatalf("unexpected draft: %#v", draft)
	}
	if draft.ProposedSpecs[0].Kind != "spec" {
		t.Fatalf("promotion created non-spec intermediate: %#v", draft.ProposedSpecs[0])
	}
}

func TestLocalPromotionAppliesOnceAndLinksGuidedSession(t *testing.T) {
	repository := newFakeRepository()
	id := planning.ArtifactID("local-promotion")
	repository.brainstorms[id] = matureBrainstorm(id)
	repository.guided = GuidedSessionState{
		SchemaVersion:   3,
		LastActiveChain: "brainstorm/local-promotion",
		Sessions: map[string]GuidedSessionRecord{
			"brainstorm/local-promotion": {
				ChainID:       "brainstorm/local-promotion",
				Brainstorm:    "local-promotion",
				CurrentStage:  "brainstorm",
				StageStatuses: map[string]string{"brainstorm": "in_progress"},
			},
		},
	}
	service := New(repository, Options{ModuleID: "test", Now: func() time.Time { return time.Unix(4, 0) }})
	events := &eventRecorder{}

	if _, err := service.PromoteLocalBrainstorm(context.Background(), LocalPromotionInput{BrainstormID: id}, allowAuthorizer{}, events); !errors.Is(err, ErrConfirmationRequired) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
	result, err := service.PromoteLocalBrainstorm(context.Background(), LocalPromotionInput{BrainstormID: id, Confirmed: true}, allowAuthorizer{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != MutationCreate || len(result.Specs) != 1 || result.Specs[0].Artifact.ID != id {
		t.Fatalf("unexpected promotion: %#v", result)
	}
	if result.Event == nil || result.Event.Name != EventBrainstormPromoted || len(events.events) != 1 {
		t.Fatalf("unexpected promotion events: %#v events=%d", result.Event, len(events.events))
	}
	session := repository.guided.Sessions["brainstorm/local-promotion"]
	if session.Spec != "local-promotion" || session.CurrentStage != "spec" || session.StageStatuses["brainstorm"] != "done" {
		t.Fatalf("promotion did not advance guided session: %#v", session)
	}
	rerun, err := service.PromoteLocalBrainstorm(context.Background(), LocalPromotionInput{BrainstormID: id, Confirmed: true}, allowAuthorizer{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if rerun.Action != MutationUnchanged || rerun.Event != nil || len(events.events) != 1 {
		t.Fatalf("promotion rerun was not idempotent: %#v events=%d", rerun, len(events.events))
	}
}

func matureBrainstorm(id planning.ArtifactID) BrainstormDocument {
	return BrainstormDocument{
		Artifact: planning.Brainstorm{ID: id, Title: "Local Promotion"},
		Path:     ".plan/brainstorms/local-promotion.md",
		Body: `# Brainstorm: Local Promotion

## Constraints

- Stay local.

## Refinement

### Problem

Promotion lacks a shared contract.

### User / Value

Agents get one canonical spec.

### Decision Snapshot

Promote directly.

## Challenge

### No-Gos

- No epic intermediate.
`,
	}
}
