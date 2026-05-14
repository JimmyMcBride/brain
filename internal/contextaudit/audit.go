package contextaudit

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"brain/internal/notes"
	"brain/internal/session"
	"brain/internal/structure"
)

type Manager struct {
	structure *structure.Manager
	notes     *notes.Manager
	session   *session.Manager
}

type Request struct {
	ProjectDir string
	Since      string
	Proposal   bool
}

type Report struct {
	Summary  Summary       `json:"summary"`
	Base     BaseInfo      `json:"base"`
	Findings []Finding     `json:"findings"`
	Proposal *ProposalInfo `json:"proposal,omitempty"`
}

type Summary struct {
	Runtime          string `json:"runtime"`
	DocumentsAudited int    `json:"documents_audited"`
	Findings         int    `json:"findings"`
	CoverageFindings int    `json:"coverage_findings"`
	DiffFindings     int    `json:"diff_findings"`
	StaleFindings    int    `json:"stale_findings"`
}

type BaseInfo struct {
	Source        string   `json:"source"`
	Ref           string   `json:"ref,omitempty"`
	Commit        string   `json:"commit,omitempty"`
	DiffAvailable bool     `json:"diff_available"`
	ChangedFiles  []string `json:"changed_files,omitempty"`
}

type Finding struct {
	ID              string `json:"id"`
	Category        string `json:"category"`
	Severity        string `json:"severity"`
	Source          string `json:"source"`
	EvidencePath    string `json:"evidence_path"`
	SuggestedTarget string `json:"suggested_target"`
	Recommendation  string `json:"recommendation"`
	Details         string `json:"details,omitempty"`
}

type ProposalInfo struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

type SessionHint struct {
	ShouldAudit  bool     `json:"should_audit"`
	Since        string   `json:"since,omitempty"`
	ChangedFiles []string `json:"changed_files,omitempty"`
}

type auditDoc struct {
	Path    string
	Content string
	Lower   string
}

var pathTokenPattern = regexp.MustCompile("`([^`]+)`|\\[[^\\]]+\\]\\(([^)]+)\\)")

func New(structureManager *structure.Manager, notesManager *notes.Manager, sessionManager *session.Manager) *Manager {
	return &Manager{structure: structureManager, notes: notesManager, session: sessionManager}
}

func (m *Manager) Audit(ctx context.Context, req Request) (*Report, error) {
	if m == nil || m.structure == nil {
		return nil, errors.New("context audit requires structural context")
	}
	projectDir, err := filepath.Abs(defaultProjectDir(req.ProjectDir))
	if err != nil {
		return nil, err
	}
	snapshot, err := m.structure.Snapshot(ctx, "")
	if err != nil {
		return nil, err
	}
	docs, err := readAuditDocs(projectDir)
	if err != nil {
		return nil, err
	}
	base, err := m.resolveBase(ctx, projectDir, req.Since)
	if err != nil {
		return nil, err
	}
	if base.DiffAvailable {
		base.ChangedFiles = changedFilesSince(ctx, projectDir, base.Commit)
	}

	corpus := strings.ToLower(joinDocs(docs))
	findings := make([]Finding, 0)
	findings = append(findings, coverageFindings(projectDir, snapshot, corpus)...)
	findings = append(findings, staleReferenceFindings(projectDir, docs)...)
	findings = append(findings, diffFindings(base.ChangedFiles)...)
	findings = dedupeFindings(findings)
	sortFindings(findings)

	report := &Report{
		Summary: Summary{
			Runtime:          snapshot.Summary.Runtime,
			DocumentsAudited: len(docs),
			Findings:         len(findings),
		},
		Base:     base,
		Findings: findings,
	}
	for _, finding := range findings {
		switch finding.Source {
		case "coverage":
			report.Summary.CoverageFindings++
		case "diff":
			report.Summary.DiffFindings++
		case "stale_reference":
			report.Summary.StaleFindings++
		}
	}
	if req.Proposal {
		proposal, err := m.createProposal(report)
		if err != nil {
			return nil, err
		}
		report.Proposal = proposal
	}
	return report, nil
}

