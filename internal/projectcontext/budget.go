package projectcontext

import (
	"regexp"
	"strings"
)

const (
	CompileBudgetPresetDefault = "default"
	CompileBudgetPresetSmall   = "small"
	CompileBudgetPresetLarge   = "large"
)

type CompileBudget struct {
	Preset string `json:"preset,omitempty"`
	Target int    `json:"target"`
}

type BudgetOmittedItem struct {
	ID              string        `json:"id"`
	Section         string        `json:"section"`
	Title           string        `json:"title"`
	Anchor          ContextAnchor `json:"anchor"`
	EstimatedTokens int           `json:"estimated_tokens"`
	Reason          string        `json:"reason"`
}

type PacketBudget struct {
	Preset              string              `json:"preset,omitempty"`
	Target              int                 `json:"target"`
	Used                int                 `json:"used"`
	Remaining           int                 `json:"remaining"`
	ReserveBaseContract int                 `json:"reserve_base_contract"`
	ReserveVerification int                 `json:"reserve_verification"`
	ReserveDiagnostics  int                 `json:"reserve_diagnostics"`
	OmittedDueToBudget  int                 `json:"omitted_due_to_budget"`
	OmittedCandidates   []BudgetOmittedItem `json:"omitted_candidates,omitempty"`
	MandatoryOverTarget bool                `json:"mandatory_over_target,omitempty"`
}

var estimateTokenPattern = regexp.MustCompile(`[[:alnum:]_]+|[^\s[:alnum:]_]`)

func EstimateTokens(parts ...string) int {
	text := strings.TrimSpace(strings.Join(parts, " "))
	if text == "" {
		return 0
	}
	return len(estimateTokenPattern.FindAllString(text, -1))
}
