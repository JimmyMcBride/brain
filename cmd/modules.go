package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/JimmyMcBride/brain/internal/modules"

	"github.com/spf13/cobra"
)

func addModulesCommand(root *cobra.Command, _ *rootFlagsState, loadApp appLoader) {
	modulesCmd := &cobra.Command{
		Use:   "modules",
		Short: "Inspect and manage optional Brain modules",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available and configured modules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withModuleRuntime(loadApp, func(runtime *modules.Runtime, printer modulePrinter) error {
				reports, err := runtime.List(cmd.Context())
				if err != nil {
					return err
				}
				return printer.Print(reports, func(w io.Writer) error {
					if len(reports) == 0 {
						_, err := fmt.Fprintln(w, "No modules registered.")
						return err
					}
					return printModuleReports(w, reports)
				})
			})
		},
	}

	showCmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show one module's descriptor and project state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withModuleRuntime(loadApp, func(runtime *modules.Runtime, printer modulePrinter) error {
				report, err := runtime.Show(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				return printer.Print(report, func(w io.Writer) error {
					return printModuleDetails(w, report)
				})
			})
		},
	}

	grantCmd := &cobra.Command{
		Use:   "grant <id> <permission>...",
		Short: "Grant local permissions to a module",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutateModule(cmd, loadApp, func(runtime *modules.Runtime) (modules.Report, error) {
				return runtime.Grant(cmd.Context(), args[0], args[1:]...)
			})
		},
	}

	revokeCmd := &cobra.Command{
		Use:   "revoke <id> <permission>...",
		Short: "Revoke local permissions from a module",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutateModule(cmd, loadApp, func(runtime *modules.Runtime) (modules.Report, error) {
				return runtime.Revoke(cmd.Context(), args[0], args[1:]...)
			})
		},
	}

	enableCmd := &cobra.Command{
		Use:   "enable <id>",
		Short: "Enable a module for this project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutateModule(cmd, loadApp, func(runtime *modules.Runtime) (modules.Report, error) {
				return runtime.Enable(cmd.Context(), args[0])
			})
		},
	}

	disableCmd := &cobra.Command{
		Use:   "disable <id>",
		Short: "Disable a module without deleting its config or grants",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mutateModule(cmd, loadApp, func(runtime *modules.Runtime) (modules.Report, error) {
				return runtime.Disable(cmd.Context(), args[0])
			})
		},
	}

	healthCmd := &cobra.Command{
		Use:   "health [id]",
		Short: "Check enabled module health",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			return withModuleRuntime(loadApp, func(runtime *modules.Runtime, printer modulePrinter) error {
				reports, err := runtime.Health(cmd.Context(), id)
				if err != nil {
					return err
				}
				return printer.Print(reports, func(w io.Writer) error {
					if len(reports) == 0 {
						_, err := fmt.Fprintln(w, "No enabled modules.")
						return err
					}
					return printModuleReports(w, reports)
				})
			})
		},
	}

	modulesCmd.AddCommand(listCmd, showCmd, grantCmd, revokeCmd, enableCmd, disableCmd, healthCmd)
	root.AddCommand(modulesCmd)
}

type modulePrinter interface {
	Print(data any, human func(io.Writer) error) error
}

func withModuleRuntime(loadApp appLoader, run func(*modules.Runtime, modulePrinter) error) error {
	appCtx, err := loadApp()
	if err != nil {
		return err
	}
	defer appCtx.Close()
	return run(appCtx.Modules, appCtx.Output)
}

func mutateModule(cmd *cobra.Command, loadApp appLoader, mutate func(*modules.Runtime) (modules.Report, error)) error {
	return withModuleRuntime(loadApp, func(runtime *modules.Runtime, printer modulePrinter) error {
		report, err := mutate(runtime)
		if err != nil {
			return err
		}
		return printer.Print(report, func(w io.Writer) error {
			return printModuleReports(w, []modules.Report{report})
		})
	})
}

func printModuleReports(w io.Writer, reports []modules.Report) error {
	for _, report := range reports {
		if _, err := fmt.Fprintf(w, "%s\t%s", report.ID, report.State); err != nil {
			return err
		}
		if report.Health != nil {
			if _, err := fmt.Fprintf(w, "\t%s", report.Health.Status); err != nil {
				return err
			}
		}
		if report.Failure != nil {
			if _, err := fmt.Fprintf(w, "\t%s", report.Failure.Code); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func printModuleDetails(w io.Writer, report modules.Report) error {
	lines := [][2]string{
		{"id", report.ID},
		{"name", report.Name},
		{"version", report.Version},
		{"state", string(report.State)},
		{"desired enabled", fmt.Sprintf("%t", report.DesiredEnabled)},
		{"config version", fmt.Sprintf("%d", report.ConfigVersion)},
		{"capabilities", strings.Join(report.Capabilities, ", ")},
		{"permissions", strings.Join(report.Permissions, ", ")},
		{"grants", strings.Join(report.Grants, ", ")},
	}
	if len(report.Commands) > 0 {
		lines = append(lines, [2]string{"commands", strings.Join(report.Commands, ", ")})
	}
	if len(report.Events) > 0 {
		lines = append(lines, [2]string{"events", strings.Join(report.Events, ", ")})
	}
	if report.Health != nil {
		lines = append(lines, [2]string{"health", string(report.Health.Status)})
	}
	if report.Failure != nil {
		lines = append(lines, [2]string{"failure", string(report.Failure.Code)}, [2]string{"message", report.Failure.Message})
		if report.Failure.Command != "" {
			lines = append(lines, [2]string{"command", report.Failure.Command})
		}
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "%s: %s\n", line[0], line[1]); err != nil {
			return err
		}
	}
	return nil
}
