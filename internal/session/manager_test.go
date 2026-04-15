package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"brain/internal/history"
	"brain/internal/projectcontext"
)

func TestPathMatchesAny(t *testing.T) {
	if !pathMatchesAny(".brain/resources/changes/project-change.md", []string{".brain/resources/**/project*.md"}) {
		t.Fatal("expected glob to match capture path")
	}
	if pathMatchesAny(".brain/resources/elsewhere/note.md", []string{"docs/project/**"}) {
		t.Fatal("did not expect glob to match unrelated path")
	}
}

func TestCommandProfileSatisfied(t *testing.T) {
	profile := projectcontext.VerificationProfile{
		Name:     "tests",
		Commands: []string{"go test ./..."},
	}
	runs := []CommandRun{
		{Command: "go test ./...", ExitCode: 0},
	}
	if !commandProfileSatisfied(profile, runs) {
		t.Fatal("expected profile to be satisfied")
	}
	runs[0].ExitCode = 1
	if commandProfileSatisfied(profile, runs) {
		t.Fatal("expected failed command run not to satisfy profile")
	}
}

func TestSnapshotGitIgnoresVolatileBrainRuntimeFiles(t *testing.T) {
	project := makeSessionProject(t, sessionPolicyYAML(t, nil, false))
	mustInitGitRepo(t, project)

	for rel := range map[string]string{
		".brain/session.json":            "{}\n",
		".brain/sessions/ledger.json":    "{}\n",
		".brain/state/brain.sqlite3-wal": "",
		".brain/state/brain.sqlite3-shm": "",
		".brain/state/history.jsonl":     "",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(map[string]string{
			".brain/session.json":            "{}\n",
			".brain/sessions/ledger.json":    "{}\n",
			".brain/state/brain.sqlite3-wal": "",
			".brain/state/brain.sqlite3-shm": "",
			".brain/state/history.jsonl":     "",
		}[rel]), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	snapshot := snapshotGit(context.Background(), project)
	if !snapshot.Available {
		t.Fatal("expected git snapshot to be available")
	}
	if len(snapshot.Status) != 0 {
		t.Fatalf("expected volatile brain runtime files to be ignored, got %v", snapshot.Status)
	}
}

func TestSessionLockBusyTreatsPermissionOnExistingDirAsContention(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "session.json.lock")
	if err := os.Mkdir(lockPath, 0o700); err != nil {
		t.Fatalf("mkdir lock dir: %v", err)
	}
	if !sessionLockBusy(lockPath, os.ErrPermission) {
		t.Fatal("expected existing lock dir plus permission error to be treated as contention")
	}
}

func TestRunCommandConcurrentRecordsAll(t *testing.T) {
	project := makeSessionProject(t, sessionPolicyYAML(t, nil, false))
	manager := New(nil)
	if _, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "concurrency"}); err != nil {
		t.Fatalf("start session: %v", err)
	}

	const runs = 6
	var wg sync.WaitGroup
	errCh := make(chan error, runs)
	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			argv := helperCommand("sleep-ms", "75", fmt.Sprintf("run-%d", i))
			result, err := manager.RunCommand(context.Background(), RunRequest{
				ProjectDir:    project,
				Argv:          argv,
				CaptureOutput: true,
			}, nil, nil)
			if err != nil {
				errCh <- fmt.Errorf("run %d failed: %w", i, err)
				return
			}
			if !result.Recorded {
				errCh <- fmt.Errorf("run %d was not recorded", i)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}

	active, err := loadActiveSession(filepath.Join(project, ".brain", "session.json"))
	if err != nil {
		t.Fatalf("load active session: %v", err)
	}
	if got := len(active.CommandRuns); got != runs {
		t.Fatalf("expected %d command runs, got %d", runs, got)
	}
}

