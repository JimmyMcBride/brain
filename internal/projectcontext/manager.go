package projectcontext

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"brain/internal/skills"
)

const localNotesSection = "## Local Notes\n\nAdd repo-specific notes here. `brain context refresh` preserves content outside managed blocks.\n"

type Manager struct {
	Home string
}

type Request struct {
	ProjectDir string
	Agents     []string
	DryRun     bool
	Force      bool
	Adopt      bool
}

type Result struct {
	Path                 string   `json:"path"`
	Kind                 string   `json:"kind"`
	Action               string   `json:"action"`
	PreservedUserContent bool     `json:"preserved_user_content"`
	ManagedBlocks        []string `json:"managed_blocks"`
}

type Snapshot struct {
	ProjectDir      string
	ProjectName     string
	CurrentBranch   string
	DefaultBranch   string
	RemoteURL       string
	Dirty           bool
	ManifestFiles   []string
	DocFiles        []string
	CIFiles         []string
	RootDirectories []string
	InternalDirs    []string
	TestFiles       int
	PrimaryRuntime  string
	GoModule        string
	HasGit          bool
}

type fileSpec struct {
	Path          string
	Kind          string
	Title         string
	BlockID       string
	Body          string
	Style         string
	LocalNote     bool
	CommentPrefix string
}

func New(home string) *Manager {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return &Manager{Home: home}
}

func (m *Manager) Install(ctx context.Context, req Request) ([]Result, error) {
	return m.apply(ctx, req)
}

func (m *Manager) Adopt(ctx context.Context, req Request) ([]Result, error) {
	req.Force = true
	req.Adopt = true
	return m.apply(ctx, req)
}

func (m *Manager) Refresh(ctx context.Context, req Request) ([]Result, error) {
	return m.apply(ctx, req)
}

func (m *Manager) apply(ctx context.Context, req Request) ([]Result, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(req.ProjectDir))
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(projectDir)
	if err != nil {
		return nil, fmt.Errorf("project dir %s: %w", projectDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project dir is not a directory: %s", projectDir)
	}

	snapshot := scanRepo(ctx, projectDir)
	policyBody, err := RenderPolicy(snapshot)
	if err != nil {
		return nil, err
	}

	specs := bundleSpecs(snapshot, m.resolveAgents(req.Agents), policyBody)
	results := make([]Result, 0, len(specs))
	for _, spec := range specs {
		result, err := syncSpec(spec, req.DryRun, req.Force, req.Adopt)
		if err != nil {
			return nil, err
		}
		if rel, relErr := filepath.Rel(projectDir, spec.Path); relErr == nil {
			result.Path = filepath.ToSlash(rel)
		}
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})
	return results, nil
}

func (m *Manager) resolveAgents(explicit []string) []string {
	if len(explicit) != 0 {
		return normalizeAgents(explicit)
	}
	var agents []string
	for _, agent := range skills.KnownAgents() {
		root := skills.GlobalSkillRoot(m.Home, agent)
		if _, err := os.Stat(filepath.Join(root, "brain", "SKILL.md")); err == nil {
			agents = append(agents, agent)
		}
	}
	return agents
}

func wrapperFile(projectDir, agent string) string {
	switch agent {
	case "claude":
		return filepath.Join(projectDir, ".claude", "CLAUDE.md")
	default:
		return filepath.Join(projectDir, "."+agent, "AGENTS.md")
	}
}

