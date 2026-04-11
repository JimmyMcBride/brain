package cmd

import (
	"fmt"
	"io"

	"brain/internal/notes"
	"brain/internal/plan"

	"github.com/spf13/cobra"
)

func addPlanCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Epic-only spec-driven planning",
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize epic/spec/story planning for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			info, err := appCtx.Project.Init()
			if err != nil {
				return err
			}
			return appCtx.Output.Print(info, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Initialized epic-only planning at %s\n", info.MetaPath)
				return err
			})
		},
	}

	epicCmd := &cobra.Command{
		Use:   "epic",
		Short: "Manage epics",
	}

	epicCreateCmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create an epic and its primary draft spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			bundle, err := appCtx.Plan.CreateEpic(args[0], "")
			if err != nil {
				return err
			}
			return appCtx.Output.Print(bundle, func(w io.Writer) error {
				if _, err := fmt.Fprintf(w, "Created epic %s at %s\n", bundle.Epic.Title, bundle.Epic.Path); err != nil {
					return err
				}
				_, err := fmt.Fprintf(w, "Created draft spec %s at %s\n", bundle.Spec.Title, bundle.Spec.Path)
				return err
			})
		},
	}

	epicListCmd := &cobra.Command{
		Use:   "list",
		Short: "List epics in the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			epics, err := appCtx.Plan.ListEpics()
			if err != nil {
				return err
			}
			return appCtx.Output.Print(epics, func(w io.Writer) error {
				if len(epics) == 0 {
					_, err := fmt.Fprintln(w, "No epics found.")
					return err
				}
				if _, err := fmt.Fprintln(w, "Epics:"); err != nil {
					return err
				}
				for _, epic := range epics {
					if _, err := fmt.Fprintf(w, "  %s [%s] (%d/%d stories done)\n", epic.Title, epic.SpecStatus, epic.DoneStories, epic.TotalStories); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	epicShowCmd := &cobra.Command{
		Use:   "show <epic-slug>",
		Short: "Show an epic note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Plan.ReadEpic(args[0])
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "%s\n\n%s", note.Path, note.Content)
				return err
			})
		},
	}

	epicPromoteCmd := &cobra.Command{
		Use:   "promote <brainstorm-slug>",
		Short: "Promote a brainstorm into an epic and seeded draft spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			bundle, err := appCtx.Plan.PromoteBrainstorm(args[0])
			if err != nil {
				return err
			}
			return appCtx.Output.Print(bundle, func(w io.Writer) error {
				if _, err := fmt.Fprintf(w, "Created epic %s at %s\n", bundle.Epic.Title, bundle.Epic.Path); err != nil {
					return err
				}
				_, err := fmt.Fprintf(w, "Created seeded draft spec %s at %s\n", bundle.Spec.Title, bundle.Spec.Path)
				return err
			})
		},
	}
	epicCmd.AddCommand(epicCreateCmd, epicListCmd, epicShowCmd, epicPromoteCmd)

	specCmd := &cobra.Command{
		Use:   "spec",
		Short: "Manage epic specs",
	}

	specShowCmd := &cobra.Command{
		Use:   "show <epic-slug>",
		Short: "Show the canonical spec for an epic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Plan.ReadSpec(args[0])
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "%s\n\n%s", note.Path, note.Content)
				return err
			})
		},
	}

	var specTitle string
	var specBody string
	var specFromStdin bool
	var specMeta []string
	var specEditor string
	specUpdateCmd := &cobra.Command{
		Use:   "update <epic-slug>",
		Short: "Update an epic spec using flags, stdin, or your editor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			if specTitle == "" && specBody == "" && !specFromStdin && len(specMeta) == 0 {
				note, err := appCtx.Plan.ReadSpec(args[0])
				if err != nil {
					return err
				}
				edited, err := appCtx.Notes.EditInEditor(note.Path, specEditor)
				if err != nil {
					return err
				}
				return appCtx.Output.Print(edited, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "Edited %s\n", edited.Path)
					return err
				})
			}

			metadata, err := parseMeta(specMeta)
			if err != nil {
				return err
			}
			body, err := readBody(cmd.InOrStdin(), specBody, specFromStdin)
			if err != nil {
				return err
			}
			update := notes.UpdateInput{
				Metadata: metadata,
				Summary:  "updated spec via CLI",
			}
			if specTitle != "" {
				update.Title = &specTitle
			}
			if body != "" || specFromStdin {
				update.Body = &body
			}
			note, err := appCtx.Plan.UpdateSpec(args[0], update)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Updated %s\n", note.Path)
				return err
			})
		},
	}
	specUpdateCmd.Flags().StringVarP(&specTitle, "title", "t", "", "update the spec title")
	specUpdateCmd.Flags().StringVarP(&specBody, "body", "b", "", "replace body content")
	specUpdateCmd.Flags().BoolVar(&specFromStdin, "stdin", false, "read body from stdin")
	specUpdateCmd.Flags().StringArrayVarP(&specMeta, "set", "m", nil, "metadata key=value")
	specUpdateCmd.Flags().StringVar(&specEditor, "editor", "", "editor to launch when no direct edits are supplied")

	var specStatus string
	specStatusCmd := &cobra.Command{
		Use:   "status <epic-slug>",
		Short: "Set the status of an epic spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Plan.SetSpecStatus(args[0], specStatus)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Set spec %s to %s\n", note.Path, specStatus)
				return err
			})
		},
	}
	specStatusCmd.Flags().StringVar(&specStatus, "set", "", "new spec status (draft, approved, implementing, done)")
	_ = specStatusCmd.MarkFlagRequired("set")

	specCmd.AddCommand(specShowCmd, specUpdateCmd, specStatusCmd)

	storyCmd := &cobra.Command{
		Use:   "story",
		Short: "Manage execution stories",
	}

	var storyBody string
	var storyCriteria []string
	var storyResources []string
	storyCreateCmd := &cobra.Command{
		Use:   "create <epic-slug> <title>",
		Short: "Create a story from an approved epic spec",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Plan.CreateStory(args[0], args[1], storyBody, storyCriteria, storyResources)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Created story %s at %s\n", note.Title, note.Path)
				return err
			})
		},
	}
	storyCreateCmd.Flags().StringVarP(&storyBody, "body", "b", "", "description")
	storyCreateCmd.Flags().StringArrayVar(&storyCriteria, "criteria", nil, "acceptance criterion; repeatable")
	storyCreateCmd.Flags().StringArrayVar(&storyResources, "resource", nil, "resource reference; repeatable")

	var storyStatus string
	var storyUpdateCriteria []string
	var storyUpdateResources []string
	storyUpdateCmd := &cobra.Command{
		Use:   "update <story-slug>",
		Short: "Update a story",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Plan.UpdateStory(args[0], plan.StoryChanges{
				Status:       storyStatus,
				AddCriteria:  storyUpdateCriteria,
				AddResources: storyUpdateResources,
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Updated %s\n", note.Path)
				return err
			})
		},
	}
	storyUpdateCmd.Flags().StringVar(&storyStatus, "status", "", "new story status (todo, in_progress, blocked, done)")
	storyUpdateCmd.Flags().StringArrayVar(&storyUpdateCriteria, "criteria", nil, "acceptance criterion to add; repeatable")
	storyUpdateCmd.Flags().StringArrayVar(&storyUpdateResources, "resource", nil, "resource reference to add; repeatable")

	var listEpic string
	var listStatus string
	storyListCmd := &cobra.Command{
		Use:   "list",
		Short: "List stories in the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			stories, err := appCtx.Plan.ListStories(listEpic, listStatus)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(stories, func(w io.Writer) error {
				if len(stories) == 0 {
					_, err := fmt.Fprintln(w, "No stories found.")
					return err
				}
				if _, err := fmt.Fprintln(w, "Stories:"); err != nil {
					return err
				}
				for _, story := range stories {
					line := fmt.Sprintf("  %s %s", statusIcon(story.Status), story.Title)
					if story.Epic != "" {
						line += fmt.Sprintf(" [%s]", story.Epic)
					}
					if _, err := fmt.Fprintln(w, line); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
	storyListCmd.Flags().StringVar(&listEpic, "epic", "", "filter by epic")
	storyListCmd.Flags().StringVar(&listStatus, "status", "", "filter by status (todo, in_progress, blocked, done)")

	storyCmd.AddCommand(storyCreateCmd, storyUpdateCmd, storyListCmd)

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show epic/spec/story planning status",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			status, err := appCtx.Plan.Status()
			if err != nil {
				return err
			}
			return appCtx.Output.Print(status, func(w io.Writer) error {
				if _, err := fmt.Fprintf(w, "%s (%s)\n", status.Project, status.PlanningModel); err != nil {
					return err
				}
				remaining := status.TotalStories - status.DoneStories
				if _, err := fmt.Fprintf(w, "  Stories: %d total, %d done, %d in progress, %d blocked, %d remaining\n", status.TotalStories, status.DoneStories, status.InProgressStories, status.BlockedStories, remaining); err != nil {
					return err
				}
				for _, epic := range status.Epics {
					if _, err := fmt.Fprintf(w, "  Epic %s [%s]: %d/%d stories done\n", epic.Title, epic.SpecStatus, epic.DoneStories, epic.TotalStories); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	planCmd.AddCommand(initCmd, epicCmd, specCmd, storyCmd, statusCmd)
	root.AddCommand(planCmd)
}

func statusIcon(status string) string {
	switch status {
	case "done":
		return "[x]"
	case "in_progress":
		return "[~]"
	case "blocked":
		return "[!]"
	default:
		return "[ ]"
	}
}
