package notes

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func ParseFrontmatter(raw string) (map[string]any, string, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return map[string]any{}, raw, nil
	}
	rest := strings.TrimPrefix(raw, "---\n")
	parts := strings.SplitN(rest, "\n---\n", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("unterminated frontmatter")
	}
	meta := map[string]any{}
	if strings.TrimSpace(parts[0]) != "" {
		if err := yaml.Unmarshal([]byte(parts[0]), &meta); err != nil {
			return nil, "", fmt.Errorf("parse frontmatter: %w", err)
		}
	}
	return meta, parts[1], nil
}

func HasFrontmatter(raw string) bool {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	return strings.HasPrefix(raw, "---\n")
}

func HasNestedFrontmatter(body string) bool {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.TrimLeft(body, "\n")
	return strings.HasPrefix(body, "---\n")
}

func ComposeFrontmatter(meta map[string]any, body string) (string, error) {
	if meta == nil {
		meta = map[string]any{}
	}
	body = strings.TrimLeft(body, "\n")
	if len(meta) == 0 {
		return strings.TrimRight(body, "\n") + "\n", nil
	}
	out, err := yaml.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}
	return "---\n" + string(out) + "---\n" + strings.TrimRight(body, "\n") + "\n", nil
}