func bundleSpecs(snapshot Snapshot, agents []string, policyBody string) []fileSpec {
	specs := []fileSpec{
		{
			Path:      filepath.Join(snapshot.ProjectDir, "AGENTS.md"),
			Kind:      "contract",
			Title:     "Project Agent Contract",
			BlockID:   "agents-contract",
			Body:      renderAgents(snapshot),
			Style:     "markdown",
			LocalNote: true,
		},
		{
			Path:      filepath.Join(snapshot.ProjectDir, "docs", "project-overview.md"),
			Kind:      "doc",
			Title:     "Project Overview",
			BlockID:   "project-doc-overview",
			Body:      renderOverview(snapshot),
			Style:     "markdown",
			LocalNote: true,
		},
		{
			Path:      filepath.Join(snapshot.ProjectDir, "docs", "project-architecture.md"),
			Kind:      "doc",
			Title:     "Project Architecture",
			BlockID:   "project-doc-architecture",
			Body:      renderArchitecture(snapshot),
			Style:     "markdown",
			LocalNote: true,
		},
		{
			Path:      filepath.Join(snapshot.ProjectDir, "docs", "project-workflows.md"),
			Kind:      "doc",
			Title:     "Project Workflows",
			BlockID:   "project-doc-workflows",
			Body:      renderWorkflows(snapshot),
			Style:     "markdown",
			LocalNote: true,
		},
		{
			Path:      filepath.Join(snapshot.ProjectDir, ".brain", "context", "overview.md"),
			Kind:      "context",
			Title:     "Overview",
			BlockID:   "context-overview",
			Body:      renderOverview(snapshot),
			Style:     "markdown",
			LocalNote: true,
		},
		{
			Path:      filepath.Join(snapshot.ProjectDir, ".brain", "context", "architecture.md"),
			Kind:      "context",
			Title:     "Architecture",
			BlockID:   "context-architecture",
			Body:      renderArchitecture(snapshot),
			Style:     "markdown",
			LocalNote: true,
		},
		{
			Path:      filepath.Join(snapshot.ProjectDir, ".brain", "context", "standards.md"),
			Kind:      "context",
			Title:     "Standards",
			BlockID:   "context-standards",
			Body:      renderStandards(snapshot),
			Style:     "markdown",
			LocalNote: true,
		},
		{
			Path:      filepath.Join(snapshot.ProjectDir, ".brain", "context", "workflows.md"),
			Kind:      "context",
			Title:     "Workflows",
			BlockID:   "context-workflows",
			Body:      renderWorkflows(snapshot),
			Style:     "markdown",
			LocalNote: true,
		},
		{
			Path:      filepath.Join(snapshot.ProjectDir, ".brain", "context", "memory-policy.md"),
			Kind:      "context",
			Title:     "Memory Policy",
			BlockID:   "context-memory-policy",
			Body:      renderMemoryPolicy(snapshot),
			Style:     "markdown",
			LocalNote: true,
		},
		{
			Path:      filepath.Join(snapshot.ProjectDir, ".brain", "context", "current-state.md"),
			Kind:      "context",
			Title:     "Current State",
			BlockID:   "context-current-state",
			Body:      renderCurrentState(snapshot),
			Style:     "markdown",
			LocalNote: true,
		},
		{
			Path:  filepath.Join(snapshot.ProjectDir, ".brain", "policy.yaml"),
			Kind:  "policy",
			Body:  policyBody,
			Style: "wholefile",
		},
		{
			Path:          filepath.Join(snapshot.ProjectDir, ".gitignore"),
			Kind:          "ignore",
			BlockID:       "gitignore-session",
			Body:          renderGitIgnore(),
			Style:         "textblock",
			CommentPrefix: "# ",
		},
	}
	for _, agent := range agents {
		specs = append(specs, fileSpec{
			Path:      wrapperFile(snapshot.ProjectDir, agent),
			Kind:      "wrapper",
			Title:     strings.ToUpper(agent[:1]) + agent[1:] + " Wrapper",
			BlockID:   "agent-wrapper-" + agent,
			Body:      renderWrapper(agent),
			Style:     "markdown",
			LocalNote: true,
		})
	}
	return specs
}

func syncSpec(spec fileSpec, dryRun, force, adopt bool) (Result, error) {
	switch spec.Style {
	case "markdown":
		return syncMarkdownDoc(spec, dryRun, force, adopt)
	case "textblock":
		return syncTextBlock(spec, dryRun)
	case "wholefile":
		return syncWholeFile(spec, dryRun)
	default:
		return Result{}, fmt.Errorf("unsupported context file style %q", spec.Style)
	}
}

func syncWholeFile(spec fileSpec, dryRun bool) (Result, error) {
	existing, err := os.ReadFile(spec.Path)
	if err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}
	action := "created"
	if err == nil {
		action = "updated"
		if string(existing) == spec.Body {
			action = "unchanged"
		}
	}
	if !dryRun && action != "unchanged" {
		if err := os.MkdirAll(filepath.Dir(spec.Path), 0o755); err != nil {
			return Result{}, err
		}
		if err := os.WriteFile(spec.Path, []byte(spec.Body), 0o644); err != nil {
			return Result{}, err
		}
	}
	return Result{
		Path:          filepath.ToSlash(spec.Path),
		Kind:          spec.Kind,
		Action:        action,
		ManagedBlocks: nil,
	}, nil
}

