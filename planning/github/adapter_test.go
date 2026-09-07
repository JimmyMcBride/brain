package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/planning/application"
)

type failingRunner struct{ calls int }

func (r *failingRunner) Run(context.Context, string, ...string) (RunResult, error) {
	r.calls++
	return RunResult{}, errors.New("runner must not be called")
}

type failingStateStore struct {
	reads  int
	writes int
}

func (s *failingStateStore) read() (githubState, error) {
	s.reads++
	return githubState{}, errors.New("state must not be read")
}

func (s *failingStateStore) write(githubState) error {
	s.writes++
	return errors.New("state must not be written")
}

func TestDisabledAdapterPerformsNoProviderOrMetadataWork(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "project with spaces")
	runner := &failingRunner{}
	state := &failingStateStore{}
	adapter := New(Config{}, Options{ProjectRoot: root, Runner: runner})
	adapter.state = state

	operations := []func() error{
		func() error {
			_, err := adapter.CollaborationSource().Read(context.Background(), application.ExternalReference{})
			return err
		},
		func() error {
			_, err := adapter.CollaborationSource().Repair(context.Background(), application.CollaborationRepairRequest{})
			return err
		},
		func() error {
			_, err := adapter.PublicationTarget().Inspect(context.Background(), application.PublicationInspectRequest{})
			return err
		},
		func() error {
			_, err := adapter.PublicationTarget().Apply(context.Background(), application.PublicationPlan{})
			return err
		},
		func() error { _, err := adapter.RepositoryEvidenceSource().Current(context.Background()); return err },
		func() error { _, err := adapter.ExternalMappingRepository().Load(context.Background()); return err },
		func() error {
			_, err := adapter.ExternalMappingRepository().Save(context.Background(), application.ExternalMappingState{}, "revision")
			return err
		},
		func() error {
			_, err := adapter.ExecutionWorkspace().Inspect(context.Background(), application.ExternalReference{})
			return err
		},
		func() error {
			_, err := adapter.ExecutionWorkspace().Attach(context.Background(), application.ExecutionAttachRequest{})
			return err
		},
		func() error {
			_, err := adapter.ExecutionWorkspace().ApplyStatus(context.Background(), application.ExecutionStatusRequest{})
			return err
		},
	}
	for _, operation := range operations {
		err := operation()
		var integrationError *application.IntegrationError
		if !errors.As(err, &integrationError) || integrationError.Class != application.IntegrationAdapterDisabled {
			t.Fatalf("disabled operation error = %#v", err)
		}
	}
	if runner.calls != 0 || state.reads != 0 || state.writes != 0 {
		t.Fatalf("disabled adapter performed work: runner=%d reads=%d writes=%d", runner.calls, state.reads, state.writes)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("constructing disabled adapter touched project root: %v", err)
	}
}

func TestEnabledSkeletonReportsUnsupportedCapability(t *testing.T) {
	t.Parallel()
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: t.TempDir(), Runner: &failingRunner{}})
	_, err := adapter.RepositoryEvidenceSource().Current(context.Background())
	var integrationError *application.IntegrationError
	if !errors.As(err, &integrationError) || integrationError.Class != application.IntegrationUnsupportedCapability {
		t.Fatalf("enabled skeleton error = %#v", err)
	}
}

func TestCapabilityViewsImplementApplicationPorts(t *testing.T) {
	t.Parallel()
	adapter := New(Config{}, Options{})
	var _ application.CollaborationSource = adapter.CollaborationSource()
	var _ application.PublicationTarget = adapter.PublicationTarget()
	var _ application.RepositoryEvidenceSource = adapter.RepositoryEvidenceSource()
	var _ application.ExternalMappingRepository = adapter.ExternalMappingRepository()
	var _ application.ExecutionWorkspace = adapter.ExecutionWorkspace()
}

