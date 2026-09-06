package conformance

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestGitHubManifestLoadsPinnedPlanRelease(t *testing.T) {
	manifest, err := LoadGitHub()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.Baseline.Tag, "v0.1.29"; got != want {
		t.Fatalf("unexpected baseline tag: got %s want %s", got, want)
	}
	if got, want := manifest.Baseline.Revision, "898b2a4c470350c0e9115302c99c6ad27bdead9c"; got != want {
		t.Fatalf("unexpected baseline revision: got %s want %s", got, want)
	}
	if manifest.Provider.RequiredCILive {
		t.Fatal("required GitHub conformance must not use live GitHub resources")
	}
}

func TestGitHubManifestCoversEveryRetainedFamily(t *testing.T) {
	manifest, err := LoadGitHub()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	caseCounts := map[string]int{}
	rerunFamilies := map[string]bool{}
	for _, family := range manifest.Families {
		got = append(got, family.ID)
	}
	for _, testCase := range manifest.Cases {
		caseCounts[testCase.Family]++
		if testCase.Rerun != nil {
			rerunFamilies[testCase.Family] = true
		}
	}
	sort.Strings(got)
	want := []string{
		"check.github",
		"discuss.assess.remote",
		"discuss.promote.remote",
		"discuss.repair.remote",
		"github.adopt",
		"github.enable",
		"github.project.status",
		"github.reconcile",
		"guide.show.remote",
		"repository.evidence",
		"story.github.compatibility",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected Phase 5 family coverage:\ngot:  %v\nwant: %v", got, want)
	}
	for _, family := range want {
		if caseCounts[family] == 0 {
			t.Fatalf("retained family %s has no captured case", family)
		}
		if !rerunFamilies[family] {
			t.Fatalf("retained family %s has no identical-rerun contract", family)
		}
	}
}

func TestGitHubManifestCapturesSafetyAndHostPolicyDifferences(t *testing.T) {
	manifest, err := LoadGitHub()
	if err != nil {
		t.Fatal(err)
	}
	legacyUnconfirmedMutations := map[string]bool{
		"github.enable":         true,
		"github.adopt":          true,
		"github.reconcile":      true,
		"github.project.status": true,
	}
	for _, testCase := range manifest.Cases {
		if !testCase.Safety.Mutation || testCase.Expected.ExitCode != 0 {
			continue
		}
		if testCase.Safety.TargetConfirmation != "confirmed" {
			t.Fatalf("successful mutation %s does not freeze confirmed target policy", testCase.ID)
		}
		if !legacyUnconfirmedMutations[testCase.Family] {
			continue
		}
		if testCase.Safety.BaselineConfirmation != "not-required" || len(testCase.IntentionalDifferences) == 0 {
			t.Fatalf("legacy unconfirmed mutation %s does not document target hardening", testCase.ID)
		}
	}
}

func TestGitHubManifestReadAndPreviewCasesHaveReadOnlyTranscripts(t *testing.T) {
	manifest, err := LoadGitHub()
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range manifest.Cases {
		if testCase.Safety.Mutation {
			continue
		}
		transcript, err := ReadProviderTranscript(testCase.Expected.ProviderTranscript)
		if err != nil {
			t.Fatal(err)
		}
		for _, call := range transcript.Calls {
			if providerMutation(call.Operation) {
				t.Fatalf("read/preview case %s includes provider mutation %s", testCase.ID, call.Operation)
			}
		}
	}
}

func TestGitHubManifestCapturesPartialFailureAndIdenticalRerun(t *testing.T) {
	manifest, err := LoadGitHub()
	if err != nil {
		t.Fatal(err)
	}
	var partial, rerun bool
	for _, testCase := range manifest.Cases {
		if testCase.Safety.PartialFailure {
			partial = true
			if !testCase.Safety.ManualFallbackAllowed || testCase.Expected.ExitCode == 0 || len(testCase.Expected.RemoteIdentities) == 0 {
				t.Fatalf("partial failure case lacks recovery evidence: %#v", testCase)
			}
			transcript, err := ReadProviderTranscript(testCase.Expected.ProviderTranscript)
			if err != nil {
				t.Fatal(err)
			}
			if !transcriptHasError(transcript) {
				t.Fatalf("partial failure case %s has no provider error", testCase.ID)
			}
		}
		if testCase.Rerun != nil {
			rerun = true
			transcript, err := ReadProviderTranscript(testCase.Rerun.ProviderTranscript)
			if err != nil {
				t.Fatal(err)
			}
			for _, call := range transcript.Calls {
				if providerMutation(call.Operation) && !testCase.Rerun.BaselineMutation {
					t.Fatalf("identical rerun %s includes undocumented provider mutation %s", testCase.ID, call.Operation)
				}
			}
			for _, classification := range testCase.Rerun.Classifications {
				if classification != "reuse" && classification != "unchanged" {
					t.Fatalf("identical rerun %s has unsafe classification %q", testCase.ID, classification)
				}
			}
		}
	}
	if !partial || !rerun {
		t.Fatalf("expected partial failure and identical rerun coverage: partial=%t rerun=%t", partial, rerun)
	}
}

func TestGitHubManifestFixturePreservesCurrentMetadataShape(t *testing.T) {
	manifest, err := LoadGitHub()
	if err != nil {
		t.Fatal(err)
	}
	var metadataAsset string
	for _, testCase := range manifest.Cases {
		if testCase.Expected.GitHubMetadata != "unchanged" {
			metadataAsset = testCase.Expected.GitHubMetadata
			break
		}
	}
	raw, err := ReadGitHubAsset(metadataAsset)
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]any
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"repo", "repo_url", "default_branch", "stories", "planning", "project_decisions"} {
		if _, ok := state[field]; !ok {
			t.Fatalf("captured GitHub metadata lacks %q", field)
		}
	}
	for field := range state {
		if strings.Contains(strings.ToLower(field), "linear") {
			t.Fatalf("GitHub metadata unexpectedly contains Linear field %q", field)
		}
	}
}

func providerMutation(operation string) bool {
	for _, fragment := range []string{".create", ".update", ".set", ".add", ".ensure"} {
		if strings.Contains(operation, fragment) {
			return true
		}
	}
	return false
}

func transcriptHasError(transcript ProviderTranscript) bool {
	for _, call := range transcript.Calls {
		if call.Error != "" {
			return true
		}
	}
	return false
}
