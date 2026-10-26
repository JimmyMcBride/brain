package application

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollaborationAssessmentMatchesPinnedDecision(t *testing.T) {
	root := filepath.Join("..", "..", "internal", "planning", "conformance", "testdata", "github-v1")
	raw, err := os.ReadFile(filepath.Join(root, "provider", "discussion.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Discussions map[string]struct {
			Title, Body string
			Comments    []struct{ Body string }
		}
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	item := fixture.Discussions["49"]
	snapshot := CollaborationSourceSnapshot{Title: item.Title, Content: item.Body}
	for _, comment := range item.Comments {
		snapshot.Contributions = append(snapshot.Contributions, CollaborationContribution{Content: comment.Body})
	}
	decision := assessCollaboration(snapshot)
	raw, err = os.ReadFile(filepath.Join(root, "golden", "discuss-assess.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		Decision struct {
			State           string
			SuggestedTitles map[string]any `json:"suggested_titles"`
		}
	}
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	// Compare the captured decision projection independently of host rendering.
	actual, _ := json.Marshal(decision.SuggestedTitles)
	var titles map[string]any
	if err := json.Unmarshal(actual, &titles); err != nil {
		t.Fatal(err)
	}
	if decision.State != golden.Decision.State || !reflect.DeepEqual(titles, golden.Decision.SuggestedTitles) {
		t.Fatalf("decision drift: %+v", decision)
	}
	if got := decision.DependencyGuess[1]["blocked_by"]; !reflect.DeepEqual(got, []string{"Collaboration source"}) {
		t.Fatalf("dependency lost: %v", got)
	}
}