func (m *Manager) SessionHint(ctx context.Context, projectDir string, active *session.ActiveSession) (*SessionHint, error) {
	if active == nil || strings.TrimSpace(active.GitBaseline.Head) == "" {
		return &SessionHint{}, nil
	}
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return nil, err
	}
	changed := changedFilesSince(ctx, projectDir, active.GitBaseline.Head)
	relevant := auditRelevantFiles(changed)
	if len(relevant) == 0 {
		return &SessionHint{}, nil
	}
	return &SessionHint{
		ShouldAudit:  true,
		Since:        active.GitBaseline.Head,
		ChangedFiles: relevant,
	}, nil
}

func RenderHuman(w io.Writer, report *Report) error {
	if report == nil {
		return errors.New("context audit report is required")
	}
	if _, err := fmt.Fprintf(w, "## Context Audit\n\n- Runtime: `%s`\n- Documents audited: %d\n- Findings: %d\n", report.Summary.Runtime, report.Summary.DocumentsAudited, report.Summary.Findings); err != nil {
		return err
	}
	if report.Base.DiffAvailable {
		if _, err := fmt.Fprintf(w, "- Diff base: `%s` (%s)\n- Changed files: %d\n", shortCommit(report.Base.Commit), report.Base.Source, len(report.Base.ChangedFiles)); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintf(w, "- Diff base: none (%s)\n", report.Base.Source); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n## Findings\n\n"); err != nil {
		return err
	}
	if len(report.Findings) == 0 {
		if _, err := io.WriteString(w, "- No context coverage issues found.\n"); err != nil {
			return err
		}
	} else {
		for _, finding := range report.Findings {
			if _, err := fmt.Fprintf(w, "- [%s] %s `%s` -> `%s`\n", finding.Severity, finding.Category, finding.EvidencePath, finding.SuggestedTarget); err != nil {
				return err
			}
			if strings.TrimSpace(finding.Details) != "" {
				if _, err := fmt.Fprintf(w, "  Details: %s\n", finding.Details); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(w, "  Recommendation: %s\n", finding.Recommendation); err != nil {
				return err
			}
		}
	}
	if report.Proposal != nil {
		if _, err := fmt.Fprintf(w, "\n## Proposal\n\n- Created: `%s`\n", report.Proposal.Path); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) resolveBase(ctx context.Context, projectDir, since string) (BaseInfo, error) {
	if !gitAvailable(ctx, projectDir) {
		return BaseInfo{Source: "none"}, nil
	}
	if strings.TrimSpace(since) != "" {
		commit, err := resolveCommit(ctx, projectDir, since)
		if err != nil {
			return BaseInfo{}, err
		}
		return BaseInfo{Source: "flag", Ref: since, Commit: commit, DiffAvailable: true}, nil
	}
	if m != nil && m.session != nil {
		active, err := m.session.Active(projectDir)
		if err == nil && active != nil && strings.TrimSpace(active.GitBaseline.Head) != "" {
			return BaseInfo{Source: "session", Ref: "session baseline", Commit: active.GitBaseline.Head, DiffAvailable: true}, nil
		}
	}
	for _, ref := range []string{"@{upstream}", "origin/HEAD", "origin/develop", "origin/main"} {
		commit := strings.TrimSpace(runGit(ctx, projectDir, "merge-base", "HEAD", ref))
		if commit != "" {
			return BaseInfo{Source: "merge-base", Ref: ref, Commit: commit, DiffAvailable: true}, nil
		}
	}
	return BaseInfo{Source: "none"}, nil
}

func (m *Manager) createProposal(report *Report) (*ProposalInfo, error) {
	if m == nil || m.notes == nil {
		return nil, errors.New("context audit proposal requires notes manager")
	}
	title := "Context Audit Proposal"
	filename := "context-audit-" + time.Now().UTC().Format("20060102-150405")
	note, err := m.notes.Create(notes.CreateInput{
		Title:    title,
		Filename: filename,
		NoteType: "context_audit_proposal",
		Section:  ".brain",
		Subdir:   "resources/changes",
		Body:     renderProposalBody(report),
		Metadata: map[string]any{
			"type":                "context_audit_proposal",
			"audit_base_source":   report.Base.Source,
			"audit_base_ref":      report.Base.Ref,
			"audit_base_commit":   report.Base.Commit,
			"audit_finding_count": report.Summary.Findings,
		},
	})
	if err != nil {
		return nil, err
	}
	return &ProposalInfo{Path: note.Path, Title: note.Title}, nil
}

func renderProposalBody(report *Report) string {
	var b strings.Builder
	b.WriteString("# Context Audit Proposal\n\n")
	b.WriteString("## Source Provenance\n\n")
	b.WriteString("- Mode: `context_audit`\n")
	b.WriteString("- Base source: `" + report.Base.Source + "`\n")
	if report.Base.Ref != "" {
		b.WriteString("- Base ref: `" + report.Base.Ref + "`\n")
	}
	if report.Base.Commit != "" {
		b.WriteString("- Base commit: `" + report.Base.Commit + "`\n")
	}
	b.WriteString("\n## Repo And Diff Summary\n\n")
	b.WriteString(fmt.Sprintf("- Runtime: `%s`\n", report.Summary.Runtime))
	b.WriteString(fmt.Sprintf("- Documents audited: `%d`\n", report.Summary.DocumentsAudited))
	b.WriteString(fmt.Sprintf("- Findings: `%d`\n", report.Summary.Findings))
	if len(report.Base.ChangedFiles) != 0 {
		b.WriteString("\n### Changed Files\n\n")
		for _, path := range report.Base.ChangedFiles {
			b.WriteString("- `" + path + "`\n")
		}
	}
	b.WriteString("\n## Findings\n\n")
	if len(report.Findings) == 0 {
		b.WriteString("- No context coverage issues found.\n")
		return b.String()
	}
	for _, finding := range report.Findings {
		b.WriteString("### " + finding.Category + ": `" + finding.EvidencePath + "`\n\n")
		b.WriteString("- Severity: `" + finding.Severity + "`\n")
		b.WriteString("- Source: `" + finding.Source + "`\n")
		b.WriteString("- Suggested target: `" + finding.SuggestedTarget + "`\n")
		if finding.Details != "" {
			b.WriteString("- Details: " + finding.Details + "\n")
		}
		b.WriteString("- Recommendation: " + finding.Recommendation + "\n\n")
	}
	b.WriteString("## Suggested Durable Updates\n\n")
	b.WriteString("- Review the findings above and update the suggested target files only where the project fact is durable.\n")
	b.WriteString("- Prefer Local Notes or focused `.brain/resources/...` notes for hand-authored context that should survive generated context refreshes.\n")
	return b.String()
}

func readAuditDocs(projectDir string) ([]auditDoc, error) {
	candidates := []string{
		"AGENTS.md",
		"README.md",
		"docs/architecture.md",
		"docs/project-architecture.md",
		"docs/project-overview.md",
		"docs/project-workflows.md",
		"docs/usage.md",
		"docs/skills.md",
		"docs/why.md",
	}
	for _, pattern := range []string{".brain/context/*.md", "docs/project-*.md"} {
		matches, _ := filepath.Glob(filepath.Join(projectDir, filepath.FromSlash(pattern)))
		for _, match := range matches {
			rel, err := filepath.Rel(projectDir, match)
			if err == nil {
				candidates = append(candidates, filepath.ToSlash(rel))
			}
		}
	}
	seen := map[string]struct{}{}
	docs := []auditDoc{}
	for _, rel := range candidates {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" {
			continue
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		raw, err := os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		content := string(raw)
		docs = append(docs, auditDoc{Path: rel, Content: content, Lower: strings.ToLower(content)})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	return docs, nil
}

func coverageFindings(projectDir string, snapshot *structure.Snapshot, corpus string) []Finding {
	if snapshot == nil {
		return nil
	}
	items := []structure.Item{}
	items = append(items, snapshot.Boundaries...)
	items = append(items, snapshot.Entrypoints...)
	items = append(items, snapshot.ConfigSurfaces...)
	items = append(items, snapshot.TestSurfaces...)
	items = append(items, deploymentItems(projectDir)...)
	findings := []Finding{}
	for _, item := range items {
		if !isHighSignalItem(item) || pathCovered(corpus, item.Path) {
			continue
		}
		target := suggestedTargetFor(item.Kind, item.Role, item.Path)
		findings = append(findings, Finding{
			ID:              findingID("missing_coverage", item.Path),
			Category:        "missing_coverage",
			Severity:        "warning",
			Source:          "coverage",
			EvidencePath:    item.Path,
			SuggestedTarget: target,
			Recommendation:  fmt.Sprintf("Document `%s` in `%s` or a focused `.brain/resources/...` note if it is durable project context.", item.Path, target),
			Details:         item.Summary,
		})
	}
	return findings
}

func diffFindings(changedFiles []string) []Finding {
	relevant := auditRelevantFiles(changedFiles)
	findings := make([]Finding, 0, len(relevant))
	for _, path := range relevant {
		target := suggestedTargetForPath(path)
		findings = append(findings, Finding{
			ID:              findingID("changed_surface", path),
			Category:        "changed_surface",
			Severity:        "info",
			Source:          "diff",
			EvidencePath:    path,
			SuggestedTarget: target,
			Recommendation:  fmt.Sprintf("Review whether `%s` changes durable project context; update `%s` or a focused `.brain/resources/...` note if yes.", path, target),
		})
	}
	return findings
}

func staleReferenceFindings(projectDir string, docs []auditDoc) []Finding {
	findings := []Finding{}
	seen := map[string]struct{}{}
	for _, doc := range docs {
		matches := pathTokenPattern.FindAllStringSubmatch(doc.Content, -1)
		for _, match := range matches {
			token := strings.TrimSpace(match[1])
			if token == "" && len(match) > 2 {
				token = strings.TrimSpace(match[2])
			}
			ref, ok := normalizeReference(doc.Path, token)
			if !ok || pathExists(projectDir, ref) {
				continue
			}
			key := doc.Path + "::" + ref
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			findings = append(findings, Finding{
				ID:              findingID("stale_reference", key),
				Category:        "stale_reference",
				Severity:        "warning",
				Source:          "stale_reference",
				EvidencePath:    doc.Path,
				SuggestedTarget: doc.Path,
				Recommendation:  fmt.Sprintf("Update or remove stale path reference `%s`.", ref),
				Details:         "referenced path was not found in the workspace",
			})
		}
	}
	return findings
}

func isHighSignalItem(item structure.Item) bool {
	path := strings.TrimSpace(item.Path)
	if path == "" || path == "." {
		return false
	}
	if strings.HasPrefix(path, ".git") || strings.HasPrefix(path, ".brain/state") || strings.HasPrefix(path, ".brain/sessions") {
		return false
	}
	switch item.Kind {
	case "entrypoint", "config_surface", "test_surface":
		return true
	case "boundary":
		if item.Role != "unknown" {
			return true
		}
		return path == "skills/" || path == "templates/"
	default:
		return false
	}
}

func deploymentItems(projectDir string) []structure.Item {
	names := []string{
		"Dockerfile",
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
		"vercel.json",
		"netlify.toml",
		"fly.toml",
		"render.yaml",
		"Procfile",
	}
	items := []structure.Item{}
	for _, name := range names {
		if pathExists(projectDir, name) {
			items = append(items, structure.Item{Kind: "config_surface", Path: name, Label: filepath.Base(name), Role: "deploy", Summary: "Deployment surface"})
		}
	}
	entries, err := os.ReadDir(filepath.Join(projectDir, "scripts"))
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			if strings.Contains(name, "deploy") || strings.Contains(name, "publish") || strings.Contains(name, "release") {
				rel := filepath.ToSlash(filepath.Join("scripts", entry.Name()))
				items = append(items, structure.Item{Kind: "config_surface", Path: rel, Label: entry.Name(), Role: "deploy", Summary: "Deployment or release script"})
			}
		}
	}
	return items
}

func auditRelevantFiles(paths []string) []string {
	out := []string{}
	for _, path := range paths {
		path = filepath.ToSlash(strings.TrimSpace(path))
		if path == "" {
			continue
		}
		if isAuditRelevantPath(path) {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return dedupeStrings(out)
}

func isAuditRelevantPath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	lower := strings.ToLower(path)
	switch {
	case path == "AGENTS.md":
		return true
	case strings.HasPrefix(path, ".brain/context/"):
		return true
	case strings.HasPrefix(path, "docs/"):
		return true
	case strings.HasPrefix(path, ".github/workflows/"):
		return true
	case strings.HasPrefix(path, "cmd/") || strings.HasPrefix(path, "internal/") || strings.HasPrefix(path, "pkg/") || strings.HasPrefix(path, "src/") || strings.HasPrefix(path, "app/"):
		return true
	case strings.Contains(lower, "deploy") || strings.Contains(lower, "release") || strings.Contains(lower, "publish"):
		return true
	case strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec."):
		return true
	}
	switch base {
	case "go.mod", "package.json", "cargo.toml", "pyproject.toml", "makefile", "dockerfile", "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml", "vercel.json", "netlify.toml", "fly.toml", "render.yaml", "procfile":
		return true
	default:
		return false
	}
}

func suggestedTargetFor(kind, role, path string) string {
	if role == "ci" || role == "deploy" || role == "config" || kind == "test_surface" {
		return ".brain/context/workflows.md"
	}
	if strings.HasPrefix(path, "docs/") || strings.HasPrefix(path, ".brain/context/") || path == "AGENTS.md" {
		return path
	}
	return ".brain/context/architecture.md"
}

func suggestedTargetForPath(path string) string {
	lower := strings.ToLower(path)
	switch {
	case path == "AGENTS.md" || strings.HasPrefix(path, ".brain/context/") || strings.HasPrefix(path, "docs/"):
		return path
	case strings.HasPrefix(path, ".github/workflows/") || strings.Contains(lower, "deploy") || strings.Contains(lower, "release") || strings.Contains(lower, "publish"):
		return ".brain/context/workflows.md"
	case strings.HasSuffix(lower, "_test.go") || strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec."):
		return ".brain/context/workflows.md"
	case strings.HasSuffix(lower, ".toml") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".json") || filepath.Base(lower) == "go.mod":
		return ".brain/context/workflows.md"
	default:
		return ".brain/context/architecture.md"
	}
}

func normalizeReference(docPath, token string) (string, bool) {
	token = strings.TrimSpace(token)
	if token == "" || strings.Contains(token, "://") || strings.HasPrefix(token, "#") {
		return "", false
	}
	token = strings.Split(token, "#")[0]
	token = strings.Split(token, "?")[0]
	token = strings.Trim(token, "\"'")
	if token == "" || strings.ContainsAny(token, "*<>|") || strings.Contains(token, " ") || strings.Contains(token, "...") {
		return "", false
	}
	for _, prefix := range []string{".plan", "release/", "origin/", "codex/"} {
		if strings.HasPrefix(token, prefix) {
			return "", false
		}
	}
	for _, optional := range []string{".brain/policy.override.yaml", ".brain/session.json"} {
		if token == optional {
			return "", false
		}
	}
	if !looksLikePath(token) {
		return "", false
	}
	var rel string
	if strings.HasPrefix(token, "./") || strings.HasPrefix(token, "../") {
		rel = filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(docPath), token)))
	} else {
		rel = filepath.ToSlash(filepath.Clean(token))
	}
	if rel == "." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
		return "", false
	}
	return rel, true
}

