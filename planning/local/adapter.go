package local

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"

	"gopkg.in/yaml.v3"
)

const (
	currentSchemaVersion = 3
	currentPlanningModel = "spec_first_v1"
)

type atomicWriter func(string, []byte, os.FileMode) error

// Adapter persists shared Planning application behavior in a schema-v3 .plan workspace.
type Adapter struct {
	projectRoot string
	writeAtomic atomicWriter
}

// New creates a schema-v3 local adapter rooted at projectRoot.
func New(projectRoot string) *Adapter {
	return newWithWriter(projectRoot, atomicWriteFile)
}

func newWithWriter(projectRoot string, writer atomicWriter) *Adapter {
	if writer == nil {
		writer = atomicWriteFile
	}
	root := filepath.Clean(projectRoot)
	if absolute, err := filepath.Abs(root); err == nil {
		root = absolute
	}
	return &Adapter{
		projectRoot: root,
		writeAtomic: writer,
	}
}

type workspaceMeta struct {
	SchemaVersion int    `json:"schema_version"`
	PlanningModel string `json:"planning_model"`
	SourceMode    string `json:"source_mode"`
}

// Status classifies workspace schema and ownership compatibility without mutation.
func (a *Adapter) Status(context.Context) (application.WorkspaceStatus, error) {
	planDir := filepath.Join(a.projectRoot, ".plan")
	if _, err := os.Stat(planDir); err != nil {
		if os.IsNotExist(err) {
			return application.WorkspaceStatus{
				State:    application.WorkspaceMissing,
				Writable: false,
				Message:  "local Planning workspace is missing",
				Guidance: []string{"initialize or migrate the workspace with standalone Plan before enabling Brain Planning writes"},
			}, nil
		}
		return application.WorkspaceStatus{}, fmt.Errorf("inspect Planning workspace: %w", err)
	}

	metaPath := filepath.Join(planDir, ".meta", "workspace.json")
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return application.WorkspaceStatus{
				State:    application.WorkspaceMigrationRequired,
				Writable: false,
				Message:  "Planning workspace metadata is missing",
				Guidance: []string{"preview and complete workspace migration with standalone Plan"},
			}, nil
		}
		return application.WorkspaceStatus{}, fmt.Errorf("read Planning workspace metadata: %w", err)
	}
	var meta workspaceMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return application.WorkspaceStatus{
			State:    application.WorkspaceInvalid,
			Writable: false,
			Message:  "Planning workspace metadata is invalid",
			Guidance: []string{"repair workspace metadata with standalone Plan before retrying"},
		}, nil
	}

	status := application.WorkspaceStatus{
		Project:       filepath.Base(a.projectRoot),
		SchemaVersion: meta.SchemaVersion,
		PlanningModel: meta.PlanningModel,
		Writable:      false,
	}
	switch {
	case meta.SchemaVersion > currentSchemaVersion:
		status.State = application.WorkspaceFutureSchema
		status.Message = fmt.Sprintf("Planning schema %d is newer than supported schema %d", meta.SchemaVersion, currentSchemaVersion)
		status.Guidance = []string{"upgrade Brain before using this workspace"}
		return status, nil
	case meta.SchemaVersion < currentSchemaVersion || meta.PlanningModel != currentPlanningModel:
		status.State = application.WorkspaceMigrationRequired
		status.Message = "Planning workspace requires standalone migration"
		status.Guidance = []string{"preview and complete migration with standalone Plan; Brain will not convert this workspace"}
		return status, nil
	}

	switch meta.SourceMode {
	case string(planning.OwnershipLocal):
		status.State = application.WorkspaceCompatible
		status.Ownership = planning.OwnershipLocal
		status.Writable = true
		status.Message = "local Planning workspace is compatible"
	case string(planning.OwnershipGitHub), string(planning.OwnershipHybrid):
		status.State = application.WorkspaceUnsupportedSource
		status.Ownership = planning.OwnershipMode(meta.SourceMode)
		status.Message = fmt.Sprintf("Planning source mode %q is not supported by the local module", meta.SourceMode)
		status.Guidance = []string{"continue with standalone Plan until the matching Brain adapter ships"}
	default:
		status.State = application.WorkspaceUnsupportedLegacy
		status.Message = "Planning workspace uses an unsupported legacy integration"
		status.Guidance = []string{"migrate source ownership with standalone Plan; Brain does not import retired integrations"}
	}
	return status, nil
}

