package modules

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Config map[string]any

type ModuleContext struct {
	Project ProjectRef
}

type ProjectRef struct {
	Root string `json:"root"`
}

type Module interface {
	Validate(context.Context, ModuleContext, Config) error
	Initialize(context.Context, ModuleContext, Config) error
	Health(context.Context, ModuleContext) Health
}

type ModuleConfig struct {
	Enabled       bool   `yaml:"enabled" json:"enabled"`
	ConfigVersion int    `yaml:"config_version" json:"config_version"`
	Config        Config `yaml:"config" json:"config"`
}

type ProjectConfig struct {
	SchemaVersion int                     `yaml:"schema_version" json:"schema_version"`
	Modules       map[string]ModuleConfig `yaml:"modules" json:"modules"`
}

type GrantSet map[string][]string

type ConfigStore interface {
	Load() (ProjectConfig, error)
	Save(ProjectConfig) error
}

type GrantStore interface {
	Load() (GrantSet, error)
	Save(GrantSet) error
}

type State string

const (
	StateAvailable   State = "available"
	StateDisabled    State = "disabled"
	StateEnabled     State = "enabled"
	StateBlocked     State = "blocked"
	StateUnavailable State = "unavailable"
)

type FailureCode string

const (
	FailureModuleUnavailable FailureCode = "module_unavailable"
	FailureAPIIncompatible   FailureCode = "api_incompatible"
	FailureConfigInvalid     FailureCode = "config_invalid"
	FailurePermissionMissing FailureCode = "permission_missing"
	FailureInitializeFailed  FailureCode = "initialize_failed"
)

type Failure struct {
	Code    FailureCode `json:"code"`
	Message string      `json:"message"`
	Command string      `json:"command,omitempty"`
}

func (f *Failure) Error() string {
	if f == nil {
		return ""
	}
	if f.Command != "" {
		return fmt.Sprintf("%s: %s; run `%s`", f.Code, f.Message, f.Command)
	}
	return fmt.Sprintf("%s: %s", f.Code, f.Message)
}

type Report struct {
	ID             string   `json:"id"`
	Name           string   `json:"name,omitempty"`
	Version        string   `json:"version,omitempty"`
	BrainAPIMajor  int      `json:"brain_api_major,omitempty"`
	ConfigVersion  int      `json:"config_version,omitempty"`
	Capabilities   []string `json:"capabilities"`
	Permissions    []string `json:"permissions"`
	Grants         []string `json:"grants"`
	DesiredEnabled bool     `json:"desired_enabled"`
	State          State    `json:"state"`
	Health         *Health  `json:"health,omitempty"`
	Failure        *Failure `json:"failure,omitempty"`
}

type Runtime struct {
	project     ProjectRef
	registry    *Registry
	configStore ConfigStore
	grantStore  GrantStore

	mu              sync.Mutex
	initializations map[string]*moduleInitialization
}

type moduleInitialization struct {
	once    sync.Once
	module  Module
	failure *Failure
}

func NewRuntime(project ProjectRef, registry *Registry, configStore ConfigStore, grantStore GrantStore) *Runtime {
	if registry == nil {
		registry, _ = NewRegistry(nil)
	}
	if configStore == nil {
		configStore = NewMemoryConfigStore()
	}
	if grantStore == nil {
		grantStore = NewMemoryGrantStore()
	}
	return &Runtime{
		project:         project,
		registry:        registry,
		configStore:     configStore,
		grantStore:      grantStore,
		initializations: map[string]*moduleInitialization{},
	}
}

