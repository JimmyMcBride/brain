package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"brain/internal/app"
	"brain/internal/history"
	"brain/internal/modules"
	"brain/internal/planning"
	"brain/internal/planning/application"

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
		Short: "Inspect local Planning workspace compatibility",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withPlanningService(cmd, loadApp, application.PermissionRead, func(appCtx *app.App, service *application.Service, _ planningAuthorizer, _ string) error {
				status, err := service.Status(cmd.Context())
				if err != nil {
					return err
				}
				return appCtx.Output.Print(status, func(w io.Writer) error {
					if _, err := fmt.Fprintf(w, "state: %s\n", status.State); err != nil {
						return err
					}
					if _, err := fmt.Fprintf(w, "writable: %t\n", status.Writable); err != nil {
						return err
					}
					if status.SchemaVersion != 0 {
						if _, err := fmt.Fprintf(w, "schema version: %d\n", status.SchemaVersion); err != nil {
							return err
						}
					}
					if status.Ownership != "" {
						if _, err := fmt.Fprintf(w, "ownership: %s\n", status.Ownership); err != nil {
							return err
						}
					}
					if _, err := fmt.Fprintf(w, "message: %s\n", status.Message); err != nil {
						return err
					}
					for _, guidance := range status.Guidance {
						if _, err := fmt.Fprintf(w, "guidance: %s\n", guidance); err != nil {
							return err
						}
					}
					return nil
				})
			})
		},
	}

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

	planCmd.AddCommand(statusCmd, brainstormCmd, specCmd)
	root.AddCommand(planCmd)
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
	provider, ok := resolution.Module.(application.ServiceProvider)
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
	return s.history.Append(history.Entry{
		ID:        strings.Join([]string{event.ModuleID, event.Name, target, fmt.Sprintf("%d", event.OccurredAt.UnixNano())}, ":"),
		Timestamp: event.OccurredAt,
		Operation: event.Name,
		File:      ".plan/brainstorms/" + target + ".md",
		Target:    target,
		Summary:   "Planning brainstorm created",
		Metadata: map[string]any{
			"module_id":     event.ModuleID,
			"artifact_kind": event.Artifact.Kind,
			"outcome":       event.Outcome,
		},
	})
}
