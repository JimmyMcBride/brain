package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"brain/internal/contextassembly"
	"brain/internal/livecontext"
	"brain/internal/projectcontext"
	"brain/internal/search"
	"brain/internal/structure"
	"brain/internal/taskcontext"

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
	var compileTask string
	var structurePath string
	var liveTask string
	var liveExplain bool

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
			activeTask := ""
			hasActiveSession := false
			active, err := appCtx.Session.Active(projectRoot)
			if err != nil {
				return err
			}
			if active != nil {
				hasActiveSession = true
				activeTask = strings.TrimSpace(active.Task)
			}
			if err := appCtx.SyncIndex(cmd.Context()); err != nil {
				return err
			}
			searchLimit := 16
			if assembleLimit > 0 && assembleLimit*4 > searchLimit {
				searchLimit = assembleLimit * 4
			}
			searchResults, err = appCtx.Search.SearchWithOptions(cmd.Context(), resolvedTask, searchLimit, search.Options{ActiveTask: activeTask})
			if err != nil {
				return err
			}
			structureSnapshot, err := appCtx.Structure.Snapshot(cmd.Context(), "")
			if err != nil {
				return err
			}
			boundaryGraph, err := appCtx.Structure.BoundaryGraph(cmd.Context())
			if err != nil {
				return err
			}
			structuralItems := append([]structure.Item{}, structureSnapshot.Boundaries...)
			structuralItems = append(structuralItems, structureSnapshot.Entrypoints...)
			structuralItems = append(structuralItems, structureSnapshot.ConfigSurfaces...)
			structuralItems = append(structuralItems, structureSnapshot.TestSurfaces...)
			livePacket, err := appCtx.Live.Collect(cmd.Context(), livecontext.Request{
				ProjectDir:    projectRoot,
				Task:          resolvedTask,
				TaskSource:    taskSource,
				Session:       active,
				BoundaryGraph: boundaryGraph,
				Explain:       assembleExplain,
			})
			if err != nil {
				return err
			}

			manager := contextassembly.New(appCtx.Context)
			packet, err := manager.Assemble(contextassembly.Request{
				ProjectDir:       projectRoot,
				Task:             resolvedTask,
				TaskSource:       taskSource,
				HasActiveSession: hasActiveSession,
				Limit:            assembleLimit,
				Explain:          assembleExplain,
				SearchResults:    searchResults,
				StructuralItems:  structuralItems,
				LivePacket:       livePacket,
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(packet, func(w io.Writer) error {
				return contextassembly.RenderHuman(w, packet, assembleExplain)
			})
		},
	}

	compileCmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile a summary-first working-set packet for a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot := contextProjectPath(project, flags.projectPath)
			appCtx, err := loadApp(projectRoot)
			if err != nil {
				return err
			}
			defer appCtx.Close()

			active, err := appCtx.Session.Active(projectRoot)
			if err != nil {
				return err
			}

			resolvedTask := strings.TrimSpace(compileTask)
			taskSource := "flag"
			if resolvedTask == "" && active != nil {
				resolvedTask = strings.TrimSpace(active.Task)
				taskSource = "session"
			}
			if resolvedTask == "" {
				return errors.New("context compile requires --task or an active session task")
			}

			activeTask := ""
			if active != nil {
				activeTask = strings.TrimSpace(active.Task)
			}
			if err := appCtx.SyncIndex(cmd.Context()); err != nil {
				return err
			}
			searchResults, err := appCtx.Search.SearchWithOptions(cmd.Context(), resolvedTask, 12, search.Options{ActiveTask: activeTask})
			if err != nil {
				return err
			}
			boundaryGraph, err := appCtx.Structure.BoundaryGraph(cmd.Context())
			if err != nil {
				return err
			}
			livePacket, err := appCtx.Live.Collect(cmd.Context(), livecontext.Request{
				ProjectDir:    projectRoot,
				Task:          resolvedTask,
				TaskSource:    taskSource,
				Session:       active,
				BoundaryGraph: boundaryGraph,
			})
			if err != nil {
				return err
			}

			manager := taskcontext.New(appCtx.Context)
			packet, err := manager.Compile(taskcontext.Request{
				ProjectDir:    projectRoot,
				Task:          resolvedTask,
				TaskSource:    taskSource,
				SearchResults: searchResults,
				LivePacket:    livePacket,
				BoundaryGraph: boundaryGraph,
			})
			if err != nil {
				return err
			}
			if active != nil {
				if err := appCtx.Session.RecordCompiledPacket(projectRoot, active.ID, packet); err != nil {
					return err
				}
			}
			return appCtx.Output.Print(packet, func(w io.Writer) error {
				return taskcontext.RenderHuman(w, packet)
			})
		},
	}

	structureCmd := &cobra.Command{
		Use:   "structure",
		Short: "Inspect structural repo context",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot := contextProjectPath(project, flags.projectPath)
			appCtx, err := loadApp(projectRoot)
			if err != nil {
				return err
			}
			defer appCtx.Close()

			snapshot, err := appCtx.Structure.Snapshot(cmd.Context(), structurePath)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(snapshot, func(w io.Writer) error {
				if _, err := fmt.Fprintf(w, "## Repository Shape\n\n- Runtime: `%s`\n- Items: %d\n\n", snapshot.Summary.Runtime, snapshot.Summary.ItemCount); err != nil {
					return err
				}
				for _, entry := range []struct {
					label string
					items []structure.Item
				}{
					{label: "Boundaries", items: snapshot.Boundaries},
					{label: "Entrypoints", items: snapshot.Entrypoints},
					{label: "Config Surfaces", items: snapshot.ConfigSurfaces},
					{label: "Test Surfaces", items: snapshot.TestSurfaces},
				} {
					if _, err := fmt.Fprintf(w, "## %s\n\n", entry.label); err != nil {
						return err
					}
					if len(entry.items) == 0 {
						if _, err := io.WriteString(w, "- None.\n\n"); err != nil {
							return err
						}
						continue
					}
					for _, item := range entry.items {
						if _, err := fmt.Fprintf(w, "- `%s` [%s]: %s\n", item.Path, item.Role, item.Summary); err != nil {
							return err
						}
					}
					if _, err := io.WriteString(w, "\n"); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	liveCmd := &cobra.Command{
		Use:   "live",
		Short: "Inspect live work context for the active task",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot := contextProjectPath(project, flags.projectPath)
			appCtx, err := loadApp(projectRoot)
			if err != nil {
				return err
			}
			defer appCtx.Close()

			active, err := appCtx.Session.Active(projectRoot)
			if err != nil {
				return err
			}
			resolvedTask := strings.TrimSpace(liveTask)
			taskSource := "flag"
			if resolvedTask == "" && active != nil {
				resolvedTask = strings.TrimSpace(active.Task)
				taskSource = "session"
			}
			if resolvedTask == "" {
				return errors.New("context live requires --task or an active session task")
			}
			boundaryGraph, err := appCtx.Structure.BoundaryGraph(cmd.Context())
			if err != nil {
				return err
			}

			packet, err := appCtx.Live.Collect(cmd.Context(), livecontext.Request{
				ProjectDir:    projectRoot,
				Task:          resolvedTask,
				TaskSource:    taskSource,
				Session:       active,
				BoundaryGraph: boundaryGraph,
				Explain:       liveExplain,
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(packet, func(w io.Writer) error {
				return livecontext.RenderHuman(w, packet, liveExplain)
			})
		},
	}

	structureStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show structural repo context freshness and counts",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot := contextProjectPath(project, flags.projectPath)
			appCtx, err := loadApp(projectRoot)
			if err != nil {
				return err
			}
			defer appCtx.Close()

			status, err := appCtx.Structure.Freshness(cmd.Context())
			if err != nil {
				return err
			}
			return appCtx.Output.Print(status, func(w io.Writer) error {
				if _, err := fmt.Fprintf(w, "state: %s (%s)\n", status.State, status.Reason); err != nil {
					return err
				}
				if status.IndexedAt != "" {
					if _, err := fmt.Fprintf(w, "indexed_at: %s\n", status.IndexedAt); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintf(w, "files: %d current, %d indexed\n", status.CurrentFileCount, status.IndexedFileCount); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "items: %d\n", status.ItemCount); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "boundaries: %d\nentrypoints: %d\nconfig_surfaces: %d\ntest_surfaces: %d\n", status.BoundaryCount, status.EntrypointCount, status.ConfigSurfaceCount, status.TestSurfaceCount); err != nil {
					return err
				}
				return nil
			})
		},
	}

	for _, sub := range []*cobra.Command{installCmd, refreshCmd} {
		sub.Flags().StringVar(&project, "project", "", "project root to scan and update")
		sub.Flags().StringArrayVarP(&agents, "agent", "a", nil, "agent instruction files to integrate when present; repeatable")
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
	compileCmd.Flags().StringVar(&project, "project", "", "project root to compile context from")
	compileCmd.Flags().StringVar(&compileTask, "task", "", "task text to compile context for; defaults to the active session task")
	structureCmd.Flags().StringVar(&project, "project", "", "project root to inspect structure from")
	structureCmd.Flags().StringVar(&structurePath, "path", "", "subtree path filter for structural context")
	structureStatusCmd.Flags().StringVar(&project, "project", "", "project root to inspect structure from")
	liveCmd.Flags().StringVar(&project, "project", "", "project root to inspect live context from")
	liveCmd.Flags().StringVar(&liveTask, "task", "", "task text for live context; defaults to the active session task")
	liveCmd.Flags().BoolVar(&liveExplain, "explain", false, "include rationale and missing-signal detail")

	structureCmd.AddCommand(structureStatusCmd)
	contextCmd.AddCommand(installCmd, refreshCmd, loadCmd, assembleCmd, compileCmd, structureCmd, liveCmd)
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
