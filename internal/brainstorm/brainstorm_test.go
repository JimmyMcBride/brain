package brainstorm

import "testing"

func TestExtractSection(t *testing.T) {
	content := `## Focus Question

What is the topic?

## Ideas

- **10:00** first idea
- **10:05** second idea

## Related
`
	got := extractSection(content, "Ideas")
	if got == "" {
		t.Fatal("expected non-empty Ideas section")
	}
	if want := "- **10:00** first idea"; !contains(got, want) {
		t.Errorf("expected %q in result, got:\n%s", want, got)
	}
	if want := "- **10:05** second idea"; !contains(got, want) {
		t.Errorf("expected %q in result, got:\n%s", want, got)
	}
}

func TestExtractSection_Missing(t *testing.T) {
	got := extractSection("# Title\n\n## Notes\n", "Ideas")
	if got != "" {
		t.Errorf("expected empty string for missing section, got %q", got)
	}
}

func TestExtractSection_CaseInsensitive(t *testing.T) {
	content := "## ideas\n\n- item\n\n## Other\n"
	got := extractSection(content, "Ideas")
	if got != "- item" {
		t.Errorf("expected '- item', got %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