// ListBrainstorms returns local brainstorm documents in deterministic order.
func (a *Adapter) ListBrainstorms(ctx context.Context) ([]application.BrainstormDocument, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return nil, err
	}
	paths, err := markdownFiles(filepath.Join(a.projectRoot, ".plan", "brainstorms"))
	if err != nil {
		return nil, err
	}
	out := make([]application.BrainstormDocument, 0, len(paths))
	for _, path := range paths {
		document, err := a.readBrainstorm(path)
		if err != nil {
			return nil, err
		}
		out = append(out, document)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Artifact.Title == out[j].Artifact.Title {
			return out[i].Artifact.ID < out[j].Artifact.ID
		}
		return out[i].Artifact.Title < out[j].Artifact.Title
	})
	return out, nil
}

// GetBrainstorm returns one local brainstorm document.
func (a *Adapter) GetBrainstorm(ctx context.Context, id planning.ArtifactID) (application.BrainstormDocument, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return application.BrainstormDocument{}, err
	}
	if err := id.Validate(); err != nil {
		return application.BrainstormDocument{}, err
	}
	return a.readBrainstorm(filepath.Join(a.projectRoot, ".plan", "brainstorms", string(id)+".md"))
}

// FindBrainstorm returns one local brainstorm document when it exists.
func (a *Adapter) FindBrainstorm(ctx context.Context, id planning.ArtifactID) (application.BrainstormDocument, bool, error) {
	document, err := a.GetBrainstorm(ctx, id)
	if err == nil {
		return document, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return application.BrainstormDocument{}, false, nil
	}
	return application.BrainstormDocument{}, false, err
}

// CreateBrainstorm atomically creates a local brainstorm or reports it unchanged.
func (a *Adapter) CreateBrainstorm(
	ctx context.Context,
	artifact planning.Brainstorm,
	createdAt time.Time,
) (application.BrainstormDocument, application.MutationAction, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return application.BrainstormDocument{}, "", err
	}
	if findings := planning.ValidateBrainstorm(artifact); hasErrorFindings(findings) {
		return application.BrainstormDocument{}, "", fmt.Errorf(
			"invalid brainstorm artifact: %s",
			joinErrorFindingMessages(findings),
		)
	}
	release, err := acquireMutationLock(ctx, filepath.Join(a.projectRoot, ".plan", "brainstorms", "."+string(artifact.ID)+".lock"))
	if err != nil {
		return application.BrainstormDocument{}, "", err
	}
	defer release()
	if existing, found, err := a.FindBrainstorm(ctx, artifact.ID); err != nil {
		return application.BrainstormDocument{}, "", err
	} else if found {
		if existing.Artifact.Title != artifact.Title {
			return application.BrainstormDocument{}, "", fmt.Errorf("%w: %s", application.ErrArtifactConflict, artifact.ID)
		}
		return existing, application.MutationUnchanged, nil
	}

	createdAt = createdAt.UTC()
	meta := map[string]any{
		"created_at": createdAt.Format(time.RFC3339),
		"project":    filepath.Base(a.projectRoot),
		"slug":       string(artifact.ID),
		"status":     "active",
		"title":      artifact.Title,
		"type":       "brainstorm",
		"updated_at": createdAt.Format(time.RFC3339),
	}
	body := renderBrainstormBody(artifact.Title, createdAt)
	raw, err := composeDocument(meta, body)
	if err != nil {
		return application.BrainstormDocument{}, "", err
	}
	path := filepath.Join(a.projectRoot, ".plan", "brainstorms", string(artifact.ID)+".md")
	if err := a.writeAtomic(path, raw, 0o644); err != nil {
		return application.BrainstormDocument{}, "", fmt.Errorf("write brainstorm atomically: %w", err)
	}
	document, err := a.readBrainstorm(path)
	if err != nil {
		return application.BrainstormDocument{}, "", err
	}
	return document, application.MutationCreate, nil
}

