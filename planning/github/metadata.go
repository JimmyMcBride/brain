package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
	Extra            map[string]json.RawMessage       `json:"-"`
}

type githubPlanningRecord struct {
	Slug              string                     `json:"slug"`
	Kind              string                     `json:"kind"`
	Title             string                     `json:"title"`
	IssueNumber       int                        `json:"issue_number,omitempty"`
	IssueURL          string                     `json:"issue_url,omitempty"`
	RemoteState       string                     `json:"remote_state,omitempty"`
	Readiness         string                     `json:"readiness,omitempty"`
	OwnershipMode     string                     `json:"ownership_mode,omitempty"`
	EntryMode         string                     `json:"entry_mode,omitempty"`
	SourceMode        string                     `json:"source_mode,omitempty"`
	DiscussionNumber  int                        `json:"discussion_number,omitempty"`
	DiscussionURL     string                     `json:"discussion_url,omitempty"`
	ParentIssueNumber int                        `json:"parent_issue_number,omitempty"`
	MilestoneNumber   int                        `json:"milestone_number,omitempty"`
	MilestoneTitle    string                     `json:"milestone_title,omitempty"`
	BlockedBy         []string                   `json:"blocked_by,omitempty"`
	UpdatedAt         string                     `json:"updated_at,omitempty"`
	Extra             map[string]json.RawMessage `json:"-"`
}

type githubProjectDecision struct {
	Slug             string                     `json:"slug"`
	Decision         string                     `json:"decision"`
	Reason           string                     `json:"reason,omitempty"`
	InitiativeSlug   string                     `json:"initiative_slug,omitempty"`
	SpecCount        int                        `json:"spec_count,omitempty"`
	MilestoneNumber  int                        `json:"milestone_number,omitempty"`
	MilestoneTitle   string                     `json:"milestone_title,omitempty"`
	ProjectOwner     string                     `json:"project_owner,omitempty"`
	ProjectNumber    int                        `json:"project_number,omitempty"`
	ProjectID        string                     `json:"project_id,omitempty"`
	ProjectURL       string                     `json:"project_url,omitempty"`
	FieldIDs         map[string]string          `json:"field_ids,omitempty"`
	SourceMode       string                     `json:"source_mode,omitempty"`
	EntryMode        string                     `json:"entry_mode,omitempty"`
	DiscussionNumber int                        `json:"discussion_number,omitempty"`
	DiscussionURL    string                     `json:"discussion_url,omitempty"`
	UpdatedAt        string                     `json:"updated_at,omitempty"`
	Extra            map[string]json.RawMessage `json:"-"`
}

type githubStoryRecord struct {
	Slug                  string                     `json:"slug"`
	Title                 string                     `json:"title"`
	Epic                  string                     `json:"epic"`
	Spec                  string                     `json:"spec"`
	Status                string                     `json:"status,omitempty"`
	Description           string                     `json:"description,omitempty"`
	AcceptanceCriteria    []string                   `json:"acceptance_criteria,omitempty"`
	Verification          []string                   `json:"verification,omitempty"`
	Resources             []string                   `json:"resources,omitempty"`
	Dependencies          []string                   `json:"dependencies,omitempty"`
	AsyncNotes            []string                   `json:"async_notes,omitempty"`
	ScopeFit              string                     `json:"scope_fit,omitempty"`
	VerticalSliceCheck    string                     `json:"vertical_slice_check,omitempty"`
	HiddenPrerequisites   string                     `json:"hidden_prerequisites,omitempty"`
	VerificationGaps      string                     `json:"verification_gaps,omitempty"`
	RewriteRecommendation string                     `json:"rewrite_recommendation,omitempty"`
	IssueNumber           int                        `json:"issue_number,omitempty"`
	IssueURL              string                     `json:"issue_url,omitempty"`
	RemoteState           string                     `json:"remote_state,omitempty"`
	PlanningPRNumber      int                        `json:"planning_pr_number,omitempty"`
	PlanningPRURL         string                     `json:"planning_pr_url,omitempty"`
	PlanningPRMerged      *bool                      `json:"planning_pr_merged,omitempty"`
	DocRefMode            string                     `json:"doc_ref_mode,omitempty"`
	DocRef                string                     `json:"doc_ref,omitempty"`
	Ready                 bool                       `json:"ready,omitempty"`
	BlockedReasons        []string                   `json:"blocked_reasons,omitempty"`
	VisibleReadyMarkerSet bool                       `json:"visible_ready_marker_set,omitempty"`
	UpdatedAt             string                     `json:"updated_at,omitempty"`
	Extra                 map[string]json.RawMessage `json:"-"`
}

