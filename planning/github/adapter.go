package github

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"

	"github.com/JimmyMcBride/brain/planning/application"
)

const providerName = "github"

// Config controls the optional GitHub adapter independently from Planning
// artifact ownership.
type Config struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// RunResult contains one gh process result. Provider-specific interpretation
// remains inside this package.
type RunResult struct {
	Stdout []byte
	Stderr []byte
}

// Runner executes gh with caller cancellation and an explicit project root.
// Tests inject a fake runner; the adapter stores no credentials.
type Runner interface {
	Run(ctx context.Context, projectRoot string, args ...string) (RunResult, error)
}

// InputRunner supports provider requests whose content must not travel through
// command arguments. Publication requires this capability for complete spec bodies.
type InputRunner interface {
	Runner
	RunInput(ctx context.Context, projectRoot string, input []byte, args ...string) (RunResult, error)
}

// Options supplies adapter dependencies.
type Options struct {
	ProjectRoot string
	Runner      Runner
	GitRunner   Runner
}

// Adapter implements provider-neutral Planning ports through GitHub.
type Adapter struct {
	enabled     bool
	projectRoot string
	runner      Runner
	gitRunner   Runner
	state       stateStore
}

// New constructs an adapter without performing preflight, filesystem, process,
// credential, or network work.
func New(config Config, options Options) *Adapter {
	root := filepath.Clean(options.ProjectRoot)
	if options.ProjectRoot == "" {
		root = "."
	}
	runner := options.Runner
	if runner == nil {
		runner = commandRunner{}
	}
	gitRunner := options.GitRunner
	if gitRunner == nil {
		gitRunner = commandRunnerWithExecutable{executable: "git"}
	}
	return &Adapter{
		enabled:     config.Enabled,
		projectRoot: root,
		runner:      runner,
		gitRunner:   gitRunner,
		state:       newFileStateStore(root),
	}
}

// Enabled reports whether provider operations are allowed.
func (a *Adapter) Enabled() bool {
	return a != nil && a.enabled
}

// CollaborationSource returns the collaboration capability.
func (a *Adapter) CollaborationSource() application.CollaborationSource {
	return collaborationSource{adapter: a}
}

// PublicationTarget returns the publication capability.
func (a *Adapter) PublicationTarget() application.PublicationTarget {
	return publicationTarget{adapter: a}
}

// RepositoryEvidenceSource returns the repository-evidence capability.
func (a *Adapter) RepositoryEvidenceSource() application.RepositoryEvidenceSource {
	return repositoryEvidenceSource{adapter: a}
}

// ExternalMappingRepository returns the durable-mapping capability.
func (a *Adapter) ExternalMappingRepository() application.ExternalMappingRepository {
	return externalMappingRepository{adapter: a}
}

// ExecutionWorkspace returns the execution-workspace capability.
func (a *Adapter) ExecutionWorkspace() application.ExecutionWorkspace {
	return executionWorkspace{adapter: a}
}

func (a *Adapter) unavailable(operation string) error {
	class := application.IntegrationUnsupportedCapability
	message := "GitHub adapter capability is not implemented"
	if !a.Enabled() {
		class = application.IntegrationAdapterDisabled
		message = "GitHub adapter is disabled"
	}
	return &application.IntegrationError{
		Class:     class,
		Provider:  providerName,
		Operation: operation,
		Message:   message,
	}
}

type collaborationSource struct{ adapter *Adapter }

func (p collaborationSource) Read(ctx context.Context, ref application.ExternalReference) (application.CollaborationSourceSnapshot, error) {
	return p.adapter.readDiscussion(ctx, ref)
}

func (p collaborationSource) Repair(ctx context.Context, request application.CollaborationRepairRequest) (application.CollaborationRepairEvidence, error) {
	return p.adapter.repairDiscussion(ctx, request)
}

type publicationTarget struct{ adapter *Adapter }

func (p publicationTarget) Inspect(ctx context.Context, request application.PublicationInspectRequest) (application.PublicationSnapshot, error) {
	return p.adapter.inspectPublication(ctx, request)
}

func (p publicationTarget) Apply(ctx context.Context, plan application.PublicationPlan) (application.PublicationResult, error) {
	return p.adapter.applyPublication(ctx, plan)
}

type repositoryEvidenceSource struct{ adapter *Adapter }

func (p repositoryEvidenceSource) Current(ctx context.Context) (application.RepositoryEvidence, error) {
	return p.adapter.currentRepositoryEvidence(ctx)
}

type externalMappingRepository struct{ adapter *Adapter }

func (p externalMappingRepository) Load(ctx context.Context) (application.ExternalMappingState, error) {
	return p.adapter.loadExternalMappings(ctx)
}

func (p externalMappingRepository) Save(ctx context.Context, state application.ExternalMappingState, expectedRevision string) (application.ExternalMappingState, error) {
	return p.adapter.saveExternalMappings(ctx, state, expectedRevision)
}

type executionWorkspace struct{ adapter *Adapter }

func (p executionWorkspace) Inspect(ctx context.Context, ref application.ExternalReference) (application.ExecutionWorkspaceSnapshot, error) {
	return p.adapter.inspectExecutionWorkspace(ctx, ref)
}

func (p executionWorkspace) Attach(ctx context.Context, request application.ExecutionAttachRequest) (application.ExecutionWorkItem, error) {
	return p.adapter.attachExecutionWork(ctx, request)
}

func (p executionWorkspace) ApplyStatus(ctx context.Context, request application.ExecutionStatusRequest) (application.ExecutionWorkItem, error) {
	return p.adapter.applyExecutionStatus(ctx, request)
}

type commandRunner struct{}

func (commandRunner) Run(ctx context.Context, projectRoot string, args ...string) (RunResult, error) {
	return commandRunnerWithExecutable{executable: "gh"}.Run(ctx, projectRoot, args...)
}

func (commandRunner) RunInput(ctx context.Context, projectRoot string, input []byte, args ...string) (RunResult, error) {
	return commandRunnerWithExecutable{executable: "gh"}.RunInput(ctx, projectRoot, input, args...)
}

type commandRunnerWithExecutable struct {
	executable  string
	environment []string
}

func (r commandRunnerWithExecutable) Run(ctx context.Context, projectRoot string, args ...string) (RunResult, error) {
	return r.RunInput(ctx, projectRoot, nil, args...)
}

func (r commandRunnerWithExecutable) RunInput(ctx context.Context, projectRoot string, input []byte, args ...string) (RunResult, error) {
	command := exec.CommandContext(ctx, r.executable, args...)
	command.Dir = projectRoot
	command.Stdin = bytes.NewReader(input)
	if r.environment != nil {
		command.Env = r.environment
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return RunResult{
		Stdout: append([]byte(nil), stdout.Bytes()...),
		Stderr: append([]byte(nil), stderr.Bytes()...),
	}, err
}
