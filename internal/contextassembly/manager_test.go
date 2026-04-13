package contextassembly

import (
	"bytes"
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"brain/internal/projectcontext"
	"brain/internal/search"
)

func TestAssembleReturnsStableEmptyPacketShape(t *testing.T) {
	manager := New(nil)

	packet, err := manager.Assemble(Request{
		Task:       "tighten auth flow",
		TaskSource: "flag",
	})
	if err != nil {
		t.Fatal(err)
	}

	if packet.Task.Text != "tighten auth flow" || packet.Task.Source != "flag" {
		t.Fatalf("unexpected task payload: %#v", packet.Task)
	}
	if packet.Summary.Confidence != "low" || packet.Summary.SelectedCount != 0 {
		t.Fatalf("unexpected summary: %#v", packet.Summary)
	}
	if packet.Selected.DurableNotes == nil || packet.Selected.GeneratedContext == nil || packet.Selected.StructuralRepo == nil || packet.Selected.LiveWork == nil || packet.Selected.PolicyWorkflow == nil {
		t.Fatalf("expected empty selected groups to be initialized: %#v", packet.Selected)
	}
	if packet.OmittedNearby.DurableNotes == nil || packet.OmittedNearby.GeneratedContext == nil || packet.OmittedNearby.StructuralRepo == nil || packet.OmittedNearby.LiveWork == nil || packet.OmittedNearby.PolicyWorkflow == nil {
		t.Fatalf("expected empty omitted groups to be initialized: %#v", packet.OmittedNearby)
	}
}

func TestAssembleRequiresTask(t *testing.T) {
	manager := New(nil)
	if _, err := manager.Assemble(Request{}); err == nil {
		t.Fatal("expected task requirement error")
	}
}

func TestRenderHumanCompactPacket(t *testing.T) {
	packet := &Packet{
		Task: TaskInfo{Text: "tighten auth flow", Source: "flag"},
		Summary: Summary{
			Confidence:    "low",
			SelectedCount: 0,
		},
		Selected:      newGroupedItems(),
		Ambiguities:   []string{},
		OmittedNearby: newGroupedItems(),
	}

	var out bytes.Buffer
	if err := RenderHuman(&out, packet, false); err != nil {
		t.Fatal(err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, "## Task Context") || !strings.Contains(rendered, "## Selected Context") {
		t.Fatalf("expected compact sections in human output:\n%s", rendered)
	}
	if strings.Contains(rendered, "## Why This Was Selected") {
		t.Fatalf("did not expect explain sections in compact output:\n%s", rendered)
	}
}

func TestAssembleSelectsFirstWaveGroupsDeterministically(t *testing.T) {
	project := t.TempDir()
	contextManager := projectcontext.New(t.TempDir())
	if _, err := contextManager.Install(context.Background(), projectcontext.Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	manager := New(contextManager)
	results := []search.Result{
		{NotePath: "docs/auth-flow.md", NoteTitle: "Auth Flow", Heading: "Overview", Snippet: "Auth flow details.", Score: 0.91},
		{NotePath: "docs/auth-flow.md", NoteTitle: "Auth Flow", Heading: "Overview", Snippet: "Duplicate auth flow details.", Score: 0.90},
		{NotePath: "docs/token-refresh.md", NoteTitle: "Token Refresh", Heading: "Plan", Snippet: "Token refresh details.", Score: 0.80},
		{NotePath: ".brain/resources/changes/auth-rollout.md", NoteTitle: "Auth Rollout", Heading: "Status", Snippet: "Rollout notes.", Score: 0.70},
		{NotePath: "docs/extra-auth.md", NoteTitle: "Extra Auth", Heading: "", Snippet: "Extra auth context.", Score: 0.60},
		{NotePath: "AGENTS.md", NoteTitle: "Project Agent Contract", Heading: "Required Workflow", Snippet: "Workflow guidance.", Score: 0.95},
		{NotePath: ".brain/context/overview.md", NoteTitle: "Overview", Heading: "", Snippet: "Generated overview.", Score: 0.88},
	}

	packet, err := manager.Assemble(Request{
		ProjectDir:    project,
		Task:          "auth flow",
		TaskSource:    "flag",
		Limit:         8,
		SearchResults: results,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(packet.Selected.DurableNotes), 4; got != want {
		t.Fatalf("unexpected durable note count: got=%d want=%d", got, want)
	}
	if got, want := len(packet.Selected.GeneratedContext), 2; got != want {
		t.Fatalf("unexpected generated context count: got=%d want=%d", got, want)
	}
	if got, want := len(packet.Selected.PolicyWorkflow), 2; got != want {
		t.Fatalf("unexpected policy workflow count: got=%d want=%d", got, want)
	}
	if got := len(packet.Selected.StructuralRepo) + len(packet.Selected.LiveWork); got != 0 {
		t.Fatalf("expected reserved groups to remain empty: %#v", packet.Selected)
	}
	if packet.Summary.SelectedCount != 8 {
		t.Fatalf("expected selected count to match packet contents: %#v", packet.Summary)
	}

	second, err := manager.Assemble(Request{
		ProjectDir:    project,
		Task:          "auth flow",
		TaskSource:    "flag",
		Limit:         8,
		SearchResults: results,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(packet.Selected, second.Selected) {
		t.Fatalf("expected deterministic selected groups:\nfirst=%#v\nsecond=%#v", packet.Selected, second.Selected)
	}
}

func TestAssembleUsesSearchAndStaticSourcesWithoutLeakingFutureGroups(t *testing.T) {
	project := t.TempDir()
	contextManager := projectcontext.New(t.TempDir())
	if _, err := contextManager.Install(context.Background(), projectcontext.Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	manager := New(contextManager)
	packet, err := manager.Assemble(Request{
		ProjectDir: project,
		Task:       "workflow",
		TaskSource: "flag",
		Limit:      6,
		SearchResults: []search.Result{
			{NotePath: filepath.ToSlash(filepath.Join("docs", "workflow-overview.md")), NoteTitle: "Workflow Overview", Heading: "Summary", Snippet: "Workflow details.", Score: 0.77},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(packet.Selected.DurableNotes) == 0 || len(packet.Selected.GeneratedContext) == 0 || len(packet.Selected.PolicyWorkflow) == 0 {
		t.Fatalf("expected first-wave groups to be populated: %#v", packet.Selected)
	}
	if packet.Summary.GroupCounts.StructuralRepo != 0 || packet.Summary.GroupCounts.LiveWork != 0 {
		t.Fatalf("expected future groups to remain empty in summary: %#v", packet.Summary)
	}
}
