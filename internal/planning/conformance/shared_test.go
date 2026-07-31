package conformance

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

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

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
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
