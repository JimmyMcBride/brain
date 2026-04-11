package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brain/internal/workspace"
)

func TestValidateWorkspaceMarkdownRejectsNestedFrontmatter(t *testing.T) {
	root := t.TempDir()
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	content := "---\ntitle: Current State\ntype: resource\n---\n---\ntitle: Broken\n---\n# Body\n"
	if err := os.WriteFile(filepath.Join(root, ".brain", "context", "current-state.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ValidateWorkspaceMarkdown(workspaceSvc)
	if err == nil || !strings.Contains(err.Error(), "nested frontmatter block at body start") {
		t.Fatalf("expected nested frontmatter error, got %v", err)
	}
}
