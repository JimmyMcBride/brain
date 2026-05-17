package projectcontext

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	guidanceDecisionSchemaVersion = 1
	guidanceDecisionStateFile     = "guidance-decisions.json"

	GuidanceDecisionUnset    = "unset"
	GuidanceDecisionAccepted = "accepted"
	GuidanceDecisionDeclined = "declined"

	KarpathyGuidanceTopic   = "karpathy_guidelines"
	KarpathyGuidanceHeading = "## Karpathy Guidelines"
)

type GuidanceDecisionState struct {
	SchemaVersion      int    `json:"schema_version"`
	KarpathyGuidelines string `json:"karpathy_guidelines,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
}

type GuidanceRecommendation struct {
	Topic    string   `json:"topic"`
	Message  string   `json:"message"`
	Commands []string `json:"commands,omitempty"`
}

type GuidanceStatus struct {
	ProjectDir        string                  `json:"project_dir"`
	StatePath         string                  `json:"state_path"`
	StateStatus       string                  `json:"state_status"`
	Topic             string                  `json:"topic"`
	Decision          string                  `json:"decision"`
	AgentsManaged     bool                    `json:"agents_managed"`
	GuidelinesPresent bool                    `json:"guidelines_present"`
	Recommendation    *GuidanceRecommendation `json:"recommendation,omitempty"`
}

type GuidanceApplyResult struct {
	GuidanceStatus
	Action  string   `json:"action"`
	Results []Result `json:"results,omitempty"`
}

func (m *Manager) KarpathyGuidanceStatus(projectDir string) (*GuidanceStatus, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return nil, err
	}
	state, stateStatus, err := loadGuidanceDecisionState(projectDir)
	if err != nil {
		return nil, err
	}

	agentsManaged, guidelinesPresent, err := inspectKarpathyGuidelines(projectDir)
	if err != nil {
		return nil, err
	}
	decision := normalizeGuidanceDecision(state.KarpathyGuidelines)
	status := &GuidanceStatus{
		ProjectDir:        projectDir,
		StatePath:         guidanceDecisionStatePath(projectDir),
		StateStatus:       stateStatus,
		Topic:             KarpathyGuidanceTopic,
		Decision:          decision,
		AgentsManaged:     agentsManaged,
		GuidelinesPresent: guidelinesPresent,
	}
	if shouldRecommendKarpathyGuidance(status) {
		status.Recommendation = KarpathyGuidanceRecommendation()
	}
	return status, nil
}

func (m *Manager) SetKarpathyGuidanceDecision(ctx context.Context, projectDir, decision string) (*GuidanceApplyResult, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return nil, err
	}
	decision = normalizeGuidanceDecision(decision)
	if decision != GuidanceDecisionAccepted && decision != GuidanceDecisionDeclined {
		return nil, fmt.Errorf("unsupported Karpathy guidance decision %q", decision)
	}

	var results []Result
	action := "recorded"
	if decision == GuidanceDecisionAccepted {
		managed, present, err := inspectKarpathyGuidelines(projectDir)
		if err != nil {
			return nil, err
		}
		if !managed {
			return nil, errors.New("AGENTS.md is not Brain-managed; run `brain adopt --project .` before accepting Karpathy Guidelines")
		}
		if !present {
			result, err := m.syncAgentsContract(ctx, projectDir)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
			action = "updated"
		}
	}

	state, _, err := loadGuidanceDecisionState(projectDir)
	if err != nil {
		return nil, err
	}
	state.KarpathyGuidelines = decision
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := saveGuidanceDecisionState(projectDir, state); err != nil {
		return nil, err
	}

	status, err := m.KarpathyGuidanceStatus(projectDir)
	if err != nil {
		return nil, err
	}
	return &GuidanceApplyResult{
		GuidanceStatus: *status,
		Action:         action,
		Results:        results,
	}, nil
}

func KarpathyGuidanceRecommendation() *GuidanceRecommendation {
	return &GuidanceRecommendation{
		Topic:   KarpathyGuidanceTopic,
		Message: "Ask the user whether they want to add Karpathy Guidelines to this repo's Brain-managed AGENTS.md. Record the answer so Brain does not ask again.",
		Commands: []string{
			"brain context guidance karpathy --accept --project .",
			"brain context guidance karpathy --decline --project .",
		},
	}
}

func shouldRecommendKarpathyGuidance(status *GuidanceStatus) bool {
	return status != nil &&
		status.AgentsManaged &&
		!status.GuidelinesPresent &&
		status.Decision == GuidanceDecisionUnset
}

func (m *Manager) syncAgentsContract(ctx context.Context, projectDir string) (Result, error) {
	snapshot := scanRepo(ctx, projectDir)
	spec := fileSpec{
		Path:      filepath.Join(snapshot.ProjectDir, "AGENTS.md"),
		Kind:      "contract",
		Title:     "Project Agent Contract",
		BlockID:   "agents-contract",
		Body:      renderAgents(snapshot),
		Style:     "markdown",
		LocalNote: true,
	}
	result, err := syncSpec(spec, false, false, false)
	if err != nil {
		return Result{}, err
	}
	if rel, relErr := filepath.Rel(projectDir, spec.Path); relErr == nil {
		result.Path = filepath.ToSlash(rel)
	}
	return result, nil
}

func inspectKarpathyGuidelines(projectDir string) (bool, bool, error) {
	raw, err := os.ReadFile(filepath.Join(projectDir, "AGENTS.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	body := string(raw)
	managed := strings.Contains(body, managedBegin("agents-contract")) && strings.Contains(body, managedEnd("agents-contract"))
	return managed, strings.Contains(body, KarpathyGuidanceHeading), nil
}

func guidanceDecisionStatePath(projectDir string) string {
	return filepath.Join(projectDir, ".brain", "state", guidanceDecisionStateFile)
}

func loadGuidanceDecisionState(projectDir string) (GuidanceDecisionState, string, error) {
	path := guidanceDecisionStatePath(projectDir)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultGuidanceDecisionState(), projectMigrationStateMissing, nil
		}
		return GuidanceDecisionState{}, "", fmt.Errorf("read guidance decision state: %w", err)
	}

	var state GuidanceDecisionState
	if err := json.Unmarshal(raw, &state); err != nil {
		return defaultGuidanceDecisionState(), projectMigrationStateInvalid, nil
	}
	if state.SchemaVersion != 0 && state.SchemaVersion != guidanceDecisionSchemaVersion {
		return defaultGuidanceDecisionState(), projectMigrationStateInvalid, nil
	}
	return normalizeGuidanceDecisionState(state), projectMigrationStateReady, nil
}

func defaultGuidanceDecisionState() GuidanceDecisionState {
	return GuidanceDecisionState{
		SchemaVersion: guidanceDecisionSchemaVersion,
	}
}

func normalizeGuidanceDecisionState(state GuidanceDecisionState) GuidanceDecisionState {
	if state.SchemaVersion == 0 {
		state.SchemaVersion = guidanceDecisionSchemaVersion
	}
	state.KarpathyGuidelines = normalizeGuidanceDecision(state.KarpathyGuidelines)
	if state.KarpathyGuidelines == GuidanceDecisionUnset {
		state.KarpathyGuidelines = ""
	}
	return state
}

func normalizeGuidanceDecision(decision string) string {
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case GuidanceDecisionAccepted:
		return GuidanceDecisionAccepted
	case GuidanceDecisionDeclined:
		return GuidanceDecisionDeclined
	default:
		return GuidanceDecisionUnset
	}
}

func saveGuidanceDecisionState(projectDir string, state GuidanceDecisionState) error {
	state = normalizeGuidanceDecisionState(state)
	if state.SchemaVersion == 0 {
		state.SchemaVersion = guidanceDecisionSchemaVersion
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal guidance decision state: %w", err)
	}
	raw = append(raw, '\n')
	path := guidanceDecisionStatePath(projectDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create guidance decision state dir: %w", err)
	}
	return writeProjectMigrationFile(path, raw, 0o644)
}
