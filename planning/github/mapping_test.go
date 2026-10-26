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

func TestExternalMappingPersistsCanonicalWorkspaceAndPreservesPrivateDecisionFields(t *testing.T) {
	adapter, _, store := mappingAdapter(t, "promoted.github.json")
	loaded, err := adapter.ExternalMappingRepository().Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	project := application.ExternalReference{Provider: providerName, Kind: "project", OpaqueID: "PVT_new", DisplayID: "12", URL: "https://github.com/users/JimmyMcBride/projects/12"}
	loaded.WorkspaceReferences = []application.ExternalReference{project}
	saved, err := adapter.ExternalMappingRepository().Save(context.Background(), loaded, loaded.Revision)
	if err != nil || len(saved.WorkspaceReferences) != 1 || saved.WorkspaceReferences[0] != project {
		t.Fatalf("saved=%+v err=%v", saved, err)
	}
	state, err := store.read()
	if err != nil {
		t.Fatal(err)
	}
	record := state.ProjectDecisions["github-collaboration-foundation"]
	if record.Slug != "github-collaboration-foundation" || record.Decision != "connect" || record.InitiativeSlug != "github-collaboration-foundation" || record.ProjectOwner != "JimmyMcBride" || record.ProjectNumber != 12 || record.ProjectID != "PVT_new" || record.ProjectURL != project.URL {
		t.Fatalf("project decision=%+v", record)
	}

	adapter, _, store = mappingAdapter(t, "reconciled.github.json")
	raw, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if json.Unmarshal(raw, &document) != nil {
		t.Fatal("invalid fixture")
	}
	document["private_extension"] = map[string]any{"keep": true}
	decision := document["project_decisions"].(map[string]any)["github-adapter"].(map[string]any)
	decision["private_note"] = "keep me"
	raw, _ = json.MarshalIndent(document, "", "  ")
	if err := os.WriteFile(store.path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err = adapter.ExternalMappingRepository().Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	before, _ := store.read()
	replacement := application.ExternalReference{Provider: providerName, Kind: "project", OpaqueID: "PVT_replacement", DisplayID: "13", URL: "https://github.com/users/JimmyMcBride/projects/13"}
	loaded.WorkspaceReferences = []application.ExternalReference{replacement}
	if _, err := adapter.ExternalMappingRepository().Save(context.Background(), loaded, loaded.Revision); err != nil {
		t.Fatal(err)
	}
	after, _ := store.read()
	want, got := before.ProjectDecisions["github-adapter"], after.ProjectDecisions["github-adapter"]
	got.ProjectOwner, got.ProjectNumber, got.ProjectID, got.ProjectURL = want.ProjectOwner, want.ProjectNumber, want.ProjectID, want.ProjectURL
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("private decision fields changed\nbefore=%+v\nafter=%+v", want, after.ProjectDecisions["github-adapter"])
	}
	afterRaw, _ := os.ReadFile(store.path)
	var afterDocument map[string]any
	_ = json.Unmarshal(afterRaw, &afterDocument)
	if afterDocument["private_extension"].(map[string]any)["keep"] != true || afterDocument["project_decisions"].(map[string]any)["github-adapter"].(map[string]any)["private_note"] != "keep me" {
		t.Fatal("unknown metadata was not preserved")
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

func TestExternalMappingReconciliationEvidenceRoundTripsExactly(t *testing.T) {
	adapter, _, store := mappingAdapter(t, "promoted.github.json")
	loaded, err := adapter.ExternalMappingRepository().Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	evidence := application.RepositoryEvidence{
		SchemaVersion: 1,
		Repository:    application.ExternalReference{Provider: providerName, Kind: "repository", OpaqueID: "JimmyMcBride/plan", DisplayID: "JimmyMcBride/plan", URL: "https://github.com/JimmyMcBride/plan"},
		DefaultRef:    "develop", CurrentRef: "codex/github-adapter", Commit: "0123456789abcdef0123456789abcdef01234567",
		PlanningChangeRequest: &application.PlanningChangeRequestEvidence{
			Reference: application.ExternalReference{Provider: providerName, Kind: "pull_request", OpaqueID: "PR_node", DisplayID: "#41", URL: "https://github.com/JimmyMcBride/plan/pull/41", Revision: "0123456789abcdef0123456789abcdef01234567"},
			State:     application.ChangeRequestOpen, HeadRef: "codex/github-adapter", BaseRef: "develop", Draft: true,
		},
	}
	completed := time.Unix(5, 0).UTC()
	loaded.LastReconciliation = &application.ReconciliationEvidence{Repository: evidence, CompletedAt: completed}
	saved, err := adapter.ExternalMappingRepository().Save(context.Background(), loaded, loaded.Revision)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := adapter.ExternalMappingRepository().Load(context.Background())
	if err != nil || !reflect.DeepEqual(saved, reloaded) || reloaded.LastReconciliation == nil || !reflect.DeepEqual(reloaded.LastReconciliation.Repository, evidence) || !reloaded.LastReconciliation.CompletedAt.Equal(completed) {
		t.Fatalf("saved=%+v reloaded=%+v err=%v", saved.LastReconciliation, reloaded.LastReconciliation, err)
	}
	state, err := store.read()
	if err != nil || state.Reconciliation == nil || state.Reconciliation.PlanningChangeRequest == nil || state.DefaultBranch != evidence.DefaultRef {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	before, _ := os.ReadFile(store.path)
	unchanged, err := adapter.ExternalMappingRepository().Save(context.Background(), reloaded, reloaded.Revision)
	after, _ := os.ReadFile(store.path)
	if err != nil || unchanged.Revision != reloaded.Revision || !bytes.Equal(before, after) {
		t.Fatal("unchanged exact evidence rewrote metadata", err)
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
		"multiple workspaces": func(state *application.ExternalMappingState) {
			state.WorkspaceReferences = append(state.WorkspaceReferences,
				application.ExternalReference{Provider: providerName, Kind: "project", OpaqueID: "P1", DisplayID: "1", URL: "https://github.com/users/JimmyMcBride/projects/1"},
				application.ExternalReference{Provider: providerName, Kind: "project", OpaqueID: "P2", DisplayID: "2", URL: "https://github.com/users/JimmyMcBride/projects/2"})
		},
		"foreign workspace": func(state *application.ExternalMappingState) {
			state.WorkspaceReferences = []application.ExternalReference{{Provider: providerName, Kind: "project", OpaqueID: "P", DisplayID: "1", URL: "https://github.com/users/other/projects/1"}}
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
