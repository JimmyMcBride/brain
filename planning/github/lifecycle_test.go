package github

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/JimmyMcBride/brain/planning/application"
)

type preflightRunner struct {
	calls [][]string
	err   error
}

func (r *preflightRunner) Run(_ context.Context, _ string, args ...string) (RunResult, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	return RunResult{Stdout: []byte(`{"nameWithOwner":"owner/repo","url":"https://github.com/owner/repo","defaultBranchRef":{"name":"develop"},"hasIssuesEnabled":true}`)}, r.err
}

func TestEnableInitializesAndPreservesAdapterState(t *testing.T) {
	root := t.TempDir()
	runner := &preflightRunner{}
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: root, Runner: runner, Now: func() time.Time { return time.Unix(1, 0) }})
	status, err := adapter.Enable(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Action != application.MutationCreate || status.Repository.OpaqueID != "owner/repo" || status.DefaultBranch != "develop" {
		t.Fatalf("status=%+v", status)
	}
	state, err := adapter.state.read()
	if err != nil {
		t.Fatal(err)
	}
	state.Planning["spec"] = githubPlanningRecord{Slug: "spec", Kind: "spec", IssueNumber: 1}
	if err := adapter.state.write(state); err != nil {
		t.Fatal(err)
	}
	status, err = adapter.Enable(context.Background())
	if err != nil || status.Action != application.MutationUnchanged {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	preserved, err := adapter.state.read()
	if err != nil || !reflect.DeepEqual(preserved.Planning, state.Planning) {
		t.Fatalf("preserved=%+v err=%v", preserved.Planning, err)
	}
	if err := adapter.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("provider calls=%d", len(runner.calls))
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(metadataRelativePath))); err != nil {
		t.Fatal(err)
	}
}

func TestHealthRequiresCompatibleInitializedState(t *testing.T) {
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: t.TempDir(), Runner: &preflightRunner{}})
	if err := adapter.Health(context.Background()); err == nil {
		t.Fatal("expected missing state health failure")
	}
}
