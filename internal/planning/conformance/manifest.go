package conformance

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const manifestPath = "testdata/manifest.yaml"

var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

//go:embed all:testdata
var testdata embed.FS

type Manifest struct {
	SchemaVersion int           `yaml:"schema_version"`
	Baseline      Baseline      `yaml:"baseline"`
	Normalization Normalization `yaml:"normalization"`
	Commands      []Command     `yaml:"commands"`
	Cases         []Case        `yaml:"cases"`
}

type Baseline struct {
	Repository      string `yaml:"repository"`
	Revision        string `yaml:"revision"`
	Command         string `yaml:"command"`
	WorkspaceSchema int    `yaml:"workspace_schema"`
}

type Normalization struct {
	LineEndings      string `yaml:"line_endings"`
	PathSeparator    string `yaml:"path_separator"`
	ProjectRootToken string `yaml:"project_root_token"`
}

type Command struct {
	ID          string   `yaml:"id"`
	Standalone  []string `yaml:"standalone"`
	Native      []string `yaml:"native,omitempty"`
	Disposition string   `yaml:"disposition"`
	TargetPhase string   `yaml:"target_phase"`
	Behavior    string   `yaml:"behavior"`
	Rationale   string   `yaml:"rationale"`
}

type Case struct {
	ID       string     `yaml:"id"`
	Command  string     `yaml:"command"`
	Fixture  string     `yaml:"fixture"`
	Args     []string   `yaml:"args"`
	Expected Expected   `yaml:"expected"`
	Compare  Comparison `yaml:"compare"`
}

type Expected struct {
	ExitCode int    `yaml:"exit_code"`
	Stdout   string `yaml:"stdout"`
	Stderr   string `yaml:"stderr"`
	Files    string `yaml:"files"`
	Rerun    string `yaml:"rerun"`
}

type Comparison struct {
	ExitCode string `yaml:"exit_code"`
	Stdout   string `yaml:"stdout"`
	Stderr   string `yaml:"stderr"`
	Files    string `yaml:"files"`
	Rerun    string `yaml:"rerun"`
}

func Load() (Manifest, error) {
	raw, err := testdata.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("read conformance manifest: %w", err)
	}
	var manifest Manifest
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse conformance manifest: %w", err)
	}
	if err := manifest.validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func ReadGolden(name string) (string, error) {
	if name == "empty" {
		return "", nil
	}
	if !validAssetName(name, "golden") {
		return "", fmt.Errorf("invalid golden reference %q", name)
	}
	raw, err := testdata.ReadFile(path.Join("testdata", name))
	if err != nil {
		return "", fmt.Errorf("read conformance golden %s: %w", name, err)
	}
	return string(raw), nil
}

