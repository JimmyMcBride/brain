package application

import (
	"context"
	"errors"
	"fmt"
)

const (
	// EventExecutionAttached records a newly tracked Planning artifact.
	EventExecutionAttached = "planning.execution.attached"
	// EventExecutionStatusUpdated records a changed semantic delivery status.
	EventExecutionStatusUpdated = "planning.execution.status-updated"
)

// PreviewExecutionAttach classifies one published artifact as an existing or
// missing execution-workspace item without mutation.
func (s *Service) PreviewExecutionAttach(ctx context.Context, workspace ExecutionWorkspace, request ExecutionAttachRequest, authorizer Authorizer) (ExecutionAttachPlan, error) {
	var empty ExecutionAttachPlan
	if err := authorizeExecutionRead(ctx, workspace, authorizer); err != nil {
		return empty, err
	}
	if err := validateExecutionAttachRequest(request); err != nil {
		return empty, err
	}
	snapshot, err := workspace.Inspect(ctx, request.Workspace)
	if err != nil {
		return empty, err
	}
	if err := validateExecutionSnapshot(snapshot, request.Workspace); err != nil {
		return empty, err
	}
	plan := ExecutionAttachPlan{SchemaVersion: IntegrationContractVersion, Action: MutationCreate, Request: request}
	plan.Request.Workspace = snapshot.Workspace
	for i := range snapshot.Items {
		item := snapshot.Items[i]
		if !sameExternalLocation(item.PublishedWork, request.PublishedWork) {
			continue
		}
		if plan.Item != nil || item.Artifact != request.Artifact {
			return empty, executionManual("execution.attach", "published work has ambiguous Project membership")
		}
		copy := item
		plan.Item = &copy
		plan.Action = MutationUnchanged
		plan.Request.PublishedWork = item.PublishedWork
	}
	return plan, nil
}

// ApplyExecutionAttach requires a confirmed fresh preview, execution
// permission, and audit persistence before adding one Project item.
func (s *Service) ApplyExecutionAttach(ctx context.Context, workspace ExecutionWorkspace, input ExecutionAttachApplyInput, authorizer Authorizer, events EventSink) (ExecutionMutationResult, error) {
	result := ExecutionMutationResult{SchemaVersion: IntegrationContractVersion}
	if !input.Confirmed {
		return result, ErrConfirmationRequired
	}
	if err := authorizeExecutionMutation(ctx, workspace, authorizer, events); err != nil {
		return result, err
	}
	if input.ExpectedPlan.SchemaVersion != IntegrationContractVersion {
		return result, executionConflict("execution.attach", "attachment preview is missing or changed")
	}
	fresh, err := s.PreviewExecutionAttach(ctx, workspace, input.ExpectedPlan.Request, authorizer)
	if err != nil {
		return result, err
	}
	if !samePublicationJSON(fresh, input.ExpectedPlan) {
		return result, executionConflict("execution.attach", "attachment preview changed before apply")
	}
	result.Action = fresh.Action
	if fresh.Action == MutationUnchanged {
		result.Item = *fresh.Item
		return result, nil
	}
	item, err := workspace.Attach(ctx, fresh.Request)
	if err != nil {
		if validPartialAttachedExecutionItem(item, fresh.Request) {
			result.Item = item
			event := Event{Name: EventExecutionAttached, ModuleID: s.moduleID, Artifact: item.Artifact, Source: &item.Reference, Outcome: MutationCreate, OccurredAt: s.now().UTC()}
			result.Event = &event
			if auditErr := events.Publish(context.WithoutCancel(ctx), event); auditErr != nil {
				err = errors.Join(err, &IntegrationError{Class: IntegrationPartialFailure, Operation: "execution.attach.audit", Message: "Project item changed but audit persistence failed", Err: auditErr})
			}
		}
		return result, err
	}
	if err := validateAttachedExecutionItem(item, fresh.Request); err != nil {
		return result, &IntegrationError{Class: IntegrationPartialFailure, Operation: "execution.attach", Message: "Project attachment returned invalid evidence; inspect before retrying", Err: err}
	}
	result.Item = item
	event := Event{Name: EventExecutionAttached, ModuleID: s.moduleID, Artifact: item.Artifact, Source: &item.Reference, Outcome: MutationCreate, OccurredAt: s.now().UTC()}
	result.Event = &event
	if err := events.Publish(context.WithoutCancel(ctx), event); err != nil {
		return result, &IntegrationError{Class: IntegrationPartialFailure, Operation: "execution.attach.audit", Message: "Project item attached but audit persistence failed", Err: err}
	}
	return result, nil
}

