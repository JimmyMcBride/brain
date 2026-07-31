package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JimmyMcBride/brain/internal/app"
	"github.com/JimmyMcBride/brain/internal/history"
	"github.com/JimmyMcBride/brain/internal/modules"
	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"

	"github.com/spf13/cobra"
)

const planningCommandGroup = "plan"

func addPlanningCommand(root *cobra.Command, _ *rootFlagsState, loadApp appLoader) {
	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Use the optional Brain Planning module",
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show overall local Planning status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				status, err := service.ProjectStatus(cmd.Context())
				if err != nil {
					return err
				}
				return appCtx.Output.Print(status, func(w io.Writer) error {
					if _, err := fmt.Fprintf(w, "project: %s\n", status.Project); err != nil {
						return err
					}
					if _, err := fmt.Fprintf(w, "planning_model: %s\n", status.PlanningModel); err != nil {
						return err
					}
					if _, err := fmt.Fprintf(w, "source_mode: %s\n", status.SourceMode); err != nil {
						return err
					}
					if _, err := fmt.Fprintf(w, "specs: %d total, %d draft, %d approved, %d implementing, %d done\n",
						status.TotalSpecs, status.DraftSpecs, status.ApprovedSpecs, status.ImplementingSpecs, status.DoneSpecs); err != nil {
						return err
					}
					if len(status.ReadySpecs) > 0 {
						if _, err := fmt.Fprintf(w, "ready_specs: %d\n", len(status.ReadySpecs)); err != nil {
							return err
						}
						for _, spec := range status.ReadySpecs {
							initiative := ""
							if spec.Initiative != nil {
								initiative = " initiative=" + string(*spec.Initiative)
							}
							if _, err := fmt.Fprintf(w, "  - %s%s status=%s\n", spec.Title, initiative, spec.Status); err != nil {
								return err
							}
						}
					}
					return nil
				})
			})
		},
	}

	checkCmd := &cobra.Command{
		Use:   "check [project|spec] [slug]",
		Short: "Run local Planning quality checks",
		Args:  cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := resolvePlanningCheckInput(args)
			if err != nil {
				return err
			}
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				report, err := service.Check(cmd.Context(), input)
				if err != nil {
					return err
				}
				if err := appCtx.Output.Print(report, func(w io.Writer) error { return printPlanningCheckReport(w, report) }); err != nil {
					return err
				}
				if report.HasErrors() {
					return fmt.Errorf("planning check found %d blocking issue(s)", report.ErrorCount())
				}
				return nil
			})
		},
	}

	roadmapCmd := &cobra.Command{Use: "roadmap", Short: "Show or edit local Planning roadmap Markdown"}
	roadmapShowCmd := &cobra.Command{
		Use: "show", Short: "Show ROADMAP.md", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				document, err := service.ReadRoadmap(cmd.Context())
				if err != nil {
					return err
				}
				return appCtx.Output.Print(document, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\n\n%s", document.Path, document.Body)
					return err
				})
			})
		},
	}
	var roadmapBody string
	var roadmapStdin bool
	var roadmapEditor string
	var confirmRoadmap bool
	roadmapEditCmd := &cobra.Command{
		Use: "edit", Short: "Edit ROADMAP.md via --body, --stdin, or an editor", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRoadmap, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, _ string) error {
				usingEditor := !roadmapStdin && roadmapBody == ""
				if usingEditor && !confirmRoadmap {
					return fmt.Errorf("%w: pass --confirm before opening the roadmap editor", application.ErrConfirmationRequired)
				}
				body, err := readBody(cmd.InOrStdin(), roadmapBody, roadmapStdin)
				if err != nil {
					return err
				}
				if usingEditor {
					current, err := service.ReadRoadmap(cmd.Context())
					if err != nil {
						return err
					}
					body, err = editPlanningText(current.Body, roadmapEditor)
					if err != nil {
						return err
					}
				}
				preview, err := service.PreviewRoadmap(cmd.Context(), body)
				if err != nil {
					return err
				}
				if !confirmRoadmap || preview.Action == application.MutationUnchanged {
					return appCtx.Output.Print(preview, func(w io.Writer) error {
						if _, err := fmt.Fprintf(w, "%s\t%s\n", preview.Action, preview.Document.Path); err != nil {
							return err
						}
						if preview.Action == application.MutationUpdate {
							_, err := fmt.Fprintln(w, "Preview only. Rerun with --confirm to write.")
							return err
						}
						return nil
					})
				}
				result, err := service.UpdateRoadmap(cmd.Context(), application.UpdateRoadmapInput{Body: body, Confirmed: true}, authorizer, planningEventSink{history: appCtx.History})
				if err != nil {
					return err
				}
				return appCtx.Output.Print(result, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\t%s\n", result.Action, result.Document.Path)
					return err
				})
			})
		},
	}
	roadmapEditCmd.Flags().StringVarP(&roadmapBody, "body", "b", "", "replacement body")
	roadmapEditCmd.Flags().BoolVar(&roadmapStdin, "stdin", false, "read replacement body from stdin")
	roadmapEditCmd.Flags().StringVar(&roadmapEditor, "editor", "", "editor command")
	roadmapEditCmd.Flags().BoolVar(&confirmRoadmap, "confirm", false, "confirm the roadmap write")
	roadmapCmd.AddCommand(roadmapShowCmd, roadmapEditCmd)

	brainstormCmd := &cobra.Command{
		Use:   "brainstorm",
		Short: "Read and create local Planning brainstorms",
	}
	brainstormListCmd := &cobra.Command{
		Use:   "list",
		Short: "List local brainstorms",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				documents, err := service.ListBrainstorms(cmd.Context())
				if err != nil {
					return err
				}
				items := make([]planningArtifactOutput, 0, len(documents))
				for _, document := range documents {
					items = append(items, brainstormOutput(document, false))
				}
				return appCtx.Output.Print(items, func(w io.Writer) error {
					if len(items) == 0 {
						_, err := fmt.Fprintln(w, "No brainstorms.")
						return err
					}
					for _, item := range items {
						if _, err := fmt.Fprintf(w, "%s\t%s\n", item.ID, item.Title); err != nil {
							return err
						}
					}
					return nil
				})
			})
		},
	}
	brainstormShowCmd := &cobra.Command{
		Use:   "show <slug>",
		Short: "Show one local brainstorm",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				document, err := service.GetBrainstorm(cmd.Context(), planning.ArtifactID(args[0]))
				if err != nil {
					return err
				}
				item := brainstormOutput(document, true)
				return appCtx.Output.Print(item, func(w io.Writer) error {
					_, err := fmt.Fprint(w, document.Body)
					return err
				})
			})
		},
	}
	var confirmBrainstorm bool
	brainstormStartCmd := &cobra.Command{
		Use:   "start <title>",
		Short: "Preview or create one local brainstorm",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionBrainstorm, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, moduleID string) error {
				preview, err := service.PreviewBrainstorm(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				if !confirmBrainstorm || preview.Action == application.MutationUnchanged {
					item := planningMutationOutput{
						Action: preview.Action,
						Path:   preview.Document.Path,
						ID:     string(preview.Document.Artifact.ID),
						Title:  preview.Document.Artifact.Title,
					}
					return appCtx.Output.Print(item, func(w io.Writer) error {
						if _, err := fmt.Fprintf(w, "%s\t%s\n", item.Action, item.Path); err != nil {
							return err
						}
						if preview.Action == application.MutationCreate {
							_, err := fmt.Fprintln(w, "Preview only. Rerun with --confirm to write.")
							return err
						}
						return nil
					})
				}
				result, err := service.CreateBrainstorm(
					cmd.Context(),
					application.CreateBrainstormInput{Title: args[0], Confirmed: true},
					authorizer,
					planningEventSink{history: appCtx.History},
				)
				if err != nil {
					return err
				}
				item := planningMutationOutput{
					Action: result.Action,
					Path:   result.Document.Path,
					ID:     string(result.Document.Artifact.ID),
					Title:  result.Document.Artifact.Title,
					Event:  result.Event,
				}
				return appCtx.Output.Print(item, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\t%s\n", item.Action, item.Path)
					return err
				})
			})
		},
	}
	brainstormStartCmd.Flags().BoolVar(&confirmBrainstorm, "confirm", false, "confirm the brainstorm write")
	brainstormCmd.AddCommand(brainstormListCmd, brainstormShowCmd, brainstormStartCmd)

	specCmd := &cobra.Command{
		Use:   "spec",
		Short: "Read local Planning specs",
	}
	specListCmd := &cobra.Command{
		Use:   "list",
		Short: "List local specs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				documents, err := service.ListSpecs(cmd.Context())
				if err != nil {
					return err
				}
				items := make([]planningArtifactOutput, 0, len(documents))
				for _, document := range documents {
					items = append(items, specOutput(document, false))
				}
				return appCtx.Output.Print(items, func(w io.Writer) error {
					if len(items) == 0 {
						_, err := fmt.Fprintln(w, "No specs.")
						return err
					}
					for _, item := range items {
						if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", item.ID, item.Status, item.Title); err != nil {
							return err
						}
					}
					return nil
				})
			})
		},
	}
	specShowCmd := &cobra.Command{
		Use:   "show <slug>",
		Short: "Show one local spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				document, err := service.GetSpec(cmd.Context(), planning.ArtifactID(args[0]))
				if err != nil {
					return err
				}
				item := specOutput(document, true)
				return appCtx.Output.Print(item, func(w io.Writer) error {
					_, err := fmt.Fprint(w, document.Body)
					return err
				})
			})
		},
	}
	specCmd.AddCommand(specListCmd, specShowCmd)

	planCmd.AddCommand(statusCmd, checkCmd, roadmapCmd, brainstormCmd, specCmd)
	root.AddCommand(planCmd)
}

