package conformance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

const githubManifestPath = "testdata/github-v1/manifest.yaml"

type GitHubManifest struct {
	SchemaVersion int                    `yaml:"schema_version"`
	Baseline      GitHubBaseline         `yaml:"baseline"`
	Normalization Normalization          `yaml:"normalization"`
	Provider      GitHubProviderContract `yaml:"provider"`
	Families      []GitHubFamily         `yaml:"families"`
	Fixtures      []GitHubFixture        `yaml:"fixtures"`
	Cases         []GitHubCase           `yaml:"cases"`
}

type GitHubBaseline struct {
	Repository      string `yaml:"repository"`
	Tag             string `yaml:"tag"`
	Revision        string `yaml:"revision"`
	Command         string `yaml:"command"`
	WorkspaceSchema int    `yaml:"workspace_schema"`
}

type GitHubProviderContract struct {
	Name           string `yaml:"name"`
	Transport      string `yaml:"transport"`
	RequiredCILive bool   `yaml:"required_ci_live"`
}

type GitHubFamily struct {
	ID             string   `yaml:"id"`
	Standalone     []string `yaml:"standalone,omitempty"`
	Native         []string `yaml:"native,omitempty"`
	Operation      string   `yaml:"operation,omitempty"`
	Retention      string   `yaml:"retention"`
	Behavior       string   `yaml:"behavior"`
	SourceEvidence []string `yaml:"source_evidence"`
}

type GitHubFixture struct {
	ID            string `yaml:"id"`
	Workspace     string `yaml:"workspace"`
	ProviderState string `yaml:"provider_state"`
}

type GitHubCase struct {
	ID                     string           `yaml:"id"`
	Family                 string           `yaml:"family"`
	Fixture                string           `yaml:"fixture"`
	Invocation             GitHubInvocation `yaml:"invocation"`
	Safety                 GitHubSafety     `yaml:"safety"`
	Expected               GitHubExpected   `yaml:"expected"`
	Rerun                  *GitHubRerun     `yaml:"rerun,omitempty"`
	IntentionalDifferences []string         `yaml:"intentional_differences,omitempty"`
}

type GitHubInvocation struct {
	Kind      string   `yaml:"kind"`
	Args      []string `yaml:"args,omitempty"`
	Operation string   `yaml:"operation,omitempty"`
}

type GitHubSafety struct {
	Mutation              bool   `yaml:"mutation"`
	BaselineConfirmation  string `yaml:"baseline_confirmation"`
	TargetConfirmation    string `yaml:"target_confirmation"`
	PartialFailure        bool   `yaml:"partial_failure"`
	ManualFallbackAllowed bool   `yaml:"manual_fallback_allowed"`
}

type GitHubExpected struct {
	ExitCode           int              `yaml:"exit_code"`
	Format             string           `yaml:"format"`
	StdoutComparison   string           `yaml:"stdout_comparison"`
	Stdout             string           `yaml:"stdout"`
	Stderr             string           `yaml:"stderr"`
	Files              string           `yaml:"files"`
	ProviderTranscript string           `yaml:"provider_transcript"`
	GitHubMetadata     string           `yaml:"github_metadata"`
	RemoteIdentities   []RemoteIdentity `yaml:"remote_identities,omitempty"`
	Classifications    []string         `yaml:"classifications,omitempty"`
}

type GitHubRerun struct {
	Kind               string   `yaml:"kind"`
	BaselineMutation   bool     `yaml:"baseline_mutation"`
	ExitCode           int      `yaml:"exit_code"`
	Stdout             string   `yaml:"stdout"`
	Stderr             string   `yaml:"stderr"`
	Files              string   `yaml:"files"`
	ProviderTranscript string   `yaml:"provider_transcript"`
	GitHubMetadata     string   `yaml:"github_metadata"`
	Classifications    []string `yaml:"classifications"`
}

