package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strconv"

	"brain/internal/config"
	"brain/internal/embeddings"
	"brain/internal/index"
	"brain/internal/output"
	"brain/internal/workspace"

	"github.com/spf13/cobra"
)

func addDoctorCommand(root *cobra.Command, flags *rootFlagsState) {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Validate project-local Brain setup and embedding configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, globalPaths, err := config.LoadOrCreate(flags.configPath)
			if err != nil {
				return err
			}
			projectDir, err := filepath.Abs(flags.projectPath)
			if err != nil {
				return err
			}
			paths := config.ProjectPaths(globalPaths, projectDir)
			checks := []map[string]any{
				check("config", exists(globalPaths.ConfigFile)),
				check("project", exists(projectDir)),
				check("brain", exists(paths.BrainDir)),
				check("sqlite", exists(paths.DBFile)),
			}
			if _, err := embeddings.New(cfg); err != nil {
				checks = append(checks, map[string]any{"name": "embedding", "ok": false, "details": err.Error()})
			} else {
				checks = append(checks, map[string]any{"name": "embedding", "ok": true, "details": cfg.EmbeddingProvider + "/" + cfg.EmbeddingModel})
			}
			if err := workspace.New(projectDir).Validate(); err != nil {
				checks = append(checks, map[string]any{"name": "workspace", "ok": false, "details": err.Error()})
			} else {
				checks = append(checks, map[string]any{"name": "workspace", "ok": true, "details": "project-local workspace present"})
			}
			if exists(paths.DBFile) {
				store, err := index.New(paths.DBFile)
				if err == nil {
					defer store.Close()
					stats, err := store.Stats(cmd.Context())
					if err == nil {
						checks = append(checks,
							map[string]any{"name": "notes", "ok": true, "details": strconv.Itoa(stats.Notes)},
							map[string]any{"name": "chunks", "ok": true, "details": strconv.Itoa(stats.Chunks)},
							map[string]any{"name": "embeddings", "ok": true, "details": strconv.Itoa(stats.Embeddings)},
						)
					}
				}
			}

			printer := output.New(modeFromFlag(flags, cfg.OutputMode), cmd.OutOrStdout())
			return printer.Print(checks, func(w io.Writer) error {
				for _, item := range checks {
					status := "ok"
					if item["ok"] == false {
						status = "fail"
					}
					if _, err := io.WriteString(w, item["name"].(string)+": "+status+" ("+item["details"].(string)+")\n"); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
	root.AddCommand(cmd)
}

func check(name string, ok bool) map[string]any {
	details := "present"
	if !ok {
		details = "missing"
	}
	return map[string]any{"name": name, "ok": ok, "details": details}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
