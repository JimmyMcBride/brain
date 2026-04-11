package projectcontext

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Policy struct {
	Version   int             `yaml:"version" json:"version"`
	Project   PolicyProject   `yaml:"project" json:"project"`
	Session   PolicySession   `yaml:"session" json:"session"`
	Preflight PolicyPreflight `yaml:"preflight" json:"preflight"`
	Closeout  PolicyCloseout  `yaml:"closeout" json:"closeout"`
}

type PolicyProject struct {
	Name    string              `yaml:"name" json:"name"`
	Slug    string              `yaml:"slug" json:"slug"`
	Runtime string              `yaml:"runtime" json:"runtime"`
	Memory  PolicyProjectMemory `yaml:"memory" json:"memory"`
}

type PolicyProjectMemory struct {
	AcceptedNoteGlobs []string `yaml:"accepted_note_globs" json:"accepted_note_globs"`
}

type PolicySession struct {
	RequireTask  bool   `yaml:"require_task" json:"require_task"`
	SingleActive bool   `yaml:"single_active" json:"single_active"`
	ActiveFile   string `yaml:"active_file" json:"active_file"`
	LedgerDir    string `yaml:"ledger_dir" json:"ledger_dir"`
}

type PolicyPreflight struct {
	RequireBrainDoctor bool     `yaml:"require_brain_doctor" json:"require_brain_doctor"`
	RequiredDocs       []string `yaml:"required_docs" json:"required_docs"`
	SuggestedCommands  []string `yaml:"suggested_commands" json:"suggested_commands"`
}

type PolicyCloseout struct {
	AcceptableHistoryOperations     []string              `yaml:"acceptable_history_operations" json:"acceptable_history_operations"`
	RequireMemoryUpdateOnRepoChange bool                  `yaml:"require_memory_update_on_repo_change" json:"require_memory_update_on_repo_change"`
	RequireReindexAfterNoteUpdates  bool                  `yaml:"require_reindex_after_note_updates" json:"require_reindex_after_note_updates"`
	VerificationProfiles            []VerificationProfile `yaml:"verification_profiles" json:"verification_profiles"`
}

type VerificationProfile struct {
	Name     string   `yaml:"name" json:"name"`
	Commands []string `yaml:"commands" json:"commands"`
}

func DefaultPolicy(snapshot Snapshot) Policy {
	slug := policySlug(snapshot.ProjectName)
	folder := policyFolderName(snapshot.ProjectName)
	policy := Policy{
		Version: 1,
		Project: PolicyProject{
			Name:    snapshot.ProjectName,
			Slug:    slug,
			Runtime: snapshot.PrimaryRuntime,
			Memory: PolicyProjectMemory{
				AcceptedNoteGlobs: []string{
					fmt.Sprintf("Projects/%s.md", slug),
					fmt.Sprintf("Projects/%s/**", folder),
					fmt.Sprintf("Resources/Captures/**/%s*.md", slug),
				},
			},
		},
		Session: PolicySession{
			RequireTask:  true,
			SingleActive: true,
			ActiveFile:   ".brain/session.json",
			LedgerDir:    ".brain/sessions",
		},
		Preflight: PolicyPreflight{
			RequireBrainDoctor: true,
			RequiredDocs: []string{
				"AGENTS.md",
				".brain/context/overview.md",
				".brain/context/workflows.md",
				".brain/context/memory-policy.md",
			},
			SuggestedCommands: []string{
				fmt.Sprintf("brain find %s", slug),
				fmt.Sprintf("brain search \"%s {task}\"", slug),
			},
		},
		Closeout: PolicyCloseout{
			AcceptableHistoryOperations:     []string{"create", "update", "move", "rename", "publish", "seed"},
			RequireMemoryUpdateOnRepoChange: true,
			RequireReindexAfterNoteUpdates:  true,
		},
	}
	if snapshot.PrimaryRuntime == "go" {
		policy.Closeout.VerificationProfiles = []VerificationProfile{
			{Name: "tests", Commands: []string{"go test ./..."}},
			{Name: "build", Commands: []string{"go build ./..."}},
		}
	}
	return policy
}

