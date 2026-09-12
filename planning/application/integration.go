package application

import (
	"context"
	"fmt"
	"time"

	"github.com/JimmyMcBride/brain/planning"
)

// IntegrationContractVersion is the current provider-neutral Planning
// integration contract version.
const IntegrationContractVersion = 1

// ExternalReference identifies provider-owned evidence without exposing the
// provider's API types to Planning.
type ExternalReference struct {
	Provider  string `json:"provider"`
	Kind      string `json:"kind"`
	OpaqueID  string `json:"opaque_id"`
	DisplayID string `json:"display_id"`
	URL       string `json:"url"`
	Revision  string `json:"revision,omitempty"`
}

// CollaborationContribution is content contributed beneath a collaboration
// source. Reference is optional when the provider cannot address a contribution.
type CollaborationContribution struct {
	Content   string             `json:"content"`
	Reference *ExternalReference `json:"reference,omitempty"`
}

// CollaborationSourceSnapshot is the content and opaque revision observed at
// one canonical collaboration source.
type CollaborationSourceSnapshot struct {
	SchemaVersion int                         `json:"schema_version"`
	Source        ExternalReference           `json:"source"`
	Title         string                      `json:"title"`
	Content       string                      `json:"content"`
	Contributions []CollaborationContribution `json:"contributions,omitempty"`
}

// CollaborationRepairRequest requires the adapter to check ExpectedRevision
// before replacing content. Adapters must document whether their provider can
// make that check atomic with the write.
type CollaborationRepairRequest struct {
	Source           ExternalReference `json:"source"`
	ExpectedRevision string            `json:"expected_revision"`
	Content          string            `json:"content"`
}

// CollaborationRepairEvidence records a guarded repair. Revision may be empty
// when a fresh read is needed; Changed distinguishes a write from an unchanged
// provider response.
type CollaborationRepairEvidence struct {
	Source  ExternalReference `json:"source"`
	Changed bool              `json:"changed"`
}

// CollaborationSource is the provider boundary for collaboration reads and
// guarded source repair. Planning owns assessment and repair-content decisions.
type CollaborationSource interface {
	Read(context.Context, ExternalReference) (CollaborationSourceSnapshot, error)
	Repair(context.Context, CollaborationRepairRequest) (CollaborationRepairEvidence, error)
}

// PublicationActionKind identifies the semantic subject of a publication
// action.
type PublicationActionKind string

const (
	// PublicationArtifactAction applies an initiative or spec artifact.
	PublicationArtifactAction PublicationActionKind = "artifact"
	// PublicationGroupAction applies one shared delivery grouping.
	PublicationGroupAction PublicationActionKind = "group"
	// PublicationRelationshipAction applies a relationship between artifacts.
	PublicationRelationshipAction PublicationActionKind = "relationship"
	// PublicationWorkspaceAction records an execution-workspace decision.
	PublicationWorkspaceAction PublicationActionKind = "workspace"
)

// PublicationWorkspaceChoice is the reviewed handling for an optional
// execution workspace.
type PublicationWorkspaceChoice string

const (
	// PublicationWorkspaceCreate provisions a new execution workspace.
	PublicationWorkspaceCreate PublicationWorkspaceChoice = "create"
	// PublicationWorkspaceConnect uses an explicitly selected existing workspace.
	PublicationWorkspaceConnect PublicationWorkspaceChoice = "connect"
	// PublicationWorkspaceSkip records an explicit decision not to use a workspace.
	PublicationWorkspaceSkip PublicationWorkspaceChoice = "skip"
)

// PublicationRelationshipKind identifies a provider-neutral Planning
// relationship.
type PublicationRelationshipKind string

const (
	// PublicationRelationshipContains connects a grouping artifact to a member.
	PublicationRelationshipContains PublicationRelationshipKind = "contains"
	// PublicationRelationshipDependsOn connects a dependent artifact to its prerequisite.
	PublicationRelationshipDependsOn PublicationRelationshipKind = "depends_on"
)

// PublicationArtifact is the semantic provider state for one initiative or
// spec. Group and Dependencies contain Planning identities, not provider IDs.
type PublicationArtifact struct {
	Artifact     planning.ArtifactRef    `json:"artifact"`
	Title        string                  `json:"title"`
	Content      string                  `json:"content"`
	Readiness    planning.ReadinessState `json:"readiness"`
	Group        *planning.ArtifactRef   `json:"group,omitempty"`
	Dependencies []planning.ArtifactRef  `json:"dependencies,omitempty"`
	Reference    *ExternalReference      `json:"reference,omitempty"`
}