func resolvePlanningCheckInput(args []string) (application.CheckInput, error) {
	switch len(args) {
	case 0:
		return application.CheckInput{}, nil
	case 1:
		if args[0] == "project" {
			return application.CheckInput{}, nil
		}
		return application.CheckInput{}, fmt.Errorf("unsupported check scope %q", args[0])
	case 2:
		if args[0] != "spec" {
			return application.CheckInput{}, fmt.Errorf("unsupported check scope %q", args[0])
		}
		id := planning.ArtifactID(args[1])
		return application.CheckInput{SpecID: &id}, nil
	default:
		return application.CheckInput{}, fmt.Errorf("invalid check scope")
	}
}

func printPlanningCheckReport(w io.Writer, report application.CheckReport) error {
	if _, err := fmt.Fprintf(w, "check_scope: %s\n", report.Scope); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "findings: %d total, %d blocking, %d guidance\n", len(report.Findings), report.ErrorCount(), report.WarningCount()); err != nil {
		return err
	}
	if len(report.Findings) == 0 {
		_, err := fmt.Fprintln(w, "status: ok")
		return err
	}
	for _, finding := range report.Findings {
		if _, err := fmt.Fprintf(w, "- [%s] %s %s :: %s\n", finding.Severity, finding.ArtifactType, finding.ArtifactPath, finding.Section); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  %s\n", finding.Message); err != nil {
			return err
		}
		if finding.Suggestion != "" {
			if _, err := fmt.Fprintf(w, "  fix: %s\n", finding.Suggestion); err != nil {
				return err
			}
		}
	}
	return nil
}

