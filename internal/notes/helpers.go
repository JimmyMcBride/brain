package notes

import (
	"regexp"
	"strings"
)

var checkboxDone = regexp.MustCompile(`(?m)^\s*- \[x\] .+`)
var checkboxOpen = regexp.MustCompile(`(?m)^\s*- \[ \] .+`)
var headingPattern = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)

// AppendUnderHeading inserts text at the end of the section identified by
// heading. It matches both ## and ### headings. If the heading is not found
// it appends a new section at the end of content.
func AppendUnderHeading(content, heading, text string) string {
	lines := strings.Split(content, "\n")
	headingIdx := -1
	headingLevel := 0

	for i, line := range lines {
		m := headingPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(m[2]), strings.TrimSpace(heading)) {
			headingIdx = i
			headingLevel = len(m[1])
			break
		}
	}

	if headingIdx < 0 {
		// Heading not found — append new section at end.
		suffix := "\n## " + heading + "\n\n" + text + "\n"
		return strings.TrimRight(content, "\n") + "\n" + suffix
	}

	// Find the end of this section (next heading at same or higher level).
	insertIdx := len(lines)
	for i := headingIdx + 1; i < len(lines); i++ {
		m := headingPattern.FindStringSubmatch(lines[i])
		if m != nil && len(m[1]) <= headingLevel {
			insertIdx = i
			break
		}
	}

	// Walk backwards over trailing blank lines to insert before them but
	// still inside the section.
	for insertIdx > headingIdx+1 && strings.TrimSpace(lines[insertIdx-1]) == "" {
		insertIdx--
	}

	// If the section is completely empty (only the heading), add a blank line first.
	if insertIdx == headingIdx+1 {
		newLines := make([]string, 0, len(lines)+3)
		newLines = append(newLines, lines[:insertIdx]...)
		newLines = append(newLines, "", text, "")
		newLines = append(newLines, lines[insertIdx:]...)
		return strings.Join(newLines, "\n")
	}

	newLines := make([]string, 0, len(lines)+2)
	newLines = append(newLines, lines[:insertIdx]...)
	newLines = append(newLines, text, "")
	newLines = append(newLines, lines[insertIdx:]...)
	return strings.Join(newLines, "\n")
}

// ParseCheckboxes counts total and done checkboxes in content.
func ParseCheckboxes(content string) (total, done int) {
	done = len(checkboxDone.FindAllString(content, -1))
	open := len(checkboxOpen.FindAllString(content, -1))
	total = done + open
	return total, done
}

// MilestoneStatus holds checkbox counts for a single milestone.
type MilestoneStatus struct {
	Name  string `json:"name"`
	Total int    `json:"total_tasks"`
	Done  int    `json:"done_tasks"`
}

// ParseMilestoneCheckboxes groups checkbox counts by ### milestone headings.
// Tasks not under any ### heading are grouped under "".
func ParseMilestoneCheckboxes(content string) []MilestoneStatus {
	lines := strings.Split(content, "\n")
	var milestones []MilestoneStatus
	current := ""
	counts := map[string][2]int{} // [total, done]
	var order []string

	for _, line := range lines {
		m := headingPattern.FindStringSubmatch(line)
		if m != nil && len(m[1]) == 3 {
			current = strings.TrimSpace(m[2])
			if _, exists := counts[current]; !exists {
				order = append(order, current)
				counts[current] = [2]int{}
			}
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- [x] ") {
			c := counts[current]
			c[0]++
			c[1]++
			counts[current] = c
		} else if strings.HasPrefix(trimmed, "- [ ] ") {
			c := counts[current]
			c[0]++
			counts[current] = c
		}
	}

	for _, name := range order {
		c := counts[name]
		if c[0] > 0 {
			milestones = append(milestones, MilestoneStatus{
				Name:  name,
				Total: c[0],
				Done:  c[1],
			})
		}
	}
	return milestones
}