func TestCommandRunnerPreservesArgumentsOutputAndProjectRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project with spaces")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	runner := commandRunnerWithExecutable{
		executable:  executable,
		environment: append(os.Environ(), "BRAIN_GITHUB_RUNNER_HELPER=1"),
	}
	result, err := runner.Run(
		context.Background(),
		root,
		"-test.run=TestCommandRunnerHelperProcess",
		"--",
		"value with spaces",
		"$(not-executed)",
	)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Directory string   `json:"directory"`
		Arguments []string `json:"arguments"`
	}
	if err := json.Unmarshal(result.Stdout, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Directory != root {
		t.Fatalf("runner directory = %q, want %q", payload.Directory, root)
	}
	if want := []string{"value with spaces", "$(not-executed)"}; !slices.Equal(payload.Arguments, want) {
		t.Fatalf("runner arguments = %q, want %q", payload.Arguments, want)
	}
	if got := string(result.Stderr); got != "provider stderr" {
		t.Fatalf("runner stderr = %q", got)
	}
}

func TestCommandRunnerHonorsCanceledContext(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = (commandRunnerWithExecutable{executable: executable}).Run(ctx, t.TempDir(), "-test.run=TestCommandRunnerHelperProcess")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled runner error = %v", err)
	}
}

func TestCommandRunnerHelperProcess(t *testing.T) {
	if os.Getenv("BRAIN_GITHUB_RUNNER_HELPER") != "1" {
		return
	}
	directory, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	separator := slices.Index(os.Args, "--")
	arguments := []string{}
	if separator >= 0 {
		arguments = os.Args[separator+1:]
	}
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"directory": directory, "arguments": arguments, "input": string(input)}); err != nil {
		panic(err)
	}
	if _, err := fmt.Fprint(os.Stderr, "provider stderr"); err != nil {
		panic(err)
	}
	os.Exit(0)
}

func TestCommandRunnerPreservesLargeStdin(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	runner := commandRunnerWithExecutable{executable: executable, environment: append(os.Environ(), "BRAIN_GITHUB_RUNNER_HELPER=1")}
	input := strings.Repeat("complete brief $(no shell) @file\n", 3000)
	result, err := runner.RunInput(context.Background(), t.TempDir(), []byte(input), "-test.run=TestCommandRunnerHelperProcess", "--", "--input", "-")
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Input     string
		Arguments []string
	}
	if err := json.Unmarshal(result.Stdout, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Input != input || !slices.Equal(payload.Arguments, []string{"--input", "-"}) {
		t.Fatal("stdin content changed or leaked into arguments")
	}
}

func TestMetadataCompatibilityFixturesRoundTrip(t *testing.T) {
	t.Parallel()
	fixtures := []string{
		"enabled.github.json",
		"adopted.github.json",
		"promoted.github.json",
		"reconciled.github.json",
	}
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture, func(t *testing.T) {
			t.Parallel()
			raw := readConformanceMetadata(t, fixture)
			root := t.TempDir()
			store := newFileStateStore(root)
			if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(store.path, raw, 0o644); err != nil {
				t.Fatal(err)
			}
			state, err := store.read()
			if err != nil {
				t.Fatal(err)
			}
			if err := store.write(state); err != nil {
				t.Fatal(err)
			}
			written, err := os.ReadFile(store.path)
			if err != nil {
				t.Fatal(err)
			}
			assertEquivalentJSON(t, written, raw)
			matches, err := filepath.Glob(filepath.Join(filepath.Dir(store.path), ".github.json.tmp-*"))
			if err != nil {
				t.Fatal(err)
			}
			if len(matches) != 0 {
				t.Fatalf("metadata write left temporary files: %v", matches)
			}
		})
	}
}

func TestMetadataStateContainsNoCredentialFields(t *testing.T) {
	t.Parallel()
	for _, value := range []any{githubState{}, githubPlanningRecord{}, githubProjectDecision{}, githubStoryRecord{}} {
		typeOf := reflect.TypeOf(value)
		for i := range typeOf.NumField() {
			field := typeOf.Field(i)
			name := strings.ToLower(field.Name + " " + field.Tag.Get("json"))
			for _, forbidden := range []string{"token", "password", "secret", "credential"} {
				if strings.Contains(name, forbidden) {
					t.Errorf("credential field %s.%s", typeOf.Name(), field.Name)
				}
			}
		}
	}
}

func readConformanceMetadata(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "internal", "planning", "conformance", "testdata", "github-v1", "golden", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func assertEquivalentJSON(t *testing.T, got, want []byte) {
	t.Helper()
	var gotValue any
	var wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("decode written JSON: %v", err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("decode fixture JSON: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("metadata changed during round trip\ngot:  %s\nwant: %s", got, want)
	}
}