// PublicationGroup is one shared delivery grouping for a publication. Its
// provider representation remains adapter-owned.
type PublicationGroup struct {
	Title     string                 `json:"title"`
	Members   []planning.ArtifactRef `json:"members"`
	Reference *ExternalReference     `json:"reference,omitempty"`
}

// PublicationWorkspaceDecision records whether coordinated execution should
// create, connect, or explicitly skip an external workspace.
type PublicationWorkspaceDecision struct {
	Choice    PublicationWorkspaceChoice `json:"choice"`
	Title     string                     `json:"title,omitempty"`
	Reason    string                     `json:"reason,omitempty"`
	Reference *ExternalReference         `json:"reference,omitempty"`
}

// PublicationRelationship is a semantic relationship between Planning
// artifacts.
type PublicationRelationship struct {
	Kind   PublicationRelationshipKind `json:"kind"`
	Source planning.ArtifactRef        `json:"source"`
	Target planning.ArtifactRef        `json:"target"`
}

// PublicationInspectRequest selects provider state needed to classify a
// publication plan.
type PublicationInspectRequest struct {
	Target     ExternalReference             `json:"target"`
	Source     *ExternalReference            `json:"source,omitempty"`
	Artifacts  []planning.ArtifactRef        `json:"artifacts"`
	Candidates []ArtifactExternalReference   `json:"candidates,omitempty"`
	Group      *PublicationGroup             `json:"group,omitempty"`
	Workspace  *PublicationWorkspaceDecision `json:"workspace,omitempty"`
}

// PublicationSnapshot is provider state used by Planning to derive actions.
type PublicationSnapshot struct {
	SchemaVersion int                           `json:"schema_version"`
	Target        ExternalReference             `json:"target"`
	Artifacts     []PublicationArtifact         `json:"artifacts,omitempty"`
	Relationships []PublicationRelationship     `json:"relationships,omitempty"`
	Group         *PublicationGroup             `json:"group,omitempty"`
	Workspace     *PublicationWorkspaceDecision `json:"workspace,omitempty"`
}

// PublicationApplyAction is one Planning-classified action. Exactly one subject
// corresponds to Kind.
type PublicationApplyAction struct {
	Kind         PublicationActionKind         `json:"kind"`
	Action       MutationAction                `json:"action"`
	Artifact     *PublicationArtifact          `json:"artifact,omitempty"`
	Group        *PublicationGroup             `json:"group,omitempty"`
	Relationship *PublicationRelationship      `json:"relationship,omitempty"`
	Workspace    *PublicationWorkspaceDecision `json:"workspace,omitempty"`
}

// PublicationPlan is an ordered, provider-neutral publication request.
type PublicationPlan struct {
	SchemaVersion int                      `json:"schema_version"`
	Target        ExternalReference        `json:"target"`
	Source        *ExternalReference       `json:"source,omitempty"`
	Actions       []PublicationApplyAction `json:"actions"`
}

// PublicationActionEvidence records the requested action and any durable
// provider identities produced while applying it.
type PublicationActionEvidence struct {
	Action     PublicationApplyAction `json:"action"`
	References []ExternalReference    `json:"references,omitempty"`
}

// PublicationResult reports completed and failing actions without prescribing
// destructive rollback.
type PublicationResult struct {
	SchemaVersion         int                         `json:"schema_version"`
	Completed             []PublicationActionEvidence `json:"completed,omitempty"`
	Failed                *PublicationActionEvidence  `json:"failed,omitempty"`
	ManualFallbackAllowed bool                        `json:"manual_fallback_allowed"`
	ManualFallbackReason  string                      `json:"manual_fallback_reason,omitempty"`
}

// PublicationTarget is the provider boundary for inspecting and applying
// initiative/spec publication. Planning owns action classification and order.
// Apply must guard references against intervening changes, preserve action order,
// and return the completed prefix (including no-op actions) plus a failing action
// on partial failure. Evidence retains original actions; new identities belong in
// References. Reuse and unchanged actions perform no remote writes. References
// on completed artifact, group, and non-skipped workspace actions must include
// the resulting stable identity. Known identities must be retained rather than
// replaced. References on a failed action identify writes already made, not
// merely inspected objects.
type PublicationTarget interface {
	Inspect(context.Context, PublicationInspectRequest) (PublicationSnapshot, error)
	Apply(context.Context, PublicationPlan) (PublicationResult, error)
}

// ChangeRequestState identifies the lifecycle state of repository change
// evidence.
type ChangeRequestState string

const (
	// ChangeRequestOpen is active and not merged.
	ChangeRequestOpen ChangeRequestState = "open"
	// ChangeRequestClosed ended without merge.
	ChangeRequestClosed ChangeRequestState = "closed"
	// ChangeRequestMerged was merged.
	ChangeRequestMerged ChangeRequestState = "merged"
)

