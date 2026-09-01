package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/JimmyMcBride/brain/planning"
)

// Service coordinates shared Planning reads and guarded mutations.
type Service struct {
	repository  Repository
	moduleID    string
	projectRoot string
	now         func() time.Time
}

// Options configures host identity and time for emitted events.
type Options struct {
	ModuleID    string
	ProjectRoot string
	Now         func() time.Time
}

// New creates a Planning application service over repository.
func New(repository Repository, options Options) *Service {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		repository:  repository,
		moduleID:    strings.TrimSpace(options.ModuleID),
		projectRoot: strings.TrimSpace(options.ProjectRoot),
		now:         now,
	}
}

// Status reports workspace compatibility without requiring it.
func (s *Service) Status(ctx context.Context) (WorkspaceStatus, error) {
	if s == nil || s.repository == nil {
		return WorkspaceStatus{}, fmt.Errorf("planning repository is unavailable")
	}
	return s.repository.Status(ctx)
}

// ListBrainstorms returns readable brainstorm documents.
func (s *Service) ListBrainstorms(ctx context.Context) ([]BrainstormDocument, error) {
	if err := s.requireReadable(ctx); err != nil {
		return nil, err
	}
	return s.repository.ListBrainstorms(ctx)
}

// GetBrainstorm returns one readable brainstorm document.
func (s *Service) GetBrainstorm(ctx context.Context, id planning.ArtifactID) (BrainstormDocument, error) {
	if err := id.Validate(); err != nil {
		return BrainstormDocument{}, err
	}
	if err := s.requireReadable(ctx); err != nil {
		return BrainstormDocument{}, err
	}
	return s.repository.GetBrainstorm(ctx, id)
}

// ListSpecs returns readable validated spec documents.
func (s *Service) ListSpecs(ctx context.Context) ([]SpecDocument, error) {
	if err := s.requireReadable(ctx); err != nil {
		return nil, err
	}
	return s.repository.ListSpecs(ctx)
}

// GetSpec returns one readable validated spec document.
func (s *Service) GetSpec(ctx context.Context, id planning.ArtifactID) (SpecDocument, error) {
	if err := id.Validate(); err != nil {
		return SpecDocument{}, err
	}
	if err := s.requireReadable(ctx); err != nil {
		return SpecDocument{}, err
	}
	return s.repository.GetSpec(ctx, id)
}

// PreviewBrainstorm returns the mutation needed for a title without writing.
func (s *Service) PreviewBrainstorm(ctx context.Context, title string) (BrainstormPreview, error) {
	if err := s.requireWritable(ctx); err != nil {
		return BrainstormPreview{}, err
	}
	artifact, err := newBrainstorm(title)
	if err != nil {
		return BrainstormPreview{}, err
	}
	existing, found, err := s.repository.FindBrainstorm(ctx, artifact.ID)
	if err != nil {
		return BrainstormPreview{}, err
	}
	if found {
		if existing.Artifact.Title != artifact.Title {
			return BrainstormPreview{}, fmt.Errorf("%w: %s", ErrArtifactConflict, artifact.ID)
		}
		return BrainstormPreview{Action: MutationUnchanged, Document: existing}, nil
	}
	return BrainstormPreview{
		Action: MutationCreate,
		Document: BrainstormDocument{
			Artifact: artifact,
			Path:     ".plan/brainstorms/" + string(artifact.ID) + ".md",
		},
	}, nil
}

// CreateBrainstorm applies a confirmed, authorized, audited brainstorm mutation.
func (s *Service) CreateBrainstorm(
	ctx context.Context,
	input CreateBrainstormInput,
	authorizer Authorizer,
	events EventSink,
) (BrainstormResult, error) {
	preview, err := s.PreviewBrainstorm(ctx, input.Title)
	if err != nil {
		return BrainstormResult{}, err
	}
	if preview.Action == MutationUnchanged {
		return BrainstormResult{Action: MutationUnchanged, Document: preview.Document}, nil
	}
	if !input.Confirmed {
		return BrainstormResult{}, ErrConfirmationRequired
	}
	if authorizer == nil {
		return BrainstormResult{}, fmt.Errorf("planning mutation requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionBrainstorm); err != nil {
		return BrainstormResult{}, err
	}
	if events == nil {
		return BrainstormResult{}, ErrEventSinkRequired
	}

	now := s.now().UTC()
	document, action, err := s.repository.CreateBrainstorm(ctx, preview.Document.Artifact, now)
	if err != nil {
		return BrainstormResult{}, err
	}
	if action == MutationCreate {
		if err := s.createGuidedSession(ctx, document, now); err != nil {
			if rollbackErr := s.repository.RollbackBrainstormCreation(ctx, document.Artifact, now); rollbackErr != nil {
				return BrainstormResult{}, errors.Join(
					fmt.Errorf("create guided session: %w", err),
					fmt.Errorf("rollback brainstorm creation: %w", rollbackErr),
				)
			}
			return BrainstormResult{}, fmt.Errorf("create guided session: %w", err)
		}
	}
	result := BrainstormResult{Action: action, Document: document}
	if action != MutationCreate {
		return result, nil
	}
	event := Event{
		Name:       EventBrainstormCreated,
		ModuleID:   s.moduleID,
		Artifact:   planning.ArtifactRef{Kind: planning.ArtifactBrainstorm, ID: document.Artifact.ID},
		Outcome:    action,
		OccurredAt: now,
	}
	if err := events.Publish(ctx, event); err != nil {
		return BrainstormResult{}, fmt.Errorf("publish planning event: %w", err)
	}
	result.Event = &event
	return result, nil
}

func (s *Service) requireReadable(ctx context.Context) error {
	_, err := s.readableWorkspace(ctx)
	return err
}

func (s *Service) readableWorkspace(ctx context.Context) (WorkspaceStatus, error) {
	status, err := s.Status(ctx)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	if status.State != WorkspaceCompatible {
		return WorkspaceStatus{}, fmt.Errorf("%w: %s", ErrWorkspaceNotReadable, status.Message)
	}
	return status, nil
}

func (s *Service) requireWritable(ctx context.Context) error {
	status, err := s.Status(ctx)
	if err != nil {
		return err
	}
	if !status.Writable || status.State != WorkspaceCompatible {
		return fmt.Errorf("%w: %s", ErrWorkspaceNotWritable, status.Message)
	}
	return nil
}

func newBrainstorm(title string) (planning.Brainstorm, error) {
	title = strings.TrimSpace(title)
	artifact := planning.Brainstorm{
		ID:    planning.ArtifactID(slugify(title)),
		Title: title,
	}
	if findings := planning.ValidateBrainstorm(artifact); hasErrors(findings) {
		return planning.Brainstorm{}, fmt.Errorf("invalid brainstorm: %s", joinFindingMessages(findings))
	}
	return artifact, nil
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastDash := false
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			out.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func hasErrors(findings []planning.Finding) bool {
	for _, finding := range findings {
		if finding.Severity == planning.SeverityError {
			return true
		}
	}
	return false
}

func joinFindingMessages(findings []planning.Finding) string {
	messages := make([]string, 0, len(findings))
	for _, finding := range findings {
		if finding.Severity == planning.SeverityError {
			messages = append(messages, finding.Message)
		}
	}
	return strings.Join(messages, "; ")
}
