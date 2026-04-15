package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"brain/internal/config"
	"brain/internal/embeddings"
	"brain/internal/index"
	"brain/internal/notes"
	"brain/internal/output"
	"brain/internal/projectcontext"
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
			var provider embeddings.Provider
			if provider, err = embeddings.New(cfg); err != nil {
				checks = append(checks, map[string]any{"name": "embedding", "ok": false, "details": err.Error()})
			} else {
				checks = append(checks, map[string]any{"name": "embedding", "ok": true, "details": cfg.EmbeddingProvider + "/" + cfg.EmbeddingModel})
			}
			workspaceSvc := workspace.New(projectDir)
			if err := workspaceSvc.Validate(); err != nil {
				checks = append(checks, map[string]any{"name": "workspace", "ok": false, "details": err.Error()})
			} else {
				checks = append(checks, map[string]any{"name": "workspace", "ok": true, "details": "project-local workspace present"})
			}
			checks = append(checks, projectMigrationDoctorCheck(projectDir))
			if files, err := notes.ValidateWorkspaceMarkdown(workspaceSvc); err != nil {
				checks = append(checks, map[string]any{"name": "note_integrity", "ok": false, "details": err.Error()})
			} else {
				checks = append(checks, map[string]any{"name": "note_integrity", "ok": true, "details": strconv.Itoa(files) + " files checked"})
			}
			if exists(paths.DBFile) {
				store, err := index.New(paths.DBFile)
				if err == nil {
					defer store.Close()
					if freshness, err := store.Freshness(cmd.Context(), workspaceSvc, provider); err == nil {
						ok := freshness.State == "fresh"
						details := freshness.State
						if freshness.Reason != "" {
							details += " (" + freshness.Reason + ")"
						}
						checks = append(checks, map[string]any{"name": "index_freshness", "ok": ok, "details": details})
					}
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

func projectMigrationDoctorCheck(projectDir string) map[string]any {
	plan, err := contextManager().PlanProjectMigrations(projectDir)
	if err != nil {
		return map[string]any{"name": "project_migrations", "ok": false, "details": err.Error()}
	}

	switch plan.Status {
	case "not_brain_project":
		return map[string]any{"name": "project_migrations", "ok": true, "details": "not a Brain project"}
	case "current":
		return map[string]any{"name": "project_migrations", "ok": true, "details": "current"}
	case "pending":
		return map[string]any{"name": "project_migrations", "ok": false, "details": "pending (" + projectMigrationPendingSummary(plan.PendingMigrations) + ")"}
	case "broken":
		details := "broken"
		if reason := strings.TrimSpace(plan.BrokenReason); reason != "" {
			details += " (" + reason
			if pending := projectMigrationPendingSummary(plan.PendingMigrations); pending != "" {
				details += "; pending: " + pending
			}
			details += ")"
			return map[string]any{"name": "project_migrations", "ok": false, "details": details}
		}
		if pending := projectMigrationPendingSummary(plan.PendingMigrations); pending != "" {
			details += " (pending: " + pending + ")"
		}
		return map[string]any{"name": "project_migrations", "ok": false, "details": details}
	default:
		return map[string]any{"name": "project_migrations", "ok": false, "details": fmt.Sprintf("unknown status %q", plan.Status)}
	}
}

func projectMigrationPendingSummary(migrations []projectcontext.ProjectMigrationDefinition) string {
	if len(migrations) == 0 {
		return ""
	}
	ids := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		if id := strings.TrimSpace(migration.ID); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return ""
	}
	return strings.Join(ids, ", ")
}
