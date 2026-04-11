package cmd

import (
	"brain/internal/output"
	"brain/internal/projectcontext"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func addInitCommand(root *cobra.Command, flags *rootFlagsState) {
	var provider string
	var model string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the current project with a local Brain workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			boot, err := bootstrapProject(flags, provider, model)
			if err != nil {
				return err
			}

			if _, err := contextManager().Install(cmd.Context(), projectcontext.Request{ProjectDir: boot.ProjectDir}); err != nil {
				return err
			}

			printer := output.New(modeFromFlag(flags, boot.Config.OutputMode), cmd.OutOrStdout())
			payload := map[string]any{
				"config_file": boot.Global.ConfigFile,
				"project_dir": boot.ProjectDir,
				"brain_dir":   boot.Project.BrainDir,
				"db_file":     boot.Project.DBFile,
			}
			return printer.Print(payload, func(w io.Writer) error {
				if err := output.KeyValue(w, "Config", boot.Global.ConfigFile); err != nil {
					return err
				}
				if err := output.KeyValue(w, "Project", boot.ProjectDir); err != nil {
					return err
				}
				if err := output.KeyValue(w, "Brain", boot.Project.BrainDir); err != nil {
					return err
				}
				if err := output.KeyValue(w, "SQLite", boot.Project.DBFile); err != nil {
					return err
				}
				_, err := fmt.Fprintf(w, "Initialized project-local Brain workspace.\n")
				return err
			})
		},
	}

	cmd.Flags().StringVar(&provider, "embedding-provider", "", "embedding provider (localhash, openai, none)")
	cmd.Flags().StringVar(&model, "embedding-model", "", "embedding model name")
	root.AddCommand(cmd)
}
