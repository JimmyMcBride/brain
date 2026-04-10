package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func addSearchCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Run hybrid search against indexed markdown chunks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			results, err := appCtx.Search.Search(cmd.Context(), args[0], limit)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(results, func(w io.Writer) error {
				if len(results) == 0 {
					stats, statErr := appCtx.Index.Stats(cmd.Context())
					if statErr == nil && stats.Chunks == 0 {
						_, err := io.WriteString(w, "No indexed content. Run `brain reindex` first.\n")
						return err
					}
					_, err := io.WriteString(w, "No results.\n")
					return err
				}
				for _, result := range results {
					if _, err := fmt.Fprintf(w, "%.3f  %s", result.Score, result.NotePath); err != nil {
						return err
					}
					if result.Heading != "" {
						if _, err := fmt.Fprintf(w, " -> %s", result.Heading); err != nil {
							return err
						}
					}
					if _, err := fmt.Fprintf(w, "\n  %s\n", result.Snippet); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "maximum results")
	root.AddCommand(cmd)
}
