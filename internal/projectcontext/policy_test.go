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
      - AGENTS.md
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
`
	override := `project:
  memory:
    accepted_note_globs:
      - .brain/resources/**/demo*.md
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
	if len(policy.Project.Memory.AcceptedNoteGlobs) != 1 || policy.Project.Memory.AcceptedNoteGlobs[0] != ".brain/resources/**/demo*.md" {
		t.Fatalf("override globs not applied: %+v", policy.Project.Memory.AcceptedNoteGlobs)
	}
	if len(policy.Closeout.VerificationProfiles) != 1 || policy.Closeout.VerificationProfiles[0].Commands[0] != "go test ./..." {
		t.Fatalf("override verification profiles not applied: %+v", policy.Closeout.VerificationProfiles)
	}
}

func TestLoadPolicyOverrideCanDisableBooleans(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".brain"), 0o755); err != nil {
		t.Fatal(err)
	}
	base := `version: 1
project:
  name: demo
  slug: demo
  runtime: go
session:
  require_task: true
  single_active: true
  active_file: .brain/session.json
  ledger_dir: .brain/sessions
preflight:
  require_brain_doctor: true
  required_docs:
    - AGENTS.md
closeout:
  require_memory_update_on_repo_change: true
`
	override := `session:
  require_task: false
  single_active: false
preflight:
  require_brain_doctor: false
closeout:
  require_memory_update_on_repo_change: false
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
	if policy.Session.RequireTask {
		t.Fatal("expected require_task to be disabled by override")
	}
	if policy.Session.SingleActive {
		t.Fatal("expected single_active to be disabled by override")
	}
	if policy.Preflight.RequireBrainDoctor {
		t.Fatal("expected require_brain_doctor to be disabled by override")
	}
	if policy.Closeout.RequireMemoryUpdateOnRepoChange {
		t.Fatal("expected require_memory_update_on_repo_change to be disabled by override")
	}
}

func TestLoadPolicyOverrideCanEnableBooleans(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".brain"), 0o755); err != nil {
		t.Fatal(err)
	}
	base := `version: 1
project:
  name: demo
  slug: demo
  runtime: go
session:
  require_task: false
  single_active: false
  active_file: .brain/session.json
  ledger_dir: .brain/sessions
preflight:
  require_brain_doctor: false
  required_docs:
    - AGENTS.md
closeout:
  require_memory_update_on_repo_change: false
`
	override := `session:
  require_task: true
  single_active: true
preflight:
  require_brain_doctor: true
closeout:
  require_memory_update_on_repo_change: true
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
	if !policy.Session.RequireTask {
		t.Fatal("expected require_task to be enabled by override")
	}
	if !policy.Session.SingleActive {
		t.Fatal("expected single_active to be enabled by override")
	}
	if !policy.Preflight.RequireBrainDoctor {
		t.Fatal("expected require_brain_doctor to be enabled by override")
	}
	if !policy.Closeout.RequireMemoryUpdateOnRepoChange {
		t.Fatal("expected require_memory_update_on_repo_change to be enabled by override")
	}
}
