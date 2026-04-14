package projectcontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedDocumentGoldens(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo-project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(project, "go.mod"), "module example.com/demo\n\ngo 1.26\n")
	mustWriteFile(t, filepath.Join(project, "README.md"), "# demo\n")

	manager := New(t.TempDir())
	if _, err := manager.Install(context.Background(), Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		path string
	}{
		{"agents.golden", "AGENTS.md"},
		{"overview.golden", ".brain/context/overview.md"},
		{"current-state.golden", ".brain/context/current-state.md"},
		{"policy.golden", ".brain/policy.yaml"},
		{"gitignore.golden", ".gitignore"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotBytes, err := os.ReadFile(filepath.Join(project, filepath.FromSlash(tc.path)))
			if err != nil {
				t.Fatal(err)
			}
			got := strings.ReplaceAll(filepath.ToSlash(string(gotBytes)), filepath.ToSlash(project), "<PROJECT>")
			got = strings.ReplaceAll(got, "demo-project", "<PROJECT_NAME>")
			got = strings.ReplaceAll(got, "DemoProject", "<PROJECT_TITLE>")
			got = strings.ReplaceAll(got, "\r\n", "\n")
			want, err := os.ReadFile(filepath.Join("testdata", tc.name))
			if err != nil {
				t.Fatal(err)
			}
			wantText := strings.ReplaceAll(string(want), "\r\n", "\n")
			if got != wantText {
				t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", tc.name, wantText, got)
			}
		})
	}
}

func TestAgentIntegrationGoldens(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo-project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}

	manager := New(t.TempDir())

	mustWriteFile(t, filepath.Join(project, ".codex", "AGENTS.md"), "# Existing Codex Notes\n")
	if _, err := manager.Adopt(context.Background(), Request{ProjectDir: project}); err != nil {
		t.Fatal(err)
	}
	assertGolden(t, filepath.Join(project, ".codex", "AGENTS.md"), "agent-integration-appended-codex.golden")

	projectCreate := filepath.Join(t.TempDir(), "demo-project")
	if err := os.MkdirAll(projectCreate, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Adopt(context.Background(), Request{
		ProjectDir: projectCreate,
		Agents:     []string{"claude"},
	}); err != nil {
		t.Fatal(err)
	}
	assertGolden(t, filepath.Join(projectCreate, ".claude", "CLAUDE.md"), "agent-integration-created-claude.golden")

	projectUpdate := filepath.Join(t.TempDir(), "demo-project")
	if err := os.MkdirAll(projectUpdate, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(projectUpdate, ".codex", "AGENTS.md"), "# Existing Codex Notes\n\n## Brain\n\n<!-- brain:begin agent-integration-codex -->\nstale\n<!-- brain:end agent-integration-codex -->\n")
	if _, err := manager.Refresh(context.Background(), Request{
		ProjectDir: projectUpdate,
		Agents:     []string{"codex"},
	}); err != nil {
		t.Fatal(err)
	}
	assertGolden(t, filepath.Join(projectUpdate, ".codex", "AGENTS.md"), "agent-integration-updated-codex.golden")
}

func assertGolden(t *testing.T, path, golden string) {
	t.Helper()
	gotBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.ReplaceAll(string(gotBytes), "\r\n", "\n")
	want, err := os.ReadFile(filepath.Join("testdata", golden))
	if err != nil {
		t.Fatal(err)
	}
	wantText := strings.ReplaceAll(string(want), "\r\n", "\n")
	if got != wantText {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", golden, wantText, got)
	}
}