func TestValidateFinishSeesConcurrentRecordedCommands(t *testing.T) {
	testsCmd := helperCommand("sleep-ms", "75", "tests")
	buildCmd := helperCommand("sleep-ms", "75", "build")
	project := makeSessionProject(t, sessionPolicyYAML(t, []string{
		strings.Join(testsCmd, " "),
		strings.Join(buildCmd, " "),
	}, false))
	mustInitGitRepo(t, project)

	manager := New(nil)
	if _, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "verify"}); err != nil {
		t.Fatalf("start session: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "README.md"), []byte("# changed\n"), 0o644); err != nil {
		t.Fatalf("mark repo dirty: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for _, argv := range [][]string{testsCmd, buildCmd} {
		wg.Add(1)
		go func(argv []string) {
			defer wg.Done()
			result, err := manager.RunCommand(context.Background(), RunRequest{
				ProjectDir:    project,
				Argv:          argv,
				CaptureOutput: true,
			}, nil, nil)
			if err != nil {
				errCh <- err
				return
			}
			if !result.Recorded {
				errCh <- fmt.Errorf("command %q was not recorded", result.Command)
			}
		}(argv)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}

	result, err := manager.Validate(context.Background(), ValidateRequest{ProjectDir: project, Stage: "finish"})
	if err != nil {
		t.Fatalf("validate finish: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected finish validation ok, got obligations=%v remediation=%v missing=%v", result.Obligations, result.Remediation, result.MissingCommands)
	}
}

func TestRunCommandAfterAbortDoesNotRecreateActiveSession(t *testing.T) {
	project := makeSessionProject(t, sessionPolicyYAML(t, nil, false))
	manager := New(nil)
	if _, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "abort race"}); err != nil {
		t.Fatalf("start session: %v", err)
	}

	resultCh := make(chan *RunResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := manager.RunCommand(context.Background(), RunRequest{
			ProjectDir:    project,
			Argv:          helperCommand("sleep-ms", "200", "late"),
			CaptureOutput: true,
		}, nil, nil)
		resultCh <- result
		errCh <- err
	}()

	time.Sleep(50 * time.Millisecond)
	if _, err := manager.Abort(context.Background(), AbortRequest{ProjectDir: project, Reason: "test abort"}); err != nil {
		t.Fatalf("abort session: %v", err)
	}

	result := <-resultCh
	err := <-errCh
	if err == nil {
		t.Fatal("expected run command error after abort")
	}
	if result == nil {
		t.Fatal("expected run result even when recording fails")
	}
	if result.Recorded {
		t.Fatal("expected aborted command run to be unrecorded")
	}
	if _, statErr := os.Stat(filepath.Join(project, ".brain", "session.json")); !os.IsNotExist(statErr) {
		t.Fatalf("expected active session to stay removed, stat err=%v", statErr)
	}
}