func syncTextBlock(spec fileSpec, dryRun bool) (Result, error) {
	existing, err := os.ReadFile(spec.Path)
	if err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}
	merged, preserved, action, err := mergeTextBlock(string(existing), spec, os.IsNotExist(err))
	if err != nil {
		return Result{}, err
	}
	if !dryRun && action != "unchanged" {
		if err := os.MkdirAll(filepath.Dir(spec.Path), 0o755); err != nil {
			return Result{}, err
		}
		if err := os.WriteFile(spec.Path, []byte(merged), 0o644); err != nil {
			return Result{}, err
		}
	}
	return Result{
		Path:                 filepath.ToSlash(spec.Path),
		Kind:                 spec.Kind,
		Action:               action,
		PreservedUserContent: preserved,
		ManagedBlocks:        []string{spec.BlockID},
	}, nil
}

func syncMarkdownDoc(spec fileSpec, dryRun, force, adopt bool) (Result, error) {
	existing, err := os.ReadFile(spec.Path)
	if err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}
	content, preserved, action, err := mergeDocument(string(existing), spec, force, adopt, os.IsNotExist(err))
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", filepath.ToSlash(spec.Path), err)
	}
	if !dryRun && action != "unchanged" {
		if err := os.MkdirAll(filepath.Dir(spec.Path), 0o755); err != nil {
			return Result{}, err
		}
		if err := os.WriteFile(spec.Path, []byte(content), 0o644); err != nil {
			return Result{}, err
		}
	}
	return Result{
		Path:                 filepath.ToSlash(spec.Path),
		Kind:                 spec.Kind,
		Action:               action,
		PreservedUserContent: preserved,
		ManagedBlocks:        []string{spec.BlockID},
	}, nil
}

func mergeDocument(existing string, spec fileSpec, force, adopt, missing bool) (string, bool, string, error) {
	begin := managedBegin(spec.BlockID)
	end := managedEnd(spec.BlockID)
	managed := managedBody(spec)
	if missing || strings.TrimSpace(existing) == "" {
		return managed, false, "created", nil
	}

	if strings.Contains(existing, begin) && strings.Contains(existing, end) {
		start := strings.Index(existing, begin)
		finish := strings.Index(existing[start:], end)
		if finish < 0 {
			return "", false, "", fmt.Errorf("missing managed block end marker")
		}
		finish += start + len(end)
		if finish < len(existing) && existing[finish] == '\n' {
			finish++
		}
		replaced := existing[:start] + managedSection(spec) + existing[finish:]
		action := "updated"
		if replaced == existing {
			action = "unchanged"
		}
		return replaced, hasUserContent(existing, spec.BlockID), action, nil
	}

	if !force {
		return "", false, "", fmt.Errorf("existing file is not brain-managed; rerun with --force to adopt it")
	}

	trimmed := strings.TrimSpace(existing)
	adopted := managed
	if trimmed != "" {
		adopted = strings.TrimRight(adopted, "\n") + "\n\n## Local Notes\n\n" + trimmed + "\n"
	}
	action := "updated"
	if adopt {
		action = "adopted"
	}
	return adopted, trimmed != "", action, nil
}

func mergeTextBlock(existing string, spec fileSpec, missing bool) (string, bool, string, error) {
	begin := textBlockBegin(spec)
	end := textBlockEnd(spec)
	block := textBlock(spec)
	if missing || strings.TrimSpace(existing) == "" {
		return block, false, "created", nil
	}
	if strings.Contains(existing, begin) && strings.Contains(existing, end) {
		start := strings.Index(existing, begin)
		finish := strings.Index(existing[start:], end)
		if finish < 0 {
			return "", false, "", fmt.Errorf("missing managed text block end marker")
		}
		finish += start + len(end)
		if finish < len(existing) && existing[finish] == '\n' {
			finish++
		}
		replaced := existing[:start] + block + existing[finish:]
		action := "updated"
		if replaced == existing {
			action = "unchanged"
		}
		return replaced, normalizeOutsideText(existing, begin, end) != "", action, nil
	}
	content := strings.TrimRight(existing, "\n")
	if content != "" {
		content += "\n\n"
	}
	content += block
	return content, strings.TrimSpace(existing) != "", "updated", nil
}

