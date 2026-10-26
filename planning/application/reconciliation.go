package application

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JimmyMcBride/brain/planning"
)

// EventIntegrationReconciled records a completed provider or reconciliation-
// evidence mutation.
const EventIntegrationReconciled = "planning.integration.reconciled"

// PreviewReconciliation loads revisioned mappings, resolves the checked-out
// repository change, and classifies publication drift without mutation.
func (s *Service) PreviewReconciliation(ctx context.Context, target PublicationTarget, repository RepositoryEvidenceSource, mappings ExternalMappingRepository, execution ExecutionWorkspace, input ReconciliationPreviewInput, authorizer Authorizer) (ReconciliationPlan, error) {
	plan, _, err := s.previewReconciliation(ctx, target, repository, mappings, execution, input, authorizer)
	return plan, err
}

func (s *Service) previewReconciliation(ctx context.Context, target PublicationTarget, repository RepositoryEvidenceSource, mappings ExternalMappingRepository, execution ExecutionWorkspace, input ReconciliationPreviewInput, authorizer Authorizer) (ReconciliationPlan, ExternalMappingState, error) {
	var empty ReconciliationPlan
	if authorizer == nil {
		return empty, ExternalMappingState{}, fmt.Errorf("reconciliation preview requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionRead); err != nil {
		return empty, ExternalMappingState{}, err
	}
	if target == nil || repository == nil || mappings == nil {
		return empty, ExternalMappingState{}, fmt.Errorf("reconciliation capabilities are unavailable")
	}
	base, err := mappings.Load(ctx)
	if err != nil {
		return empty, ExternalMappingState{}, err
	}
	if base.SchemaVersion != IntegrationContractVersion || base.Revision == "" {
		return empty, base, reconciliationConflict("mapping contract changed; review a fresh preview")
	}
	evidence, err := repository.Current(ctx)
	if err != nil {
		return empty, base, err
	}
	if err := validateRepositoryEvidence(evidence); err != nil {
		return empty, base, err
	}
	if !sameExternalIdentity(input.Publication.Target, evidence.Repository) {
		return empty, base, reconciliationManual("publication target disagrees with the checked-out repository")
	}
	publicationInput, err := mappedReconciliationInput(input.Publication, base)
	if err != nil {
		return empty, base, err
	}
	publication, err := s.previewPublication(ctx, target, publicationInput)
	if err != nil {
		return empty, base, err
	}
	for _, action := range publication.Actions {
		if action.Artifact != nil && action.Action == MutationCreate {
			return empty, base, reconciliationManual("an unmapped planning artifact requires explicit publication or adoption")
		}
		if action.Group != nil && action.Action == MutationCreate {
			return empty, base, reconciliationManual("an unmapped coordination group requires explicit publication")
		}
		if action.Workspace != nil && publicationWrites(action) {
			return empty, base, reconciliationManual("execution workspace drift is deferred to execution-workspace reconciliation")
		}
	}
	plan := ReconciliationPlan{
		SchemaVersion: IntegrationContractVersion, Repository: evidence, Publication: publication,
		ExpectedMappingRevision: base.Revision,
	}
	if len(input.Execution) == 0 {
		return plan, base, nil
	}
	if execution == nil || len(base.WorkspaceReferences) != 1 {
		return empty, base, reconciliationManual("execution reconciliation requires one durable workspace mapping")
	}
	snapshot, err := execution.Inspect(ctx, base.WorkspaceReferences[0])
	if err != nil {
		return empty, base, err
	}
	if err := validateExecutionSnapshot(snapshot, base.WorkspaceReferences[0]); err != nil {
		return empty, base, err
	}
	published := map[planning.ArtifactRef]ExternalReference{}
	for _, action := range publication.Actions {
		if action.Artifact != nil && action.Artifact.Reference != nil {
			published[action.Artifact.Artifact] = *action.Artifact.Reference
		}
	}
	seen := map[planning.ArtifactRef]bool{}
	for _, intent := range input.Execution {
		if intent.Artifact.Validate() != nil || validateExecutionStatus(intent.Status) != nil || seen[intent.Artifact] {
			return empty, base, reconciliationManual("execution intent contains invalid or duplicate artifacts")
		}
		seen[intent.Artifact] = true
		issue, ok := published[intent.Artifact]
		if !ok || intent.PublishedWork.OpaqueID != "" && !reconcilesExternalIdentity(intent.PublishedWork, issue) {
			return empty, base, reconciliationManual("execution intent disagrees with durable publication identity")
		}
		var item *ExecutionWorkItem
		for i := range snapshot.Items {
			if snapshot.Items[i].Artifact == intent.Artifact {
				if item != nil || !sameExternalLocation(snapshot.Items[i].PublishedWork, issue) {
					return empty, base, reconciliationManual("execution workspace has ambiguous artifact membership")
				}
				copy := snapshot.Items[i]
				item = &copy
			}
		}
		if item == nil {
			plan.Attachments = append(plan.Attachments, ExecutionAttachRequest{Workspace: snapshot.Workspace, Artifact: intent.Artifact, PublishedWork: issue})
			continue
		}
		if item.Status != intent.Status {
			if !executionStatusSupported(snapshot.Fields, intent.Status) {
				return empty, base, reconciliationManual("required semantic Status option is unavailable")
			}
			plan.Execution = append(plan.Execution, ExecutionStatusRequest{Workspace: snapshot.Workspace, WorkItem: item.Reference, Status: intent.Status})
		}
	}
	slices.SortFunc(plan.Attachments, func(a, b ExecutionAttachRequest) int {
		return strings.Compare(string(a.Artifact.ID), string(b.Artifact.ID))
	})
	slices.SortFunc(plan.Execution, func(a, b ExecutionStatusRequest) int { return strings.Compare(a.WorkItem.URL, b.WorkItem.URL) })
	return plan, base, nil
}

// ApplyReconciliation rechecks the complete reviewed plan, applies ordered safe
// publication repairs, and records repository evidence only after provider and
// mapping completion. Remote writes are never rolled back.
func (s *Service) ApplyReconciliation(ctx context.Context, target PublicationTarget, repository RepositoryEvidenceSource, mappings ExternalMappingRepository, execution ExecutionWorkspace, input ReconciliationApplyInput, authorizer Authorizer, events EventSink) (ReconciliationResult, error) {
	result := ReconciliationResult{SchemaVersion: IntegrationContractVersion, Publication: PublicationResult{SchemaVersion: IntegrationContractVersion}}
	if !input.Confirmed {
		return result, ErrConfirmationRequired
	}
	if authorizer == nil {
		return result, fmt.Errorf("reconciliation requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionPublish); err != nil {
		return result, err
	}
	if len(input.Intent.Execution) > 0 {
		if err := authorizer.Require(ctx, PermissionExecute); err != nil {
			return result, err
		}
	}
	if events == nil {
		return result, ErrEventSinkRequired
	}
	if input.ExpectedPlan.SchemaVersion != IntegrationContractVersion || input.ExpectedPlan.Publication.SchemaVersion != IntegrationContractVersion || len(input.ExpectedPlan.Publication.Actions) == 0 || input.ExpectedPlan.ExpectedMappingRevision == "" {
		return result, reconciliationApplyConflict()
	}
	fresh, base, err := s.previewReconciliation(ctx, target, repository, mappings, execution, input.Intent, authorizer)
	if err != nil {
		return result, err
	}
	result.Mappings = base
	if !samePublicationJSON(fresh, input.ExpectedPlan) {
		return result, reconciliationApplyConflict()
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	providerWrites := false
	for _, action := range fresh.Publication.Actions {
		providerWrites = providerWrites || publicationWrites(action)
	}
	if providerWrites {
		result.Publication, err = target.Apply(ctx, fresh.Publication)
		if evidenceErr := validatePublicationCompletion(fresh.Publication, result.Publication, err); evidenceErr != nil {
			err = errors.Join(err, evidenceErr)
		}
	} else {
		for _, action := range fresh.Publication.Actions {
			evidence := PublicationActionEvidence{Action: action}
			if ref := publicationActionReference(action); ref != nil {
				evidence.References = []ExternalReference{*ref}
			}
			result.Publication.Completed = append(result.Publication.Completed, evidence)
		}
	}
	providerChanged := publicationEvidenceChanged(fresh.Publication, result.Publication)
	executionChanged := false
	if err == nil && (len(fresh.Attachments) > 0 || len(fresh.Execution) > 0) {
		var applied []ExecutionWorkItem
		applied, executionChanged, err = applyReconciliationExecution(ctx, execution, fresh)
		result.Execution = applied
	}
	if err == nil {
		desired, mergeErr := mergeCompletedAdoptionMappings(base, fresh.Publication, result.Publication.Completed)
		if mergeErr != nil {
			class := IntegrationManualRemediationRequired
			var integration *IntegrationError
			if errors.As(mergeErr, &integration) && integration.Class != "" {
				class = integration.Class
			}
			err = &IntegrationError{Class: class, Operation: "reconciliation.apply", Message: "completed provider evidence cannot be persisted safely", ManualFallbackAllowed: class == IntegrationManualRemediationRequired, Err: mergeErr}
		} else {
			result.Mappings = desired
			sameEvidence := base.LastReconciliation != nil && samePublicationJSON(base.LastReconciliation.Repository, fresh.Repository)
			evidence := ReconciliationEvidence{Repository: fresh.Repository, CompletedAt: s.now().UTC()}
			if sameEvidence {
				evidence = *base.LastReconciliation
			}
			desired.LastReconciliation = &evidence
			result.Evidence = evidence
			if !samePublicationJSON(desired, base) {
				saved, saveErr := mappings.Save(ctx, desired, base.Revision)
				if saveErr != nil {
					err = saveErr
					if providerChanged {
						err = &IntegrationError{Class: IntegrationPartialFailure, Operation: "reconciliation.mapping", Message: "provider changed before reconciliation evidence could be saved; inspect before retrying", Err: err}
					}
				} else {
					result.Mappings = saved
				}
			}
		}
	}
	changed := providerChanged || executionChanged || result.Mappings.Revision != base.Revision
	if (providerChanged || executionChanged) && err != nil {
		var integration *IntegrationError
		if !errors.As(err, &integration) || integration.Class != IntegrationPartialFailure {
			err = &IntegrationError{Class: IntegrationPartialFailure, Operation: "reconciliation.apply", Message: "reconciliation changed provider state before failure", Err: err}
		}
	}
	if changed {
		source := fresh.Repository.Repository
		event := Event{Name: EventIntegrationReconciled, ModuleID: s.moduleID, Source: &source, Outcome: MutationUpdate, OccurredAt: s.now().UTC()}
		result.Event = &event
		if auditErr := events.Publish(context.WithoutCancel(ctx), event); auditErr != nil {
			err = errors.Join(err, &IntegrationError{Class: IntegrationPartialFailure, Operation: "reconciliation.audit", Message: "reconciliation changed state but audit persistence failed", Err: auditErr})
		}
	}
	return result, err
}

func applyReconciliationExecution(ctx context.Context, workspace ExecutionWorkspace, plan ReconciliationPlan) ([]ExecutionWorkItem, bool, error) {
	if workspace == nil {
		return nil, false, reconciliationManual("execution workspace is unavailable")
	}
	result := make([]ExecutionWorkItem, 0, len(plan.Attachments)+len(plan.Execution))
	changed := false
	for _, planned := range plan.Attachments {
		snapshot, err := workspace.Inspect(ctx, planned.Workspace)
		if err != nil {
			return result, changed, err
		}
		request := planned
		request.Workspace = snapshot.Workspace
		already := false
		for _, item := range snapshot.Items {
			if sameExternalLocation(item.PublishedWork, planned.PublishedWork) {
				if item.Artifact != planned.Artifact {
					return result, changed, reconciliationManual("published work maps to a different artifact")
				}
				result = append(result, item)
				already = true
			}
		}
		if already {
			continue
		}
		item, err := workspace.Attach(ctx, request)
		if err != nil {
			return result, changed, reconciliationExecutionFailure(changed, err)
		}
		result = append(result, item)
		changed = true
	}
	for _, planned := range plan.Execution {
		snapshot, err := workspace.Inspect(ctx, planned.Workspace)
		if err != nil {
			return result, changed, reconciliationExecutionFailure(changed, err)
		}
		request := planned
		request.Workspace = snapshot.Workspace
		var current *ExecutionWorkItem
		for i := range snapshot.Items {
			if snapshot.Items[i].Reference.OpaqueID == planned.WorkItem.OpaqueID {
				copy := snapshot.Items[i]
				current = &copy
			}
		}
		if current == nil {
			return result, changed, reconciliationExecutionFailure(changed, reconciliationManual("tracked Project item is missing"))
		}
		if current.Reference.Revision != planned.WorkItem.Revision {
			return result, changed, reconciliationExecutionFailure(changed, reconciliationApplyConflict())
		}
		request.WorkItem = current.Reference
		if current.Status == planned.Status {
			result = append(result, *current)
			continue
		}
		item, err := workspace.ApplyStatus(ctx, request)
		if err != nil {
			return result, changed, reconciliationExecutionFailure(changed, err)
		}
		result = append(result, item)
		changed = true
	}
	return result, changed, nil
}

func reconciliationExecutionFailure(changed bool, err error) error {
	if !changed {
		return err
	}
	return &IntegrationError{Class: IntegrationPartialFailure, Operation: "reconciliation.execution", Message: "execution workspace changed before reconciliation stopped; inspect before retrying", Err: err}
}

func mappedReconciliationInput(input PublicationPreviewInput, state ExternalMappingState) (PublicationPreviewInput, error) {
	result := input
	result.Artifacts = slices.Clone(input.Artifacts)
	result.Candidates = nil
	if len(state.SourceReferences) > 1 {
		return result, reconciliationManual("durable mappings contain multiple collaboration sources")
	}
	if len(state.SourceReferences) == 1 {
		mapped := state.SourceReferences[0]
		if !validPublicationReference(mapped) || mapped.Provider != input.Target.Provider {
			return result, reconciliationManual("collaboration source mapping has an invalid identity")
		}
		if input.Source != nil && !sameExternalIdentity(*input.Source, mapped) {
			return result, reconciliationManual("canonical collaboration source disagrees with durable mappings")
		}
		copyRef := mapped
		result.Source = &copyRef
	}
	byArtifact := map[planning.ArtifactRef]ExternalReference{}
	for _, mapping := range state.ArtifactReferences {
		if mapping.Artifact.Validate() != nil || !validPublicationReference(mapping.Reference) || mapping.Reference.Provider != input.Target.Provider {
			return result, reconciliationManual("artifact mappings contain an invalid identity")
		}
		if _, duplicate := byArtifact[mapping.Artifact]; duplicate {
			return result, reconciliationManual("artifact mappings contain duplicate identities")
		}
		byArtifact[mapping.Artifact] = mapping.Reference
	}
	for i := range result.Artifacts {
		ref, found := byArtifact[result.Artifacts[i].Artifact]
		if !found {
			return result, reconciliationManual("planning artifact has no durable provider mapping")
		}
		if result.Artifacts[i].Reference != nil && !sameExternalIdentity(*result.Artifacts[i].Reference, ref) {
			return result, reconciliationManual("canonical artifact reference disagrees with durable mappings")
		}
		copyRef := ref
		result.Artifacts[i].Reference = &copyRef
	}
	if input.Group != nil {
		var groupRef *ExternalReference
		members := map[planning.ArtifactRef]bool{}
		for _, member := range input.Group.Members {
			members[member] = true
		}
		if len(members) == 0 {
			for _, artifact := range result.Artifacts {
				members[artifact.Artifact] = true
			}
		}
		mappedMembers := map[planning.ArtifactRef]bool{}
		for _, mapping := range state.GroupReferences {
			if !members[mapping.Artifact] {
				continue
			}
			if mapping.Reference.Provider != input.Target.Provider || !validPublicationReference(mapping.Reference) {
				return result, reconciliationManual("publication group mapping has an invalid identity")
			}
			if groupRef != nil && !sameExternalIdentity(*groupRef, mapping.Reference) {
				return result, reconciliationManual("publication group mappings disagree")
			}
			copyRef := mapping.Reference
			groupRef = &copyRef
			mappedMembers[mapping.Artifact] = true
		}
		if groupRef == nil || len(mappedMembers) != len(members) {
			return result, reconciliationManual("publication group has no durable provider mapping")
		}
		group := *input.Group
		group.Reference = groupRef
		result.Group = &group
	}
	if input.Workspace != nil {
		workspace := *input.Workspace
		switch workspace.Choice {
		case PublicationWorkspaceSkip:
			if len(state.WorkspaceReferences) != 0 {
				return result, reconciliationManual("workspace skip disagrees with durable workspace mappings")
			}
		case PublicationWorkspaceConnect:
			if len(state.WorkspaceReferences) != 1 || !validPublicationReference(state.WorkspaceReferences[0]) || state.WorkspaceReferences[0].Provider != input.Target.Provider {
				return result, reconciliationManual("execution workspace has no unique durable provider mapping")
			}
			mapped := state.WorkspaceReferences[0]
			if workspace.Reference != nil && !sameExternalIdentity(*workspace.Reference, mapped) {
				return result, reconciliationManual("selected execution workspace disagrees with durable mappings")
			}
			workspace.Reference = &mapped
		case PublicationWorkspaceCreate:
			return result, reconciliationManual("execution workspace creation requires explicit publication")
		default:
			return result, reconciliationManual("execution workspace decision is invalid")
		}
		result.Workspace = &workspace
	}
	return result, nil
}

func validateRepositoryEvidence(evidence RepositoryEvidence) error {
	if evidence.SchemaVersion != IntegrationContractVersion || !validPublicationReference(evidence.Repository) || evidence.Repository.Kind != "repository" ||
		evidence.DefaultRef == "" || evidence.DefaultRef != strings.TrimSpace(evidence.DefaultRef) || evidence.CurrentRef == "" || evidence.CurrentRef != strings.TrimSpace(evidence.CurrentRef) || !validRepositoryCommit(evidence.Commit) {
		return reconciliationManual("repository evidence is incomplete")
	}
	if change := evidence.PlanningChangeRequest; change != nil {
		if !validPublicationReference(change.Reference) || change.Reference.Provider != evidence.Repository.Provider || change.Reference.Kind != "pull_request" ||
			change.HeadRef != evidence.CurrentRef || change.BaseRef != evidence.DefaultRef || change.Reference.Revision != evidence.Commit {
			return reconciliationManual("change-request evidence disagrees with the checked-out repository")
		}
		switch change.State {
		case ChangeRequestOpen, ChangeRequestClosed, ChangeRequestMerged:
		default:
			return reconciliationManual("change-request evidence has an invalid state")
		}
	}
	return nil
}

func validRepositoryCommit(value string) bool {
	if len(value) < 6 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}

func reconciliationConflict(message string) error {
	return &IntegrationError{Class: IntegrationRevisionConflict, Operation: "reconciliation.preview", Message: message}
}

func reconciliationManual(message string) error {
	return &IntegrationError{Class: IntegrationManualRemediationRequired, Operation: "reconciliation.preview", Message: message, ManualFallbackAllowed: true}
}

func reconciliationApplyConflict() error {
	return &IntegrationError{Class: IntegrationRevisionConflict, Operation: "reconciliation.apply", Message: "reconciliation preview is missing or changed; review a fresh preview"}
}
