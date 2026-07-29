package modules_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"brain/internal/modules"
	"brain/internal/modules/testmodule"
)

func TestDescriptorAndRegistryValidation(t *testing.T) {
	valid := testmodule.Registration(&testmodule.Counters{}, testmodule.Options{})
	tests := []struct {
		name          string
		registrations []modules.Registration
		want          string
	}{
		{name: "duplicate", registrations: []modules.Registration{valid, valid}, want: "duplicate module registration"},
		{name: "invalid ID", registrations: []modules.Registration{withDescriptor(valid, func(d *modules.Descriptor) { d.ID = "Invalid" })}, want: "reverse-domain"},
		{name: "invalid semver", registrations: []modules.Registration{withDescriptor(valid, func(d *modules.Descriptor) { d.Version = "one" })}, want: "semantic version"},
		{name: "incompatible API", registrations: []modules.Registration{withDescriptor(valid, func(d *modules.Descriptor) { d.BrainAPIMajor++ })}, want: "supported major"},
		{name: "invalid config version", registrations: []modules.Registration{withDescriptor(valid, func(d *modules.Descriptor) { d.ConfigVersion = 0 })}, want: "config version"},
		{name: "duplicate permission", registrations: []modules.Registration{withDescriptor(valid, func(d *modules.Descriptor) { d.Permissions = []string{"test.read", "test.read"} })}, want: "duplicate permission"},
		{name: "permission whitespace", registrations: []modules.Registration{withDescriptor(valid, func(d *modules.Descriptor) { d.Permissions = []string{" test.read"} })}, want: "surrounding whitespace"},
		{name: "invalid command", registrations: []modules.Registration{withDescriptor(valid, func(d *modules.Descriptor) { d.Commands = []string{"Plan"} })}, want: "invalid command"},
		{name: "invalid event", registrations: []modules.Registration{withDescriptor(valid, func(d *modules.Descriptor) { d.Events = []string{"created"} })}, want: "invalid event"},
		{name: "missing factory", registrations: []modules.Registration{{Descriptor: valid.Descriptor}}, want: "no factory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := modules.NewRegistry(tt.registrations)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestRegistryRejectsCommandAndEventCollisions(t *testing.T) {
	first := withDescriptor(testmodule.Registration(nil, testmodule.Options{}), func(d *modules.Descriptor) {
		d.Commands = []string{"plan"}
		d.Events = []string{"planning.brainstorm.created"}
	})
	second := withDescriptor(first, func(d *modules.Descriptor) {
		d.ID = "dev.brain.other"
	})
	if _, err := modules.NewRegistry([]modules.Registration{first, second}); err == nil || !strings.Contains(err.Error(), "command") {
		t.Fatalf("expected command collision, got %v", err)
	}
	second = withDescriptor(second, func(d *modules.Descriptor) {
		d.Commands = []string{"other"}
	})
	if _, err := modules.NewRegistry([]modules.Registration{first, second}); err == nil || !strings.Contains(err.Error(), "event") {
		t.Fatalf("expected event collision, got %v", err)
	}
}

func TestRuntimeResolvesDeclaredCommandsOnlyWhenEnabledAndAuthorized(t *testing.T) {
	ctx := context.Background()
	registration := withDescriptor(testmodule.Registration(nil, testmodule.Options{}), func(d *modules.Descriptor) {
		d.Commands = []string{"plan"}
		d.Events = []string{"planning.brainstorm.created"}
	})
	runtime := modules.NewRuntime(
		modules.ProjectRef{Root: t.TempDir()},
		mustRegistry(t, registration),
		modules.NewMemoryConfigStore(),
		modules.NewMemoryGrantStore(),
	)

	if _, err := runtime.ResolveCommand(ctx, "missing", "test.read"); err == nil || !strings.Contains(err.Error(), "not compiled") {
		t.Fatalf("expected unavailable command, got %v", err)
	}
	if _, err := runtime.ResolveCommand(ctx, "plan", "test.read"); err == nil {
		t.Fatal("expected disabled command failure")
	} else {
		var failure *modules.Failure
		if !errors.As(err, &failure) || failure.Code != modules.FailureModuleDisabled {
			t.Fatalf("expected module_disabled, got %v", err)
		}
	}
	if _, err := runtime.Grant(ctx, testmodule.ID, "test.read"); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Enable(ctx, testmodule.ID); err != nil {
		t.Fatal(err)
	}
	resolution, err := runtime.ResolveCommand(ctx, "plan", "test.read")
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ModuleID != testmodule.ID || resolution.Module == nil {
		t.Fatalf("unexpected resolution: %#v", resolution)
	}
	if _, err := runtime.ResolveCommand(ctx, "plan", "test.write"); err == nil || !strings.Contains(err.Error(), "does not declare") {
		t.Fatalf("expected undeclared permission failure, got %v", err)
	}
}

func TestRuntimeDoesNotInstantiateUnavailableDisabledOrBlockedModules(t *testing.T) {
	ctx := context.Background()
	counters := &testmodule.Counters{}
	registry := mustRegistry(t, testmodule.Registration(counters, testmodule.Options{}))
	configs := modules.NewMemoryConfigStore()
	grants := modules.NewMemoryGrantStore()
	runtime := modules.NewRuntime(modules.ProjectRef{Root: t.TempDir()}, registry, configs, grants)

	reports, err := runtime.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	requireState(t, reports[0], modules.StateAvailable)
	requireCounters(t, counters, 0, 0, 0, 0)

	if _, err := runtime.Disable(ctx, testmodule.ID); err != nil {
		t.Fatal(err)
	}
	reports, err = runtime.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	requireState(t, reports[0], modules.StateDisabled)
	requireCounters(t, counters, 0, 0, 0, 0)

	config := modules.ProjectConfig{
		SchemaVersion: 1,
		Modules: map[string]modules.ModuleConfig{
			testmodule.ID: {Enabled: true, ConfigVersion: 1, Config: modules.Config{}},
		},
	}
	if err := configs.Save(config); err != nil {
		t.Fatal(err)
	}
	reports, err = runtime.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	requireState(t, reports[0], modules.StateBlocked)
	if reports[0].Failure == nil || reports[0].Failure.Code != modules.FailurePermissionMissing {
		t.Fatalf("expected permission failure, got %#v", reports[0].Failure)
	}
	requireCounters(t, counters, 0, 0, 0, 0)
}

func TestRuntimeGrantEnableHealthAndInitializationOnce(t *testing.T) {
	ctx := context.Background()
	counters := &testmodule.Counters{}
	registry := mustRegistry(t, testmodule.Registration(counters, testmodule.Options{}))
	runtime := modules.NewRuntime(modules.ProjectRef{Root: t.TempDir()}, registry, modules.NewMemoryConfigStore(), modules.NewMemoryGrantStore())

	if _, err := runtime.Enable(ctx, testmodule.ID); err == nil || !strings.Contains(err.Error(), "brain modules grant "+testmodule.ID+" test.read") {
		t.Fatalf("expected exact grant command, got %v", err)
	}
	requireCounters(t, counters, 0, 0, 0, 0)

	if _, err := runtime.Grant(ctx, testmodule.ID, "test.read"); err != nil {
		t.Fatal(err)
	}
	report, err := runtime.Enable(ctx, testmodule.ID)
	if err != nil {
		t.Fatal(err)
	}
	requireState(t, report, modules.StateEnabled)
	if report.Health == nil || report.Health.Status != modules.HealthHealthy {
		t.Fatalf("expected healthy report, got %#v", report.Health)
	}
	if _, err := runtime.Show(ctx, testmodule.ID); err != nil {
		t.Fatal(err)
	}
	requireCounters(t, counters, 2, 2, 1, 2)
}

func TestRuntimeInvalidConfigAndInitializeFailure(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		opts testmodule.Options
		code modules.FailureCode
	}{
		{name: "validation", opts: testmodule.Options{ValidationError: errors.New("bad config")}, code: modules.FailureConfigInvalid},
		{name: "initialize", opts: testmodule.Options{InitializeError: errors.New("boom")}, code: modules.FailureInitializeFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counters := &testmodule.Counters{}
			registry := mustRegistry(t, testmodule.Registration(counters, tt.opts))
			configs := modules.NewMemoryConfigStore()
			grants := modules.NewMemoryGrantStore()
			if err := grants.Save(modules.GrantSet{testmodule.ID: {"test.read"}}); err != nil {
				t.Fatal(err)
			}
			runtime := modules.NewRuntime(modules.ProjectRef{Root: t.TempDir()}, registry, configs, grants)
			report, err := runtime.Enable(ctx, testmodule.ID)
			if err == nil {
				t.Fatal("expected enable error")
			}
			var failure *modules.Failure
			if !errors.As(err, &failure) || failure.Code != tt.code {
				t.Fatalf("expected %s, got report=%#v err=%v", tt.code, report, err)
			}
		})
	}
}

