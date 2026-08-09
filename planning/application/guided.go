package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/JimmyMcBride/brain/planning"
)

var guidedStages = []string{"brainstorm", "spec", "execution"}

// ListGuidedSessions returns guided sessions in stable chain order.
func (s *Service) ListGuidedSessions(ctx context.Context) ([]GuidedSessionRecord, error) {
	if err := s.requireReadable(ctx); err != nil {
		return nil, err
	}
	state, err := s.repository.ReadGuidedSessions(ctx)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(state.Sessions))
	for key := range state.Sessions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]GuidedSessionRecord, 0, len(keys))
	for _, key := range keys {
		out = append(out, state.Sessions[key])
	}
	return out, nil
}

func (s *Service) createGuidedSession(ctx context.Context, document BrainstormDocument, now time.Time) error {
	state, err := s.repository.ReadGuidedSessions(ctx)
	if err != nil {
		return err
	}
	chainID := "brainstorm/" + string(document.Artifact.ID)
	if _, exists := state.Sessions[chainID]; exists {
		return nil
	}
	timestamp := now.UTC().Format(time.RFC3339)
	record := GuidedSessionRecord{
		ChainID: chainID, Brainstorm: string(document.Artifact.ID), CurrentStage: "brainstorm",
		CurrentCluster: 1, CurrentClusterLabel: "vision-intake",
		StageStatuses: map[string]string{"brainstorm": "in_progress"},
		Summary:       "Vision still missing. No supporting material recorded yet.",
		NextAction:    "Capture the user's vision in plain language.", CreatedAt: timestamp, UpdatedAt: timestamp,
	}
	state.LastActiveChain = chainID
	state.LastUpdatedAt = timestamp
	state.Sessions[chainID] = record
	_, _, err = s.repository.ReplaceGuidedSessions(ctx, state)
	return err
}

// CurrentGuidedSession returns the last-active guided session.
func (s *Service) CurrentGuidedSession(ctx context.Context) (GuidedSessionRecord, error) {
	if err := s.requireReadable(ctx); err != nil {
		return GuidedSessionRecord{}, err
	}
	state, err := s.repository.ReadGuidedSessions(ctx)
	if err != nil {
		return GuidedSessionRecord{}, err
	}
	if strings.TrimSpace(state.LastActiveChain) == "" {
		return GuidedSessionRecord{}, fmt.Errorf("no active guided session")
	}
	record, ok := state.Sessions[state.LastActiveChain]
	if !ok {
		return GuidedSessionRecord{}, fmt.Errorf("guided session %q not found", state.LastActiveChain)
	}
	return record, nil
}

// GetGuidedSession returns one guided session by chain id.
func (s *Service) GetGuidedSession(ctx context.Context, chainID string) (GuidedSessionRecord, error) {
	if err := s.requireReadable(ctx); err != nil {
		return GuidedSessionRecord{}, err
	}
	state, err := s.repository.ReadGuidedSessions(ctx)
	if err != nil {
		return GuidedSessionRecord{}, err
	}
	chainID = normalizeChainID(chainID)
	record, ok := state.Sessions[chainID]
	if !ok {
		return GuidedSessionRecord{}, fmt.Errorf("guided session %q not found", chainID)
	}
	return record, nil
}

// SwitchGuidedSession changes the last-active chain after confirmation.
func (s *Service) SwitchGuidedSession(ctx context.Context, input GuidedSessionMutationInput, authorizer Authorizer, events EventSink) (GuidedSessionResult, error) {
	state, record, err := s.guidedMutationState(ctx, input.ChainID)
	if err != nil {
		return GuidedSessionResult{}, err
	}
	if state.LastActiveChain == record.ChainID {
		return GuidedSessionResult{Action: MutationUnchanged, Session: record}, nil
	}
	state.LastActiveChain = record.ChainID
	return s.applyGuidedState(ctx, state, record, nil, input.Confirmed, authorizer, events)
}

