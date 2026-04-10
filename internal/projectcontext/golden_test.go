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
	if _, err := manager.Install(context.Background(), Request{
		ProjectDir: project,
		Agents:     []string{"codex"},
	}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		path string
	}{
		{"agents.golden", "AGENTS.md"},
		{"wrapper-codex.golden", ".codex/AGENTS.md"},
		{"overview.golden", ".brain/context/overview.md"},
		{"current-state.golden", ".brain/context/current-state.md"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotBytes, err := os.ReadFile(filepath.Join(project, filepath.FromSlash(tc.path)))
			if err != nil {
				t.Fatal(err)
			}
			got := strings.ReplaceAll(filepath.ToSlash(string(gotBytes)), filepath.ToSlash(project), "<PROJECT>")
			got = strings.ReplaceAll(got, "demo-project", "<PROJECT_NAME>")
			want, err := os.ReadFile(filepath.Join("testdata", tc.name))
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", tc.name, string(want), got)
			}
		})
	}
}
