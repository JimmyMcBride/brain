package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/JimmyMcBride/brain/planning"
)

// PreviewRoadmapParking returns one idempotent parking-lot append without writing.
func (s *Service) PreviewRoadmapParking(ctx context.Context, input RoadmapParkingInput) (RoadmapPreview, error) {
	if err := input.BrainstormID.Validate(); err != nil {
		return RoadmapPreview{}, err
	}
	if strings.TrimSpace(input.Title) == "" {
		return RoadmapPreview{}, fmt.Errorf("parking title is required")
	}
	roadmap, err := s.ReadRoadmap(ctx)
	if err != nil {
		return RoadmapPreview{}, err
	}
	entry := fmt.Sprintf("- %s | value: %s | parked because: %s | unlock: %s | source: [Brainstorm](../brainstorms/%s.md)",
		strings.TrimSpace(input.Title), strings.TrimSpace(input.Value), strings.TrimSpace(input.Reason),
		strings.TrimSpace(input.Unlock), input.BrainstormID,
	)
	body, changed := appendUniqueSection(roadmap.Body, "Parking Lot", entry)
	action := MutationUpdate
	if !changed {
		action = MutationUnchanged
	}
	roadmap.Body = body
	return RoadmapPreview{Action: action, Document: roadmap}, nil
}

// ParkRoadmap applies one confirmed, authorized, audited parking-lot append.
func (s *Service) ParkRoadmap(ctx context.Context, input RoadmapParkingInput, authorizer Authorizer, events EventSink) (RoadmapResult, error) {
	preview, err := s.PreviewRoadmapParking(ctx, input)
	if err != nil {
		return RoadmapResult{}, err
	}
	if preview.Action == MutationUnchanged {
		return RoadmapResult{Action: MutationUnchanged, Document: preview.Document}, nil
	}
	return s.UpdateRoadmap(ctx, UpdateRoadmapInput{Body: preview.Document.Body, Confirmed: input.Confirmed}, authorizer, events)
}

// ReadRoadmap returns the complete local roadmap Markdown.
func (s *Service) ReadRoadmap(ctx context.Context) (RoadmapDocument, error) {
	if err := s.requireReadable(ctx); err != nil {
		return RoadmapDocument{}, err
	}
	return s.repository.ReadRoadmap(ctx)
}

// PreviewRoadmap returns the mutation needed for replacement Markdown without writing.
func (s *Service) PreviewRoadmap(ctx context.Context, body string) (RoadmapPreview, error) {
	if err := s.requireWritable(ctx); err != nil {
		return RoadmapPreview{}, err
	}
	current, err := s.repository.ReadRoadmap(ctx)
	if err != nil {
		return RoadmapPreview{}, err
	}
	action := MutationUpdate
	if current.Body == body {
		action = MutationUnchanged
	}
	current.Body = body
	return RoadmapPreview{Action: action, Document: current}, nil
}

// UpdateRoadmap applies a confirmed, authorized, audited roadmap replacement.
func (s *Service) UpdateRoadmap(
	ctx context.Context,
	input UpdateRoadmapInput,
	authorizer Authorizer,
	events EventSink,
) (RoadmapResult, error) {
	preview, err := s.PreviewRoadmap(ctx, input.Body)
	if err != nil {
		return RoadmapResult{}, err
	}
	if preview.Action == MutationUnchanged {
		return RoadmapResult{Action: MutationUnchanged, Document: preview.Document}, nil
	}
	if !input.Confirmed {
		return RoadmapResult{}, ErrConfirmationRequired
	}
	if authorizer == nil {
		return RoadmapResult{}, fmt.Errorf("planning mutation requires an authorizer")
	}
	if err := authorizer.Require(ctx, PermissionRoadmap); err != nil {
		return RoadmapResult{}, err
	}
	if events == nil {
		return RoadmapResult{}, ErrEventSinkRequired
	}
	document, action, err := s.repository.ReplaceRoadmap(ctx, input.Body)
	if err != nil {
		return RoadmapResult{}, err
	}
	result := RoadmapResult{Action: action, Document: document}
	if action == MutationUnchanged {
		return result, nil
	}
	now := s.now().UTC()
	event := Event{
		Name: EventRoadmapUpdated, ModuleID: s.moduleID,
		Artifact: planning.ArtifactRef{Kind: planning.ArtifactRoadmap, ID: "roadmap"},
		Outcome:  action, OccurredAt: now,
	}
	if err := events.Publish(ctx, event); err != nil {
		return RoadmapResult{}, fmt.Errorf("publish planning event: %w", err)
	}
	result.Event = &event
	return result, nil
}
