package cmd

import (
	"fmt"
	"io"

	"brain/internal/notes"

	"github.com/spf13/cobra"
)

func addAddCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var section string
	var templateName string
	var noteType string
	var subdir string
	var meta []string
	var bodyFlag string
	var fromStdin bool
	var overwrite bool

	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Create a note from a template",
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
			metadata, err := parseMeta(meta)
			if err != nil {
				return err
			}
			body, err := readBody(cmd.InOrStdin(), bodyFlag, fromStdin)
			if err != nil {
				return err
			}
			title := args[0]
			resolvedType := chooseType(section, noteType)
			note, err := appCtx.Notes.Create(notes.CreateInput{
				Title:     title,
				NoteType:  resolvedType,
				Template:  chooseTemplate(resolvedType, templateName),
				Section:   chooseSection(section),
				Subdir:    subdir,
				Body:      body,
				Metadata:  metadata,
				Overwrite: overwrite,
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Created %s\n", note.Path)
				return err
			})
		},
	}

	cmd.Flags().StringVarP(&section, "section", "s", "Resources", "PARA section")
	cmd.Flags().StringVarP(&templateName, "template", "T", "", "template file")
	cmd.Flags().StringVarP(&noteType, "type", "t", "", "note type")
	cmd.Flags().StringVar(&subdir, "subdir", "", "subdirectory within the section")
	cmd.Flags().StringArrayVarP(&meta, "meta", "m", nil, "metadata key=value")
	cmd.Flags().StringVarP(&bodyFlag, "body", "b", "", "body content")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read body from stdin")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing note if present")
	root.AddCommand(cmd)
}