// ReopenGuidedSession reopens one stage and marks downstream stages for review.
func (s *Service) ReopenGuidedSession(ctx context.Context, input GuidedSessionMutationInput, authorizer Authorizer, events EventSink) (GuidedSessionResult, error) {
	state, record, err := s.guidedMutationState(ctx, input.ChainID)
	if err != nil {
		return GuidedSessionResult{}, err
	}
	stage := strings.TrimSpace(input.Stage)
	index := -1
	for candidateIndex, candidate := range guidedStages {
		if stage == candidate {
			index = candidateIndex
			break
		}
	}
	if index < 0 {
		return GuidedSessionResult{}, fmt.Errorf("unsupported guided stage %q", stage)
	}
	if record.StageStatuses == nil {
		record.StageStatuses = map[string]string{}
	}
	impacted := append([]string(nil), guidedStages[index+1:]...)
	changed := state.LastActiveChain != record.ChainID || record.CurrentStage != stage || record.StageStatuses[stage] != "in_progress"
	nextAction := fmt.Sprintf("Continue %s after reopening it.", stage)
	if len(impacted) > 0 {
		nextAction = fmt.Sprintf("Review %s after reopening %s.", strings.Join(impacted, ", "), stage)
	}
	if record.NextAction != nextAction {
		changed = true
	}
	record.CurrentStage = stage
	record.StageStatuses[stage] = "in_progress"
	for _, later := range impacted {
		if record.StageStatuses[later] != "needs_review" {
			changed = true
		}
		record.StageStatuses[later] = "needs_review"
	}
	record.NextAction = nextAction
	if !changed {
		return GuidedSessionResult{Action: MutationUnchanged, Session: record, Impacted: impacted}, nil
	}
	state.LastActiveChain = record.ChainID
	state.Sessions[record.ChainID] = record
	return s.applyGuidedState(ctx, state, record, impacted, input.Confirmed, authorizer, events)
}

// ReviewGuidedSession marks downstream needs-review stages reviewed.
func (s *Service) ReviewGuidedSession(ctx context.Context, input GuidedSessionMutationInput, authorizer Authorizer, events EventSink) (GuidedSessionResult, error) {
	state, record, err := s.guidedMutationState(ctx, input.ChainID)
	if err != nil {
		return GuidedSessionResult{}, err
	}
	var reviewed []string
	for _, stage := range guidedStages {
		if record.StageStatuses[stage] == "needs_review" {
			record.StageStatuses[stage] = "reviewed"
			reviewed = append(reviewed, stage)
		}
	}
	if len(reviewed) == 0 {
		return GuidedSessionResult{Action: MutationUnchanged, Session: record}, nil
	}
	record.NextAction = "Downstream review checkpoints complete. Continue the planning flow."
	state.LastActiveChain = record.ChainID
	state.Sessions[record.ChainID] = record
	return s.applyGuidedState(ctx, state, record, reviewed, input.Confirmed, authorizer, events)
}

// CurrentGuidePacket renders the last-active local guide packet without mutation.
func (s *Service) CurrentGuidePacket(ctx context.Context) (GuidePacket, error) {
	record, err := s.CurrentGuidedSession(ctx)
	if err != nil {
		return GuidePacket{}, err
	}
	return s.buildGuidePacket(ctx, "brain plan guide current", record, "")
}

// GuidePacketForChain renders one local guide packet without mutation.
func (s *Service) GuidePacketForChain(ctx context.Context, chainID, checkpoint string) (GuidePacket, error) {
	record, err := s.GetGuidedSession(ctx, chainID)
	if err != nil {
		return GuidePacket{}, err
	}
	return s.buildGuidePacket(ctx, "brain plan guide show", record, checkpoint)
}

