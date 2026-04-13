package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"brain/internal/projectcontext"
	"brain/internal/search"

	"github.com/spf13/cobra"
)

func addContextCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	var project string
	var agents []string
	var dryRun bool
	var force bool
	var level int
	var query string

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
			projectRoot := contextProjectPath(project, flags.projectPath)
			return runContextCommand(cmd, loadApp, projectcontext.Request{
				ProjectDir: projectRoot,
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
			projectRoot := contextProjectPath(project, flags.projectPath)
			return runContextCommand(cmd, loadApp, projectcontext.Request{
				ProjectDir: projectRoot,
				Agents:     agents,
				DryRun:     dryRun,
				Force:      force,
			}, false)
		},
	}

	loadCmd := &cobra.Command{
		Use:   "load",
		Short: "Load a deterministic context bundle by level",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot := contextProjectPath(project, flags.projectPath)
			appCtx, err := loadApp(projectRoot)
			if err != nil {
				return err
			}
			defer appCtx.Close()

			bundle, err := appCtx.Context.Load(projectcontext.LoadRequest{
				ProjectDir: projectRoot,
				Level:      level,
			})
			if err != nil {
				return err
			}

			if level == 3 {
				activeTask := ""
				active, err := appCtx.Session.Active(projectRoot)
				if err == nil && active != nil {
					activeTask = active.Task
				}
				resolvedQuery := strings.TrimSpace(query)
				if resolvedQuery == "" {
					resolvedQuery = strings.TrimSpace(activeTask)
				}
				if resolvedQuery == "" {
					return errors.New("context load --level 3 requires --query or an active session task")
				}
				if err := appCtx.SyncIndex(cmd.Context()); err != nil {
					return err
				}
				results, err := appCtx.Search.SearchWithOptions(cmd.Context(), resolvedQuery, 5, search.Options{ActiveTask: activeTask})
				if err != nil {
					return err
				}
				bundle.Sources = append(bundle.Sources, fmt.Sprintf("search:%s", resolvedQuery))
				bundle.Content = strings.TrimRight(bundle.Content, "\n") + "\n\n## Source: search:" + resolvedQuery + "\n\n" + strings.TrimSpace(search.BuildContextBlock(results)) + "\n"
			}

			return appCtx.Output.Print(bundle, func(w io.Writer) error {
				_, err := io.WriteString(w, bundle.Content)
				return err
			})
		},
	}

	for _, sub := range []*cobra.Command{installCmd, refreshCmd} {
		sub.Flags().StringVar(&project, "project", "", "project root to scan and update")
		sub.Flags().StringArrayVarP(&agents, "agent", "a", nil, "agent wrapper to generate; repeatable")
		sub.Flags().BoolVar(&dryRun, "dry-run", false, "show planned changes without writing files")
		sub.Flags().BoolVar(&force, "force", false, "adopt unmanaged files by preserving existing content under Local Notes")
	}
	loadCmd.Flags().StringVar(&project, "project", "", "project root to load context from")
	loadCmd.Flags().IntVar(&level, "level", 0, "context depth to load: 0, 1, 2, or 3")
	loadCmd.Flags().StringVar(&query, "query", "", "search query for level 3 context")

	contextCmd.AddCommand(installCmd, refreshCmd, loadCmd)
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

func contextProjectPath(localProject, rootProject string) string {
	if strings.TrimSpace(localProject) != "" {
		return localProject
	}
	return rootProject
}