func Fixture(name string) (fs.FS, error) {
	if name == "empty" {
		return emptyFS{}, nil
	}
	if !validAssetName(name, "fixtures") {
		return nil, fmt.Errorf("invalid fixture reference %q", name)
	}
	fixturePath := path.Join("testdata", name)
	info, err := fs.Stat(testdata, fixturePath)
	if err != nil {
		return nil, fmt.Errorf("open conformance fixture %s: %w", name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("conformance fixture %s is not a directory", name)
	}
	fixture, err := fs.Sub(testdata, fixturePath)
	if err != nil {
		return nil, fmt.Errorf("open conformance fixture %s: %w", name, err)
	}
	return fixture, nil
}

func (m Manifest) validate() error {
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported conformance manifest schema %d", m.SchemaVersion)
	}
	if m.Baseline.Repository != "https://github.com/JimmyMcBride/plan" {
		return fmt.Errorf("unexpected baseline repository %q", m.Baseline.Repository)
	}
	if !revisionPattern.MatchString(m.Baseline.Revision) {
		return fmt.Errorf("baseline revision %q must be a full lowercase commit SHA", m.Baseline.Revision)
	}
	if m.Baseline.Command != "plan" {
		return fmt.Errorf("unexpected baseline command %q", m.Baseline.Command)
	}
	if m.Baseline.WorkspaceSchema != 3 {
		return fmt.Errorf("unexpected baseline workspace schema %d", m.Baseline.WorkspaceSchema)
	}
	if m.Normalization.LineEndings != "lf" ||
		m.Normalization.PathSeparator != "/" ||
		m.Normalization.ProjectRootToken != "<PROJECT>" {
		return fmt.Errorf("unsupported conformance normalization: %#v", m.Normalization)
	}

	commandIDs := make(map[string]struct{}, len(m.Commands))
	for _, command := range m.Commands {
		if strings.TrimSpace(command.ID) == "" {
			return fmt.Errorf("conformance command id is required")
		}
		if _, exists := commandIDs[command.ID]; exists {
			return fmt.Errorf("duplicate conformance command %q", command.ID)
		}
		commandIDs[command.ID] = struct{}{}
		if len(command.Standalone) == 0 {
			return fmt.Errorf("conformance command %s has no standalone invocation", command.ID)
		}
		if strings.TrimSpace(command.Behavior) == "" || strings.TrimSpace(command.Rationale) == "" {
			return fmt.Errorf("conformance command %s is missing behavior or rationale", command.ID)
		}
		switch command.Disposition {
		case "shared":
			if command.TargetPhase != "phase-4" || len(command.Native) == 0 {
				return fmt.Errorf("shared conformance command %s must map to a Phase 4 native invocation", command.ID)
			}
		case "deferred", "redesign", "retire", "transitional":
			if strings.TrimSpace(command.TargetPhase) == "" {
				return fmt.Errorf("conformance command %s has no target phase", command.ID)
			}
		default:
			return fmt.Errorf("conformance command %s has invalid disposition %q", command.ID, command.Disposition)
		}
	}

	caseIDs := make(map[string]struct{}, len(m.Cases))
	for _, testCase := range m.Cases {
		if strings.TrimSpace(testCase.ID) == "" {
			return fmt.Errorf("conformance case id is required")
		}
		if _, exists := caseIDs[testCase.ID]; exists {
			return fmt.Errorf("duplicate conformance case %q", testCase.ID)
		}
		caseIDs[testCase.ID] = struct{}{}
		if _, exists := commandIDs[testCase.Command]; !exists {
			return fmt.Errorf("conformance case %s references unknown command %s", testCase.ID, testCase.Command)
		}
		if len(testCase.Args) == 0 {
			return fmt.Errorf("conformance case %s has no arguments", testCase.ID)
		}
		if _, err := Fixture(testCase.Fixture); err != nil {
			return fmt.Errorf("conformance case %s: %w", testCase.ID, err)
		}
		if _, err := ReadGolden(testCase.Expected.Stdout); err != nil {
			return fmt.Errorf("conformance case %s stdout: %w", testCase.ID, err)
		}
		if _, err := ReadGolden(testCase.Expected.Stderr); err != nil {
			return fmt.Errorf("conformance case %s stderr: %w", testCase.ID, err)
		}
		if testCase.Expected.Files != "unchanged" || testCase.Expected.Rerun != "same" {
			return fmt.Errorf("initial conformance case %s must preserve files and rerun behavior", testCase.ID)
		}
		for field, mode := range map[string]string{
			"exit_code": testCase.Compare.ExitCode,
			"stdout":    testCase.Compare.Stdout,
			"stderr":    testCase.Compare.Stderr,
			"files":     testCase.Compare.Files,
			"rerun":     testCase.Compare.Rerun,
		} {
			if mode != "exact" && mode != "structural" {
				return fmt.Errorf("conformance case %s has invalid %s comparison %q", testCase.ID, field, mode)
			}
		}
	}
	return nil
}

func validAssetName(name, directory string) bool {
	clean := path.Clean(name)
	return clean == name && strings.HasPrefix(clean, directory+"/") && !strings.Contains(clean, "..")
}

type emptyFS struct{}

func (emptyFS) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}
