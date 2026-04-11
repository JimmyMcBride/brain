package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"brain/internal/session"

	"github.com/spf13/cobra"
)

func addSessionCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var project string

	sessionCmd := &cobra.Command{
		Use:   "session",
		Short: "Manage enforced project work sessions",
	}

	var startTask string
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a validated project session",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			result, err := appCtx.Session.Start(cmd.Context(), session.StartRequest{
				ProjectDir: project,
				Task:       startTask,
				ConfigPath: flags.configPath,
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(result, func(w io.Writer) error {
				if _, err := fmt.Fprintf(w, "Started session %s for task: %s\n", result.Session.ID, result.Session.Task); err != nil {
					return err
				}
				if _, err := io.WriteString(w, "Required docs:\n"); err != nil {
					return err
				}
				for _, doc := range result.RequiredDocs {
					if _, err := fmt.Fprintf(w, "- %s\n", doc); err != nil {
						return err
					}
				}
				if _, err := io.WriteString(w, "Suggested commands:\n"); err != nil {
					return err
				}
				for _, command := range result.SuggestedCommands {
					if _, err := fmt.Fprintf(w, "- %s\n", command); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
	startCmd.Flags().StringVar(&startTask, "task", "", "required task summary")

	var validateStage string
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the active session or finish-stage obligations",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			result, err := appCtx.Session.Validate(cmd.Context(), session.ValidateRequest{
				ProjectDir: project,
				Stage:      validateStage,
			})
			if err != nil {
				return err
			}
			printErr := appCtx.Output.Print(result, func(w io.Writer) error {
				status := "ok"
				if !result.OK {
					status = "fail"
				}
				if _, err := fmt.Fprintf(w, "Session validation: %s (%s)\n", status, result.Stage); err != nil {
					return err
				}
				if result.SessionID != "" {
					if _, err := fmt.Fprintf(w, "Session: %s\nTask: %s\n", result.SessionID, result.Task); err != nil {
						return err
					}
				}
				for _, obligation := range result.Obligations {
					if _, err := fmt.Fprintf(w, "Need: %s\n", obligation); err != nil {
						return err
					}
				}
				for _, remediation := range result.Remediation {
					if _, err := fmt.Fprintf(w, "Fix: %s\n", remediation); err != nil {
						return err
					}
				}
				return nil
			})
			if printErr != nil {
				return printErr
			}
			if !result.OK {
				return errors.New("session validation failed")
			}
			return nil
		},
	}
	validateCmd.Flags().StringVar(&validateStage, "stage", "active", "validation stage: active or finish")

	var finishSummary string
	var finishForce bool
	var finishReason string
	finishCmd := &cobra.Command{
		Use:   "finish",
		Short: "Finish the active session after closeout validation",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			result, err := appCtx.Session.Finish(cmd.Context(), session.FinishRequest{
				ProjectDir: project,
				Summary:    finishSummary,
				Force:      finishForce,
				Reason:     finishReason,
			})
			if err != nil {
				return err
			}
			printErr := appCtx.Output.Print(result, func(w io.Writer) error {
				if _, err := fmt.Fprintf(w, "Session %s: %s\n", result.SessionID, result.Status); err != nil {
					return err
				}
				for _, obligation := range result.Validation.Obligations {
					if _, err := fmt.Fprintf(w, "Need: %s\n", obligation); err != nil {
						return err
					}
				}
				for _, remediation := range result.Validation.Remediation {
					if _, err := fmt.Fprintf(w, "Fix: %s\n", remediation); err != nil {
						return err
					}
				}
				if result.LedgerPath != "" {
					if _, err := fmt.Fprintf(w, "Ledger: %s\n", result.LedgerPath); err != nil {
						return err
					}
				}
				return nil
			})
			if printErr != nil {
				return printErr
			}
			if result.Status == "blocked" {
				return errors.New("session finish blocked")
			}
			return nil
		},
	}
	finishCmd.Flags().StringVar(&finishSummary, "summary", "", "optional session summary")
	finishCmd.Flags().BoolVar(&finishForce, "force", false, "force finish and record an override")
	finishCmd.Flags().StringVar(&finishReason, "reason", "", "required reason when forcing finish")

	var abortReason string
	abortCmd := &cobra.Command{
		Use:   "abort",
		Short: "Abort the active session",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			result, err := appCtx.Session.Abort(cmd.Context(), session.AbortRequest{
				ProjectDir: project,
				Reason:     abortReason,
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(result, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Session %s: aborted\n", result.SessionID)
				return err
			})
		},
	}
	abortCmd.Flags().StringVar(&abortReason, "reason", "", "optional abort reason")

	runCmd := &cobra.Command{
		Use:   "run -- <command> [args...]",
		Short: "Run and record an external command for the active session",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("session run requires a command after --")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			capture := appCtx.Output.JSONEnabled()
			result, runErr := appCtx.Session.RunCommand(cmd.Context(), session.RunRequest{
				ProjectDir:    project,
				Argv:          args,
				CaptureOutput: capture,
			}, cmd.OutOrStdout(), cmd.ErrOrStderr())
			if result != nil {
				if err := appCtx.Output.Print(result, func(w io.Writer) error {
					if _, err := fmt.Fprintf(w, "Recorded command: %s (exit %d)\n", result.Command, result.ExitCode); err != nil {
						return err
					}
					return nil
				}); err != nil {
					return err
				}
			}
			if runErr != nil {
				return fmt.Errorf("session run failed: %w", runErr)
			}
			return nil
		},
	}
	runCmd.DisableFlagParsing = false
	runCmd.SetHelpFunc(func(c *cobra.Command, s []string) {
		_, _ = io.WriteString(c.OutOrStdout(), strings.TrimSpace(`
Run and record an external command for the active session.

Usage:
  brain session run -- <command> [args...]
`)+"\n")
	})

	for _, sub := range []*cobra.Command{startCmd, validateCmd, finishCmd, abortCmd, runCmd} {
		sub.Flags().StringVar(&project, "project", ".", "project root for the session")
	}

	sessionCmd.AddCommand(startCmd, validateCmd, runCmd, finishCmd, abortCmd)
	root.AddCommand(sessionCmd)
}