// RollbackBrainstormCreation removes only the exact brainstorm created by a failed application mutation.
func (a *Adapter) RollbackBrainstormCreation(ctx context.Context, artifact planning.Brainstorm, createdAt time.Time) error {
	if err := a.requireCompatible(ctx); err != nil {
		return err
	}
	if findings := planning.ValidateBrainstorm(artifact); hasErrorFindings(findings) {
		return fmt.Errorf("invalid brainstorm artifact: %s", joinErrorFindingMessages(findings))
	}
	id := artifact.ID
	path := filepath.Join(a.projectRoot, ".plan", "brainstorms", string(id)+".md")
	release, err := acquireMutationLock(ctx, filepath.Join(a.projectRoot, ".plan", "brainstorms", "."+string(id)+".lock"))
	if err != nil {
		return err
	}
	defer release()
	meta, body, err := readDocument(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	expected := createdAt.UTC().Format(time.RFC3339)
	if artifactID(meta, path) != id ||
		stringValue(meta["title"]) != artifact.Title ||
		stringValue(meta["type"]) != string(planning.ArtifactBrainstorm) ||
		stringValue(meta["created_at"]) != expected ||
		stringValue(meta["updated_at"]) != expected ||
		body != renderBrainstormBody(artifact.Title, createdAt.UTC()) {
		return fmt.Errorf("%w: refuse rollback of brainstorm %s created outside this mutation", application.ErrArtifactConflict, id)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rollback brainstorm creation: %w", err)
	}
	return nil
}

// ReplaceBrainstorm atomically replaces brainstorm Markdown or reports it unchanged.
func (a *Adapter) ReplaceBrainstorm(
	ctx context.Context,
	id planning.ArtifactID,
	body string,
	updatedAt time.Time,
) (application.BrainstormDocument, application.MutationAction, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return application.BrainstormDocument{}, "", err
	}
	if err := id.Validate(); err != nil {
		return application.BrainstormDocument{}, "", err
	}
	path := filepath.Join(a.projectRoot, ".plan", "brainstorms", string(id)+".md")
	release, err := acquireMutationLock(ctx, filepath.Join(a.projectRoot, ".plan", "brainstorms", "."+string(id)+".lock"))
	if err != nil {
		return application.BrainstormDocument{}, "", err
	}
	defer release()
	meta, currentBody, err := readDocument(path)
	if err != nil {
		return application.BrainstormDocument{}, "", err
	}
	if currentBody == body {
		document, err := a.readBrainstorm(path)
		return document, application.MutationUnchanged, err
	}
	meta["updated_at"] = updatedAt.UTC().Format(time.RFC3339)
	raw, err := composeDocument(meta, body)
	if err != nil {
		return application.BrainstormDocument{}, "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return application.BrainstormDocument{}, "", err
	}
	if err := a.writeAtomic(path, raw, info.Mode().Perm()); err != nil {
		return application.BrainstormDocument{}, "", fmt.Errorf("write brainstorm atomically: %w", err)
	}
	document, err := a.readBrainstorm(path)
	return document, application.MutationUpdate, err
}

// ReadGuidedSessions returns schema-v3 local guided-session state without creating it.
func (a *Adapter) ReadGuidedSessions(ctx context.Context) (application.GuidedSessionState, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return application.GuidedSessionState{}, err
	}
	path := filepath.Join(a.projectRoot, ".plan", ".meta", "guided_sessions.json")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return application.GuidedSessionState{SchemaVersion: currentSchemaVersion, Sessions: map[string]application.GuidedSessionRecord{}}, nil
	}
	if err != nil {
		return application.GuidedSessionState{}, err
	}
	state := application.GuidedSessionState{}
	if err := json.Unmarshal(raw, &state); err != nil {
		return application.GuidedSessionState{}, fmt.Errorf("parse guided session state: %w", err)
	}
	if state.SchemaVersion > currentSchemaVersion {
		return application.GuidedSessionState{}, fmt.Errorf("guided session schema version %d is newer than supported version %d", state.SchemaVersion, currentSchemaVersion)
	}
	if state.SchemaVersion == 0 {
		state.SchemaVersion = currentSchemaVersion
	}
	if state.Sessions == nil {
		state.Sessions = map[string]application.GuidedSessionRecord{}
	}
	return state, nil
}

