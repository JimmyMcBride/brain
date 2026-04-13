package cmd

import (
	"errors"
	"fmt"
	"io"

	"brain/internal/notes"

	"github.com/spf13/cobra"
)

func addDistillCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var sessionScope bool
	var brainstormPath string
	var limit int

	distillCmd := &cobra.Command{
		Use:   "distill",
		Short: "Create a reviewed distillation proposal from a session or brainstorm",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionScope == (brainstormPath != "") {
				return errors.New("choose exactly one distill scope: --session or --brainstorm <path>")
			}

			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()

			var note *notes.Note
			if sessionScope {
				note, err = appCtx.Distill.FromSession(cmd.Context(), limit)
			} else {
				if err := appCtx.SyncIndex(cmd.Context()); err != nil {
					return err
				}
				note, err = appCtx.Distill.FromBrainstorm(cmd.Context(), brainstormPath, limit)
			}
			if err != nil {
				return err
			}

			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Created distill proposal %s\n", note.Path)
				return err
			})
		},
	}

	distillCmd.Flags().BoolVar(&sessionScope, "session", false, "distill from the active session")
	distillCmd.Flags().StringVar(&brainstormPath, "brainstorm", "", "distill from a brainstorm note path")
	distillCmd.Flags().IntVarP(&limit, "limit", "n", 6, "maximum related notes or recent history entries")

	root.AddCommand(distillCmd)
}
