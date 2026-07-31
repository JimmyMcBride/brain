package planning

import (
	"context"
	"fmt"
	"sync"

	"github.com/JimmyMcBride/brain/internal/modules"
	"github.com/JimmyMcBride/brain/planning/application"
	"github.com/JimmyMcBride/brain/planning/local"
)

const (
	ID = "official.planning"

	CommandGroup = "plan"

	PermissionContextRead = "project.context.read"
)

func Registration() modules.Registration {
	return modules.Registration{
		Descriptor: modules.Descriptor{
			ID:            ID,
			Name:          "Brain Planning",
			Version:       "0.1.0",
			BrainAPIMajor: modules.BrainAPIMajor,
			ConfigVersion: 1,
			Capabilities: []string{
				"commands",
				"context.read",
				"events.publish",
				"storage.local-plan",
			},
			Permissions: []string{
				application.PermissionRead,
				application.PermissionBrainstorm,
				application.PermissionRoadmap,
				PermissionContextRead,
			},
			Commands: []string{CommandGroup},
			Events: []string{
				application.EventBrainstormCreated,
				application.EventRoadmapUpdated,
			},
		},
		Factory: func() modules.Module {
			return &Module{}
		},
	}
}

type Module struct {
	mu      sync.RWMutex
	service *application.Service
}

func (m *Module) Validate(_ context.Context, _ modules.ModuleContext, config modules.Config) error {
	if len(config) != 0 {
		return fmt.Errorf("Planning does not accept project configuration in this phase")
	}
	return nil
}

func (m *Module) Initialize(_ context.Context, moduleContext modules.ModuleContext, _ modules.Config) error {
	repository := local.New(moduleContext.Project.Root)
	service := application.New(repository, application.Options{ModuleID: ID})
	m.mu.Lock()
	m.service = service
	m.mu.Unlock()
	return nil
}

func (m *Module) Health(ctx context.Context, _ modules.ModuleContext) modules.Health {
	service := m.PlanningService()
	if service == nil {
		return modules.Health{Status: modules.HealthUnhealthy, Message: "Planning is not initialized"}
	}
	status, err := service.Status(ctx)
	if err != nil {
		return modules.Health{Status: modules.HealthUnhealthy, Message: err.Error()}
	}
	switch status.State {
	case application.WorkspaceCompatible:
		return modules.Health{Status: modules.HealthHealthy, Message: status.Message}
	case application.WorkspaceInvalid:
		return modules.Health{Status: modules.HealthUnhealthy, Message: status.Message}
	default:
		return modules.Health{Status: modules.HealthDegraded, Message: status.Message}
	}
}

func (m *Module) PlanningService() *application.Service {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.service
}
