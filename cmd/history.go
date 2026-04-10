package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
)

func addHistoryCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var limit int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show recent operations and backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			entries, err := appCtx.History.List(limit)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(entries, func(w io.Writer) error {
				if len(entries) == 0 {
					_, err := io.WriteString(w, "No history entries.\n")
					return err
				}
				for _, entry := range entries {
					if _, err := fmt.Fprintf(w, "%s  %-8s %s", entry.Timestamp.Format(time.RFC3339), entry.Operation, entry.File); err != nil {
						return err
					}
					if entry.Target != "" {
						if _, err := fmt.Fprintf(w, " -> %s", entry.Target); err != nil {
							return err
						}
					}
					if _, err := fmt.Fprintf(w, "  %s\n", entry.Summary); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "maximum entries")
	root.AddCommand(cmd)
}
