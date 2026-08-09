package application

import (
	"context"
	"errors"
	"time"

	"github.com/JimmyMcBride/brain/planning"
)

const (
	// PermissionRead authorizes Planning reads.
	PermissionRead = "planning.read"
	// PermissionBrainstorm authorizes brainstorm mutations.
	PermissionBrainstorm = "planning.brainstorm"
	// PermissionRoadmap authorizes roadmap mutations.
	PermissionRoadmap = "planning.roadmap"
	// EventBrainstormCreated identifies a created-brainstorm audit event.
	EventBrainstormCreated = "planning.brainstorm.created"
	// EventRoadmapUpdated identifies an updated-roadmap audit event.
	EventRoadmapUpdated = "planning.roadmap.updated"
)

// WorkspaceState classifies local workspace compatibility.
type WorkspaceState string

const (
	// WorkspaceMissing means no local Planning workspace exists.
	WorkspaceMissing WorkspaceState = "missing"
	// WorkspaceCompatible means the workspace is readable and writable.
	WorkspaceCompatible WorkspaceState = "compatible"
	// WorkspaceMigrationRequired means standalone migration must run first.
	WorkspaceMigrationRequired WorkspaceState = "migration_required"
	// WorkspaceFutureSchema means the workspace schema is newer than supported.
	WorkspaceFutureSchema WorkspaceState = "future_schema"
	// WorkspaceUnsupportedSource means the workspace uses a deferred source adapter.
	WorkspaceUnsupportedSource WorkspaceState = "unsupported_source"
	// WorkspaceUnsupportedLegacy means the workspace uses a retired integration.
	WorkspaceUnsupportedLegacy WorkspaceState = "unsupported_legacy_integration"
	// WorkspaceInvalid means workspace metadata cannot be interpreted safely.
	WorkspaceInvalid WorkspaceState = "invalid"
)

// WorkspaceStatus describes local schema and ownership compatibility.
type WorkspaceStatus struct {
	Project       string                 `json:"project,omitempty"`
	State         WorkspaceState         `json:"state"`
	SchemaVersion int                    `json:"schema_version,omitempty"`
	PlanningModel string                 `json:"planning_model,omitempty"`
	Ownership     planning.OwnershipMode `json:"ownership,omitempty"`
	Writable      bool                   `json:"writable"`
	Message       string                 `json:"message"`
	Guidance      []string               `json:"guidance,omitempty"`
}

// SpecSummary is the status-facing identity and lifecycle view of a spec.
type SpecSummary struct {
	ID         planning.ArtifactID  `json:"id"`
	Title      string               `json:"title"`
	Status     planning.SpecStatus  `json:"status"`
	Initiative *planning.ArtifactID `json:"initiative,omitempty"`
}

// ProjectStatus aggregates local spec lifecycle state.
type ProjectStatus struct {
	Project           string                 `json:"project"`
	PlanningModel     string                 `json:"planning_model"`
	SourceMode        planning.OwnershipMode `json:"source_mode"`
	TotalSpecs        int                    `json:"total_specs"`
	DraftSpecs        int                    `json:"draft_specs"`
	ApprovedSpecs     int                    `json:"approved_specs"`
	ImplementingSpecs int                    `json:"implementing_specs"`
	DoneSpecs         int                    `json:"done_specs"`
	ReadySpecs        []SpecSummary          `json:"ready_specs,omitempty"`
}

// BrainstormDocument combines a domain brainstorm with its local document data.
type BrainstormDocument struct {
	Artifact planning.Brainstorm `json:"artifact"`
	Path     string              `json:"path"`
	Body     string              `json:"body,omitempty"`
	Metadata map[string]any      `json:"metadata,omitempty"`
}

