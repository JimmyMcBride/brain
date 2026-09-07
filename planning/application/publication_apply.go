package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// PermissionPublish authorizes publication and integration mapping mutations.
const PermissionPublish = "planning.publish"

// EventPublicationApplied records publication changes, including partial apply.
const EventPublicationApplied = "planning.publication.applied"

// PublicationApplyInput binds confirmation to the complete reviewed plan and
// canonical intent. A fresh preview must agree before the provider can apply it.
type PublicationApplyInput struct {
	Intent       PublicationPreviewInput `json:"intent"`
	ExpectedPlan PublicationPlan         `json:"expected_plan"`
	Confirmed    bool                    `json:"confirmed"`
}

// PublicationApplyResult retains the fresh plan, provider evidence, and attempted
// audit event even when publication or audit persistence fails. It does not claim
// that external mappings have been saved.
type PublicationApplyResult struct {
	SchemaVersion int               `json:"schema_version"`
	Plan          PublicationPlan   `json:"plan"`
	Evidence      PublicationResult `json:"evidence"`
	Event         *Event            `json:"event,omitempty"`
}

// ApplyPublication checks confirmation, publish/read grants, audit availability,
// and fresh provider evidence before applying the reviewed actions. It never
// rolls back remote changes or saves mappings. Adapters must still guard the
// final read/write window and report any partial mutation evidence.
func (s *Service) ApplyPublication(ctx context.Context, target PublicationTarget, input PublicationApplyInput, authorizer Authorizer, events EventSink) (PublicationApplyResult, error) {
	var result PublicationApplyResult
	if !input.Confirmed {
		return result, ErrConfirmationRequired
	}
	if authorizer == nil {
		return result, fmt.Errorf("publication requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionPublish); err != nil {
		return result, err
	}
	if events == nil {
		return result, ErrEventSinkRequired
	}
	if input.ExpectedPlan.SchemaVersion != IntegrationContractVersion || len(input.ExpectedPlan.Actions) == 0 {
		return result, publicationRevisionConflict()
	}
	plan, err := s.PreviewPublication(ctx, target, input.Intent, authorizer)
	if err != nil {
		return result, err
	}
	result = PublicationApplyResult{SchemaVersion: IntegrationContractVersion, Plan: plan, Evidence: PublicationResult{SchemaVersion: IntegrationContractVersion}}
	if !samePublicationJSON(plan, input.ExpectedPlan) {
		return result, publicationRevisionConflict()
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	changed := false
	for _, action := range plan.Actions {
		changed = changed || publicationWrites(action)
	}
	if !changed {
		for _, action := range plan.Actions {
			evidence := PublicationActionEvidence{Action: action}
			if action.Artifact != nil && action.Artifact.Reference != nil {
				evidence.References = []ExternalReference{*action.Artifact.Reference}
			}
			result.Evidence.Completed = append(result.Evidence.Completed, evidence)
		}
		return result, nil
	}
	result.Evidence, err = target.Apply(ctx, plan)
	if evidenceErr := validatePublicationCompletion(plan, result.Evidence, err); evidenceErr != nil {
		err = errors.Join(err, evidenceErr)
	}
	// Audit evidence of actual writes, not merely planned writes. Preserve all
	// partial evidence for callers on either failure path.
	changed = false
	for i, evidence := range result.Evidence.Completed {
		if i < len(plan.Actions) && samePublicationJSON(evidence.Action, plan.Actions[i]) && publicationWrites(evidence.Action) {
			changed = true
		}
	}
	if failed := result.Evidence.Failed; failed != nil && len(failed.References) > 0 && len(result.Evidence.Completed) < len(plan.Actions) &&
		samePublicationJSON(failed.Action, plan.Actions[len(result.Evidence.Completed)]) && publicationWrites(failed.Action) {
		changed = true
	}
	if changed {
		if err != nil {
			err = &IntegrationError{Class: IntegrationPartialFailure, Operation: "publication.apply", Message: "publication changed provider state before failure", Err: err}
		}
		source := plan.Target
		event := Event{Name: EventPublicationApplied, ModuleID: s.moduleID, Source: &source, Outcome: MutationUpdate, OccurredAt: s.now().UTC()}
		result.Event = &event
		// Cancellation must not suppress auditing earlier provider writes.
		if auditErr := events.Publish(context.WithoutCancel(ctx), event); auditErr != nil {
			err = errors.Join(err, &IntegrationError{Class: IntegrationPartialFailure, Operation: "publication.audit", Message: "publication changed provider state but audit persistence failed", Err: auditErr})
		}
	}
	return result, err
}

func publicationRevisionConflict() error {
	return &IntegrationError{Class: IntegrationRevisionConflict, Operation: "publication.apply", Message: "publication preview is missing or changed; review a fresh preview"}
}

func publicationWrites(action PublicationApplyAction) bool {
	return action.Action == MutationCreate || action.Action == MutationUpdate
}

// Compare wire representations so omitempty slices survive JSON round trips.
func samePublicationJSON(a, b any) bool {
	left, leftErr := json.Marshal(a)
	right, rightErr := json.Marshal(b)
	return leftErr == nil && rightErr == nil && bytes.Equal(left, right)
}

func validatePublicationCompletion(plan PublicationPlan, evidence PublicationResult, applyErr error) error {
	invalid := func() error {
		return &IntegrationError{Class: IntegrationPartialFailure, Operation: "publication.apply", Message: "provider returned incomplete or inconsistent publication evidence; inspect before retrying"}
	}
	// A provider can fail before producing evidence, for example when disabled.
	if applyErr != nil && len(evidence.Completed) == 0 && evidence.Failed == nil {
		return nil
	}
	if evidence.SchemaVersion != IntegrationContractVersion || len(evidence.Completed) > len(plan.Actions) {
		return invalid()
	}
	for i, completed := range evidence.Completed {
		if !samePublicationJSON(completed.Action, plan.Actions[i]) {
			return invalid()
		}
		if artifact := completed.Action.Artifact; artifact != nil {
			matched := false
			for _, ref := range completed.References {
				if ref.Provider == plan.Target.Provider && ref.Kind != "" && ref.OpaqueID != "" &&
					(artifact.Reference == nil || sameExternalIdentity(ref, *artifact.Reference)) {
					matched = true
				}
			}
			if !matched {
				return invalid()
			}
		}
	}
	if failed := evidence.Failed; failed != nil {
		if len(evidence.Completed) == len(plan.Actions) || !samePublicationJSON(failed.Action, plan.Actions[len(evidence.Completed)]) {
			return invalid()
		}
	}
	if applyErr == nil && (evidence.Failed != nil || len(evidence.Completed) != len(plan.Actions)) {
		return invalid()
	}
	return nil
}