func managedBody(spec fileSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", spec.Title)
	b.WriteString(managedSection(spec))
	if spec.LocalNote {
		b.WriteString("\n")
		b.WriteString(localNotesSection)
	}
	return b.String()
}

func managedSection(spec fileSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", managedBegin(spec.BlockID))
	b.WriteString(strings.TrimSpace(spec.Body))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%s\n", managedEnd(spec.BlockID))
	return b.String()
}

func textBlock(spec fileSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", textBlockBegin(spec))
	b.WriteString(strings.TrimRight(spec.Body, "\n"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%s\n", textBlockEnd(spec))
	return b.String()
}

func managedBegin(id string) string {
	return "<!-- brain:begin " + id + " -->"
}

func managedEnd(id string) string {
	return "<!-- brain:end " + id + " -->"
}

func textBlockBegin(spec fileSpec) string {
	return spec.CommentPrefix + "brain:begin " + spec.BlockID
}

func textBlockEnd(spec fileSpec) string {
	return spec.CommentPrefix + "brain:end " + spec.BlockID
}

func hasUserContent(existing, blockID string) bool {
	begin := managedBegin(blockID)
	end := managedEnd(blockID)
	start := strings.Index(existing, begin)
	finish := strings.Index(existing, end)
	if start < 0 || finish < 0 {
		return strings.TrimSpace(existing) != ""
	}
	finish += len(end)
	prefix := normalizeOutsideContent(existing[:start])
	suffix := normalizeOutsideContent(existing[finish:])
	return prefix != "" || suffix != ""
}

func normalizeOutsideContent(s string) string {
	s = strings.TrimSpace(s)
	for _, title := range []string{
		"# Project Agent Contract",
		"# Overview",
		"# Architecture",
		"# Standards",
		"# Workflows",
		"# Memory Policy",
		"# Current State",
		"# Claude Wrapper",
		"# Codex Wrapper",
		"# Openclaw Wrapper",
		"# Pi Wrapper",
		"# Ai Wrapper",
	} {
		s = strings.TrimSpace(strings.ReplaceAll(s, title, ""))
	}
	s = strings.TrimSpace(strings.ReplaceAll(s, strings.TrimSpace(localNotesSection), ""))
	return s
}

func normalizeOutsideText(s, begin, end string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, begin, ""))
	s = strings.TrimSpace(strings.ReplaceAll(s, end, ""))
	return s
}

func defaultProjectDir(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return "."
	}
	return dir
}

func normalizeAgents(agents []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(agents))
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

func scanRepo(ctx context.Context, projectDir string) Snapshot {
	snapshot := Snapshot{
		ProjectDir:  projectDir,
		ProjectName: filepath.Base(projectDir),
	}
	snapshot.ManifestFiles = existingFiles(projectDir, "go.mod", "package.json", "Cargo.toml", "pyproject.toml", "Makefile")
	snapshot.DocFiles = discoverDocs(projectDir)
	snapshot.CIFiles = discoverCIFiles(projectDir)
	snapshot.RootDirectories = discoverRootDirectories(projectDir)
	snapshot.InternalDirs = discoverSubdirectories(filepath.Join(projectDir, "internal"))
	snapshot.TestFiles = countTestFiles(projectDir)
	snapshot.PrimaryRuntime = primaryRuntime(snapshot.ManifestFiles, projectDir)
	snapshot.GoModule = readGoModule(projectDir)
	if gitAvailable(ctx, projectDir) {
		snapshot.HasGit = true
		snapshot.CurrentBranch = strings.TrimSpace(runGit(ctx, projectDir, "branch", "--show-current"))
		snapshot.RemoteURL = strings.TrimSpace(runGit(ctx, projectDir, "config", "--get", "remote.origin.url"))
		head := strings.TrimSpace(runGit(ctx, projectDir, "symbolic-ref", "refs/remotes/origin/HEAD", "--short"))
		snapshot.DefaultBranch = strings.TrimPrefix(head, "origin/")
		snapshot.Dirty = strings.TrimSpace(runGit(ctx, projectDir, "status", "--porcelain")) != ""
	}
	return snapshot
}

