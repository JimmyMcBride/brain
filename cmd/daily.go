package cmd

import (
	"fmt"
	"io"
	"time"

	"brain/internal/notes"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "daily [yyyy-mm-dd]",
		Short: "Create or open a daily note",
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
			day := time.Now()
			if len(args) == 1 {
				day, err = time.Parse("2006-01-02", args[0])
				if err != nil {
					return err
				}
			}
			relPath := day.Format("Areas/Daily/2006/2006-01-02.md")
			if note, err := appCtx.Notes.Read(relPath); err == nil {
				return appCtx.Output.Print(note, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "%s\n", note.Path)
					return err
				})
			}
			body, err := appCtx.Templates.Render("daily.md", map[string]any{
				"Title": "Daily " + day.Format("2006-01-02"),
				"Date":  day.Format("2006-01-02"),
				"Now":   day.Format(time.RFC3339),
			})
			if err != nil {
				return err
			}
			note, err := appCtx.Notes.Create(notes.CreateInput{
				Title:    "Daily " + day.Format("2006-01-02"),
				Filename: day.Format("2006-01-02"),
				NoteType: "daily",
				Template: "daily.md",
				Section:  "Areas",
				Subdir:   day.Format("Daily/2006"),
				Body:     body,
				Metadata: map[string]any{"date": day.Format("2006-01-02")},
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "%s\n", note.Path)
				return err
			})
		},
	}
	rootCmd.AddCommand(cmd)
}
