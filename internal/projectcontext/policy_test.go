package projectcontext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolicyMergesOverride(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".brain"), 0o755); err != nil {
		t.Fatal(err)
	}
	base := `version: 1
project:
  name: demo
  slug: demo
  runtime: go
  memory:
    accepted_note_globs:
      - Projects/demo.md
session:
  require_task: true
  single_active: true
  active_file: .brain/session.json
  ledger_dir: .brain/sessions
preflight:
  require_brain_doctor: true
  required_docs:
    - AGENTS.md
  suggested_commands:
    - brain find demo
closeout:
  acceptable_history_operations:
    - create
  require_memory_update_on_repo_change: true
  require_reindex_after_note_updates: true
`
	override := `project:
  memory:
    accepted_note_globs:
      - Resources/Captures/**/demo*.md
closeout:
  verification_profiles:
    - name: tests
      commands:
        - go test ./...
`
	if err := os.WriteFile(filepath.Join(project, ".brain", "policy.yaml"), []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".brain", "policy.override.yaml"), []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	policy, _, _, err := LoadPolicy(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(policy.Project.Memory.AcceptedNoteGlobs) != 1 || policy.Project.Memory.AcceptedNoteGlobs[0] != "Resources/Captures/**/demo*.md" {
		t.Fatalf("override globs not applied: %+v", policy.Project.Memory.AcceptedNoteGlobs)
	}
	if len(policy.Closeout.VerificationProfiles) != 1 || policy.Closeout.VerificationProfiles[0].Commands[0] != "go test ./..." {
		t.Fatalf("override verification profiles not applied: %+v", policy.Closeout.VerificationProfiles)
	}
}
