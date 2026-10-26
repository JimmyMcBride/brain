package application

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JimmyMcBride/brain/planning"
)

// EventIntegrationAdopted records durable adoption changes.
const EventIntegrationAdopted = "planning.integration.adopted"

// AdoptionPreviewInput binds canonical Planning intent to explicitly selected
// provider objects. Candidates are selectors; the provider returns fresh stable
// identities and revisions in the resulting publication plan.
type AdoptionPreviewInput struct {
	Target     ExternalReference             `json:"target"`
	Source     *ExternalReference            `json:"source,omitempty"`
	Group      *PublicationGroup             `json:"group,omitempty"`
	Workspace  *PublicationWorkspaceDecision `json:"workspace,omitempty"`
	Artifacts  []PublicationArtifact         `json:"artifacts"`
	Candidates []ArtifactExternalReference   `json:"candidates"`
}

// AdoptionPlan contains the reviewed provider actions and complete desired
// mapping state. The mapping revision guards application against stale state.
type AdoptionPlan struct {
	SchemaVersion int                  `json:"schema_version"`
	Publication   PublicationPlan      `json:"publication"`
	Mappings      ExternalMappingState `json:"mappings"`
}

// AdoptionApplyInput requires a fresh preview to match the reviewed plan.
type AdoptionApplyInput struct {
	Intent       AdoptionPreviewInput `json:"intent"`
	ExpectedPlan AdoptionPlan         `json:"expected_plan"`
	Confirmed    bool                 `json:"confirmed"`
}