func gitAvailable(ctx context.Context, dir string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func runGit(ctx context.Context, dir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func existingFiles(projectDir string, names ...string) []string {
	var files []string
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(projectDir, name)); err == nil {
			files = append(files, filepath.ToSlash(name))
		}
	}
	return files
}

func discoverDocs(projectDir string) []string {
	var docs []string
	if _, err := os.Stat(filepath.Join(projectDir, "README.md")); err == nil {
		docs = append(docs, "README.md")
	}
	entries, err := os.ReadDir(filepath.Join(projectDir, "docs"))
	if err != nil {
		return docs
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		docs = append(docs, filepath.ToSlash(filepath.Join("docs", entry.Name())))
	}
	sort.Strings(docs)
	return docs
}

func discoverCIFiles(projectDir string) []string {
	root := filepath.Join(projectDir, ".github", "workflows")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, filepath.ToSlash(filepath.Join(".github", "workflows", entry.Name())))
	}
	sort.Strings(files)
	return files
}

func discoverRootDirectories(projectDir string) []string {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".git") {
			continue
		}
		dirs = append(dirs, name+"/")
	}
	sort.Strings(dirs)
	return dirs
}

func discoverSubdirectories(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name()+"/")
		}
	}
	sort.Strings(dirs)
	return dirs
}

func countTestFiles(projectDir string) int {
	count := 0
	_ = filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			count++
		}
		return nil
	})
	return count
}

