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
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, _ string) error {
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
	addPlanningBrainstormWorkflowCommands(brainstormCmd, loadApp)

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
	addPlanningSpecWorkflowCommands(specCmd, loadApp)

	guideCmd := newPlanningGuideCommand(loadApp)
	planCmd.AddCommand(statusCmd, checkCmd, roadmapCmd, brainstormCmd, guideCmd, specCmd)
	root.AddCommand(planCmd)
}

func addPlanningSpecWorkflowCommands(specCmd *cobra.Command, loadApp appLoader) {
	var editBody string
	var editStdin, editConfirm bool
	var editEditor string
	edit := &cobra.Command{Use: "edit <spec-slug>", Short: "Preview or replace canonical spec Markdown", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, auth planningAuthorizer, _ string) error {
			usingEditor := !editStdin && editBody == ""
			if usingEditor && !editConfirm {
				return fmt.Errorf("%w: pass --confirm before opening the spec editor", application.ErrConfirmationRequired)
			}
			body, err := readBody(cmd.InOrStdin(), editBody, editStdin)
			if err != nil {
				return err
			}
			if usingEditor {
				current, err := service.GetSpec(cmd.Context(), planning.ArtifactID(args[0]))
				if err != nil {
					return err
				}
				body, err = editPlanningText(current.Body, editEditor)
				if err != nil {
					return err
				}
			}
			input := application.SpecEditInput{ID: planning.ArtifactID(args[0]), Body: body, Confirmed: editConfirm}
			var result application.SpecMutationResult
			if editConfirm {
				result, err = service.EditSpec(cmd.Context(), input, auth, planningEventSink{history: appCtx.History})
			} else {
				result, err = service.PreviewSpecEdit(cmd.Context(), input)
			}
			if err != nil {
				return err
			}
			return printSpecMutation(appCtx, result, !editConfirm)
		})
	}}
	edit.Flags().StringVarP(&editBody, "body", "b", "", "replacement body")
	edit.Flags().BoolVar(&editStdin, "stdin", false, "read replacement body from stdin")
	edit.Flags().StringVar(&editEditor, "editor", "", "editor command")
	edit.Flags().BoolVar(&editConfirm, "confirm", false, "confirm the spec write")

	var setStatus string
	var statusConfirm bool
	status := &cobra.Command{Use: "status <spec-slug>", Short: "Preview or set spec status", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, auth planningAuthorizer, _ string) error {
			input := application.SpecStatusInput{ID: planning.ArtifactID(args[0]), Status: planning.SpecStatus(setStatus), Confirmed: statusConfirm}
			var result application.SpecMutationResult
			var err error
			if statusConfirm {
				result, err = service.SetSpecStatus(cmd.Context(), input, auth, planningEventSink{history: appCtx.History})
			} else {
				result, err = service.PreviewSpecStatus(cmd.Context(), input)
			}
			if err != nil {
				return err
			}
			return printSpecMutation(appCtx, result, !statusConfirm)
		})
	}}
	status.Flags().StringVar(&setStatus, "set", "", "new status: draft, approved, done")
	_ = status.MarkFlagRequired("set")
	status.Flags().BoolVar(&statusConfirm, "confirm", false, "confirm the status write")

	var analyzeConfirm bool
	analyze := &cobra.Command{Use: "analyze <spec-slug>", Short: "Pressure-test a spec and preview or retain its additive report", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, auth planningAuthorizer, _ string) error {
			var report application.SpecAnalysisReport
			var err error
			if analyzeConfirm {
				report, err = service.AnalyzeSpec(cmd.Context(), planning.ArtifactID(args[0]), true, auth, planningEventSink{history: appCtx.History})
			} else {
				report, err = service.PreviewSpecAnalysis(cmd.Context(), planning.ArtifactID(args[0]))
			}
			if err != nil {
				return err
			}
			if err := appCtx.Output.Print(report, func(w io.Writer) error { return renderSpecAnalysis(w, report, !analyzeConfirm) }); err != nil {
				return err
			}
			if report.ErrorCount() > 0 {
				return fmt.Errorf("spec analysis found %d blocking issue(s)", report.ErrorCount())
			}
			return nil
		})
	}}
	analyze.Flags().BoolVar(&analyzeConfirm, "confirm", false, "retain the additive analysis report")

	var profile string
	var checklistConfirm bool
	checklist := &cobra.Command{Use: "checklist <spec-slug>", Short: "Run a profile-driven spec checklist", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, auth planningAuthorizer, _ string) error {
			var report application.SpecChecklistReport
			var err error
			if checklistConfirm {
				report, err = service.RunSpecChecklist(cmd.Context(), planning.ArtifactID(args[0]), profile, true, auth, planningEventSink{history: appCtx.History})
			} else {
				report, err = service.PreviewSpecChecklist(cmd.Context(), planning.ArtifactID(args[0]), profile)
			}
			if err != nil {
				return err
			}
			if err := appCtx.Output.Print(report, func(w io.Writer) error { return renderSpecChecklist(w, report, !checklistConfirm) }); err != nil {
				return err
			}
			if report.ErrorCount() > 0 {
				return fmt.Errorf("spec checklist found %d blocking issue(s)", report.ErrorCount())
			}
			return nil
		})
	}}
	checklist.Flags().StringVar(&profile, "profile", "general", "checklist profile")
	checklist.Flags().BoolVar(&checklistConfirm, "confirm", false, "retain the additive checklist report")

	var initiativeSlug, initiativeTitle, initiativeSummary string
	var initiativeClear, initiativeConfirm bool
	initiative := &cobra.Command{Use: "initiative <spec-slug>", Short: "Preview or update initiative metadata", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, auth planningAuthorizer, _ string) error {
			var id *planning.ArtifactID
			if strings.TrimSpace(initiativeSlug) != "" {
				value := planning.ArtifactID(initiativeSlug)
				id = &value
			}
			input := application.SpecInitiativeInput{ID: planning.ArtifactID(args[0]), Initiative: id, Title: initiativeTitle, Summary: initiativeSummary, Clear: initiativeClear, Confirmed: initiativeConfirm}
			var result application.SpecMutationResult
			var err error
			if initiativeConfirm {
				result, err = service.SetSpecInitiative(cmd.Context(), input, auth, planningEventSink{history: appCtx.History})
			} else {
				result, err = service.PreviewSpecInitiative(cmd.Context(), input)
			}
			if err != nil {
				return err
			}
			return printSpecMutation(appCtx, result, !initiativeConfirm)
		})
	}}
	initiative.Flags().StringVar(&initiativeSlug, "set", "", "initiative slug")
	initiative.Flags().StringVar(&initiativeTitle, "title", "", "initiative title")
	initiative.Flags().StringVar(&initiativeSummary, "summary", "", "initiative summary")
	initiative.Flags().BoolVar(&initiativeClear, "clear", false, "clear initiative metadata")
	initiative.Flags().BoolVar(&initiativeConfirm, "confirm", false, "confirm initiative metadata write")

	var branchPrefix string
	var executeConfirm bool
	execute := &cobra.Command{Use: "execute <spec-slug>", Short: "Preview or start spec execution with ephemeral slices", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, auth planningAuthorizer, _ string) error {
			input := application.SpecExecutionInput{ID: planning.ArtifactID(args[0]), BranchPrefix: branchPrefix, Confirmed: executeConfirm}
			var result application.SpecExecutionResult
			var err error
			if executeConfirm {
				result, err = service.BeginSpecExecution(cmd.Context(), input, auth, planningEventSink{history: appCtx.History})
			} else {
				result, err = service.PreviewSpecExecution(cmd.Context(), input)
			}
			if err != nil {
				return err
			}
			return appCtx.Output.Print(result, func(w io.Writer) error { return renderSpecExecution(w, result, !executeConfirm) })
		})
	}}
	execute.Flags().StringVar(&branchPrefix, "branch-prefix", "feature/", "suggested branch prefix")
	execute.Flags().BoolVar(&executeConfirm, "confirm", false, "confirm execution start")
	var handoffPrefix string
	var handoffConfirm bool
	handoff := &cobra.Command{Use: "handoff <spec-slug>", Short: "Preview or continue a guided spec into execution", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, auth planningAuthorizer, _ string) error {
			input := application.SpecExecutionInput{ID: planning.ArtifactID(args[0]), BranchPrefix: handoffPrefix, Confirmed: handoffConfirm}
			var result application.SpecExecutionResult
			var err error
			if handoffConfirm {
				result, err = service.HandoffSpec(cmd.Context(), input, auth, planningEventSink{history: appCtx.History})
			} else {
				result, err = service.PreviewSpecHandoff(cmd.Context(), input)
			}
			if err != nil {
				return err
			}
			return appCtx.Output.Print(result, func(w io.Writer) error {
				if result.Session != nil {
					fmt.Fprintf(w, "Spec recap:\nCurrent understanding: %s\nRecommended next stage: continue into execution.\n", result.Recap)
				}
				return renderSpecExecution(w, result, !handoffConfirm)
			})
		})
	}}
	handoff.Flags().StringVar(&handoffPrefix, "branch-prefix", "feature/", "suggested branch prefix")
	handoff.Flags().BoolVar(&handoffConfirm, "confirm", false, "confirm guided execution handoff")

	specCmd.AddCommand(edit, status, analyze, checklist, initiative, execute, handoff)
}

