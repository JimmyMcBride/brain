package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"brain/internal/app"

	"github.com/spf13/cobra"
)

type rootOptions struct {
	in      io.Reader
	out     io.Writer
	errOut  io.Writer
	appLoad func(configPath, projectPath string, jsonOutput bool, out io.Writer, errOut io.Writer) (*app.App, error)
}

type rootFlagsState struct {
	configPath  string
	projectPath string
	jsonOutput  bool
}

type appLoader func(projectPath ...string) (*app.App, error)

var rootCmd = newRootCommand(rootOptions{})

func Execute() error {
	return rootCmd.Execute()
}

func newRootCommand(opts rootOptions) *cobra.Command {
	if opts.in == nil {
		opts.in = os.Stdin
	}
	if opts.out == nil {
		opts.out = os.Stdout
	}
	if opts.errOut == nil {
		opts.errOut = os.Stderr
	}
	if opts.appLoad == nil {
		opts.appLoad = func(configPath, projectPath string, jsonOutput bool, out io.Writer, errOut io.Writer) (*app.App, error) {
			return app.New(configPath, projectPath, jsonOutput, app.Options{
				Stdout: out,
				Stderr: errOut,
			})
		}
	}

	flags := &rootFlagsState{}
	var enableProjectPreflight bool
	preflightDone := map[string]bool{}
	cmd := &cobra.Command{
		Use:           "brain",
		Short:         "Project-local memory, retrieval, context, and workflow for AI agents",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			enableProjectPreflight = shouldPreflightProjectRepairs(cmd)
			return nil
		},
	}
	cmd.SetIn(opts.in)
	cmd.SetOut(opts.out)
	cmd.SetErr(opts.errOut)
	cmd.PersistentFlags().StringVar(&flags.configPath, "config", "", "config file path")
	cmd.PersistentFlags().StringVar(&flags.projectPath, "project", ".", "project root path")
	cmd.PersistentFlags().BoolVar(&flags.jsonOutput, "json", false, "render output as JSON")

	loadApp := func(projectPath ...string) (*app.App, error) {
		resolvedProject := flags.projectPath
		if len(projectPath) != 0 && strings.TrimSpace(projectPath[0]) != "" {
			resolvedProject = projectPath[0]
		}
		if enableProjectPreflight {
			absProject, err := filepath.Abs(resolvedProject)
			if err != nil {
				return nil, err
			}
			if !preflightDone[absProject] {
				if err := repairLocalSkillsIfNeeded(absProject); err != nil {
					return nil, err
				}
				if err := applyProjectMigrationsIfNeeded(absProject); err != nil {
					return nil, err
				}
				preflightDone[absProject] = true
			}
			resolvedProject = absProject
		}
		return opts.appLoad(flags.configPath, resolvedProject, flags.jsonOutput, cmd.OutOrStdout(), cmd.ErrOrStderr())
	}

	addCommands(cmd, flags, loadApp)
	return cmd
}

func addCommands(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	addInitCommand(root, flags)
	addAdoptCommand(root, flags)
	addDoctorCommand(root, flags)
	addVersionCommand(root, flags)
	addUpdateCommand(root, flags, loadApp)
	addReadCommand(root, flags, loadApp)
	addEditCommand(root, flags, loadApp)
	addFindCommand(root, flags, loadApp)
	addSearchCommand(root, flags, loadApp)
	addPrepCommand(root, flags, loadApp)
	addDistillCommand(root, flags, loadApp)
	addHistoryCommand(root, flags, loadApp)
	addUndoCommand(root, flags, loadApp)
	addContextCommand(root, flags, loadApp)
	addSessionCommand(root, flags, loadApp)
	addSkillsCommand(root, flags, loadApp)
}

func parseMeta(entries []string) (map[string]any, error) {
	meta := map[string]any{}
	for _, entry := range entries {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid metadata assignment %q, expected key=value", entry)
		}
		meta[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return meta, nil
}

func readBody(in io.Reader, flagValue string, fromStdin bool) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if !fromStdin {
		return "", nil
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func modeFromFlag(flags *rootFlagsState, configMode string) string {
	if flags != nil && flags.jsonOutput {
		return "json"
	}
	if configMode == "" {
		return "human"
	}
	return configMode
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

func chooseSection(section string) string {
	if section == "" {
		return ".brain"
	}
	return section
}

func chooseType(section, noteType string) string {
	if noteType != "" {
		return noteType
	}
	return "resource"
}

func chooseTemplate(noteType, templateName string) string {
	if templateName != "" {
		return templateName
	}
	switch noteType {
	case "decision":
		return "decision.md"
	default:
		return "resource.md"
	}
}

func shouldPreflightProjectRepairs(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	name := topLevelCommandName(cmd)
	switch name {
	case "", "help", "init", "adopt", "doctor", "version", "update", "skills":
		return false
	default:
		return true
	}
}

func topLevelCommandName(cmd *cobra.Command) string {
	current := cmd
	for current != nil && current.Parent() != nil && current.Parent().Parent() != nil {
		current = current.Parent()
	}
	if current == nil {
		return ""
	}
	return current.Name()
}