func TestRecordCompiledPacketAppendsPacketMetadata(t *testing.T) {
	project := makeSessionProject(t, sessionPolicyYAML(t, nil, false))
	manager := New(nil)
	started, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "compile packet"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	packet := &projectcontext.CompiledPacket{
		Task: projectcontext.CompiledTask{
			Text:    "compile packet",
			Summary: "compile packet",
			Source:  "session",
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
			Boundaries: []projectcontext.CompiledBoundary{
				{Path: "internal/taskcontext/", Label: "internal/taskcontext", Role: "library", Reason: "task touches compiler code"},
			},
			Files: []projectcontext.CompiledFile{
				{Path: "cmd/context.go", Status: "modified", Source: "worktree", Reason: "compile command changed"},
			},
			Tests: []projectcontext.CompiledTest{
				{Path: "internal/taskcontext/manager_test.go", Relation: "same_dir", Reason: "adjacent test surface"},
			},
			Notes: []projectcontext.CompiledItem{
				{
					ContextItem: projectcontext.ContextItem{
						ID:      "note:abc123",
						Title:   "Compiler Notes",
						Summary: "Keep packet output compact.",
						Anchor:  projectcontext.ContextAnchor{Path: "docs/compiler.md", Section: "Notes"},
					},
					Reason: "ranked highly in local durable-note search for the task",
				},
			},
		},
		Verification: []projectcontext.VerificationHint{
			{ID: "profile:tests", Label: "tests", Summary: "Verification profile is not satisfied yet.", Source: ".brain/policy.yaml", Reason: "required verification profile is still missing"},
		},
	}

	if err := manager.RecordCompiledPacket(project, started.Session.ID, packet); err != nil {
		t.Fatalf("record compiled packet: %v", err)
	}
	if err := manager.RecordCompiledPacket(project, started.Session.ID, packet); err != nil {
		t.Fatalf("record second compiled packet: %v", err)
	}

	active, err := loadActiveSession(filepath.Join(project, ".brain", "session.json"))
	if err != nil {
		t.Fatalf("load active session: %v", err)
	}
	if got := len(active.PacketRecords); got != 2 {
		t.Fatalf("expected two packet records, got %d", got)
	}
	first := active.PacketRecords[0]
	if first.PacketHash == "" || first.TaskText != "compile packet" || first.TaskSource != "session" {
		t.Fatalf("unexpected packet record: %#v", first)
	}
	if !reflect.DeepEqual(first.IncludedItemIDs, []string{
		"base_boot_summary",
		"note:abc123",
		"boundary:internal/taskcontext/",
		"file:cmd/context.go",
		"test:internal/taskcontext/manager_test.go",
		"profile:tests",
	}) {
		t.Fatalf("unexpected included item ids: %#v", first)
	}
	if len(first.IncludedAnchors) != 6 || len(first.InclusionReasons) != 6 {
		t.Fatalf("expected anchor and reason metadata: %#v", first)
	}
	if got := len(active.TelemetryEvents); got != 2 {
		t.Fatalf("expected one compiled telemetry event per packet record, got %d", got)
	}
	if active.TelemetryEvents[0].Type != PacketTelemetryEventCompiled || active.TelemetryEvents[0].PacketHash == "" {
		t.Fatalf("unexpected telemetry event payload: %#v", active.TelemetryEvents)
	}
}

func TestPacketTelemetryLinksCompileExpandVerificationAndCloseout(t *testing.T) {
	verifyCmd := helperCommand("sleep-ms", "10", "verify")
	project := makeSessionProject(t, sessionPolicyYAML(t, []string{strings.Join(verifyCmd, " ")}, false))
	historyLog := history.New(filepath.Join(project, ".brain", "state", "history.log"))
	manager := New(historyLog)
	started, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "telemetry linkage"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	packet := makeCompiledPacket("telemetry linkage", "session")
	if err := manager.RecordCompiledPacket(project, started.Session.ID, packet); err != nil {
		t.Fatalf("record compiled packet: %v", err)
	}
	if err := manager.RecordPacketExpansion(project, "docs/compiler.md"); err != nil {
		t.Fatalf("record expansion: %v", err)
	}
	if _, err := manager.RunCommand(context.Background(), RunRequest{
		ProjectDir:    project,
		Argv:          verifyCmd,
		CaptureOutput: true,
	}, nil, nil); err != nil {
		t.Fatalf("run verification command: %v", err)
	}
	if err := historyLog.Append(history.Entry{
		Operation: "update",
		File:      "docs/compiler.md",
		Summary:   "updated compiler note",
	}); err != nil {
		t.Fatalf("append history entry: %v", err)
	}

	finish, err := manager.Finish(context.Background(), FinishRequest{
		ProjectDir: project,
		Summary:    "telemetry linkage complete",
	})
	if err != nil {
		t.Fatalf("finish session: %v", err)
	}
	if finish.Status != "finished" {
		t.Fatalf("expected finished session, got %#v", finish)
	}

	raw, err := os.ReadFile(filepath.FromSlash(finish.LedgerPath))
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	var ledger ActiveSession
	if err := jsonUnmarshal(raw, &ledger); err != nil {
		t.Fatalf("parse ledger: %v\n%s", err, raw)
	}
	if ledger.TelemetryVersion != packetTelemetryVersion {
		t.Fatalf("unexpected telemetry version: %#v", ledger)
	}
	if !hasTelemetryEvent(ledger.TelemetryEvents, PacketTelemetryEventCompiled, packet.Hash(), "", "", "") {
		t.Fatalf("expected compiled telemetry event: %#v", ledger.TelemetryEvents)
	}
	if !hasTelemetryEvent(ledger.TelemetryEvents, PacketTelemetryEventExpanded, packet.Hash(), "note:abc123", "docs/compiler.md", "") {
		t.Fatalf("expected expanded telemetry event: %#v", ledger.TelemetryEvents)
	}
	if !hasTelemetryEvent(ledger.TelemetryEvents, PacketTelemetryEventVerification, packet.Hash(), "", "", strings.Join(verifyCmd, " ")) {
		t.Fatalf("expected verification telemetry event: %#v", ledger.TelemetryEvents)
	}
	if !hasTelemetryEvent(ledger.TelemetryEvents, PacketTelemetryEventDurableUpdate, packet.Hash(), "", "docs/compiler.md", "") {
		t.Fatalf("expected durable update telemetry event: %#v", ledger.TelemetryEvents)
	}
	if !hasTelemetryEvent(ledger.TelemetryEvents, PacketTelemetryEventSessionClosed, packet.Hash(), "", "", "") {
		t.Fatalf("expected session closed telemetry event: %#v", ledger.TelemetryEvents)
	}
}

