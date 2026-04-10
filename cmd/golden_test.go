package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCoreCommandGoldens(t *testing.T) {
	env := newCLIEnv(t)
	requireOK(t, env.run(t, "", "--config", env.config, "init", "--vault", env.vault, "--data", env.data))
	requireOK(t, env.run(t, "", "--config", env.config, "add", "Golden Note", "-s", "Resources", "-t", "resource", "-b", "# Golden Note\n\nLexical and semantic retrieval both matter."))
	requireOK(t, env.run(t, "", "--config", env.config, "reindex"))

	cases := []struct {
		name string
		args []string
	}{
		{"doctor.human.golden", []string{"--config", env.config, "doctor"}},
		{"doctor.json.golden", []string{"--config", env.config, "--json", "doctor"}},
		{"find.human.golden", []string{"--config", env.config, "find", "golden"}},
		{"find.json.golden", []string{"--config", env.config, "--json", "find", "golden"}},
		{"search.human.golden", []string{"--config", env.config, "search", "lexical semantic retrieval"}},
		{"search.json.golden", []string{"--config", env.config, "--json", "search", "lexical semantic retrieval"}},
		{"read.human.golden", []string{"--config", env.config, "read", "Resources/golden-note.md"}},
		{"read.json.golden", []string{"--config", env.config, "--json", "read", "Resources/golden-note.md"}},
		{"skills-targets.human.golden", []string{"skills", "targets", "--scope", "both", "-a", "codex", "--project", env.project, "--skill-root", env.custom}},
		{"skills-targets.json.golden", []string{"--json", "skills", "targets", "--scope", "both", "-a", "codex", "--project", env.project, "--skill-root", env.custom}},
		{"context-install.human.golden", []string{"context", "install", "--project", env.project, "--agent", "codex", "--dry-run"}},
		{"context-install.json.golden", []string{"--json", "context", "install", "--project", env.project, "--agent", "codex", "--dry-run"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := requireOK(t, env.run(t, "", tc.args...))
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
