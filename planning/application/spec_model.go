package application

import (
	"github.com/JimmyMcBride/brain/planning"
)

// SpecEditInput replaces canonical spec Markdown after explicit confirmation.
type SpecEditInput struct {
	ID        planning.ArtifactID `json:"id"`
	Body      string              `json:"body"`
	Confirmed bool                `json:"confirmed"`
}

// SpecStatusInput changes one local spec lifecycle state.
type SpecStatusInput struct {
	ID        planning.ArtifactID `json:"id"`
	Status    planning.SpecStatus `json:"status"`
	Confirmed bool                `json:"confirmed"`
}

// SpecInitiativeInput sets or clears lightweight initiative metadata.
type SpecInitiativeInput struct {
	ID         planning.ArtifactID  `json:"id"`
	Initiative *planning.ArtifactID `json:"initiative,omitempty"`
	Title      string               `json:"title,omitempty"`
	Summary    string               `json:"summary,omitempty"`
	Clear      bool                 `json:"clear,omitempty"`
	Confirmed  bool                 `json:"confirmed"`
}

// SpecMutationResult describes a previewed, applied, or unchanged spec mutation.
type SpecMutationResult struct {
	Action   MutationAction `json:"action"`
	Document SpecDocument   `json:"document"`
	Event    *Event         `json:"event,omitempty"`
}

// SpecAnalysisFinding is one categorized refinement finding.
type SpecAnalysisFinding struct {
	Severity       string `json:"severity"`
	Category       string `json:"category"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation,omitempty"`
}

// SpecAnalysisReport pressure-tests a spec and retains the resulting additive report.
type SpecAnalysisReport struct {
	SpecPath string                `json:"spec_path"`
	Findings []SpecAnalysisFinding `json:"findings"`
	Action   MutationAction        `json:"action"`
	Document SpecDocument          `json:"document"`
	Event    *Event                `json:"event,omitempty"`
}

// ErrorCount returns blocking analysis findings.
func (r SpecAnalysisReport) ErrorCount() int { return countSpecSeverities(r.Findings, "error") }

// WarningCount returns guidance analysis findings.
func (r SpecAnalysisReport) WarningCount() int { return countSpecSeverities(r.Findings, "warn") }

// SpecChecklistFinding is one profile-specific readiness finding.
type SpecChecklistFinding struct {
	Severity       string `json:"severity"`
	Area           string `json:"area"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation,omitempty"`
}

// SpecChecklistReport contains one additive checklist pass.
type SpecChecklistReport struct {
	SpecPath string                 `json:"spec_path"`
	Profile  string                 `json:"profile"`
	Findings []SpecChecklistFinding `json:"findings"`
	Action   MutationAction         `json:"action"`
	Document SpecDocument           `json:"document"`
	Event    *Event                 `json:"event,omitempty"`
}

// ErrorCount returns blocking checklist findings.
func (r SpecChecklistReport) ErrorCount() int {
	count := 0
	for _, f := range r.Findings {
		if f.Severity == "error" {
			count++
		}
	}
	return count
}

// WarningCount returns guidance checklist findings.
func (r SpecChecklistReport) WarningCount() int {
	count := 0
	for _, f := range r.Findings {
		if f.Severity == "warn" {
			count++
		}
	}
	return count
}

// SpecExecutionInput starts or previews deterministic ephemeral execution slices.
type SpecExecutionInput struct {
	ID           planning.ArtifactID `json:"id"`
	BranchPrefix string              `json:"branch_prefix,omitempty"`
	Confirmed    bool                `json:"confirmed"`
}

// SpecExecutionView is the stable CLI-facing execution projection.
type SpecExecutionView struct {
	SpecPath        string               `json:"spec_path"`
	Status          planning.SpecStatus  `json:"status"`
	SuggestedBranch string               `json:"suggested_branch"`
	Slices          []SpecExecutionSlice `json:"slices"`
}

// SpecExecutionSlice is the stable JSON projection of one ephemeral runtime slice.
type SpecExecutionSlice struct {
	ID           planning.ArtifactID `json:"id"`
	Title        string              `json:"title"`
	Goal         string              `json:"goal"`
	Verification []string            `json:"verification"`
	Position     int                 `json:"position"`
}

// SpecExecutionResult describes execution start and optional guided handoff.
type SpecExecutionResult struct {
	Action    MutationAction       `json:"action"`
	Document  SpecDocument         `json:"document"`
	Execution SpecExecutionView    `json:"execution"`
	Recap     string               `json:"recap,omitempty"`
	Session   *GuidedSessionRecord `json:"session,omitempty"`
	Event     *Event               `json:"event,omitempty"`
}

func countSpecSeverities(findings []SpecAnalysisFinding, severity string) int {
	count := 0
	for _, finding := range findings {
		if finding.Severity == severity {
			count++
		}
	}
	return count
}
