package taskcontext

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"brain/internal/livecontext"
	"brain/internal/projectcontext"
	"brain/internal/search"
)

func TestCompileRequiresTask(t *testing.T) {
	manager := New(projectcontext.New(t.TempDir()))
	if _, err := manager.Compile(Request{}); err == nil {
		t.Fatal("expected task requirement error")
	}
}

func TestCompileBuildsDeterministicPacket(t *testing.T) {
	project := t.TempDir()
	contextManager := projectcontext.New(t.TempDir())
	if _, err := contextManager.Install(context.Background(), projectcontext.Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	manager := New(contextManager)
	req := Request{
		ProjectDir: project,
		Task:       "tighten context compile output",
		TaskSource: "flag",
		SearchResults: []search.Result{
			{NotePath: "docs/context-compiler.md", NoteTitle: "Context Compiler", Heading: "Packet", Snippet: "Compiler packet output should stay small and deterministic.", NoteType: "doc", Score: 0.91},
			{NotePath: ".brain/resources/changes/packet-notes.md", NoteTitle: "Packet Notes", Heading: "Observations", Snippet: "Keep provenance visible for packet items.", NoteType: "brain", Score: 0.87},
			{NotePath: ".brain/context/overview.md", NoteTitle: "Overview", Heading: "", Snippet: "Generated overview that should stay out of durable note selection.", NoteType: "generated", Score: 0.89},
			{NotePath: "AGENTS.md", NoteTitle: "Project Agent Contract", Heading: "Required Workflow", Snippet: "Workflow contract.", NoteType: "contract", Score: 0.99},
		},
		LivePacket: &livecontext.Packet{
			Worktree: livecontext.WorktreeInfo{
				ChangedFiles: []livecontext.ChangedFile{
					{Path: "cmd/context.go", Status: "modified", Source: "worktree", Why: "current worktree change for the compile command"},
				},
				TouchedBoundaries: []livecontext.TouchedBoundary{
					{Path: "internal/taskcontext/", Label: "internal/taskcontext", Role: "library", Why: "changed files map to the compiler package"},
				},
			},
			NearbyTests: []livecontext.NearbyTest{
				{Path: "internal/taskcontext/manager_test.go", Relation: "same_dir", Why: "test surface adjacent to compiler code"},
			},
			Verification: livecontext.Verification{
				Profiles: []livecontext.VerificationProfile{
					{Name: "tests", Satisfied: false},
					{Name: "build", Satisfied: true, MatchedCommand: "go build ./..."},
				},
			},
			PolicyHints: []livecontext.PolicyHint{
				{Source: ".brain/context/workflows.md", Label: "Verification workflow", Excerpt: "Run required verification commands through brain session run.", Why: "repo changes detected but verification is still required"},
			},
			Ambiguities: []string{"task spans more than one context surface"},
		},
	}

	packet, err := manager.Compile(req)
	if err != nil {
		t.Fatal(err)
	}
	if packet.Task.Text != req.Task || packet.Task.Source != "flag" {
		t.Fatalf("unexpected task payload: %#v", packet.Task)
	}
	if got, want := len(packet.BaseContract), 5; got != want {
		t.Fatalf("unexpected base contract count: got=%d want=%d", got, want)
	}
	if len(packet.WorkingSet.Files) != 1 || len(packet.WorkingSet.Boundaries) != 1 || len(packet.WorkingSet.Tests) != 1 {
		t.Fatalf("expected first-wave live-work groups to be populated: %#v", packet.WorkingSet)
	}
	if got, want := len(packet.WorkingSet.Notes), 2; got != want {
		t.Fatalf("expected durable note selection to exclude AGENTS.md and keep two notes: got=%d want=%d", got, want)
	}
	if len(packet.Verification) != 3 {
		t.Fatalf("expected verification hints from profiles and policy hints: %#v", packet.Verification)
	}
	if len(packet.Provenance) != len(packet.BaseContract)+len(packet.WorkingSet.Notes) {
		t.Fatalf("unexpected provenance shape: %#v", packet.Provenance)
	}

	second, err := manager.Compile(req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(packet, second) {
		t.Fatalf("expected deterministic packet compilation:\nfirst=%#v\nsecond=%#v", packet, second)
	}
	if packet.Hash() == "" {
		t.Fatal("expected packet hash")
	}
}

func TestRenderHumanIncludesCompilerSections(t *testing.T) {
	packet := &projectcontext.CompiledPacket{
		Task: projectcontext.CompiledTask{
			Text:    "tighten context compile output",
			Summary: "tighten context compile output",
			Source:  "flag",
		},
		BaseContract: []projectcontext.CompiledItem{
			{
				ContextItem: projectcontext.ContextItem{
					ID:      "base_boot_summary",
					Title:   "Boot Summary",
					Summary: "Use Brain-managed repo context before substantial work.",
					Anchor:  projectcontext.ContextAnchor{Path: "AGENTS.md", Section: "Project Agent Contract"},
				},
				Reason: "always included as part of the base contract",
			},
		},
		WorkingSet: projectcontext.CompiledWorkingSet{
			Files: []projectcontext.CompiledFile{
				{Path: "cmd/context.go", Status: "modified", Source: "worktree", Reason: "current worktree change"},
			},
			Boundaries: []projectcontext.CompiledBoundary{
				{Path: "internal/taskcontext/", Label: "internal/taskcontext", Role: "library", Reason: "changed files map to the compiler package"},
			},
			Tests: []projectcontext.CompiledTest{
				{Path: "internal/taskcontext/manager_test.go", Relation: "same_dir", Reason: "test surface adjacent to compiler code"},
			},
		},
		Verification: []projectcontext.VerificationHint{
			{ID: "profile:tests", Label: "tests", Summary: "Verification profile is not satisfied yet.", Source: ".brain/policy.yaml", Reason: "required verification profile is still missing"},
		},
		Provenance: []projectcontext.PacketProvenance{
			{ItemID: "base_boot_summary", Section: "base_contract", Anchor: projectcontext.ContextAnchor{Path: "AGENTS.md", Section: "Project Agent Contract"}, Reason: "always included as part of the base contract"},
		},
	}

	var out bytes.Buffer
	if err := RenderHuman(&out, packet); err != nil {
		t.Fatal(err)
	}
	rendered := out.String()
	for _, section := range []string{"## Compiled Context Packet", "## Base Contract", "## Working Set", "## Verification Hints", "## Provenance"} {
		if !strings.Contains(rendered, section) {
			t.Fatalf("expected section %q in output:\n%s", section, rendered)
		}
	}
}
