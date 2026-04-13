package cmd

import (
	"fmt"
	"io"

	"brain/internal/search"

	"github.com/spf13/cobra"
)

func addBrainstormCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	brainstormCmd := &cobra.Command{
		Use:   "brainstorm",
		Short: "Project-scoped brainstorm recording and ideation",
	}

	startCmd := &cobra.Command{
		Use:   "start <topic>",
		Short: "Start a new brainstorm session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Brainstorm.Start(args[0])
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Started brainstorm %s\n", note.Path)
				return err
			})
		},
	}

	var ideaBody string
	var ideaStdin bool
	ideaCmd := &cobra.Command{
		Use:   "idea <path>",
		Short: "Add an idea to a brainstorm session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readBody(cmd.InOrStdin(), ideaBody, ideaStdin)
			if err != nil {
				return err
			}
			if body == "" {
				return fmt.Errorf("idea body is required (use --body or --stdin)")
			}
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Brainstorm.Idea(args[0], body)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Added idea to %s\n", note.Path)
				return err
			})
		},
	}
	ideaCmd.Flags().StringVarP(&ideaBody, "body", "b", "", "idea text")
	ideaCmd.Flags().BoolVar(&ideaStdin, "stdin", false, "read idea from stdin")

	var gatherLimit int
	gatherCmd := &cobra.Command{
		Use:   "gather <path>",
		Short: "Gather related notes for a brainstorm",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			if err := appCtx.SyncIndex(cmd.Context()); err != nil {
				return err
			}
			payload, err := appCtx.Brainstorm.Gather(cmd.Context(), args[0], gatherLimit)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(payload, func(w io.Writer) error {
				related := payload["related"].([]search.Result)
				if len(related) == 0 {
					_, err := io.WriteString(w, "No related notes found.\n")
					return err
				}
				for _, item := range related {
					if _, err := fmt.Fprintf(w, "%s", item.NotePath); err != nil {
						return err
					}
					if item.Heading != "" {
						if _, err := fmt.Fprintf(w, " -> %s", item.Heading); err != nil {
							return err
						}
					}
					if _, err := fmt.Fprintf(w, "\n  %s\n", item.Snippet); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
	gatherCmd.Flags().IntVarP(&gatherLimit, "limit", "n", 6, "maximum related notes")

	var distillLimit int
	distillCmd := &cobra.Command{
		Use:   "distill <path>",
		Short: "Create a distillation proposal from a brainstorm",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			if err := appCtx.SyncIndex(cmd.Context()); err != nil {
				return err
			}
			note, err := appCtx.Distill.FromBrainstorm(cmd.Context(), args[0], distillLimit)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Created distill proposal %s\n", note.Path)
				return err
			})
		},
	}
	distillCmd.Flags().IntVarP(&distillLimit, "limit", "n", 6, "maximum related notes")

	brainstormCmd.AddCommand(startCmd, ideaCmd, gatherCmd, distillCmd)
	root.AddCommand(brainstormCmd)
}