// PreviewExecutionStatus classifies a semantic status transition without
// exposing provider field or option identities.
func (s *Service) PreviewExecutionStatus(ctx context.Context, workspace ExecutionWorkspace, request ExecutionStatusRequest, authorizer Authorizer) (ExecutionStatusPlan, error) {
	var empty ExecutionStatusPlan
	if err := authorizeExecutionRead(ctx, workspace, authorizer); err != nil {
		return empty, err
	}
	if err := validateExecutionStatus(request.Status); err != nil || !validExecutionReference(request.WorkItem, "project_item") {
		return empty, executionManual("execution.status", "status request has an invalid semantic status or item identity")
	}
	snapshot, err := workspace.Inspect(ctx, request.Workspace)
	if err != nil {
		return empty, err
	}
	if err := validateExecutionSnapshot(snapshot, request.Workspace); err != nil {
		return empty, err
	}
	if !executionStatusSupported(snapshot.Fields, request.Status) {
		return empty, executionManual("execution.status", "Project status option is unavailable or ambiguous")
	}
	var found *ExecutionWorkItem
	for i := range snapshot.Items {
		item := snapshot.Items[i]
		if !sameExternalLocation(item.Reference, request.WorkItem) {
			continue
		}
		if found != nil {
			return empty, executionManual("execution.status", "Project item identity is ambiguous")
		}
		copy := item
		found = &copy
	}
	if found == nil {
		return empty, executionManual("execution.status", "published work is not attached to the Project")
	}
	action := MutationUpdate
	if found.Status == request.Status {
		action = MutationUnchanged
	}
	request.Workspace = snapshot.Workspace
	request.WorkItem = found.Reference
	request.ConfigurationRevision = snapshot.ConfigurationRevision
	return ExecutionStatusPlan{SchemaVersion: IntegrationContractVersion, Action: action, Request: request, Item: *found}, nil
}

// ApplyExecutionStatus rechecks a confirmed preview and emits one event only
// after a real semantic transition.
func (s *Service) ApplyExecutionStatus(ctx context.Context, workspace ExecutionWorkspace, input ExecutionStatusApplyInput, authorizer Authorizer, events EventSink) (ExecutionMutationResult, error) {
	result := ExecutionMutationResult{SchemaVersion: IntegrationContractVersion}
	if !input.Confirmed {
		return result, ErrConfirmationRequired
	}
	if err := authorizeExecutionMutation(ctx, workspace, authorizer, events); err != nil {
		return result, err
	}
	if input.ExpectedPlan.SchemaVersion != IntegrationContractVersion {
		return result, executionConflict("execution.status", "status preview is missing or changed")
	}
	fresh, err := s.PreviewExecutionStatus(ctx, workspace, input.ExpectedPlan.Request, authorizer)
	if err != nil {
		return result, err
	}
	if !samePublicationJSON(fresh, input.ExpectedPlan) {
		return result, executionConflict("execution.status", "status preview changed before apply")
	}
	result.Action = fresh.Action
	result.Item = fresh.Item
	if fresh.Action == MutationUnchanged {
		return result, nil
	}
	item, err := workspace.ApplyStatus(ctx, fresh.Request)
	if err != nil {
		if validPartialExecutionStatusItem(item, fresh.Item, fresh.Request) {
			result.Item = item
			event := Event{Name: EventExecutionStatusUpdated, ModuleID: s.moduleID, Artifact: item.Artifact, Source: &item.Reference, Outcome: MutationUpdate, OccurredAt: s.now().UTC()}
			result.Event = &event
			if auditErr := events.Publish(context.WithoutCancel(ctx), event); auditErr != nil {
				err = errors.Join(err, &IntegrationError{Class: IntegrationPartialFailure, Operation: "execution.status.audit", Message: "Project status changed but audit persistence failed", Err: auditErr})
			}
		}
		return result, err
	}
	if item.Artifact != fresh.Item.Artifact || !sameExternalLocation(item.Reference, fresh.Item.Reference) || item.Status != fresh.Request.Status {
		return result, &IntegrationError{Class: IntegrationPartialFailure, Operation: "execution.status", Message: "Project status mutation returned invalid evidence; inspect before retrying"}
	}
	result.Item = item
	event := Event{Name: EventExecutionStatusUpdated, ModuleID: s.moduleID, Artifact: item.Artifact, Source: &item.Reference, Outcome: MutationUpdate, OccurredAt: s.now().UTC()}
	result.Event = &event
	if err := events.Publish(context.WithoutCancel(ctx), event); err != nil {
		return result, &IntegrationError{Class: IntegrationPartialFailure, Operation: "execution.status.audit", Message: "Project status changed but audit persistence failed", Err: err}
	}
	return result, nil
}

