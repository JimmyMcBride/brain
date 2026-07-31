package conformance

import (
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestManifestLoadsPinnedStandaloneBaseline(t *testing.T) {
	manifest, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Baseline.Revision, "53ebd96bb954472d844651179bfe4e75baab96a0"; got != want {
		t.Fatalf("unexpected baseline revision: got %s want %s", got, want)
	}
	if len(manifest.Commands) == 0 || len(manifest.Cases) == 0 {
		t.Fatalf("expected populated manifest: %#v", manifest)
	}
}

func TestManifestRevisionErrorIncludesInvalidValue(t *testing.T) {
	manifest, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	manifest.Baseline.Revision = "not-a-full-sha"
	err = manifest.validate()
	if err == nil || !strings.Contains(err.Error(), `"not-a-full-sha"`) {
		t.Fatalf("expected invalid revision in validation error, got %v", err)
	}
}

func TestManifestCoversPhaseFourMappedCommands(t *testing.T) {
	manifest, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, command := range manifest.Commands {
		if command.Disposition == "shared" {
			got = append(got, command.ID)
		}
	}
	sort.Strings(got)
	want := []string{
		"brainstorm.challenge",
		"brainstorm.idea",
		"brainstorm.park",
		"brainstorm.refine",
		"brainstorm.reopen",
		"brainstorm.resume",
		"brainstorm.review",
		"brainstorm.sessions",
		"brainstorm.show",
		"brainstorm.start",
		"brainstorm.switch",
		"check.default",
		"check.project",
		"check.spec",
		"discuss.assess.local",
		"discuss.promote.local",
		"discuss.repair.local",
		"doctor",
		"guide.current",
		"guide.show.local",
		"roadmap.edit",
		"roadmap.show",
		"spec.analyze",
		"spec.checklist",
		"spec.edit",
		"spec.execute",
		"spec.handoff",
		"spec.initiative",
		"spec.show",
		"spec.status",
		"status",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected Phase 4 command coverage:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestManifestCoversEveryStandaloneRootFamily(t *testing.T) {
	manifest, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	families := make(map[string]struct{})
	for _, command := range manifest.Commands {
		families[command.Standalone[0]] = struct{}{}
	}
	var got []string
	for family := range families {
		got = append(got, family)
	}
	sort.Strings(got)
	want := []string{
		"adopt",
		"brainstorm",
		"check",
		"completion",
		"discuss",
		"doctor",
		"epic",
		"github",
		"guide",
		"help",
		"init",
		"roadmap",
		"skills",
		"source",
		"spec",
		"status",
		"story",
		"update",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected standalone root coverage:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestManifestKeepsProviderAndLinearCommandsOutOfPhaseFour(t *testing.T) {
	manifest, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range manifest.Commands {
		if command.Disposition == "shared" {
			continue
		}
		if command.TargetPhase == "phase-5" ||
			command.TargetPhase == "phase-6" ||
			command.TargetPhase == "phase-9" ||
			command.TargetPhase == "separate-lifecycle" ||
			command.TargetPhase == "future-module-discovery" {
			continue
		}
		t.Fatalf("deferred command %s has unexpected target %q", command.ID, command.TargetPhase)
	}
}

func TestInitialStatusCasesReferenceReadableAssets(t *testing.T) {
	manifest, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	wantCases := map[string]int{
		"status.compatible":    0,
		"status.missing":       1,
		"status.future-schema": 1,
	}
	for _, testCase := range manifest.Cases {
		wantExit, exists := wantCases[testCase.ID]
		if !exists {
			continue
		}
		delete(wantCases, testCase.ID)
		if testCase.Command != "status" || testCase.Expected.ExitCode != wantExit {
			t.Fatalf("unexpected status case: %#v", testCase)
		}
		fixture, err := Fixture(testCase.Fixture)
		if err != nil {
			t.Fatal(err)
		}
		if testCase.Fixture != "empty" {
			if _, err := fs.Stat(fixture, ".plan/.meta/workspace.json"); err != nil {
				t.Fatalf("fixture %s has no workspace metadata: %v", testCase.Fixture, err)
			}
		}
		if _, err := ReadGolden(testCase.Expected.Stdout); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadGolden(testCase.Expected.Stderr); err != nil {
			t.Fatal(err)
		}
	}
	if len(wantCases) != 0 {
		t.Fatalf("missing initial status cases: %v", wantCases)
	}
}

func TestWorkspaceQueryAndRoadmapCasesReferenceReadableAssets(t *testing.T) {
	manifest, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	wantCases := map[string]struct{}{
		"check.default.blocking": {},
		"check.project.blocking": {},
		"check.spec.blocking":    {},
		"roadmap.show":           {},
		"roadmap.edit.body":      {},
	}
	for _, testCase := range manifest.Cases {
		if _, exists := wantCases[testCase.ID]; !exists {
			continue
		}
		delete(wantCases, testCase.ID)
		if testCase.Fixture != "fixtures/schema-v3-compatible" {
			t.Fatalf("unexpected fixture for %s: %s", testCase.ID, testCase.Fixture)
		}
		for _, asset := range []string{testCase.Expected.Stdout, testCase.Expected.Stderr} {
			if _, err := ReadGolden(asset); err != nil {
				t.Fatal(err)
			}
		}
		if testCase.Expected.Files != "unchanged" {
			if _, err := ReadGolden(testCase.Expected.Files); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(wantCases) != 0 {
		t.Fatalf("missing workspace/query/roadmap cases: %v", wantCases)
	}
}