func TestPacketTelemetryRetentionIsBounded(t *testing.T) {
	project := makeSessionProject(t, sessionPolicyYAML(t, nil, false))
	manager := New(nil)
	started, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "telemetry retention"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	for i := 0; i < maxSessionPacketRecords+10; i++ {
		packet := makeCompiledPacket(fmt.Sprintf("packet-%d", i), "session")
		packet.Task.Text = fmt.Sprintf("packet-%d", i)
		packet.Task.Summary = fmt.Sprintf("packet-%d", i)
		if err := manager.RecordCompiledPacket(project, started.Session.ID, packet); err != nil {
			t.Fatalf("record compiled packet %d: %v", i, err)
		}
	}
	for i := 0; i < maxSessionTelemetryEvents+40; i++ {
		if err := manager.RecordPacketExpansion(project, "docs/compiler.md"); err != nil {
			t.Fatalf("record expansion %d: %v", i, err)
		}
	}

	active, err := loadActiveSession(filepath.Join(project, ".brain", "session.json"))
	if err != nil {
		t.Fatalf("load active session: %v", err)
	}
	if got := len(active.PacketRecords); got != maxSessionPacketRecords {
		t.Fatalf("expected bounded packet records, got %d", got)
	}
	if got := len(active.TelemetryEvents); got != maxSessionTelemetryEvents {
		t.Fatalf("expected bounded telemetry events, got %d", got)
	}
}

