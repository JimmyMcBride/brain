package cmd

import (
	"fmt"
	"io"

	"brain/internal/search"

	"github.com/spf13/cobra"
)

func addSearchCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var limit int
	var explain bool
	var inject bool

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Run hybrid search against indexed markdown chunks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			if _, err := appCtx.EnsureFreshIndex(cmd.Context()); err != nil {
				return err
			}
			searchOpts := search.Options{}
			active, err := appCtx.Session.Active(flags.projectPath)
			if err != nil {
				return err
			}
			if active != nil {
				searchOpts.ActiveTask = active.Task
			}

			if explain {
				results, err := appCtx.Search.SearchWithExplainOptions(cmd.Context(), args[0], limit, searchOpts)
				if err != nil {
					return err
				}
				if inject {
					payload := map[string]any{
						"results":       results,
						"context_block": search.BuildContextBlock(results),
					}
					return appCtx.Output.Print(payload, func(w io.Writer) error {
						if len(results) == 0 {
							_, err := io.WriteString(w, "No results.\n")
							return err
						}
						for _, result := range results {
							if _, err := fmt.Fprintf(w, "%.3f  [%s lex=%.3f sem=%.3f rec=%.3f type=%.3f ctx=%.3f] %s", result.Score, result.Source, result.LexicalScore, result.SemanticScore, result.RecencyBoost, result.TypeBoost, result.ContextBoost, result.NotePath); err != nil {
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
						_, err := fmt.Fprintf(w, "\n%s", search.BuildContextBlock(results))
						return err
					})
				}
				return appCtx.Output.Print(results, func(w io.Writer) error {
					if len(results) == 0 {
						_, err := io.WriteString(w, "No results.\n")
						return err
					}
					for _, result := range results {
						if _, err := fmt.Fprintf(w, "%.3f  [%s lex=%.3f sem=%.3f rec=%.3f type=%.3f ctx=%.3f] %s", result.Score, result.Source, result.LexicalScore, result.SemanticScore, result.RecencyBoost, result.TypeBoost, result.ContextBoost, result.NotePath); err != nil {
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
			}

			results, err := appCtx.Search.SearchWithOptions(cmd.Context(), args[0], limit, searchOpts)
			if err != nil {
				return err
			}
			if inject {
				payload := map[string]any{
					"results":       results,
					"context_block": search.BuildContextBlock(results),
				}
				return appCtx.Output.Print(payload, func(w io.Writer) error {
					if len(results) == 0 {
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
					_, err := fmt.Fprintf(w, "\n%s", search.BuildContextBlock(results))
					return err
				})
			}
			return appCtx.Output.Print(results, func(w io.Writer) error {
				if len(results) == 0 {
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
	searchCmd.Flags().IntVarP(&limit, "limit", "n", 10, "maximum results")
	searchCmd.Flags().BoolVar(&explain, "explain", false, "show lexical and semantic ranking contributions")
	searchCmd.Flags().BoolVar(&inject, "inject", false, "include an agent-ready relevant-context block")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show search index freshness and metadata without reindexing",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()

			status, err := appCtx.IndexStatus(cmd.Context())
			if err != nil {
				return err
			}
			if status == nil {
				return appCtx.Output.Print(map[string]any{}, func(w io.Writer) error {
					_, err := io.WriteString(w, "Index status unavailable.\n")
					return err
				})
			}

			payload := map[string]any{
				"state":              status.State,
				"reason":             status.Reason,
				"indexed_at":         status.IndexedAt,
				"current_file_count": status.CurrentFileCount,
				"indexed_file_count": status.IndexedFileCount,
				"notes":              status.Notes,
				"chunks":             status.Chunks,
				"embeddings":         status.Embeddings,
				"embedding_provider": status.EmbeddingProvider,
				"embedding_model":    status.EmbeddingModel,
				"db_path":            appCtx.Paths.DBFile,
			}
			return appCtx.Output.Print(payload, func(w io.Writer) error {
				if _, err := fmt.Fprintf(w, "state: %s (%s)\n", status.State, status.Reason); err != nil {
					return err
				}
				if status.IndexedAt != "" {
					if _, err := fmt.Fprintf(w, "indexed_at: %s\n", status.IndexedAt); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintf(w, "files: %d current, %d indexed\n", status.CurrentFileCount, status.IndexedFileCount); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "notes: %d\nchunks: %d\nembeddings: %d\n", status.Notes, status.Chunks, status.Embeddings); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "embedding: %s/%s\n", status.EmbeddingProvider, status.EmbeddingModel); err != nil {
					return err
				}
				_, err = fmt.Fprintf(w, "db: %s\n", appCtx.Paths.DBFile)
				return err
			})
		},
	}

	searchCmd.AddCommand(statusCmd)
	root.AddCommand(searchCmd)
}
