package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/internal/modules"
	officialplanning "github.com/JimmyMcBride/brain/internal/official/planning"
	"github.com/JimmyMcBride/brain/planning/application"
)

func TestCLIPlanningLocalWorkflowAndIdempotentAudit(t *testing.T) {
	env := newCLIEnv(t)
	fixture := filepath.Join(env.moduleRoot, "planning", "local", "testdata", "compatible")
	if err := os.CopyFS(env.project, os.DirFS(fixture)); err != nil {
		t.Fatal(err)
	}

	result := runPlanningCLI(t, env, "", "--project", env.project, "plan", "status")
	if result.err == nil || !strings.Contains(result.err.Error(), "module_disabled") {
		t.Fatalf("expected disabled diagnostic, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".brain", "modules.yaml")); !os.IsNotExist(err) {
		t.Fatalf("disabled Planning command wrote module config: %v", err)
	}

	permissions := officialplanning.Registration().Descriptor.Permissions
	args := []string{"--project", env.project, "modules", "grant", officialplanning.ID}
	args = append(args, permissions...)
	requireOK(t, runPlanningCLI(t, env, "", args...))
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "modules", "enable", officialplanning.ID))

	status := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "status"))
	for _, expected := range []string{"project: project", "planning_model: spec_first_v1", "source_mode: local", "specs: 1 total, 0 draft, 1 approved, 0 implementing, 0 done"} {
		if !strings.Contains(status, expected) {
			t.Fatalf("status missing %q:\n%s", expected, status)
		}
	}
	brainstorms := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "list"))
	if !strings.Contains(brainstorms, "alpha\tAlpha Brainstorm") {
		t.Fatalf("unexpected brainstorm list:\n%s", brainstorms)
	}
	brainstorm := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "show", "alpha"))
	if !strings.Contains(brainstorm, "# Brainstorm: Alpha Brainstorm") {
		t.Fatalf("unexpected brainstorm show:\n%s", brainstorm)
	}
	specs := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "list"))
	if !strings.Contains(specs, "alpha-spec\tapproved\tAlpha Spec") {
		t.Fatalf("unexpected spec list:\n%s", specs)
	}
	spec := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "show", "alpha-spec"))
	if !strings.Contains(spec, "# Alpha Spec") {
		t.Fatalf("unexpected spec show:\n%s", spec)
	}
	check := runPlanningCLI(t, env, "", "--project", env.project, "plan", "check", "spec", "alpha-spec")
	if check.err == nil || !strings.Contains(check.err.Error(), "4 blocking issue(s)") ||
		!strings.Contains(check.stdout, "check_scope: spec:alpha-spec") ||
		!strings.Contains(check.stdout, "spec .plan/specs/alpha-spec.md :: Problem") {
		t.Fatalf("unexpected spec check: %#v", check)
	}
	roadmap := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "roadmap", "show"))
	if !strings.Contains(roadmap, ".plan/ROADMAP.md\n\n# Roadmap") {
		t.Fatalf("unexpected roadmap show:\n%s", roadmap)
	}
	editorWithoutConfirmation := runPlanningCLI(t, env, "", "--project", env.project, "plan", "roadmap", "edit", "--editor", "unused")
	if editorWithoutConfirmation.err == nil || !errors.Is(editorWithoutConfirmation.err, application.ErrConfirmationRequired) {
		t.Fatalf("expected editor confirmation error, got %#v", editorWithoutConfirmation)
	}
	updatedRoadmap := "# Roadmap\n\n## Overview\n\nUpdated from Brain.\n"
	roadmapPreview := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "roadmap", "edit", "--body", updatedRoadmap))
	if !strings.Contains(roadmapPreview, "update\t.plan/ROADMAP.md") || !strings.Contains(roadmapPreview, "Preview only") {
		t.Fatalf("unexpected roadmap preview:\n%s", roadmapPreview)
	}
	roadmapPath := filepath.Join(env.project, ".plan", "ROADMAP.md")
	beforeRoadmap, err := os.ReadFile(roadmapPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(beforeRoadmap), "Updated from Brain") {
		t.Fatal("roadmap preview changed the file")
	}
	roadmapUpdate := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "roadmap", "edit", "--body", updatedRoadmap, "--confirm"))
	if !strings.Contains(roadmapUpdate, "update\t.plan/ROADMAP.md") {
		t.Fatalf("unexpected roadmap update:\n%s", roadmapUpdate)
	}
	roadmapRerun := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "roadmap", "edit", "--body", updatedRoadmap, "--confirm"))
	if !strings.Contains(roadmapRerun, "unchanged\t.plan/ROADMAP.md") {
		t.Fatalf("unexpected roadmap rerun:\n%s", roadmapRerun)
	}

	preview := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "start", "CLI Flow"))
	if !strings.Contains(preview, "create\t.plan/brainstorms/cli-flow.md") ||
		!strings.Contains(preview, "Preview only") {
		t.Fatalf("unexpected preview:\n%s", preview)
	}
	createdPath := filepath.Join(env.project, ".plan", "brainstorms", "cli-flow.md")
	if _, err := os.Stat(createdPath); !os.IsNotExist(err) {
		t.Fatalf("preview wrote artifact: %v", err)
	}

	created := runPlanningCLI(t, env, "", "--json", "--project", env.project, "plan", "brainstorm", "start", "CLI Flow", "--confirm")
	output := requireOK(t, created)
	if created.stderr != "" {
		t.Fatalf("JSON command wrote stderr: %q", created.stderr)
	}
	var mutation planningMutationOutput
	if err := json.Unmarshal([]byte(output), &mutation); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, output)
	}
	if mutation.Action != application.MutationCreate || mutation.Event == nil ||
		mutation.Event.Name != application.EventBrainstormCreated {
		t.Fatalf("unexpected mutation: %#v", mutation)
	}
	if _, err := os.Stat(createdPath); err != nil {
		t.Fatal(err)
	}

	unchanged := requireOK(t, runPlanningCLI(t, env, "", "--json", "--project", env.project, "plan", "brainstorm", "start", "CLI Flow", "--confirm"))
	mutation = planningMutationOutput{}
	if err := json.Unmarshal([]byte(unchanged), &mutation); err != nil {
		t.Fatal(err)
	}
	if mutation.Action != application.MutationUnchanged || mutation.Event != nil {
		t.Fatalf("unexpected idempotent mutation: %#v", mutation)
	}
	history, err := os.ReadFile(filepath.Join(env.project, ".brain", "state", "history.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(history), application.EventBrainstormCreated); count != 2 {
		t.Fatalf("expected one event record with operation and ID references, got count=%d:\n%s", count, history)
	}
	if count := strings.Count(string(history), application.EventRoadmapUpdated); count != 2 {
		t.Fatalf("expected one roadmap event record with operation and ID references, got count=%d:\n%s", count, history)
	}

	before, err := os.ReadFile(createdPath)
	if err != nil {
		t.Fatal(err)
	}
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "modules", "disable", officialplanning.ID))
	result = runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "start", "Blocked", "--confirm")
	if result.err == nil || !strings.Contains(result.err.Error(), "module_disabled") {
		t.Fatalf("expected disabled command failure, got %#v", result)
	}
	after, err := os.ReadFile(createdPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("disabled command changed existing Planning artifact")
	}
	if _, err := os.Stat(filepath.Join(env.project, ".plan", "brainstorms", "blocked.md")); !os.IsNotExist(err) {
		t.Fatalf("disabled command created artifact: %v", err)
	}
}

