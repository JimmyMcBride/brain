package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func addDistillCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var sessionScope bool
	var limit int

	distillCmd := &cobra.Command{
		Use:   "distill",
		Short: "Create a reviewed distillation proposal from the active session",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !sessionScope {
				return fmt.Errorf("distill currently supports only --session")
			}

			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()

			note, err := appCtx.Distill.FromSession(cmd.Context(), limit)
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
	distillCmd.Flags().IntVarP(&limit, "limit", "n", 6, "maximum related notes or recent history entries")

	root.AddCommand(distillCmd)
}
