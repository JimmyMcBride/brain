package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"brain/internal/organize"

	"github.com/spf13/cobra"
)

func init() {
	var apply bool
	var allowArchive bool

	cmd := &cobra.Command{
		Use:   "organize",
		Short: "Dry-run simple PARA organizing suggestions",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			if err := appCtx.EnsureVault(); err != nil {
				return err
			}
			files, err := appCtx.Vault.WalkMarkdownFiles()
			if err != nil {
				return err
			}
			type plan struct {
				From string `json:"from"`
				To   string `json:"to"`
				Diff string `json:"diff"`
			}
			var plans []plan
			for _, file := range files {
				rel, err := appCtx.Vault.Rel(file)
				if err != nil {
					return err
				}
				note, err := appCtx.Notes.Read(rel)
				if err != nil {
					return err
				}
				targetSection := recommendedSection(note.Type)
				if targetSection == "" {
					continue
				}
				if note.Metadata["status"] == "archived" && allowArchive {
					targetSection = "Archives"
				}
				if strings.HasPrefix(note.Path, targetSection+"/") {
					continue
				}
				target := targetSection + "/" + filepath.Base(note.Path)
				diff, err := organize.UnifiedDiff(note.Path, target, note.Path+"\n", target+"\n")
				if err != nil {
					return err
				}
				plans = append(plans, plan{From: note.Path, To: target, Diff: diff})
				if apply {
					if _, _, err := appCtx.Notes.Move(note.Path, target); err != nil {
						return err
					}
				}
			}
			return appCtx.Output.Print(plans, func(w io.Writer) error {
				for _, p := range plans {
					if _, err := fmt.Fprintf(w, "%s -> %s\n%s\n", p.From, p.To, p.Diff); err != nil {
						return err
					}
				}
				if len(plans) == 0 {
					_, err := io.WriteString(w, "No organize actions needed.\n")
					return err
				}
				return nil
			})
		},
	}

	cmd.Flags().BoolVar(&apply, "apply", false, "apply the suggested moves")
	cmd.Flags().BoolVar(&allowArchive, "archive", false, "allow moving archived notes into Archives")
	rootCmd.AddCommand(cmd)
}

func recommendedSection(noteType string) string {
	switch noteType {
	case "project":
		return "Projects"
	case "area", "daily":
		return "Areas"
	case "resource", "capture", "lesson", "content_seed", "content_outline":
		return "Resources"
	case "archive":
		return "Archives"
	default:
		return ""
	}
}
