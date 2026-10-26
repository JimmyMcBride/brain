package application

import (
	"regexp"
	"strings"
)

func collaborationSection(body, heading, keyword string) string {
	if section := strings.TrimSpace(extractMarkdownSection(body, heading)); section != "" {
		return section
	}
	pattern := regexp.MustCompile(`(?im)^\s*[-*]?\s*` + regexp.QuoteMeta(keyword) + `s?\s*:\s*(.+)$`)
	if match := pattern.FindStringSubmatch(body); len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func collaborationTitles(body string) ([]string, bool) {
	lower := strings.ToLower(body)
	multi := false
	for _, phrase := range []string{"create spec issues", "spec issues for:", "create specs for:", "create spec issue", "multiple specs", "multi-spec", "split into"} {
		multi = multi || strings.Contains(lower, phrase)
	}
	if section := extractMarkdownSection(body, "Promotion map"); section != "" {
		titles := []string{}
		for _, match := range regexp.MustCompile(`(?m)^###\s+Spec\s+[0-9]+\s+(?:—|-)\s+(.+?)\s*$`).FindAllStringSubmatch(section, -1) {
			titles = append(titles, match[1])
		}
		if len(titles) > 0 {
			titles = uniqueCollaborationTitles(titles)
			return titles, len(titles) > 1
		}
	}
	// Canonical repaired Specs takes precedence over older split suggestions.
	for _, heading := range []string{"Specs", "Spec Split", "Initial Spec Split", "Planned Specs"} {
		if titles := collaborationList(extractMarkdownSection(body, heading), false); len(titles) > 0 {
			return titles, multi || len(titles) > 1
		}
	}
	lines := strings.Split(body, "\n")
	phrase := regexp.MustCompile(`(?i)\b(?:create\s+)?spec\s+issues?\s+for\s*:\s*(.*)$`)
	var titles []string
	for i, line := range lines {
		match := phrase.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		if strings.TrimSpace(match[1]) != "" {
			titles = append(titles, regexp.MustCompile(`[;,]`).Split(match[1], -1)...)
			continue
		}
		var block []string
		for _, next := range lines[i+1:] {
			if strings.TrimSpace(next) == "" || strings.HasPrefix(strings.TrimSpace(next), "#") {
				break
			}
			block = append(block, next)
		}
		titles = append(titles, collaborationList(strings.Join(block, "\n"), true)...)
	}
	if len(titles) > 0 {
		return uniqueCollaborationTitles(titles), true
	}
	if multi {
		for _, heading := range []string{"Desired Outcome", "Candidate Approaches", "Ideas"} {
			if items := collaborationList(extractMarkdownSection(body, heading), true); len(items) > 0 {
				return items, true
			}
		}
		return collaborationList(body, true), true
	}
	return nil, false
}

func collaborationList(body string, numbered bool) []string {
	var titles []string
	pattern := regexp.MustCompile(`^\d+[.)]\s+(.+)$`)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			titles = append(titles, line[2:])
			continue
		}
		if numbered {
			if match := pattern.FindStringSubmatch(line); len(match) == 2 {
				titles = append(titles, match[1])
			}
		}
	}
	return uniqueCollaborationTitles(titles)
}

func uniqueCollaborationTitles(titles []string) []string {
	var result []string
	seen := map[string]bool{}
	for _, title := range titles {
		title = strings.TrimSpace(title)
		if strings.HasPrefix(title, "[ ] ") || strings.HasPrefix(strings.ToLower(title), "[x] ") {
			title = title[4:]
		}
		title = strings.Trim(title, " \t-*:;,.")
		key := strings.ToLower(title)
		if title != "" && !seen[key] {
			seen[key] = true
			result = append(result, title)
		}
	}
	return result
}
