package notes

import (
	"strings"
	"testing"
)

func TestAppendUnderHeading_ExistingSection(t *testing.T) {
	content := "# Title\n\n## Ideas\n\n## Notes\n"
	result := AppendUnderHeading(content, "Ideas", "- new idea")
	if !strings.Contains(result, "## Ideas\n\n- new idea\n") {
		t.Fatalf("expected idea appended under Ideas heading, got:\n%s", result)
	}
	if idx := strings.Index(result, "## Notes"); idx < 0 {
		t.Fatal("Notes heading should still exist")
	}
}

func TestAppendUnderHeading_ExistingContent(t *testing.T) {
	content := "# Title\n\n## Ideas\n\n- old idea\n\n## Notes\n"
	result := AppendUnderHeading(content, "Ideas", "- new idea")
	if !strings.Contains(result, "- old idea\n- new idea\n") {
		t.Fatalf("expected new idea after old idea, got:\n%s", result)
	}
}

func TestAppendUnderHeading_MissingHeading(t *testing.T) {
	content := "# Title\n\n## Notes\n"
	result := AppendUnderHeading(content, "Ideas", "- new idea")
	if !strings.Contains(result, "## Ideas\n\n- new idea\n") {
		t.Fatalf("expected new Ideas section appended, got:\n%s", result)
	}
}

func TestAppendUnderHeading_H3(t *testing.T) {
	content := "## Milestones\n\n### Alpha\n\n- existing\n\n### Beta\n"
	result := AppendUnderHeading(content, "Alpha", "- new task")
	if !strings.Contains(result, "- existing\n- new task\n") {
		t.Fatalf("expected task under Alpha h3, got:\n%s", result)
	}
}

func TestAppendUnderHeading_CaseInsensitive(t *testing.T) {
	content := "## ideas\n\n## Notes\n"
	result := AppendUnderHeading(content, "Ideas", "- item")
	if !strings.Contains(result, "## ideas\n\n- item\n") {
		t.Fatalf("expected case-insensitive match, got:\n%s", result)
	}
}

func TestParseCheckboxes(t *testing.T) {
	content := `## Tasks

- [x] done task
- [ ] open task
- [x] another done
- [ ] another open
- [ ] third open
`
	total, done := ParseCheckboxes(content)
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if done != 2 {
		t.Errorf("expected done 2, got %d", done)
	}
}

func TestParseCheckboxes_Empty(t *testing.T) {
	total, done := ParseCheckboxes("# Plan\n\n## Tasks\n")
	if total != 0 || done != 0 {
		t.Errorf("expected 0/0, got %d/%d", total, done)
	}
}

func TestParseMilestoneCheckboxes(t *testing.T) {
	content := `## Milestones

### Alpha

- [x] task 1
- [ ] task 2

### Beta

- [ ] task 3
- [x] task 4
- [x] task 5
`
	milestones := ParseMilestoneCheckboxes(content)
	if len(milestones) != 2 {
		t.Fatalf("expected 2 milestones, got %d", len(milestones))
	}
	if milestones[0].Name != "Alpha" || milestones[0].Total != 2 || milestones[0].Done != 1 {
		t.Errorf("Alpha: expected 2/1, got %d/%d", milestones[0].Total, milestones[0].Done)
	}
	if milestones[1].Name != "Beta" || milestones[1].Total != 3 || milestones[1].Done != 2 {
		t.Errorf("Beta: expected 3/2, got %d/%d", milestones[1].Total, milestones[1].Done)
	}
}
