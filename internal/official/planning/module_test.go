package planning

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/internal/modules"
	"github.com/JimmyMcBride/brain/planning/application"
	planninggithub "github.com/JimmyMcBride/brain/planning/github"
)

func TestRegistrationDeclaresBoundedPlanningContract(t *testing.T) {
	descriptor := Registration().Descriptor
	if descriptor.ID != ID || descriptor.Version != "0.2.0" || descriptor.ConfigVersion != 2 {
		t.Fatalf("unexpected descriptor identity: %#v", descriptor)
	}
	if strings.Join(descriptor.Commands, ",") != CommandGroup {
		t.Fatalf("unexpected commands: %v", descriptor.Commands)
	}
	for _, required := range []string{application.EventBrainstormCreated, application.EventRoadmapUpdated} {
		if !strings.Contains(strings.Join(descriptor.Events, ","), required) {
			t.Fatalf("missing event %q: %v", required, descriptor.Events)
		}
	}
	permissions := strings.Join(descriptor.Permissions, ",")
	for _, required := range []string{
		application.PermissionRead,
		application.PermissionBrainstorm,
		application.PermissionCollaboration,
		application.PermissionPublish,
		application.PermissionRoadmap,
		PermissionContextRead,
	} {
		if !strings.Contains(permissions, required) {
			t.Fatalf("missing permission %q: %v", required, descriptor.Permissions)
		}
	}
	if strings.Contains(permissions, "memory.") {
		t.Fatalf("Planning must not declare memory mutation permission: %v", descriptor.Permissions)
	}
}

type failingGitHubRunner struct{ calls int }

func (r *failingGitHubRunner) Run(context.Context, string, ...string) (planninggithub.RunResult, error) {
	r.calls++
	return planninggithub.RunResult{}, errors.New("unavailable")
}

func TestPlanningConfigV1MigratesDisabledAndValidatesV2(t *testing.T) {
	root := t.TempDir()
	copyCompatibleWorkspace(t, root)
	configs := modules.NewMemoryConfigStore()
	if err := configs.Save(modules.ProjectConfig{SchemaVersion: 1, Modules: map[string]modules.ModuleConfig{ID: {Enabled: true, ConfigVersion: 1, Config: modules.Config{}}}}); err != nil {
		t.Fatal(err)
	}
	grants := modules.NewMemoryGrantStore()
	permissions := Registration().Descriptor.Permissions
	if err := grants.Save(modules.GrantSet{ID: permissions}); err != nil {
		t.Fatal(err)
	}
	runtime := modules.NewRuntime(modules.ProjectRef{Root: root}, mustPlanningRegistry(t, Registration()), configs, grants)
	report, err := runtime.Show(context.Background(), ID)
	if err != nil || report.State != modules.StateEnabled || report.ConfigVersion != 2 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	stored, err := configs.Load()
	if err != nil {
		t.Fatal(err)
	}
	githubConfig, ok := stored.Modules[ID].Config["github"].(modules.Config)
	if !ok || githubConfig["enabled"] != false {
		t.Fatalf("migrated config=%#v", stored.Modules[ID].Config)
	}
	if _, err := runtime.Configure(context.Background(), ID, modules.Config{"unknown": true}); err == nil {
		t.Fatal("expected unknown config rejection")
	}
}

func TestGitHubHealthDegradesWithoutBreakingLocalPlanning(t *testing.T) {
	root := t.TempDir()
	copyCompatibleWorkspace(t, root)
	runner := &failingGitHubRunner{}
	registration := RegistrationWithOptions(Options{GitHubRunner: runner})
	configs := modules.NewMemoryConfigStore()
	if err := configs.Save(modules.ProjectConfig{SchemaVersion: 1, Modules: map[string]modules.ModuleConfig{ID: {Enabled: true, ConfigVersion: 2, Config: modules.Config{"github": modules.Config{"enabled": true}}}}}); err != nil {
		t.Fatal(err)
	}
	grants := modules.NewMemoryGrantStore()
	if err := grants.Save(modules.GrantSet{ID: registration.Descriptor.Permissions}); err != nil {
		t.Fatal(err)
	}
	runtime := modules.NewRuntime(modules.ProjectRef{Root: root}, mustPlanningRegistry(t, registration), configs, grants)
	report, err := runtime.Show(context.Background(), ID)
	if err != nil || report.State != modules.StateEnabled || report.Health == nil || report.Health.Status != modules.HealthDegraded || runner.calls != 1 {
		t.Fatalf("report=%+v calls=%d err=%v", report, runner.calls, err)
	}
	resolution, err := runtime.ResolveCommand(context.Background(), CommandGroup, application.PermissionRead)
	if err != nil {
		t.Fatal(err)
	}
	provider := resolution.Module.(interface{ PlanningService() *application.Service })
	if _, err := provider.PlanningService().ProjectStatus(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func mustPlanningRegistry(t *testing.T, registration modules.Registration) *modules.Registry {
	t.Helper()
	registry, err := modules.NewRegistry([]modules.Registration{registration})
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestModuleRuntimeHealthAndServiceFollowWorkspaceState(t *testing.T) {
	root := t.TempDir()
	copyCompatibleWorkspace(t, root)
	registry, err := modules.NewRegistry([]modules.Registration{Registration()})
	if err != nil {
		t.Fatal(err)
	}
	runtime := modules.NewRuntime(
		modules.ProjectRef{Root: root},
		registry,
		modules.NewMemoryConfigStore(),
		modules.NewMemoryGrantStore(),
	)
	ctx := context.Background()
	for _, permission := range Registration().Descriptor.Permissions {
		if _, err := runtime.Grant(ctx, ID, permission); err != nil {
			t.Fatal(err)
		}
	}
	report, err := runtime.Enable(ctx, ID)
	if err != nil {
		t.Fatal(err)
	}
	if report.State != modules.StateEnabled || report.Health == nil || report.Health.Status != modules.HealthHealthy {
		t.Fatalf("unexpected enabled report: %#v", report)
	}
	resolution, err := runtime.ResolveCommand(ctx, CommandGroup, application.PermissionRead)
	if err != nil {
		t.Fatal(err)
	}
	provider, ok := resolution.Module.(interface {
		PlanningService() *application.Service
	})
	if !ok || provider.PlanningService() == nil {
		t.Fatalf("resolved module does not provide Planning service: %#v", resolution)
	}
}

func copyCompatibleWorkspace(t *testing.T, root string) {
	t.Helper()
	if err := os.CopyFS(root, os.DirFS(filepath.Join("..", "..", "..", "planning", "local", "testdata", "compatible"))); err != nil {
		t.Fatal(err)
	}
}