func TestCLIPlanningGuidedBrainstormAndDirectPromotionWorkflow(t *testing.T) {
	env := newCLIEnv(t)
	fixture := filepath.Join(env.moduleRoot, "internal", "planning", "conformance", "testdata", "fixtures", "schema-v3-guided")
	if err := os.CopyFS(env.project, os.DirFS(fixture)); err != nil {
		t.Fatal(err)
	}
	permissions := officialplanning.Registration().Descriptor.Permissions
	grant := []string{"--project", env.project, "modules", "grant", officialplanning.ID}
	grant = append(grant, permissions...)
	requireOK(t, runPlanningCLI(t, env, "", grant...))
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "modules", "enable", officialplanning.ID))

	sessions := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "sessions"))
	for _, expected := range []string{
		"* brainstorm/local-promotion stage=brainstorm",
		"  brainstorm/secondary stage=brainstorm",
	} {
		if !strings.Contains(sessions, expected) {
			t.Fatalf("sessions missing %q:\n%s", expected, sessions)
		}
	}
	guideJSON := requireOK(t, runPlanningCLI(t, env, "", "--json", "--project", env.project, "plan", "guide", "current"))
	var packet application.GuidePacket
	if err := json.Unmarshal([]byte(guideJSON), &packet); err != nil {
		t.Fatal(err)
	}
	if packet.Kind != "guide_packet" || packet.Artifact["slug"] != "local-promotion" || packet.Mode["checkpoint"] != "clarify-open-approaches" {
		t.Fatalf("unexpected guide packet: %#v", packet)
	}
	assessmentJSON := requireOK(t, runPlanningCLI(t, env, "", "--json", "--project", env.project, "plan", "brainstorm", "assess", "local-promotion"))
	var assessment application.CollaborationAssessment
	if err := json.Unmarshal([]byte(assessmentJSON), &assessment); err != nil {
		t.Fatal(err)
	}
	if assessment.Decision.State != "ready_single_spec" || assessment.Decision.RecommendedPath != "single_spec" {
		t.Fatalf("unexpected maturity assessment: %#v", assessment)
	}

	brainstormPath := filepath.Join(env.project, ".plan", "brainstorms", "local-promotion.md")
	beforeIdea, err := os.ReadFile(brainstormPath)
	if err != nil {
		t.Fatal(err)
	}
	preview := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "idea", "local-promotion", "--body", "Keep the guide packet stable."))
	if !strings.Contains(preview, "Preview only") {
		t.Fatalf("unexpected idea preview:\n%s", preview)
	}
	afterPreview, err := os.ReadFile(brainstormPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterPreview) != string(beforeIdea) {
		t.Fatal("idea preview mutated brainstorm")
	}
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "idea", "local-promotion", "--body", "Keep the guide packet stable.", "--confirm"))
	ideaRerun := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "idea", "local-promotion", "--body", "Keep the guide packet stable.", "--confirm"))
	if !strings.Contains(ideaRerun, "unchanged") {
		t.Fatalf("unexpected idea rerun:\n%s", ideaRerun)
	}

	promotionPreviewJSON := requireOK(t, runPlanningCLI(t, env, "", "--json", "--project", env.project, "plan", "brainstorm", "promote", "local-promotion"))
	var draft application.LocalPromotionDraft
	if err := json.Unmarshal([]byte(promotionPreviewJSON), &draft); err != nil {
		t.Fatal(err)
	}
	if draft.PromotionDecision != "single_spec" || len(draft.ProposedSpecs) != 1 || draft.ProposedSpecs[0].Kind != "spec" {
		t.Fatalf("unexpected direct promotion preview: %#v", draft)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".plan", "specs", "local-promotion.md")); !os.IsNotExist(err) {
		t.Fatalf("promotion preview wrote spec: %v", err)
	}
	promotionJSON := requireOK(t, runPlanningCLI(t, env, "", "--json", "--project", env.project, "plan", "brainstorm", "promote", "local-promotion", "--confirm"))
	var promotion application.LocalPromotionResult
	if err := json.Unmarshal([]byte(promotionJSON), &promotion); err != nil {
		t.Fatal(err)
	}
	if promotion.Action != application.MutationCreate || len(promotion.Specs) != 1 || promotion.Event == nil {
		t.Fatalf("unexpected direct promotion: %#v", promotion)
	}
	for _, legacy := range []string{"epics", "stories"} {
		if _, err := os.Stat(filepath.Join(env.project, ".plan", legacy)); !os.IsNotExist(err) {
			t.Fatalf("direct promotion created legacy %s hierarchy: %v", legacy, err)
		}
	}
	promotionRerunJSON := requireOK(t, runPlanningCLI(t, env, "", "--json", "--project", env.project, "plan", "brainstorm", "promote", "local-promotion", "--confirm"))
	promotion = application.LocalPromotionResult{}
	if err := json.Unmarshal([]byte(promotionRerunJSON), &promotion); err != nil {
		t.Fatal(err)
	}
	if promotion.Action != application.MutationUnchanged || promotion.Event != nil {
		t.Fatalf("unexpected promotion rerun: %#v", promotion)
	}
	history, err := os.ReadFile(filepath.Join(env.project, ".brain", "state", "history.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(history), application.EventBrainstormUpdated); count != 2 {
		t.Fatalf("expected one brainstorm update event record, got count=%d", count)
	}
	if count := strings.Count(string(history), application.EventBrainstormPromoted); count != 2 {
		t.Fatalf("expected one promotion event record, got count=%d", count)
	}
}

