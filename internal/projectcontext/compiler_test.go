package projectcontext

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestBuildBaseContractItems(t *testing.T) {
	project := t.TempDir()
	manager := New(t.TempDir())
	if _, err := manager.Install(context.Background(), Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	items, err := manager.BuildBaseContractItems(project)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(items), 5; got != want {
		t.Fatalf("unexpected base contract item count: got=%d want=%d", got, want)
	}

	gotIDs := []string{}
	for _, item := range items {
		gotIDs = append(gotIDs, item.ID)
		if item.Kind != ContextItemKindBaseContract {
			t.Fatalf("expected base contract kind, got %#v", item)
		}
		if item.Summary == "" || item.Anchor.Path == "" || item.SourceHash == "" || item.ExpansionCost == 0 {
			t.Fatalf("expected populated compiler item fields: %#v", item)
		}
		if words := len(strings.Fields(item.Summary)); words > 50 {
			t.Fatalf("expected compact base-contract summary, got %d words in %#v", words, item)
		}
	}

	wantIDs := []string{
		"base_boot_summary",
		"base_workflow_contract",
		"base_memory_update_rules",
		"base_architecture_summary",
		"base_verification_summary",
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("unexpected base contract order: got=%v want=%v", gotIDs, wantIDs)
	}

	second, err := manager.BuildBaseContractItems(project)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(items, second) {
		t.Fatalf("expected deterministic base contract items:\nfirst=%#v\nsecond=%#v", items, second)
	}
}

func TestBuildSourceSummaryItems(t *testing.T) {
	project := t.TempDir()
	manager := New(t.TempDir())
	if _, err := manager.Install(context.Background(), Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	items, err := manager.BuildSourceSummaryItems(project)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(items), 7; got != want {
		t.Fatalf("unexpected source summary item count: got=%d want=%d", got, want)
	}

	for _, item := range items {
		if item.Summary == "" {
			t.Fatalf("expected non-empty summary: %#v", item)
		}
		if item.Anchor.Path == "" || item.Anchor.Section == "" {
			t.Fatalf("expected anchor path and section: %#v", item)
		}
		if item.SourceHash == "" || item.ExpansionCost == 0 {
			t.Fatalf("expected source metadata: %#v", item)
		}
	}

	foundWorkflow := false
	foundPolicy := false
	for _, item := range items {
		if item.ID == "source_workflows_summary" {
			foundWorkflow = true
			if item.Kind != ContextItemKindWorkflowRule {
				t.Fatalf("expected workflow-rule kind: %#v", item)
			}
		}
		if item.ID == "source_policy_summary" {
			foundPolicy = true
			if item.Kind != ContextItemKindVerificationRecipe {
				t.Fatalf("expected verification-recipe kind: %#v", item)
			}
		}
	}
	if !foundWorkflow || !foundPolicy {
		t.Fatalf("expected workflow and policy summary items, got=%#v", items)
	}
}