func printSpecMutation(appCtx *app.App, result application.SpecMutationResult, preview bool) error {
	return appCtx.Output.Print(result, func(w io.Writer) error {
		if preview {
			fmt.Fprintln(w, "Preview only; rerun with --confirm to apply.")
		}
		_, err := fmt.Fprintf(w, "%s\t%s\n", result.Action, result.Document.Path)
		return err
	})
}
func renderSpecAnalysis(w io.Writer, report application.SpecAnalysisReport, preview bool) error {
	fmt.Fprintf(w, "spec_analysis: %s\nfindings: %d total, %d blocking, %d guidance\n", report.SpecPath, len(report.Findings), report.ErrorCount(), report.WarningCount())
	if len(report.Findings) == 0 {
		fmt.Fprintln(w, "status: ok")
	}
	for _, f := range report.Findings {
		fmt.Fprintf(w, "- [%s] %s: %s\n", f.Severity, f.Category, f.Message)
		if f.Recommendation != "" {
			fmt.Fprintln(w, "  fix: "+f.Recommendation)
		}
	}
	if preview {
		fmt.Fprintln(w, "Preview only; rerun with --confirm to retain the report.")
	}
	return nil
}
func renderSpecChecklist(w io.Writer, report application.SpecChecklistReport, preview bool) error {
	fmt.Fprintf(w, "spec_checklist: %s\nprofile: %s\nfindings: %d total, %d blocking, %d guidance\n", report.SpecPath, report.Profile, len(report.Findings), report.ErrorCount(), report.WarningCount())
	if len(report.Findings) == 0 {
		fmt.Fprintln(w, "status: ok")
	}
	for _, f := range report.Findings {
		fmt.Fprintf(w, "- [%s] %s: %s\n", f.Severity, f.Area, f.Message)
		if f.Recommendation != "" {
			fmt.Fprintln(w, "  fix: "+f.Recommendation)
		}
	}
	if preview {
		fmt.Fprintln(w, "Preview only; rerun with --confirm to retain the report.")
	}
	return nil
}
func renderSpecExecution(w io.Writer, result application.SpecExecutionResult, preview bool) error {
	view := result.Execution
	fmt.Fprintf(w, "spec_execution: %s\nstatus: %s\nbranch: %s\nslices: %d\n", view.SpecPath, view.Status, view.SuggestedBranch, len(view.Slices))
	for i, slice := range view.Slices {
		fmt.Fprintf(w, "%d. %s\n   goal: %s\n", i+1, slice.Title, slice.Goal)
		for _, verify := range slice.Verification {
			fmt.Fprintln(w, "   verify: "+verify)
		}
	}
	fmt.Fprintln(w, "workflow:\n- implement one slice at a time\n- review and verify each slice before committing it\n- open a PR after the full spec is built")
	if preview {
		fmt.Fprintln(w, "Preview only; rerun with --confirm to start execution.")
	}
	return nil
}