// ReplaceGuidedSessions atomically replaces guided-session state or reports it unchanged.
func (a *Adapter) ReplaceGuidedSessions(ctx context.Context, state application.GuidedSessionState) (application.GuidedSessionState, application.MutationAction, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return application.GuidedSessionState{}, "", err
	}
	state.SchemaVersion = currentSchemaVersion
	if state.Sessions == nil {
		state.Sessions = map[string]application.GuidedSessionRecord{}
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return application.GuidedSessionState{}, "", err
	}
	raw = append(raw, '\n')
	path := filepath.Join(a.projectRoot, ".plan", ".meta", "guided_sessions.json")
	release, err := acquireMutationLock(ctx, filepath.Join(a.projectRoot, ".plan", ".meta", ".guided-sessions.lock"))
	if err != nil {
		return application.GuidedSessionState{}, "", err
	}
	defer release()
	current, err := os.ReadFile(path)
	if err == nil && bytes.Equal(current, raw) {
		return state, application.MutationUnchanged, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return application.GuidedSessionState{}, "", err
	}
	action := application.MutationUpdate
	if os.IsNotExist(err) {
		action = application.MutationCreate
	}
	if err := a.writeAtomic(path, raw, 0o644); err != nil {
		return application.GuidedSessionState{}, "", fmt.Errorf("write guided session state atomically: %w", err)
	}
	return state, action, nil
}

// ListSpecs returns validated local spec documents in deterministic order.
func (a *Adapter) ListSpecs(ctx context.Context) ([]application.SpecDocument, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return nil, err
	}
	paths, err := markdownFiles(filepath.Join(a.projectRoot, ".plan", "specs"))
	if err != nil {
		return nil, err
	}
	out := make([]application.SpecDocument, 0, len(paths))
	for _, path := range paths {
		document, err := a.readSpec(path)
		if err != nil {
			return nil, err
		}
		out = append(out, document)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Artifact.Title == out[j].Artifact.Title {
			return out[i].Artifact.ID < out[j].Artifact.ID
		}
		return out[i].Artifact.Title < out[j].Artifact.Title
	})
	return out, nil
}

// GetSpec returns one validated local spec document.
func (a *Adapter) GetSpec(ctx context.Context, id planning.ArtifactID) (application.SpecDocument, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return application.SpecDocument{}, err
	}
	if err := id.Validate(); err != nil {
		return application.SpecDocument{}, err
	}
	return a.readSpec(filepath.Join(a.projectRoot, ".plan", "specs", string(id)+".md"))
}

// FindSpec returns one local spec document when it exists.
func (a *Adapter) FindSpec(ctx context.Context, id planning.ArtifactID) (application.SpecDocument, bool, error) {
	document, err := a.GetSpec(ctx, id)
	if err == nil {
		return document, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return application.SpecDocument{}, false, nil
	}
	return application.SpecDocument{}, false, err
}

// WritePromotionSpecs atomically writes a direct local promotion set.
func (a *Adapter) WritePromotionSpecs(
	ctx context.Context,
	writes []application.PromotionSpecWrite,
	updatedAt time.Time,
) ([]application.SpecDocument, application.MutationAction, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return nil, "", err
	}
	release, err := acquireMutationLock(ctx, filepath.Join(a.projectRoot, ".plan", ".promotion.lock"))
	if err != nil {
		return nil, "", err
	}
	defer release()
	type pendingWrite struct {
		path     string
		raw      []byte
		previous []byte
		existed  bool
		mode     os.FileMode
	}
	pending := make([]pendingWrite, 0, len(writes))
	overall := application.MutationUnchanged
	for _, write := range writes {
		if findings := planning.ValidateSpec(write.Artifact); hasErrorFindings(findings) {
			return nil, "", fmt.Errorf("invalid promoted spec: %s", joinErrorFindingMessages(findings))
		}
		path := filepath.Join(a.projectRoot, ".plan", "specs", string(write.Artifact.ID)+".md")
		meta := cloneMetadata(write.Metadata)
		meta["project"] = filepath.Base(a.projectRoot)
		meta["slug"] = string(write.Artifact.ID)
		meta["status"] = string(write.Artifact.Status)
		meta["title"] = write.Artifact.Title
		meta["type"] = "spec"
		meta["updated_at"] = updatedAt.UTC().Format(time.RFC3339)
		previous, readErr := os.ReadFile(path)
		existed := readErr == nil
		mode := os.FileMode(0o644)
		if existed {
			currentMeta, currentBody, err := readDocument(path)
			if err != nil {
				return nil, "", err
			}
			meta["created_at"] = currentMeta["created_at"]
			if currentBody == write.Body && promotionMetadataEqual(currentMeta, meta) {
				continue
			}
			info, err := os.Stat(path)
			if err != nil {
				return nil, "", err
			}
			mode = info.Mode().Perm()
			if overall == application.MutationUnchanged {
				overall = application.MutationUpdate
			}
		} else if os.IsNotExist(readErr) {
			meta["created_at"] = updatedAt.UTC().Format(time.RFC3339)
			overall = application.MutationCreate
		} else {
			return nil, "", readErr
		}
		raw, err := composeDocument(meta, write.Body)
		if err != nil {
			return nil, "", err
		}
		pending = append(pending, pendingWrite{path: path, raw: raw, previous: previous, existed: existed, mode: mode})
	}
	written := make([]pendingWrite, 0, len(pending))
	for _, item := range pending {
		if err := a.writeAtomic(item.path, item.raw, item.mode); err != nil {
			for index := len(written) - 1; index >= 0; index-- {
				prior := written[index]
				if prior.existed {
					_ = atomicWriteFile(prior.path, prior.previous, prior.mode)
				} else {
					_ = os.Remove(prior.path)
				}
			}
			return nil, "", fmt.Errorf("write promoted specs atomically: %w", err)
		}
		written = append(written, item)
	}
	documents := make([]application.SpecDocument, 0, len(writes))
	for _, write := range writes {
		document, err := a.readSpec(filepath.Join(a.projectRoot, ".plan", "specs", string(write.Artifact.ID)+".md"))
		if err != nil {
			return nil, "", err
		}
		documents = append(documents, document)
	}
	return documents, overall, nil
}