type RemoteIdentity struct {
	Provider  string `yaml:"provider"`
	Kind      string `yaml:"kind"`
	OpaqueID  string `yaml:"opaque_id"`
	DisplayID string `yaml:"display_id"`
	URL       string `yaml:"url"`
	Revision  string `yaml:"revision,omitempty"`
}

type ProviderTranscript struct {
	SchemaVersion int                      `json:"schema_version"`
	Calls         []ProviderTranscriptCall `json:"calls"`
}

type ProviderTranscriptCall struct {
	Operation string         `json:"operation"`
	Input     map[string]any `json:"input,omitempty"`
	Output    map[string]any `json:"output,omitempty"`
	Error     string         `json:"error,omitempty"`
}

func LoadGitHub() (GitHubManifest, error) {
	raw, err := testdata.ReadFile(githubManifestPath)
	if err != nil {
		return GitHubManifest{}, fmt.Errorf("read GitHub conformance manifest: %w", err)
	}
	var manifest GitHubManifest
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return GitHubManifest{}, fmt.Errorf("parse GitHub conformance manifest: %w", err)
	}
	if err := manifest.validate(); err != nil {
		return GitHubManifest{}, err
	}
	return manifest, nil
}

func ReadGitHubAsset(name string) ([]byte, error) {
	if name == "empty" || name == "unchanged" {
		return nil, nil
	}
	if !validGitHubAssetName(name) {
		return nil, fmt.Errorf("invalid GitHub conformance asset reference %q", name)
	}
	raw, err := testdata.ReadFile(path.Join("testdata", name))
	if err != nil {
		return nil, fmt.Errorf("read GitHub conformance asset %s: %w", name, err)
	}
	return raw, nil
}

func ReadProviderTranscript(name string) (ProviderTranscript, error) {
	raw, err := ReadGitHubAsset(name)
	if err != nil {
		return ProviderTranscript{}, err
	}
	var transcript ProviderTranscript
	if err := json.Unmarshal(raw, &transcript); err != nil {
		return ProviderTranscript{}, fmt.Errorf("parse provider transcript %s: %w", name, err)
	}
	if transcript.SchemaVersion != 1 {
		return ProviderTranscript{}, fmt.Errorf("provider transcript %s has unsupported schema %d", name, transcript.SchemaVersion)
	}
	for i, call := range transcript.Calls {
		if strings.TrimSpace(call.Operation) == "" {
			return ProviderTranscript{}, fmt.Errorf("provider transcript %s call %d has no operation", name, i+1)
		}
		if call.Error != "" && call.Output != nil {
			return ProviderTranscript{}, fmt.Errorf("provider transcript %s call %d has both output and error", name, i+1)
		}
	}
	return transcript, nil
}

