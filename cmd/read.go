package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func addReadCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	cmd := &cobra.Command{
		Use:   "read <path>",
		Short: "Read a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			if err := appCtx.EnsureWorkspace(); err != nil {
				return err
			}
			note, err := appCtx.Notes.Read(args[0])
			if err != nil {
				return err
			}
			if err := appCtx.Session.RecordPacketExpansion(appCtx.Paths.ProjectDir, note.Path); err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				if _, err := fmt.Fprintf(w, "Path: %s\nType: %s\n\n", note.Path, note.Type); err != nil {
					return err
				}
				_, err := io.WriteString(w, note.Content)
				return err
			})
		},
	}
	root.AddCommand(cmd)
}
