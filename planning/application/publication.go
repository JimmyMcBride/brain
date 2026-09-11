package application

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/JimmyMcBride/brain/planning"
)

// PublicationPreviewInput supplies canonical artifact intent, including complete
// content. The planner never synthesizes spec text or matches objects by title.
type PublicationPreviewInput struct {
	Target     ExternalReference             `json:"target"`
	Source     *ExternalReference            `json:"source,omitempty"`
	Artifacts  []PublicationArtifact         `json:"artifacts"`
	Candidates []ArtifactExternalReference   `json:"candidates,omitempty"`
	Group      *PublicationGroup             `json:"group,omitempty"`
	Workspace  *PublicationWorkspaceDecision `json:"workspace,omitempty"`
}

// PreviewPublication inspects provider evidence and returns a deterministic plan
// without applying it or changing mappings. All relationship endpoints must be
// present in Artifacts so dependencies can be validated before provider work.
func (s *Service) PreviewPublication(ctx context.Context, target PublicationTarget, input PublicationPreviewInput, authorizer Authorizer) (PublicationPlan, error) {
	var empty PublicationPlan
	if authorizer == nil {
		return empty, fmt.Errorf("publication preview requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionRead); err != nil {
		return empty, err
	}
	if target == nil {
		return empty, fmt.Errorf("publication target is unavailable")
	}
	if input.Target.Provider == "" || input.Target.Kind == "" || input.Target.OpaqueID == "" {
		return empty, fmt.Errorf("publication target requires a stable identity")
	}
	artifacts, err := orderedPublicationArtifacts(input.Artifacts)
	if err != nil {
		return empty, err
	}
	group, workspace, err := publicationCoordinationInput(input.Target.Provider, input.Group, input.Workspace)
	if err != nil {
		return empty, err
	}
	if publicationRequiresWorkspaceChoice(artifacts) && workspace == nil {
		return empty, publicationConflict("publication with five or more specs requires an explicit workspace choice")
	}
	refs := make([]planning.ArtifactRef, len(artifacts))
	candidates := slices.Clone(input.Candidates)
	selected := map[planning.ArtifactRef]bool{}
	for _, candidate := range candidates {
		selected[candidate.Artifact] = true
	}
	for i, artifact := range artifacts {
		refs[i] = artifact.Artifact
		if artifact.Reference != nil && !selected[artifact.Artifact] {
			candidates = append(candidates, ArtifactExternalReference{Artifact: artifact.Artifact, Reference: *artifact.Reference})
		}
	}
	snapshot, err := target.Inspect(ctx, PublicationInspectRequest{Target: input.Target, Source: input.Source, Artifacts: refs, Candidates: candidates, Group: group, Workspace: workspace})
	if err != nil {
		return empty, err
	}
	return publicationPlanFromSnapshot(input, snapshot)
}

func publicationPlanFromSnapshot(input PublicationPreviewInput, snapshot PublicationSnapshot) (PublicationPlan, error) {
	var empty PublicationPlan
	artifacts, err := orderedPublicationArtifacts(input.Artifacts)
	if err != nil {
		return empty, err
	}
	group, workspace, err := publicationCoordinationInput(input.Target.Provider, input.Group, input.Workspace)
	if err != nil {
		return empty, err
	}
	if snapshot.SchemaVersion != IntegrationContractVersion || !sameExternalIdentity(snapshot.Target, input.Target) {
		return empty, publicationConflict("provider returned a different target or contract version")
	}
	existing := map[planning.ArtifactRef]PublicationArtifact{}
	remoteOwners := map[[3]string]planning.ArtifactRef{}
	for _, artifact := range snapshot.Artifacts {
		if _, duplicate := existing[artifact.Artifact]; duplicate {
			return empty, publicationConflict("multiple provider candidates match one artifact")
		}
		if artifact.Reference == nil || artifact.Reference.OpaqueID == "" || artifact.Reference.Kind == "" || artifact.Reference.Provider != input.Target.Provider {
			return empty, publicationConflict("provider candidate has no valid stable reference")
		}
		key := [3]string{artifact.Reference.Provider, artifact.Reference.Kind, artifact.Reference.OpaqueID}
		if owner, duplicate := remoteOwners[key]; duplicate && owner != artifact.Artifact {
			return empty, publicationConflict("provider identity maps to multiple artifacts")
		}
		remoteOwners[key] = artifact.Artifact
		existing[artifact.Artifact] = artifact
	}
	plan := PublicationPlan{SchemaVersion: IntegrationContractVersion, Target: snapshot.Target, Actions: []PublicationApplyAction{}}
	if input.Source != nil {
		source := *input.Source
		plan.Source = &source
	}
	groupAction, err := publicationGroupAction(input.Target.Provider, group, snapshot.Group)
	if err != nil {
		return empty, err
	}
	if groupAction != nil {
		plan.Actions = append(plan.Actions, *groupAction)
	}
	var relationships []PublicationRelationship
	for _, desired := range artifacts {
		current, found := existing[desired.Artifact]
		action := MutationCreate
		if desired.Reference != nil && (!found || !sameExternalIdentity(*desired.Reference, *current.Reference)) {
			return empty, publicationConflict("known artifact reference could not be reconciled")
		}
		if found {
			action = MutationUpdate
			if equivalentPublicationArtifact(desired, current) {
				action = MutationReuse
				if desired.Reference != nil {
					action = MutationUnchanged
				}
			}
			copyRef := *current.Reference
			desired.Reference = &copyRef
		}
		plan.Actions = append(plan.Actions, PublicationApplyAction{Kind: PublicationArtifactAction, Action: action, Artifact: &desired})
		if desired.Group != nil {
			relationships = append(relationships, PublicationRelationship{Kind: PublicationRelationshipContains, Source: *desired.Group, Target: desired.Artifact})
		}
		for _, dependency := range desired.Dependencies {
			relationships = append(relationships, PublicationRelationship{Kind: PublicationRelationshipDependsOn, Source: desired.Artifact, Target: dependency})
		}
	}
	known := map[PublicationRelationship]bool{}
	for _, relationship := range snapshot.Relationships {
		known[relationship] = true
	}
	for _, relationship := range relationships {
		action := MutationCreate
		if known[relationship] {
			action = MutationUnchanged
		}
		plan.Actions = append(plan.Actions, PublicationApplyAction{Kind: PublicationRelationshipAction, Action: action, Relationship: &relationship})
	}
	workspaceAction, err := publicationWorkspaceAction(input.Target.Provider, workspace, snapshot.Workspace)
	if err != nil {
		return empty, err
	}
	if workspaceAction != nil {
		plan.Actions = append(plan.Actions, *workspaceAction)
	}
	return plan, nil
}

func publicationCoordinationInput(provider string, group *PublicationGroup, workspace *PublicationWorkspaceDecision) (*PublicationGroup, *PublicationWorkspaceDecision, error) {
	var groupCopy *PublicationGroup
	if group != nil {
		value := *group
		if strings.TrimSpace(value.Title) == "" || value.Title != strings.TrimSpace(value.Title) || !validOptionalPublicationReference(value.Reference) {
			return nil, nil, publicationConflict("publication group has an invalid identity")
		}
		if value.Reference != nil && value.Reference.Provider != provider {
			return nil, nil, publicationConflict("publication group belongs to a different provider")
		}
		if value.Reference != nil {
			ref := *value.Reference
			value.Reference = &ref
		}
		groupCopy = &value
	}
	if workspace == nil {
		return groupCopy, nil, nil
	}
	value := *workspace
	if value.Title != strings.TrimSpace(value.Title) || value.Reason != strings.TrimSpace(value.Reason) || !validOptionalPublicationReference(value.Reference) {
		return nil, nil, publicationConflict("publication workspace decision has an invalid identity")
	}
	if value.Reference != nil && value.Reference.Provider != provider {
		return nil, nil, publicationConflict("publication workspace belongs to a different provider")
	}
	switch value.Choice {
	case PublicationWorkspaceCreate:
		if value.Title == "" || value.Reference != nil {
			return nil, nil, publicationConflict("creating a publication workspace requires a title and no existing reference")
		}
	case PublicationWorkspaceConnect:
		if value.Reference == nil {
			return nil, nil, publicationConflict("connecting a publication workspace requires an existing reference")
		}
	case PublicationWorkspaceSkip:
		if value.Title != "" || value.Reference != nil {
			return nil, nil, publicationConflict("skipping a publication workspace cannot select a workspace")
		}
	default:
		return nil, nil, publicationConflict("publication workspace choice must be create, connect, or skip")
	}
	if value.Reference != nil {
		ref := *value.Reference
		value.Reference = &ref
	}
	return groupCopy, &value, nil
}

func publicationRequiresWorkspaceChoice(artifacts []PublicationArtifact) bool {
	specs := 0
	for _, artifact := range artifacts {
		if artifact.Artifact.Kind == planning.ArtifactSpec {
			specs++
		}
	}
	return specs >= 5
}

func publicationGroupAction(provider string, desired, current *PublicationGroup) (*PublicationApplyAction, error) {
	if desired == nil {
		if current != nil {
			return nil, publicationConflict("provider returned an unrequested publication group")
		}
		return nil, nil
	}
	value := *desired
	if desired.Reference != nil && desired.Reference.Provider != provider {
		return nil, publicationConflict("publication group belongs to a different provider")
	}
	if current == nil {
		if desired.Reference != nil {
			return nil, publicationConflict("known publication group could not be reconciled")
		}
		return &PublicationApplyAction{Kind: PublicationGroupAction, Action: MutationCreate, Group: &value}, nil
	}
	if current.Reference == nil || !validPublicationReference(*current.Reference) || current.Reference.Provider != provider {
		return nil, publicationConflict("provider group has no valid stable reference")
	}
	if desired.Reference != nil && !sameExternalIdentity(*desired.Reference, *current.Reference) {
		return nil, publicationConflict("known publication group could not be reconciled")
	}
	ref := *current.Reference
	value.Reference = &ref
	action := MutationReuse
	if desired.Reference != nil {
		action = MutationUnchanged
	}
	if desired.Title != current.Title {
		action = MutationUpdate
	}
	return &PublicationApplyAction{Kind: PublicationGroupAction, Action: action, Group: &value}, nil
}

func publicationWorkspaceAction(provider string, desired, current *PublicationWorkspaceDecision) (*PublicationApplyAction, error) {
	if desired == nil {
		if current != nil {
			return nil, publicationConflict("provider returned an unrequested publication workspace")
		}
		return nil, nil
	}
	value := *desired
	if desired.Reference != nil && desired.Reference.Provider != provider {
		return nil, publicationConflict("publication workspace belongs to a different provider")
	}
	if desired.Choice == PublicationWorkspaceSkip {
		if current != nil {
			return nil, publicationConflict("provider returned a workspace for an explicit skip decision")
		}
		return &PublicationApplyAction{Kind: PublicationWorkspaceAction, Action: MutationUnchanged, Workspace: &value}, nil
	}
	if current == nil {
		if desired.Choice == PublicationWorkspaceConnect {
			return nil, publicationConflict("selected publication workspace could not be reconciled")
		}
		return &PublicationApplyAction{Kind: PublicationWorkspaceAction, Action: MutationCreate, Workspace: &value}, nil
	}
	if current.Reference == nil || !validPublicationReference(*current.Reference) || current.Reference.Provider != provider {
		return nil, publicationConflict("provider workspace has no valid stable reference")
	}
	if desired.Reference != nil && !sameExternalIdentity(*desired.Reference, *current.Reference) {
		return nil, publicationConflict("selected publication workspace changed identity")
	}
	ref := *current.Reference
	value.Reference = &ref
	return &PublicationApplyAction{Kind: PublicationWorkspaceAction, Action: MutationReuse, Workspace: &value}, nil
}

func validOptionalPublicationReference(ref *ExternalReference) bool {
	return ref == nil || validPublicationReference(*ref)
}

func validPublicationReference(ref ExternalReference) bool {
	return ref.Provider != "" && ref.Provider == strings.TrimSpace(ref.Provider) && ref.Kind != "" && ref.Kind == strings.TrimSpace(ref.Kind) &&
		ref.OpaqueID != "" && ref.OpaqueID == strings.TrimSpace(ref.OpaqueID) && ref.DisplayID != "" && ref.DisplayID == strings.TrimSpace(ref.DisplayID) &&
		ref.URL != "" && ref.URL == strings.TrimSpace(ref.URL)
}

func publicationConflict(message string) error {
	return &IntegrationError{Class: IntegrationAmbiguousIdentity, Operation: "publication.preview", Message: message}
}

func sameExternalIdentity(a, b ExternalReference) bool {
	return a.Provider == b.Provider && a.Kind == b.Kind && a.OpaqueID == b.OpaqueID
}

func artifactLess(a, b planning.ArtifactRef) int {
	if a.Kind != b.Kind {
		return strings.Compare(string(a.Kind), string(b.Kind))
	}
	return strings.Compare(string(a.ID), string(b.ID))
}

func equivalentPublicationArtifact(a, b PublicationArtifact) bool {
	a.Reference = nil
	b.Reference = nil
	a.Dependencies = slices.Clone(a.Dependencies)
	b.Dependencies = slices.Clone(b.Dependencies)
	slices.SortFunc(a.Dependencies, artifactLess)
	slices.SortFunc(b.Dependencies, artifactLess)
	if len(a.Dependencies) == 0 {
		a.Dependencies = nil
	}
	if len(b.Dependencies) == 0 {
		b.Dependencies = nil
	}
	return reflect.DeepEqual(a, b)
}

func orderedPublicationArtifacts(input []PublicationArtifact) ([]PublicationArtifact, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("publication requires artifacts")
	}
	byRef := map[planning.ArtifactRef]PublicationArtifact{}
	keys := make([]planning.ArtifactRef, 0, len(input))
	for _, artifact := range input {
		if artifact.Artifact.Validate() != nil || (artifact.Artifact.Kind != planning.ArtifactSpec && artifact.Artifact.Kind != planning.ArtifactInitiative) || strings.TrimSpace(artifact.Title) == "" {
			return nil, fmt.Errorf("publication requires valid initiative/spec identities and titles")
		}
		if _, duplicate := byRef[artifact.Artifact]; duplicate {
			return nil, publicationConflict("duplicate publication artifact identity")
		}
		artifact.Dependencies = slices.Clone(artifact.Dependencies)
		slices.SortFunc(artifact.Dependencies, artifactLess)
		artifact.Dependencies = slices.Compact(artifact.Dependencies)
		if artifact.Group != nil {
			group := *artifact.Group
			artifact.Group = &group
		}
		if artifact.Reference != nil {
			ref := *artifact.Reference
			artifact.Reference = &ref
		}
		byRef[artifact.Artifact] = artifact
		keys = append(keys, artifact.Artifact)
	}
	slices.SortFunc(keys, artifactLess)
	state := map[planning.ArtifactRef]int{}
	var ordered []PublicationArtifact
	var visit func(planning.ArtifactRef) error
	visit = func(ref planning.ArtifactRef) error {
		if state[ref] == 2 {
			return nil
		}
		if state[ref] == 1 {
			return fmt.Errorf("publication dependency cycle at %s", ref.ID)
		}
		artifact, found := byRef[ref]
		if !found {
			return fmt.Errorf("publication relationship references absent artifact %s", ref.ID)
		}
		state[ref] = 1
		if artifact.Group != nil {
			if artifact.Artifact.Kind != planning.ArtifactSpec || artifact.Group.Kind != planning.ArtifactInitiative {
				return fmt.Errorf("publication groups must connect an initiative to a spec")
			}
			if err := visit(*artifact.Group); err != nil {
				return err
			}
		}
		for _, dependency := range artifact.Dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[ref] = 2
		ordered = append(ordered, artifact)
		return nil
	}
	for _, ref := range keys {
		if err := visit(ref); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}
