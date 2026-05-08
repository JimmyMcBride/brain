package cmd

import (
	"fmt"
	"io"

	"brain/internal/output"
	"brain/internal/projectcontext"

	"github.com/spf13/cobra"
)

func addAdoptCommand(root *cobra.Command, flags *rootFlagsState) {
	var provider string
	var model string
	var agents []string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "adopt",
		Short: "Adopt an existing repo into the Brain managed context model",
		RunE: func(cmd *cobra.Command, args []string) error {
			boot, err := bootstrapProject(flags, provider, model)
			if err != nil {
				return err
			}

			results, err := contextManager().Adopt(cmd.Context(), projectcontext.Request{
				ProjectDir: boot.ProjectDir,
				Agents:     agents,
				DryRun:     dryRun,
			})
			if err != nil {
				return err
			}

			payload := map[string]any{
				"config_file": boot.Global.ConfigFile,
				"project_dir": boot.ProjectDir,
				"brain_dir":   boot.Project.BrainDir,
				"db_file":     boot.Project.DBFile,
				"results":     results,
				"next_steps":  adoptNextSteps(),
			}
			printer := output.New(modeFromFlag(flags, boot.Config.OutputMode), cmd.OutOrStdout())
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
				if dryRun {
					if _, err := fmt.Fprintln(w, "Adoption plan:"); err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintln(w, "Adopted project-local Brain context:"); err != nil {
						return err
					}
				}
				for _, result := range results {
					preserve := ""
					if result.PreservedUserContent {
						preserve = " preserve-user"
					}
					if _, err := fmt.Fprintf(w, "%-9s %-8s %s%s\n", result.Action, result.Kind, result.Path, preserve); err != nil {
						return err
					}
				}
				if !dryRun {
					if _, err := fmt.Fprintln(w, "\nNext for AI agent:"); err != nil {
						return err
					}
					for _, step := range adoptNextSteps() {
						if _, err := fmt.Fprintf(w, "- %s\n", step); err != nil {
							return err
						}
					}
				}
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&provider, "embedding-provider", "", "embedding provider (localhash, openai, none)")
	cmd.Flags().StringVar(&model, "embedding-model", "", "embedding model name")
	cmd.Flags().StringArrayVarP(&agents, "agent", "a", nil, "agent instruction files to integrate or create; repeatable")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the adoption plan without writing files")
	root.AddCommand(cmd)
}

func adoptNextSteps() []string {
	return []string{
		"treat generated context as starter context, not complete repo memory",
		"scan repo structure, docs, entrypoints, tests, CI, config, and deployment surfaces",
		"update AGENTS.md, docs, or .brain notes with durable project-specific findings",
		"add focused .brain/resources notes when details are too specific for the main templates",
	}
}