// QuerySpecs returns minimally parsed specs for aggregate status and quality checks.
func (a *Adapter) QuerySpecs(ctx context.Context, id *planning.ArtifactID) ([]application.SpecQueryDocument, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return nil, err
	}
	var paths []string
	if id != nil {
		if err := id.Validate(); err != nil {
			return nil, err
		}
		paths = []string{filepath.Join(a.projectRoot, ".plan", "specs", string(*id)+".md")}
	} else {
		var err error
		paths, err = markdownFiles(filepath.Join(a.projectRoot, ".plan", "specs"))
		if err != nil {
			return nil, err
		}
	}
	documents := make([]application.SpecQueryDocument, 0, len(paths))
	for _, path := range paths {
		meta, body, err := readDocument(path)
		if err != nil {
			return nil, err
		}
		status := stringValue(meta["status"])
		if status == "" {
			status = string(planning.SpecDraft)
		}
		var initiativeID *planning.ArtifactID
		if value := strings.TrimSpace(stringValue(meta["initiative"])); value != "" {
			initiative := planning.ArtifactID(value)
			initiativeID = &initiative
		}
		documents = append(documents, application.SpecQueryDocument{
			ID: artifactID(meta, path), Title: stringValue(meta["title"]), Status: status,
			Initiative: initiativeID, Path: relativePlanningPath(a.projectRoot, path), Body: body,
		})
	}
	return documents, nil
}

// ReadRoadmap returns complete local roadmap Markdown.
func (a *Adapter) ReadRoadmap(ctx context.Context) (application.RoadmapDocument, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return application.RoadmapDocument{}, err
	}
	path := filepath.Join(a.projectRoot, ".plan", "ROADMAP.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return application.RoadmapDocument{}, err
	}
	return application.RoadmapDocument{Path: ".plan/ROADMAP.md", Body: string(raw)}, nil
}

// ReplaceRoadmap atomically replaces roadmap Markdown or reports it unchanged.
func (a *Adapter) ReplaceRoadmap(ctx context.Context, body string) (application.RoadmapDocument, application.MutationAction, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return application.RoadmapDocument{}, "", err
	}
	path := filepath.Join(a.projectRoot, ".plan", "ROADMAP.md")
	release, err := acquireMutationLock(ctx, filepath.Join(a.projectRoot, ".plan", ".roadmap.lock"))
	if err != nil {
		return application.RoadmapDocument{}, "", err
	}
	defer release()
	current, err := os.ReadFile(path)
	if err != nil {
		return application.RoadmapDocument{}, "", err
	}
	document := application.RoadmapDocument{Path: ".plan/ROADMAP.md", Body: body}
	if string(current) == body {
		return document, application.MutationUnchanged, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return application.RoadmapDocument{}, "", err
	}
	if err := a.writeAtomic(path, []byte(body), info.Mode().Perm()); err != nil {
		return application.RoadmapDocument{}, "", fmt.Errorf("write roadmap atomically: %w", err)
	}
	return document, application.MutationUpdate, nil
}

