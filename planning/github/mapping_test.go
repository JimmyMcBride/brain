package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"
)

func mappingAdapter(t *testing.T, fixture string) (*Adapter, *failingRunner, *fileStateStore) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "project with spaces")
	store := newFileStateStore(root)
	if fixture != "" {
		if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
			t.Fatal(err)
		}
		raw := bytes.ReplaceAll(readConformanceMetadata(t, fixture), []byte("<TIMESTAMP>"), []byte("2026-09-07T00:00:00Z"))
		if err := os.WriteFile(store.path, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runner := &failingRunner{}
	return New(Config{Enabled: true}, Options{ProjectRoot: root, Runner: runner}), runner, store
}

func TestExternalMappingLoadIsLocalDeterministicAndNonMutating(t *testing.T) {
	adapter, runner, store := mappingAdapter(t, "promoted.github.json")
	before, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	first, err := adapter.ExternalMappingRepository().Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.ExternalMappingRepository().Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || first.SchemaVersion != 1 || first.Revision == "" || len(first.ArtifactReferences) != 3 || len(first.SourceReferences) != 1 || len(first.GroupReferences) != 3 || len(first.WorkspaceReferences) != 0 {
		t.Fatalf("mapping view = %+v", first)
	}
	for _, mapping := range first.ArtifactReferences {
		if mapping.Reference.OpaqueID != mapping.Reference.URL || mapping.Reference.Provider != providerName || mapping.Reference.Kind != "issue" {
			t.Fatal("legacy identity was not normalized", mapping)
		}
	}
	after, err := os.ReadFile(store.path)
	if err != nil || string(before) != string(after) || runner.calls != 0 {
		t.Fatal("mapping load performed work", err, runner.calls)
	}
}

func TestExternalMappingLoadMissingDoesNotCreateWorkspace(t *testing.T) {
	adapter, runner, _ := mappingAdapter(t, "")
	state, err := adapter.ExternalMappingRepository().Load(context.Background())
	if err != nil || state.SchemaVersion != 1 || state.Revision != "" || len(state.ArtifactReferences) != 0 || runner.calls != 0 {
		t.Fatalf("state=%+v calls=%d err=%v", state, runner.calls, err)
	}
	if _, err := os.Stat(adapter.projectRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("load created project root", err)
	}
}

func TestExternalMappingSavePreservesLegacyMetadata(t *testing.T) {
	for _, fixture := range []string{"promoted.github.json", "adopted.github.json", "reconciled.github.json"} {
		t.Run(fixture, func(t *testing.T) {
			adapter, runner, store := mappingAdapter(t, fixture)
			beforeRaw, err := os.ReadFile(store.path)
			if err != nil {
				t.Fatal(err)
			}
			before, err := store.read()
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := adapter.ExternalMappingRepository().Load(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			saved, err := adapter.ExternalMappingRepository().Save(context.Background(), loaded, loaded.Revision)
			if err != nil || saved.Revision == "" || saved.Revision != loaded.Revision {
				t.Fatalf("saved=%+v err=%v", saved, err)
			}
			after, err := store.read()
			if err != nil {
				t.Fatal(err)
			}
			before.LastUpdatedAt, after.LastUpdatedAt = "", ""
			if !reflect.DeepEqual(before, after) || runner.calls != 0 {
				t.Fatalf("legacy metadata changed\nbefore=%+v\nafter=%+v", before, after)
			}
			afterRaw, err := os.ReadFile(store.path)
			if err != nil || !bytes.Equal(beforeRaw, afterRaw) {
				t.Fatal("unchanged mappings rewrote metadata", err)
			}
			if matches, _ := filepath.Glob(store.path + ".*"); len(matches) != 0 {
				t.Fatalf("temporary mapping files remain: %v", matches)
			}
		})
	}
}

func TestExternalMappingCreateRoundTripAndCompleteReplacement(t *testing.T) {
	adapter, _, store := mappingAdapter(t, "")
	sourceURL := "https://github.com/owner/repo/discussions/49"
	issueURL := "https://github.com/owner/repo/issues/10"
	milestoneURL := "https://github.com/owner/repo/milestone/2"
	artifact := planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: "spec"}
	desired := application.ExternalMappingState{SchemaVersion: 1,
		ArtifactReferences: []application.ArtifactExternalReference{{Artifact: artifact, Reference: application.ExternalReference{Provider: providerName, Kind: "issue", OpaqueID: "I_node", DisplayID: "10", URL: issueURL}}},
		SourceReferences:   []application.ExternalReference{{Provider: providerName, Kind: "discussion", OpaqueID: "D_node", DisplayID: "49", URL: sourceURL}},
		GroupReferences:    []application.ArtifactExternalReference{{Artifact: artifact, Reference: application.ExternalReference{Provider: providerName, Kind: "milestone", OpaqueID: "M_2", DisplayID: "2", URL: milestoneURL}}},
	}
	saved, err := adapter.ExternalMappingRepository().Save(context.Background(), desired, "")
	if err != nil || saved.Revision == "" {
		t.Fatal(saved, err)
	}
	loaded, err := adapter.ExternalMappingRepository().Load(context.Background())
	if err != nil || !reflect.DeepEqual(saved, loaded) {
		t.Fatalf("round trip = %+v %+v %v", saved, loaded, err)
	}
	legacy, err := store.read()
	if err != nil {
		t.Fatal(err)
	}
	record := legacy.Planning["spec"]
	if legacy.Repo != "owner/repo" || legacy.RepoURL != "https://github.com/owner/repo" || record.IssueNumber != 10 || record.DiscussionNumber != 49 || record.MilestoneNumber != 2 || record.OwnershipMode != "github" || record.EntryMode != "github_discussion" {
		t.Fatalf("legacy record = %+v state=%+v", record, legacy)
	}
	saved.ArtifactReferences = nil
	saved.SourceReferences = nil
	saved.GroupReferences = nil
	replaced, err := adapter.ExternalMappingRepository().Save(context.Background(), saved, saved.Revision)
	if err != nil || len(replaced.ArtifactReferences) != 0 {
		t.Fatal(replaced, err)
	}
	legacy, _ = store.read()
	if len(legacy.Planning) != 0 {
		t.Fatal("complete replacement retained removed mappings")
	}
}

