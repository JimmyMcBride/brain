package cmd

import (
	"fmt"
	"io"

	"brain/internal/history"

	"github.com/spf13/cobra"
)

func addReindexCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	cmd := &cobra.Command{
		Use:   "reindex",
		Short: "Rebuild the SQLite FTS and embedding index from the vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			if err := appCtx.EnsureVault(); err != nil {
				return err
			}
			stats, err := appCtx.Index.Reindex(cmd.Context(), appCtx.Vault, appCtx.Embedder)
			if err != nil {
				return err
			}
			if err := appCtx.History.Append(history.Entry{
				Operation: "reindex",
				Summary:   "reindexed vault",
				Metadata: map[string]any{
					"notes":      stats.Notes,
					"chunks":     stats.Chunks,
					"embeddings": stats.Embeddings,
				},
			}); err != nil {
				return err
			}
			return appCtx.Output.Print(stats, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Indexed %d notes, %d chunks, %d embeddings\n", stats.Notes, stats.Chunks, stats.Embeddings)
				return err
			})
		},
	}
	root.AddCommand(cmd)
}
