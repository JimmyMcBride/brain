package cmd

import (
	"fmt"
	"io"
	"strings"

	"brain/internal/plan"

	"github.com/spf13/cobra"
)

func addPlanCommand(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Project-local planning and work tracking",
	}

	var paradigmFlag string
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize project management for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			info, err := appCtx.Project.Init(paradigmFlag)
			if err != nil {
				return err
			}
			return appCtx.Output.Print(info, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Initialized planning with %s paradigm at %s\n", info.Paradigm.Name, info.MetaPath)
				return err
			})
		},
	}
	initCmd.Flags().StringVar(&paradigmFlag, "paradigm", "", "PM paradigm (epics, milestones, cycles)")
	_ = initCmd.MarkFlagRequired("paradigm")

	groupCmd := &cobra.Command{
		Use:   "group",
		Short: "Manage containers (epics, milestones, or cycles)",
	}
	groupCreateCmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a new container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Plan.CreateContainer(args[0])
			if err != nil {
				return err
			}
			info, _ := appCtx.Project.Resolve()
			label := "group"
			if info != nil && info.Paradigm != nil {
				label = info.Paradigm.ContainerType
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Created %s %s at %s\n", label, note.Title, note.Path)
				return err
			})
		},
	}
	groupListCmd := &cobra.Command{
		Use:   "list",
		Short: "List containers in the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			containers, err := appCtx.Plan.ListContainers()
			if err != nil {
				return err
			}
			info, _ := appCtx.Project.Resolve()
			label := "Groups"
			if info != nil && info.Paradigm != nil {
				label = strings.Title(info.Paradigm.ContainerPlural)
			}
			return appCtx.Output.Print(containers, func(w io.Writer) error {
				if len(containers) == 0 {
					_, err := fmt.Fprintf(w, "No %s found.\n", strings.ToLower(label))
					return err
				}
				if _, err := fmt.Fprintf(w, "%s:\n", label); err != nil {
					return err
				}
				for _, c := range containers {
					if _, err := fmt.Fprintf(w, "  %s (%d/%d done)\n", c.Title, c.DoneItems, c.TotalItems); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
	groupCmd.AddCommand(groupCreateCmd, groupListCmd)

	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "Manage work items (stories or tasks)",
	}
	var itemGroup string
	var itemBody string
	var itemCriteria []string
	var itemResources []string
	itemCreateCmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a new work item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Plan.CreateItem(args[0], itemGroup, itemBody, itemCriteria, itemResources)
			if err != nil {
				return err
			}
			info, _ := appCtx.Project.Resolve()
			label := "item"
			if info != nil && info.Paradigm != nil {
				label = info.Paradigm.ItemType
			}
			return appCtx.Output.Print(note, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Created %s %s at %s\n", label, note.Title, note.Path)
				return err
			})
		},
	}
	itemCreateCmd.Flags().StringVar(&itemGroup, "group", "", "container to assign this item to")
	itemCreateCmd.Flags().StringVarP(&itemBody, "body", "b", "", "description")
	itemCreateCmd.Flags().StringArrayVar(&itemCriteria, "criteria", nil, "acceptance criterion; repeatable")
	itemCreateCmd.Flags().StringArrayVar(&itemResources, "resource", nil, "resource reference; repeatable")

	var updateStatus string
	var updateCriteria []string
	var updateResources []string
	itemUpdateCmd := &cobra.Command{
		Use:   "update <item-slug>",
		Short: "Update a work item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			note, err := appCtx.Plan.UpdateItem(args[0], plan.ItemChanges{
				Status:       updateStatus,
				AddCriteria:  updateCriteria,
				AddResources: updateResources,
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
	itemUpdateCmd.Flags().StringVar(&updateStatus, "status", "", "new status (todo, in_progress, done)")
	itemUpdateCmd.Flags().StringArrayVar(&updateCriteria, "criteria", nil, "acceptance criterion to add; repeatable")
	itemUpdateCmd.Flags().StringArrayVar(&updateResources, "resource", nil, "resource reference to add; repeatable")

	var listGroup string
	var listStatus string
	itemListCmd := &cobra.Command{
		Use:   "list",
		Short: "List work items in the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			items, err := appCtx.Plan.ListItems(listGroup, listStatus)
			if err != nil {
				return err
			}
			info, _ := appCtx.Project.Resolve()
			label := "Items"
			if info != nil && info.Paradigm != nil {
				label = strings.Title(info.Paradigm.ItemPlural)
			}
			return appCtx.Output.Print(items, func(w io.Writer) error {
				if len(items) == 0 {
					_, err := fmt.Fprintf(w, "No %s found.\n", strings.ToLower(label))
					return err
				}
				if _, err := fmt.Fprintf(w, "%s:\n", label); err != nil {
					return err
				}
				for _, item := range items {
					line := fmt.Sprintf("  %s %s", statusIcon(item.Status), item.Title)
					if item.Container != "" {
						line += fmt.Sprintf(" [%s]", item.Container)
					}
					if _, err := fmt.Fprintln(w, line); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
	itemListCmd.Flags().StringVar(&listGroup, "group", "", "filter by container")
	itemListCmd.Flags().StringVar(&listStatus, "status", "", "filter by status (todo, in_progress, done)")
	itemCmd.AddCommand(itemCreateCmd, itemUpdateCmd, itemListCmd)

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show project planning status",
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
				if _, err := fmt.Fprintf(w, "%s (%s)\n", status.Project, status.ParadigmName); err != nil {
					return err
				}
				remaining := status.TotalItems - status.DoneItems
				if _, err := fmt.Fprintf(w, "  %s: %d total, %d done, %d remaining\n", strings.Title(status.ItemPlural), status.TotalItems, status.DoneItems, remaining); err != nil {
					return err
				}
				for _, c := range status.Containers {
					if _, err := fmt.Fprintf(w, "  %s %s: %d/%d done\n", strings.Title(status.ContainerType), c.Title, c.DoneItems, c.TotalItems); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	promoteCmd := &cobra.Command{
		Use:   "promote <brainstorm-slug>",
		Short: "Promote brainstorm ideas into work items",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			items, err := appCtx.Plan.Promote(args[0])
			if err != nil {
				return err
			}
			info, _ := appCtx.Project.Resolve()
			label := "items"
			if info != nil && info.Paradigm != nil {
				label = info.Paradigm.ItemPlural
			}
			return appCtx.Output.Print(items, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Created %d %s from brainstorm\n", len(items), label)
				return err
			})
		},
	}

	planCmd.AddCommand(initCmd, groupCmd, itemCmd, statusCmd, promoteCmd)
	root.AddCommand(planCmd)
}

func statusIcon(status string) string {
	switch status {
	case "done":
		return "[x]"
	case "in_progress":
		return "[~]"
	default:
		return "[ ]"
	}
}
