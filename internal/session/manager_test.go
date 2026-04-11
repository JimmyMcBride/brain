package session

import (
	"testing"

	"brain/internal/projectcontext"
)

func TestPathMatchesAny(t *testing.T) {
	if !pathMatchesAny("Resources/Captures/2026/04/project-change.md", []string{"Resources/Captures/**/project*.md"}) {
		t.Fatal("expected glob to match capture path")
	}
	if pathMatchesAny("Resources/Elsewhere/note.md", []string{"Projects/Project/**"}) {
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
