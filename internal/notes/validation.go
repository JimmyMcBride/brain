package notes

import (
	"fmt"
	"os"

	"brain/internal/workspace"
)

type RawNote struct {
	Metadata map[string]any
	Body     string
}

func ReadRaw(path string) (*RawNote, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read raw note: %w", err)
	}
	meta, body, err := ParseFrontmatter(string(raw))
	if err != nil {
		return nil, err
	}
	return &RawNote{Metadata: meta, Body: body}, nil
}

func ValidateWorkspaceMarkdown(workspaceSvc *workspace.Service) (int, error) {
	files, err := workspaceSvc.WalkMarkdownFiles()
	if err != nil {
		return 0, err
	}
	for _, file := range files {
		rel, err := workspaceSvc.Rel(file)
		if err != nil {
			return 0, err
		}
		note, err := ReadRaw(file)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", rel, err)
		}
		if HasNestedFrontmatter(note.Body) {
			return 0, fmt.Errorf("%s: nested frontmatter block at body start", rel)
		}
	}
	return len(files), nil
}