func addPlanningBrainstormWorkflowCommands(brainstormCmd *cobra.Command, loadApp appLoader) {
	var ideaBody string
	var ideaStdin bool
	var ideaSection string
	var confirmIdea bool
	ideaCmd := &cobra.Command{
		Use: "idea <brainstorm-slug>", Short: "Preview or append an idea to a local brainstorm", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readBody(cmd.InOrStdin(), ideaBody, ideaStdin)
			if err != nil {
				return err
			}
			input := application.BrainstormUpdateInput{ID: planning.ArtifactID(args[0]), Section: ideaSection, Body: body, Confirmed: confirmIdea}
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, _ string) error {
				preview, err := service.PreviewBrainstormUpdate(cmd.Context(), input)
				if err != nil {
					return err
				}
				if !confirmIdea || preview.Action == application.MutationUnchanged {
					return printBrainstormMutation(appCtx, preview, !confirmIdea)
				}
				result, err := service.UpdateBrainstorm(cmd.Context(), input, authorizer, planningEventSink{history: appCtx.History})
				if err != nil {
					return err
				}
				return printBrainstormMutation(appCtx, result, false)
			})
		},
	}
	ideaCmd.Flags().StringVarP(&ideaBody, "body", "b", "", "idea body")
	ideaCmd.Flags().BoolVar(&ideaStdin, "stdin", false, "read idea body from stdin")
	ideaCmd.Flags().StringVar(&ideaSection, "section", "ideas", "brainstorm section")
	ideaCmd.Flags().BoolVar(&confirmIdea, "confirm", false, "confirm the brainstorm write")

	var refinement application.BrainstormRefinementInput
	refineCmd := &cobra.Command{
		Use: "refine <brainstorm-slug>", Short: "Preview or apply structured brainstorm refinement", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			refinement.ID = planning.ArtifactID(args[0])
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, _ string) error {
				preview, err := service.PreviewBrainstormRefinement(cmd.Context(), refinement)
				if err != nil {
					return err
				}
				if !refinement.Confirmed || preview.Action == application.MutationUnchanged {
					return printBrainstormMutation(appCtx, preview, !refinement.Confirmed)
				}
				result, err := service.RefineBrainstorm(cmd.Context(), refinement, authorizer, planningEventSink{history: appCtx.History})
				if err != nil {
					return err
				}
				return printBrainstormMutation(appCtx, result, false)
			})
		},
	}
	refineCmd.Flags().StringVar(&refinement.Problem, "problem", "", "core problem")
	refineCmd.Flags().StringVar(&refinement.UserValue, "user-value", "", "user and value")
	refineCmd.Flags().StringVar(&refinement.Constraints, "constraints", "", "newline-separated constraints")
	refineCmd.Flags().StringVar(&refinement.Appetite, "appetite", "", "scope appetite")
	refineCmd.Flags().StringVar(&refinement.RemainingOpenQuestions, "open-questions", "", "newline-separated open questions")
	refineCmd.Flags().StringVar(&refinement.CandidateApproaches, "approaches", "", "newline-separated candidate approaches")
	refineCmd.Flags().StringVar(&refinement.DecisionSnapshot, "decision", "", "decision snapshot")
	refineCmd.Flags().BoolVar(&refinement.Confirmed, "confirm", false, "confirm the brainstorm write")

	var challenge application.BrainstormChallengeInput
	challengeCmd := &cobra.Command{
		Use: "challenge <brainstorm-slug>", Short: "Preview or apply a structured challenge pass", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			challenge.ID = planning.ArtifactID(args[0])
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, _ string) error {
				preview, err := service.PreviewBrainstormChallenge(cmd.Context(), challenge)
				if err != nil {
					return err
				}
				if !challenge.Confirmed || preview.Action == application.MutationUnchanged {
					return printBrainstormMutation(appCtx, preview, !challenge.Confirmed)
				}
				result, err := service.ChallengeBrainstorm(cmd.Context(), challenge, authorizer, planningEventSink{history: appCtx.History})
				if err != nil {
					return err
				}
				return printBrainstormMutation(appCtx, result, false)
			})
		},
	}
	challengeCmd.Flags().StringVar(&challenge.RabbitHoles, "rabbit-holes", "", "newline-separated rabbit holes")
	challengeCmd.Flags().StringVar(&challenge.NoGos, "no-gos", "", "newline-separated no-gos")
	challengeCmd.Flags().StringVar(&challenge.Assumptions, "assumptions", "", "newline-separated assumptions")
	challengeCmd.Flags().StringVar(&challenge.LikelyOverengineering, "overengineering", "", "likely overengineering")
	challengeCmd.Flags().StringVar(&challenge.SimplerAlternative, "simpler-alternative", "", "simpler alternative")
	challengeCmd.Flags().BoolVar(&challenge.Confirmed, "confirm", false, "confirm the brainstorm write")

	resumeCmd := &cobra.Command{
		Use: "resume [brainstorm-slug]", Short: "Show the active guided brainstorm session", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				var session application.GuidedSessionRecord
				var err error
				if len(args) == 1 {
					session, err = service.GetGuidedSession(cmd.Context(), args[0])
				} else {
					session, err = service.CurrentGuidedSession(cmd.Context())
				}
				if err != nil {
					return err
				}
				return appCtx.Output.Print(session, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "Resuming %s\nSummary: %s\nNext: %s\n", session.ChainID, session.Summary, session.NextAction)
					return err
				})
			})
		},
	}

	sessionsCmd := &cobra.Command{
		Use: "sessions", Short: "List guided brainstorm sessions", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				sessions, err := service.ListGuidedSessions(cmd.Context())
				if err != nil {
					return err
				}
				current, _ := service.CurrentGuidedSession(cmd.Context())
				return appCtx.Output.Print(sessions, func(w io.Writer) error {
					for _, session := range sessions {
						marker := " "
						if session.ChainID == current.ChainID {
							marker = "*"
						}
						if _, err := fmt.Fprintf(w, "%s %s stage=%s next=%s\n", marker, session.ChainID, session.CurrentStage, session.NextAction); err != nil {
							return err
						}
					}
					return nil
				})
			})
		},
	}

	brainstormCmd.AddCommand(ideaCmd, refineCmd, challengeCmd, resumeCmd, sessionsCmd)
	addPlanningGuidedMutationCommands(brainstormCmd, loadApp)
	addPlanningPromotionCommands(brainstormCmd, loadApp)
}