func (s *Service) guidedMutationState(ctx context.Context, chainID string) (GuidedSessionState, GuidedSessionRecord, error) {
	if err := s.requireWritable(ctx); err != nil {
		return GuidedSessionState{}, GuidedSessionRecord{}, err
	}
	state, err := s.repository.ReadGuidedSessions(ctx)
	if err != nil {
		return GuidedSessionState{}, GuidedSessionRecord{}, err
	}
	chainID = normalizeChainID(chainID)
	record, ok := state.Sessions[chainID]
	if !ok {
		return GuidedSessionState{}, GuidedSessionRecord{}, fmt.Errorf("guided session %q not found", chainID)
	}
	return state, record, nil
}

func (s *Service) applyGuidedState(ctx context.Context, state GuidedSessionState, record GuidedSessionRecord, impacted []string, confirmed bool, authorizer Authorizer, events EventSink) (GuidedSessionResult, error) {
	if !confirmed {
		return GuidedSessionResult{}, ErrConfirmationRequired
	}
	if authorizer == nil {
		return GuidedSessionResult{}, fmt.Errorf("planning mutation requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionBrainstorm); err != nil {
		return GuidedSessionResult{}, err
	}
	if events == nil {
		return GuidedSessionResult{}, ErrEventSinkRequired
	}
	now := s.now().UTC()
	timestamp := now.Format("2006-01-02T15:04:05Z07:00")
	record.UpdatedAt = timestamp
	state.LastUpdatedAt = timestamp
	state.Sessions[record.ChainID] = record
	state, action, err := s.repository.ReplaceGuidedSessions(ctx, state)
	if err != nil {
		return GuidedSessionResult{}, err
	}
	record = state.Sessions[record.ChainID]
	result := GuidedSessionResult{Action: action, Session: record, Impacted: impacted}
	if action == MutationUnchanged {
		return result, nil
	}
	event := Event{Name: EventGuidedSessionUpdated, ModuleID: s.moduleID, Artifact: planning.ArtifactRef{Kind: planning.ArtifactBrainstorm, ID: planning.ArtifactID(record.Brainstorm)}, Outcome: action, OccurredAt: now}
	if err := events.Publish(ctx, event); err != nil {
		return GuidedSessionResult{}, fmt.Errorf("publish planning event: %w", err)
	}
	result.Event = &event
	return result, nil
}

func (s *Service) buildGuidePacket(ctx context.Context, command string, record GuidedSessionRecord, checkpoint string) (GuidePacket, error) {
	if record.Brainstorm == "" {
		return GuidePacket{}, fmt.Errorf("guided session %q is not linked to a brainstorm", record.ChainID)
	}
	document, err := s.repository.GetBrainstorm(ctx, planning.ArtifactID(record.Brainstorm))
	if err != nil {
		return GuidePacket{}, err
	}
	workspace, err := s.Status(ctx)
	if err != nil {
		return GuidePacket{}, err
	}
	checkpoint = strings.TrimSpace(checkpoint)
	if checkpoint == "" {
		checkpoint = record.CurrentClusterLabel
	}
	if checkpoint == "" {
		checkpoint = "vision-intake"
	}
	goal, pass, strengthen := guideCheckpoint(checkpoint)
	artifactPath := document.Path
	contract := map[string]any{
		"role":   "co_planning_facilitator",
		"stance": []string{"collaborative", "direct", "skeptical_when_needed", "keep_scope_small"},
		"goal":   goal,
		"question_strategy": map[string]any{
			"cluster_size_min": 2, "cluster_size_max": 4, "reflect_once_per_cluster": true,
			"gap_guidance": "one_recommended_plus_up_to_two_alternatives", "menu_actions": []string{"continue", "refine", "stop_for_now"},
		},
		"artifact_strategy": map[string]any{
			"write_mode": "additive", "durable_artifact": artifactPath, "strengthen_sections": strengthen,
			"preserve_rules": []string{"Use the user's own language when it clarifies intent.", "Do not invent a new durable planning layer during brainstorm guidance.", "Do not draft implementation slices during brainstorm guidance."},
		},
		"do":              []string{"Ask small clusters of focused questions instead of dumping a long form.", "Reflect back what changed before moving to the next checkpoint.", "Keep scope bounded and surface simpler alternatives when the work sprawls.", "Prefer one recommended path plus up to two alternatives."},
		"avoid":           []string{"Do not call model APIs from `plan`.", "Do not mutate guided session state while rendering the packet.", "Do not jump into execution slices during brainstorm guidance."},
		"quality_bar":     []string{"The user should leave this checkpoint with clearer scope and fewer hidden assumptions.", "The brainstorm note should be stronger than it was before the checkpoint started."},
		"completion_gate": []string{"The checkpoint-specific sections are strong enough to keep moving without guessing.", "The recap and next action are specific enough for the next guided move."},
		"command_hints": []map[string]any{
			{"purpose": "resume_current_stage", "command": "brain plan brainstorm resume " + record.Brainstorm + " --project ."},
			{"purpose": "preview_current_checkpoint", "command": "brain plan guide show --project . --chain " + record.ChainID + " --stage brainstorm --checkpoint " + checkpoint + " --json"},
		},
	}
	sources := []string{".plan/.meta/guided_sessions.json", ".plan/PROJECT.md", ".plan/ROADMAP.md", artifactPath}
	sort.Strings(sources)
	prompt := fmt.Sprintf("You are guiding the brainstorm stage for `brain plan`.\nGoal: %s\nCurrent summary: %s\nNext action: %s\nDurable artifact: %s\nCheckpoint: %s", goal, record.Summary, record.NextAction, artifactPath, checkpoint)
	return GuidePacket{
		SchemaVersion: 2, Kind: "guide_packet", GeneratedAt: s.now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		Builder:   map[string]any{"command": command, "format": "json"},
		Workspace: map[string]any{"project_root": s.projectRoot, "planning_mode": "guided", "planning_model": workspace.PlanningModel, "source_mode": string(workspace.Ownership), "story_backend": "local"},
		Ownership: localOwnership(),
		Session:   map[string]any{"chain_id": record.ChainID, "current_stage": "brainstorm", "current_cluster": record.CurrentCluster, "current_cluster_label": checkpoint, "stage_statuses": record.StageStatuses, "summary": record.Summary, "next_action": record.NextAction},
		Artifact:  map[string]any{"type": "brainstorm", "slug": record.Brainstorm, "title": document.Artifact.Title, "path": artifactPath, "status": "active"},
		Mode:      map[string]any{"stage": "brainstorm", "checkpoint": checkpoint, "pass": pass}, Sources: sources, Contract: contract, RenderedPrompt: prompt,
	}, nil
}

func guideCheckpoint(checkpoint string) (string, string, []string) {
	switch checkpoint {
	case "vision-intake":
		return "Capture the user's vision and supporting material before narrowing the problem.", "brainstorm_intake", []string{"Vision", "Supporting Material"}
	case "clarify-problem-user-value":
		return "Clarify the core problem and who benefits before discussing solutions.", "brainstorm_refine", []string{"Problem", "User / Value"}
	case "clarify-constraints-appetite":
		return "Bound the work with explicit constraints and appetite.", "brainstorm_refine", []string{"Constraints", "Appetite"}
	case "clarify-open-approaches":
		return "Trim the open questions down to the real blockers and identify the best candidate approaches.", "brainstorm_refine", []string{"Remaining Open Questions", "Candidate Approaches"}
	default:
		return "Review the brainstorm before direct spec promotion.", "brainstorm_review", []string{"Decision Snapshot", "Challenge"}
	}
}

func localOwnership() map[string]any {
	return map[string]any{
		"mode": "local", "entry_mode": "local_promotion", "canonical_discussion": "local_brainstorm",
		"canonical_planning_artifact": "local_spec", "readiness_source": "local_spec", "write_targets": []string{"local_spec"}, "mirror_local_meta": true,
	}
}

func normalizeChainID(value string) string {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "/") {
		return value
	}
	return "brainstorm/" + value
}