func primaryRuntime(manifests []string, projectDir string) string {
	switch {
	case contains(manifests, "go.mod"):
		return "go"
	case contains(manifests, "package.json"):
		return "node"
	case contains(manifests, "Cargo.toml"):
		return "rust"
	case contains(manifests, "pyproject.toml"):
		return "python"
	default:
		if matches, _ := filepath.Glob(filepath.Join(projectDir, "*.go")); len(matches) != 0 {
			return "go"
		}
		return "unknown"
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func readGoModule(projectDir string) string {
	data, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func renderAgents(snapshot Snapshot) string {
	var b strings.Builder
	slug := policySlug(snapshot.ProjectName)
	fmt.Fprintf(&b, "Use this file as the canonical project contract for `%s`.\n\n", snapshot.ProjectName)
	b.WriteString("Read the linked context files before substantial work. Prefer the `brain` skill and `brain` CLI for project memory, retrieval, and durable context updates.\n\n")
	b.WriteString("## Table Of Contents\n\n")
	for _, entry := range []struct {
		name string
		path string
	}{
		{"Overview", "./.brain/context/overview.md"},
		{"Architecture", "./.brain/context/architecture.md"},
		{"Standards", "./.brain/context/standards.md"},
		{"Workflows", "./.brain/context/workflows.md"},
		{"Memory Policy", "./.brain/context/memory-policy.md"},
		{"Current State", "./.brain/context/current-state.md"},
		{"Policy", "./.brain/policy.yaml"},
	} {
		fmt.Fprintf(&b, "- [%s](%s)\n", entry.name, entry.path)
	}
	if len(snapshot.DocFiles) != 0 {
		b.WriteString("\n## Human Docs\n\n")
		for _, file := range snapshot.DocFiles {
			fmt.Fprintf(&b, "- [%s](./%s)\n", filepath.Base(file), file)
		}
	}
	b.WriteString("\n## Required Workflow\n\n")
	b.WriteString("1. If no validated session is active, run `brain session start --task \"<task>\"`.\n")
	b.WriteString("2. If a session is already active, run `brain session validate` before substantial work.\n")
	b.WriteString("3. Read this file and the linked context files needed for the task.\n")
	fmt.Fprintf(&b, "4. Retrieve project memory with `brain find %s` or `brain search \"%s <task>\"`.\n", slug, slug)
	b.WriteString("5. Use `brain edit` for durable context updates to AGENTS.md, docs, or .brain notes.\n")
	b.WriteString("6. Use `brain session run -- <command>` for required verification commands.\n")
	b.WriteString("7. Finish with `brain session finish` so policy checks can enforce memory updates and required command runs.\n")
	return b.String()
}

func renderOverview(snapshot Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Project: `%s`\n\n", snapshot.ProjectName)
	if snapshot.GoModule != "" {
		fmt.Fprintf(&b, "Go module: `%s`\n\n", snapshot.GoModule)
	}
	fmt.Fprintf(&b, "Primary runtime: `%s`\n\n", snapshot.PrimaryRuntime)
	if len(snapshot.ManifestFiles) != 0 {
		b.WriteString("## Manifests\n\n")
		for _, file := range snapshot.ManifestFiles {
			fmt.Fprintf(&b, "- `%s`\n", file)
		}
		b.WriteString("\n")
	}
	if len(snapshot.RootDirectories) != 0 {
		b.WriteString("## Repo Map\n\n")
		for _, dir := range snapshot.RootDirectories {
			fmt.Fprintf(&b, "- `%s`\n", dir)
		}
	}
	return b.String()
}

func renderArchitecture(snapshot Snapshot) string {
	var b strings.Builder
	b.WriteString("Use this file for the structural shape of the repository.\n\n")
	if len(snapshot.InternalDirs) != 0 {
		b.WriteString("## Internal Packages\n\n")
		for _, dir := range snapshot.InternalDirs {
			fmt.Fprintf(&b, "- `internal/%s`\n", dir)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Architecture Notes\n\n")
	if snapshot.PrimaryRuntime == "go" {
		b.WriteString("- Favor small package boundaries and explicit CLI/app wiring.\n")
		b.WriteString("- Keep public CLI behavior stable; add internal seams only when they improve testability or safety.\n")
		b.WriteString("- Treat generated project context as deterministic repo state, not LLM-authored prose.\n")
		b.WriteString("- Treat session enforcement as the hard-control layer above soft context files.\n")
	} else {
		b.WriteString("- Keep repo boundaries explicit and document key entrypoints in this file.\n")
		b.WriteString("- Update this file when runtime architecture or integration boundaries change.\n")
	}
	return b.String()
}

func renderStandards(snapshot Snapshot) string {
	var b strings.Builder
	b.WriteString("Use this file for implementation and review expectations.\n\n")
	b.WriteString("## Standards\n\n")
	switch snapshot.PrimaryRuntime {
	case "go":
		b.WriteString("- Keep code idiomatic Go with small, concrete abstractions.\n")
		b.WriteString("- Prefer explicit tests for CLI behavior, indexing, retrieval, safety flows, and session enforcement.\n")
		b.WriteString("- Record required verification through `brain session run -- ...` so finish-stage enforcement can validate it.\n")
	default:
		b.WriteString("- Preserve existing repo conventions and testing workflows.\n")
		b.WriteString("- Prefer narrow, reviewable changes over broad speculative rewrites.\n")
	}
	if len(snapshot.CIFiles) != 0 {
		b.WriteString("\n## CI\n\n")
		for _, file := range snapshot.CIFiles {
			fmt.Fprintf(&b, "- `%s`\n", file)
		}
	}
	return b.String()
}

func renderWorkflows(snapshot Snapshot) string {
	var b strings.Builder
	slug := policySlug(snapshot.ProjectName)
	b.WriteString("Use this file for agent operating workflow inside the repo.\n\n")
	b.WriteString("## Startup\n\n")
	b.WriteString("1. If no validated session is active, run `brain session start --task \"<task>\"`.\n")
	b.WriteString("2. If a session already exists, run `brain session validate`.\n")
	b.WriteString("3. Read `AGENTS.md`, `.brain/policy.yaml`, and the linked context files needed for the task.\n")
	fmt.Fprintf(&b, "4. If project memory matters, run `brain find %s` or `brain search \"%s <task>\"`.\n\n", slug, slug)
	b.WriteString("## During Work\n\n")
	b.WriteString("- Keep durable discoveries, decisions, and risks in AGENTS.md, /docs, or .brain notes.\n")
	b.WriteString("- Update existing durable notes instead of duplicating context.\n")
	b.WriteString("- Run required verification commands through `brain session run -- <command>`.\n")
	b.WriteString("- Re-read context before large changes if the task shifts.\n\n")
	b.WriteString("## Close-Out\n\n")
	b.WriteString("- Refresh or update durable notes for meaningful behavior, config, or architecture changes.\n")
	b.WriteString("- Finish with `brain session finish`.\n")
	b.WriteString("- If you must bypass enforcement, use `brain session finish --force --reason \"...\"` so the override is recorded.\n")
	return b.String()
}

func renderMemoryPolicy(snapshot Snapshot) string {
	var b strings.Builder
	b.WriteString("Use this file to decide what is worth keeping in project memory.\n\n")
	b.WriteString("## Capture Required\n\n")
	b.WriteString("- Non-obvious implementation decisions.\n")
	b.WriteString("- Bugs, regressions, and the fix or mitigation.\n")
	b.WriteString("- Config, schema, deployment, or interface changes.\n")
	b.WriteString("- Risks, follow-ups, and unresolved tradeoffs.\n\n")
	b.WriteString("## Capture Optional\n\n")
	b.WriteString("- Small implementation details that are likely to matter later.\n")
	b.WriteString("- Helpful command sequences worth repeating.\n\n")
	b.WriteString("## Do Not Capture\n\n")
	b.WriteString("- Trivial edits with no future value.\n")
	b.WriteString("- Temporary command noise already obvious from code or tests.\n")
	b.WriteString("- Duplicate notes when an existing note can be updated.\n")
	return b.String()
}

func renderCurrentState(snapshot Snapshot) string {
	var b strings.Builder
	b.WriteString("This file is a deterministic snapshot of the repository state at the last refresh.\n\n")
	b.WriteString("## Repository\n\n")
	fmt.Fprintf(&b, "- Project: `%s`\n", snapshot.ProjectName)
	b.WriteString("- Root: `.`\n")
	fmt.Fprintf(&b, "- Runtime: `%s`\n", snapshot.PrimaryRuntime)
	if snapshot.GoModule != "" {
		fmt.Fprintf(&b, "- Go module: `%s`\n", snapshot.GoModule)
	}
	if snapshot.HasGit {
		if snapshot.CurrentBranch != "" {
			fmt.Fprintf(&b, "- Current branch: `%s`\n", snapshot.CurrentBranch)
		}
		if snapshot.DefaultBranch != "" {
			fmt.Fprintf(&b, "- Default branch: `%s`\n", snapshot.DefaultBranch)
		}
		if snapshot.RemoteURL != "" {
			fmt.Fprintf(&b, "- Remote: `%s`\n", snapshot.RemoteURL)
		}
	}
	fmt.Fprintf(&b, "- Go test files: `%d`\n", snapshot.TestFiles)
	if len(snapshot.DocFiles) != 0 {
		b.WriteString("\n## Docs\n\n")
		for _, file := range snapshot.DocFiles {
			fmt.Fprintf(&b, "- `%s`\n", file)
		}
	}
	return b.String()
}

func renderWrapper(agent string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "This `%s` wrapper delegates to the root project contract.\n\n", agent)
	b.WriteString("## Required Reads\n\n")
	b.WriteString("- `../AGENTS.md`\n")
	b.WriteString("- `../.brain/policy.yaml`\n")
	b.WriteString("- `../.brain/context/overview.md`\n")
	b.WriteString("- `../.brain/context/architecture.md`\n")
	b.WriteString("- `../.brain/context/workflows.md`\n")
	b.WriteString("- `../.brain/context/memory-policy.md`\n\n")
	b.WriteString("## Required Behavior\n\n")
	b.WriteString("- Treat `../AGENTS.md` as the canonical project contract.\n")
	b.WriteString("- If no validated session is active, run `brain session start --task \"<task>\"`.\n")
	b.WriteString("- If a session is already active, run `brain session validate` before substantial work.\n")
	b.WriteString("- Use the `brain` skill and `brain` CLI when project memory or project-local context matters.\n")
	b.WriteString("- Use `brain session run -- <command>` for required verification commands.\n")
	b.WriteString("- Finish with `brain session finish` and mention relevant note updates in the final response.\n")
	return b.String()
}

func renderGitIgnore() string {
	return strings.TrimSpace(`
.brain/session.json
.brain/sessions/
.brain/policy.override.yaml
`) + "\n"
}