func addPlanningGuidedMutationCommands(brainstormCmd *cobra.Command, loadApp appLoader) {
	var confirmSwitch bool
	switchCmd := &cobra.Command{
		Use: "switch <brainstorm-slug>", Short: "Switch the active guided session", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, _ string) error {
				if !confirmSwitch {
					session, err := service.GetGuidedSession(cmd.Context(), args[0])
					if err != nil {
						return err
					}
					return appCtx.Output.Print(session, func(w io.Writer) error {
						_, err := fmt.Fprintf(w, "Preview switch to %s. Rerun with --confirm.\n", session.ChainID)
						return err
					})
				}
				result, err := service.SwitchGuidedSession(cmd.Context(), application.GuidedSessionMutationInput{ChainID: args[0], Confirmed: true}, authorizer, planningEventSink{history: appCtx.History})
				if err != nil {
					return err
				}
				return appCtx.Output.Print(result, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\t%s\n", result.Action, result.Session.ChainID)
					return err
				})
			})
		},
	}
	switchCmd.Flags().BoolVar(&confirmSwitch, "confirm", false, "confirm the guided-session write")

	var confirmReopen bool
	reopenCmd := &cobra.Command{
		Use: "reopen <brainstorm-slug> <stage>", Short: "Reopen a stage and mark downstream review", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmReopen {
				return fmt.Errorf("%w: rerun guided stage reopen with --confirm", application.ErrConfirmationRequired)
			}
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, _ string) error {
				result, err := service.ReopenGuidedSession(cmd.Context(), application.GuidedSessionMutationInput{ChainID: args[0], Stage: args[1], Confirmed: true}, authorizer, planningEventSink{history: appCtx.History})
				if err != nil {
					return err
				}
				return appCtx.Output.Print(result, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\t%s impacted=%s\n", result.Action, result.Session.ChainID, strings.Join(result.Impacted, ","))
					return err
				})
			})
		},
	}
	reopenCmd.Flags().BoolVar(&confirmReopen, "confirm", false, "confirm the guided-session write")

	var confirmReview bool
	reviewCmd := &cobra.Command{
		Use: "review <brainstorm-slug>", Short: "Review downstream guided stages", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmReview {
				return fmt.Errorf("%w: rerun guided stage review with --confirm", application.ErrConfirmationRequired)
			}
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, _ string) error {
				result, err := service.ReviewGuidedSession(cmd.Context(), application.GuidedSessionMutationInput{ChainID: args[0], Confirmed: true}, authorizer, planningEventSink{history: appCtx.History})
				if err != nil {
					return err
				}
				return appCtx.Output.Print(result, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\t%s reviewed=%s\n", result.Action, result.Session.ChainID, strings.Join(result.Impacted, ","))
					return err
				})
			})
		},
	}
	reviewCmd.Flags().BoolVar(&confirmReview, "confirm", false, "confirm the guided-session write")

	var parkValue, parkReason, parkUnlock string
	var confirmPark bool
	parkCmd := &cobra.Command{
		Use: "park <brainstorm-slug> <title>", Short: "Preview or park a roadmap idea", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := application.RoadmapParkingInput{BrainstormID: planning.ArtifactID(args[0]), Title: args[1], Value: parkValue, Reason: parkReason, Unlock: parkUnlock, Confirmed: confirmPark}
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, _ string) error {
				preview, err := service.PreviewRoadmapParking(cmd.Context(), input)
				if err != nil {
					return err
				}
				if !confirmPark || preview.Action == application.MutationUnchanged {
					return appCtx.Output.Print(preview, func(w io.Writer) error {
						_, err := fmt.Fprintf(w, "%s\t%s\n", preview.Action, preview.Document.Path)
						return err
					})
				}
				result, err := service.ParkRoadmap(cmd.Context(), input, authorizer, planningEventSink{history: appCtx.History})
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
	parkCmd.Flags().StringVar(&parkValue, "value", "", "future value")
	parkCmd.Flags().StringVar(&parkReason, "reason", "", "why parked")
	parkCmd.Flags().StringVar(&parkUnlock, "unlock", "", "unlock condition")
	parkCmd.Flags().BoolVar(&confirmPark, "confirm", false, "confirm the roadmap write")
	brainstormCmd.AddCommand(switchCmd, reopenCmd, reviewCmd, parkCmd)
}

