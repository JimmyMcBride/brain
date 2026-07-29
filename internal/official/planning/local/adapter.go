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

	"brain/internal/planning"
	"brain/internal/planning/application"

	"gopkg.in/yaml.v3"
)

const (
	currentSchemaVersion = 3
	currentPlanningModel = "spec_first_v1"
)

type AtomicWriter func(string, []byte, os.FileMode) error

type Adapter struct {
	projectRoot string
	writeAtomic AtomicWriter
}

func New(projectRoot string) *Adapter {
	return NewWithWriter(projectRoot, atomicWriteFile)
}

func NewWithWriter(projectRoot string, writer AtomicWriter) *Adapter {
	if writer == nil {
		writer = atomicWriteFile
	}
	return &Adapter{
		projectRoot: filepath.Clean(projectRoot),
		writeAtomic: writer,
	}
}

type workspaceMeta struct {
	SchemaVersion int    `json:"schema_version"`
	PlanningModel string `json:"planning_model"`
	SourceMode    string `json:"source_mode"`
}

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

func (a *Adapter) GetBrainstorm(ctx context.Context, id planning.ArtifactID) (application.BrainstormDocument, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return application.BrainstormDocument{}, err
	}
	if err := id.Validate(); err != nil {
		return application.BrainstormDocument{}, err
	}
	return a.readBrainstorm(filepath.Join(a.projectRoot, ".plan", "brainstorms", string(id)+".md"))
}

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

func (a *Adapter) CreateBrainstorm(
	ctx context.Context,
	artifact planning.Brainstorm,
	createdAt time.Time,
) (application.BrainstormDocument, application.MutationAction, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return application.BrainstormDocument{}, "", err
	}
	if findings := planning.ValidateBrainstorm(artifact); hasErrorFindings(findings) {
		return application.BrainstormDocument{}, "", fmt.Errorf("invalid brainstorm artifact")
	}
	release, err := acquireCreationLock(ctx, filepath.Join(a.projectRoot, ".plan", "brainstorms", "."+string(artifact.ID)+".lock"))
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

func (a *Adapter) GetSpec(ctx context.Context, id planning.ArtifactID) (application.SpecDocument, error) {
	if err := a.requireCompatible(ctx); err != nil {
		return application.SpecDocument{}, err
	}
	if err := id.Validate(); err != nil {
		return application.SpecDocument{}, err
	}
	return a.readSpec(filepath.Join(a.projectRoot, ".plan", "specs", string(id)+".md"))
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
		return application.BrainstormDocument{}, fmt.Errorf("invalid brainstorm %s", relativePlanningPath(a.projectRoot, path))
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
	artifact := planning.Spec{
		ID:           id,
		Title:        stringValue(meta["title"]),
		Status:       status,
		Approval:     approval,
		Dependencies: metadataIDs(meta, "dependencies", "blocked_by"),
		Verification: bulletItems(extractSection(body, "Verification")),
		ExecutionID:  executionID,
	}
	if findings := planning.ValidateSpec(artifact); hasErrorFindings(findings) {
		return application.SpecDocument{}, fmt.Errorf("invalid spec %s", relativePlanningPath(a.projectRoot, path))
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

func acquireCreationLock(ctx context.Context, path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create brainstorm directory: %w", err)
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
		if !os.IsExist(err) {
			return nil, fmt.Errorf("create brainstorm lock: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, fmt.Errorf("timed out waiting for brainstorm creation lock: %s", path)
		case <-ticker.C:
		}
	}
}
