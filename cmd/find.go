package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func addFindCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var noteType string
	var pathFilter string
	var limit int

	cmd := &cobra.Command{
		Use:   "find [query]",
		Short: "Find notes by title, path, metadata type, or content",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			if err := appCtx.EnsureVault(); err != nil {
				return err
			}
			query := ""
			if len(args) == 1 {
				query = args[0]
			}
			results, err := appCtx.Notes.Find(query, noteType, pathFilter, limit)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(results, func(w io.Writer) error {
				if len(results) == 0 {
					_, err := io.WriteString(w, "No results.\n")
					return err
				}
				for _, result := range results {
					if _, err := fmt.Fprintf(w, "%s [%s] %s\n", result["path"], result["type"], result["title"]); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	cmd.Flags().StringVarP(&noteType, "type", "t", "", "filter by note type")
	cmd.Flags().StringVarP(&pathFilter, "path", "p", "", "filter by path fragment")
	cmd.Flags().IntVarP(&limit, "limit", "n", 25, "maximum results")
	root.AddCommand(cmd)
}