func addPlanningPromotionCommands(brainstormCmd *cobra.Command, loadApp appLoader) {
	assessCmd := &cobra.Command{
		Use: "assess <brainstorm-slug>", Short: "Assess local brainstorm promotion maturity", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				assessment, err := service.AssessLocalBrainstorm(cmd.Context(), planning.ArtifactID(args[0]))
				if err != nil {
					return err
				}
				return appCtx.Output.Print(assessment, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\t%s\n", assessment.Decision.State, assessment.Decision.Reason)
					return err
				})
			})
		},
	}
	var confirmPromote bool
	promoteCmd := &cobra.Command{
		Use: "promote <brainstorm-slug>", Short: "Preview or apply direct local spec promotion", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := planning.ArtifactID(args[0])
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, _ string) error {
				if !confirmPromote {
					draft, err := service.PreviewLocalPromotion(cmd.Context(), id)
					if err != nil {
						return err
					}
					return appCtx.Output.Print(draft, func(w io.Writer) error {
						_, err := fmt.Fprintf(w, "%s\t%d spec action(s)\n", draft.PromotionDecision, len(draft.ProposedSpecs))
						return err
					})
				}
				result, err := service.PromoteLocalBrainstorm(cmd.Context(), application.LocalPromotionInput{BrainstormID: id, Confirmed: true}, authorizer, planningEventSink{history: appCtx.History})
				if err != nil {
					return err
				}
				return appCtx.Output.Print(result, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\t%d spec(s)\n", result.Action, len(result.Specs))
					return err
				})
			})
		},
	}
	promoteCmd.Flags().BoolVar(&confirmPromote, "confirm", false, "confirm direct local spec writes")

	var repairSpecs []string
	var confirmRepair bool
	repairCmd := &cobra.Command{
		Use: "repair <brainstorm-slug>", Short: "Preview or repair the local spec split", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, authorizer planningAuthorizer, _ string) error {
				input := application.LocalPromotionRepairInput{BrainstormID: planning.ArtifactID(args[0]), Specs: repairSpecs, Confirmed: confirmRepair}
				if !confirmRepair {
					result, err := service.PreviewLocalPromotionRepair(cmd.Context(), input)
					if err != nil {
						return err
					}
					return appCtx.Output.Print(result, func(w io.Writer) error {
						if _, err := fmt.Fprintf(w, "%s\t%s\n", result.Action, result.UpdatedPath); err != nil {
							return err
						}
						if result.Action != application.MutationUnchanged {
							_, err := fmt.Fprintln(w, "Preview only. Rerun with --confirm to write.")
							return err
						}
						return nil
					})
				}
				result, err := service.RepairLocalPromotionSource(cmd.Context(), application.LocalPromotionRepairInput{BrainstormID: planning.ArtifactID(args[0]), Specs: repairSpecs, Confirmed: true}, authorizer, planningEventSink{history: appCtx.History})
				if err != nil {
					return err
				}
				return appCtx.Output.Print(result, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\t%s\n", result.Action, result.UpdatedPath)
					return err
				})
			})
		},
	}
	repairCmd.Flags().StringArrayVar(&repairSpecs, "spec", nil, "spec title; repeat for each spec")
	repairCmd.Flags().BoolVar(&confirmRepair, "confirm", false, "confirm the brainstorm write")
	brainstormCmd.AddCommand(assessCmd, promoteCmd, repairCmd)
}

