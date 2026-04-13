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

type appLoader func() (*app.App, error)

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
	cmd := &cobra.Command{
		Use:           "brain",
		Short:         "Project-local brain for docs, planning, context, and workflow",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetIn(opts.in)
	cmd.SetOut(opts.out)
	cmd.SetErr(opts.errOut)
	cmd.PersistentFlags().StringVar(&flags.configPath, "config", "", "config file path")
	cmd.PersistentFlags().StringVar(&flags.projectPath, "project", ".", "project root path")
	cmd.PersistentFlags().BoolVar(&flags.jsonOutput, "json", false, "render output as JSON")

	loadApp := func() (*app.App, error) {
		return opts.appLoad(flags.configPath, flags.projectPath, flags.jsonOutput, cmd.OutOrStdout(), cmd.ErrOrStderr())
	}

	addCommands(cmd, flags, loadApp)
	return cmd
}

func addCommands(root *cobra.Command, flags *rootFlagsState, loadApp appLoader) {
	addInitCommand(root, flags)
	addAdoptCommand(root, flags)
	addDoctorCommand(root, flags)
	addVersionCommand(root, flags)
	addUpdateCommand(root, flags)
	addReadCommand(root, flags, loadApp)
	addEditCommand(root, flags, loadApp)
	addFindCommand(root, flags, loadApp)
	addSearchCommand(root, flags, loadApp)
	addDistillCommand(root, flags, loadApp)
	addBrainstormCommand(root, flags, loadApp)
	addPlanCommand(root, flags, loadApp)
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

func repoRoot() string {
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
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
	case "brainstorm", "brainstorm_distill":
		return "brainstorm.md"
	case "decision":
		return "decision.md"
	case "epic":
		return "epic.md"
	case "spec":
		return "spec.md"
	case "story":
		return "story.md"
	default:
		return "resource.md"
	}
}