// PlanningChangeRequestEvidence describes the current Planning change request.
type PlanningChangeRequestEvidence struct {
	Reference ExternalReference  `json:"reference"`
	State     ChangeRequestState `json:"state"`
	HeadRef   string             `json:"head_ref"`
	BaseRef   string             `json:"base_ref"`
	Draft     bool               `json:"draft"`
}

// RepositoryEvidence records repository identity and the checked-out planning
// revision without assuming a provider's repository model.
type RepositoryEvidence struct {
	SchemaVersion         int                            `json:"schema_version"`
	Repository            ExternalReference              `json:"repository"`
	DefaultRef            string                         `json:"default_ref"`
	CurrentRef            string                         `json:"current_ref"`
	Commit                string                         `json:"commit"`
	PlanningChangeRequest *PlanningChangeRequestEvidence `json:"planning_change_request,omitempty"`
}

// RepositoryEvidenceSource provides the current repository context used by
// reconciliation and compatibility workflows.
type RepositoryEvidenceSource interface {
	Current(context.Context) (RepositoryEvidence, error)
}

// ArtifactExternalReference maps one stable Planning artifact identity to a
// provider-owned identity.
type ArtifactExternalReference struct {
	Artifact  planning.ArtifactRef `json:"artifact"`
	Reference ExternalReference    `json:"reference"`
}

// ReconciliationEvidence records the repository evidence used for the last
// successful reconciliation.
type ReconciliationEvidence struct {
	Repository  RepositoryEvidence `json:"repository"`
	CompletedAt time.Time          `json:"completed_at"`
}

// ExternalMappingState retains stable provider identities needed for safe
// reruns. Revision guards atomic replacement of the complete state.
type ExternalMappingState struct {
	SchemaVersion       int                         `json:"schema_version"`
	Revision            string                      `json:"revision"`
	ArtifactReferences  []ArtifactExternalReference `json:"artifact_references,omitempty"`
	SourceReferences    []ExternalReference         `json:"source_references,omitempty"`
	GroupReferences     []ArtifactExternalReference `json:"group_references,omitempty"`
	WorkspaceReferences []ExternalReference         `json:"workspace_references,omitempty"`
	LastReconciliation  *ReconciliationEvidence     `json:"last_reconciliation,omitempty"`
}

// ExternalMappingRepository loads and atomically replaces provider mapping
// state. Save must fail with IntegrationRevisionConflict when ExpectedRevision
// no longer matches.
type ExternalMappingRepository interface {
	Load(context.Context) (ExternalMappingState, error)
	Save(ctx context.Context, state ExternalMappingState, expectedRevision string) (ExternalMappingState, error)
}

// AdoptionResult reports provider objects adopted into stable Planning
// mappings. Actions retain the application-owned classification.
type AdoptionResult struct {
	SchemaVersion         int                         `json:"schema_version"`
	Actions               []PublicationActionEvidence `json:"actions,omitempty"`
	Failed                *PublicationActionEvidence  `json:"failed,omitempty"`
	ManualFallbackAllowed bool                        `json:"manual_fallback_allowed"`
	ManualFallbackReason  string                      `json:"manual_fallback_reason,omitempty"`
	Mappings              ExternalMappingState        `json:"mappings"`
	Event                 *Event                      `json:"event,omitempty"`
}

// ExecutionStatus is the provider-neutral delivery status Planning can apply.
type ExecutionStatus string

const (
	// ExecutionTodo is ready but not started.
	ExecutionTodo ExecutionStatus = "todo"
	// ExecutionInProgress is actively being implemented.
	ExecutionInProgress ExecutionStatus = "in_progress"
	// ExecutionInReview is awaiting review or merge.
	ExecutionInReview ExecutionStatus = "in_review"
	// ExecutionDone is complete.
	ExecutionDone ExecutionStatus = "done"
)

// ExecutionField describes one semantic field supported by an execution
// workspace. SupportedStatuses is populated for a status field.
type ExecutionField struct {
	Role              string            `json:"role"`
	SupportedStatuses []ExecutionStatus `json:"supported_statuses,omitempty"`
}

// ExecutionFieldRoleStatus is the semantic role for an execution-status field.
const ExecutionFieldRoleStatus = "status"

// ExecutionWorkItem connects published work to its execution workspace item.
type ExecutionWorkItem struct {
	Artifact      planning.ArtifactRef `json:"artifact"`
	PublishedWork ExternalReference    `json:"published_work"`
	Reference     ExternalReference    `json:"reference"`
	Status        ExecutionStatus      `json:"status"`
}

