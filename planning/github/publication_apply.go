package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"
)

const publicationApplyOperation = "publication.apply"

// Provider ports do not grant host permissions. The shared planner is reused
// here solely to validate action classification against a fresh provider read;
// the invoking application service owns confirmation, authorization, and audit.
type publicationInspectionPolicy struct{}

func (publicationInspectionPolicy) Require(context.Context, string) error { return nil }

func publicationApplyConflict(message string) error {
	return providerError(application.IntegrationRevisionConflict, publicationApplyOperation, message)
}

func (a *Adapter) applyPublication(ctx context.Context, plan application.PublicationPlan) (application.PublicationResult, error) {
	result := application.PublicationResult{SchemaVersion: application.IntegrationContractVersion}
	if !a.Enabled() {
		return result, a.unavailable(publicationApplyOperation)
	}
	input, err := publicationPlanIntent(plan)
	if err != nil {
		return result, err
	}
	fresh, err := application.New(nil, application.Options{}).PreviewPublication(ctx, a.PublicationTarget(), input, publicationInspectionPolicy{})
	if err != nil {
		return result, err
	}
	want, _ := json.Marshal(plan)
	got, _ := json.Marshal(fresh)
	if !bytes.Equal(want, got) {
		return result, publicationApplyConflict("publication plan changed before apply")
	}
	// A source-discovered issue must stay discoverable until mappings are saved.
	// Existing durable mappings may retain a renamed or source-less issue.
	state, stateErr := a.state.read()
	if stateErr != nil && !errors.Is(stateErr, os.ErrNotExist) {
		return result, providerError(application.IntegrationProviderUnavailable, publicationApplyOperation, "cannot verify publication mappings")
	}
	if stateErr == nil && ((state.Repo != "" && state.Repo != plan.Target.OpaqueID) || (state.Repo == "" && len(state.Planning) > 0)) {
		return result, publicationApplyConflict("metadata belongs to a different repository")
	}
	for _, action := range plan.Actions {
		if action.Action != application.MutationUpdate || action.Artifact == nil {
			continue
		}
		artifact := action.Artifact
		record, mapped := state.Planning[string(artifact.Artifact.ID)]
		mapped = mapped && record.Kind == string(artifact.Artifact.Kind) && strconv.Itoa(record.IssueNumber) == artifact.Reference.DisplayID
		if !mapped && (plan.Source == nil || !publicationSourceLink(artifact.Content, plan.Source.URL) || publicationSlug(artifact.Title) != string(artifact.Artifact.ID)) {
			return result, publicationApplyConflict("update would lose recoverable identity before mappings are saved")
		}
	}
	write := false
	for _, action := range plan.Actions {
		write = write || action.Action == application.MutationCreate || action.Action == application.MutationUpdate
	}
	labels := map[string]bool{}
	if write {
		if _, ok := a.runner.(InputRunner); !ok {
			return result, providerError(application.IntegrationUnsupportedCapability, publicationApplyOperation, "publication requires a runner with stdin support")
		}
		labels, err = a.publicationLabels(ctx, plan.Target.OpaqueID)
		if err != nil {
			return result, err
		}
	}
	resolved := map[planning.ArtifactRef]application.ExternalReference{}
	for _, action := range plan.Actions {
		evidence := application.PublicationActionEvidence{Action: action}
		fail := func(err error) (application.PublicationResult, error) {
			result.Failed = &evidence
			changed := len(evidence.References) > 0
			for _, completed := range result.Completed {
				changed = changed || completed.Action.Action == application.MutationCreate || completed.Action.Action == application.MutationUpdate
			}
			if changed {
				err = &application.IntegrationError{Class: application.IntegrationPartialFailure, Provider: providerName, Operation: publicationApplyOperation, Message: "publication stopped after provider changes; inspect before retrying", Err: err}
			}
			return result, err
		}
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		if artifact := action.Artifact; artifact != nil {
			if action.Action == application.MutationReuse || action.Action == application.MutationUnchanged {
				evidence.References = []application.ExternalReference{*artifact.Reference}
				resolved[artifact.Artifact] = *artifact.Reference
			} else {
				for _, label := range publicationArtifactLabels(*artifact) {
					if labels[label] {
						continue
					}
					payload := map[string]string{"name": label, "color": publicationLabelColor(label)}
					raw, err := a.publicationRequest(ctx, "POST", "repos/"+plan.Target.OpaqueID+"/labels", payload)
					if err != nil {
						return fail(err)
					}
					var created struct{ Name string }
					if json.Unmarshal(raw, &created) != nil || created.Name != label {
						return fail(providerError(application.IntegrationPartialFailure, publicationApplyOperation, "label mutation returned incomplete evidence; inspect before retrying"))
					}
					labels[label] = true
					evidence.References = append(evidence.References, application.ExternalReference{Provider: providerName, Kind: "label", OpaqueID: plan.Target.OpaqueID + "/" + label, DisplayID: label})
				}
				issue, err := a.writePublicationIssue(ctx, plan, input, action)
				if issue != nil {
					ref := publicationIssueReference(*issue)
					evidence.References = append(evidence.References, ref)
					resolved[artifact.Artifact] = ref
				}
				if err != nil {
					return fail(err)
				}
			}
		} else if action.Action == application.MutationCreate {
			if err := a.writePublicationRelationship(ctx, *action.Relationship, resolved); err != nil {
				return fail(err)
			}
		}
		result.Completed = append(result.Completed, evidence)
	}
	return result, nil
}