func TestExplainPacketUsesLatestMatchingPacketRecord(t *testing.T) {
	project := makeSessionProject(t, sessionPolicyYAML(t, nil, false))
	manager := New(nil)
	started, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "explain latest packet"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	packet := makeCompiledPacket("explain latest packet", "session")
	if err := manager.RecordCompiledPacket(project, started.Session.ID, packet); err != nil {
		t.Fatalf("record first compiled packet: %v", err)
	}
	active, err := loadActiveSession(filepath.Join(project, ".brain", "session.json"))
	if err != nil {
		t.Fatalf("load active session after first packet: %v", err)
	}
	firstCompiledAt := active.PacketRecords[len(active.PacketRecords)-1].CompiledAt
	time.Sleep(10 * time.Millisecond)

	if err := manager.RecordCompiledPacket(project, started.Session.ID, packet); err != nil {
		t.Fatalf("record second compiled packet: %v", err)
	}
	if err := manager.RecordPacketExpansion(project, "docs/compiler.md"); err != nil {
		t.Fatalf("record packet expansion: %v", err)
	}

	explanation, err := manager.ExplainPacket(PacketExplainRequest{ProjectDir: project, PacketHash: packet.Hash()})
	if err != nil {
		t.Fatalf("explain packet: %v", err)
	}
	if !explanation.Packet.CompiledAt.After(firstCompiledAt) {
		t.Fatalf("expected explain to pick the latest matching packet, got %#v", explanation.Packet)
	}
	foundExpanded := false
	for _, item := range explanation.IncludedItems {
		if item.ItemID == "note:abc123" && item.Expanded == 1 {
			foundExpanded = true
			break
		}
	}
	if !foundExpanded {
		t.Fatalf("expected latest packet explanation to include the recorded expansion: %#v", explanation)
	}
}

func TestContextStatsSurfaceConservativeUtilitySignals(t *testing.T) {
	project := makeSessionProject(t, sessionPolicyYAML(t, nil, false))
	manager := New(nil)
	started, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "stats utility"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	packet := makeCompiledPacket("stats utility", "session")
	if err := manager.RecordCompiledPacket(project, started.Session.ID, packet); err != nil {
		t.Fatalf("record first compiled packet: %v", err)
	}
	if err := manager.RecordPacketExpansion(project, "docs/compiler.md"); err != nil {
		t.Fatalf("record first expansion: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := manager.RecordCompiledPacket(project, started.Session.ID, packet); err != nil {
		t.Fatalf("record second compiled packet: %v", err)
	}
	if err := manager.RecordPacketExpansion(project, "docs/compiler.md"); err != nil {
		t.Fatalf("record second expansion: %v", err)
	}

	stats, err := manager.ContextStats(ContextStatsRequest{ProjectDir: project, Limit: 3})
	if err != nil {
		t.Fatalf("context stats: %v", err)
	}
	if len(stats.TopSignal) == 0 {
		t.Fatalf("expected at least one likely signal item: %#v", stats)
	}
	if stats.TopSignal[0].ItemID != "note:abc123" || stats.TopSignal[0].LikelyUtility != "likely_signal" {
		t.Fatalf("expected note:abc123 to rank as likely signal: %#v", stats.TopSignal)
	}
	if len(stats.FrequentlyExpanded) == 0 || stats.FrequentlyExpanded[0].ItemID != "note:abc123" {
		t.Fatalf("expected note:abc123 in frequently expanded stats: %#v", stats.FrequentlyExpanded)
	}
	if len(stats.TopNoise) != 0 {
		t.Fatalf("did not expect likely-noise items in this scenario: %#v", stats.TopNoise)
	}
}

func TestValidateFinishUsesCommittedDurableNotesFromSessionCommitRange(t *testing.T) {
	verifyCmd := helperCommand("sleep-ms", "10", "verify")
	project := makeSessionProject(t, sessionPolicyYAML(t, []string{strings.Join(verifyCmd, " ")}, true))
	mustInitGitRepo(t, project)

	if err := os.WriteFile(filepath.Join(project, "main.go"), []byte("package main\nfunc main() { println(\"published\") }\n"), 0o644); err != nil {
		t.Fatalf("write code change: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "AGENTS.md"), []byte("# contract\n\nCommitted durable note.\n"), 0o644); err != nil {
		t.Fatalf("write note change: %v", err)
	}

	manager := New(nil)
	if _, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "publish"}); err != nil {
		t.Fatalf("start session: %v", err)
	}
	if _, err := manager.RunCommand(context.Background(), RunRequest{
		ProjectDir:    project,
		Argv:          verifyCmd,
		CaptureOutput: true,
	}, nil, nil); err != nil {
		t.Fatalf("run verification: %v", err)
	}

	mustRunGit(t, project, "add", ".")
	mustRunGit(t, project, "commit", "-m", "publish changes")

	result, err := manager.Validate(context.Background(), ValidateRequest{ProjectDir: project, Stage: "finish"})
	if err != nil {
		t.Fatalf("validate finish: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected finish validation ok, got obligations=%v remediation=%v", result.Obligations, result.Remediation)
	}
	if result.MemorySatisfiedBy != "git_committed_notes" {
		t.Fatalf("expected git_committed_notes, got %q", result.MemorySatisfiedBy)
	}
}

