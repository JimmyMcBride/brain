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

var rootFlags struct {
	configPath string
	jsonOutput bool
}

var rootCmd = &cobra.Command{
	Use:           "brain",
	Short:         "Local-first knowledge CLI for PARA-style markdown vaults",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&rootFlags.configPath, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVar(&rootFlags.jsonOutput, "json", false, "render output as JSON")
}

func loadApp() (*app.App, error) {
	return app.New(rootFlags.configPath, rootFlags.jsonOutput)
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

func readBody(flagValue string, fromStdin bool) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if !fromStdin {
		return "", nil
	}
	data, err := io.ReadAll(os.Stdin)
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

func modeFromFlag(configMode string) string {
	if rootFlags.jsonOutput {
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
		return "Resources"
	}
	return section
}

func chooseType(section, noteType string) string {
	if noteType != "" {
		return noteType
	}
	switch chooseSection(section) {
	case "Projects":
		return "project"
	case "Areas":
		return "area"
	case "Archives":
		return "archive"
	default:
		return "resource"
	}
}

func chooseTemplate(noteType, templateName string) string {
	if templateName != "" {
		return templateName
	}
	switch noteType {
	case "project":
		return "project.md"
	case "area":
		return "area.md"
	case "capture":
		return "capture.md"
	case "lesson":
		return "lesson.md"
	case "content_seed":
		return "content_seed.md"
	case "daily":
		return "daily.md"
	default:
		return "resource.md"
	}
}
