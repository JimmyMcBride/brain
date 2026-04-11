package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"brain/internal/config"
	"brain/internal/embeddings"
	"brain/internal/index"
	"brain/internal/output"
	"brain/internal/projectcontext"
	"brain/internal/workspace"

	"github.com/spf13/cobra"
)

func addInitCommand(root *cobra.Command, flags *rootFlagsState) {
	var provider string
	var model string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the current project with a local Brain workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, globalPaths, err := config.LoadOrCreate(flags.configPath)
			if err != nil {
				return err
			}
			if provider != "" {
				cfg.EmbeddingProvider = provider
			}
			if model != "" {
				cfg.EmbeddingModel = model
			}
			if err := config.Save(cfg, globalPaths.ConfigFile); err != nil {
				return err
			}

			projectDir, err := filepath.Abs(flags.projectPath)
			if err != nil {
				return err
			}
			projectPaths := config.ProjectPaths(globalPaths, projectDir)
			if err := config.EnsureProjectPaths(projectPaths); err != nil {
				return err
			}

			projectWorkspace := workspace.New(projectDir)
			if err := projectWorkspace.Initialize(); err != nil {
				return err
			}
			if _, err := embeddings.New(cfg); err != nil {
				return err
			}
			store, err := index.New(projectPaths.DBFile)
			if err != nil {
				return err
			}
			_ = store.Close()

			ctxManager := projectcontext.New(userHomeDir())
			if _, err := ctxManager.Install(cmd.Context(), projectcontext.Request{ProjectDir: projectDir}); err != nil {
				return err
			}

			printer := output.New(modeFromFlag(flags, cfg.OutputMode), cmd.OutOrStdout())
			payload := map[string]any{
				"config_file": globalPaths.ConfigFile,
				"project_dir": projectDir,
				"brain_dir":   projectPaths.BrainDir,
				"db_file":     projectPaths.DBFile,
			}
			return printer.Print(payload, func(w io.Writer) error {
				if err := output.KeyValue(w, "Config", globalPaths.ConfigFile); err != nil {
					return err
				}
				if err := output.KeyValue(w, "Project", projectDir); err != nil {
					return err
				}
				if err := output.KeyValue(w, "Brain", projectPaths.BrainDir); err != nil {
					return err
				}
				if err := output.KeyValue(w, "SQLite", projectPaths.DBFile); err != nil {
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