func (state *githubState) UnmarshalJSON(raw []byte) error {
	type plain githubState
	var value plain
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	*state = githubState(value)
	state.Extra = unknownJSONFields(raw, "repo", "repo_url", "default_branch", "last_enabled_at", "last_updated_at", "last_reconciled_at", "stories", "planning", "project_decisions")
	return nil
}

func (state githubState) MarshalJSON() ([]byte, error) {
	type plain githubState
	return marshalWithExtra(plain(state), state.Extra)
}

func (record *githubPlanningRecord) UnmarshalJSON(raw []byte) error {
	type plain githubPlanningRecord
	var value plain
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	*record = githubPlanningRecord(value)
	record.Extra = unknownJSONFields(raw, "slug", "kind", "title", "issue_number", "issue_url", "remote_state", "readiness", "ownership_mode", "entry_mode", "source_mode", "discussion_number", "discussion_url", "parent_issue_number", "milestone_number", "milestone_title", "blocked_by", "updated_at")
	return nil
}

func (record githubPlanningRecord) MarshalJSON() ([]byte, error) {
	type plain githubPlanningRecord
	return marshalWithExtra(plain(record), record.Extra)
}

func (record *githubProjectDecision) UnmarshalJSON(raw []byte) error {
	type plain githubProjectDecision
	var value plain
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	*record = githubProjectDecision(value)
	record.Extra = unknownJSONFields(raw, "slug", "decision", "reason", "initiative_slug", "spec_count", "milestone_number", "milestone_title", "project_owner", "project_number", "project_id", "project_url", "field_ids", "source_mode", "entry_mode", "discussion_number", "discussion_url", "updated_at")
	return nil
}

func (record githubProjectDecision) MarshalJSON() ([]byte, error) {
	type plain githubProjectDecision
	return marshalWithExtra(plain(record), record.Extra)
}

func (record *githubStoryRecord) UnmarshalJSON(raw []byte) error {
	type plain githubStoryRecord
	var value plain
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	*record = githubStoryRecord(value)
	record.Extra = unknownJSONFields(raw, "slug", "title", "epic", "spec", "status", "description", "acceptance_criteria", "verification", "resources", "dependencies", "async_notes", "scope_fit", "vertical_slice_check", "hidden_prerequisites", "verification_gaps", "rewrite_recommendation", "issue_number", "issue_url", "remote_state", "planning_pr_number", "planning_pr_url", "planning_pr_merged", "doc_ref_mode", "doc_ref", "ready", "blocked_reasons", "visible_ready_marker_set", "updated_at")
	return nil
}

func (record githubStoryRecord) MarshalJSON() ([]byte, error) {
	type plain githubStoryRecord
	return marshalWithExtra(plain(record), record.Extra)
}

func unknownJSONFields(raw []byte, known ...string) map[string]json.RawMessage {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return nil
	}
	for _, key := range known {
		delete(fields, key)
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func marshalWithExtra(value any, extra map[string]json.RawMessage) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil || len(extra) == 0 {
		return raw, err
	}
	keys := make([]string, 0, len(extra))
	for key := range extra {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var suffix bytes.Buffer
	for _, key := range keys {
		encodedKey, err := json.Marshal(key)
		if err != nil || !json.Valid(extra[key]) {
			return nil, fmt.Errorf("marshal GitHub adapter metadata extension %q", key)
		}
		suffix.WriteByte(',')
		suffix.Write(encodedKey)
		suffix.WriteByte(':')
		suffix.Write(extra[key])
	}
	return append(append(raw[:len(raw)-1], suffix.Bytes()...), '}'), nil
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

func encodeGitHubState(state githubState) ([]byte, error) {
	state = normalizeGitHubState(state)
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal GitHub adapter state: %w", err)
	}
	return append(raw, '\n'), nil
}

func (s *fileStateStore) write(state githubState) error {
	raw, err := encodeGitHubState(state)
	if err != nil {
		return err
	}
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
