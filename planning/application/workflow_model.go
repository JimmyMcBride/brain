package application

import "github.com/JimmyMcBride/brain/planning"

const (
	// EventBrainstormUpdated identifies a brainstorm-content mutation.
	EventBrainstormUpdated = "planning.brainstorm.updated"
	// EventGuidedSessionUpdated identifies a guided-session mutation.
	EventGuidedSessionUpdated = "planning.guided-session.updated"
	// EventBrainstormPromoted identifies direct local spec promotion.
	EventBrainstormPromoted = "planning.brainstorm.promoted"
)

// BrainstormUpdateInput appends one entry to a supported brainstorm section.
type BrainstormUpdateInput struct {
	ID        planning.ArtifactID `json:"id"`
	Section   string              `json:"section"`
	Body      string              `json:"body"`
	Confirmed bool                `json:"confirmed"`
}

// BrainstormRefinementInput replaces supplied refinement fields and preserves omitted fields.
type BrainstormRefinementInput struct {
	ID                     planning.ArtifactID `json:"id"`
	Problem                string              `json:"problem,omitempty"`
	UserValue              string              `json:"user_value,omitempty"`
	Constraints            string              `json:"constraints,omitempty"`
	Appetite               string              `json:"appetite,omitempty"`
	RemainingOpenQuestions string              `json:"remaining_open_questions,omitempty"`
	CandidateApproaches    string              `json:"candidate_approaches,omitempty"`
	DecisionSnapshot       string              `json:"decision_snapshot,omitempty"`
	Confirmed              bool                `json:"confirmed"`
}

// BrainstormChallengeInput replaces supplied challenge fields and preserves omitted fields.
type BrainstormChallengeInput struct {
	ID                    planning.ArtifactID `json:"id"`
	RabbitHoles           string              `json:"rabbit_holes,omitempty"`
	NoGos                 string              `json:"no_gos,omitempty"`
	Assumptions           string              `json:"assumptions,omitempty"`
	LikelyOverengineering string              `json:"likely_overengineering,omitempty"`
	SimplerAlternative    string              `json:"simpler_alternative,omitempty"`
	Confirmed             bool                `json:"confirmed"`
}

// BrainstormMutationResult describes a previewed, applied, or unchanged brainstorm update.
type BrainstormMutationResult struct {
	Action   MutationAction     `json:"action"`
	Document BrainstormDocument `json:"document"`
	Event    *Event             `json:"event,omitempty"`
}

// GuidedSessionState is the schema-v3 local guided-session document.
type GuidedSessionState struct {
	SchemaVersion   int                            `json:"schema_version"`
	LastActiveChain string                         `json:"last_active_chain,omitempty"`
	LastUpdatedAt   string                         `json:"last_updated_at,omitempty"`
	Sessions        map[string]GuidedSessionRecord `json:"sessions"`
}

// GuidedSessionRecord is one resumable Planning workflow chain.
type GuidedSessionRecord struct {
	ChainID             string            `json:"chain_id"`
	Brainstorm          string            `json:"brainstorm,omitempty"`
	Spec                string            `json:"spec,omitempty"`
	CurrentStage        string            `json:"current_stage,omitempty"`
	CurrentCluster      int               `json:"current_cluster,omitempty"`
	CurrentClusterLabel string            `json:"current_cluster_label,omitempty"`
	StageStatuses       map[string]string `json:"stage_statuses,omitempty"`
	Summary             string            `json:"summary,omitempty"`
	NextAction          string            `json:"next_action,omitempty"`
	CreatedAt           string            `json:"created_at,omitempty"`
	UpdatedAt           string            `json:"updated_at,omitempty"`
}

// GuidedSessionMutationInput selects a chain/stage and confirmation policy.
type GuidedSessionMutationInput struct {
	ChainID   string `json:"chain_id"`
	Stage     string `json:"stage,omitempty"`
	Confirmed bool   `json:"confirmed"`
}

// GuidedSessionResult describes a guided-session mutation.
type GuidedSessionResult struct {
	Action   MutationAction      `json:"action"`
	Session  GuidedSessionRecord `json:"session"`
	Impacted []string            `json:"impacted,omitempty"`
	Event    *Event              `json:"event,omitempty"`
}

// RoadmapParkingInput describes a deferred idea and its unlock condition.
type RoadmapParkingInput struct {
	BrainstormID planning.ArtifactID `json:"brainstorm_id"`
	Title        string              `json:"title"`
	Value        string              `json:"value,omitempty"`
	Reason       string              `json:"reason,omitempty"`
	Unlock       string              `json:"unlock,omitempty"`
	Confirmed    bool                `json:"confirmed"`
}