func TestCLIPlanningCanonicalSpecWorkflow(t *testing.T) {
	env := newCLIEnv(t)
	fixture := filepath.Join(env.moduleRoot, "internal", "planning", "conformance", "testdata", "fixtures", "schema-v3-spec")
	if err := os.CopyFS(env.project, os.DirFS(fixture)); err != nil {
		t.Fatal(err)
	}
	grant := []string{"--project", env.project, "modules", "grant", officialplanning.ID}
	grant = append(grant, officialplanning.Registration().Descriptor.Permissions...)
	requireOK(t, runPlanningCLI(t, env, "", grant...))
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "modules", "enable", officialplanning.ID))
	specPath := filepath.Join(env.project, ".plan", "specs", "execution-ready.md")
	before, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	preview := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "analyze", "execution-ready"))
	if !strings.Contains(preview, "Preview only") || !strings.Contains(preview, "status: ok") {
		t.Fatalf("unexpected analysis preview:\n%s", preview)
	}
	after, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("analysis preview mutated spec")
	}
	editedBody := string(before) + "\n## Notes\n\nEdited through Brain.\n"
	editPreview := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "edit", "execution-ready", "--body", editedBody))
	if !strings.Contains(editPreview, "Preview only") {
		t.Fatalf("unexpected edit preview:\n%s", editPreview)
	}
	unchanged, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(unchanged) != string(before) {
		t.Fatal("edit preview mutated spec")
	}
	editOutput := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "edit", "execution-ready", "--body", editedBody, "--confirm"))
	assertPlanningGoldenOutput(t, env.moduleRoot, "spec-edit.stdout", editOutput)
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "analyze", "execution-ready", "--confirm"))
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "checklist", "execution-ready", "--profile", "general", "--confirm"))
	initiativeOutput := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "initiative", "execution-ready", "--set", "phase-four", "--title", "Phase Four", "--confirm"))
	assertPlanningGoldenOutput(t, env.moduleRoot, "spec-initiative.stdout", initiativeOutput)
	statusPreview := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "status", "draft-spec", "--set", "approved"))
	if !strings.Contains(statusPreview, "Preview only") {
		t.Fatalf("unexpected status preview:\n%s", statusPreview)
	}
	statusOutput := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "status", "draft-spec", "--set", "approved", "--confirm"))
	assertPlanningGoldenOutput(t, env.moduleRoot, "spec-status.stdout", statusOutput)
	executionPreview := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "execute", "execution-ready"))
	if !strings.Contains(executionPreview, "slices: 2") || !strings.Contains(executionPreview, "Preview only") {
		t.Fatalf("unexpected execution preview:\n%s", executionPreview)
	}
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "execute", "execution-ready", "--confirm"))
	handoff := requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "plan", "spec", "handoff", "execution-ready", "--confirm"))
	if !strings.Contains(handoff, "Recommended next stage: continue into execution") {
		t.Fatalf("unexpected handoff:\n%s", handoff)
	}
	sessions, err := os.ReadFile(filepath.Join(env.project, ".plan", ".meta", "guided_sessions.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sessions), `"current_stage": "execution"`) {
		t.Fatalf("handoff did not advance session:\n%s", sessions)
	}
	if _, err := os.Stat(filepath.Join(env.project, ".plan", "stories")); !os.IsNotExist(err) {
		t.Fatalf("spec execution created stories: %v", err)
	}
}

