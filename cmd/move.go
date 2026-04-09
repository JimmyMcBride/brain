package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func init() {
	var title string

	cmd := &cobra.Command{
		Use:   "move <path> <destination>",
		Short: "Move a note to a new location in the vault",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			if err := appCtx.EnsureVault(); err != nil {
				return err
			}
			oldPath, newPath, err := appCtx.Notes.Move(args[0], args[1])
			if err != nil {
				return err
			}
			if title != "" {
				_, newPath, err = appCtx.Notes.Rename(newPath, title)
				if err != nil {
					return err
				}
			}
			payload := map[string]string{"from": oldPath, "to": newPath}
			return appCtx.Output.Print(payload, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "%s -> %s\n", oldPath, newPath)
				return err
			})
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "rename the note after moving it")
	rootCmd.AddCommand(cmd)
}
