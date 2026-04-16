package taskcontext

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"brain/internal/livecontext"
	"brain/internal/projectcontext"
	"brain/internal/search"
	"brain/internal/structure"
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
			{NotePath: "docs/context-compiler.md", NoteTitle: "Context Compiler", Heading: "Packet", Snippet: "Compiler packet output for internal/taskcontext should stay small and deterministic around cmd/context.go.", NoteType: "doc", Score: 0.91},
			{NotePath: ".brain/resources/changes/packet-notes.md", NoteTitle: "Packet Notes", Heading: "Observations", Snippet: "Keep provenance visible for packet items touching internal/taskcontext and internal/livecontext.", NoteType: "brain", Score: 0.87},
			{NotePath: ".brain/context/overview.md", NoteTitle: "Overview", Heading: "", Snippet: "Generated overview that should stay out of durable note selection.", NoteType: "generated", Score: 0.89},
			{NotePath: "AGENTS.md", NoteTitle: "Project Agent Contract", Heading: "Required Workflow", Snippet: "Workflow contract.", NoteType: "contract", Score: 0.99},
		},
		LivePacket: &livecontext.Packet{
			Worktree: livecontext.WorktreeInfo{
				ChangedFiles: []livecontext.ChangedFile{
					{Path: "cmd/context.go", Status: "modified", Source: "worktree", Why: "current worktree change for the compile command"},
					{Path: "internal/taskcontext/manager.go", Status: "modified", Source: "worktree", Why: "boundary-aware compiler selection is under active edit"},
				},
				TouchedBoundaries: []livecontext.TouchedBoundary{
					{
						Path:               "internal/taskcontext/",
						Label:              "internal/taskcontext",
						Role:               "library",
						Why:                "changed files map to the compiler package",
						AdjacentBoundaries: []string{"internal/livecontext"},
						Responsibilities:   []string{"Compile summary-first context packets"},
					},
				},
			},
			NearbyTests: []livecontext.NearbyTest{
				{Path: "internal/taskcontext/manager_test.go", Relation: "same_dir", Why: "test surface adjacent to compiler code"},
			},
			Verification: livecontext.Verification{
				Recipes: []livecontext.VerificationRecipe{
					{Label: "tests", Command: "go test ./internal/taskcontext", Source: "session_history", Strength: "suggested", Reason: "recent successful session command scoped to the touched boundary"},
					{Label: "tests", Command: "go test ./...", Source: ".brain/policy.yaml", Strength: "strong", Reason: "required by verification profile \"tests\""},
					{Label: "build", Command: "go build ./...", Source: ".brain/policy.yaml", Strength: "strong", Reason: "required by verification profile \"build\""},
				},
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
		BoundaryGraph: &structure.BoundaryGraph{
			Boundaries: []structure.BoundaryRecord{
				{
					ID:                 "internal/taskcontext",
					Label:              "internal/taskcontext",
					Role:               "library",
					RootPath:           "internal/taskcontext/",
					Files:              []string{"internal/taskcontext/manager.go", "internal/taskcontext/manager_test.go"},
					OwnedTests:         []string{"internal/taskcontext/manager_test.go"},
					AdjacentBoundaries: []string{"internal/livecontext"},
					Responsibilities:   []string{"Compile summary-first context packets"},
				},
				{
					ID:               "internal/livecontext",
					Label:            "internal/livecontext",
					Role:             "library",
					RootPath:         "internal/livecontext/",
					Files:            []string{"internal/livecontext/manager.go", "internal/livecontext/manager_test.go"},
					OwnedTests:       []string{"internal/livecontext/manager_test.go"},
					Responsibilities: []string{"Collect live work signals"},
				},
				{
					ID:               "cmd",
					Label:            "cmd",
					Role:             "application",
					RootPath:         "cmd/",
					Files:            []string{"cmd/context.go"},
					Responsibilities: []string{"CLI entrypoints"},
				},
			},
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
	if packet.Budget.Target == 0 || packet.Budget.Used == 0 {
		t.Fatalf("expected budget diagnostics on compiled packet: %#v", packet.Budget)
	}
	if packet.Budget.Used > packet.Budget.Target {
		t.Fatalf("expected default compile to stay within target budget: %#v", packet.Budget)
	}
	if len(packet.WorkingSet.Files) < 1 || len(packet.WorkingSet.Boundaries) != 1 || len(packet.WorkingSet.Tests) != 1 {
		t.Fatalf("expected first-wave live-work groups to be populated: %#v", packet.WorkingSet)
	}
	if got := len(packet.WorkingSet.Notes); got < 2 {
		t.Fatalf("expected boundary-aware note selection plus lexical fallback to keep at least two notes: got=%d packet=%#v", got, packet.WorkingSet.Notes)
	}
	for _, item := range packet.BaseContract {
		if item.EstimatedTokens == 0 {
			t.Fatalf("expected estimated token cost on base item: %#v", item)
		}
	}
	hasBoundaryLinkedNote := false
	for _, item := range packet.WorkingSet.Notes {
		if item.EstimatedTokens == 0 {
			t.Fatalf("expected estimated token cost on note: %#v", item)
		}
		if len(item.Boundaries) > 0 {
			hasBoundaryLinkedNote = true
			break
		}
	}
	if !hasBoundaryLinkedNote {
		t.Fatalf("expected boundary-linked note selection: %#v", packet.WorkingSet.Notes)
	}
	if len(packet.Verification) != 4 {
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
		Budget: projectcontext.PacketBudget{
			Preset:              "default",
			Target:              900,
			Used:                640,
			Remaining:           260,
			ReserveBaseContract: 220,
			ReserveVerification: 80,
			ReserveDiagnostics:  66,
		},
		BaseContract: []projectcontext.CompiledItem{
			{
				ContextItem: projectcontext.ContextItem{
					ID:              "base_boot_summary",
					Title:           "Boot Summary",
					Summary:         "Use Brain-managed repo context before substantial work.",
					Anchor:          projectcontext.ContextAnchor{Path: "AGENTS.md", Section: "Project Agent Contract"},
					EstimatedTokens: 24,
				},
				Reason: "always included as part of the base contract",
			},
		},
		WorkingSet: projectcontext.CompiledWorkingSet{
			Files: []projectcontext.CompiledFile{
				{Path: "cmd/context.go", Status: "modified", Source: "worktree", Reason: "current worktree change", EstimatedTokens: 18},
			},
			Boundaries: []projectcontext.CompiledBoundary{
				{
					Path:               "internal/taskcontext/",
					Label:              "internal/taskcontext",
					Role:               "library",
					Reason:             "changed files map to the compiler package",
					AdjacentBoundaries: []string{"internal/livecontext"},
					Responsibilities:   []string{"Compile summary-first context packets"},
					EstimatedTokens:    28,
				},
			},
			Tests: []projectcontext.CompiledTest{
				{Path: "internal/taskcontext/manager_test.go", Relation: "same_dir", Reason: "test surface adjacent to compiler code", EstimatedTokens: 16},
			},
		},
		Verification: []projectcontext.VerificationHint{
			{ID: "recipe:tests", Label: "tests", Command: "go test ./...", Summary: "required by verification profile \"tests\"", Source: ".brain/policy.yaml", Strength: "strong", Reason: "required by verification profile \"tests\"", EstimatedTokens: 20},
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
	for _, section := range []string{"## Compiled Context Packet", "## Budget", "## Base Contract", "## Working Set", "## Verification Hints", "## Provenance"} {
		if !strings.Contains(rendered, section) {
			t.Fatalf("expected section %q in output:\n%s", section, rendered)
		}
	}
}

func TestCompileBlendsUtilitySignalsConservatively(t *testing.T) {
	project := t.TempDir()
	contextManager := projectcontext.New(t.TempDir())
	if _, err := contextManager.Install(context.Background(), projectcontext.Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	manager := New(contextManager)
	req := Request{
		ProjectDir: project,
		Task:       "compiler telemetry signal",
		TaskSource: "flag",
		SearchResults: []search.Result{
			{NotePath: ".brain/brainstorms/compiler-telemetry-signal.md", NoteTitle: "Compiler Telemetry Signal", Heading: "Notes", Snippet: "Compiler telemetry signal details for cmd/context.go and internal/taskcontext/manager.go.", NoteType: "brainstorm", Score: 0.82},
			{NotePath: "docs/compiler-boundaries.md", NoteTitle: "Compiler Boundaries", Heading: "Boundaries", Snippet: "Compiler boundary notes for internal/taskcontext/manager.go and cmd/context.go.", NoteType: "doc", Score: 0.93},
			{NotePath: ".brain/resources/references/irrelevant.md", NoteTitle: "Irrelevant", Heading: "Elsewhere", Snippet: "Historical note about packaging only.", NoteType: "resource", Score: 0.99},
		},
		LivePacket: &livecontext.Packet{
			Worktree: livecontext.WorktreeInfo{
				ChangedFiles: []livecontext.ChangedFile{
					{Path: "cmd/context.go", Status: "modified", Source: "worktree", Why: "compile command changed"},
					{Path: "internal/taskcontext/manager.go", Status: "modified", Source: "worktree", Why: "compiler selection changed"},
				},
				TouchedBoundaries: []livecontext.TouchedBoundary{
					{
						Path:               "internal/taskcontext/",
						Label:              "internal/taskcontext",
						Role:               "library",
						Why:                "changed files map to the compiler package",
						AdjacentBoundaries: []string{"cmd"},
						Responsibilities:   []string{"Compile summary-first context packets"},
					},
				},
			},
		},
		BoundaryGraph: &structure.BoundaryGraph{
			Boundaries: []structure.BoundaryRecord{
				{
					ID:                 "internal/taskcontext",
					Label:              "internal/taskcontext",
					Role:               "library",
					RootPath:           "internal/taskcontext/",
					Files:              []string{"internal/taskcontext/manager.go"},
					AdjacentBoundaries: []string{"cmd"},
					Responsibilities:   []string{"Compile summary-first context packets"},
				},
				{
					ID:               "cmd",
					Label:            "cmd",
					Role:             "application",
					RootPath:         "cmd/",
					Files:            []string{"cmd/context.go"},
					Responsibilities: []string{"CLI entrypoints"},
				},
			},
		},
		UtilitySignals: map[string]ItemUtilitySignal{
			"durable_note:" + shortHash(".brain/brainstorms/compiler-telemetry-signal.md#Notes"): {
				LikelyUtility:               "likely_signal",
				IncludeCount:                3,
				ExpandCount:                 2,
				SuccessfulVerificationCount: 1,
				DurableUpdateCount:          1,
			},
			"durable_note:" + shortHash(".brain/resources/references/irrelevant.md#Elsewhere"): {
				LikelyUtility:      "likely_signal",
				IncludeCount:       5,
				ExpandCount:        3,
				DurableUpdateCount: 1,
			},
		},
	}

	packet, err := manager.Compile(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(packet.WorkingSet.Notes) < 2 {
		t.Fatalf("expected multiple working-set notes: %#v", packet.WorkingSet.Notes)
	}
	if packet.WorkingSet.Notes[0].Anchor.Path != ".brain/brainstorms/compiler-telemetry-signal.md" {
		t.Fatalf("expected utility-supported boundary note to rank first: %#v", packet.WorkingSet.Notes)
	}
	if !strings.Contains(packet.WorkingSet.Notes[0].Reason, "boosted by local utility signal") {
		t.Fatalf("expected utility-aware reason on selected note: %#v", packet.WorkingSet.Notes[0])
	}
}

func TestCompileBudgetPresetsShrinkRepresentativePacket(t *testing.T) {
	project := t.TempDir()
	contextManager := projectcontext.New(t.TempDir())
	if _, err := contextManager.Install(context.Background(), projectcontext.Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	manager := New(contextManager)
	req := Request{
		ProjectDir: project,
		Task:       "tighten compiler packet budget behavior",
		TaskSource: "flag",
		SearchResults: []search.Result{
			{NotePath: "docs/context-compiler.md", NoteTitle: "Context Compiler", Heading: "Budget", Snippet: "Keep context compile output compact and deterministic while preserving provenance and verification hints for cmd/context.go internal/taskcontext/manager.go and internal/session/telemetry.go.", NoteType: "doc", Score: 0.98},
			{NotePath: "docs/packet-ux.md", NoteTitle: "Packet UX", Heading: "Diagnostics", Snippet: "Show packet budget diagnostics and omitted-candidate pressure clearly in human and JSON surfaces without hiding anchors or provenance.", NoteType: "doc", Score: 0.93},
			{NotePath: ".brain/resources/changes/compiler-budget.md", NoteTitle: "Compiler Budget", Heading: "Rollout", Snippet: "Budget-aware selection should keep the highest-value working set that fits and preserve mandatory sections under tight packet budgets.", NoteType: "brain", Score: 0.91},
			{NotePath: ".brain/brainstorms/token-packets.md", NoteTitle: "Token Packets", Heading: "Direction", Snippet: "Hard packet budgets come first, packet reuse second, capsules third. Avoid generic rule-pack systems.", NoteType: "brainstorm", Score: 0.89},
		},
		LivePacket: &livecontext.Packet{
			Worktree: livecontext.WorktreeInfo{
				ChangedFiles: []livecontext.ChangedFile{
					{Path: "cmd/context.go", Status: "modified", Source: "worktree", Why: "compile CLI surface changed"},
					{Path: "internal/taskcontext/manager.go", Status: "modified", Source: "worktree", Why: "working-set selection changed"},
					{Path: "internal/taskcontext/budget.go", Status: "added", Source: "worktree", Why: "budget helpers added"},
					{Path: "internal/session/telemetry.go", Status: "modified", Source: "worktree", Why: "explain telemetry changed"},
					{Path: "docs/usage.md", Status: "modified", Source: "worktree", Why: "usage guidance changed"},
				},
				TouchedBoundaries: []livecontext.TouchedBoundary{
					{
						Path:               "internal/taskcontext/",
						Label:              "internal/taskcontext",
						Role:               "library",
						Why:                "compiler package changed",
						AdjacentBoundaries: []string{"cmd", "internal/session"},
						Responsibilities:   []string{"Compile summary-first context packets", "Apply budget-aware working-set selection"},
					},
					{
						Path:               "internal/session/",
						Label:              "internal/session",
						Role:               "library",
						Why:                "packet explain telemetry changed",
						AdjacentBoundaries: []string{"internal/taskcontext"},
						Responsibilities:   []string{"Record packet telemetry", "Render packet explanations"},
					},
				},
			},
			NearbyTests: []livecontext.NearbyTest{
				{Path: "internal/taskcontext/manager_test.go", Relation: "same_dir", Why: "compiler tests cover packet selection"},
				{Path: "internal/session/manager_test.go", Relation: "adjacent_boundary", Why: "session tests cover packet explain output"},
				{Path: "cmd/cli_test.go", Relation: "entrypoint", Why: "CLI tests cover compile and explain surfaces"},
			},
			Verification: livecontext.Verification{
				Recipes: []livecontext.VerificationRecipe{
					{Label: "taskcontext", Command: "go test ./internal/taskcontext", Source: "session_history", Strength: "suggested", Reason: "recent focused taskcontext verification"},
					{Label: "session", Command: "go test ./internal/session", Source: "session_history", Strength: "suggested", Reason: "recent focused session verification"},
					{Label: "tests", Command: "go test ./...", Source: ".brain/policy.yaml", Strength: "strong", Reason: "required by verification profile \"tests\""},
					{Label: "build", Command: "go build ./...", Source: ".brain/policy.yaml", Strength: "strong", Reason: "required by verification profile \"build\""},
				},
			},
			PolicyHints: []livecontext.PolicyHint{
				{Source: ".brain/context/workflows.md", Label: "Verification workflow", Excerpt: "Run required verification commands through brain session run.", Why: "verification is still required for repo changes"},
			},
			Ambiguities: []string{"packet budget target may be tighter than the working set candidates"},
		},
		BoundaryGraph: &structure.BoundaryGraph{
			Boundaries: []structure.BoundaryRecord{
				{
					ID:                 "internal/taskcontext",
					Label:              "internal/taskcontext",
					Role:               "library",
					RootPath:           "internal/taskcontext/",
					Files:              []string{"internal/taskcontext/manager.go", "internal/taskcontext/budget.go"},
					OwnedTests:         []string{"internal/taskcontext/manager_test.go"},
					AdjacentBoundaries: []string{"cmd", "internal/session"},
					Responsibilities:   []string{"Compile summary-first context packets", "Apply budget-aware working-set selection"},
				},
				{
					ID:                 "internal/session",
					Label:              "internal/session",
					Role:               "library",
					RootPath:           "internal/session/",
					Files:              []string{"internal/session/telemetry.go"},
					OwnedTests:         []string{"internal/session/manager_test.go"},
					AdjacentBoundaries: []string{"internal/taskcontext"},
					Responsibilities:   []string{"Record packet telemetry", "Render packet explanations"},
				},
				{
					ID:               "cmd",
					Label:            "cmd",
					Role:             "application",
					RootPath:         "cmd/",
					Files:            []string{"cmd/context.go"},
					OwnedTests:       []string{"cmd/cli_test.go"},
					Responsibilities: []string{"CLI entrypoints"},
				},
			},
		},
	}

	largePacket, err := manager.Compile(Request{
		ProjectDir:     req.ProjectDir,
		Task:           req.Task,
		TaskSource:     req.TaskSource,
		Budget:         "large",
		SearchResults:  req.SearchResults,
		LivePacket:     req.LivePacket,
		BoundaryGraph:  req.BoundaryGraph,
		UtilitySignals: req.UtilitySignals,
	})
	if err != nil {
		t.Fatal(err)
	}
	smallPacket, err := manager.Compile(Request{
		ProjectDir:     req.ProjectDir,
		Task:           req.Task,
		TaskSource:     req.TaskSource,
		Budget:         "small",
		SearchResults:  req.SearchResults,
		LivePacket:     req.LivePacket,
		BoundaryGraph:  req.BoundaryGraph,
		UtilitySignals: req.UtilitySignals,
	})
	if err != nil {
		t.Fatal(err)
	}
	if workingSetItemCount(smallPacket.WorkingSet) >= workingSetItemCount(largePacket.WorkingSet) {
		t.Fatalf("expected small preset to emit a leaner working set: small=%d large=%d", workingSetItemCount(smallPacket.WorkingSet), workingSetItemCount(largePacket.WorkingSet))
	}
	if smallPacket.Budget.Used > smallPacket.Budget.Target || largePacket.Budget.Used > largePacket.Budget.Target {
		t.Fatalf("expected preset packets to stay within target budgets: small=%#v large=%#v", smallPacket.Budget, largePacket.Budget)
	}
	if smallPacket.Budget.OmittedDueToBudget == 0 {
		t.Fatalf("expected small preset to omit candidates due to budget: %#v", smallPacket.Budget)
	}
	if len(smallPacket.BaseContract) == 0 || len(smallPacket.Verification) == 0 {
		t.Fatalf("expected mandatory sections to survive tight budget: %#v", smallPacket)
	}
}

func workingSetItemCount(set projectcontext.CompiledWorkingSet) int {
	return len(set.Boundaries) + len(set.Files) + len(set.Tests) + len(set.Notes)
}

func TestBuildFingerprintInputsStaysStableAndDetectsRelevantChanges(t *testing.T) {
	project := t.TempDir()
	contextManager := projectcontext.New(t.TempDir())
	if _, err := contextManager.Install(context.Background(), projectcontext.Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}
	manager := New(contextManager)

	req := Request{
		ProjectDir: project,
		Task:       "tighten session packet reuse",
		TaskSource: "flag",
		SearchResults: []search.Result{
			{NotePath: "docs/context-compiler.md", NoteTitle: "Context Compiler", Heading: "Reuse", ModifiedAt: "2026-04-15T00:00:00Z", Snippet: "Reuse packets conservatively.", NoteType: "doc", Score: 0.91},
		},
		LivePacket: &livecontext.Packet{
			Worktree: livecontext.WorktreeInfo{
				ChangedFiles: []livecontext.ChangedFile{
					{Path: "cmd/context.go", Status: "modified", Source: "worktree", Why: "context compile CLI changed"},
				},
				TouchedBoundaries: []livecontext.TouchedBoundary{
					{Path: "cmd/", Label: "cmd", Role: "application", Why: "CLI entrypoint changed"},
				},
			},
			Verification: livecontext.Verification{
				Recipes: []livecontext.VerificationRecipe{
					{Label: "tests", Command: "go test ./...", Source: ".brain/policy.yaml", Strength: "strong", Reason: "required by verification profile \"tests\""},
				},
			},
			PolicyHints: []livecontext.PolicyHint{
				{Source: ".brain/context/workflows.md", Label: "Verification workflow", Excerpt: "Run required verification commands through brain session run.", Why: "verification is still required"},
			},
		},
	}

	base, err := manager.BuildFingerprintInputs(req)
	if err != nil {
		t.Fatal(err)
	}
	again, err := manager.BuildFingerprintInputs(req)
	if err != nil {
		t.Fatal(err)
	}
	if base.Hash() == "" || base.Hash() != again.Hash() {
		t.Fatalf("expected stable fingerprint hash: base=%#v again=%#v", base, again)
	}

	changedFilesReq := req
	changedFilesReq.LivePacket = cloneLivePacket(req.LivePacket)
	changedFilesReq.LivePacket.Worktree.ChangedFiles = append(changedFilesReq.LivePacket.Worktree.ChangedFiles, livecontext.ChangedFile{Path: "main.go", Status: "modified", Source: "worktree", Why: "main entrypoint changed"})
	changedFiles, err := manager.BuildFingerprintInputs(changedFilesReq)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(changedFiles.InvalidationReasons(base), "changed files changed") {
		t.Fatalf("expected changed-files invalidation, got %v", changedFiles.InvalidationReasons(base))
	}

	boundaryReq := req
	boundaryReq.LivePacket = cloneLivePacket(req.LivePacket)
	boundaryReq.LivePacket.Worktree.TouchedBoundaries = append(boundaryReq.LivePacket.Worktree.TouchedBoundaries, livecontext.TouchedBoundary{Path: "internal/session/", Label: "internal/session", Role: "library", Why: "session boundary changed"})
	changedBoundaries, err := manager.BuildFingerprintInputs(boundaryReq)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(changedBoundaries.InvalidationReasons(base), "touched boundaries changed") {
		t.Fatalf("expected touched-boundary invalidation, got %v", changedBoundaries.InvalidationReasons(base))
	}

	if err := os.WriteFile(filepath.Join(project, ".brain", "context", "current-state.md"), []byte("# Current State\n\n## Repository\n\nUpdated source summary state.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changedSource, err := manager.BuildFingerprintInputs(req)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(changedSource.InvalidationReasons(base), "source summary state changed") {
		t.Fatalf("expected source-summary invalidation, got %v", changedSource.InvalidationReasons(base))
	}

	verificationReq := req
	verificationReq.LivePacket = cloneLivePacket(req.LivePacket)
	verificationReq.LivePacket.Verification.Recipes = append(verificationReq.LivePacket.Verification.Recipes, livecontext.VerificationRecipe{Label: "build", Command: "go build ./...", Source: ".brain/policy.yaml", Strength: "strong", Reason: "required by verification profile \"build\""})
	changedVerification, err := manager.BuildFingerprintInputs(verificationReq)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(changedVerification.InvalidationReasons(changedSource), "verification requirements changed") {
		t.Fatalf("expected verification invalidation, got %v", changedVerification.InvalidationReasons(changedSource))
	}
}

func TestPacketDiffReportsChangedSectionsAndItemIDs(t *testing.T) {
	previous := &projectcontext.CompiledPacket{
		Task:   projectcontext.CompiledTask{Text: "reuse", Summary: "reuse", Source: "session"},
		Budget: projectcontext.PacketBudget{Preset: "default", Target: 900, Used: 600},
		BaseContract: []projectcontext.CompiledItem{
			{ContextItem: projectcontext.ContextItem{ID: "base_boot_summary", Title: "Boot", Summary: "Boot", Anchor: projectcontext.ContextAnchor{Path: "AGENTS.md"}}, Reason: "always"},
		},
		WorkingSet: projectcontext.CompiledWorkingSet{
			Files: []projectcontext.CompiledFile{{Path: "cmd/context.go", Status: "modified", Source: "worktree", Reason: "changed"}},
			Notes: []projectcontext.CompiledItem{
				{ContextItem: projectcontext.ContextItem{ID: "note:abc", Title: "Note", Summary: "Before", Anchor: projectcontext.ContextAnchor{Path: "docs/one.md"}}, Reason: "ranked"},
			},
		},
		Verification: []projectcontext.VerificationHint{
			{ID: "recipe:tests", Label: "tests", Command: "go test ./...", Summary: "run tests", Source: ".brain/policy.yaml", Reason: "required"},
		},
		Provenance: []projectcontext.PacketProvenance{
			{ItemID: "note:abc", Section: "working_set.notes", Anchor: projectcontext.ContextAnchor{Path: "docs/one.md"}, Reason: "ranked"},
		},
	}
	current := &projectcontext.CompiledPacket{
		Task:   projectcontext.CompiledTask{Text: "reuse", Summary: "reuse", Source: "session"},
		Budget: projectcontext.PacketBudget{Preset: "default", Target: 900, Used: 650},
		BaseContract: []projectcontext.CompiledItem{
			{ContextItem: projectcontext.ContextItem{ID: "base_boot_summary", Title: "Boot", Summary: "Boot", Anchor: projectcontext.ContextAnchor{Path: "AGENTS.md"}}, Reason: "always"},
		},
		WorkingSet: projectcontext.CompiledWorkingSet{
			Files: []projectcontext.CompiledFile{{Path: "cmd/context.go", Status: "modified", Source: "worktree", Reason: "changed again"}},
			Notes: []projectcontext.CompiledItem{
				{ContextItem: projectcontext.ContextItem{ID: "note:abc", Title: "Note", Summary: "After", Anchor: projectcontext.ContextAnchor{Path: "docs/one.md"}}, Reason: "ranked"},
			},
		},
		Verification: []projectcontext.VerificationHint{
			{ID: "recipe:tests", Label: "tests", Command: "go test ./...", Summary: "run tests", Source: ".brain/policy.yaml", Reason: "required"},
			{ID: "recipe:build", Label: "build", Command: "go build ./...", Summary: "run build", Source: ".brain/policy.yaml", Reason: "required"},
		},
		Provenance: []projectcontext.PacketProvenance{
			{ItemID: "note:abc", Section: "working_set.notes", Anchor: projectcontext.ContextAnchor{Path: "docs/one.md"}, Reason: "ranked"},
		},
	}

	sections, itemIDs := PacketDiff(previous, current)
	for _, want := range []string{"budget", "verification", "working_set.files", "working_set.notes"} {
		if !containsString(sections, want) {
			t.Fatalf("expected changed section %q, got %v", want, sections)
		}
	}
	for _, want := range []string{"file:cmd/context.go", "note:abc", "recipe:build"} {
		if !containsString(itemIDs, want) {
			t.Fatalf("expected changed item id %q, got %v", want, itemIDs)
		}
	}
}

func TestRenderCompileResponseHumanKeepsCompactReuseOutputLean(t *testing.T) {
	packet := &projectcontext.CompiledPacket{
		Task:   projectcontext.CompiledTask{Text: "reuse", Summary: "reuse", Source: "session"},
		Budget: projectcontext.PacketBudget{Preset: "default", Target: 900, Used: 600, Remaining: 300},
		BaseContract: []projectcontext.CompiledItem{
			{ContextItem: projectcontext.ContextItem{ID: "base_boot_summary", Title: "Boot", Summary: "Boot", Anchor: projectcontext.ContextAnchor{Path: "AGENTS.md"}}, Reason: "always"},
		},
		WorkingSet: projectcontext.CompiledWorkingSet{
			Files: []projectcontext.CompiledFile{{Path: "cmd/context.go", Status: "modified", Source: "worktree", Reason: "changed"}},
		},
	}
	response := projectcontext.NewCompileResponse(packet, projectcontext.PacketCacheMetadata{
		CacheStatus:        projectcontext.PacketCacheStatusReused,
		Fingerprint:        "abc123",
		ReusedFrom:         packet.Hash(),
		FullPacketIncluded: false,
	})

	var out bytes.Buffer
	if err := RenderCompileResponseHuman(&out, response); err != nil {
		t.Fatal(err)
	}
	rendered := out.String()
	for _, snippet := range []string{"## Compiled Context Packet", "## Budget", "## Lineage", "re-emitted"} {
		if !strings.Contains(rendered, snippet) {
			t.Fatalf("expected compact reuse output to contain %q:\n%s", snippet, rendered)
		}
	}
	if strings.Contains(rendered, "## Base Contract") || strings.Contains(rendered, "## Working Set") {
		t.Fatalf("expected compact reuse output to stay lean:\n%s", rendered)
	}
}

func cloneLivePacket(packet *livecontext.Packet) *livecontext.Packet {
	if packet == nil {
		return nil
	}
	clone := *packet
	clone.Worktree.ChangedFiles = append([]livecontext.ChangedFile(nil), packet.Worktree.ChangedFiles...)
	clone.Worktree.TouchedBoundaries = append([]livecontext.TouchedBoundary(nil), packet.Worktree.TouchedBoundaries...)
	clone.NearbyTests = append([]livecontext.NearbyTest(nil), packet.NearbyTests...)
	clone.Verification.Recipes = append([]livecontext.VerificationRecipe(nil), packet.Verification.Recipes...)
	clone.Verification.Profiles = append([]livecontext.VerificationProfile(nil), packet.Verification.Profiles...)
	clone.PolicyHints = append([]livecontext.PolicyHint(nil), packet.PolicyHints...)
	clone.Ambiguities = append([]string(nil), packet.Ambiguities...)
	return &clone
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