func looksLikePath(token string) bool {
	if strings.HasPrefix(token, ".brain/") || strings.HasPrefix(token, "docs/") || strings.HasPrefix(token, "cmd/") || strings.HasPrefix(token, "internal/") || strings.HasPrefix(token, "skills/") || strings.HasPrefix(token, "templates/") || strings.HasPrefix(token, ".github/") {
		return true
	}
	if strings.Contains(token, "/") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(token))
	switch ext {
	case ".md", ".go", ".yaml", ".yml", ".json", ".toml", ".mod", ".sum":
		return true
	default:
		return false
	}
}

func pathCovered(corpus, path string) bool {
	path = strings.ToLower(filepath.ToSlash(strings.TrimSpace(path)))
	if path == "" {
		return true
	}
	variants := []string{path}
	if strings.HasSuffix(path, "/") {
		variants = append(variants, strings.TrimSuffix(path, "/"))
	} else {
		variants = append(variants, path+"/")
	}
	for _, variant := range variants {
		if variant != "" && strings.Contains(corpus, variant) {
			return true
		}
	}
	return false
}

func pathExists(projectDir, rel string) bool {
	_, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(rel)))
	return err == nil
}

func changedFilesSince(ctx context.Context, projectDir, base string) []string {
	changed := []string{}
	if strings.TrimSpace(base) != "" {
		out := runGit(ctx, projectDir, "diff", "--name-only", "--diff-filter=ACMRTUXB", base, "--")
		for _, line := range strings.Split(out, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				changed = append(changed, filepath.ToSlash(line))
			}
		}
	}
	status := runGit(ctx, projectDir, "status", "--porcelain")
	for _, line := range strings.Split(status, "\n") {
		if strings.TrimSpace(line) == "" || len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = parts[len(parts)-1]
		}
		if path != "" {
			changed = append(changed, filepath.ToSlash(path))
		}
	}
	sort.Strings(changed)
	return dedupeStrings(changed)
}

