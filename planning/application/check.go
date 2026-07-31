package application

import (
	"context"
	"fmt"
	"strings"
)

type specSectionRule struct {
	heading    string
	key        string
	suggestion string
}

var requiredSpecSectionRules = []specSectionRule{
	{heading: "Problem", key: "problem", suggestion: "Add a concrete problem statement under ## Problem that explains what is broken or missing today."},
	{heading: "Goals", key: "goals", suggestion: "Expand ## Goals with the specific outcomes this spec must deliver."},
	{heading: "Non-Goals", key: "non_goals", suggestion: "Use ## Non-Goals to define what this work will explicitly not do."},
	{heading: "Constraints", key: "constraints", suggestion: "List the design or implementation limits under ## Constraints so tradeoffs stay clear."},
	{heading: "Verification", key: "verification", suggestion: "Describe how this spec will be validated under ## Verification with explicit checks or test flows."},
}

// Check evaluates deterministic local project or spec quality rules.
func (s *Service) Check(ctx context.Context, input CheckInput) (CheckReport, error) {
	if input.SpecID != nil {
		if err := input.SpecID.Validate(); err != nil {
			return CheckReport{}, err
		}
	}
	workspace, err := s.readableWorkspace(ctx)
	if err != nil {
		return CheckReport{}, err
	}
	documents, err := s.repository.QuerySpecs(ctx, input.SpecID)
	if err != nil {
		return CheckReport{}, err
	}
	scope := "project"
	if input.SpecID != nil {
		scope = "spec:" + string(*input.SpecID)
	}
	report := CheckReport{Project: workspace.Project, Scope: scope}
	for _, document := range documents {
		report.Findings = append(report.Findings, checkSpecDocument(document)...)
	}
	return report, nil
}

func checkSpecDocument(document SpecQueryDocument) []CheckFinding {
	var findings []CheckFinding
	for _, rule := range requiredSpecSectionRules {
		section := extractMarkdownSection(document.Body, rule.heading)
		switch {
		case strings.TrimSpace(section) == "":
			findings = append(findings, CheckFinding{
				Severity: "error", Rule: "spec.missing_" + rule.key,
				ArtifactType: "spec", ArtifactPath: document.Path, ArtifactTitle: document.Title,
				Section: rule.heading, Message: fmt.Sprintf("Missing required ## %s section content.", rule.heading),
				Suggestion: rule.suggestion,
			})
		case sectionLooksThin(section):
			findings = append(findings, CheckFinding{
				Severity: "warn", Rule: "spec.thin_" + rule.key,
				ArtifactType: "spec", ArtifactPath: document.Path, ArtifactTitle: document.Title,
				Section: rule.heading, Message: fmt.Sprintf("## %s is present but too thin to guide execution.", rule.heading),
				Suggestion: rule.suggestion,
			})
		}
	}
	return findings
}

func extractMarkdownSection(body, heading string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	needle := strings.ToLower(strings.TrimSpace(heading))
	inSection := false
	level := 0
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			currentLevel := 0
			for _, character := range trimmed {
				if character != '#' {
					break
				}
				currentLevel++
			}
			title := strings.ToLower(strings.TrimSpace(strings.TrimLeft(trimmed, "#")))
			if inSection && currentLevel <= level {
				break
			}
			if title == needle {
				inSection = true
				level = currentLevel
				continue
			}
		}
		if inSection {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func sectionLooksThin(content string) bool {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	meaningfulLines := 0
	totalWords := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimPrefix(trimmed, "- [ ] ")
		trimmed = strings.TrimPrefix(trimmed, "- [x] ")
		trimmed = strings.TrimPrefix(trimmed, "- ")
		trimmed = strings.TrimPrefix(trimmed, "* ")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == "" {
			continue
		}
		meaningfulLines++
		totalWords += len(strings.Fields(trimmed))
	}
	if meaningfulLines == 0 {
		return true
	}
	if meaningfulLines >= 2 && totalWords >= 6 {
		return false
	}
	return totalWords < 6
}
