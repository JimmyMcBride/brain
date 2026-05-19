package contextaudit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"brain/internal/backup"
	"brain/internal/history"
	"brain/internal/index"
	"brain/internal/notes"
	"brain/internal/session"
	"brain/internal/structure"
	"brain/internal/templates"
	"brain/internal/workspace"
)

func TestAuditWellCoveredRepoHasNoFindings(t *testing.T) {
	root := newAuditFixture(t)
	writeAuditFile(t, root, "go.mod", "module example.com/audit\n")
	writeAuditFile(t, root, "main.go", "package main\nfunc main() {}\n")
	writeAuditFile(t, root, "internal/foo/foo.go", "package foo\n")
	writeAuditFile(t, root, "internal/foo/foo_test.go", "package foo\n")
	writeAuditFile(t, root, ".github/workflows/ci.yml", "name: ci\n")
	writeAuditContext(t, root, strings.Join([]string{
		"AGENTS.md",
		"README.md",
		".brain/",
		"docs/",
		"internal/",
		"internal/foo/",
		"go.mod",
		"main.go",
		".github/workflows/ci.yml",
	}, "\n"))

	report, err := newAuditManager(t, root).Audit(context.Background(), Request{ProjectDir: root})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("expected no findings, got %#v", report.Findings)
	}
}

func TestAuditReportsMissingCoverage(t *testing.T) {
	root := newAuditFixture(t)
	writeAuditFile(t, root, "go.mod", "module example.com/audit\n")
	writeAuditFile(t, root, "main.go", "package main\nfunc main() {}\n")
	writeAuditFile(t, root, "Dockerfile", "FROM scratch\n")
	writeAuditContext(t, root, "AGENTS.md\nREADME.md\n.brain/\ndocs/\nmain.go\n")

	report, err := newAuditManager(t, root).Audit(context.Background(), Request{ProjectDir: root})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if !hasFinding(report, "missing_coverage", "go.mod") || !hasFinding(report, "missing_coverage", "Dockerfile") {
		t.Fatalf("expected missing coverage findings, got %#v", report.Findings)
	}
}

func TestAuditReportsStaleReferences(t *testing.T) {
	root := newAuditFixture(t)
	writeAuditFile(t, root, "go.mod", "module example.com/audit\n")
	writeAuditFile(t, root, "main.go", "package main\nfunc main() {}\n")
	writeAuditContext(t, root, "AGENTS.md\nREADME.md\n.brain/\ndocs/\ngo.mod\nmain.go\n`internal/missing/`\n")

	report, err := newAuditManager(t, root).Audit(context.Background(), Request{ProjectDir: root})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if !hasFinding(report, "stale_reference", ".brain/context/architecture.md") {
		t.Fatalf("expected stale reference finding, got %#v", report.Findings)
	}
}

func TestAuditSinceAddsDiffFindings(t *testing.T) {
	root := newAuditFixture(t)
	writeAuditFile(t, root, "go.mod", "module example.com/audit\n")
	writeAuditFile(t, root, "main.go", "package main\nfunc main() {}\n")
	writeAuditContext(t, root, "AGENTS.md\nREADME.md\n.brain/\ndocs/\ngo.mod\nmain.go\n")
	runTestGit(t, root, "init")
	runTestGit(t, root, "config", "user.email", "test@example.com")
	runTestGit(t, root, "config", "user.name", "Test")
	runTestGit(t, root, "add", ".")
	runTestGit(t, root, "commit", "-m", "baseline")
	writeAuditFile(t, root, "go.mod", "module example.com/audit\n\ngo 1.23\n")
	writeAuditFile(t, root, "scratch.txt", "ignored\n")

	report, err := newAuditManager(t, root).Audit(context.Background(), Request{ProjectDir: root, Since: "HEAD"})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if report.Base.Source != "flag" || !report.Base.DiffAvailable {
		t.Fatalf("expected explicit diff base, got %#v", report.Base)
	}
	if !hasFinding(report, "changed_surface", "go.mod") {
		t.Fatalf("expected changed_surface for go.mod, got %#v", report.Findings)
	}
	if hasFinding(report, "changed_surface", "scratch.txt") {
		t.Fatalf("did not expect non-relevant scratch diff finding: %#v", report.Findings)
	}
}

func hasFinding(report *Report, category, evidence string) bool {
	for _, finding := range report.Findings {
		if finding.Category == category && finding.EvidencePath == evidence {
			return true
		}
	}
	return false
}

func newAuditFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{".brain/context", ".brain/state", ".brain/resources/changes", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func writeAuditContext(t *testing.T, root, coverage string) {
	t.Helper()
	body := "# Context\n\n" + coverage + "\n"
	for _, path := range []string{
		"AGENTS.md",
		"README.md",
		".brain/context/architecture.md",
		".brain/context/workflows.md",
		"docs/project-architecture.md",
		"docs/project-workflows.md",
	} {
		writeAuditFile(t, root, path, body)
	}
}

func newAuditManager(t *testing.T, root string) *Manager {
	t.Helper()
	store, err := index.New(filepath.Join(root, ".brain/state/brain.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	workspaceSvc := workspace.New(root)
	structureManager, err := structure.New(store, workspaceSvc)
	if err != nil {
		t.Fatal(err)
	}
	historyLog := history.New(filepath.Join(root, ".brain/state/history.jsonl"))
	notesManager := notes.New(workspaceSvc, templates.New(), backup.New(filepath.Join(root, ".brain/state/backups")), historyLog)
	return New(structureManager, notesManager, session.New(historyLog))
}

func writeAuditFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runTestGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(out))
	}
}