func TestValidateFinishStillBlocksCommittedCodeOnlyPublishSession(t *testing.T) {
	verifyCmd := helperCommand("sleep-ms", "10", "verify")
	project := makeSessionProject(t, sessionPolicyYAML(t, []string{strings.Join(verifyCmd, " ")}, true))
	mustInitGitRepo(t, project)

	if err := os.WriteFile(filepath.Join(project, "main.go"), []byte("package main\nfunc main() { println(\"published\") }\n"), 0o644); err != nil {
		t.Fatalf("write code change: %v", err)
	}

	manager := New(nil)
	if _, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "publish"}); err != nil {
		t.Fatalf("start session: %v", err)
	}
	if _, err := manager.RunCommand(context.Background(), RunRequest{
		ProjectDir:    project,
		Argv:          verifyCmd,
		CaptureOutput: true,
	}, nil, nil); err != nil {
		t.Fatalf("run verification: %v", err)
	}

	mustRunGit(t, project, "add", ".")
	mustRunGit(t, project, "commit", "-m", "publish code only")

	result, err := manager.Validate(context.Background(), ValidateRequest{ProjectDir: project, Stage: "finish"})
	if err != nil {
		t.Fatalf("validate finish: %v", err)
	}
	if result.OK {
		t.Fatalf("expected finish validation to fail without committed durable note, got %+v", result)
	}
	if result.MemorySatisfiedBy != "" {
		t.Fatalf("expected no memory satisfaction source, got %q", result.MemorySatisfiedBy)
	}
	if !containsString(result.Remediation, "run `brain distill --session` to generate a session-scoped memory proposal") {
		t.Fatalf("expected distill remediation, got %v", result.Remediation)
	}
	if len(result.PromotionSuggestions) != 0 {
		t.Fatalf("expected no closeout suggestions without packet-backed evidence, got %#v", result.PromotionSuggestions)
	}
}

func TestValidateFinishSuggestsPacketBackedPromotions(t *testing.T) {
	verifyCmd := helperCommand("sleep-ms", "10", "verify")
	project := makeSessionProject(t, sessionPolicyYAML(t, []string{strings.Join(verifyCmd, " ")}, true))
	mustInitGitRepo(t, project)
	manager := New(nil)
	started, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "replace context loading flow"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	if err := os.WriteFile(filepath.Join(project, "internal.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write code change: %v", err)
	}
	packet := makeCompiledPacket("replace context loading flow", "session")
	if err := manager.RecordCompiledPacket(project, started.Session.ID, packet); err != nil {
		t.Fatalf("record compiled packet: %v", err)
	}
	if _, err := manager.RunCommand(context.Background(), RunRequest{
		ProjectDir:    project,
		Argv:          verifyCmd,
		CaptureOutput: true,
	}, nil, nil); err != nil {
		t.Fatalf("run verification: %v", err)
	}

	result, err := manager.Validate(context.Background(), ValidateRequest{ProjectDir: project, Stage: "finish"})
	if err != nil {
		t.Fatalf("validate finish: %v", err)
	}
	if result.OK {
		t.Fatalf("expected finish validation to fail without durable memory, got %+v", result)
	}
	if len(result.PromotionSuggestions) == 0 {
		t.Fatalf("expected packet-backed promotion suggestions, got %+v", result)
	}
	if result.PromotionSuggestions[0].SuggestedTarget == "" || len(result.PromotionSuggestions[0].SupportingPacketHashes) == 0 {
		t.Fatalf("expected packet-backed suggestion metadata, got %#v", result.PromotionSuggestions)
	}
	if !containsPromotionCategory(result.PromotionSuggestions, "boundary_fact") {
		t.Fatalf("expected boundary_fact suggestion, got %#v", result.PromotionSuggestions)
	}
}

