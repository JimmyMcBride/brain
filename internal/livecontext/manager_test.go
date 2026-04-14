package livecontext

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"brain/internal/history"
	"brain/internal/projectcontext"
	"brain/internal/session"
	"brain/internal/structure"
)

func TestCollectRequiresTask(t *testing.T) {
	manager := New(nil)
	if _, err := manager.Collect(context.Background(), Request{}); err == nil {
		t.Fatal("expected task requirement error")
	}
}

func TestCollectReturnsStablePacketShape(t *testing.T) {
	manager := New(nil)
	project := t.TempDir()
	writePolicyFixture(t, project)
	packet, err := manager.Collect(context.Background(), Request{
		ProjectDir: project,
		Task:       "tighten auth flow",
		TaskSource: "flag",
	})
	if err != nil {
		t.Fatal(err)
	}
	if packet.Task.Text != "tighten auth flow" || packet.Task.Source != "flag" {
		t.Fatalf("unexpected task payload: %#v", packet.Task)
	}
	if packet.Worktree.ChangedFiles == nil || packet.Worktree.TouchedBoundaries == nil || packet.NearbyTests == nil || packet.Verification.RecentCommands == nil || packet.Verification.Profiles == nil || packet.PolicyHints == nil {
		t.Fatalf("expected packet arrays to be initialized: %#v", packet)
	}
	if len(packet.Ambiguities) == 0 {
		t.Fatalf("expected baseline ambiguities for thin first-wave packet: %#v", packet)
	}
}