func gitAvailable(ctx context.Context, projectDir string) bool {
	return strings.TrimSpace(runGit(ctx, projectDir, "rev-parse", "--is-inside-work-tree")) == "true"
}

func resolveCommit(ctx context.Context, projectDir, ref string) (string, error) {
	commit := strings.TrimSpace(runGit(ctx, projectDir, "rev-parse", "--verify", ref+"^{commit}"))
	if commit == "" {
		return "", fmt.Errorf("resolve audit base %q: not a commit", ref)
	}
	return commit, nil
}

func runGit(ctx context.Context, projectDir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", projectDir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func joinDocs(docs []auditDoc) string {
	var b strings.Builder
	for _, doc := range docs {
		b.WriteString(doc.Path)
		b.WriteString("\n")
		b.WriteString(doc.Content)
		b.WriteString("\n")
	}
	return b.String()
}

func dedupeFindings(findings []Finding) []Finding {
	out := make([]Finding, 0, len(findings))
	seen := map[string]struct{}{}
	for _, finding := range findings {
		key := finding.Category + "::" + finding.EvidencePath + "::" + finding.SuggestedTarget
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, finding)
	}
	return out
}

func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return severityRank(findings[i].Severity) < severityRank(findings[j].Severity)
		}
		if findings[i].Category != findings[j].Category {
			return findings[i].Category < findings[j].Category
		}
		return findings[i].EvidencePath < findings[j].EvidencePath
	})
}

func severityRank(severity string) int {
	switch severity {
	case "warning":
		return 0
	case "info":
		return 1
	default:
		return 2
	}
}

func findingID(category, path string) string {
	hash := sha1.Sum([]byte(category + ":" + path))
	return category + "-" + hex.EncodeToString(hash[:])[:10]
}

func defaultProjectDir(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return "."
	}
	return dir
}

func shortCommit(commit string) string {
	if len(commit) <= 12 {
		return commit
	}
	return commit[:12]
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