// GuidePacket is the versioned local agent-consumption contract.
type GuidePacket struct {
	SchemaVersion  int            `json:"schema_version"`
	Kind           string         `json:"kind"`
	GeneratedAt    string         `json:"generated_at"`
	Builder        map[string]any `json:"builder"`
	Workspace      map[string]any `json:"workspace"`
	Ownership      map[string]any `json:"ownership"`
	Session        map[string]any `json:"session"`
	Artifact       map[string]any `json:"artifact"`
	Mode           map[string]any `json:"mode"`
	Sources        []string       `json:"sources"`
	Contract       map[string]any `json:"contract"`
	RenderedPrompt string         `json:"rendered_prompt"`
}

// LocalPromotionInput selects a local brainstorm and confirmation policy.
type LocalPromotionInput struct {
	BrainstormID planning.ArtifactID `json:"brainstorm_id"`
	Confirmed    bool                `json:"confirmed"`
}

// LocalPromotionRepairInput replaces the brainstorm Specs section.
type LocalPromotionRepairInput struct {
	BrainstormID planning.ArtifactID `json:"brainstorm_id"`
	Specs        []string            `json:"specs"`
	Confirmed    bool                `json:"confirmed"`
}

// LocalPromotionRepairResult describes a local source repair.
type LocalPromotionRepairResult struct {
	SchemaVersion int            `json:"schema_version"`
	Kind          string         `json:"kind"`
	GeneratedAt   string         `json:"generated_at"`
	Source        map[string]any `json:"source"`
	Specs         []string       `json:"specs"`
	UpdatedPath   string         `json:"updated_path,omitempty"`
	NextCommands  []string       `json:"next_commands"`
	Action        MutationAction `json:"action"`
	Event         *Event         `json:"event,omitempty"`
}

// MaturityDecision is the stable local brainstorm readiness decision.
type MaturityDecision struct {
	State           string           `json:"state"`
	Confidence      string           `json:"confidence"`
	SourceMode      string           `json:"source_mode"`
	Reason          string           `json:"reason"`
	Strengths       []string         `json:"strengths,omitempty"`
	Gaps            []string         `json:"gaps,omitempty"`
	RecommendedPath string           `json:"recommended_path,omitempty"`
	SuggestedTitles map[string]any   `json:"suggested_titles"`
	DependencyGuess []map[string]any `json:"dependency_guess,omitempty"`
}

// CollaborationAssessment is the versioned local maturity response.
type CollaborationAssessment struct {
	SchemaVersion int              `json:"schema_version"`
	Kind          string           `json:"kind"`
	GeneratedAt   string           `json:"generated_at"`
	Source        map[string]any   `json:"source"`
	Ownership     map[string]any   `json:"ownership"`
	Decision      MaturityDecision `json:"maturity_decision"`
}

// PromotionSpecDraft is one direct local spec mutation plan.
type PromotionSpecDraft struct {
	Kind           string         `json:"kind"`
	Title          string         `json:"title"`
	Body           string         `json:"body"`
	Slug           string         `json:"slug"`
	Action         MutationAction `json:"action"`
	Readiness      string         `json:"readiness"`
	Labels         []string       `json:"labels,omitempty"`
	SourceLinks    []string       `json:"source_links,omitempty"`
	BlockedBy      []string       `json:"blocked_by,omitempty"`
	ReadyByDefault bool           `json:"ready_by_default"`
}

// LocalPromotionDraft is the versioned direct-spec promotion preview.
type LocalPromotionDraft struct {
	SchemaVersion         int                  `json:"schema_version"`
	Kind                  string               `json:"kind"`
	GeneratedAt           string               `json:"generated_at"`
	Source                map[string]any       `json:"source"`
	Ownership             map[string]any       `json:"ownership"`
	Assessment            MaturityDecision     `json:"assessment"`
	PromotionDecision     string               `json:"promotion_decision,omitempty"`
	WhyThisPath           string               `json:"why_this_path,omitempty"`
	ProposedSpecs         []PromotionSpecDraft `json:"proposed_spec_issues"`
	AgentPolicy           map[string]any       `json:"agent_policy"`
	ManualFallbackAllowed bool                 `json:"manual_fallback_allowed"`
	ConfirmationRequired  bool                 `json:"confirmation_required"`
}

// LocalPromotionResult describes direct local spec writes.
type LocalPromotionResult struct {
	Draft  LocalPromotionDraft `json:"draft"`
	Specs  []SpecDocument      `json:"specs"`
	Action MutationAction      `json:"action"`
	Event  *Event              `json:"event,omitempty"`
}

// PromotionSpecWrite is the local repository write contract.
type PromotionSpecWrite struct {
	Artifact planning.Spec  `json:"artifact"`
	Body     string         `json:"body"`
	Metadata map[string]any `json:"metadata"`
}
