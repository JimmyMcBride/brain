package livecontext

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"brain/internal/projectcontext"
	"brain/internal/session"
)

type VerificationRecipe struct {
	Label    string `json:"label"`
	Command  string `json:"command"`
	Source   string `json:"source"`
	Strength string `json:"strength"`
	Reason   string `json:"reason"`
}

type verificationRecipeCandidate struct {
	recipe   VerificationRecipe
	priority int
}

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

var (
	makeTargetPattern  = regexp.MustCompile(`^([A-Za-z0-9_.-]+):(?:\s|$)`)
	workflowRunPattern = regexp.MustCompile(`^\s*-?\s*run:\s*(.+?)\s*$`)
)

func collectVerificationRecipes(projectDir string, policy *projectcontext.Policy, active *session.ActiveSession, profiles []VerificationProfile) []VerificationRecipe {
	candidates := map[string]verificationRecipeCandidate{}
	add := func(recipe VerificationRecipe, priority int) {
		recipe.Command = strings.TrimSpace(recipe.Command)
		if recipe.Command == "" || !looksLikeVerificationCommand(recipe.Command) {
			return
		}
		existing, ok := candidates[recipe.Command]
		if ok && existing.priority > priority {
			return
		}
		if ok && existing.priority == priority && existing.recipe.Strength == "strong" && recipe.Strength != "strong" {
			return
		}
		candidates[recipe.Command] = verificationRecipeCandidate{recipe: recipe, priority: priority}
	}

	matchedByProfile := map[string]string{}
	for _, profile := range profiles {
		if profile.MatchedCommand != "" {
			matchedByProfile[profile.Name] = profile.MatchedCommand
		}
	}
	if policy != nil {
		for _, profile := range policy.Closeout.VerificationProfiles {
			for _, command := range profile.Commands {
				reason := fmt.Sprintf("required by verification profile %q", profile.Name)
				if matchedByProfile[profile.Name] == command {
					reason += " and already satisfied in this session"
				}
				add(VerificationRecipe{
					Label:    profile.Name,
					Command:  command,
					Source:   ".brain/policy.yaml",
					Strength: "strong",
					Reason:   reason,
				}, 100)
			}
		}
	}

	for _, target := range collectMakeTargets(projectDir) {
		add(VerificationRecipe{
			Label:    "Make target: " + target,
			Command:  "make " + target,
			Source:   "Makefile",
			Strength: verificationStrengthForName(target),
			Reason:   fmt.Sprintf("repo Makefile exposes the %q verification target", target),
		}, 80)
	}

	for name := range collectPackageScripts(projectDir) {
		command := packageScriptCommand(projectDir, name)
		add(VerificationRecipe{
			Label:    "Package script: " + name,
			Command:  command,
			Source:   "package.json",
			Strength: verificationStrengthForName(name),
			Reason:   fmt.Sprintf("package.json defines the %q script", name),
		}, 75)
	}

	for _, workflow := range collectWorkflowCommands(projectDir) {
		add(VerificationRecipe{
			Label:    "CI command",
			Command:  workflow.Command,
			Source:   workflow.Source,
			Strength: "strong",
			Reason:   "CI workflow runs this verification command",
		}, 90)
	}

	if active != nil {
		for i := len(active.CommandRuns) - 1; i >= 0; i-- {
			run := active.CommandRuns[i]
			if run.ExitCode != 0 || !looksLikeVerificationCommand(run.Command) {
				continue
			}
			add(VerificationRecipe{
				Label:    "Recent successful command",
				Command:  run.Command,
				Source:   "session_history",
				Strength: "suggested",
				Reason:   "recent successful session command",
			}, 60)
		}
	}

	out := make([]VerificationRecipe, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, candidate.recipe)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Strength == out[j].Strength {
			if out[i].Source == out[j].Source {
				return out[i].Command < out[j].Command
			}
			return out[i].Source < out[j].Source
		}
		return out[i].Strength == "strong"
	})
	return out
}

