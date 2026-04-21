package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"brain/internal/projectcontext"
	"brain/internal/session"
	"brain/internal/taskcontext"

	"github.com/spf13/cobra"
)

type prepSession struct {
	ID     string `json:"id"`
	Task   string `json:"task"`
	Status string `json:"status"`
}

type prepResponse struct {
	SessionAction string                          `json:"session_action"`
	ValidationRan bool                            `json:"validation_ran"`
	Session       prepSession                     `json:"session"`
	Validation    *session.ValidationResult       `json:"validation,omitempty"`
	Packet        *projectcontext.CompileResponse `json:"packet"`
	NextSteps     []string                        `json:"next_steps,omitempty"`
}

func addPrepCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var task string
	var budget string
	var fresh bool

	prepCmd := &cobra.Command{
		Use:   "prep",
		Short: "Start or reuse a session and compile the first task packet",
		Long: strings.TrimSpace(`
Prepare Brain-managed task context in one step.

brain prep starts a new session when needed, reuses and validates the active session when one already exists,
and then compiles the smallest justified startup packet for the task.
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot := flags.projectPath
			appCtx, err := loadApp(projectRoot)
			if err != nil {
				return err
			}
			defer appCtx.Close()

			active, err := appCtx.Session.Active(projectRoot)
			if err != nil {
				return err
			}

			resolvedTask := strings.TrimSpace(task)
			taskSource := "flag"
			sessionAction := "started"
			var validation *session.ValidationResult

			if active != nil {
				validation, err = appCtx.Session.Validate(cmd.Context(), session.ValidateRequest{
					ProjectDir: projectRoot,
					Stage:      "active",
				})
				if err != nil {
					return err
				}
				if !validation.OK {
					return prepValidationError(validation)
				}
				sessionAction = "reused"
				if resolvedTask == "" {
					resolvedTask = strings.TrimSpace(active.Task)
					taskSource = "session"
				} else if resolvedTask != strings.TrimSpace(active.Task) {
					return fmt.Errorf("prep task %q does not match active session task %q", resolvedTask, active.Task)
				}
			} else {
				if resolvedTask == "" {
					return errors.New("prep requires --task when no active session exists")
				}
				started, err := appCtx.Session.Start(cmd.Context(), session.StartRequest{
					ProjectDir: projectRoot,
					Task:       resolvedTask,
					ConfigPath: flags.configPath,
				})
				if err != nil {
					return err
				}
				active = &started.Session
			}

			response, err := runCompilePacketFlow(cmd.Context(), appCtx, compilePacketRequest{
				ProjectRoot:   projectRoot,
				Task:          resolvedTask,
				TaskSource:    taskSource,
				Budget:        budget,
				Fresh:         fresh,
				ActiveSession: active,
			})
			if err != nil {
				return err
			}

			payload := prepResponse{
				SessionAction: sessionAction,
				ValidationRan: validation != nil,
				Session: prepSession{
					ID:     active.ID,
					Task:   active.Task,
					Status: active.Status,
				},
				Validation: validation,
				Packet:     response,
				NextSteps: []string{
					"Use this packet first before manually assembling repo context.",
					fmt.Sprintf("If it is not enough, run `brain search %q` or `brain find <keyword>`.", resolvedTask),
					"Run required verification through `brain session run -- <command>`.",
				},
			}

			return appCtx.Output.Print(payload, func(w io.Writer) error {
				return renderPrepHuman(w, payload)
			})
		},
	}

	prepCmd.Flags().StringVar(&task, "task", "", "task summary; required when no active session exists")
	prepCmd.Flags().StringVar(&budget, "budget", "default", "compile budget preset or explicit token target")
	prepCmd.Flags().BoolVar(&fresh, "fresh", false, "bypass session-local packet reuse and force a full packet")
	root.AddCommand(prepCmd)
}

func renderPrepHuman(w io.Writer, response prepResponse) error {
	if _, err := io.WriteString(w, "## Brain Prep\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Session: `%s`\n", response.SessionAction); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Session ID: `%s`\n", response.Session.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Task: `%s`\n", response.Session.Task); err != nil {
		return err
	}
	if response.ValidationRan {
		if _, err := io.WriteString(w, "- Validation: active session ok\n"); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	if err := taskcontext.RenderCompileResponseHuman(w, response.Packet); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n## Next Steps\n\n"); err != nil {
		return err
	}
	for _, step := range response.NextSteps {
		if _, err := fmt.Fprintf(w, "- %s\n", step); err != nil {
			return err
		}
	}
	return nil
}

func prepValidationError(result *session.ValidationResult) error {
	if result == nil {
		return errors.New("session validation failed")
	}
	if len(result.Obligations) != 0 {
		return fmt.Errorf("session validation failed: %s", strings.Join(result.Obligations, "; "))
	}
	return errors.New("session validation failed")
}