func TestExternalMappingRevisionConflictAndConcurrentWriters(t *testing.T) {
	adapterA, _, store := mappingAdapter(t, "promoted.github.json")
	adapterB := New(Config{Enabled: true}, Options{ProjectRoot: adapterA.projectRoot, Runner: &failingRunner{}})
	a, err := adapterA.ExternalMappingRepository().Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	b, err := adapterB.ExternalMappingRepository().Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a.LastReconciliation = &application.ReconciliationEvidence{Repository: application.RepositoryEvidence{SchemaVersion: 1, Repository: application.ExternalReference{Provider: providerName, Kind: "repository", OpaqueID: "JimmyMcBride/plan", URL: "https://github.com/JimmyMcBride/plan"}}, CompletedAt: time.Unix(1, 0).UTC()}
	aSaved, err := adapterA.ExternalMappingRepository().Save(context.Background(), a, a.Revision)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(store.path)
	if _, err := adapterB.ExternalMappingRepository().Save(context.Background(), b, b.Revision); err == nil {
		t.Fatal("stale writer replaced mappings")
	} else {
		var integration *application.IntegrationError
		if !errors.As(err, &integration) || integration.Class != application.IntegrationRevisionConflict {
			t.Fatal(err)
		}
	}
	after, _ := os.ReadFile(store.path)
	if string(before) != string(after) || aSaved.Revision != mappingRevision(after) {
		t.Fatal("conflict changed mapping file")
	}
}

func TestExternalMappingLockCancellationAndCleanup(t *testing.T) {
	adapter, _, store := mappingAdapter(t, "promoted.github.json")
	loaded, err := adapter.ExternalMappingRepository().Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.path+".lock", []byte("held"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := adapter.ExternalMappingRepository().Save(ctx, loaded, loaded.Revision); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := os.Remove(store.path + ".lock"); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.ExternalMappingRepository().Save(context.Background(), loaded, loaded.Revision); err != nil {
		t.Fatal("released lock prevented save", err)
	}
}

func TestExternalMappingRejectsAmbiguousAndUnsupportedChanges(t *testing.T) {
	adapter, _, store := mappingAdapter(t, "promoted.github.json")
	loaded, err := adapter.ExternalMappingRepository().Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(store.path)
	cases := map[string]func(*application.ExternalMappingState){
		"duplicate artifact": func(state *application.ExternalMappingState) {
			state.ArtifactReferences = append(state.ArtifactReferences, state.ArtifactReferences[0])
		},
		"duplicate issue": func(state *application.ExternalMappingState) {
			state.ArtifactReferences[1].Reference = state.ArtifactReferences[0].Reference
		},
		"multiple sources": func(state *application.ExternalMappingState) {
			state.SourceReferences = append(state.SourceReferences, application.ExternalReference{Provider: providerName, Kind: "discussion", OpaqueID: "D", DisplayID: "50", URL: "https://github.com/JimmyMcBride/plan/discussions/50"})
		},
		"source without artifact": func(state *application.ExternalMappingState) {
			state.ArtifactReferences = nil
			state.GroupReferences = nil
		},
		"noncanonical milestone display ID": func(state *application.ExternalMappingState) {
			state.GroupReferences[0].Reference.DisplayID = "07"
		},
		"new workspace": func(state *application.ExternalMappingState) {
			state.WorkspaceReferences = append(state.WorkspaceReferences, application.ExternalReference{Provider: providerName, Kind: "project", OpaqueID: "P", DisplayID: "1", URL: "https://github.com/users/JimmyMcBride/projects/1"})
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			state := loaded
			state.ArtifactReferences = slices.Clone(loaded.ArtifactReferences)
			state.SourceReferences = slices.Clone(loaded.SourceReferences)
			state.GroupReferences = slices.Clone(loaded.GroupReferences)
			state.WorkspaceReferences = slices.Clone(loaded.WorkspaceReferences)
			mutate(&state)
			if _, err := adapter.ExternalMappingRepository().Save(context.Background(), state, loaded.Revision); err == nil {
				t.Fatal("accepted invalid mapping change")
			}
			after, _ := os.ReadFile(store.path)
			if string(before) != string(after) {
				t.Fatal("invalid change mutated metadata")
			}
		})
	}
}

func TestExternalMappingRejectsCorruptIdentity(t *testing.T) {
	adapter, _, store := mappingAdapter(t, "promoted.github.json")
	state, err := store.read()
	if err != nil {
		t.Fatal(err)
	}
	state.RepoURL = "https://github.com/other/repo"
	raw, _ := json.Marshal(state)
	if err := os.WriteFile(store.path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = adapter.ExternalMappingRepository().Load(context.Background())
	var integration *application.IntegrationError
	if !errors.As(err, &integration) || integration.Class != application.IntegrationAmbiguousIdentity {
		t.Fatal(err)
	}
}
