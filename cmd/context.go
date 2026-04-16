package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"brain/internal/config"
	"brain/internal/contextassembly"
	"brain/internal/livecontext"
	"brain/internal/output"
	"brain/internal/projectcontext"
	"brain/internal/search"
	"brain/internal/session"
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
	var compileBudget string
	var compileFresh bool
	var explainPacket string
	var explainLast bool
	var statsLimit int
	var structurePath string
	var liveTask string
	var liveExplain bool

	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Compile and manage project-local context",
		Long: strings.TrimSpace(`
Compile task-sized context packets and manage the project-local context files owned by brain.

Prefer brain context compile when you need context for a task.
Use the other subcommands to inspect compatibility views or refresh the Brain-managed context bundle on disk.
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
		Short: "Load a legacy static context bundle by level",
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

	migrateCmd := &cobra.Command{
		Use:    "migrate",
		Short:  "Apply pending Brain project migrations",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.LoadOrCreate(flags.configPath)
			if err != nil {
				return err
			}
			projectRoot := contextProjectPath(project, flags.projectPath)
			result, err := contextManager().ApplyProjectMigrations(cmd.Context(), projectRoot)
			if err != nil {
				return err
			}
			printer := output.New(modeFromFlag(flags, cfg.OutputMode), cmd.OutOrStdout())
			return printer.Print(result, func(w io.Writer) error {
				if _, err := fmt.Fprintf(w, "migrations: %s\n", result.Status); err != nil {
					return err
				}
				for _, applied := range result.AppliedMigrationIDs {
					if _, err := fmt.Fprintf(w, "migration: %s\n", applied); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	assembleCmd := &cobra.Command{
		Use:   "assemble",
		Short: "Assemble a broader task-focused context packet",
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
			utilitySnapshot, err := appCtx.Session.BuildUtilitySnapshot(projectRoot)
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
			compileRequest := taskcontext.Request{
				ProjectDir:     projectRoot,
				Task:           resolvedTask,
				TaskSource:     taskSource,
				Budget:         compileBudget,
				SearchResults:  searchResults,
				LivePacket:     livePacket,
				BoundaryGraph:  boundaryGraph,
				UtilitySignals: utilitySignalsFromSnapshot(utilitySnapshot),
			}
			fingerprintInputs, err := manager.BuildFingerprintInputs(compileRequest)
			if err != nil {
				return err
			}
			fingerprint := fingerprintInputs.Hash()
			if active != nil && !compileFresh {
				if reusable := latestMatchingPacketRecord(active.PacketRecords, fingerprint); reusable != nil && reusable.Packet != nil {
					meta := projectcontext.PacketCacheMetadata{
						CacheStatus:        projectcontext.PacketCacheStatusReused,
						Fingerprint:        fingerprint,
						ReusedFrom:         reusable.PacketHash,
						FullPacketIncluded: false,
					}
					if err := appCtx.Session.RecordCompiledPacket(projectRoot, active.ID, reusable.Packet, fingerprintInputs, meta); err != nil {
						return err
					}
					response := projectcontext.NewCompileResponse(reusable.Packet, meta)
					return appCtx.Output.Print(response, func(w io.Writer) error {
						return taskcontext.RenderCompileResponseHuman(w, response)
					})
				}
			}

			packet, err := manager.Compile(compileRequest)
			if err != nil {
				return err
			}

			meta := projectcontext.PacketCacheMetadata{
				CacheStatus:        projectcontext.PacketCacheStatusFresh,
				Fingerprint:        fingerprint,
				FullPacketIncluded: true,
			}
			if compileFresh {
				meta.FallbackReason = "fresh compile requested"
			} else if active == nil {
				meta.FallbackReason = "no active session; emitted a standalone full packet"
			}
			if active != nil && !compileFresh {
				if previous := latestTaskPacketRecord(active.PacketRecords, packet.Task.Text); previous != nil {
					if previous.Fingerprint == "" {
						meta.FallbackReason = "prior packet lineage unavailable; emitted a standalone full packet"
					} else if previous.Packet != nil {
						meta.InvalidationReasons = fingerprintInputs.InvalidationReasons(previous.FingerprintInputs)
						changedSections, changedItemIDs := taskcontext.PacketDiff(previous.Packet, packet)
						if len(meta.InvalidationReasons) != 0 || len(changedSections) != 0 || len(changedItemIDs) != 0 {
							meta.CacheStatus = projectcontext.PacketCacheStatusDelta
							meta.DeltaFrom = previous.PacketHash
							meta.ChangedSections = changedSections
							meta.ChangedItemIDs = changedItemIDs
							meta.FullPacketIncluded = false
						}
					} else {
						meta.FallbackReason = "prior packet body unavailable; emitted a standalone full packet"
					}
				} else {
					meta.FallbackReason = "no prior session packet available"
				}
			}
			if active != nil {
				if err := appCtx.Session.RecordCompiledPacket(projectRoot, active.ID, packet, fingerprintInputs, meta); err != nil {
					return err
				}
			}
			response := projectcontext.NewCompileResponse(packet, meta)
			return appCtx.Output.Print(response, func(w io.Writer) error {
				return taskcontext.RenderCompileResponseHuman(w, response)
			})
		},
	}

	explainCmd := &cobra.Command{
		Use:   "explain",
		Short: "Inspect the latest compiled packet and its recorded outcomes",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot := contextProjectPath(project, flags.projectPath)
			appCtx, err := loadApp(projectRoot)
			if err != nil {
				return err
			}
			defer appCtx.Close()

			explanation, err := appCtx.Session.ExplainPacket(session.PacketExplainRequest{
				ProjectDir: projectRoot,
				PacketHash: explainPacket,
				Last:       explainLast,
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(explanation, func(w io.Writer) error {
				return session.RenderPacketExplanationHuman(w, explanation)
			})
		},
	}

	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Summarize likely signal, likely noise, and expansion patterns",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot := contextProjectPath(project, flags.projectPath)
			appCtx, err := loadApp(projectRoot)
			if err != nil {
				return err
			}
			defer appCtx.Close()

			stats, err := appCtx.Session.ContextStats(session.ContextStatsRequest{
				ProjectDir: projectRoot,
				Limit:      statsLimit,
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(stats, func(w io.Writer) error {
				return session.RenderContextStatsHuman(w, stats)
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
	loadCmd.Flags().IntVar(&level, "level", 0, "compatibility context depth to load: 0, 1, 2, or 3")
	loadCmd.Flags().StringVar(&query, "query", "", "search query for level 3 context")
	migrateCmd.Flags().StringVar(&project, "project", "", "project root to migrate")
	assembleCmd.Flags().StringVar(&project, "project", "", "project root to assemble context from")
	assembleCmd.Flags().StringVar(&assembleTask, "task", "", "task text to assemble context for")
	assembleCmd.Flags().IntVar(&assembleLimit, "limit", 8, "maximum selected context items")
	assembleCmd.Flags().BoolVar(&assembleExplain, "explain", false, "include selection rationale and omitted context")
	compileCmd.Flags().StringVar(&project, "project", "", "project root to compile context from")
	compileCmd.Flags().StringVar(&compileTask, "task", "", "task text to compile context for; defaults to the active session task")
	compileCmd.Flags().StringVar(&compileBudget, "budget", "", "packet budget preset or explicit token target; presets: small, default, large")
	compileCmd.Flags().BoolVar(&compileFresh, "fresh", false, "bypass session-local packet reuse and emit a full standalone packet")
	explainCmd.Flags().StringVar(&project, "project", "", "project root to inspect context telemetry from")
	explainCmd.Flags().StringVar(&explainPacket, "packet", "", "specific packet hash to inspect; defaults to the latest packet")
	explainCmd.Flags().BoolVar(&explainLast, "last", false, "inspect the latest packet explicitly")
	statsCmd.Flags().StringVar(&project, "project", "", "project root to inspect context telemetry from")
	statsCmd.Flags().IntVar(&statsLimit, "limit", 5, "maximum number of entries to show per stats section")
	structureCmd.Flags().StringVar(&project, "project", "", "project root to inspect structure from")
	structureCmd.Flags().StringVar(&structurePath, "path", "", "subtree path filter for structural context")
	structureStatusCmd.Flags().StringVar(&project, "project", "", "project root to inspect structure from")
	liveCmd.Flags().StringVar(&project, "project", "", "project root to inspect live context from")
	liveCmd.Flags().StringVar(&liveTask, "task", "", "task text for live context; defaults to the active session task")
	liveCmd.Flags().BoolVar(&liveExplain, "explain", false, "include rationale and missing-signal detail")

	structureCmd.AddCommand(structureStatusCmd)
	contextCmd.AddCommand(installCmd, refreshCmd, loadCmd, migrateCmd, assembleCmd, compileCmd, explainCmd, statsCmd, structureCmd, liveCmd)
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

func utilitySignalsFromSnapshot(snapshot *session.UtilitySnapshot) map[string]taskcontext.ItemUtilitySignal {
	if snapshot == nil || len(snapshot.Items) == 0 {
		return nil
	}
	signals := make(map[string]taskcontext.ItemUtilitySignal, len(snapshot.Items))
	for _, item := range snapshot.Items {
		signals[item.ItemID] = taskcontext.ItemUtilitySignal{
			LikelyUtility:               item.LikelyUtility,
			IncludeCount:                item.IncludeCount,
			ExpandCount:                 item.ExpandCount,
			SuccessfulVerificationCount: item.SuccessfulVerificationCount,
			DurableUpdateCount:          item.DurableUpdateCount,
			UnusedIncludeCount:          item.UnusedIncludeCount,
			UtilityScore:                item.UtilityScore,
			NoiseScore:                  item.NoiseScore,
			Reasons:                     append([]string(nil), item.Reasons...),
		}
	}
	return signals
}
