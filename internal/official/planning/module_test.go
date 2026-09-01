package planning

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/internal/modules"
	"github.com/JimmyMcBride/brain/planning/application"
)

func TestRegistrationDeclaresBoundedPlanningContract(t *testing.T) {
	descriptor := Registration().Descriptor
	if descriptor.ID != ID || descriptor.Version != "0.1.0" {
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