func (m GitHubManifest) validate() error {
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported GitHub conformance manifest schema %d", m.SchemaVersion)
	}
	if m.Baseline.Repository != "https://github.com/JimmyMcBride/plan" || m.Baseline.Tag != "v0.1.29" {
		return fmt.Errorf("unexpected GitHub baseline repository or tag: %#v", m.Baseline)
	}
	if m.Baseline.Revision != "898b2a4c470350c0e9115302c99c6ad27bdead9c" || !revisionPattern.MatchString(m.Baseline.Revision) {
		return fmt.Errorf("unexpected GitHub baseline revision %q", m.Baseline.Revision)
	}
	if m.Baseline.Command != "plan" || m.Baseline.WorkspaceSchema != 3 {
		return fmt.Errorf("unexpected GitHub baseline command or workspace schema: %#v", m.Baseline)
	}
	if m.Normalization.LineEndings != "lf" || m.Normalization.PathSeparator != "/" ||
		m.Normalization.ProjectRootToken != "<PROJECT>" || m.Normalization.Timestamps != "rfc3339_token" {
		return fmt.Errorf("unsupported GitHub conformance normalization: %#v", m.Normalization)
	}
	if m.Provider.Name != "github" || m.Provider.Transport != "injected_fake" || m.Provider.RequiredCILive {
		return fmt.Errorf("unsupported GitHub provider contract: %#v", m.Provider)
	}

	families := make(map[string]GitHubFamily, len(m.Families))
	for _, family := range m.Families {
		if strings.TrimSpace(family.ID) == "" || strings.TrimSpace(family.Behavior) == "" || len(family.SourceEvidence) == 0 {
			return fmt.Errorf("incomplete GitHub conformance family %#v", family)
		}
		if _, exists := families[family.ID]; exists {
			return fmt.Errorf("duplicate GitHub conformance family %q", family.ID)
		}
		if len(family.Standalone) == 0 && strings.TrimSpace(family.Operation) == "" {
			return fmt.Errorf("GitHub conformance family %s has no entrypoint", family.ID)
		}
		if family.Retention != "migrate" && family.Retention != "compatibility-only" {
			return fmt.Errorf("GitHub conformance family %s has invalid retention %q", family.ID, family.Retention)
		}
		families[family.ID] = family
	}

	fixtures := make(map[string]struct{}, len(m.Fixtures))
	for _, fixture := range m.Fixtures {
		if strings.TrimSpace(fixture.ID) == "" {
			return fmt.Errorf("GitHub conformance fixture id is required")
		}
		if _, exists := fixtures[fixture.ID]; exists {
			return fmt.Errorf("duplicate GitHub conformance fixture %q", fixture.ID)
		}
		if !validGitHubFixtureName(fixture.Workspace) {
			return fmt.Errorf("invalid GitHub workspace fixture %q", fixture.Workspace)
		}
		info, err := fs.Stat(testdata, path.Join("testdata", fixture.Workspace))
		if err != nil || !info.IsDir() {
			return fmt.Errorf("open GitHub workspace fixture %s: %w", fixture.Workspace, err)
		}
		if err := validateJSONAsset(fixture.ProviderState); err != nil {
			return fmt.Errorf("GitHub fixture %s provider state: %w", fixture.ID, err)
		}
		fixtures[fixture.ID] = struct{}{}
	}

	cases := make(map[string]struct{}, len(m.Cases))
	for _, testCase := range m.Cases {
		if strings.TrimSpace(testCase.ID) == "" {
			return fmt.Errorf("GitHub conformance case id is required")
		}
		if _, exists := cases[testCase.ID]; exists {
			return fmt.Errorf("duplicate GitHub conformance case %q", testCase.ID)
		}
		cases[testCase.ID] = struct{}{}
		if _, exists := families[testCase.Family]; !exists {
			return fmt.Errorf("GitHub conformance case %s references unknown family %s", testCase.ID, testCase.Family)
		}
		if _, exists := fixtures[testCase.Fixture]; !exists {
			return fmt.Errorf("GitHub conformance case %s references unknown fixture %s", testCase.ID, testCase.Fixture)
		}
		if err := testCase.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c GitHubCase) validate() error {
	switch c.Invocation.Kind {
	case "cli":
		if len(c.Invocation.Args) == 0 || c.Invocation.Operation != "" {
			return fmt.Errorf("GitHub conformance case %s has invalid CLI invocation", c.ID)
		}
	case "manager":
		if strings.TrimSpace(c.Invocation.Operation) == "" || len(c.Invocation.Args) != 0 {
			return fmt.Errorf("GitHub conformance case %s has invalid manager invocation", c.ID)
		}
	default:
		return fmt.Errorf("GitHub conformance case %s has invalid invocation kind %q", c.ID, c.Invocation.Kind)
	}
	for _, confirmation := range []string{c.Safety.BaselineConfirmation, c.Safety.TargetConfirmation} {
		if confirmation != "not-required" && confirmation != "required" && confirmation != "confirmed" {
			return fmt.Errorf("GitHub conformance case %s has invalid confirmation %q", c.ID, confirmation)
		}
	}
	if c.Safety.PartialFailure && !c.Safety.Mutation {
		return fmt.Errorf("GitHub conformance case %s marks partial failure without mutation", c.ID)
	}
	if c.Expected.Format != "json" && c.Expected.Format != "text" && c.Expected.Format != "none" {
		return fmt.Errorf("GitHub conformance case %s has invalid output format %q", c.ID, c.Expected.Format)
	}
	if c.Expected.StdoutComparison != "exact" && c.Expected.StdoutComparison != "structural" {
		return fmt.Errorf("GitHub conformance case %s has invalid stdout comparison %q", c.ID, c.Expected.StdoutComparison)
	}
	for _, asset := range []string{c.Expected.Stdout, c.Expected.Stderr, c.Expected.Files} {
		if _, err := ReadGitHubAsset(asset); err != nil {
			return fmt.Errorf("GitHub conformance case %s: %w", c.ID, err)
		}
	}
	if c.Expected.Format == "json" {
		if err := validateJSONAsset(c.Expected.Stdout); err != nil {
			return fmt.Errorf("GitHub conformance case %s stdout: %w", c.ID, err)
		}
	}
	if _, err := ReadProviderTranscript(c.Expected.ProviderTranscript); err != nil {
		return fmt.Errorf("GitHub conformance case %s: %w", c.ID, err)
	}
	if c.Expected.GitHubMetadata != "unchanged" {
		if err := validateJSONAsset(c.Expected.GitHubMetadata); err != nil {
			return fmt.Errorf("GitHub conformance case %s metadata: %w", c.ID, err)
		}
	}
	for _, identity := range c.Expected.RemoteIdentities {
		if identity.Provider != "github" || identity.Kind == "" || identity.OpaqueID == "" || identity.DisplayID == "" || identity.URL == "" {
			return fmt.Errorf("GitHub conformance case %s has incomplete remote identity %#v", c.ID, identity)
		}
	}
	if c.Rerun != nil {
		if c.Rerun.Kind != "identical" || len(c.Rerun.Classifications) == 0 {
			return fmt.Errorf("GitHub conformance case %s has incomplete rerun contract", c.ID)
		}
		for _, asset := range []string{c.Rerun.Stdout, c.Rerun.Stderr, c.Rerun.Files} {
			if _, err := ReadGitHubAsset(asset); err != nil {
				return fmt.Errorf("GitHub conformance case %s rerun: %w", c.ID, err)
			}
		}
		if c.Expected.Format == "json" {
			if err := validateJSONAsset(c.Rerun.Stdout); err != nil {
				return fmt.Errorf("GitHub conformance case %s rerun stdout: %w", c.ID, err)
			}
		}
		if _, err := ReadProviderTranscript(c.Rerun.ProviderTranscript); err != nil {
			return fmt.Errorf("GitHub conformance case %s rerun: %w", c.ID, err)
		}
		if c.Rerun.BaselineMutation && len(c.IntentionalDifferences) == 0 {
			return fmt.Errorf("GitHub conformance case %s has an undocumented mutating baseline rerun", c.ID)
		}
		if c.Rerun.GitHubMetadata != "unchanged" {
			if err := validateJSONAsset(c.Rerun.GitHubMetadata); err != nil {
				return fmt.Errorf("GitHub conformance case %s rerun metadata: %w", c.ID, err)
			}
		}
	}
	return nil
}

func validateJSONAsset(name string) error {
	raw, err := ReadGitHubAsset(name)
	if err != nil {
		return err
	}
	if !json.Valid(raw) {
		return fmt.Errorf("GitHub conformance asset %s is not valid JSON", name)
	}
	return nil
}

func validGitHubAssetName(name string) bool {
	clean := path.Clean(name)
	return clean == name && strings.HasPrefix(clean, "github-v1/") && !strings.Contains(clean, "..")
}

func validGitHubFixtureName(name string) bool {
	return validGitHubAssetName(name) && strings.HasPrefix(name, "github-v1/fixtures/")
}