func TestCollectIncludesSessionMetadata(t *testing.T) {
	manager := New(nil)
	project := t.TempDir()
	writePolicyFixture(t, project)
	now := time.Date(2026, 4, 14, 0, 30, 0, 0, time.UTC)
	packet, err := manager.Collect(context.Background(), Request{
		ProjectDir: project,
		Task:       "session task",
		TaskSource: "session",
		Session: &session.ActiveSession{
			ID:        "123",
			Task:      "session task",
			StartedAt: now,
			GitBaseline: session.GitSnapshot{
				Head: "abc123",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !packet.Session.Active || packet.Session.ID != "123" || packet.Session.StartedAt != "2026-04-14T00:30:00Z" {
		t.Fatalf("unexpected session payload: %#v", packet.Session)
	}
	if packet.Worktree.BaselineHead != "abc123" {
		t.Fatalf("expected baseline head in worktree payload: %#v", packet.Worktree)
	}
}

func TestCollectDerivesChangedFilesTouchedBoundariesAndNearbyTests(t *testing.T) {
	project := t.TempDir()
	writePolicyFixture(t, project)
	runGitCmd(t, project, "init")
	runGitCmd(t, project, "config", "user.name", "Test User")
	runGitCmd(t, project, "config", "user.email", "test@example.com")
	for path, body := range map[string]string{
		"go.mod":                         "module example.com/test\n\ngo 1.26\n",
		"internal/search/search.go":      "package search\n",
		"internal/search/search_test.go": "package search\n",
		"config/search.yaml":             "enabled: true\n",
	} {
		if err := os.MkdirAll(filepath.Join(project, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runGitCmd(t, project, "add", ".")
	runGitCmd(t, project, "commit", "-m", "baseline")
	baseline := strings.TrimSpace(runGitOutputCmd(t, project, "rev-parse", "HEAD"))

	if err := os.WriteFile(filepath.Join(project, "config", "search.yaml"), []byte("enabled: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, project, "add", "config/search.yaml")
	runGitCmd(t, project, "commit", "-m", "adjust config")
	if err := os.WriteFile(filepath.Join(project, "internal", "search", "search.go"), []byte("package search\n\nfunc Search() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	manager := New(nil)
	packet, err := manager.Collect(context.Background(), Request{
		ProjectDir: project,
		Task:       "search config",
		TaskSource: "session",
		Session: &session.ActiveSession{
			ID:        "s1",
			Task:      "search config",
			StartedAt: time.Now().UTC(),
			GitBaseline: session.GitSnapshot{
				Available: true,
				Head:      baseline,
			},
		},
		StructuralSnapshot: &structure.Snapshot{
			Boundaries: []structure.Item{
				{Kind: "boundary", Path: "internal/search/", Label: "internal/search", Role: "library"},
				{Kind: "boundary", Path: "config/", Label: "config", Role: "config"},
			},
			TestSurfaces: []structure.Item{
				{Kind: "test_surface", Path: "internal/search/search_test.go", Label: "search tests", Role: "tests"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(packet.Worktree.ChangedFiles) < 2 {
		t.Fatalf("expected changed files: %#v", packet)
	}
	byPath := map[string]ChangedFile{}
	for _, file := range packet.Worktree.ChangedFiles {
		byPath[file.Path] = file
	}
	if byPath["config/search.yaml"].Source != "commit_range" {
		t.Fatalf("expected committed-since-baseline file in payload: %#v", packet.Worktree.ChangedFiles)
	}
	if byPath["internal/search/search.go"].Source != "worktree" {
		t.Fatalf("expected worktree file in payload: %#v", packet.Worktree.ChangedFiles)
	}
	if len(packet.Worktree.TouchedBoundaries) < 2 {
		t.Fatalf("expected touched boundary: %#v", packet.Worktree.TouchedBoundaries)
	}
	if len(packet.NearbyTests) == 0 || packet.NearbyTests[0].Path != "internal/search/search_test.go" {
		t.Fatalf("expected nearby test: %#v", packet.NearbyTests)
	}
}

func TestCollectAddsVerificationProfilesAndStrongMatchPolicyHints(t *testing.T) {
	project := t.TempDir()
	runGitCmd(t, project, "init")
	runGitCmd(t, project, "config", "user.name", "Test User")
	runGitCmd(t, project, "config", "user.email", "test@example.com")
	policyBody, err := projectcontext.RenderPolicy(projectcontext.Snapshot{ProjectName: "brain", PrimaryRuntime: "go"})
	if err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{
		".brain/policy.yaml":              policyBody,
		"AGENTS.md":                       "# Project Agent Contract\n",
		".brain/context/overview.md":      "# Overview\n",
		".brain/context/workflows.md":     "# Workflows\n",
		".brain/context/memory-policy.md": "# Memory Policy\n",
		"go.mod":                          "module example.com/test\n\ngo 1.26\n",
		"internal/search/search.go":       "package search\n",
		"internal/search/search_test.go":  "package search\n",
	} {
		if err := os.MkdirAll(filepath.Join(project, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runGitCmd(t, project, "add", ".")
	runGitCmd(t, project, "commit", "-m", "baseline")
	baseline := strings.TrimSpace(runGitOutputCmd(t, project, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(project, "internal", "search", "search.go"), []byte("package search\n\nfunc Search() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	historyLog := history.New(filepath.Join(project, ".brain", "history.log"))
	manager := New(historyLog)
	packet, err := manager.Collect(context.Background(), Request{
		ProjectDir: project,
		Task:       "search config",
		TaskSource: "session",
		Session: &session.ActiveSession{
			ID:        "s2",
			Task:      "search config",
			StartedAt: time.Now().UTC(),
			GitBaseline: session.GitSnapshot{
				Available: true,
				Head:      baseline,
			},
			HistoryBaseline: session.HistoryBaseline{
				LastTimestamp: time.Now().Add(-time.Hour),
			},
			CommandRuns: []session.CommandRun{
				{
					Command:   "go test ./...",
					ExitCode:  0,
					StartedAt: time.Now().Add(-2 * time.Minute),
					EndedAt:   time.Now().Add(-time.Minute),
				},
			},
		},
		StructuralSnapshot: &structure.Snapshot{
			Boundaries: []structure.Item{
				{Kind: "boundary", Path: "internal/search/", Label: "internal/search", Role: "library"},
			},
			TestSurfaces: []structure.Item{
				{Kind: "test_surface", Path: "internal/search/search_test.go", Label: "search tests", Role: "tests"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(packet.Verification.RecentCommands) != 1 || packet.Verification.RecentCommands[0].Command != "go test ./..." {
		t.Fatalf("expected recorded verification command: %#v", packet.Verification)
	}
	if len(packet.Verification.Profiles) < 2 {
		t.Fatalf("expected verification profiles from policy: %#v", packet.Verification)
	}
	var buildMissing bool
	for _, profile := range packet.Verification.Profiles {
		if profile.Name == "build" && !profile.Satisfied {
			buildMissing = true
		}
	}
	if !buildMissing {
		t.Fatalf("expected unsatisfied build profile: %#v", packet.Verification.Profiles)
	}
	if len(packet.PolicyHints) < 2 {
		t.Fatalf("expected strong-match policy hints: %#v", packet.PolicyHints)
	}
}

func TestRenderHumanIncludesCoreSections(t *testing.T) {
	packet := &Packet{
		Task:        TaskInfo{Text: "tighten auth flow", Source: "flag"},
		Session:     SessionInfo{Active: false},
		Worktree:    WorktreeInfo{ChangedFiles: []ChangedFile{}, TouchedBoundaries: []TouchedBoundary{}},
		NearbyTests: []NearbyTest{},
		Verification: Verification{
			RecentCommands: []VerificationCommand{},
			Profiles:       []VerificationProfile{},
		},
		PolicyHints: []PolicyHint{},
		Ambiguities: []string{"using explicit task text without an active session"},
	}
	var out bytes.Buffer
	if err := RenderHuman(&out, packet, true); err != nil {
		t.Fatal(err)
	}
	rendered := out.String()
	for _, heading := range []string{"## Task", "## Session", "## Changed Files", "## Touched Boundaries", "## Nearby Tests", "## Verification", "## Ambiguities", "## Why These Signals Matter", "## Missing Live Signals"} {
		if !strings.Contains(rendered, heading) {
			t.Fatalf("expected %s in human output:\n%s", heading, rendered)
		}
	}
}

func TestNormalizePathUsesSlashSeparators(t *testing.T) {
	got := normalizePath(filepath.Join("internal", "search", "search.go"))
	if strings.Contains(got, "\\") {
		t.Fatalf("expected slash-normalized path, got %q", got)
	}
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func runGitOutputCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return string(out)
}

func writePolicyFixture(t *testing.T, project string) {
	t.Helper()
	body, err := projectcontext.RenderPolicy(projectcontext.Snapshot{ProjectName: "brain", PrimaryRuntime: "go"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, ".brain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".brain", "policy.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
