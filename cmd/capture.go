package cmd

import (
	"fmt"
	"io"
	"time"

	"brain/internal/notes"

	"github.com/spf13/cobra"
)

func addCaptureCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var meta []string
	var bodyFlag string
	var fromStdin bool

	cmd := &cobra.Command{
		Use:   "capture [title]",
		Short: "Quickly capture a note into Resources/Captures",
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
			title := "Capture " + time.Now().Format("2006-01-02 15:04")
			if len(args) == 1 {
				title = args[0]
			}
			metadata, err := parseMeta(meta)
			if err != nil {
				return err
			}
			body, err := readBody(cmd.InOrStdin(), bodyFlag, fromStdin)
			if err != nil {
				return err
			}
			note, err := appCtx.Notes.Create(notes.CreateInput{
				Title:    title,
				NoteType: "capture",
				Template: "capture.md",
				Section:  "Resources",
				Subdir:   time.Now().Format("Captures/2006/01"),
				Body:     body,
				Metadata: metadata,
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Captured %s\n", note.Path)
				return err
			})
		},
	}

	cmd.Flags().StringArrayVarP(&meta, "meta", "m", nil, "metadata key=value")
	cmd.Flags().StringVarP(&bodyFlag, "body", "b", "", "body content")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read body from stdin")
	root.AddCommand(cmd)
}