func RenderPolicy(snapshot Snapshot) (string, error) {
	policy := DefaultPolicy(snapshot)
	out, err := yaml.Marshal(&policy)
	if err != nil {
		return "", fmt.Errorf("marshal policy: %w", err)
	}
	return string(out), nil
}

func LoadPolicy(projectDir string) (*Policy, string, string, error) {
	basePath := filepath.Join(projectDir, ".brain", "policy.yaml")
	overridePath := filepath.Join(projectDir, ".brain", "policy.override.yaml")
	raw, err := os.ReadFile(basePath)
	if err != nil {
		return nil, basePath, overridePath, fmt.Errorf("read policy: %w", err)
	}
	var base Policy
	if err := yaml.Unmarshal(raw, &base); err != nil {
		return nil, basePath, overridePath, fmt.Errorf("parse policy: %w", err)
	}
	normalizePolicy(&base)
	if overrideRaw, err := os.ReadFile(overridePath); err == nil && len(strings.TrimSpace(string(overrideRaw))) != 0 {
		var override Policy
		if err := yaml.Unmarshal(overrideRaw, &override); err != nil {
			return nil, basePath, overridePath, fmt.Errorf("parse policy override: %w", err)
		}
		mergePolicy(&base, &override)
		normalizePolicy(&base)
	}
	return &base, basePath, overridePath, nil
}

func normalizePolicy(policy *Policy) {
	if policy == nil {
		return
	}
	if policy.Version == 0 {
		policy.Version = 1
	}
	if policy.Session.ActiveFile == "" {
		policy.Session.ActiveFile = ".brain/session.json"
	}
	if policy.Session.LedgerDir == "" {
		policy.Session.LedgerDir = ".brain/sessions"
	}
	if len(policy.Preflight.RequiredDocs) == 0 {
		policy.Preflight.RequiredDocs = []string{
			"AGENTS.md",
			".brain/context/overview.md",
			".brain/context/workflows.md",
			".brain/context/memory-policy.md",
		}
	}
}

func mergePolicy(base, override *Policy) {
	if override == nil || base == nil {
		return
	}
	if override.Version != 0 {
		base.Version = override.Version
	}
	if override.Project.Name != "" {
		base.Project.Name = override.Project.Name
	}
	if override.Project.Slug != "" {
		base.Project.Slug = override.Project.Slug
	}
	if override.Project.Runtime != "" {
		base.Project.Runtime = override.Project.Runtime
	}
	if len(override.Project.Memory.AcceptedNoteGlobs) != 0 {
		base.Project.Memory.AcceptedNoteGlobs = append([]string(nil), override.Project.Memory.AcceptedNoteGlobs...)
	}
	if override.Session.ActiveFile != "" {
		base.Session.ActiveFile = override.Session.ActiveFile
	}
	if override.Session.LedgerDir != "" {
		base.Session.LedgerDir = override.Session.LedgerDir
	}
	if override.Session.RequireTask {
		base.Session.RequireTask = true
	}
	if override.Session.SingleActive {
		base.Session.SingleActive = true
	}
	if len(override.Preflight.RequiredDocs) != 0 {
		base.Preflight.RequiredDocs = append([]string(nil), override.Preflight.RequiredDocs...)
	}
	if len(override.Preflight.SuggestedCommands) != 0 {
		base.Preflight.SuggestedCommands = append([]string(nil), override.Preflight.SuggestedCommands...)
	}
	if override.Preflight.RequireBrainDoctor {
		base.Preflight.RequireBrainDoctor = true
	}
	if len(override.Closeout.AcceptableHistoryOperations) != 0 {
		base.Closeout.AcceptableHistoryOperations = append([]string(nil), override.Closeout.AcceptableHistoryOperations...)
	}
	if override.Closeout.RequireMemoryUpdateOnRepoChange {
		base.Closeout.RequireMemoryUpdateOnRepoChange = true
	}
	if override.Closeout.RequireReindexAfterNoteUpdates {
		base.Closeout.RequireReindexAfterNoteUpdates = true
	}
	if len(override.Closeout.VerificationProfiles) != 0 {
		base.Closeout.VerificationProfiles = append([]VerificationProfile(nil), override.Closeout.VerificationProfiles...)
	}
}

func policySlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() != 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func policyFolderName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	if len(parts) == 0 {
		return name
	}
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}