func publicationPlanIntent(plan application.PublicationPlan) (application.PublicationPreviewInput, error) {
	input := application.PublicationPreviewInput{Target: plan.Target, Source: plan.Source}
	if plan.SchemaVersion != application.IntegrationContractVersion || len(plan.Actions) == 0 {
		return input, publicationApplyConflict("a versioned publication plan is required")
	}
	for _, action := range plan.Actions {
		if action.Kind == application.PublicationGroupAction || action.Kind == application.PublicationWorkspaceAction {
			return input, providerError(application.IntegrationUnsupportedCapability, publicationApplyOperation, "publication group and workspace actions are not implemented by the GitHub adapter")
		}
		if action.Kind == application.PublicationRelationshipAction && action.Relationship != nil && action.Artifact == nil {
			continue
		}
		if action.Kind != application.PublicationArtifactAction || action.Artifact == nil || action.Relationship != nil {
			return input, publicationApplyConflict("invalid publication action")
		}
		artifact := *action.Artifact
		if artifact.Reference != nil {
			input.Candidates = append(input.Candidates, application.ArtifactExternalReference{Artifact: artifact.Artifact, Reference: *artifact.Reference})
		}
		if action.Action == application.MutationCreate || action.Action == application.MutationUpdate {
			switch artifact.Readiness {
			case planning.ReadinessReady, planning.ReadinessBlocked, planning.ReadinessNeedsRefinement, planning.ReadinessDone:
			default:
				return input, providerError(application.IntegrationUnsupportedCapability, publicationApplyOperation, "publication readiness has no retained provider representation")
			}
		}
		if action.Action == application.MutationCreate {
			// Without persisted mappings these are the exact recovery keys used by
			// Inspect. Refuse an unrecoverable create rather than inventing content.
			if artifact.Reference != nil || plan.Source == nil || plan.Source.URL == "" || !publicationSourceLink(artifact.Content, plan.Source.URL) || publicationSlug(artifact.Title) != string(artifact.Artifact.ID) {
				return input, publicationApplyConflict("new issues require a canonical source link and recoverable title slug")
			}
			if artifact.Readiness == planning.ReadinessDone {
				return input, providerError(application.IntegrationUnsupportedCapability, publicationApplyOperation, "creating completed issues requires a separate status transition")
			}
		} else {
			if artifact.Reference == nil || artifact.Reference.Revision == "" {
				return input, publicationApplyConflict("existing issues require revision-bound references")
			}
		}
		if action.Action == application.MutationReuse {
			artifact.Reference = nil
		}
		input.Artifacts = append(input.Artifacts, artifact)
	}
	return input, nil
}

