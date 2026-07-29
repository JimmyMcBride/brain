package application

import (
	"context"
	"errors"
	"time"

	"brain/internal/planning"
)

const (
	PermissionRead         = "planning.read"
	PermissionBrainstorm   = "planning.brainstorm"
	EventBrainstormCreated = "planning.brainstorm.created"
)

type WorkspaceState string

const (
	WorkspaceMissing           WorkspaceState = "missing"
	WorkspaceCompatible        WorkspaceState = "compatible"
	WorkspaceMigrationRequired WorkspaceState = "migration_required"
	WorkspaceFutureSchema      WorkspaceState = "future_schema"
	WorkspaceUnsupportedSource WorkspaceState = "unsupported_source"
	WorkspaceUnsupportedLegacy WorkspaceState = "unsupported_legacy_integration"
	WorkspaceInvalid           WorkspaceState = "invalid"
)

type WorkspaceStatus struct {
	State         WorkspaceState         `json:"state"`
	SchemaVersion int                    `json:"schema_version,omitempty"`
	PlanningModel string                 `json:"planning_model,omitempty"`
	Ownership     planning.OwnershipMode `json:"ownership,omitempty"`
	Writable      bool                   `json:"writable"`
	Message       string                 `json:"message"`
	Guidance      []string               `json:"guidance,omitempty"`
}

type BrainstormDocument struct {
	Artifact planning.Brainstorm `json:"artifact"`
	Path     string              `json:"path"`
	Body     string              `json:"body,omitempty"`
	Metadata map[string]any      `json:"metadata,omitempty"`
}

type SpecDocument struct {
	Artifact planning.Spec  `json:"artifact"`
	Path     string         `json:"path"`
	Body     string         `json:"body,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type MutationAction string

const (
	MutationCreate    MutationAction = "create"
	MutationUnchanged MutationAction = "unchanged"
)

type BrainstormPreview struct {
	Action   MutationAction     `json:"action"`
	Document BrainstormDocument `json:"document"`
}

type BrainstormResult struct {
	Action   MutationAction     `json:"action"`
	Document BrainstormDocument `json:"document"`
	Event    *Event             `json:"event,omitempty"`
}

type CreateBrainstormInput struct {
	Title     string
	Confirmed bool
}

type Event struct {
	Name       string               `json:"name"`
	ModuleID   string               `json:"module_id"`
	Artifact   planning.ArtifactRef `json:"artifact"`
	Outcome    MutationAction       `json:"outcome"`
	OccurredAt time.Time            `json:"occurred_at"`
}

type Repository interface {
	Status(context.Context) (WorkspaceStatus, error)
	ListBrainstorms(context.Context) ([]BrainstormDocument, error)
	GetBrainstorm(context.Context, planning.ArtifactID) (BrainstormDocument, error)
	FindBrainstorm(context.Context, planning.ArtifactID) (BrainstormDocument, bool, error)
	CreateBrainstorm(context.Context, planning.Brainstorm, time.Time) (BrainstormDocument, MutationAction, error)
	ListSpecs(context.Context) ([]SpecDocument, error)
	GetSpec(context.Context, planning.ArtifactID) (SpecDocument, error)
}

type Authorizer interface {
	Require(context.Context, string) error
}

type EventSink interface {
	Publish(context.Context, Event) error
}

type ServiceProvider interface {
	PlanningService() *Service
}

var (
	ErrConfirmationRequired = errors.New("planning mutation requires explicit confirmation")
	ErrWorkspaceNotReadable = errors.New("planning workspace is not readable")
	ErrWorkspaceNotWritable = errors.New("planning workspace is not writable")
	ErrArtifactConflict     = errors.New("planning artifact conflicts with existing content")
	ErrEventSinkRequired    = errors.New("planning mutation requires an event sink")
)
