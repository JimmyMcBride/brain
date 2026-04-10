package skills

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type InstallMode string
type Scope string

const (
	ModeSymlink InstallMode = "symlink"
	ModeCopy    InstallMode = "copy"

	ScopeGlobal Scope = "global"
	ScopeLocal  Scope = "local"
	ScopeBoth   Scope = "both"
)

type Installer struct {
	Home string
}

type InstallRequest struct {
	Mode       InstallMode
	Scope      Scope
	Agents     []string
	ProjectDir string
	SkillRoots []string
	RepoRoot   string
}

type InstallResult struct {
	Agent  string `json:"agent"`
	Scope  string `json:"scope"`
	Root   string `json:"root"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

type Target struct {
	Agent string `json:"agent"`
	Scope string `json:"scope"`
	Root  string `json:"root"`
	Path  string `json:"path"`
}

func NewInstaller(home string) *Installer {
	if home == "" {
		if userHome, err := os.UserHomeDir(); err == nil {
			home = userHome
		}
	}
	return &Installer{Home: home}
}

func (i *Installer) Install(req InstallRequest) ([]InstallResult, error) {
	source := filepath.Join(req.RepoRoot, "skills", "brain")
	if err := validateSkillSource(source); err != nil {
		return nil, err
	}

	targets, err := i.ResolveTargets(req)
	if err != nil {
		return nil, err
	}
	results := make([]InstallResult, 0, len(targets))
	for _, target := range targets {
		if err := os.MkdirAll(target.Root, 0o755); err != nil {
			return nil, fmt.Errorf("create skill root %s: %w", target.Root, err)
		}
		mode := effectiveMode(req.Mode, target)
		if err := installPath(mode, source, target.Path); err != nil {
			return nil, err
		}
		results = append(results, InstallResult{
			Agent:  target.Agent,
			Scope:  target.Scope,
			Root:   target.Root,
			Path:   target.Path,
			Method: string(mode),
		})
	}
	return results, nil
}

func (i *Installer) ResolveTargets(req InstallRequest) ([]Target, error) {
	mode := req.Mode
	if mode == "" {
		mode = ModeSymlink
	}
	if mode != ModeSymlink && mode != ModeCopy {
		return nil, fmt.Errorf("unsupported install mode: %s", mode)
	}
	scope := req.Scope
	if scope == "" {
		scope = ScopeGlobal
	}
	if scope != ScopeGlobal && scope != ScopeLocal && scope != ScopeBoth {
		return nil, fmt.Errorf("unsupported scope: %s", scope)
	}

	agents := normalizeAgents(req.Agents)
	if len(agents) == 0 {
		agents = knownAgents()
	}

	var targets []Target
	for _, root := range req.SkillRoots {
		root = filepath.Clean(expandHome(root, i.Home))
		targets = append(targets, Target{
			Agent: agentNameFromRoot(root),
			Scope: "custom",
			Root:  root,
			Path:  filepath.Join(root, "brain"),
		})
	}

	if scope == ScopeLocal || scope == ScopeBoth {
		projectDir := req.ProjectDir
		if projectDir == "" {
			projectDir = "."
		}
		projectDir = filepath.Clean(expandHome(projectDir, i.Home))
		for _, agent := range agents {
			root := filepath.Join(projectDir, "."+agent, "skills")
			targets = append(targets, Target{
				Agent: agent,
				Scope: string(ScopeLocal),
				Root:  root,
				Path:  filepath.Join(root, "brain"),
			})
		}
	}

	if scope == ScopeGlobal || scope == ScopeBoth {
		for _, agent := range agents {
			root := knownGlobalSkillRoot(i.Home, agent)
			targets = append(targets, Target{
				Agent: agent,
				Scope: string(ScopeGlobal),
				Root:  root,
				Path:  filepath.Join(root, "brain"),
			})
		}
	}

	return dedupeTargets(targets), nil
}

func validateSkillSource(source string) error {
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("skill source %s: %w", source, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("skill source is not a directory: %s", source)
	}
	if _, err := os.Stat(filepath.Join(source, "SKILL.md")); err != nil {
		return fmt.Errorf("skill source missing SKILL.md: %w", err)
	}
	return nil
}

func effectiveMode(mode InstallMode, target Target) InstallMode {
	if mode == "" {
		mode = ModeSymlink
	}
	// OpenClaw's managed skill loader currently ignores symlinked skill
	// directories, so copy is the only discoverable install mode for it.
	if target.Agent == "openclaw" && target.Scope != "custom" {
		return ModeCopy
	}
	return mode
}

func installPath(mode InstallMode, source, target string) error {
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("clear target %s: %w", target, err)
	}
	switch mode {
	case ModeSymlink:
		return os.Symlink(source, target)
	case ModeCopy:
		return copyDir(source, target)
	default:
		return fmt.Errorf("unsupported install mode: %s", mode)
	}
}

func copyDir(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		if info.IsDir() {
			return os.MkdirAll(dest, info.Mode().Perm())
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return copyFile(path, dest)
	})
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy skill: %w", err)
	}
	return out.Close()
}

func knownAgents() []string {
	return []string{"codex", "claude", "openclaw", "pi", "ai"}
}

func knownGlobalSkillRoot(home, agent string) string {
	switch agent {
	case "codex":
		return filepath.Join(home, ".codex", "skills")
	case "claude":
		return filepath.Join(home, ".claude", "skills")
	case "openclaw":
		return filepath.Join(home, ".openclaw", "skills")
	case "pi":
		return filepath.Join(home, ".pi", "skills")
	case "ai":
		return filepath.Join(home, ".ai", "skills")
	default:
		return filepath.Join(home, "."+agent, "skills")
	}
}

func normalizeAgents(agents []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, agent := range agents {
		agent = strings.TrimSpace(strings.ToLower(agent))
		if agent == "" {
			continue
		}
		if _, ok := seen[agent]; ok {
			continue
		}
		seen[agent] = struct{}{}
		out = append(out, agent)
	}
	sort.Strings(out)
	return out
}

func dedupeTargets(targets []Target) []Target {
	seen := map[string]struct{}{}
	var out []Target
	for _, target := range targets {
		key := target.Path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Scope == out[j].Scope {
			if out[i].Agent == out[j].Agent {
				return out[i].Path < out[j].Path
			}
			return out[i].Agent < out[j].Agent
		}
		return out[i].Scope < out[j].Scope
	})
	return out
}

func expandHome(path, home string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/"))
}

func agentNameFromRoot(root string) string {
	base := strings.TrimPrefix(filepath.Base(root), ".")
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "custom"
	}
	if base == "skills" {
		return "custom"
	}
	if strings.HasSuffix(base, "-skills") {
		base = strings.TrimSuffix(base, "-skills")
	}
	if base == "" {
		return "custom"
	}
	return base
}
