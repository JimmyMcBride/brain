package conformance

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"
	"github.com/JimmyMcBride/brain/planning/local"
)

func TestSharedWorkspaceQueryAndRoadmapBehaviorMatchesCapturedBaseline(t *testing.T) {
	root := filepath.Join(t.TempDir(), "fixture")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture, err := Fixture("fixtures/schema-v3-compatible")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(root, fixture); err != nil {
		t.Fatal(err)
	}
	service := application.New(local.New(root), application.Options{ModuleID: "conformance"})
	ctx := context.Background()

	status, err := service.ProjectStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/status-compatible.stdout", renderProjectStatus(status))

	projectCheck, err := service.Check(ctx, application.CheckInput{})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/check-project-blocking.stdout", renderCheckReport(projectCheck))

	specID := status.ReadySpecs[0].ID
	specCheck, err := service.Check(ctx, application.CheckInput{SpecID: &specID})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/check-spec-blocking.stdout", renderCheckReport(specCheck))

	roadmap, err := service.ReadRoadmap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/roadmap-show.stdout", roadmap.Path+"\n\n"+roadmap.Body)

	updated := "# Roadmap\n\n## Overview\n\nUpdated fixture roadmap.\n"
	events := &recordingSink{}
	result, err := service.UpdateRoadmap(ctx, application.UpdateRoadmapInput{Body: updated, Confirmed: true}, allowAll{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != application.MutationUpdate || len(events.events) != 1 {
		t.Fatalf("unexpected roadmap result: %#v events=%d", result, len(events.events))
	}
	raw, err := os.ReadFile(filepath.Join(root, ".plan", "ROADMAP.md"))
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/roadmap-edit.files", ".plan/ROADMAP.md\n---\n"+string(raw))
	rerun, err := service.UpdateRoadmap(ctx, application.UpdateRoadmapInput{Body: updated, Confirmed: true}, allowAll{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if rerun.Action != application.MutationUnchanged || rerun.Event != nil || len(events.events) != 1 {
		t.Fatalf("roadmap rerun was not idempotent: %#v events=%d", rerun, len(events.events))
	}
}

func TestSharedGuidedBrainstormAndDirectPromotionBehaviorMatchesCapturedBaseline(t *testing.T) {
	root := filepath.Join(t.TempDir(), "fixture")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture, err := Fixture("fixtures/schema-v3-guided")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(root, fixture); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 9, 7, 45, 22, 0, time.UTC)
	service := application.New(local.New(root), application.Options{
		ModuleID: "conformance", ProjectRoot: root, Now: func() time.Time { return now },
	})
	ctx := context.Background()
	events := &recordingSink{}

	sessions, err := service.ListGuidedSessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	current, err := service.CurrentGuidedSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/brainstorm-sessions.stdout", renderGuidedSessions(sessions, current.ChainID))

	packet, err := service.CurrentGuidePacket(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/guide-current.structural", renderGuideProjection(packet))

	id := planning.ArtifactID("local-promotion")
	assessment, err := service.AssessLocalBrainstorm(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/brainstorm-assess.structural", renderAssessmentProjection(assessment))
	draft, err := service.PreviewLocalPromotion(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/brainstorm-promote.structural", renderPromotionProjection(draft))

	idea, err := service.UpdateBrainstorm(ctx, application.BrainstormUpdateInput{
		ID: id, Section: "ideas", Body: "Keep the guide packet stable.", Confirmed: true,
	}, allowAll{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if idea.Action != application.MutationUpdate {
		t.Fatalf("unexpected idea action: %s", idea.Action)
	}
	assertGolden(t, "golden/brainstorm-idea.stdout", "Updated brainstorm "+idea.Document.Path+"\n")
	assertGolden(t, "golden/brainstorm-idea.files", renderBrainstormIdeaProjection(idea.Document))
	ideaRerun, err := service.UpdateBrainstorm(ctx, application.BrainstormUpdateInput{
		ID: id, Section: "ideas", Body: "Keep the guide packet stable.", Confirmed: true,
	}, allowAll{}, events)
	if err != nil || ideaRerun.Action != application.MutationUnchanged || ideaRerun.Event != nil {
		t.Fatalf("idea rerun was not idempotent: %#v err=%v", ideaRerun, err)
	}

	promoted, err := service.PromoteLocalBrainstorm(ctx, application.LocalPromotionInput{BrainstormID: id, Confirmed: true}, allowAll{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if promoted.Action != application.MutationCreate || len(promoted.Specs) != 1 || promoted.Specs[0].Artifact.ID != id {
		t.Fatalf("unexpected direct promotion: %#v", promoted)
	}
	for _, legacy := range []string{"epics", "stories"} {
		if _, err := os.Stat(filepath.Join(root, ".plan", legacy)); !os.IsNotExist(err) {
			t.Fatalf("direct promotion created legacy %s hierarchy: %v", legacy, err)
		}
	}
	promotionRerun, err := service.PromoteLocalBrainstorm(ctx, application.LocalPromotionInput{BrainstormID: id, Confirmed: true}, allowAll{}, events)
	if err != nil || promotionRerun.Action != application.MutationUnchanged || promotionRerun.Event != nil {
		t.Fatalf("promotion rerun was not idempotent: %#v err=%v", promotionRerun, err)
	}
	if len(events.events) != 2 {
		t.Fatalf("expected one idea and one promotion event, got %d", len(events.events))
	}
}

func TestSharedSpecWorkflowMatchesCapturedBaseline(t *testing.T) {
	root := filepath.Join(t.TempDir(), "fixture")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture, err := Fixture("fixtures/schema-v3-spec")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(root, fixture); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 9, 7, 45, 22, 0, time.UTC)
	service := application.New(local.New(root), application.Options{ModuleID: "conformance", ProjectRoot: root, Now: func() time.Time { return now }})
	ctx := context.Background()
	events := &recordingSink{}
	draftID := planning.ArtifactID("draft-spec")
	approved, err := service.SetSpecStatus(ctx, application.SpecStatusInput{ID: draftID, Status: planning.SpecApproved, Confirmed: true}, allowAll{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Document.Artifact.Status != planning.SpecApproved || approved.Action != application.MutationUpdate {
		t.Fatalf("unexpected approval: %#v", approved)
	}
	id := planning.ArtifactID("execution-ready")
	analysis, err := service.AnalyzeSpec(ctx, id, true, allowAll{}, events)
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/spec-analyze.stdout", renderAnalysisReport(analysis))
	checklist, err := service.RunSpecChecklist(ctx, id, "general", true, allowAll{}, events)
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/spec-checklist.stdout", renderChecklistReport(checklist))
	initiative := planning.ArtifactID("phase-four")
	updated, err := service.SetSpecInitiative(ctx, application.SpecInitiativeInput{ID: id, Initiative: &initiative, Title: "Phase Four", Summary: "Complete local compatibility.", Confirmed: true}, allowAll{}, events)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Document.Metadata["initiative_title"] != "Phase Four" {
		t.Fatalf("initiative metadata missing: %#v", updated.Document.Metadata)
	}
	execution, err := service.BeginSpecExecution(ctx, application.SpecExecutionInput{ID: id, BranchPrefix: "feature/", Confirmed: true}, allowAll{}, events)
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "golden/spec-execute.stdout", renderExecutionResult(execution))
	if _, err := os.Stat(filepath.Join(root, ".plan", "stories")); !os.IsNotExist(err) {
		t.Fatalf("execution persisted stories: %v", err)
	}
	rerun, err := service.BeginSpecExecution(ctx, application.SpecExecutionInput{ID: id, BranchPrefix: "feature/", Confirmed: true}, allowAll{}, events)
	if err != nil || rerun.Action != application.MutationUnchanged || rerun.Event != nil {
		t.Fatalf("execution rerun not idempotent: %#v err=%v", rerun, err)
	}
}

func renderAnalysisReport(report application.SpecAnalysisReport) string {
	return fmt.Sprintf("spec_analysis: %s\nfindings: %d total, %d blocking, %d guidance\nstatus: ok\n", report.SpecPath, len(report.Findings), report.ErrorCount(), report.WarningCount())
}
func renderChecklistReport(report application.SpecChecklistReport) string {
	return fmt.Sprintf("spec_checklist: %s\nprofile: %s\nfindings: %d total, %d blocking, %d guidance\nstatus: ok\n", report.SpecPath, report.Profile, len(report.Findings), report.ErrorCount(), report.WarningCount())
}
func renderExecutionResult(result application.SpecExecutionResult) string {
	var out bytes.Buffer
	view := result.Execution
	fmt.Fprintf(&out, "spec_execution: %s\nstatus: %s\nbranch: %s\nslices: %d\n", view.SpecPath, view.Status, view.SuggestedBranch, len(view.Slices))
	for i, slice := range view.Slices {
		fmt.Fprintf(&out, "%d. %s\n   goal: %s\n", i+1, slice.Title, slice.Goal)
		for _, verify := range slice.Verification {
			fmt.Fprintf(&out, "   verify: %s\n", verify)
		}
	}
	fmt.Fprintln(&out, "workflow:\n- implement one slice at a time\n- review and verify each slice before committing it\n- open a PR after the full spec is built")
	return out.String()
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	got = normalizeGolden(got)
	want, err := ReadGolden(name)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s mismatch:\ngot:\n%s\nwant:\n%s", name, got, want)
	}
}

func renderProjectStatus(status application.ProjectStatus) string {
	var out bytes.Buffer
	fmt.Fprintf(&out, "project: %s\n", status.Project)
	fmt.Fprintf(&out, "planning_model: %s\n", status.PlanningModel)
	fmt.Fprintf(&out, "source_mode: %s\n", status.SourceMode)
	fmt.Fprintf(&out, "specs: %d total, %d draft, %d approved, %d implementing, %d done\n",
		status.TotalSpecs, status.DraftSpecs, status.ApprovedSpecs, status.ImplementingSpecs, status.DoneSpecs)
	if len(status.ReadySpecs) > 0 {
		fmt.Fprintf(&out, "ready_specs: %d\n", len(status.ReadySpecs))
		for _, spec := range status.ReadySpecs {
			initiative := ""
			if spec.Initiative != nil {
				initiative = " initiative=" + string(*spec.Initiative)
			}
			fmt.Fprintf(&out, "  - %s%s status=%s\n", spec.Title, initiative, spec.Status)
		}
	}
	return out.String()
}

func renderCheckReport(report application.CheckReport) string {
	var out bytes.Buffer
	writeCheckReport(&out, report)
	return out.String()
}

func renderGuidedSessions(sessions []application.GuidedSessionRecord, current string) string {
	var out bytes.Buffer
	for _, session := range sessions {
		marker := " "
		if session.ChainID == current {
			marker = "*"
		}
		fmt.Fprintf(&out, "%s %s stage=%s next=%s\n", marker, session.ChainID, session.CurrentStage, session.NextAction)
	}
	return out.String()
}

func renderGuideProjection(packet application.GuidePacket) string {
	return fmt.Sprintf("kind: %s\nschema_version: %d\nchain_id: %v\nartifact: %v\ncheckpoint: %v\npass: %v\nownership: %v\ngenerated_at: %s\n",
		packet.Kind, packet.SchemaVersion, packet.Session["chain_id"], packet.Artifact["slug"], packet.Mode["checkpoint"],
		packet.Mode["pass"], packet.Ownership["mode"], packet.GeneratedAt)
}

func renderAssessmentProjection(assessment application.CollaborationAssessment) string {
	title := firstString(assessment.Decision.SuggestedTitles["specs"])
	return fmt.Sprintf("kind: %s\nschema_version: %d\nsource: %v\nstate: %s\nconfidence: %s\nrecommended_path: %s\nspec: %s\ngenerated_at: %s\n",
		assessment.Kind, assessment.SchemaVersion, assessment.Source["brainstorm_slug"], assessment.Decision.State,
		assessment.Decision.Confidence, assessment.Decision.RecommendedPath, title, assessment.GeneratedAt)
}

func renderPromotionProjection(draft application.LocalPromotionDraft) string {
	var spec application.PromotionSpecDraft
	if len(draft.ProposedSpecs) > 0 {
		spec = draft.ProposedSpecs[0]
	}
	return fmt.Sprintf("kind: %s\nschema_version: %d\nsource: %v\nstate: %s\npromotion_decision: %s\nspec: %s\nspec_kind: %s\nspec_action: %s\nconfirmation_required: %t\ngenerated_at: %s\n",
		draft.Kind, draft.SchemaVersion, draft.Source["brainstorm_slug"], draft.Assessment.State, draft.PromotionDecision,
		spec.Title, spec.Kind, spec.Action, draft.ConfirmationRequired, draft.GeneratedAt)
}

func renderBrainstormIdeaProjection(document application.BrainstormDocument) string {
	return fmt.Sprintf("%s\nupdated_at: %v\n## Ideas\n%s\n", document.Path, document.Metadata["updated_at"], strings.TrimSpace(extractProjectedSection(document.Body, "Ideas")))
}

func extractProjectedSection(body, heading string) string {
	marker := "## " + heading
	start := strings.Index(body, marker)
	if start < 0 {
		return ""
	}
	value := body[start+len(marker):]
	if end := strings.Index(value, "\n## "); end >= 0 {
		value = value[:end]
	}
	return strings.TrimSpace(value)
}

func firstString(value any) string {
	switch values := value.(type) {
	case []string:
		if len(values) > 0 {
			return values[0]
		}
	case []any:
		if len(values) > 0 {
			text, _ := values[0].(string)
			return text
		}
	}
	return ""
}

func writeCheckReport(out io.Writer, report application.CheckReport) {
	fmt.Fprintf(out, "check_scope: %s\n", report.Scope)
	fmt.Fprintf(out, "findings: %d total, %d blocking, %d guidance\n", len(report.Findings), report.ErrorCount(), report.WarningCount())
	if len(report.Findings) == 0 {
		fmt.Fprintln(out, "status: ok")
		return
	}
	for _, finding := range report.Findings {
		fmt.Fprintf(out, "- [%s] %s %s :: %s\n", finding.Severity, finding.ArtifactType, finding.ArtifactPath, finding.Section)
		fmt.Fprintf(out, "  %s\n", finding.Message)
		if finding.Suggestion != "" {
			fmt.Fprintf(out, "  fix: %s\n", finding.Suggestion)
		}
	}
}

type allowAll struct{}

func (allowAll) Require(context.Context, string) error { return nil }

type recordingSink struct{ events []application.Event }

func (s *recordingSink) Publish(_ context.Context, event application.Event) error {
	s.events = append(s.events, event)
	return nil
}
