package github

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"
)

const (
	executionInspectOperation = "execution.inspect"
	executionAttachOperation  = "execution.attach"
	executionStatusOperation  = "execution.status"
)

var executionStatusNames = map[string]application.ExecutionStatus{
	"Todo":        application.ExecutionTodo,
	"In Progress": application.ExecutionInProgress,
	"In Review":   application.ExecutionInReview,
	"Done":        application.ExecutionDone,
}

type executionOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type executionFieldNode struct {
	Typename string            `json:"__typename"`
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	DataType string            `json:"dataType"`
	Options  []executionOption `json:"options"`
}

type executionItemNode struct {
	ID      string `json:"id"`
	Content *struct {
		Typename   string `json:"__typename"`
		ID         string `json:"id"`
		Number     int    `json:"number"`
		URL        string `json:"url"`
		Repository struct {
			NameWithOwner string `json:"nameWithOwner"`
		} `json:"repository"`
	} `json:"content"`
	Status *struct {
		Typename string `json:"__typename"`
		OptionID string `json:"optionId"`
		Name     string `json:"name"`
		Field    *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"field"`
	} `json:"status"`
}

type executionProject struct {
	publicationProject
	Fields struct {
		Nodes    []executionFieldNode `json:"nodes"`
		PageInfo struct {
			HasNextPage bool `json:"hasNextPage"`
		} `json:"pageInfo"`
	} `json:"fields"`
	Items struct {
		Nodes    []executionItemNode `json:"nodes"`
		PageInfo struct {
			HasNextPage bool `json:"hasNextPage"`
		} `json:"pageInfo"`
	} `json:"items"`
}

type executionInspection struct {
	snapshot  application.ExecutionWorkspaceSnapshot
	project   executionProject
	field     executionFieldNode
	options   map[application.ExecutionStatus]executionOption
	repo      string
	artifacts map[string]planning.ArtifactRef
}

func (a *Adapter) inspectExecutionWorkspace(ctx context.Context, ref application.ExternalReference) (application.ExecutionWorkspaceSnapshot, error) {
	inspection, err := a.executionInspection(ctx, ref)
	return inspection.snapshot, err
}

func (a *Adapter) executionInspection(ctx context.Context, ref application.ExternalReference) (executionInspection, error) {
	var result executionInspection
	if !a.Enabled() {
		return result, a.unavailable(executionInspectOperation)
	}
	if _, _, err := publicationProjectLocation(ref); err != nil {
		return result, providerError(application.IntegrationAmbiguousIdentity, executionInspectOperation, "workspace reference has no canonical Project identity")
	}
	state, err := a.state.read()
	if err != nil {
		return result, providerError(application.IntegrationProviderUnavailable, executionInspectOperation, "cannot read GitHub mapping metadata")
	}
	state = normalizeGitHubState(state)
	if state.Repo == "" {
		return result, providerError(application.IntegrationAmbiguousIdentity, executionInspectOperation, "mapping metadata has no repository identity")
	}
	decision, err := executionProjectDecision(state, ref)
	if err != nil {
		return result, err
	}
	query := `query($id:ID!) { node(id:$id) { ... on ProjectV2 {` + publicationProjectFields + `
    fields(first:100) { nodes { __typename ... on ProjectV2SingleSelectField { id name dataType options { id name } } } pageInfo { hasNextPage } }
    items(first:100) { nodes { id content { __typename ... on Issue { id number url repository { nameWithOwner } } } status: fieldValueByName(name:"Status") { __typename ... on ProjectV2ItemFieldSingleSelectValue { optionId name field { ... on ProjectV2SingleSelectField { id name } } } } } pageInfo { hasNextPage } }
  } } }`
	var response struct {
		Data struct {
			Node *executionProject `json:"node"`
		} `json:"data"`
	}
	if err := a.graphql(ctx, executionInspectOperation, &response, "-f", "query="+query, "-f", "id="+ref.OpaqueID); err != nil {
		return result, err
	}
	if response.Data.Node == nil {
		return result, providerError(application.IntegrationAmbiguousIdentity, executionInspectOperation, "selected Project no longer exists")
	}
	project := *response.Data.Node
	if err := validatePublicationProject(project.publicationProject, executionInspectOperation); err != nil {
		return result, err
	}
	base := publicationProjectReference(project.publicationProject)
	if !sameExecutionIdentity(base, ref) {
		return result, providerError(application.IntegrationAmbiguousIdentity, executionInspectOperation, "selected Project identity disagrees with provider")
	}
	if project.Fields.PageInfo.HasNextPage || len(project.Fields.Nodes) >= 100 || project.Items.PageInfo.HasNextPage || len(project.Items.Nodes) >= 100 {
		return result, providerError(application.IntegrationProviderUnavailable, executionInspectOperation, "Project inspection reached safety limit")
	}
	var statusField *executionFieldNode
	for i := range project.Fields.Nodes {
		field := &project.Fields.Nodes[i]
		if field.Typename == "ProjectV2SingleSelectField" && strings.EqualFold(strings.TrimSpace(field.Name), "Status") {
			if statusField != nil {
				return result, providerError(application.IntegrationManualRemediationRequired, executionInspectOperation, "Project has duplicate Status fields")
			}
			statusField = field
		}
	}
	if statusField == nil || statusField.ID == "" {
		return result, providerError(application.IntegrationManualRemediationRequired, executionInspectOperation, "Project has no supported Status field")
	}
	if expected := decision.FieldIDs["Status"]; expected != "" && expected != statusField.ID {
		return result, providerError(application.IntegrationManualRemediationRequired, executionInspectOperation, "stored Status field identity is stale")
	}
	options := map[application.ExecutionStatus]executionOption{}
	for _, option := range statusField.Options {
		status, known := executionStatusNames[option.Name]
		if !known {
			continue
		}
		if option.ID == "" || options[status].ID != "" {
			return result, providerError(application.IntegrationManualRemediationRequired, executionInspectOperation, "Project has ambiguous semantic Status options")
		}
		options[status] = option
	}
	statuses := make([]application.ExecutionStatus, 0, len(options))
	for _, status := range []application.ExecutionStatus{application.ExecutionTodo, application.ExecutionInProgress, application.ExecutionInReview, application.ExecutionDone} {
		if options[status].ID != "" {
			statuses = append(statuses, status)
		}
	}
	configurationRaw, _ := json.Marshal(struct {
		FieldID string
		Options map[application.ExecutionStatus]executionOption
	}{statusField.ID, options})
	snapshot := application.ExecutionWorkspaceSnapshot{SchemaVersion: application.IntegrationContractVersion, ConfigurationRevision: fmt.Sprintf("%x", sha256.Sum256(configurationRaw)), Fields: []application.ExecutionField{{Role: application.ExecutionFieldRoleStatus, SupportedStatuses: statuses}}}
	mappings, err := executionArtifactMappings(state)
	if err != nil {
		return result, err
	}
	seenArtifacts := map[planning.ArtifactRef]bool{}
	for _, item := range project.Items.Nodes {
		if item.Content == nil || item.Content.Typename != "Issue" {
			continue
		}
		artifact, tracked := mappings[item.Content.URL]
		if !tracked {
			continue
		}
		if item.ID == "" || item.Content.ID == "" || item.Content.Number < 1 || !strings.EqualFold(item.Content.Repository.NameWithOwner, state.Repo) || item.Content.URL != fmt.Sprintf("https://github.com/%s/issues/%d", state.Repo, item.Content.Number) || seenArtifacts[artifact] {
			return result, providerError(application.IntegrationManualRemediationRequired, executionInspectOperation, "Project has invalid or duplicate tracked items")
		}
		seenArtifacts[artifact] = true
		status := application.ExecutionStatus("")
		optionID, optionName := "", ""
		if item.Status != nil {
			optionID, optionName = item.Status.OptionID, item.Status.Name
			semantic, known := executionStatusNames[item.Status.Name]
			if item.Status.Field == nil || item.Status.Field.ID != statusField.ID || !strings.EqualFold(item.Status.Field.Name, "Status") || !known || options[semantic].ID != item.Status.OptionID {
				return result, providerError(application.IntegrationManualRemediationRequired, executionInspectOperation, "tracked Project item has an unsupported Status value")
			}
			status = semantic
		}
		published := application.ExternalReference{Provider: providerName, Kind: "issue", OpaqueID: item.Content.ID, DisplayID: strconv.Itoa(item.Content.Number), URL: item.Content.URL}
		raw, _ := json.Marshal([]string{item.ID, item.Content.ID, statusField.ID, optionID, optionName})
		itemRef := application.ExternalReference{Provider: providerName, Kind: "project_item", OpaqueID: item.ID, DisplayID: "#" + strconv.Itoa(item.Content.Number), URL: item.Content.URL, Revision: fmt.Sprintf("%x", sha256.Sum256(raw))}
		snapshot.Items = append(snapshot.Items, application.ExecutionWorkItem{Artifact: artifact, PublishedWork: published, Reference: itemRef, Status: status})
	}
	slices.SortFunc(snapshot.Items, func(a, b application.ExecutionWorkItem) int {
		return strings.Compare(string(a.Artifact.ID), string(b.Artifact.ID))
	})
	revisionRaw, _ := json.Marshal(struct {
		Base    string
		FieldID string
		Options map[application.ExecutionStatus]executionOption
		Items   []application.ExecutionWorkItem
	}{base.Revision, statusField.ID, options, snapshot.Items})
	base.Revision = fmt.Sprintf("%x", sha256.Sum256(revisionRaw))
	snapshot.Workspace = base
	result = executionInspection{snapshot: snapshot, project: project, field: *statusField, options: options, repo: state.Repo, artifacts: mappings}
	return result, nil
}

func executionProjectDecision(state githubState, ref application.ExternalReference) (githubProjectDecision, error) {
	var found *githubProjectDecision
	for _, decision := range state.ProjectDecisions {
		if decision.ProjectID != ref.OpaqueID {
			continue
		}
		if decision.ProjectNumber != mustProjectNumber(ref.DisplayID) || decision.ProjectURL != ref.URL {
			return githubProjectDecision{}, providerError(application.IntegrationManualRemediationRequired, executionInspectOperation, "workspace mapping is missing or ambiguous")
		}
		if found == nil {
			copy := decision
			found = &copy
			continue
		}
		currentStatus, nextStatus := found.FieldIDs["Status"], decision.FieldIDs["Status"]
		if currentStatus != "" && nextStatus != "" && currentStatus != nextStatus {
			return githubProjectDecision{}, providerError(application.IntegrationManualRemediationRequired, executionInspectOperation, "workspace decisions disagree about the Status field")
		}
		if currentStatus == "" && nextStatus != "" {
			if found.FieldIDs == nil {
				found.FieldIDs = map[string]string{}
			}
			found.FieldIDs["Status"] = nextStatus
		}
	}
	if found == nil {
		return githubProjectDecision{}, providerError(application.IntegrationManualRemediationRequired, executionInspectOperation, "workspace has no durable Project mapping")
	}
	return *found, nil
}

func mustProjectNumber(value string) int { number, _ := strconv.Atoi(value); return number }

func executionArtifactMappings(state githubState) (map[string]planning.ArtifactRef, error) {
	result := map[string]planning.ArtifactRef{}
	for key, record := range state.Planning {
		ref := planning.ArtifactRef{Kind: planning.ArtifactKind(record.Kind), ID: planning.ArtifactID(record.Slug)}
		if record.IssueURL == "" {
			continue
		}
		wantURL := fmt.Sprintf("https://github.com/%s/issues/%d", state.Repo, record.IssueNumber)
		if record.Slug != key || ref.Validate() != nil || record.IssueNumber < 1 || record.IssueURL != wantURL {
			return nil, providerError(application.IntegrationManualRemediationRequired, executionInspectOperation, "artifact mapping is invalid")
		}
		if existing, duplicate := result[record.IssueURL]; duplicate && existing != ref {
			return nil, providerError(application.IntegrationManualRemediationRequired, executionInspectOperation, "issue maps to multiple Planning artifacts")
		}
		result[record.IssueURL] = ref
	}
	return result, nil
}

func sameExecutionIdentity(a, b application.ExternalReference) bool {
	return a.Provider == b.Provider && a.Kind == b.Kind && a.OpaqueID == b.OpaqueID && a.DisplayID == b.DisplayID && a.URL == b.URL
}

func (a *Adapter) attachExecutionWork(ctx context.Context, request application.ExecutionAttachRequest) (application.ExecutionWorkItem, error) {
	var empty application.ExecutionWorkItem
	if !a.Enabled() {
		return empty, a.unavailable(executionAttachOperation)
	}
	inspection, err := a.executionInspection(ctx, request.Workspace)
	if err != nil {
		return empty, err
	}
	if inspection.snapshot.Workspace != request.Workspace {
		return empty, providerError(application.IntegrationRevisionConflict, executionAttachOperation, "Project changed since preview")
	}
	resolvedIssue, err := a.resolveExecutionIssue(ctx, inspection.repo, request.PublishedWork)
	if err != nil {
		return empty, err
	}
	if mapped, ok := inspection.artifacts[resolvedIssue.URL]; !ok || mapped != request.Artifact {
		return empty, providerError(application.IntegrationManualRemediationRequired, executionAttachOperation, "published issue is not durably mapped to the requested artifact")
	}
	for _, item := range inspection.snapshot.Items {
		if sameExecutionIdentity(item.PublishedWork, resolvedIssue) {
			if item.Artifact != request.Artifact {
				return empty, providerError(application.IntegrationManualRemediationRequired, executionAttachOperation, "published work maps to a different artifact")
			}
			return item, nil
		}
	}
	if request.Artifact.Validate() != nil {
		return empty, providerError(application.IntegrationAmbiguousIdentity, executionAttachOperation, "attachment request has invalid identity")
	}
	payload := map[string]any{"query": `mutation($projectId:ID!, $contentId:ID!) { addProjectV2ItemById(input:{projectId:$projectId, contentId:$contentId}) { item { id } } }`, "variables": map[string]string{"projectId": inspection.project.ID, "contentId": resolvedIssue.OpaqueID}}
	raw, err := a.executionRequest(ctx, executionAttachOperation, payload)
	if err != nil {
		return empty, err
	}
	var response struct {
		Data struct {
			Add struct {
				Item struct {
					ID string `json:"id"`
				} `json:"item"`
			} `json:"addProjectV2ItemById"`
		} `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	if json.Unmarshal(raw, &response) != nil || len(response.Errors) != 0 || response.Data.Add.Item.ID == "" {
		return empty, providerError(application.IntegrationPartialFailure, executionAttachOperation, "Project attachment returned incomplete evidence; inspect before retrying")
	}
	fresh, err := a.executionInspection(ctx, request.Workspace)
	if err != nil {
		return empty, executionPartial(executionAttachOperation, err)
	}
	for _, item := range fresh.snapshot.Items {
		if sameExecutionIdentity(item.PublishedWork, resolvedIssue) && item.Artifact == request.Artifact && item.Reference.OpaqueID == response.Data.Add.Item.ID {
			return item, nil
		}
	}
	return empty, providerError(application.IntegrationPartialFailure, executionAttachOperation, "Project attachment could not be verified; inspect before retrying")
}

func (a *Adapter) resolveExecutionIssue(ctx context.Context, repo string, ref application.ExternalReference) (application.ExternalReference, error) {
	number, err := strconv.Atoi(ref.DisplayID)
	wantURL := fmt.Sprintf("https://github.com/%s/issues/%d", repo, number)
	if err != nil || number < 1 || ref.Provider != providerName || ref.Kind != "issue" || ref.URL != wantURL || ref.OpaqueID == "" {
		return application.ExternalReference{}, providerError(application.IntegrationAmbiguousIdentity, executionAttachOperation, "published issue reference is invalid")
	}
	raw, err := a.runProvider(ctx, executionAttachOperation, "api", "--method", "GET", fmt.Sprintf("repos/%s/issues/%d", repo, number))
	if err != nil {
		return application.ExternalReference{}, err
	}
	var issue publicationIssue
	if json.Unmarshal(raw, &issue) != nil || validatePublicationIssue(&issue, repo) != nil || issue.Number != number || issue.URL != wantURL || ref.OpaqueID != wantURL && ref.OpaqueID != issue.ID {
		return application.ExternalReference{}, providerError(application.IntegrationAmbiguousIdentity, executionAttachOperation, "published issue identity disagrees with provider")
	}
	return application.ExternalReference{Provider: providerName, Kind: "issue", OpaqueID: issue.ID, DisplayID: strconv.Itoa(number), URL: issue.URL}, nil
}

func (a *Adapter) applyExecutionStatus(ctx context.Context, request application.ExecutionStatusRequest) (application.ExecutionWorkItem, error) {
	var empty application.ExecutionWorkItem
	if !a.Enabled() {
		return empty, a.unavailable(executionStatusOperation)
	}
	inspection, err := a.executionInspection(ctx, request.Workspace)
	if err != nil {
		return empty, err
	}
	if inspection.snapshot.Workspace != request.Workspace {
		return empty, providerError(application.IntegrationRevisionConflict, executionStatusOperation, "Project changed since preview")
	}
	if request.ConfigurationRevision == "" || inspection.snapshot.ConfigurationRevision != request.ConfigurationRevision {
		return empty, providerError(application.IntegrationRevisionConflict, executionStatusOperation, "Project Status configuration changed since preview")
	}
	var current *application.ExecutionWorkItem
	for i := range inspection.snapshot.Items {
		if sameExecutionIdentity(inspection.snapshot.Items[i].Reference, request.WorkItem) {
			current = &inspection.snapshot.Items[i]
		}
	}
	if current == nil || current.Reference != request.WorkItem {
		return empty, providerError(application.IntegrationRevisionConflict, executionStatusOperation, "Project item changed since preview")
	}
	option := inspection.options[request.Status]
	if option.ID == "" {
		return empty, providerError(application.IntegrationManualRemediationRequired, executionStatusOperation, "requested semantic Status option is unavailable")
	}
	if current.Status == request.Status {
		return *current, nil
	}
	payload := map[string]any{"query": `mutation($projectId:ID!, $itemId:ID!, $fieldId:ID!, $optionId:String!) { updateProjectV2ItemFieldValue(input:{projectId:$projectId, itemId:$itemId, fieldId:$fieldId, value:{singleSelectOptionId:$optionId}}) { projectV2Item { id } } }`, "variables": map[string]string{"projectId": inspection.project.ID, "itemId": current.Reference.OpaqueID, "fieldId": inspection.field.ID, "optionId": option.ID}}
	raw, err := a.executionRequest(ctx, executionStatusOperation, payload)
	if err != nil {
		return empty, err
	}
	var response struct {
		Data struct {
			Update struct {
				Item struct {
					ID string `json:"id"`
				} `json:"projectV2Item"`
			} `json:"updateProjectV2ItemFieldValue"`
		} `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	if json.Unmarshal(raw, &response) != nil || len(response.Errors) != 0 || response.Data.Update.Item.ID != current.Reference.OpaqueID {
		return empty, providerError(application.IntegrationPartialFailure, executionStatusOperation, "Project status mutation returned incomplete evidence; inspect before retrying")
	}
	fresh, err := a.executionInspection(ctx, request.Workspace)
	if err != nil {
		return empty, executionPartial(executionStatusOperation, err)
	}
	for _, item := range fresh.snapshot.Items {
		if item.Reference.OpaqueID == current.Reference.OpaqueID && item.Status == request.Status {
			return item, nil
		}
	}
	return empty, providerError(application.IntegrationPartialFailure, executionStatusOperation, "Project status mutation could not be verified; inspect before retrying")
}

func (a *Adapter) executionRequest(ctx context.Context, operation string, payload any) ([]byte, error) {
	runner, ok := a.runner.(InputRunner)
	if !ok {
		return nil, providerError(application.IntegrationUnsupportedCapability, operation, "Project mutation requires a runner with stdin support")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	output, runErr := runner.RunInput(ctx, a.projectRoot, raw, "api", "--method", "POST", "graphql", "--input", "-")
	data, err := providerOutput(ctx, operation, output, runErr)
	if err == nil {
		return data, nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}
	var integration *application.IntegrationError
	if errors.As(err, &integration) && (integration.Class == application.IntegrationUnauthenticated || integration.Class == application.IntegrationUnauthorized) {
		return nil, err
	}
	return nil, executionPartial(operation, err)
}

func executionPartial(operation string, err error) error {
	return &application.IntegrationError{Class: application.IntegrationPartialFailure, Provider: providerName, Operation: operation, Message: "Project mutation outcome may be incomplete; inspect before retrying", Err: err}
}
