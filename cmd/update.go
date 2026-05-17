package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"brain/internal/buildinfo"
	"brain/internal/config"
	"brain/internal/output"
	"brain/internal/projectcontext"
	"brain/internal/update"

	"github.com/spf13/cobra"
)

type updater interface {
	Update(context.Context, update.Request) (update.Result, error)
}

var newUpdater = func(cfg *config.Config, paths config.Paths) updater {
	return update.New(cfg, paths, update.Options{})
}

type updateCommandOutput struct {
	update.Result
	skillRefreshResult
	projectMigrationStatusResult
	KarpathyGuidance *projectcontext.GuidanceStatus `json:"karpathy_guidance,omitempty"`
}

func addUpdateCommand(root *cobra.Command, flags *rootFlagsState, _ appLoader) {
	var checkOnly bool
	var prerelease bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and install the latest brain release",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, paths, err := config.LoadOrCreate(flags.configPath)
			if err != nil {
				return err
			}

			manager := newUpdater(cfg, paths)
			projectRoot, err := filepath.Abs(flags.projectPath)
			if err != nil {
				return err
			}
			globalTargets, localTargets, err := inspectInstalledSkills(projectRoot)
			if err != nil {
				return err
			}
			result, err := manager.Update(cmd.Context(), update.Request{
				CheckOnly:         checkOnly,
				IncludePrerelease: prerelease,
			})
			if err != nil {
				return err
			}
			out := updateCommandOutput{Result: result}

			var refreshErr error
			var migrationErr error
			if !checkOnly && shouldRefreshSkills(result.Status) {
				refreshBinary := result.CurrentPath
				if result.InstalledPath != "" {
					refreshBinary = result.InstalledPath
				}
				if refreshBinary == "" {
					refreshBinary = buildinfo.Current().Path
				}
				out.skillRefreshResult, refreshErr = refreshInstalledSkills(refreshBinary, flags.configPath, projectRoot, globalTargets, localTargets)
				if refreshErr == nil {
					migrationResult, err := projectMigrationRunner(refreshBinary, flags.configPath, projectRoot)
					out.projectMigrationStatusResult = summarizeProjectMigrationResult(migrationResult)
					migrationErr = err
					if migrationErr == nil {
						guidanceStatus, err := contextManager().KarpathyGuidanceStatus(projectRoot)
						if err != nil {
							return err
						}
						if guidanceStatus.Recommendation != nil {
							out.KarpathyGuidance = guidanceStatus
						}
					}
				}
			}

			printer := output.New(modeFromFlag(flags, cfg.OutputMode), cmd.OutOrStdout())
			if err := printer.Print(out, func(w io.Writer) error {
				switch result.Status {
				case "up_to_date", "no_releases":
					if _, err := fmt.Fprintf(w, "update: %s\n", result.Message); err != nil {
						return err
					}
				case "update_available":
					if _, err := fmt.Fprintf(w, "update: %s\n", result.Message); err != nil {
						return err
					}
					if _, err := fmt.Fprintf(w, "release: %s\n", result.ReleaseURL); err != nil {
						return err
					}
				case "updated", "installed_to_fallback":
					if _, err := fmt.Fprintf(w, "update: %s\n", result.Message); err != nil {
						return err
					}
					if _, err := fmt.Fprintf(w, "installed: %s\n", result.InstalledPath); err != nil {
						return err
					}
					if _, err := fmt.Fprintf(w, "fallback:  %t\n", result.FallbackUsed); err != nil {
						return err
					}
					if result.ReleaseURL != "" {
						if _, err := fmt.Fprintf(w, "release:   %s\n", result.ReleaseURL); err != nil {
							return err
						}
					}
					if result.Message != "" && result.FallbackUsed && result.LookPathTarget != "" {
						if _, err := fmt.Fprintf(w, "lookup:    %s\n", result.LookPathTarget); err != nil {
							return err
						}
					}
				case "unsupported_platform":
					if _, err := fmt.Fprintf(w, "update: %s\n", result.Message); err != nil {
						return err
					}
				default:
					if _, err := fmt.Fprintf(w, "update: %s\n", result.Status); err != nil {
						return err
					}
					if result.Message != "" {
						if _, err := fmt.Fprintf(w, "details: %s\n", result.Message); err != nil {
							return err
						}
					}
				}
				if !checkOnly && shouldRefreshSkills(result.Status) {
					if _, err := fmt.Fprintf(w, "skills:  %s\n", out.SkillRefreshStatus); err != nil {
						return err
					}
					for _, refreshed := range out.RefreshedSkills {
						if _, err := fmt.Fprintf(w, "skill:   %s [%s] -> %s\n", refreshed.Agent, refreshed.Scope, refreshed.Path); err != nil {
							return err
						}
					}
					if out.ProjectMigrationStatus != "" {
						if _, err := fmt.Fprintf(w, "project migrations: %s\n", out.ProjectMigrationStatus); err != nil {
							return err
						}
						for _, applied := range out.AppliedProjectMigrations {
							if _, err := fmt.Fprintf(w, "migration: %s\n", applied); err != nil {
								return err
							}
						}
						for _, message := range out.ProjectMigrationMessages {
							if _, err := fmt.Fprintf(w, "message: %s\n", message); err != nil {
								return err
							}
						}
					}
					if out.KarpathyGuidance != nil && out.KarpathyGuidance.Recommendation != nil {
						if _, err := fmt.Fprintln(w); err != nil {
							return err
						}
						renderKarpathyGuidanceRecommendation(w, out.KarpathyGuidance.Recommendation)
					}
				}
				return nil
			}); err != nil {
				return err
			}
			if refreshErr != nil {
				if result.Updated {
					return fmt.Errorf("binary updated, skill refresh incomplete: %w", refreshErr)
				}
				return errors.New("skill refresh incomplete: " + refreshErr.Error())
			}
			if migrationErr != nil {
				if result.Updated {
					return fmt.Errorf("binary updated, project migration incomplete: %w", migrationErr)
				}
				return errors.New("project migration incomplete: " + migrationErr.Error())
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "check for updates without installing")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "include prereleases when selecting the latest release")
	root.AddCommand(cmd)
}

func shouldRefreshSkills(status string) bool {
	switch status {
	case "updated", "installed_to_fallback", "up_to_date":
		return true
	default:
		return false
	}
}
