package cmd

import (
	"fmt"
	"io"

	"brain/internal/notes"

	"github.com/spf13/cobra"
)

func init() {
	var title string
	var bodyFlag string
	var fromStdin bool
	var meta []string
	var editor string

	cmd := &cobra.Command{
		Use:   "edit <path>",
		Short: "Edit a note using flags, stdin, or your editor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			if err := appCtx.EnsureVault(); err != nil {
				return err
			}
			path := args[0]

			if title == "" && bodyFlag == "" && !fromStdin && len(meta) == 0 {
				note, err := appCtx.Notes.EditInEditor(path, editor)
				if err != nil {
					return err
				}
				return appCtx.Output.Print(note, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "Edited %s\n", note.Path)
					return err
				})
			}

			metadata, err := parseMeta(meta)
			if err != nil {
				return err
			}
			body, err := readBody(bodyFlag, fromStdin)
			if err != nil {
				return err
			}
			update := notes.UpdateInput{
				Metadata: metadata,
				Summary:  "edited note via CLI",
			}
			if title != "" {
				update.Title = &title
			}
			if body != "" || fromStdin {
				update.Body = &body
			}
			note, err := appCtx.Notes.Update(path, update)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Updated %s\n", note.Path)
				return err
			})
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "update the note title")
	cmd.Flags().StringVarP(&bodyFlag, "body", "b", "", "replace body content")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read body from stdin")
	cmd.Flags().StringArrayVarP(&meta, "set", "m", nil, "metadata key=value")
	cmd.Flags().StringVar(&editor, "editor", "", "editor to launch when no direct edits are supplied")
	rootCmd.AddCommand(cmd)
}
