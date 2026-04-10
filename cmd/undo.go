package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func addUndoCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	cmd := &cobra.Command{
		Use:   "undo",
		Short: "Revert the last tracked file operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			entry, err := appCtx.Undoer.Undo()
			if err != nil {
				return err
			}
			return appCtx.Output.Print(entry, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Reverted %s on %s\n", entry.Operation, entry.File)
				return err
			})
		},
	}
	root.AddCommand(cmd)
}