func authorizeExecutionRead(ctx context.Context, workspace ExecutionWorkspace, authorizer Authorizer) error {
	if workspace == nil {
		return fmt.Errorf("execution workspace is unavailable")
	}
	if authorizer == nil {
		return fmt.Errorf("execution workspace requires an authorizer")
	}
	return authorizer.Require(ctx, PermissionRead)
}

func authorizeExecutionMutation(ctx context.Context, workspace ExecutionWorkspace, authorizer Authorizer, events EventSink) error {
	if workspace == nil {
		return fmt.Errorf("execution workspace is unavailable")
	}
	if authorizer == nil {
		return fmt.Errorf("execution workspace requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionExecute); err != nil {
		return err
	}
	if events == nil {
		return ErrEventSinkRequired
	}
	return nil
}

func validateExecutionAttachRequest(request ExecutionAttachRequest) error {
	if request.Artifact.Validate() != nil || !validExecutionReference(request.Workspace, "project") || !validExecutionReference(request.PublishedWork, "issue") {
		return executionManual("execution.attach", "attachment request has an invalid stable identity")
	}
	return nil
}

func validateExecutionSnapshot(snapshot ExecutionWorkspaceSnapshot, requested ExternalReference) error {
	if snapshot.SchemaVersion != IntegrationContractVersion || !validExecutionReference(snapshot.Workspace, "project") || !sameExternalIdentity(snapshot.Workspace, requested) || snapshot.Workspace.Revision == "" || snapshot.ConfigurationRevision == "" {
		return executionConflict("execution.inspect", "Project workspace identity or revision changed")
	}
	seenItems := map[[4]string]bool{}
	statusFields := 0
	seenStatuses := map[ExecutionStatus]bool{}
	for _, field := range snapshot.Fields {
		if field.Role != ExecutionFieldRoleStatus {
			continue
		}
		statusFields++
		for _, status := range field.SupportedStatuses {
			if validateExecutionStatus(status) != nil || seenStatuses[status] {
				return executionManual("execution.inspect", "Project returned invalid or duplicate semantic Status options")
			}
			seenStatuses[status] = true
		}
	}
	if statusFields != 1 {
		return executionManual("execution.inspect", "Project must expose exactly one semantic Status field")
	}
	for _, item := range snapshot.Items {
		key := externalLocationKey(item.Reference)
		if item.Artifact.Validate() != nil || !validExecutionReference(item.PublishedWork, "issue") || !validExecutionReference(item.Reference, "project_item") || item.Reference.Revision == "" || key == ([4]string{}) || seenItems[key] {
			return executionManual("execution.inspect", "Project returned invalid or duplicate tracked item evidence")
		}
		seenItems[key] = true
		if item.Status != "" {
			if err := validateExecutionStatus(item.Status); err != nil {
				return executionManual("execution.inspect", "Project returned an unsupported item status")
			}
		}
	}
	return nil
}

func validateAttachedExecutionItem(item ExecutionWorkItem, request ExecutionAttachRequest) error {
	if !validPartialAttachedExecutionItem(item, request) || item.Reference.Revision == "" {
		return errors.New("attached item identity disagrees with reviewed request")
	}
	return nil
}

func validPartialAttachedExecutionItem(item ExecutionWorkItem, request ExecutionAttachRequest) bool {
	return item.Artifact == request.Artifact && sameExternalLocation(item.PublishedWork, request.PublishedWork) && validExecutionReference(item.Reference, "project_item")
}

func validPartialExecutionStatusItem(item, previous ExecutionWorkItem, request ExecutionStatusRequest) bool {
	return item.Artifact == previous.Artifact && sameExternalLocation(item.Reference, previous.Reference) && item.Status == request.Status
}

func executionStatusSupported(fields []ExecutionField, status ExecutionStatus) bool {
	matches := 0
	for _, field := range fields {
		if field.Role != ExecutionFieldRoleStatus {
			continue
		}
		for _, supported := range field.SupportedStatuses {
			if supported == status {
				matches++
			}
		}
	}
	return matches == 1
}

func validateExecutionStatus(status ExecutionStatus) error {
	switch status {
	case ExecutionTodo, ExecutionInProgress, ExecutionInReview, ExecutionDone:
		return nil
	default:
		return fmt.Errorf("unsupported execution status %q", status)
	}
}

func validExecutionReference(ref ExternalReference, kind string) bool {
	return validPublicationReference(ref) && ref.Kind == kind && ref.URL != "" && ref.DisplayID != ""
}

func executionConflict(operation, message string) error {
	return &IntegrationError{Class: IntegrationRevisionConflict, Operation: operation, Message: message}
}

func executionManual(operation, message string) error {
	return &IntegrationError{Class: IntegrationManualRemediationRequired, Operation: operation, Message: message, ManualFallbackAllowed: true}
}
