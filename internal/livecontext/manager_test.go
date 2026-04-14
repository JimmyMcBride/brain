package livecontext

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"brain/internal/session"
)

func TestCollectRequiresTask(t *testing.T) {
	manager := New()
	if _, err := manager.Collect(context.Background(), Request{}); err == nil {
		t.Fatal("expected task requirement error")
	}
}

func TestCollectReturnsStablePacketShape(t *testing.T) {
	manager := New()
	packet, err := manager.Collect(context.Background(), Request{
		ProjectDir: t.TempDir(),
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
	manager := New()
	now := time.Date(2026, 4, 14, 0, 30, 0, 0, time.UTC)
	packet, err := manager.Collect(context.Background(), Request{
		ProjectDir: t.TempDir(),
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