func (a *Adapter) requireCompatible(ctx context.Context) error {
	status, err := a.Status(ctx)
	if err != nil {
		return err
	}
	if status.State != application.WorkspaceCompatible {
		return fmt.Errorf("%w: %s", application.ErrWorkspaceNotWritable, status.Message)
	}
	return nil
}

func (a *Adapter) readBrainstorm(path string) (application.BrainstormDocument, error) {
	meta, body, err := readDocument(path)
	if err != nil {
		return application.BrainstormDocument{}, err
	}
	id := artifactID(meta, path)
	if kind := stringValue(meta["type"]); kind != string(planning.ArtifactBrainstorm) {
		return application.BrainstormDocument{}, fmt.Errorf("invalid brainstorm type %q in %s", kind, relativePlanningPath(a.projectRoot, path))
	}
	artifact := planning.Brainstorm{
		ID:      id,
		Title:   stringValue(meta["title"]),
		Summary: firstParagraph(extractSection(body, "Desired Outcome"), extractSection(body, "Focus Question")),
	}
	if findings := planning.ValidateBrainstorm(artifact); hasErrorFindings(findings) {
		return application.BrainstormDocument{}, fmt.Errorf(
			"invalid brainstorm %s: %s",
			relativePlanningPath(a.projectRoot, path),
			joinErrorFindingMessages(findings),
		)
	}
	return application.BrainstormDocument{
		Artifact: artifact,
		Path:     relativePlanningPath(a.projectRoot, path),
		Body:     body,
		Metadata: cloneMetadata(meta),
	}, nil
}

func (a *Adapter) readSpec(path string) (application.SpecDocument, error) {
	meta, body, err := readDocument(path)
	if err != nil {
		return application.SpecDocument{}, err
	}
	id := artifactID(meta, path)
	status := planning.SpecStatus(stringValue(meta["status"]))
	if kind := stringValue(meta["type"]); kind != string(planning.ArtifactSpec) {
		return application.SpecDocument{}, fmt.Errorf("invalid spec type %q in %s", kind, relativePlanningPath(a.projectRoot, path))
	}
	if status == "" {
		status = planning.SpecDraft
	}
	approval := planning.Approval{State: planning.ApprovalPending}
	if status != planning.SpecDraft {
		approval = planning.Approval{State: planning.ApprovalApproved, Reason: "preserved local spec status"}
	}
	var executionID *planning.ArtifactID
	if status == planning.SpecImplementing {
		value := planning.ArtifactID(string(id) + "-execution")
		executionID = &value
	}
	var initiativeID *planning.ArtifactID
	if value := strings.TrimSpace(stringValue(meta["initiative"])); value != "" {
		initiative := planning.ArtifactID(value)
		initiativeID = &initiative
	}
	artifact := planning.Spec{
		ID:           id,
		Title:        stringValue(meta["title"]),
		Status:       status,
		Approval:     approval,
		Dependencies: metadataIDs(meta, "dependencies", "blocked_by"),
		Verification: bulletItems(extractSection(body, "Verification")),
		Initiative:   initiativeID,
		ExecutionID:  executionID,
	}
	if findings := planning.ValidateSpec(artifact); hasErrorFindings(findings) {
		return application.SpecDocument{}, fmt.Errorf(
			"invalid spec %s: %s",
			relativePlanningPath(a.projectRoot, path),
			joinErrorFindingMessages(findings),
		)
	}
	return application.SpecDocument{
		Artifact: artifact,
		Path:     relativePlanningPath(a.projectRoot, path),
		Body:     body,
		Metadata: cloneMetadata(meta),
	}, nil
}

func readDocument(path string) (map[string]any, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return nil, "", fmt.Errorf("Planning document %s has no frontmatter", path)
	}
	rest := strings.TrimPrefix(content, "---\n")
	index := strings.Index(rest, "\n---\n")
	if index < 0 {
		return nil, "", fmt.Errorf("Planning document %s has unterminated frontmatter", path)
	}
	meta := map[string]any{}
	if err := yaml.Unmarshal([]byte(rest[:index]), &meta); err != nil {
		return nil, "", fmt.Errorf("parse Planning document %s: %w", path, err)
	}
	return meta, strings.TrimLeft(rest[index+len("\n---\n"):], "\n"), nil
}