func assertPlanningGoldenOutput(t *testing.T, moduleRoot, name, got string) {
	t.Helper()
	want, err := os.ReadFile(filepath.Join(moduleRoot, "internal", "planning", "conformance", "testdata", "golden", name))
	if err != nil {
		t.Fatal(err)
	}
	normalizedWant := strings.ReplaceAll(string(want), "\r\n", "\n")
	normalizedGot := strings.ReplaceAll(got, "\r\n", "\n")
	if normalizedGot != normalizedWant {
		t.Fatalf("output differs from %s:\nwant:\n%s\ngot:\n%s", name, want, got)
	}
}

func TestCLIPlanningFutureSchemaIsReadOnly(t *testing.T) {
	env := newCLIEnv(t)
	fixture := filepath.Join(env.moduleRoot, "planning", "local", "testdata", "future")
	if err := os.CopyFS(env.project, os.DirFS(fixture)); err != nil {
		t.Fatal(err)
	}
	permissions := officialplanning.Registration().Descriptor.Permissions
	args := []string{"--project", env.project, "modules", "grant", officialplanning.ID}
	args = append(args, permissions...)
	requireOK(t, runPlanningCLI(t, env, "", args...))
	requireOK(t, runPlanningCLI(t, env, "", "--project", env.project, "modules", "enable", officialplanning.ID))
	status := runPlanningCLI(t, env, "", "--project", env.project, "plan", "status")
	if status.err == nil || !strings.Contains(status.err.Error(), "newer than supported schema 3") || status.stdout != "" {
		t.Fatalf("unexpected future status: %#v", status)
	}
	result := runPlanningCLI(t, env, "", "--project", env.project, "plan", "brainstorm", "start", "Blocked", "--confirm")
	if result.err == nil || !strings.Contains(result.err.Error(), application.ErrWorkspaceNotWritable.Error()) {
		t.Fatalf("expected future-schema write refusal, got %#v", result)
	}
}