func TestValidateFinishDoesNotUseCommittedDurableNotesWhenWorktreeDirty(t *testing.T) {
	verifyCmd := helperCommand("sleep-ms", "10", "verify")
	project := makeSessionProject(t, sessionPolicyYAML(t, []string{strings.Join(verifyCmd, " ")}, true))
	mustInitGitRepo(t, project)

	if err := os.WriteFile(filepath.Join(project, "main.go"), []byte("package main\nfunc main() { println(\"published\") }\n"), 0o644); err != nil {
		t.Fatalf("write code change: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "AGENTS.md"), []byte("# contract\n\nCommitted durable note.\n"), 0o644); err != nil {
		t.Fatalf("write note change: %v", err)
	}

	manager := New(nil)
	if _, err := manager.Start(context.Background(), StartRequest{ProjectDir: project, Task: "publish"}); err != nil {
		t.Fatalf("start session: %v", err)
	}
	if _, err := manager.RunCommand(context.Background(), RunRequest{
		ProjectDir:    project,
		Argv:          verifyCmd,
		CaptureOutput: true,
	}, nil, nil); err != nil {
		t.Fatalf("run verification: %v", err)
	}

	mustRunGit(t, project, "add", ".")
	mustRunGit(t, project, "commit", "-m", "publish changes")

	if err := os.WriteFile(filepath.Join(project, "README.md"), []byte("# still dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty change: %v", err)
	}

	result, err := manager.Validate(context.Background(), ValidateRequest{ProjectDir: project, Stage: "finish"})
	if err != nil {
		t.Fatalf("validate finish: %v", err)
	}
	if result.OK {
		t.Fatalf("expected finish validation to fail when worktree is dirty, got %+v", result)
	}
	if result.MemorySatisfiedBy != "" {
		t.Fatalf("expected no memory satisfaction source for dirty worktree, got %q", result.MemorySatisfiedBy)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsPromotionCategory(values []PromotionSuggestion, want string) bool {
	for _, value := range values {
		if value.Category == want {
			return true
		}
	}
	return false
}

func TestSessionCommandHelper(t *testing.T) {
	idx := -1
	for i, arg := range os.Args {
		if arg == "--" {
			idx = i
			break
		}
	}
	if idx < 0 || idx+1 >= len(os.Args) {
		return
	}
	switch os.Args[idx+1] {
	case "sleep-ms":
		if idx+2 >= len(os.Args) {
			fmt.Fprintln(os.Stderr, "missing duration")
			os.Exit(2)
		}
		duration, err := time.ParseDuration(os.Args[idx+2] + "ms")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		time.Sleep(duration)
		os.Exit(0)
	default:
		return
	}
}

func makeSessionProject(t *testing.T, policy string) string {
	t.Helper()
	project := t.TempDir()
	for _, rel := range []string{
		".brain/state",
		".brain/context",
		".brain/resources",
		".brain/sessions",
		"docs",
	} {
		if err := os.MkdirAll(filepath.Join(project, rel), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
	}
	files := map[string]string{
		"AGENTS.md":                       "# contract\n",
		".brain/context/overview.md":      "# overview\n",
		".brain/context/workflows.md":     "# workflows\n",
		".brain/context/memory-policy.md": "# memory policy\n",
		"README.md":                       "# project\n",
		".brain/policy.yaml":              policy,
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(project, rel), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return project
}

func makeCompiledPacket(task, source string) *projectcontext.CompiledPacket {
	return &projectcontext.CompiledPacket{
		Task: projectcontext.CompiledTask{
			Text:    task,
			Summary: task,
			Source:  source,
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
			Boundaries: []projectcontext.CompiledBoundary{
				{Path: "internal/taskcontext/", Label: "internal/taskcontext", Role: "library", Reason: "task touches compiler code"},
			},
			Files: []projectcontext.CompiledFile{
				{Path: "cmd/context.go", Status: "modified", Source: "worktree", Reason: "compile command changed"},
			},
			Tests: []projectcontext.CompiledTest{
				{Path: "internal/taskcontext/manager_test.go", Relation: "same_dir", Reason: "adjacent test surface"},
			},
			Notes: []projectcontext.CompiledItem{
				{
					ContextItem: projectcontext.ContextItem{
						ID:      "note:abc123",
						Title:   "Compiler Notes",
						Summary: "Keep packet output compact.",
						Anchor:  projectcontext.ContextAnchor{Path: "docs/compiler.md", Section: "Notes"},
					},
					Reason: "ranked highly in local durable-note search for the task",
				},
			},
		},
		Verification: []projectcontext.VerificationHint{
			{ID: "profile:tests", Label: "tests", Summary: "Verification profile is not satisfied yet.", Source: ".brain/policy.yaml", Reason: "required verification profile is still missing"},
		},
	}
}

func hasTelemetryEvent(events []PacketTelemetryEvent, eventType PacketTelemetryEventType, packetHash, itemID, fileOrPath, command string) bool {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		if packetHash != "" && event.PacketHash != packetHash {
			continue
		}
		if itemID != "" && event.ItemID != itemID {
			continue
		}
		if fileOrPath != "" && event.File != fileOrPath && event.AnchorPath != fileOrPath {
			continue
		}
		if command != "" && event.Command != command {
			continue
		}
		return true
	}
	return false
}

func sessionPolicyYAML(t *testing.T, verificationCommands []string, requireMemoryUpdate bool) string {
	t.Helper()
	commands := []string{
		"go test ./...",
		"go build ./...",
	}
	if len(verificationCommands) != 0 {
		commands = verificationCommands
	}
	lines := make([]string, 0, len(commands)*2)
	names := []string{"tests", "build", "verify-3", "verify-4", "verify-5", "verify-6"}
	for i, command := range commands {
		name := names[i]
		lines = append(lines, fmt.Sprintf("    - name: %s", name))
		lines = append(lines, fmt.Sprintf("      commands:\n        - %q", command))
	}
	return strings.TrimSpace(fmt.Sprintf(`
version: 1
project:
  name: brain
  slug: brain
  runtime: go
  memory:
    accepted_note_globs:
      - AGENTS.md
      - docs/**
      - .brain/context/**
      - .brain/planning/**
      - .brain/brainstorms/**
      - .brain/resources/**
session:
  require_task: true
  single_active: true
  active_file: .brain/session.json
  ledger_dir: .brain/sessions
preflight:
  require_brain_doctor: true
  required_docs:
    - AGENTS.md
    - .brain/context/overview.md
    - .brain/context/workflows.md
    - .brain/context/memory-policy.md
  suggested_commands:
    - brain find brain
closeout:
  acceptable_history_operations:
    - update
  require_memory_update_on_repo_change: %t
  verification_profiles:
%s
`, requireMemoryUpdate, strings.Join(lines, "\n")))
}

func helperCommand(args ...string) []string {
	base := []string{"env", "GO_WANT_SESSION_HELPER=1", os.Args[0], "-test.run=TestSessionCommandHelper", "--"}
	return append(base, args...)
}

func mustInitGitRepo(t *testing.T, dir string) {
	t.Helper()
	commands := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "brain-tests@example.com"},
		{"git", "config", "user.name", "Brain Tests"},
		{"git", "add", "."},
		{"git", "commit", "-m", "initial"},
	}
	for _, argv := range commands {
		mustRunGit(t, dir, argv[1:]...)
	}
}

func mustRunGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}
