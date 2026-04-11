package cmd

import (
	"context"
	"fmt"
	"io"

	"brain/internal/config"
	"brain/internal/output"
	"brain/internal/update"

	"github.com/spf13/cobra"
)

type updater interface {
	Update(context.Context, update.Request) (update.Result, error)
}

var newUpdater = func(cfg *config.Config, paths config.Paths) updater {
	return update.New(cfg, paths, update.Options{})
}

func addUpdateCommand(root *cobra.Command, flags *rootFlagsState) {
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
			result, err := manager.Update(cmd.Context(), update.Request{
				CheckOnly:         checkOnly,
				IncludePrerelease: prerelease,
			})
			if err != nil {
				return err
			}

			printer := output.New(modeFromFlag(flags, cfg.OutputMode), cmd.OutOrStdout())
			return printer.Print(result, func(w io.Writer) error {
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
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "check for updates without installing")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "include prereleases when selecting the latest release")
	root.AddCommand(cmd)
}