func (r *Runtime) List(ctx context.Context) ([]Report, error) {
	config, grants, err := r.load()
	if err != nil {
		return nil, err
	}
	ids := r.registry.IDs()
	known := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		known[id] = struct{}{}
	}
	for id := range config.Modules {
		if _, exists := known[id]; !exists {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	reports := make([]Report, 0, len(ids))
	for _, id := range ids {
		report, err := r.evaluate(ctx, id, config, grants, false)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (r *Runtime) Show(ctx context.Context, id string) (Report, error) {
	config, grants, err := r.load()
	if err != nil {
		return Report{}, err
	}
	return r.evaluate(ctx, id, config, grants, true)
}

func (r *Runtime) Health(ctx context.Context, id string) ([]Report, error) {
	config, grants, err := r.load()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) != "" {
		report, err := r.evaluate(ctx, id, config, grants, true)
		if err != nil {
			return nil, err
		}
		return []Report{report}, nil
	}
	ids := r.registry.IDs()
	reports := make([]Report, 0, len(ids))
	for _, moduleID := range ids {
		entry, exists := config.Modules[moduleID]
		if !exists || !entry.Enabled {
			continue
		}
		report, err := r.evaluate(ctx, moduleID, config, grants, true)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (r *Runtime) Enable(ctx context.Context, id string) (Report, error) {
	config, grants, err := r.load()
	if err != nil {
		return Report{}, err
	}
	registration, exists := r.registry.Lookup(id)
	if !exists {
		return Report{}, unavailableFailure(id)
	}
	entry, exists := config.Modules[id]
	if !exists {
		entry = ModuleConfig{ConfigVersion: registration.Descriptor.ConfigVersion, Config: Config{}}
	}
	if entry.ConfigVersion != registration.Descriptor.ConfigVersion {
		return Report{}, configFailure(id, fmt.Sprintf("config version %d does not match required version %d", entry.ConfigVersion, registration.Descriptor.ConfigVersion))
	}
	if entry.Config == nil {
		entry.Config = Config{}
	}
	if failure := permissionFailure(registration.Descriptor, grants[id]); failure != nil {
		return Report{}, failure
	}
	module := registration.Factory()
	if module == nil {
		return Report{}, configFailure(id, "factory returned a nil module")
	}
	moduleContext := ModuleContext{Project: r.project}
	if err := module.Validate(ctx, moduleContext, entry.Config); err != nil {
		return Report{}, configFailure(id, err.Error())
	}
	entry.Enabled = true
	config.Modules[id] = entry
	if err := r.configStore.Save(config); err != nil {
		return Report{}, err
	}
	report, err := r.evaluate(ctx, id, config, grants, true)
	if err != nil {
		return Report{}, err
	}
	if report.Failure != nil {
		return report, report.Failure
	}
	return report, nil
}

func (r *Runtime) Disable(ctx context.Context, id string) (Report, error) {
	config, grants, err := r.load()
	if err != nil {
		return Report{}, err
	}
	entry, configured := config.Modules[id]
	registration, available := r.registry.Lookup(id)
	if !configured {
		if !available {
			return Report{}, unavailableFailure(id)
		}
		entry = ModuleConfig{ConfigVersion: registration.Descriptor.ConfigVersion, Config: Config{}}
	}
	entry.Enabled = false
	config.Modules[id] = entry
	if err := r.configStore.Save(config); err != nil {
		return Report{}, err
	}
	return r.evaluate(ctx, id, config, grants, false)
}

func (r *Runtime) Grant(ctx context.Context, id string, permissions ...string) (Report, error) {
	config, grants, err := r.load()
	if err != nil {
		return Report{}, err
	}
	registration, exists := r.registry.Lookup(id)
	if !exists {
		return Report{}, unavailableFailure(id)
	}
	declared := stringSet(registration.Descriptor.Permissions)
	current := stringSet(grants[id])
	for _, permission := range permissions {
		if _, ok := declared[permission]; !ok {
			return Report{}, fmt.Errorf("module %s does not declare permission %q", id, permission)
		}
		current[permission] = struct{}{}
	}
	grants[id] = setStrings(current)
	if err := r.grantStore.Save(grants); err != nil {
		return Report{}, err
	}
	return r.evaluate(ctx, id, config, grants, true)
}

func (r *Runtime) Revoke(ctx context.Context, id string, permissions ...string) (Report, error) {
	config, grants, err := r.load()
	if err != nil {
		return Report{}, err
	}
	current := stringSet(grants[id])
	for _, permission := range permissions {
		delete(current, permission)
	}
	if len(current) == 0 {
		delete(grants, id)
	} else {
		grants[id] = setStrings(current)
	}
	if err := r.grantStore.Save(grants); err != nil {
		return Report{}, err
	}
	return r.evaluate(ctx, id, config, grants, true)
}

func (r *Runtime) load() (ProjectConfig, GrantSet, error) {
	config, err := r.configStore.Load()
	if err != nil {
		return ProjectConfig{}, nil, err
	}
	if config.SchemaVersion == 0 {
		config.SchemaVersion = 1
	}
	if config.Modules == nil {
		config.Modules = map[string]ModuleConfig{}
	}
	grants, err := r.grantStore.Load()
	if err != nil {
		return ProjectConfig{}, nil, err
	}
	if grants == nil {
		grants = GrantSet{}
	}
	return config, grants, nil
}

func (r *Runtime) evaluate(ctx context.Context, id string, config ProjectConfig, grants GrantSet, includeHealth bool) (Report, error) {
	entry, configured := config.Modules[id]
	registration, available := r.registry.Lookup(id)
	if !available {
		if !configured {
			return Report{}, unavailableFailure(id)
		}
		return Report{
			ID:             id,
			ConfigVersion:  entry.ConfigVersion,
			DesiredEnabled: entry.Enabled,
			State:          StateUnavailable,
			Capabilities:   []string{},
			Permissions:    []string{},
			Grants:         normalizedStrings(grants[id]),
			Failure:        unavailableFailure(id),
		}, nil
	}
	descriptor := registration.Descriptor
	report := Report{
		ID:             descriptor.ID,
		Name:           descriptor.Name,
		Version:        descriptor.Version,
		BrainAPIMajor:  descriptor.BrainAPIMajor,
		ConfigVersion:  descriptor.ConfigVersion,
		Capabilities:   append([]string(nil), descriptor.Capabilities...),
		Permissions:    append([]string(nil), descriptor.Permissions...),
		Grants:         normalizedStrings(grants[id]),
		DesiredEnabled: configured && entry.Enabled,
		State:          StateAvailable,
	}
	if !configured {
		return report, nil
	}
	report.ConfigVersion = entry.ConfigVersion
	if !entry.Enabled {
		report.State = StateDisabled
		return report, nil
	}
	if entry.ConfigVersion != descriptor.ConfigVersion {
		report.State = StateBlocked
		report.Failure = configFailure(id, fmt.Sprintf("config version %d does not match required version %d", entry.ConfigVersion, descriptor.ConfigVersion))
		return report, nil
	}
	if entry.Config == nil {
		entry.Config = Config{}
	}
	if failure := permissionFailure(descriptor, grants[id]); failure != nil {
		report.State = StateBlocked
		report.Failure = failure
		return report, nil
	}
	module, failure := r.initialize(ctx, registration, entry.Config)
	if failure != nil {
		report.State = StateBlocked
		report.Failure = failure
		return report, nil
	}
	report.State = StateEnabled
	if includeHealth {
		health := module.Health(ctx, ModuleContext{Project: r.project})
		if !health.valid() {
			health = Health{Status: HealthUnhealthy, Message: "module returned an invalid health status"}
		}
		report.Health = &health
	}
	return report, nil
}

func (r *Runtime) initialize(ctx context.Context, registration Registration, config Config) (Module, *Failure) {
	id := registration.Descriptor.ID
	r.mu.Lock()
	initialization := r.initializations[id]
	if initialization == nil {
		initialization = &moduleInitialization{}
		r.initializations[id] = initialization
	}
	r.mu.Unlock()

	initialization.once.Do(func() {
		module := registration.Factory()
		if module == nil {
			initialization.failure = configFailure(id, "factory returned a nil module")
			return
		}
		moduleContext := ModuleContext{Project: r.project}
		if err := module.Validate(ctx, moduleContext, config); err != nil {
			initialization.failure = configFailure(id, err.Error())
			return
		}
		if err := module.Initialize(ctx, moduleContext, config); err != nil {
			initialization.failure = &Failure{Code: FailureInitializeFailed, Message: fmt.Sprintf("module %s initialization failed: %v", id, err)}
			return
		}
		initialization.module = module
	})
	return initialization.module, initialization.failure
}

func permissionFailure(descriptor Descriptor, grants []string) *Failure {
	granted := stringSet(grants)
	var missing []string
	for _, permission := range descriptor.Permissions {
		if _, exists := granted[permission]; !exists {
			missing = append(missing, permission)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	command := "brain modules grant " + descriptor.ID + " " + strings.Join(missing, " ")
	return &Failure{
		Code:    FailurePermissionMissing,
		Message: fmt.Sprintf("module %s is missing permissions: %s", descriptor.ID, strings.Join(missing, ", ")),
		Command: command,
	}
}

func unavailableFailure(id string) *Failure {
	return &Failure{Code: FailureModuleUnavailable, Message: fmt.Sprintf("module %s is not compiled into this binary", id)}
}

func configFailure(id, message string) *Failure {
	return &Failure{Code: FailureConfigInvalid, Message: fmt.Sprintf("module %s config is invalid: %s", id, message)}
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func setStrings(set map[string]struct{}) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

type MemoryConfigStore struct {
	mu     sync.Mutex
	config ProjectConfig
}

func NewMemoryConfigStore() *MemoryConfigStore {
	return &MemoryConfigStore{config: ProjectConfig{SchemaVersion: 1, Modules: map[string]ModuleConfig{}}}
}

func (s *MemoryConfigStore) Load() (ProjectConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneProjectConfig(s.config), nil
}

func (s *MemoryConfigStore) Save(config ProjectConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cloneProjectConfig(config)
	return nil
}

type MemoryGrantStore struct {
	mu     sync.Mutex
	grants GrantSet
}

func NewMemoryGrantStore() *MemoryGrantStore {
	return &MemoryGrantStore{grants: GrantSet{}}
}

func (s *MemoryGrantStore) Load() (GrantSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneGrants(s.grants), nil
}

func (s *MemoryGrantStore) Save(grants GrantSet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.grants = cloneGrants(grants)
	return nil
}

func cloneProjectConfig(config ProjectConfig) ProjectConfig {
	out := ProjectConfig{SchemaVersion: config.SchemaVersion, Modules: make(map[string]ModuleConfig, len(config.Modules))}
	for id, entry := range config.Modules {
		entry.Config = cloneConfig(entry.Config)
		out.Modules[id] = entry
	}
	return out
}

func cloneConfig(config Config) Config {
	if config == nil {
		return nil
	}
	out := make(Config, len(config))
	for key, value := range config {
		out[key] = value
	}
	return out
}

func cloneGrants(grants GrantSet) GrantSet {
	out := make(GrantSet, len(grants))
	for id, permissions := range grants {
		out[id] = append([]string(nil), permissions...)
	}
	return out
}