func editPlanningText(initial, editor string) (string, error) {
	command := strings.TrimSpace(editor)
	if command == "" {
		command = strings.TrimSpace(os.Getenv("VISUAL"))
	}
	if command == "" {
		command = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if command == "" {
		return "", errors.New("no editor configured (set $EDITOR or use --editor)")
	}
	file, err := os.CreateTemp("", "brain-plan-roadmap-*.md")
	if err != nil {
		return "", err
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := file.WriteString(initial); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	parts := strings.Fields(command)
	process := exec.Command(parts[0], append(parts[1:], path)...)
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	if err := process.Run(); err != nil {
		return "", fmt.Errorf("editor %s: %w", parts[0], err)
	}
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

type planningArtifactOutput struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status,omitempty"`
	Summary string `json:"summary,omitempty"`
	Path    string `json:"path"`
	Body    string `json:"body,omitempty"`
}

type planningMutationOutput struct {
	Action application.MutationAction `json:"action"`
	Path   string                     `json:"path"`
	ID     string                     `json:"id"`
	Title  string                     `json:"title"`
	Event  *application.Event         `json:"event,omitempty"`
}

func brainstormOutput(document application.BrainstormDocument, includeBody bool) planningArtifactOutput {
	item := planningArtifactOutput{
		Kind:    string(planning.ArtifactBrainstorm),
		ID:      string(document.Artifact.ID),
		Title:   document.Artifact.Title,
		Summary: document.Artifact.Summary,
		Path:    document.Path,
	}
	if includeBody {
		item.Body = document.Body
	}
	return item
}

func specOutput(document application.SpecDocument, includeBody bool) planningArtifactOutput {
	item := planningArtifactOutput{
		Kind:   string(planning.ArtifactSpec),
		ID:     string(document.Artifact.ID),
		Title:  document.Artifact.Title,
		Status: string(document.Artifact.Status),
		Path:   document.Path,
	}
	if includeBody {
		item.Body = document.Body
	}
	return item
}

type planningRun func(*app.App, *application.Service, planningAuthorizer, string) error

type planningServiceProvider interface {
	PlanningService() *application.Service
}

func withPlanningService(cmd *cobra.Command, loadApp appLoader, permission string, run planningRun) error {
	appCtx, err := loadApp()
	if err != nil {
		return err
	}
	defer appCtx.Close()
	resolution, err := appCtx.Modules.ResolveCommand(cmd.Context(), planningCommandGroup, permission)
	if err != nil {
		return err
	}
	provider, ok := resolution.Module.(planningServiceProvider)
	if !ok || provider.PlanningService() == nil {
		return fmt.Errorf("module %s does not provide Planning services", resolution.ModuleID)
	}
	return run(
		appCtx,
		provider.PlanningService(),
		planningAuthorizer{runtime: appCtx.Modules, moduleID: resolution.ModuleID},
		resolution.ModuleID,
	)
}

type planningAuthorizer struct {
	runtime  *modules.Runtime
	moduleID string
}

func (a planningAuthorizer) Require(ctx context.Context, permission string) error {
	if a.runtime == nil {
		return fmt.Errorf("Planning permission runtime is unavailable")
	}
	return a.runtime.RequirePermission(ctx, a.moduleID, permission)
}

type planningEventSink struct {
	history *history.Logger
}

func (s planningEventSink) Publish(_ context.Context, event application.Event) error {
	if s.history == nil {
		return fmt.Errorf("Planning audit history is unavailable")
	}
	target := string(event.Artifact.ID)
	file := ".plan/brainstorms/" + target + ".md"
	summary := "Planning brainstorm created"
	if event.Artifact.Kind == planning.ArtifactRoadmap {
		file = ".plan/ROADMAP.md"
		summary = "Planning roadmap updated"
	}
	return s.history.Append(history.Entry{
		ID:        strings.Join([]string{event.ModuleID, event.Name, target, fmt.Sprintf("%d", event.OccurredAt.UnixNano())}, ":"),
		Timestamp: event.OccurredAt,
		Operation: event.Name,
		File:      file,
		Target:    target,
		Summary:   summary,
		Metadata: map[string]any{
			"module_id":     event.ModuleID,
			"artifact_kind": event.Artifact.Kind,
			"outcome":       event.Outcome,
		},
	})
}
