package cmd

import (
	"fmt"
	"io"

	"brain/internal/search"

	"github.com/spf13/cobra"
)

func addContentCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	contentCmd := &cobra.Command{
		Use:   "content",
		Short: "Content-seed and outline workflow",
	}

	var gatherLimit int
	seedCmd := &cobra.Command{
		Use:   "seed <path>",
		Short: "Promote a note into a content seed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Content.Seed(args[0])
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Seeded %s\n", note.Path)
				return err
			})
		},
	}

	gatherCmd := &cobra.Command{
		Use:   "gather <path>",
		Short: "Gather related notes around a content seed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			payload, err := appCtx.Content.Gather(cmd.Context(), args[0], gatherLimit)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(payload, func(w io.Writer) error {
				related := payload["related"].([]search.Result)
				if len(related) == 0 {
					_, err := io.WriteString(w, "No related notes found.\n")
					return err
				}
				for _, item := range related {
					if _, err := fmt.Fprintf(w, "%s", item.NotePath); err != nil {
						return err
					}
					if item.Heading != "" {
						if _, err := fmt.Fprintf(w, " -> %s", item.Heading); err != nil {
							return err
						}
					}
					if _, err := fmt.Fprintf(w, "\n  %s\n", item.Snippet); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	outlineCmd := &cobra.Command{
		Use:   "outline <path>",
		Short: "Generate an outline package note from a seed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Content.Outline(cmd.Context(), args[0], gatherLimit)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Created outline %s\n", note.Path)
				return err
			})
		},
	}

	var channel string
	var repurpose string
	publishCmd := &cobra.Command{
		Use:   "publish <path>",
		Short: "Mark a content note as published",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Content.Publish(args[0], channel, repurpose)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Published %s\n", note.Path)
				return err
			})
		},
	}

	gatherCmd.Flags().IntVarP(&gatherLimit, "limit", "n", 6, "maximum related notes")
	outlineCmd.Flags().IntVarP(&gatherLimit, "limit", "n", 6, "maximum related notes")
	publishCmd.Flags().StringVar(&channel, "channel", "", "publication channel")
	publishCmd.Flags().StringVar(&repurpose, "repurpose", "", "repurpose target")

	contentCmd.AddCommand(seedCmd, gatherCmd, outlineCmd, publishCmd)
	root.AddCommand(contentCmd)
}