func publicationArtifactLabels(artifact application.PublicationArtifact) []string {
	labels := []string{"plan:" + string(artifact.Artifact.Kind)}
	if artifact.Readiness == planning.ReadinessReady {
		labels = append(labels, "plan:ready")
	}
	if artifact.Readiness == planning.ReadinessBlocked {
		labels = append(labels, "plan:blocked")
	}
	return labels
}

func publicationLabelColor(label string) string {
	switch label {
	case "plan:initiative":
		return "5319e7"
	case "plan:spec":
		return "1d76db"
	case "plan:ready":
		return "0e8a16"
	case "plan:blocked":
		return "d93f0b"
	default:
		return "ededed"
	}
}

func (a *Adapter) publicationLabels(ctx context.Context, repo string) (map[string]bool, error) {
	raw, err := a.runProvider(ctx, publicationApplyOperation, "api", "--method", "GET", "repos/"+repo+"/labels?per_page=100")
	if err != nil {
		return nil, err
	}
	var labels []struct{ Name string }
	if json.Unmarshal(raw, &labels) != nil || strings.TrimSpace(string(raw)) == "null" || len(labels) >= 100 {
		return nil, providerError(application.IntegrationProviderUnavailable, publicationApplyOperation, "invalid or incomplete label listing")
	}
	known := map[string]bool{}
	for _, label := range labels {
		known[label.Name] = true
	}
	return known, nil
}

func (a *Adapter) publicationRequest(ctx context.Context, method, endpoint string, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	runner := a.runner.(InputRunner) // checked before any writes
	output, err := runner.RunInput(ctx, a.projectRoot, raw, "api", "--method", method, endpoint, "--input", "-")
	data, err := providerOutput(ctx, publicationApplyOperation, output, err)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		var integration *application.IntegrationError
		if errors.As(err, &integration) && (integration.Class == application.IntegrationUnauthenticated || integration.Class == application.IntegrationUnauthorized) {
			return nil, err
		}
		return nil, &application.IntegrationError{Class: application.IntegrationPartialFailure, Provider: providerName, Operation: publicationApplyOperation, Message: "provider mutation outcome may be incomplete; inspect before retrying", Err: err}
	}
	return data, nil
}

func (a *Adapter) writePublicationIssue(ctx context.Context, plan application.PublicationPlan, input application.PublicationPreviewInput, action application.PublicationApplyAction) (*publicationIssue, error) {
	artifact := *action.Artifact
	repo := plan.Target.OpaqueID
	method, endpoint := "POST", "repos/"+repo+"/issues"
	labels := publicationArtifactLabels(artifact)
	if action.Action == application.MutationUpdate {
		current, err := a.publicationGetIssue(ctx, repo, *artifact.Reference)
		if err != nil {
			return nil, err
		}
		if publicationIssueReference(current).Revision != artifact.Reference.Revision {
			return nil, publicationApplyConflict("issue changed before update")
		}
		for _, label := range current.Labels {
			if label.Name != "plan:initiative" && label.Name != "plan:spec" && label.Name != "plan:ready" && label.Name != "plan:blocked" {
				labels = append(labels, label.Name)
			}
		}
		method, endpoint = "PATCH", endpoint+"/"+strconv.Itoa(current.Number)
	} else {
		// Recheck missing identities immediately before creation, including after
		// earlier actions. A final cross-client read/write race remains possible.
		refs := make([]planning.ArtifactRef, len(input.Artifacts))
		for i, item := range input.Artifacts {
			refs[i] = item.Artifact
		}
		snapshot, err := a.inspectPublication(ctx, application.PublicationInspectRequest{Target: plan.Target, Source: plan.Source, Artifacts: refs})
		if err != nil {
			return nil, err
		}
		for _, existing := range snapshot.Artifacts {
			if existing.Artifact == artifact.Artifact {
				return nil, publicationApplyConflict("issue appeared before create")
			}
		}
	}
	slices.Sort(labels)
	labels = slices.Compact(labels)
	payload := map[string]any{"title": artifact.Title, "body": artifact.Content, "labels": labels}
	if method == "PATCH" {
		payload["state"] = "open"
		if artifact.Readiness == planning.ReadinessDone {
			payload["state"] = "closed"
		}
	}
	raw, err := a.publicationRequest(ctx, method, endpoint, payload)
	if err != nil {
		return nil, err
	}
	var issue publicationIssue
	if json.Unmarshal(raw, &issue) != nil || validatePublicationIssue(&issue, repo) != nil {
		return nil, providerError(application.IntegrationPartialFailure, publicationApplyOperation, "issue mutation returned invalid identity; inspect before retrying")
	}
	if artifact.Reference != nil && (issue.ID != artifact.Reference.OpaqueID || strconv.Itoa(issue.Number) != artifact.Reference.DisplayID) {
		return nil, providerError(application.IntegrationPartialFailure, publicationApplyOperation, "issue mutation returned a different identity; inspect before retrying")
	}
	if issue.Title != artifact.Title || issue.Body != artifact.Content {
		return &issue, providerError(application.IntegrationPartialFailure, publicationApplyOperation, "issue mutation returned different content; inspect before retrying")
	}
	for _, label := range labels {
		if !publicationHasLabel(issue, label) {
			return &issue, providerError(application.IntegrationPartialFailure, publicationApplyOperation, "issue mutation returned incomplete labels; inspect before retrying")
		}
	}
	for _, label := range []string{"plan:initiative", "plan:spec", "plan:ready", "plan:blocked"} {
		if publicationHasLabel(issue, label) != slices.Contains(labels, label) {
			return &issue, providerError(application.IntegrationPartialFailure, publicationApplyOperation, "issue mutation returned conflicting managed labels; inspect before retrying")
		}
	}
	if (artifact.Readiness == planning.ReadinessDone) != strings.EqualFold(issue.State, "closed") {
		return &issue, providerError(application.IntegrationPartialFailure, publicationApplyOperation, "issue mutation returned different state; inspect before retrying")
	}
	return &issue, nil
}

