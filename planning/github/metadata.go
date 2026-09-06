package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const metadataRelativePath = ".plan/.meta/github.json"

type githubState struct {
	Repo             string                           `json:"repo,omitempty"`
	RepoURL          string                           `json:"repo_url,omitempty"`
	DefaultBranch    string                           `json:"default_branch,omitempty"`
	LastEnabledAt    string                           `json:"last_enabled_at,omitempty"`
	LastUpdatedAt    string                           `json:"last_updated_at,omitempty"`
	LastReconciledAt string                           `json:"last_reconciled_at,omitempty"`
	Stories          map[string]githubStoryRecord     `json:"stories"`
	Planning         map[string]githubPlanningRecord  `json:"planning"`
	ProjectDecisions map[string]githubProjectDecision `json:"project_decisions"`
}

type githubPlanningRecord struct {
	Slug              string   `json:"slug"`
	Kind              string   `json:"kind"`
	Title             string   `json:"title"`
	IssueNumber       int      `json:"issue_number,omitempty"`
	IssueURL          string   `json:"issue_url,omitempty"`
	RemoteState       string   `json:"remote_state,omitempty"`
	Readiness         string   `json:"readiness,omitempty"`
	OwnershipMode     string   `json:"ownership_mode,omitempty"`
	EntryMode         string   `json:"entry_mode,omitempty"`
	SourceMode        string   `json:"source_mode,omitempty"`
	DiscussionNumber  int      `json:"discussion_number,omitempty"`
	DiscussionURL     string   `json:"discussion_url,omitempty"`
	ParentIssueNumber int      `json:"parent_issue_number,omitempty"`
	MilestoneNumber   int      `json:"milestone_number,omitempty"`
	MilestoneTitle    string   `json:"milestone_title,omitempty"`
	BlockedBy         []string `json:"blocked_by,omitempty"`
	UpdatedAt         string   `json:"updated_at,omitempty"`
}

type githubProjectDecision struct {
	Slug             string            `json:"slug"`
	Decision         string            `json:"decision"`
	Reason           string            `json:"reason,omitempty"`
	InitiativeSlug   string            `json:"initiative_slug,omitempty"`
	SpecCount        int               `json:"spec_count,omitempty"`
	MilestoneNumber  int               `json:"milestone_number,omitempty"`
	MilestoneTitle   string            `json:"milestone_title,omitempty"`
	ProjectOwner     string            `json:"project_owner,omitempty"`
	ProjectNumber    int               `json:"project_number,omitempty"`
	ProjectID        string            `json:"project_id,omitempty"`
	ProjectURL       string            `json:"project_url,omitempty"`
	FieldIDs         map[string]string `json:"field_ids,omitempty"`
	SourceMode       string            `json:"source_mode,omitempty"`
	EntryMode        string            `json:"entry_mode,omitempty"`
	DiscussionNumber int               `json:"discussion_number,omitempty"`
	DiscussionURL    string            `json:"discussion_url,omitempty"`
	UpdatedAt        string            `json:"updated_at,omitempty"`
}

type githubStoryRecord struct {
	Slug                  string   `json:"slug"`
	Title                 string   `json:"title"`
	Epic                  string   `json:"epic"`
	Spec                  string   `json:"spec"`
	Status                string   `json:"status,omitempty"`
	Description           string   `json:"description,omitempty"`
	AcceptanceCriteria    []string `json:"acceptance_criteria,omitempty"`
	Verification          []string `json:"verification,omitempty"`
	Resources             []string `json:"resources,omitempty"`
	Dependencies          []string `json:"dependencies,omitempty"`
	AsyncNotes            []string `json:"async_notes,omitempty"`
	ScopeFit              string   `json:"scope_fit,omitempty"`
	VerticalSliceCheck    string   `json:"vertical_slice_check,omitempty"`
	HiddenPrerequisites   string   `json:"hidden_prerequisites,omitempty"`
	VerificationGaps      string   `json:"verification_gaps,omitempty"`
	RewriteRecommendation string   `json:"rewrite_recommendation,omitempty"`
	IssueNumber           int      `json:"issue_number,omitempty"`
	IssueURL              string   `json:"issue_url,omitempty"`
	RemoteState           string   `json:"remote_state,omitempty"`
	PlanningPRNumber      int      `json:"planning_pr_number,omitempty"`
	PlanningPRURL         string   `json:"planning_pr_url,omitempty"`
	PlanningPRMerged      *bool    `json:"planning_pr_merged,omitempty"`
	DocRefMode            string   `json:"doc_ref_mode,omitempty"`
	DocRef                string   `json:"doc_ref,omitempty"`
	Ready                 bool     `json:"ready,omitempty"`
	BlockedReasons        []string `json:"blocked_reasons,omitempty"`
	VisibleReadyMarkerSet bool     `json:"visible_ready_marker_set,omitempty"`
	UpdatedAt             string   `json:"updated_at,omitempty"`
}

type stateStore interface {
	read() (githubState, error)
	write(githubState) error
}

type fileStateStore struct {
	path string
}

func newFileStateStore(projectRoot string) *fileStateStore {
	return &fileStateStore{path: filepath.Join(projectRoot, filepath.FromSlash(metadataRelativePath))}
}

func (s *fileStateStore) read() (githubState, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return githubState{}, fmt.Errorf("read GitHub adapter state: %w", err)
	}
	var state githubState
	if err := json.Unmarshal(raw, &state); err != nil {
		return githubState{}, fmt.Errorf("parse GitHub adapter state: %w", err)
	}
	return normalizeGitHubState(state), nil
}

func (s *fileStateStore) write(state githubState) error {
	state = normalizeGitHubState(state)
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal GitHub adapter state: %w", err)
	}
	raw = append(raw, '\n')
	if err := atomicWriteFile(s.path, raw, 0o644); err != nil {
		return fmt.Errorf("write GitHub adapter state: %w", err)
	}
	return nil
}

func normalizeGitHubState(state githubState) githubState {
	if state.Stories == nil {
		state.Stories = map[string]githubStoryRecord{}
	}
	if state.Planning == nil {
		state.Planning = map[string]githubPlanningRecord{}
	}
	if state.ProjectDecisions == nil {
		state.ProjectDecisions = map[string]githubProjectDecision{}
	}
	return state
}