func TestResolvePlanningCheckInputReportsScopeArity(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "spec missing slug", args: []string{"spec"}, want: "check spec requires a slug"},
		{name: "project extra argument", args: []string{"project", "extra"}, want: "check project does not accept arguments"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolvePlanningCheckInput(tt.args)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("got %v want %q", err, tt.want)
			}
		})
	}
}

func TestSplitEditorCommandPreservesQuotedPaths(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{
			name:    "windows path",
			command: `"C:\Program Files\Microsoft VS Code\bin\code.exe" --wait`,
			want:    []string{`C:\Program Files\Microsoft VS Code\bin\code.exe`, "--wait"},
		},
		{
			name:    "unix path",
			command: `'/opt/Visual Editor/bin/editor' -f`,
			want:    []string{"/opt/Visual Editor/bin/editor", "-f"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitEditorCommand(tt.command)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(got, "\x00") != strings.Join(tt.want, "\x00") {
				t.Fatalf("got %#v want %#v", got, tt.want)
			}
		})
	}
}

func runPlanningCLI(t *testing.T, env *cliEnv, stdin string, args ...string) cliResult {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(env.moduleRoot); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	command := newRootCommand(rootOptions{
		in:                  strings.NewReader(stdin),
		out:                 &stdout,
		errOut:              &stderr,
		moduleRegistrations: []modules.Registration{officialplanning.Registration()},
	})
	command.SetArgs(args)
	err = command.Execute()
	return cliResult{
		stdout: normalizeCLIOutput(stdout.String(), env.root),
		stderr: normalizeCLIOutput(stderr.String(), env.root),
		err:    err,
	}
}