// ExecutionWorkspaceSnapshot is the supported field and tracked-item state of
// one execution workspace.
type ExecutionWorkspaceSnapshot struct {
	SchemaVersion int                 `json:"schema_version"`
	Workspace     ExternalReference   `json:"workspace"`
	Fields        []ExecutionField    `json:"fields,omitempty"`
	Items         []ExecutionWorkItem `json:"items,omitempty"`
}

// ExecutionAttachRequest attaches published Planning work to an execution
// workspace.
type ExecutionAttachRequest struct {
	Workspace     ExternalReference    `json:"workspace"`
	Artifact      planning.ArtifactRef `json:"artifact"`
	PublishedWork ExternalReference    `json:"published_work"`
}

// ExecutionStatusRequest applies one Planning execution status to a tracked
// work item.
type ExecutionStatusRequest struct {
	Workspace ExternalReference `json:"workspace"`
	WorkItem  ExternalReference `json:"work_item"`
	Status    ExecutionStatus   `json:"status"`
}

// ReconciliationPlan is the ordered application decision for bringing
// publication, execution, and mapping evidence into agreement.
type ReconciliationPlan struct {
	SchemaVersion           int                      `json:"schema_version"`
	Repository              RepositoryEvidence       `json:"repository"`
	Publication             PublicationPlan          `json:"publication"`
	Execution               []ExecutionStatusRequest `json:"execution,omitempty"`
	ExpectedMappingRevision string                   `json:"expected_mapping_revision"`
}

// ReconciliationResult retains applied evidence and the atomically saved
// mapping state needed for safe reruns.
type ReconciliationResult struct {
	SchemaVersion int                    `json:"schema_version"`
	Publication   PublicationResult      `json:"publication"`
	Execution     []ExecutionWorkItem    `json:"execution,omitempty"`
	Mappings      ExternalMappingState   `json:"mappings"`
	Evidence      ReconciliationEvidence `json:"evidence"`
}

// ExecutionWorkspace is the provider boundary for execution tracking.
// Planning owns status transition policy and action classification.
type ExecutionWorkspace interface {
	Inspect(context.Context, ExternalReference) (ExecutionWorkspaceSnapshot, error)
	Attach(context.Context, ExecutionAttachRequest) (ExecutionWorkItem, error)
	ApplyStatus(context.Context, ExecutionStatusRequest) (ExecutionWorkItem, error)
}

// IntegrationErrorClass identifies a stable provider-neutral failure class.
type IntegrationErrorClass string

const (
	// IntegrationAdapterDisabled means the configured adapter is not enabled.
	IntegrationAdapterDisabled IntegrationErrorClass = "adapter_disabled"
	// IntegrationProviderUnavailable means the provider could not be reached.
	IntegrationProviderUnavailable IntegrationErrorClass = "provider_unavailable"
	// IntegrationUnauthenticated means provider authentication is missing or invalid.
	IntegrationUnauthenticated IntegrationErrorClass = "unauthenticated"
	// IntegrationUnauthorized means provider authentication lacks required access.
	IntegrationUnauthorized IntegrationErrorClass = "unauthorized"
	// IntegrationRevisionConflict means guarded state changed before mutation.
	IntegrationRevisionConflict IntegrationErrorClass = "revision_conflict"
	// IntegrationAmbiguousIdentity means more than one provider object matched.
	IntegrationAmbiguousIdentity IntegrationErrorClass = "ambiguous_identity"
	// IntegrationUnsupportedCapability means the provider lacks a required capability.
	IntegrationUnsupportedCapability IntegrationErrorClass = "unsupported_capability"
	// IntegrationPartialFailure means at least one earlier action completed.
	IntegrationPartialFailure IntegrationErrorClass = "partial_failure"
	// IntegrationManualRemediationRequired means safe automation cannot continue.
	IntegrationManualRemediationRequired IntegrationErrorClass = "manual_remediation_required"
)

// IntegrationError is a typed provider-neutral integration failure.
type IntegrationError struct {
	Class                 IntegrationErrorClass `json:"class"`
	Provider              string                `json:"provider,omitempty"`
	Operation             string                `json:"operation,omitempty"`
	Message               string                `json:"message,omitempty"`
	ManualFallbackAllowed bool                  `json:"manual_fallback_allowed"`
	Err                   error                 `json:"-"`
}

// Error returns the stable class and available failure message.
func (e *IntegrationError) Error() string {
	if e == nil {
		return "planning integration failed"
	}
	message := e.Message
	if message == "" && e.Err != nil {
		message = e.Err.Error()
	}
	if message == "" {
		message = "planning integration failed"
	}
	if e.Class == "" {
		return message
	}
	return fmt.Sprintf("%s: %s", e.Class, message)
}

// Unwrap returns the provider-neutral error cause, when present.
func (e *IntegrationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
