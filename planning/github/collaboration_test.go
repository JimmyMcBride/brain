package github

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/planning/application"
)

type discussionRunner struct {
	t                *testing.T
	body, cursor     string
	calls, mutations int
	fail             bool
	repeatCursor     bool
}

func (r *discussionRunner) Run(ctx context.Context, root string, args ...string) (RunResult, error) {
	r.calls++
	if err := ctx.Err(); err != nil {
		return RunResult{}, err
	}
	if r.fail {
		return RunResult{Stderr: []byte("HTTP 401 secret-token")}, errors.New("secret-token")
	}
	if len(args) < 2 || args[0] != "api" || args[1] != "graphql" {
		r.t.Fatalf("unexpected args: %q", args)
	}
	values := map[string]string{}
	for i := 2; i+1 < len(args); i += 2 {
		pair := strings.SplitN(args[i+1], "=", 2)
		if len(pair) != 2 {
			r.t.Fatal(args)
		}
		values[pair[0]] = pair[1]
	}
	item := map[string]any{"id": "D_49", "number": 49, "url": "https://github.com/owner/repo/discussions/49", "title": "Planning", "body": r.body, "updatedAt": "2026-09-06T00:00:00Z"}
	var response any
	if strings.HasPrefix(values["query"], "mutation") {
		r.mutations++
		if values["id"] != "D_49" {
			r.t.Fatal(values)
		}
		r.body = values["body"]
		item["body"] = r.body
		response = map[string]any{"data": map[string]any{"updateDiscussion": map[string]any{"discussion": item}}}
	} else {
		if values["owner"] != "owner" || values["name"] != "repo" || values["number"] != "49" {
			r.t.Fatal(values)
		}
		hasNext := r.cursor != "" && (values["after"] == "" || r.repeatCursor)
		item["comments"] = map[string]any{"nodes": []any{map[string]any{"body": "comment " + values["after"]}}, "pageInfo": map[string]any{"hasNextPage": hasNext, "endCursor": r.cursor}}
		response = map[string]any{"data": map[string]any{"repository": map[string]any{"discussion": item}}}
	}
	raw, _ := json.Marshal(response)
	return RunResult{Stdout: raw}, nil
}

func discussionReference() application.ExternalReference {
	return application.ExternalReference{Provider: "github", Kind: "discussion", URL: "https://github.com/owner/repo/discussions/49"}
}

func TestDiscussionReadPaginationIdentityAndRedaction(t *testing.T) {
	runner := &discussionRunner{t: t, body: "source", cursor: "next"}
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: filepath.Join(t.TempDir(), "project with spaces"), Runner: runner})
	snapshot, err := adapter.CollaborationSource().Read(context.Background(), discussionReference())
	if err != nil || len(snapshot.Contributions) != 2 || snapshot.Source.Revision == "" || runner.calls != 2 {
		t.Fatalf("snapshot=%+v calls=%d err=%v", snapshot, runner.calls, err)
	}
	runner.repeatCursor = true
	_, err = adapter.CollaborationSource().Read(context.Background(), discussionReference())
	if err == nil {
		t.Fatal("accepted cyclic pagination")
	}
	runner.fail = true
	_, err = adapter.CollaborationSource().Read(context.Background(), discussionReference())
	var integration *application.IntegrationError
	if !errors.As(err, &integration) || integration.Class != application.IntegrationUnauthenticated || strings.Contains(err.Error(), "secret-token") {
		t.Fatal(err)
	}
	runner.fail = false
	ref := discussionReference()
	ref.OpaqueID = "wrong"
	_, err = adapter.CollaborationSource().Read(context.Background(), ref)
	if !errors.As(err, &integration) || integration.Class != application.IntegrationAmbiguousIdentity {
		t.Fatal(err)
	}
}

func TestDiscussionRepairGuardsRevisionAndPreservesArguments(t *testing.T) {
	runner := &discussionRunner{t: t, body: "original"}
	root := filepath.Join(t.TempDir(), "untouched")
	adapter := New(Config{Enabled: true}, Options{ProjectRoot: root, Runner: runner})
	port := adapter.CollaborationSource()
	snapshot, err := port.Read(context.Background(), discussionReference())
	if err != nil {
		t.Fatal(err)
	}
	request := application.CollaborationRepairRequest{Source: snapshot.Source, ExpectedRevision: snapshot.Source.Revision, Content: "new\n$(literal) `literal` @literal"}
	runner.body = "concurrent edit"
	_, err = port.Repair(context.Background(), request)
	var integration *application.IntegrationError
	if !errors.As(err, &integration) || integration.Class != application.IntegrationRevisionConflict || runner.mutations != 0 {
		t.Fatalf("mutations=%d err=%v", runner.mutations, err)
	}
	runner.body = "original"
	evidence, err := port.Repair(context.Background(), request)
	if err != nil || !evidence.Changed || runner.body != request.Content || runner.mutations != 1 {
		t.Fatalf("evidence=%+v err=%v", evidence, err)
	}
	snapshot, err = port.Read(context.Background(), discussionReference())
	if err != nil {
		t.Fatal(err)
	}
	request.ExpectedRevision = snapshot.Source.Revision
	evidence, err = port.Repair(context.Background(), request)
	if err != nil || evidence.Changed || runner.mutations != 1 {
		t.Fatalf("rerun=%+v err=%v", evidence, err)
	}
	if _, err = os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("remote repair touched workspace: %v", err)
	}
}

func TestDiscussionRejectsForeignURLsWithoutProviderWork(t *testing.T) {
	runner := &discussionRunner{t: t}
	adapter := New(Config{Enabled: true}, Options{Runner: runner})
	for _, value := range []string{"https://evil.example/owner/repo/discussions/49", "https://github.com/owner/repo/issues/49", "https://github.com/owner/repo/discussions/49?token=secret"} {
		ref := discussionReference()
		ref.URL = value
		if _, err := adapter.CollaborationSource().Read(context.Background(), ref); err == nil {
			t.Fatal(value)
		}
	}
	if runner.calls != 0 {
		t.Fatal(runner.calls)
	}
}