// PreviewAdoption resolves explicitly selected existing objects, classifies
// required provider changes, and prepares a complete revision-guarded mapping
// replacement without mutating provider or local state.
func (s *Service) PreviewAdoption(ctx context.Context, target PublicationTarget, mappings ExternalMappingRepository, input AdoptionPreviewInput, authorizer Authorizer) (AdoptionPlan, error) {
	var empty AdoptionPlan
	if authorizer == nil {
		return empty, fmt.Errorf("adoption preview requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionRead); err != nil {
		return empty, err
	}
	if target == nil || mappings == nil {
		return empty, fmt.Errorf("adoption capabilities are unavailable")
	}
	if input.Target.Provider == "" || input.Target.Kind == "" || input.Target.OpaqueID == "" {
		return empty, fmt.Errorf("adoption target requires a stable identity")
	}
	artifacts, err := orderedPublicationArtifacts(input.Artifacts)
	if err != nil {
		return empty, err
	}
	candidates, err := orderedAdoptionCandidates(artifacts, input.Candidates)
	if err != nil {
		return empty, err
	}
	base, err := mappings.Load(ctx)
	if err != nil {
		return empty, err
	}
	if base.SchemaVersion != IntegrationContractVersion {
		return empty, adoptionConflict("mapping contract changed; review a fresh preview")
	}
	refs := make([]planning.ArtifactRef, len(artifacts))
	for i, artifact := range artifacts {
		refs[i] = artifact.Artifact
	}
	snapshot, err := target.Inspect(ctx, PublicationInspectRequest{Target: input.Target, Source: input.Source, Artifacts: refs, Candidates: candidates, Group: input.Group, Workspace: input.Workspace})
	if err != nil {
		return empty, err
	}
	publicationInput := PublicationPreviewInput{Target: input.Target, Source: input.Source, Group: input.Group, Workspace: input.Workspace, Artifacts: artifacts}
	publication, err := publicationPlanFromSnapshot(publicationInput, snapshot)
	if err != nil {
		return empty, err
	}
	resolved := map[planning.ArtifactRef]ExternalReference{}
	for _, action := range publication.Actions {
		if action.Artifact == nil {
			continue
		}
		if action.Action == MutationCreate || action.Artifact.Reference == nil {
			return empty, adoptionConflict("every adopted artifact must resolve to one existing provider object")
		}
		resolved[action.Artifact.Artifact] = *action.Artifact.Reference
	}
	for _, candidate := range candidates {
		ref, found := resolved[candidate.Artifact]
		if !found || !sameExternalLocation(candidate.Reference, ref) ||
			(candidate.Reference.OpaqueID != candidate.Reference.URL && candidate.Reference.OpaqueID != ref.OpaqueID) {
			return empty, adoptionConflict("provider candidate identity changed during adoption preview")
		}
	}
	desired, err := mergeAdoptionMappings(base, publication)
	if err != nil {
		return empty, err
	}
	return AdoptionPlan{SchemaVersion: IntegrationContractVersion, Publication: publication, Mappings: desired}, nil
}

// ApplyAdoption rechecks the complete reviewed plan, applies provider updates,
// and persists mappings only after provider success. Provider changes are never
// rolled back when mapping or audit persistence fails.
func (s *Service) ApplyAdoption(ctx context.Context, target PublicationTarget, mappings ExternalMappingRepository, input AdoptionApplyInput, authorizer Authorizer, events EventSink) (AdoptionResult, error) {
	result := AdoptionResult{SchemaVersion: IntegrationContractVersion}
	if !input.Confirmed {
		return result, ErrConfirmationRequired
	}
	if authorizer == nil {
		return result, fmt.Errorf("adoption requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionPublish); err != nil {
		return result, err
	}
	if events == nil {
		return result, ErrEventSinkRequired
	}
	if input.ExpectedPlan.SchemaVersion != IntegrationContractVersion || input.ExpectedPlan.Publication.SchemaVersion != IntegrationContractVersion || input.ExpectedPlan.Mappings.SchemaVersion != IntegrationContractVersion {
		return result, adoptionApplyConflict()
	}
	fresh, err := s.PreviewAdoption(ctx, target, mappings, input.Intent, authorizer)
	if err != nil {
		return result, err
	}
	result.Mappings = fresh.Mappings
	if !samePublicationJSON(fresh, input.ExpectedPlan) {
		return result, adoptionApplyConflict()
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	evidence := PublicationResult{SchemaVersion: IntegrationContractVersion}
	providerWrites := false
	for _, action := range fresh.Publication.Actions {
		providerWrites = providerWrites || publicationWrites(action)
	}
	if providerWrites {
		evidence, err = target.Apply(ctx, fresh.Publication)
		if evidenceErr := validatePublicationCompletion(fresh.Publication, evidence, err); evidenceErr != nil {
			err = errors.Join(err, evidenceErr)
		}
	} else {
		for _, action := range fresh.Publication.Actions {
			item := PublicationActionEvidence{Action: action}
			if ref := publicationActionReference(action); ref != nil {
				item.References = []ExternalReference{*ref}
			}
			evidence.Completed = append(evidence.Completed, item)
		}
	}
	result.Actions = evidence.Completed
	result.Failed = evidence.Failed
	result.ManualFallbackAllowed = evidence.ManualFallbackAllowed
	result.ManualFallbackReason = evidence.ManualFallbackReason
	providerChanged := publicationEvidenceChanged(fresh.Publication, evidence)
	if err == nil {
		desired, mergeErr := mergeCompletedAdoptionMappings(fresh.Mappings, fresh.Publication, evidence.Completed)
		if mergeErr != nil {
			err = mergeErr
		} else {
			result.Mappings = desired
		}
		var saved ExternalMappingState
		var saveErr error
		if err == nil {
			saved, saveErr = mappings.Save(ctx, desired, fresh.Mappings.Revision)
		}
		if saveErr != nil {
			err = saveErr
			if providerChanged {
				err = &IntegrationError{Class: IntegrationPartialFailure, Operation: "adoption.mapping", Message: "provider changed before adoption mappings could be saved; inspect before retrying", Err: err}
			}
		} else if err == nil {
			mappingChanged := saved.Revision != fresh.Mappings.Revision
			result.Mappings = saved
			providerChanged = providerChanged || mappingChanged
		}
	}
	if providerChanged {
		if err != nil {
			var integration *IntegrationError
			if !errors.As(err, &integration) || integration.Class != IntegrationPartialFailure {
				err = &IntegrationError{Class: IntegrationPartialFailure, Operation: "adoption.apply", Message: "adoption changed state before failure", Err: err}
			}
		}
		source := fresh.Publication.Target
		event := Event{Name: EventIntegrationAdopted, ModuleID: s.moduleID, Source: &source, Outcome: MutationUpdate, OccurredAt: s.now().UTC()}
		result.Event = &event
		if auditErr := events.Publish(context.WithoutCancel(ctx), event); auditErr != nil {
			err = errors.Join(err, &IntegrationError{Class: IntegrationPartialFailure, Operation: "adoption.audit", Message: "adoption changed state but audit persistence failed", Err: auditErr})
		}
	}
	return result, err
}

func orderedAdoptionCandidates(artifacts []PublicationArtifact, input []ArtifactExternalReference) ([]ArtifactExternalReference, error) {
	if len(input) != len(artifacts) {
		return nil, adoptionConflict("adoption requires exactly one candidate for every artifact")
	}
	wanted := map[planning.ArtifactRef]bool{}
	for _, artifact := range artifacts {
		wanted[artifact.Artifact] = true
	}
	seenArtifacts := map[planning.ArtifactRef]bool{}
	seenLocations := map[[4]string]bool{}
	seenOpaqueIDs := map[[3]string]bool{}
	candidates := slices.Clone(input)
	for _, candidate := range candidates {
		ref := candidate.Reference
		provider, kind := strings.TrimSpace(ref.Provider), strings.TrimSpace(ref.Kind)
		opaqueIDValue, displayID, url := strings.TrimSpace(ref.OpaqueID), strings.TrimSpace(ref.DisplayID), strings.TrimSpace(ref.URL)
		location := [4]string{provider, kind, displayID, url}
		opaqueID := [3]string{provider, kind, opaqueIDValue}
		if !wanted[candidate.Artifact] || seenArtifacts[candidate.Artifact] || provider == "" || kind == "" || opaqueIDValue == "" || displayID == "" || url == "" ||
			provider != ref.Provider || kind != ref.Kind || opaqueIDValue != ref.OpaqueID || displayID != ref.DisplayID || url != ref.URL || seenLocations[location] || seenOpaqueIDs[opaqueID] {
			return nil, adoptionConflict("adoption candidates contain an invalid or duplicate identity")
		}
		seenArtifacts[candidate.Artifact] = true
		seenLocations[location] = true
		seenOpaqueIDs[opaqueID] = true
	}
	slices.SortFunc(candidates, func(a, b ArtifactExternalReference) int { return artifactLess(a.Artifact, b.Artifact) })
	return candidates, nil
}

func mergeAdoptionMappings(base ExternalMappingState, publication PublicationPlan) (ExternalMappingState, error) {
	desired := base
	desired.ArtifactReferences = slices.Clone(base.ArtifactReferences)
	desired.SourceReferences = slices.Clone(base.SourceReferences)
	desired.GroupReferences = slices.Clone(base.GroupReferences)
	desired.WorkspaceReferences = slices.Clone(base.WorkspaceReferences)
	owners := map[[4]string]planning.ArtifactRef{}
	indices := map[planning.ArtifactRef]int{}
	for i, mapping := range desired.ArtifactReferences {
		location := externalLocationKey(mapping.Reference)
		if mapping.Artifact.Validate() != nil || location == ([4]string{}) {
			return desired, adoptionConflict("existing artifact mapping has an invalid identity")
		}
		if _, duplicate := indices[mapping.Artifact]; duplicate {
			return desired, adoptionConflict("existing mappings contain a duplicate artifact identity")
		}
		if owner, duplicate := owners[location]; duplicate && owner != mapping.Artifact {
			return desired, adoptionConflict("one provider object is already mapped to multiple artifacts")
		}
		owners[location] = mapping.Artifact
		indices[mapping.Artifact] = i
	}
	for _, action := range publication.Actions {
		if action.Artifact == nil {
			continue
		}
		artifact := action.Artifact.Artifact
		if action.Artifact.Reference == nil {
			return desired, adoptionConflict("adopted artifact has no stable provider identity")
		}
		ref := *action.Artifact.Reference
		location := externalLocationKey(ref)
		if owner, duplicate := owners[location]; duplicate && owner != artifact {
			return desired, adoptionConflict("provider object is already mapped to another artifact")
		}
		if index, found := indices[artifact]; found {
			if !sameExternalLocation(desired.ArtifactReferences[index].Reference, ref) {
				return desired, adoptionConflict("artifact is already mapped to another provider object")
			}
			continue
		}
		indices[artifact] = len(desired.ArtifactReferences)
		owners[location] = artifact
		desired.ArtifactReferences = append(desired.ArtifactReferences, ArtifactExternalReference{Artifact: artifact, Reference: ref})
	}
	slices.SortFunc(desired.ArtifactReferences, func(a, b ArtifactExternalReference) int { return artifactLess(a.Artifact, b.Artifact) })
	if len(desired.SourceReferences) > 1 {
		return desired, &IntegrationError{Class: IntegrationUnsupportedCapability, Operation: "adoption.preview", Message: "mapping metadata cannot retain multiple collaboration sources"}
	}
	if publication.Source != nil {
		if len(desired.SourceReferences) == 0 {
			desired.SourceReferences = []ExternalReference{*publication.Source}
		} else if len(desired.SourceReferences) != 1 || !sameExternalLocation(desired.SourceReferences[0], *publication.Source) {
			return desired, &IntegrationError{Class: IntegrationUnsupportedCapability, Operation: "adoption.preview", Message: "mapping metadata cannot retain multiple collaboration sources"}
		}
	}
	for _, action := range publication.Actions {
		switch {
		case action.Group != nil && action.Group.Reference != nil:
			if err := replaceGroupMappings(&desired, action.Group.Members, *action.Group.Reference); err != nil {
				return desired, err
			}
		case action.Workspace != nil && action.Workspace.Choice != PublicationWorkspaceSkip && action.Workspace.Reference != nil:
			desired.WorkspaceReferences = []ExternalReference{*action.Workspace.Reference}
		}
	}
	return desired, nil
}

func mergeCompletedAdoptionMappings(base ExternalMappingState, publication PublicationPlan, completed []PublicationActionEvidence) (ExternalMappingState, error) {
	desired, err := mergeAdoptionMappings(base, publication)
	if err != nil {
		return desired, err
	}
	for i, evidence := range completed {
		if i >= len(publication.Actions) || !samePublicationJSON(evidence.Action, publication.Actions[i]) {
			return desired, adoptionConflict("publication evidence changed before mapping persistence")
		}
		action := publication.Actions[i]
		ref := completedActionReference(action, evidence.References)
		switch {
		case action.Group != nil:
			if ref == nil {
				return desired, adoptionConflict("completed group action has no stable provider identity")
			}
			if err := replaceGroupMappings(&desired, action.Group.Members, *ref); err != nil {
				return desired, err
			}
		case action.Workspace != nil && action.Workspace.Choice != PublicationWorkspaceSkip:
			if ref == nil {
				return desired, adoptionConflict("completed workspace action has no stable provider identity")
			}
			desired.WorkspaceReferences = []ExternalReference{*ref}
		}
	}
	return desired, nil
}

func completedActionReference(action PublicationApplyAction, references []ExternalReference) *ExternalReference {
	if expected := publicationActionReference(action); expected != nil {
		for i := range references {
			if sameExternalIdentity(references[i], *expected) {
				return &references[i]
			}
		}
		return nil
	}
	if len(references) == 0 {
		return nil
	}
	return &references[0]
}

func replaceGroupMappings(state *ExternalMappingState, members []planning.ArtifactRef, reference ExternalReference) error {
	wanted := map[planning.ArtifactRef]bool{}
	for _, member := range members {
		if member.Validate() != nil || wanted[member] {
			return adoptionConflict("publication group has invalid member identities")
		}
		wanted[member] = true
	}
	retained := state.GroupReferences[:0]
	for _, mapping := range state.GroupReferences {
		if !wanted[mapping.Artifact] {
			retained = append(retained, mapping)
		}
	}
	for _, member := range members {
		retained = append(retained, ArtifactExternalReference{Artifact: member, Reference: reference})
	}
	slices.SortFunc(retained, func(a, b ArtifactExternalReference) int { return artifactLess(a.Artifact, b.Artifact) })
	state.GroupReferences = retained
	return nil
}

func publicationEvidenceChanged(plan PublicationPlan, evidence PublicationResult) bool {
	for i, completed := range evidence.Completed {
		if i < len(plan.Actions) && samePublicationJSON(completed.Action, plan.Actions[i]) && publicationWrites(completed.Action) {
			return true
		}
	}
	return evidence.Failed != nil && len(evidence.Failed.References) > 0 && len(evidence.Completed) < len(plan.Actions) &&
		samePublicationJSON(evidence.Failed.Action, plan.Actions[len(evidence.Completed)]) && publicationWrites(evidence.Failed.Action)
}

func sameExternalLocation(a, b ExternalReference) bool {
	return a.Provider == b.Provider && a.Kind == b.Kind && a.DisplayID == b.DisplayID && a.URL == b.URL
}

func externalLocationKey(ref ExternalReference) [4]string {
	return [4]string{strings.TrimSpace(ref.Provider), strings.TrimSpace(ref.Kind), strings.TrimSpace(ref.DisplayID), strings.TrimSpace(ref.URL)}
}

func adoptionConflict(message string) error {
	return &IntegrationError{Class: IntegrationAmbiguousIdentity, Operation: "adoption.preview", Message: message}
}

func adoptionApplyConflict() error {
	return &IntegrationError{Class: IntegrationRevisionConflict, Operation: "adoption.apply", Message: "adoption preview is missing or changed; review a fresh preview"}
}
