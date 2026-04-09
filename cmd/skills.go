package cmd

import (
	"fmt"
	"io"
	"strings"

	"brain/internal/skills"

	"github.com/spf13/cobra"
)

func init() {
	var mode string
	var scope string
	var agents []string
	var project string
	var skillRoots []string

	skillsCmd := &cobra.Command{
		Use:   "skills",
		Short: "Install the Brain skill bundle into global or project-local agent skill roots",
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install the Brain skill for known or custom AI agent skill directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			results, err := appCtx.Skills.Install(skills.InstallRequest{
				Mode:       skills.InstallMode(mode),
				Scope:      skills.Scope(scope),
				Agents:     agents,
				ProjectDir: project,
				SkillRoots: skillRoots,
				RepoRoot:   repoRoot(),
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(results, func(w io.Writer) error {
				for _, result := range results {
					if _, err := fmt.Fprintf(w, "%s [%s] %s -> %s\n", result.Agent, result.Scope, result.Method, result.Path); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	targetsCmd := &cobra.Command{
		Use:   "targets",
		Short: "Show resolved skill install targets without writing anything",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx, err := loadApp()
			if err != nil {
				return err
			}
			defer appCtx.Close()
			targets, err := appCtx.Skills.ResolveTargets(skills.InstallRequest{
				Mode:       skills.InstallMode(mode),
				Scope:      skills.Scope(scope),
				Agents:     agents,
				ProjectDir: project,
				SkillRoots: skillRoots,
				RepoRoot:   repoRoot(),
			})
			if err != nil {
				return err
			}
			return appCtx.Output.Print(targets, func(w io.Writer) error {
				for _, target := range targets {
					if _, err := fmt.Fprintf(w, "%s [%s] %s\n", target.Agent, target.Scope, target.Path); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}

	installCmd.Flags().StringVar(&mode, "mode", string(skills.ModeSymlink), "install mode: symlink or copy")
	installCmd.Flags().StringVar(&scope, "scope", string(skills.ScopeGlobal), "install scope: global, local, or both")
	installCmd.Flags().StringArrayVarP(&agents, "agent", "a", nil, "target agent name; repeatable, defaults to known agents")
	installCmd.Flags().StringVar(&project, "project", ".", "project root used for local installs")
	installCmd.Flags().StringArrayVar(&skillRoots, "skill-root", nil, "custom skill root directory; repeatable")

	targetsCmd.Flags().StringVar(&mode, "mode", string(skills.ModeSymlink), "install mode validation: symlink or copy")
	targetsCmd.Flags().StringVar(&scope, "scope", string(skills.ScopeGlobal), "target scope: global, local, or both")
	targetsCmd.Flags().StringArrayVarP(&agents, "agent", "a", nil, "target agent name; repeatable, defaults to known agents")
	targetsCmd.Flags().StringVar(&project, "project", ".", "project root used for local targets")
	targetsCmd.Flags().StringArrayVar(&skillRoots, "skill-root", nil, "custom skill root directory; repeatable")

	skillsCmd.Long = strings.TrimSpace(`
Install the Brain skill bundle into AI agent skill directories.

Known agents use the conventional roots:
  global: ~/.<agent>/skills
  local:  <project>/.<agent>/skills

You can also target nonstandard tools directly with --skill-root.
`)

	skillsCmd.AddCommand(installCmd, targetsCmd)
	rootCmd.AddCommand(skillsCmd)
}