func composeDocument(meta map[string]any, body string) ([]byte, error) {
	header, err := yaml.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal Planning frontmatter: %w", err)
	}
	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(header)
	out.WriteString("---\n\n")
	out.WriteString(strings.TrimRight(body, "\n"))
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func renderBrainstormBody(title string, createdAt time.Time) string {
	return fmt.Sprintf(`# Brainstorm: %s

Started: %s

## Focus Question

## Desired Outcome

## Vision

## Supporting Material

## Constraints

## Open Questions

## Ideas

## Raw Notes

## Refinement

### Problem

### User / Value

### Appetite

### Remaining Open Questions

### Candidate Approaches

### Decision Snapshot

## Challenge

### Rabbit Holes

### No-Gos

### Assumptions

### Likely Overengineering

### Simpler Alternative
`, title, createdAt.Format(time.RFC3339))
}

func markdownFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(paths)
	return paths, nil
}

func extractSection(content, heading string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	inSection := false
	level := 0
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			currentLevel := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
			title := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			if inSection && currentLevel <= level {
				break
			}
			if strings.EqualFold(title, heading) {
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

func bulletItems(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			if value := strings.TrimSpace(strings.TrimPrefix(line, "- ")); value != "" {
				out = append(out, value)
			}
		}
	}
	return out
}

func firstParagraph(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if index := strings.Index(value, "\n\n"); index >= 0 {
			value = value[:index]
		}
		return strings.Join(strings.Fields(value), " ")
	}
	return ""
}

func metadataIDs(meta map[string]any, keys ...string) []planning.ArtifactID {
	var out []planning.ArtifactID
	seen := map[planning.ArtifactID]struct{}{}
	for _, key := range keys {
		for _, value := range stringSlice(meta[key]) {
			id := planning.ArtifactID(strings.TrimSpace(value))
			if id == "" {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok {
				out = append(out, value)
			}
		}
		return out
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{typed}
	default:
		return nil
	}
}

func artifactID(meta map[string]any, path string) planning.ArtifactID {
	if slug := strings.TrimSpace(stringValue(meta["slug"])); slug != "" {
		return planning.ArtifactID(slug)
	}
	return planning.ArtifactID(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func cloneMetadata(meta map[string]any) map[string]any {
	out := make(map[string]any, len(meta))
	for key, value := range meta {
		out[key] = value
	}
	return out
}

func promotionMetadataEqual(current, desired map[string]any) bool {
	keys := []string{"slug", "status", "title", "type", "source_brainstorm", "initiative", "dependencies", "blocked_by"}
	left := make(map[string]any, len(keys))
	right := make(map[string]any, len(keys))
	for _, key := range keys {
		if value, exists := current[key]; exists {
			left[key] = value
		}
		if value, exists := desired[key]; exists {
			right[key] = value
		}
	}
	leftRaw, _ := json.Marshal(left)
	rightRaw, _ := json.Marshal(right)
	return bytes.Equal(leftRaw, rightRaw)
}

func relativePlanningPath(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(value)
}

func hasErrorFindings(findings []planning.Finding) bool {
	for _, finding := range findings {
		if finding.Severity == planning.SeverityError {
			return true
		}
	}
	return false
}

func joinErrorFindingMessages(findings []planning.Finding) string {
	messages := make([]string, 0, len(findings))
	for _, finding := range findings {
		if finding.Severity == planning.SeverityError {
			messages = append(messages, finding.Message)
		}
	}
	return strings.Join(messages, "; ")
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = os.Remove(tempPath)
		}
	}()
	if err = file.Chmod(mode); err != nil {
		return err
	}
	written, err := file.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(tempPath, path); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

func acquireMutationLock(ctx context.Context, path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create Planning lock directory: %w", err)
	}
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			return func() {
				_ = file.Close()
				_ = os.Remove(path)
			}, nil
		}
		if !isMutationLockContention(err) {
			return nil, fmt.Errorf("create Planning lock: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, fmt.Errorf("timed out waiting for Planning lock: %s", path)
		case <-ticker.C:
		}
	}
}
