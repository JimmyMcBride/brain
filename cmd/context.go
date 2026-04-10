package cmd

import (
	"fmt"
	"io"
	"strings"

	"brain/internal/projectcontext"

	"github.com/spf13/cobra"
)

func addContextCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var project string
	var agents []string
	var dryRun bool
	var force bool

	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Install or refresh project agent context files",
		Long: strings.TrimSpace(`
Manage project-local agent context files owned by brain.

This creates a minimal root AGENTS/CLAUDE contract plus a modular
.brain/context bundle that can be refreshed as the repository evolves.
`),
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Create or update the project context bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextCommand(cmd, loadApp, projectcontext.Request{
				ProjectDir: project,
				Agents:     agents,
				DryRun:     dryRun,
				Force:      force,
			}, true)
		},
	}

	refreshCmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh brain-managed project context files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextCommand(cmd, loadApp, projectcontext.Request{
				ProjectDir: project,
				Agents:     agents,
				DryRun:     dryRun,
				Force:      force,
			}, false)
		},
	}

	for _, sub := range []*cobra.Command{installCmd, refreshCmd} {
		sub.Flags().StringVar(&project, "project", ".", "project root to scan and update")
		sub.Flags().StringArrayVarP(&agents, "agent", "a", nil, "agent wrapper to generate; repeatable")
		sub.Flags().BoolVar(&dryRun, "dry-run", false, "show planned changes without writing files")
		sub.Flags().BoolVar(&force, "force", false, "adopt unmanaged files by preserving existing content under Local Notes")
	}

	contextCmd.AddCommand(installCmd, refreshCmd)
	root.AddCommand(contextCmd)
}

func runContextCommand(cmd *cobra.Command, loadApp appLoader, req projectcontext.Request, install bool) error {
	appCtx, err := loadApp()
	if err != nil {
		return err
	}
	defer appCtx.Close()

	var results []projectcontext.Result
	if install {
		results, err = appCtx.Context.Install(cmd.Context(), req)
	} else {
		results, err = appCtx.Context.Refresh(cmd.Context(), req)
	}
	if err != nil {
		return err
	}
	return appCtx.Output.Print(results, func(w io.Writer) error {
		for _, result := range results {
			preserve := ""
			if result.PreservedUserContent {
				preserve = " preserve-user"
			}
			if _, err := fmt.Fprintf(w, "%-9s %-8s %s%s\n", result.Action, result.Kind, result.Path, preserve); err != nil {
				return err
			}
		}
		return nil
	})
}
