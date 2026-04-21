package projectcontext

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

var localRuntimeIgnoreEntries = []string{
	".brain/session.json",
	".brain/sessions/",
	".brain/policy.override.yaml",
	".brain/state/",
}

var localRuntimeGitPaths = []string{
	".brain/session.json",
	".brain/sessions",
	".brain/policy.override.yaml",
	".brain/state",
}

func (m *Manager) ignoreLocalRuntimeState(ctx context.Context, projectDir string) (ProjectMigrationResult, error) {
	results, err := m.syncManagedContextForMigration(ctx, projectDir)
	if err != nil {
		return ProjectMigrationResult{}, err
	}

	result := ProjectMigrationResult{
		Action:  "unchanged",
		Results: results,
	}
	if migrationChanged(results) {
		result.Action = "updated"
	}

	insideGit, err := isGitWorkTree(ctx, projectDir)
	if err != nil {
		return result, err
	}
	if !insideGit {
		result.Messages = append(result.Messages,
			"Brain local runtime state is ignored by default.",
			"Skipped Git cleanup because this project is not inside a Git work tree.",
		)
		return result, nil
	}

	tracked, err := trackedRuntimePaths(ctx, projectDir)
	if err != nil {
		return result, err
	}
	if len(tracked) == 0 {
		result.Messages = append(result.Messages,
			"Brain local runtime state is ignored by default.",
			"No tracked Brain runtime files needed cleanup.",
		)
		return result, nil
	}
	if err := untrackRuntimePaths(ctx, projectDir); err != nil {
		return result, err
	}
	sort.Strings(tracked)
	result.Action = "updated"
	result.Messages = append(result.Messages,
		"Brain local runtime state is now ignored by default.",
		"Removed from Git tracking but kept on disk: "+strings.Join(tracked, ", "),
		"Review and commit the resulting .gitignore and index cleanup diff.",
	)
	return result, nil
}

func isGitWorkTree(ctx context.Context, projectDir string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		var execErr *exec.Error
		if errors.As(err, &exitErr) || errors.As(err, &execErr) {
			return false, nil
		}
		return false, fmt.Errorf("detect git work tree: %w", err)
	}
	return strings.TrimSpace(string(output)) == "true", nil
}

func trackedRuntimePaths(ctx context.Context, projectDir string) ([]string, error) {
	args := append([]string{"ls-files", "--cached", "--"}, localRuntimeGitPaths...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("list tracked Brain runtime paths: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	tracked := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		tracked = append(tracked, line)
	}
	return tracked, nil
}

func untrackRuntimePaths(ctx context.Context, projectDir string) error {
	args := append([]string{"rm", "-r", "--cached", "--ignore-unmatch", "--"}, localRuntimeGitPaths...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove Brain runtime paths from git index: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
