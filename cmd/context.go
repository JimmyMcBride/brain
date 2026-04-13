package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"brain/internal/contextassembly"
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
	var assembleTask string
	var assembleLimit int
	var assembleExplain bool

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

	assembleCmd := &cobra.Command{
		Use:   "assemble",
		Short: "Assemble a task-focused context packet",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot := contextProjectPath(project, flags.projectPath)
			appCtx, err := loadApp(projectRoot)
			if err != nil {
				return err
			}
			defer appCtx.Close()

			resolvedTask := strings.TrimSpace(assembleTask)
			taskSource := "flag"
			if resolvedTask == "" {
				active, err := appCtx.Session.Active(projectRoot)
				if err != nil {
					return err
				}
				if active != nil {
					resolvedTask = strings.TrimSpace(active.Task)
					taskSource = "session"
				}
			}
			if resolvedTask == "" {
				return errors.New("context assemble requires --task or an active session task")
			}
			searchResults := []search.Result{}
			if err := appCtx.SyncIndex(cmd.Context()); err != nil {
				return err
			}
			activeTask := ""
			active, err := appCtx.Session.Active(projectRoot)
			if err != nil {
				return err
			}
			if active != nil {
				activeTask = strings.TrimSpace(active.Task)
			}
			searchLimit := 16
			if assembleLimit > 0 && assembleLimit*4 > searchLimit {
				searchLimit = assembleLimit * 4
			}
			searchResults, err = appCtx.Search.SearchWithOptions(cmd.Context(), resolvedTask, searchLimit, search.Options{ActiveTask: activeTask})
			if err != nil {
				return err
			}

			manager := contextassembly.New(appCtx.Context)
			packet, err := manager.Assemble(contextassembly.Request{
				ProjectDir:    projectRoot,
				Task:          resolvedTask,
				TaskSource:    taskSource,
				Limit:         assembleLimit,
				Explain:       assembleExplain,
				SearchResults: searchResults,
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(packet, func(w io.Writer) error {
				return contextassembly.RenderHuman(w, packet, assembleExplain)
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
	assembleCmd.Flags().StringVar(&project, "project", "", "project root to assemble context from")
	assembleCmd.Flags().StringVar(&assembleTask, "task", "", "task text to assemble context for")
	assembleCmd.Flags().IntVar(&assembleLimit, "limit", 8, "maximum selected context items")
	assembleCmd.Flags().BoolVar(&assembleExplain, "explain", false, "include selection rationale and omitted context")

	contextCmd.AddCommand(installCmd, refreshCmd, loadCmd, assembleCmd)
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
