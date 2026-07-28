package modules_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"brain/internal/modules"
)

func TestFileStoresDoNotCreateFilesOnLoad(t *testing.T) {
	root := t.TempDir()
	configStore := modules.NewFileConfigStore(root)
	grantStore := modules.NewFileGrantStore(root)
	config, err := configStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.SchemaVersion != 1 || len(config.Modules) != 0 {
		t.Fatalf("unexpected default config: %#v", config)
	}
	grants, err := grantStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 0 {
		t.Fatalf("unexpected default grants: %#v", grants)
	}
	for _, path := range []string{configStore.Path(), grantStore.Path()} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s not to exist, got %v", path, err)
		}
	}
}

func TestFileConfigStoreDeterministicAndPreservesUnknownModules(t *testing.T) {
	root := t.TempDir()
	store := modules.NewFileConfigStore(root)
	config := modules.ProjectConfig{
		SchemaVersion: 1,
		Modules: map[string]modules.ModuleConfig{
			"dev.brain.zeta": {
				Enabled:       true,
				ConfigVersion: 2,
				Config:        modules.Config{"z": "last", "a": "first"},
			},
			"dev.brain.alpha": {
				Enabled:       false,
				ConfigVersion: 1,
				Config:        modules.Config{"keep": "value"},
			},
		},
	}
	if err := store.Save(config); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Modules["dev.brain.alpha"].Config["keep"] != "value" {
		t.Fatalf("unknown config was not preserved: %#v", loaded)
	}
	if err := store.Save(loaded); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("expected deterministic YAML\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if strings.Index(string(first), "dev.brain.alpha") > strings.Index(string(first), "dev.brain.zeta") {
		t.Fatalf("expected sorted module keys:\n%s", first)
	}
	assertMode(t, store.Path(), 0o644)
	assertNoTempFiles(t, filepath.Dir(store.Path()))
}

func TestFileConfigStoreRejectsRawSecrets(t *testing.T) {
	store := modules.NewFileConfigStore(t.TempDir())
	config := modules.ProjectConfig{
		SchemaVersion: 1,
		Modules: map[string]modules.ModuleConfig{
			testmoduleID: {
				ConfigVersion: 1,
				Config:        modules.Config{"token": "raw-value"},
			},
		},
	}
	if err := store.Save(config); err == nil || !strings.Contains(err.Error(), "raw secret") {
		t.Fatalf("expected raw secret rejection, got %v", err)
	}
	config.Modules[testmoduleID] = modules.ModuleConfig{
		ConfigVersion: 1,
		Config:        modules.Config{"token": "ref:TEST_TOKEN"},
	}
	if err := store.Save(config); err != nil {
		t.Fatalf("expected secret reference accepted, got %v", err)
	}
}

func TestFileGrantStoreNormalizesAndUsesPrivateMode(t *testing.T) {
	root := t.TempDir()
	store := modules.NewFileGrantStore(root)
	if err := store.Save(modules.GrantSet{
		testmoduleID: {"write", "read", "read"},
	}); err != nil {
		t.Fatal(err)
	}
	grants, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(grants[testmoduleID], ",")
	if got != "read,write" {
		t.Fatalf("expected sorted unique grants, got %q", got)
	}
	assertMode(t, store.Path(), 0o600)

	if err := store.Save(modules.GrantSet{testmoduleID: {"read"}}); err != nil {
		t.Fatal(err)
	}
	grants, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(grants[testmoduleID], ","); got != "read" {
		t.Fatalf("expected atomic replacement content, got %q", got)
	}
	assertNoTempFiles(t, filepath.Dir(store.Path()))
}

func TestRuntimeFileStoresPreserveConfigAndGrantsOnDisable(t *testing.T) {
	root := t.TempDir()
	configStore := modules.NewFileConfigStore(root)
	grantStore := modules.NewFileGrantStore(root)
	registration := testRegistration()
	registry, err := modules.NewRegistry([]modules.Registration{registration})
	if err != nil {
		t.Fatal(err)
	}
	config := modules.ProjectConfig{
		SchemaVersion: 1,
		Modules: map[string]modules.ModuleConfig{
			testmoduleID: {
				Enabled:       true,
				ConfigVersion: 1,
				Config:        modules.Config{"keep": "value"},
			},
		},
	}
	if err := configStore.Save(config); err != nil {
		t.Fatal(err)
	}
	if err := grantStore.Save(modules.GrantSet{testmoduleID: {"test.read"}}); err != nil {
		t.Fatal(err)
	}
	moduleRuntime := modules.NewRuntime(modules.ProjectRef{Root: root}, registry, configStore, grantStore)
	if _, err := moduleRuntime.Disable(t.Context(), testmoduleID); err != nil {
		t.Fatal(err)
	}
	loaded, err := configStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	entry := loaded.Modules[testmoduleID]
	if entry.Enabled || entry.Config["keep"] != "value" {
		t.Fatalf("disable did not preserve config: %#v", entry)
	}
	grants, err := grantStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(grants[testmoduleID]) != 1 || grants[testmoduleID][0] != "test.read" {
		t.Fatalf("disable did not preserve grants: %#v", grants)
	}
}

const testmoduleID = "dev.brain.store-test"

func testRegistration() modules.Registration {
	return modules.Registration{
		Descriptor: modules.Descriptor{
			ID:            testmoduleID,
			Name:          "Store Test",
			Version:       "1.0.0",
			BrainAPIMajor: modules.BrainAPIMajor,
			ConfigVersion: 1,
			Permissions:   []string{"test.read"},
		},
		Factory: func() modules.Module { return storeTestModule{} },
	}
}

type storeTestModule struct{}

func (storeTestModule) Validate(_ context.Context, _ modules.ModuleContext, _ modules.Config) error {
	return nil
}

func (storeTestModule) Initialize(_ context.Context, _ modules.ModuleContext, _ modules.Config) error {
	return nil
}

func (storeTestModule) Health(_ context.Context, _ modules.ModuleContext) modules.Health {
	return modules.Health{Status: modules.HealthHealthy}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("expected mode %o, got %o", want, got)
	}
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Fatalf("unexpected temporary file %s", entry.Name())
		}
	}
}
