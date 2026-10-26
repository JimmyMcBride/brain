package planning

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/JimmyMcBride/brain/internal/modules"
	"github.com/JimmyMcBride/brain/planning/application"
	planninggithub "github.com/JimmyMcBride/brain/planning/github"
	"github.com/JimmyMcBride/brain/planning/local"
)

const (
	ID = "official.planning"

	CommandGroup = "plan"

	PermissionContextRead = "project.context.read"
	configVersion         = 2
)

func Registration() modules.Registration {
	return RegistrationWithOptions(Options{})
}

// Options supplies host-owned GitHub dependencies without performing eager
// provider work during registration or disabled module initialization.
type Options struct {
	GitHubRunner planninggithub.Runner
	GitRunner    planninggithub.Runner
	Now          func() time.Time
}

func RegistrationWithOptions(options Options) modules.Registration {
	return modules.Registration{
		Descriptor: modules.Descriptor{
			ID:            ID,
			Name:          "Brain Planning",
			Version:       "0.2.0",
			BrainAPIMajor: modules.BrainAPIMajor,
			ConfigVersion: configVersion,
			Capabilities: []string{
				"commands",
				"context.read",
				"events.publish",
				"integration.github",
				"storage.local-plan",
			},
			Permissions: []string{
				application.PermissionRead,
				application.PermissionBrainstorm,
				application.PermissionCollaboration,
				application.PermissionRoadmap,
				application.PermissionSpecEdit,
				application.PermissionSpecApprove,
				application.PermissionPublish,
				application.PermissionExecute,
				PermissionContextRead,
			},
			Commands: []string{CommandGroup},
			Events: []string{
				application.EventBrainstormCreated,
				application.EventBrainstormUpdated,
				application.EventGuidedSessionUpdated,
				application.EventBrainstormPromoted,
				application.EventRoadmapUpdated,
				application.EventSpecUpdated,
				application.EventSpecExecutionStarted,
				application.EventCollaborationRepaired,
				application.EventPublicationApplied,
				application.EventIntegrationAdopted,
				application.EventIntegrationConfigured,
				application.EventIntegrationReconciled,
				application.EventExecutionAttached,
				application.EventExecutionStatusUpdated,
			},
		},
		Factory: func() modules.Module {
			return &Module{options: options}
		},
	}
}

type Module struct {
	mu            sync.RWMutex
	options       Options
	service       *application.Service
	github        *planninggithub.Adapter
	githubEnabled bool
	projectRoot   string
}

func (m *Module) Validate(_ context.Context, _ modules.ModuleContext, config modules.Config) error {
	_, err := normalizeConfig(config)
	return err
}

func (m *Module) MigrateConfig(_ context.Context, _ modules.ModuleContext, from int, config modules.Config) (modules.Config, error) {
	if from != 1 || len(config) != 0 {
		return nil, fmt.Errorf("Planning config version %d cannot be migrated automatically", from)
	}
	return defaultConfig(), nil
}

func (m *Module) Initialize(_ context.Context, moduleContext modules.ModuleContext, config modules.Config) error {
	normalized, err := normalizeConfig(config)
	if err != nil {
		return err
	}
	enabled := normalized["github"].(modules.Config)["enabled"].(bool)
	repository := local.New(moduleContext.Project.Root)
	service := application.New(repository, application.Options{ModuleID: ID, ProjectRoot: moduleContext.Project.Root})
	adapter := planninggithub.New(planninggithub.Config{Enabled: enabled}, planninggithub.Options{ProjectRoot: moduleContext.Project.Root, Runner: m.options.GitHubRunner, GitRunner: m.options.GitRunner, Now: m.options.Now})
	m.mu.Lock()
	m.service = service
	m.github = adapter
	m.githubEnabled = enabled
	m.projectRoot = moduleContext.Project.Root
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
	localHealth := modules.Health{Status: modules.HealthDegraded, Message: status.Message}
	switch status.State {
	case application.WorkspaceCompatible:
		localHealth = modules.Health{Status: modules.HealthHealthy, Message: status.Message}
	case application.WorkspaceInvalid:
		return modules.Health{Status: modules.HealthUnhealthy, Message: status.Message}
	}
	m.mu.RLock()
	adapter, githubEnabled := m.github, m.githubEnabled
	m.mu.RUnlock()
	if githubEnabled {
		if err := adapter.Health(ctx); err != nil {
			return modules.Health{Status: modules.HealthDegraded, Message: localHealth.Message + "; GitHub degraded: " + err.Error()}
		}
		localHealth.Message += "; GitHub healthy"
	}
	return localHealth
}

func (m *Module) PlanningService() *application.Service {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.service
}

func (m *Module) GitHubAdapter() (*planninggithub.Adapter, bool) {
	if m == nil {
		return nil, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.github, m.githubEnabled
}

func (m *Module) GitHubPreflight(ctx context.Context) (planninggithub.RepositoryStatus, error) {
	if m == nil {
		return planninggithub.RepositoryStatus{}, fmt.Errorf("Planning is not initialized")
	}
	m.mu.RLock()
	root, options := m.projectRoot, m.options
	m.mu.RUnlock()
	adapter := planninggithub.New(planninggithub.Config{Enabled: true}, planninggithub.Options{ProjectRoot: root, Runner: options.GitHubRunner, GitRunner: options.GitRunner, Now: options.Now})
	return adapter.Preflight(ctx)
}

func (m *Module) GitHubConfig(enabled bool) modules.Config {
	return modules.Config{"github": modules.Config{"enabled": enabled}}
}

func defaultConfig() modules.Config {
	return modules.Config{"github": modules.Config{"enabled": false}}
}

func normalizeConfig(config modules.Config) (modules.Config, error) {
	if config == nil {
		config = modules.Config{}
	}
	if len(config) == 0 {
		maps.Copy(config, defaultConfig())
	}
	if len(config) != 1 {
		return nil, fmt.Errorf("Planning configuration accepts only github.enabled")
	}
	raw, ok := config["github"]
	if !ok {
		return nil, fmt.Errorf("Planning configuration requires github.enabled")
	}
	var githubConfig modules.Config
	switch value := raw.(type) {
	case modules.Config:
		githubConfig = value
	case map[string]any:
		githubConfig = modules.Config(value)
	default:
		return nil, fmt.Errorf("Planning configuration github must be an object")
	}
	if len(githubConfig) != 1 {
		return nil, fmt.Errorf("Planning configuration accepts only github.enabled")
	}
	enabled, ok := githubConfig["enabled"].(bool)
	if !ok {
		return nil, fmt.Errorf("Planning configuration github.enabled must be a boolean")
	}
	config["github"] = modules.Config{"enabled": enabled}
	return config, nil
}
