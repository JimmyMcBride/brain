package index

import (
	"bufio"
	"strings"
)

type Chunk struct {
	Heading string `json:"heading"`
	Content string `json:"content"`
	Index   int    `json:"index"`
}

func SplitMarkdownByHeadings(content string) []Chunk {
	scanner := bufio.NewScanner(strings.NewReader(strings.ReplaceAll(content, "\r\n", "\n")))
	var chunks []Chunk
	currentHeading := ""
	var current []string
	flush := func() {
		body := strings.TrimSpace(strings.Join(current, "\n"))
		if body == "" && currentHeading == "" {
			return
		}
		chunks = append(chunks, Chunk{
			Heading: currentHeading,
			Content: body,
			Index:   len(chunks),
		})
		current = nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if heading, ok := parseHeading(line); ok {
			flush()
			currentHeading = heading
			continue
		}
		current = append(current, line)
	}
	flush()
	if len(chunks) == 0 {
		chunks = append(chunks, Chunk{Heading: "", Content: strings.TrimSpace(content), Index: 0})
	}
	return chunks
}

func parseHeading(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimLeft(line, "#")), true
}