func TestRuntimeIncompatibleConfigVersionBlocksWithoutInstantiation(t *testing.T) {
	ctx := context.Background()
	counters := &testmodule.Counters{}
	registry := mustRegistry(t, testmodule.Registration(counters, testmodule.Options{}))
	configs := modules.NewMemoryConfigStore()
	if err := configs.Save(modules.ProjectConfig{
		SchemaVersion: 1,
		Modules: map[string]modules.ModuleConfig{
			testmodule.ID: {Enabled: true, ConfigVersion: 2, Config: modules.Config{}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	grants := modules.NewMemoryGrantStore()
	if err := grants.Save(modules.GrantSet{testmodule.ID: {"test.read"}}); err != nil {
		t.Fatal(err)
	}
	runtime := modules.NewRuntime(modules.ProjectRef{Root: t.TempDir()}, registry, configs, grants)
	reports, err := runtime.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	requireState(t, reports[0], modules.StateBlocked)
	if reports[0].Failure == nil || reports[0].Failure.Code != modules.FailureConfigInvalid {
		t.Fatalf("expected config failure, got %#v", reports[0].Failure)
	}
	requireCounters(t, counters, 0, 0, 0, 0)
}

func TestRuntimeNormalizesOmittedConfigBeforeModuleCallbacks(t *testing.T) {
	ctx := context.Background()
	counters := &testmodule.Counters{}
	registry := mustRegistry(t, testmodule.Registration(counters, testmodule.Options{WriteConfig: true}))
	configs := modules.NewMemoryConfigStore()
	if err := configs.Save(modules.ProjectConfig{
		SchemaVersion: 1,
		Modules: map[string]modules.ModuleConfig{
			testmodule.ID: {Enabled: true, ConfigVersion: 1},
		},
	}); err != nil {
		t.Fatal(err)
	}
	grants := modules.NewMemoryGrantStore()
	if err := grants.Save(modules.GrantSet{testmodule.ID: {"test.read"}}); err != nil {
		t.Fatal(err)
	}
	runtime := modules.NewRuntime(modules.ProjectRef{Root: t.TempDir()}, registry, configs, grants)
	report, err := runtime.Show(ctx, testmodule.ID)
	if err != nil {
		t.Fatal(err)
	}
	requireState(t, report, modules.StateEnabled)
}

func TestRuntimeInitializesModuleOnceUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	counters := &testmodule.Counters{}
	registry := mustRegistry(t, testmodule.Registration(counters, testmodule.Options{}))
	configs := modules.NewMemoryConfigStore()
	if err := configs.Save(modules.ProjectConfig{
		SchemaVersion: 1,
		Modules: map[string]modules.ModuleConfig{
			testmodule.ID: {Enabled: true, ConfigVersion: 1, Config: modules.Config{}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	grants := modules.NewMemoryGrantStore()
	if err := grants.Save(modules.GrantSet{testmodule.ID: {"test.read"}}); err != nil {
		t.Fatal(err)
	}
	runtime := modules.NewRuntime(modules.ProjectRef{Root: t.TempDir()}, registry, configs, grants)

	const callers = 8
	errs := make(chan error, callers)
	var wait sync.WaitGroup
	for range callers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := runtime.Show(ctx, testmodule.ID)
			errs <- err
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	requireCounters(t, counters, 1, 1, 1, callers)
}

func TestRuntimeDoesNotHoldGlobalLockDuringModuleInitialization(t *testing.T) {
	ctx := context.Background()
	slowStarted := make(chan struct{})
	releaseSlow := make(chan struct{})
	registrations := []modules.Registration{
		lifecycleRegistration("dev.brain.slow", func() error {
			close(slowStarted)
			<-releaseSlow
			return nil
		}),
		lifecycleRegistration("dev.brain.fast", nil),
	}
	registry := mustRegistry(t, registrations...)
	configs := modules.NewMemoryConfigStore()
	if err := configs.Save(modules.ProjectConfig{
		SchemaVersion: 1,
		Modules: map[string]modules.ModuleConfig{
			"dev.brain.slow": {Enabled: true, ConfigVersion: 1, Config: modules.Config{}},
			"dev.brain.fast": {Enabled: true, ConfigVersion: 1, Config: modules.Config{}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	runtime := modules.NewRuntime(modules.ProjectRef{Root: t.TempDir()}, registry, configs, modules.NewMemoryGrantStore())
	slowDone := make(chan error, 1)
	go func() {
		_, err := runtime.Show(ctx, "dev.brain.slow")
		slowDone <- err
	}()
	<-slowStarted

	fastDone := make(chan error, 1)
	go func() {
		_, err := runtime.Show(ctx, "dev.brain.fast")
		fastDone <- err
	}()
	select {
	case err := <-fastDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("unrelated module initialization blocked on runtime lock")
	}
	close(releaseSlow)
	if err := <-slowDone; err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeUnavailableConfigAndStaleGrant(t *testing.T) {
	ctx := context.Background()
	configs := modules.NewMemoryConfigStore()
	if err := configs.Save(modules.ProjectConfig{
		SchemaVersion: 1,
		Modules: map[string]modules.ModuleConfig{
			"dev.brain.missing": {Enabled: true, ConfigVersion: 1, Config: modules.Config{"keep": "value"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	grants := modules.NewMemoryGrantStore()
	if err := grants.Save(modules.GrantSet{"dev.brain.missing": {"old.permission"}}); err != nil {
		t.Fatal(err)
	}
	registry := mustRegistry(t)
	runtime := modules.NewRuntime(modules.ProjectRef{Root: t.TempDir()}, registry, configs, grants)

	report, err := runtime.Show(ctx, "dev.brain.missing")
	if err != nil {
		t.Fatal(err)
	}
	requireState(t, report, modules.StateUnavailable)
	if _, err := runtime.Disable(ctx, "dev.brain.missing"); err != nil {
		t.Fatal(err)
	}
	report, err = runtime.Revoke(ctx, "dev.brain.missing", "old.permission")
	if err != nil {
		t.Fatal(err)
	}
	requireState(t, report, modules.StateUnavailable)
	if len(report.Grants) != 0 {
		t.Fatalf("expected stale grants revoked, got %v", report.Grants)
	}
}

func withDescriptor(registration modules.Registration, update func(*modules.Descriptor)) modules.Registration {
	update(&registration.Descriptor)
	return registration
}

func mustRegistry(t *testing.T, registrations ...modules.Registration) *modules.Registry {
	t.Helper()
	registry, err := modules.NewRegistry(registrations)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func requireState(t *testing.T, report modules.Report, state modules.State) {
	t.Helper()
	if report.State != state {
		t.Fatalf("expected state %s, got %#v", state, report)
	}
}

func requireCounters(t *testing.T, counters *testmodule.Counters, factories, validations, initializes, health int) {
	t.Helper()
	snapshot := counters.Snapshot()
	if snapshot.Factories != factories || snapshot.Validations != validations || snapshot.Initializes != initializes || snapshot.HealthCalls != health {
		t.Fatalf("unexpected counters: got factories=%d validations=%d initializes=%d health=%d", snapshot.Factories, snapshot.Validations, snapshot.Initializes, snapshot.HealthCalls)
	}
}

func lifecycleRegistration(id string, initialize func() error) modules.Registration {
	return modules.Registration{
		Descriptor: modules.Descriptor{
			ID:            id,
			Name:          id,
			Version:       "1.0.0",
			BrainAPIMajor: modules.BrainAPIMajor,
			ConfigVersion: 1,
		},
		Factory: func() modules.Module {
			return lifecycleModule{initialize: initialize}
		},
	}
}

type lifecycleModule struct {
	initialize func() error
}

func (lifecycleModule) Validate(context.Context, modules.ModuleContext, modules.Config) error {
	return nil
}

func (m lifecycleModule) Initialize(context.Context, modules.ModuleContext, modules.Config) error {
	if m.initialize != nil {
		return m.initialize()
	}
	return nil
}

func (lifecycleModule) Health(context.Context, modules.ModuleContext) modules.Health {
	return modules.Health{Status: modules.HealthHealthy}
}