func (a *Adapter) publicationGetIssue(ctx context.Context, repo string, ref application.ExternalReference) (publicationIssue, error) {
	var issue publicationIssue
	number, err := strconv.Atoi(ref.DisplayID)
	if err != nil || number < 1 || ref.Provider != providerName || ref.Kind != "issue" || ref.URL != fmt.Sprintf("https://github.com/%s/issues/%d", repo, number) {
		return issue, publicationApplyConflict("invalid issue reference")
	}
	raw, err := a.runProvider(ctx, publicationApplyOperation, "api", "--method", "GET", fmt.Sprintf("repos/%s/issues/%d", repo, number))
	if err != nil {
		return issue, err
	}
	if json.Unmarshal(raw, &issue) != nil || validatePublicationIssue(&issue, repo) != nil || issue.ID != ref.OpaqueID || issue.Number != number {
		return issue, publicationApplyConflict("issue identity changed")
	}
	return issue, nil
}

func (a *Adapter) writePublicationRelationship(ctx context.Context, relationship application.PublicationRelationship, refs map[planning.ArtifactRef]application.ExternalReference) error {
	source, target := refs[relationship.Source], refs[relationship.Target]
	mutation, field := "addSubIssue", "subIssue"
	if relationship.Kind == application.PublicationRelationshipDependsOn {
		mutation, field = "addBlockedBy", "blockingIssue"
	}
	query := fmt.Sprintf(`mutation($source:ID!,$target:ID!){%s(input:{issueId:$source,%sId:$target}){issue{id number} %s{id number}}}`, mutation, field, field)
	raw, err := a.publicationRequest(ctx, "POST", "graphql", map[string]any{"query": query, "variables": map[string]string{"source": source.OpaqueID, "target": target.OpaqueID}})
	if err != nil {
		return err
	}
	var response struct {
		Data map[string]map[string]struct {
			ID     string
			Number int
		}
		Errors []json.RawMessage
	}
	if json.Unmarshal(raw, &response) != nil || len(response.Errors) != 0 || response.Data[mutation]["issue"].ID != source.OpaqueID || response.Data[mutation][field].ID != target.OpaqueID {
		return providerError(application.IntegrationPartialFailure, publicationApplyOperation, "relationship mutation returned incomplete evidence; inspect before retrying")
	}
	return nil
}