func newPlanningGuideCommand(loadApp appLoader) *cobra.Command {
	guideCmd := &cobra.Command{Use: "guide", Short: "Render local Planning guide packets"}
	currentCmd := &cobra.Command{
		Use: "current", Short: "Render the current local guide packet", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				packet, err := service.CurrentGuidePacket(cmd.Context())
				if err != nil {
					return err
				}
				return appCtx.Output.Print(packet, func(w io.Writer) error { _, err := fmt.Fprintln(w, packet.RenderedPrompt); return err })
			})
		},
	}
	var chainID, checkpoint string
	showCmd := &cobra.Command{
		Use: "show", Short: "Render one local guide packet", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(chainID) == "" {
				return fmt.Errorf("guide show requires --chain")
			}
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				packet, err := service.GuidePacketForChain(cmd.Context(), chainID, checkpoint)
				if err != nil {
					return err
				}
				return appCtx.Output.Print(packet, func(w io.Writer) error { _, err := fmt.Fprintln(w, packet.RenderedPrompt); return err })
			})
		},
	}
	showCmd.Flags().StringVar(&chainID, "chain", "", "guided session chain id")
	showCmd.Flags().StringVar(&checkpoint, "checkpoint", "", "guide checkpoint override")
	guideCmd.AddCommand(currentCmd, showCmd)
	return guideCmd
}