func collectMakeTargets(projectDir string) []string {
	for _, name := range []string{"Makefile", "makefile"} {
		body, err := os.ReadFile(filepath.Join(projectDir, name))
		if err != nil {
			continue
		}
		targets := []string{}
		for _, line := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
			match := makeTargetPattern.FindStringSubmatch(strings.TrimSpace(line))
			if len(match) != 2 {
				continue
			}
			target := strings.TrimSpace(match[1])
			if target == "" || strings.HasPrefix(target, ".") || !looksLikeVerificationName(target) {
				continue
			}
			targets = append(targets, target)
		}
		sort.Strings(targets)
		return dedupeStrings(targets)
	}
	return []string{}
}

func collectPackageScripts(projectDir string) map[string]string {
	body, err := os.ReadFile(filepath.Join(projectDir, "package.json"))
	if err != nil {
		return map[string]string{}
	}
	var parsed packageJSON
	if err := json.Unmarshal(body, &parsed); err != nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for name, command := range parsed.Scripts {
		name = strings.TrimSpace(name)
		if name == "" || !looksLikeVerificationName(name) {
			continue
		}
		out[name] = strings.TrimSpace(command)
	}
	return out
}

func packageScriptCommand(projectDir, name string) string {
	switch detectNodeRunner(projectDir) {
	case "pnpm":
		return "pnpm " + name
	case "yarn":
		return "yarn " + name
	case "bun":
		return "bun run " + name
	default:
		if name == "test" {
			return "npm test"
		}
		return "npm run " + name
	}
}

func detectNodeRunner(projectDir string) string {
	switch {
	case fileExists(filepath.Join(projectDir, "pnpm-lock.yaml")):
		return "pnpm"
	case fileExists(filepath.Join(projectDir, "yarn.lock")):
		return "yarn"
	case fileExists(filepath.Join(projectDir, "bun.lockb")) || fileExists(filepath.Join(projectDir, "bun.lock")):
		return "bun"
	default:
		return "npm"
	}
}

type workflowCommand struct {
	Command string
	Source  string
}

func collectWorkflowCommands(projectDir string) []workflowCommand {
	matches, _ := filepath.Glob(filepath.Join(projectDir, ".github", "workflows", "*.y*ml"))
	out := []workflowCommand{}
	for _, match := range matches {
		body, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(projectDir, match)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
			match := workflowRunPattern.FindStringSubmatch(strings.TrimSpace(line))
			if len(match) != 2 {
				continue
			}
			candidate := strings.TrimSpace(strings.Trim(match[1], `"'`))
			if candidate == "" || !looksLikeVerificationCommand(candidate) {
				continue
			}
			out = append(out, workflowCommand{
				Command: candidate,
				Source:  filepath.ToSlash(rel),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source == out[j].Source {
			return out[i].Command < out[j].Command
		}
		return out[i].Source < out[j].Source
	})
	return out
}

func looksLikeVerificationName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	for _, marker := range []string{"test", "build", "check", "verify", "lint", "typecheck", "ci"} {
		if name == marker || strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func looksLikeVerificationCommand(command string) bool {
	command = strings.ToLower(strings.TrimSpace(command))
	if command == "" {
		return false
	}
	for _, marker := range []string{
		"go test", "go build", "make test", "make build", "make check", "make verify",
		"npm test", "npm run test", "npm run build", "npm run check", "npm run verify",
		"pnpm test", "pnpm build", "pnpm check", "pnpm verify",
		"yarn test", "yarn build", "yarn check", "yarn verify",
		"bun test", "bun run test", "bun run build",
		"cargo test", "cargo build", "pytest", "python -m pytest", "just test", "just build",
		"gradle test", "mvn test", "swift test",
	} {
		if strings.Contains(command, marker) {
			return true
		}
	}
	return false
}

func verificationStrengthForName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(name, "test"), strings.Contains(name, "build"), strings.Contains(name, "check"), strings.Contains(name, "verify"), name == "ci":
		return "strong"
	default:
		return "suggested"
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
