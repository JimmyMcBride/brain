package index

import "testing"

func TestSplitMarkdownByHeadings(t *testing.T) {
	content := `Intro paragraph

# First
alpha

## Second
beta
`

	chunks := SplitMarkdownByHeadings(content)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if chunks[0].Heading != "" || chunks[0].Content != "Intro paragraph" {
		t.Fatalf("unexpected intro chunk: %+v", chunks[0])
	}
	if chunks[1].Heading != "First" || chunks[1].Content != "alpha" {
		t.Fatalf("unexpected first heading chunk: %+v", chunks[1])
	}
	if chunks[2].Heading != "Second" || chunks[2].Content != "beta" {
		t.Fatalf("unexpected second heading chunk: %+v", chunks[2])
	}
}
