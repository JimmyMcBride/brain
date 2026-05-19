package projectcontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKarpathyGuidanceStatusMissingAndMalformedStateAreUnset(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), legacyManagedAgentsWithoutKarpathy())
	manager := New(t.TempDir())

	status, err := manager.KarpathyGuidanceStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if status.Decision != GuidanceDecisionUnset || status.StateStatus != projectMigrationStateMissing || status.Recommendation == nil {
		t.Fatalf("unexpected missing-state status: %+v", status)
	}

	mustWriteFile(t, guidanceDecisionStatePath(root), "{not-json")
	status, err = manager.KarpathyGuidanceStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if status.Decision != GuidanceDecisionUnset || status.StateStatus != projectMigrationStateInvalid || status.Recommendation == nil {
		t.Fatalf("unexpected malformed-state status: %+v", status)
	}
}

func TestKarpathyGuidanceDecisionSuppressesOrAppliesRecommendation(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "README.md"), "# demo\n")
	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), legacyManagedAgentsWithoutKarpathy())
	manager := New(t.TempDir())

	declined, err := manager.SetKarpathyGuidanceDecision(context.Background(), root, GuidanceDecisionDeclined)
	if err != nil {
		t.Fatal(err)
	}
	if declined.Decision != GuidanceDecisionDeclined || declined.Recommendation != nil {
		t.Fatalf("expected declined decision without recommendation: %+v", declined)
	}
	body := readFileString(t, filepath.Join(root, "AGENTS.md"))
	if strings.Contains(body, KarpathyGuidanceHeading) {
		t.Fatalf("decline should not add guidelines:\n%s", body)
	}

	accepted, err := manager.SetKarpathyGuidanceDecision(context.Background(), root, GuidanceDecisionAccepted)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Decision != GuidanceDecisionAccepted || !accepted.GuidelinesPresent || accepted.Action != "updated" {
		t.Fatalf("expected accept to update guidelines: %+v", accepted)
	}
	body = readFileString(t, filepath.Join(root, "AGENTS.md"))
	if !strings.Contains(body, KarpathyGuidanceHeading) || !strings.Contains(body, "keep me") {
		t.Fatalf("expected guidelines and preserved local notes:\n%s", body)
	}
}

func TestKarpathyGuidanceExistingGuidelinesSuppressRecommendation(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), "# Project Agent Contract\n\n<!-- brain:begin agents-contract -->\n## Karpathy Guidelines\n\nAlready here.\n<!-- brain:end agents-contract -->\n")
	manager := New(t.TempDir())

	status, err := manager.KarpathyGuidanceStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if !status.GuidelinesPresent || status.Recommendation != nil {
		t.Fatalf("expected existing guidelines to suppress recommendation: %+v", status)
	}
}

func legacyManagedAgentsWithoutKarpathy() string {
	return "# Project Agent Contract\n\n<!-- brain:begin agents-contract -->\nUse Brain.\n<!-- brain:end agents-contract -->\n\n## Local Notes\n\nkeep me\n"
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
