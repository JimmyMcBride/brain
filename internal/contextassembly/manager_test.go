package contextassembly

import (
	"bytes"
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"brain/internal/livecontext"
	"brain/internal/projectcontext"
	"brain/internal/search"
	"brain/internal/structure"
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
	if len(packet.Ambiguities) != 2 {
		t.Fatalf("expected explicit-task ambiguity in empty packet: %#v", packet.Ambiguities)
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
	if packet.Summary.GroupCounts.StructuralRepo != 0 {
		t.Fatalf("expected structural repo group to remain empty in summary: %#v", packet.Summary)
	}
}

func TestAssembleIncludesStructuralRepoCandidatesWithoutRegressingCoreGroups(t *testing.T) {
	project := t.TempDir()
	contextManager := projectcontext.New(t.TempDir())
	if _, err := contextManager.Install(context.Background(), projectcontext.Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	manager := New(contextManager)
	packet, err := manager.Assemble(Request{
		ProjectDir: project,
		Task:       "search config",
		TaskSource: "flag",
		Limit:      8,
		SearchResults: []search.Result{
			{NotePath: filepath.ToSlash(filepath.Join("docs", "search-overview.md")), NoteTitle: "Search Overview", Heading: "Summary", Snippet: "Search context overview.", Score: 0.86},
		},
		StructuralItems: []structure.Item{
			{
				Kind:     "boundary",
				Path:     "internal/search/",
				Label:    "internal/search",
				Role:     "library",
				Summary:  "Search package boundary.",
				Evidence: []string{"contains search implementation"},
			},
			{
				Kind:     "config_surface",
				Path:     "config/search.yaml",
				Label:    "search config",
				Role:     "config",
				Summary:  "Search tuning config surface.",
				Evidence: []string{"matched config path"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(packet.Selected.StructuralRepo) == 0 {
		t.Fatalf("expected structural repo group to be populated: %#v", packet.Selected)
	}
	if packet.Summary.GroupCounts.StructuralRepo == 0 {
		t.Fatalf("expected structural repo count in summary: %#v", packet.Summary)
	}
	if len(packet.Selected.DurableNotes) == 0 || len(packet.Selected.GeneratedContext) == 0 || len(packet.Selected.PolicyWorkflow) == 0 {
		t.Fatalf("expected existing first-wave groups to remain populated: %#v", packet.Selected)
	}
	first := packet.Selected.StructuralRepo[0]
	if first.Source == "" || first.Label == "" || first.Kind != "structural" || first.Excerpt == "" || first.Why == "" {
		t.Fatalf("expected structural packet item fields to be populated: %#v", first)
	}
}

func TestAssembleIncludesLiveWorkCandidatesWithoutRegressingCoreGroups(t *testing.T) {
	project := t.TempDir()
	contextManager := projectcontext.New(t.TempDir())
	if _, err := contextManager.Install(context.Background(), projectcontext.Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	manager := New(contextManager)
	packet, err := manager.Assemble(Request{
		ProjectDir: project,
		Task:       "search config",
		TaskSource: "session",
		Limit:      8,
		SearchResults: []search.Result{
			{NotePath: filepath.ToSlash(filepath.Join("docs", "search-overview.md")), NoteTitle: "Search Overview", Heading: "Summary", Snippet: "Search context overview.", Score: 0.86},
		},
		LivePacket: &livecontext.Packet{
			Worktree: livecontext.WorktreeInfo{
				ChangedFiles: []livecontext.ChangedFile{
					{Path: "internal/search/search.go", Status: "modified", Source: "worktree", Why: "present in current worktree changes"},
				},
				TouchedBoundaries: []livecontext.TouchedBoundary{
					{Path: "internal/search/", Label: "internal/search", Role: "library", Why: "contains changed files"},
				},
			},
			NearbyTests: []livecontext.NearbyTest{
				{Path: "internal/search/search_test.go", Relation: "same_dir", Why: "test surface near changed code"},
			},
			Verification: livecontext.Verification{
				Profiles: []livecontext.VerificationProfile{
					{Name: "build", Satisfied: false},
				},
			},
			PolicyHints: []livecontext.PolicyHint{
				{Source: ".brain/context/workflows.md", Label: "Verification workflow", Excerpt: "Run required verification commands through brain session run.", Why: "repo changes detected but required verification is still missing"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(packet.Selected.LiveWork) == 0 {
		t.Fatalf("expected live work group to be populated: %#v", packet.Selected)
	}
	if packet.Summary.GroupCounts.LiveWork == 0 {
		t.Fatalf("expected live work count in summary: %#v", packet.Summary)
	}
	if len(packet.Selected.DurableNotes) == 0 || len(packet.Selected.GeneratedContext) == 0 || len(packet.Selected.PolicyWorkflow) == 0 {
		t.Fatalf("expected existing first-wave groups to remain populated: %#v", packet.Selected)
	}
}

func TestAssembleComputesConfidenceFromCoverageAndAmbiguities(t *testing.T) {
	project := t.TempDir()
	contextManager := projectcontext.New(t.TempDir())
	if _, err := contextManager.Install(context.Background(), projectcontext.Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	manager := New(contextManager)
	packet, err := manager.Assemble(Request{
		ProjectDir:       project,
		Task:             "workflow",
		TaskSource:       "session",
		HasActiveSession: true,
		Limit:            7,
		SearchResults: []search.Result{
			{NotePath: "docs/workflow-guide.md", NoteTitle: "Workflow Guide", Heading: "Overview", Snippet: "Workflow guide.", Score: 0.80},
			{NotePath: "docs/workflow-checklist.md", NoteTitle: "Workflow Checklist", Heading: "Checklist", Snippet: "Workflow checklist.", Score: 0.70},
			{NotePath: "docs/workflow-ops.md", NoteTitle: "Workflow Ops", Heading: "", Snippet: "Workflow operations.", Score: 0.60},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if packet.Summary.Confidence != "high" {
		t.Fatalf("expected high confidence with three groups and no ambiguities: %#v", packet)
	}
}

func TestAssembleExplainAddsDiagnosticsAndOmittedNearby(t *testing.T) {
	project := t.TempDir()
	contextManager := projectcontext.New(t.TempDir())
	if _, err := contextManager.Install(context.Background(), projectcontext.Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	manager := New(contextManager)
	packet, err := manager.Assemble(Request{
		ProjectDir:       project,
		Task:             "workflow",
		TaskSource:       "session",
		HasActiveSession: true,
		Limit:            6,
		Explain:          true,
		SearchResults: []search.Result{
			{NotePath: "docs/workflow-overview.md", NoteTitle: "Workflow Overview", Heading: "Summary", Snippet: "Workflow details.", Score: 0.77},
			{NotePath: "docs/workflow-details.md", NoteTitle: "Workflow Details", Heading: "Checklist", Snippet: "Detailed workflow notes.", Score: 0.70},
			{NotePath: "docs/workflow-ops.md", NoteTitle: "Workflow Ops", Heading: "", Snippet: "Operational workflow notes.", Score: 0.65},
			{NotePath: "docs/workflow-extra.md", NoteTitle: "Workflow Extra", Heading: "", Snippet: "Nearby workflow notes.", Score: 0.63},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(packet.Selected.DurableNotes) == 0 || packet.Selected.DurableNotes[0].Diagnostics == nil {
		t.Fatalf("expected explain diagnostics on selected items: %#v", packet.Selected)
	}
	if packet.Selected.DurableNotes[0].SelectionMethod == "" || packet.Selected.DurableNotes[0].Rank == 0 {
		t.Fatalf("expected explain rank and selection method: %#v", packet.Selected.DurableNotes[0])
	}
	if totalItems(packet.OmittedNearby) == 0 {
		t.Fatalf("expected omitted nearby items in explain packet: %#v", packet.OmittedNearby)
	}

	var out bytes.Buffer
	if err := RenderHuman(&out, packet, true); err != nil {
		t.Fatal(err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, "## Why This Was Selected") || !strings.Contains(rendered, "## Omitted Nearby Context") || !strings.Contains(rendered, "## Missing Or Unused Source Groups") {
		t.Fatalf("expected explain sections in human output:\n%s", rendered)
	}
}