// SpecDocument combines a validated domain spec with its local document data.
type SpecDocument struct {
	Artifact planning.Spec  `json:"artifact"`
	Path     string         `json:"path"`
	Body     string         `json:"body,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// RoadmapDocument preserves the complete human-authored roadmap Markdown.
type RoadmapDocument struct {
	Path string `json:"path"`
	Body string `json:"body"`
}

// CheckInput selects either the whole project or one spec.
type CheckInput struct {
	SpecID *planning.ArtifactID `json:"spec_id,omitempty"`
}

// SpecQueryDocument is the minimally parsed spec view used by status and checks.
type SpecQueryDocument struct {
	ID         planning.ArtifactID  `json:"id"`
	Title      string               `json:"title"`
	Status     string               `json:"status"`
	Initiative *planning.ArtifactID `json:"initiative,omitempty"`
	Path       string               `json:"path"`
	Body       string               `json:"body"`
}

// CheckReport contains deterministic local quality findings.
type CheckReport struct {
	Project  string         `json:"project"`
	Scope    string         `json:"scope"`
	Findings []CheckFinding `json:"findings"`
}

// ErrorCount returns the number of blocking findings.
func (r CheckReport) ErrorCount() int {
	count := 0
	for _, finding := range r.Findings {
		if finding.Severity == "error" {
			count++
		}
	}
	return count
}

// WarningCount returns the number of guidance findings.
func (r CheckReport) WarningCount() int {
	count := 0
	for _, finding := range r.Findings {
		if finding.Severity == "warn" {
			count++
		}
	}
	return count
}

// HasErrors reports whether the check has any blocking findings.
func (r CheckReport) HasErrors() bool {
	return r.ErrorCount() > 0
}

// CheckFinding describes one local spec-quality result.
type CheckFinding struct {
	Severity      string `json:"severity"`
	Rule          string `json:"rule"`
	ArtifactType  string `json:"artifact_type"`
	ArtifactPath  string `json:"artifact_path"`
	ArtifactTitle string `json:"artifact_title"`
	Section       string `json:"section"`
	Message       string `json:"message"`
	Suggestion    string `json:"suggestion,omitempty"`
}

// MutationAction identifies the observable result of a Planning write.
type MutationAction string

const (
	// MutationCreate means a new artifact was written.
	MutationCreate MutationAction = "create"
	// MutationUpdate means an existing artifact was changed.
	MutationUpdate MutationAction = "update"
	// MutationUnchanged means the requested state already existed.
	MutationUnchanged MutationAction = "unchanged"
)

// BrainstormPreview describes a proposed brainstorm mutation.
type BrainstormPreview struct {
	Action   MutationAction     `json:"action"`
	Document BrainstormDocument `json:"document"`
}

// BrainstormResult describes an applied or unchanged brainstorm mutation.
type BrainstormResult struct {
	Action   MutationAction     `json:"action"`
	Document BrainstormDocument `json:"document"`
	Event    *Event             `json:"event,omitempty"`
}

// CreateBrainstormInput contains the requested title and confirmation decision.
type CreateBrainstormInput struct {
	Title     string `json:"title"`
	Confirmed bool   `json:"confirmed"`
}

// RoadmapPreview describes a proposed roadmap replacement.
type RoadmapPreview struct {
	Action   MutationAction  `json:"action"`
	Document RoadmapDocument `json:"document"`
}

// RoadmapResult describes an applied or unchanged roadmap replacement.
type RoadmapResult struct {
	Action   MutationAction  `json:"action"`
	Document RoadmapDocument `json:"document"`
	Event    *Event          `json:"event,omitempty"`
}

// UpdateRoadmapInput contains replacement Markdown and confirmation state.
type UpdateRoadmapInput struct {
	Body      string `json:"body"`
	Confirmed bool   `json:"confirmed"`
}

// Event is a host-neutral Planning mutation audit record.
type Event struct {
	Name       string               `json:"name"`
	ModuleID   string               `json:"module_id"`
	Artifact   planning.ArtifactRef `json:"artifact"`
	Outcome    MutationAction       `json:"outcome"`
	OccurredAt time.Time            `json:"occurred_at"`
}

// Repository is the persistence boundary required by shared Planning use cases.
type Repository interface {
	Status(context.Context) (WorkspaceStatus, error)
	ListBrainstorms(context.Context) ([]BrainstormDocument, error)
	GetBrainstorm(context.Context, planning.ArtifactID) (BrainstormDocument, error)
	FindBrainstorm(context.Context, planning.ArtifactID) (BrainstormDocument, bool, error)
	CreateBrainstorm(context.Context, planning.Brainstorm, time.Time) (BrainstormDocument, MutationAction, error)
	ListSpecs(context.Context) ([]SpecDocument, error)
	GetSpec(context.Context, planning.ArtifactID) (SpecDocument, error)
	QuerySpecs(context.Context, *planning.ArtifactID) ([]SpecQueryDocument, error)
	ReadRoadmap(context.Context) (RoadmapDocument, error)
	ReplaceRoadmap(context.Context, string) (RoadmapDocument, MutationAction, error)
}

// Authorizer lets a host enforce a Planning permission before mutation.
type Authorizer interface {
	Require(context.Context, string) error
}

// EventSink lets a host durably record a successful Planning mutation.
type EventSink interface {
	Publish(context.Context, Event) error
}

var (
	// ErrConfirmationRequired means a mutation was not explicitly confirmed.
	ErrConfirmationRequired = errors.New("planning mutation requires explicit confirmation")
	// ErrWorkspaceNotReadable means compatibility policy prevents reads.
	ErrWorkspaceNotReadable = errors.New("planning workspace is not readable")
	// ErrWorkspaceNotWritable means compatibility policy prevents writes.
	ErrWorkspaceNotWritable = errors.New("planning workspace is not writable")
	// ErrArtifactConflict means a stable artifact identity has different content.
	ErrArtifactConflict = errors.New("planning artifact conflicts with existing content")
	// ErrEventSinkRequired means a host did not provide mutation audit support.
	ErrEventSinkRequired = errors.New("planning mutation requires an event sink")
)