func printBrainstormMutation(appCtx *app.App, result application.BrainstormMutationResult, preview bool) error {
	return appCtx.Output.Print(result, func(w io.Writer) error {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", result.Action, result.Document.Path); err != nil {
			return err
		}
		if preview && result.Action != application.MutationUnchanged {
			_, err := fmt.Fprintln(w, "Preview only. Rerun with --confirm to write.")
			return err
		}
		return nil
	})
}

func resolvePlanningCheckInput(args []string) (application.CheckInput, error) {
	switch len(args) {
	case 0:
		return application.CheckInput{}, nil
	case 1:
		switch args[0] {
		case "project":
			return application.CheckInput{}, nil
		case "spec":
			return application.CheckInput{}, errors.New("check spec requires a slug")
		default:
			return application.CheckInput{}, fmt.Errorf("unsupported check scope %q", args[0])
		}
	case 2:
		switch args[0] {
		case "project":
			return application.CheckInput{}, errors.New("check project does not accept arguments")
		case "spec":
			id := planning.ArtifactID(args[1])
			return application.CheckInput{SpecID: &id}, nil
		default:
			return application.CheckInput{}, fmt.Errorf("unsupported check scope %q", args[0])
		}
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
	parts, err := splitEditorCommand(command)
	if err != nil {
		return "", err
	}
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

func splitEditorCommand(command string) ([]string, error) {
	var parts []string
	var part strings.Builder
	var quote rune
	hasPart := false
	flush := func() {
		if !hasPart {
			return
		}
		parts = append(parts, part.String())
		part.Reset()
		hasPart = false
	}
	for _, char := range command {
		if quote != 0 {
			if char == quote {
				quote = 0
				continue
			}
			part.WriteRune(char)
			hasPart = true
			continue
		}
		switch {
		case char == '\'' || char == '"':
			quote = char
			hasPart = true
		case char == ' ' || char == '\t' || char == '\r' || char == '\n':
			flush()
		default:
			part.WriteRune(char)
			hasPart = true
		}
	}
	if quote != 0 {
		return nil, errors.New("editor command has an unterminated quote")
	}
	flush()
	if len(parts) == 0 || parts[0] == "" {
		return nil, errors.New("editor command has no executable")
	}
	return parts, nil
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
	switch event.Name {
	case application.EventRoadmapUpdated:
		file = ".plan/ROADMAP.md"
		summary = "Planning roadmap updated"
	case application.EventBrainstormUpdated:
		summary = "Planning brainstorm updated"
	case application.EventGuidedSessionUpdated:
		file = ".plan/.meta/guided_sessions.json"
		summary = "Planning guided session updated"
	case application.EventBrainstormPromoted:
		summary = "Planning brainstorm promoted directly to local spec"
	case application.EventSpecUpdated:
		file = ".plan/specs/" + target + ".md"
		summary = "Planning spec updated"
	case application.EventSpecExecutionStarted:
		file = ".plan/specs/" + target + ".md"
		summary = "Planning spec execution started"
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
