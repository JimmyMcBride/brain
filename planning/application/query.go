package application

import (
	"context"
	"sort"

	"github.com/JimmyMcBride/brain/planning"
)

// ProjectStatus returns aggregate local spec lifecycle state.
func (s *Service) ProjectStatus(ctx context.Context) (ProjectStatus, error) {
	workspace, err := s.readableWorkspace(ctx)
	if err != nil {
		return ProjectStatus{}, err
	}
	documents, err := s.repository.QuerySpecs(ctx, nil)
	if err != nil {
		return ProjectStatus{}, err
	}
	status := ProjectStatus{
		Project:       workspace.Project,
		PlanningModel: workspace.PlanningModel,
		SourceMode:    workspace.Ownership,
		TotalSpecs:    len(documents),
	}
	for _, document := range documents {
		switch planning.SpecStatus(document.Status) {
		case planning.SpecApproved:
			status.ApprovedSpecs++
			status.ReadySpecs = append(status.ReadySpecs, SpecSummary{
				ID: document.ID, Title: document.Title, Status: planning.SpecApproved, Initiative: document.Initiative,
			})
		case planning.SpecImplementing:
			status.ImplementingSpecs++
		case planning.SpecDone:
			status.DoneSpecs++
		default:
			status.DraftSpecs++
		}
	}
	sort.Slice(status.ReadySpecs, func(i, j int) bool {
		if status.ReadySpecs[i].Title == status.ReadySpecs[j].Title {
			return status.ReadySpecs[i].ID < status.ReadySpecs[j].ID
		}
		return status.ReadySpecs[i].Title < status.ReadySpecs[j].Title
	})
	return status, nil
}
