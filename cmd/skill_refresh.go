package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"brain/internal/projectcontext"
	"brain/internal/skills"
)

type skillRefreshResult struct {
	SkillRefreshStatus string                 `json:"skill_refresh_status,omitempty"`
	RefreshedSkills    []skills.InstallResult `json:"refreshed_skills,omitempty"`
}

var skillInstallRunner = runSkillInstall
var projectMigrationRunner = runProjectMigration

func inspectInstalledSkills(projectPath string) ([]skills.TargetStatus, []skills.TargetStatus, error) {
	installer := skills.NewInstaller(userHomeDir())

	global, err := installer.Inspect(skills.InstallRequest{
		Scope: skills.ScopeGlobal,
	})
	if err != nil {
		return nil, nil, err
	}

	local, err := installer.Inspect(skills.InstallRequest{
		Scope:      skills.ScopeLocal,
		ProjectDir: projectPath,
	})
	if err != nil {
		return nil, nil, err
	}

	return skills.InstalledTargets(global), skills.InstalledTargets(local), nil
}

func refreshInstalledSkills(binaryPath, configPath, projectPath string, global, local []skills.TargetStatus) (skillRefreshResult, error) {
	if len(global) == 0 && len(local) == 0 {
		return skillRefreshResult{SkillRefreshStatus: "not_needed"}, nil
	}

	var refreshed []skills.InstallResult
	if len(global) != 0 {
		results, err := skillInstallRunner(binaryPath, configPath, "", skills.ScopeGlobal, skills.AgentsForTargets(global))
		if err != nil {
			return skillRefreshResult{SkillRefreshStatus: "failed", RefreshedSkills: refreshed}, err
		}
		refreshed = append(refreshed, results...)
	}
	if len(local) != 0 {
		results, err := skillInstallRunner(binaryPath, configPath, projectPath, skills.ScopeLocal, skills.AgentsForTargets(local))
		if err != nil {
			return skillRefreshResult{SkillRefreshStatus: "failed", RefreshedSkills: refreshed}, err
		}
		refreshed = append(refreshed, results...)
	}

	return skillRefreshResult{
		SkillRefreshStatus: "refreshed",
		RefreshedSkills:    refreshed,
	}, nil
}

func repairLocalSkillsIfNeeded(projectPath string) error {
	installer := skills.NewInstaller(userHomeDir())
	statuses, err := installer.Inspect(skills.InstallRequest{
		Scope:      skills.ScopeLocal,
		ProjectDir: projectPath,
	})
	if err != nil {
		return err
	}

	repairs := skills.RepairTargets(statuses)
	if len(repairs) == 0 {
		return nil
	}
	if _, err := installer.Install(skills.InstallRequest{
		Scope:      skills.ScopeLocal,
		ProjectDir: projectPath,
		Agents:     skills.AgentsForTargets(repairs),
	}); err != nil {
		return fmt.Errorf("repair local brain skills for %s: %w", projectPath, err)
	}
	return nil
}

func applyProjectMigrationsIfNeeded(projectPath string) error {
	manager := contextManager()
	plan, err := manager.PlanProjectMigrations(projectPath)
	if err != nil {
		return err
	}
	if !plan.UsesBrain || len(plan.PendingMigrations) == 0 {
		return nil
	}
	if _, err := manager.ApplyProjectMigrations(context.Background(), projectPath); err != nil {
		return fmt.Errorf("apply project migrations for %s: %w", projectPath, err)
	}
	return nil
}

func runSkillInstall(binaryPath, configPath, projectPath string, scope skills.Scope, agents []string) ([]skills.InstallResult, error) {
	args := []string{}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	args = append(args, "--json", "skills", "install", "--scope", string(scope))
	for _, agent := range agents {
		args = append(args, "--agent", agent)
	}
	if scope == skills.ScopeLocal {
		args = append(args, "--project", projectPath)
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %v: %w\n%s", filepath.Base(binaryPath), args, err, string(output))
	}

	var results []skills.InstallResult
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("parse skill refresh output: %w", err)
	}
	return results, nil
}

func runProjectMigration(binaryPath, configPath, projectPath string) (*projectcontext.ApplyProjectMigrationsResult, error) {
	args := []string{}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	args = append(args, "--json", "--project", projectPath, "context", "migrate")

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %v: %w\n%s", filepath.Base(binaryPath), args, err, string(output))
	}

	var result projectcontext.ApplyProjectMigrationsResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse project migration output: %w", err)
	}
	return &result, nil
}
