package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/planning/application"
)

type collaborationSourceStub struct{}

func (collaborationSourceStub) Read(context.Context, application.ExternalReference) (application.CollaborationSourceSnapshot, error) {
	return application.CollaborationSourceSnapshot{}, nil
}

func (collaborationSourceStub) Repair(context.Context, application.CollaborationRepairRequest) (application.CollaborationRepairEvidence, error) {
	return application.CollaborationRepairEvidence{}, nil
}

type publicationTargetStub struct{}

func (publicationTargetStub) Inspect(context.Context, application.PublicationInspectRequest) (application.PublicationSnapshot, error) {
	return application.PublicationSnapshot{}, nil
}

func (publicationTargetStub) Apply(context.Context, application.PublicationPlan) (application.PublicationResult, error) {
	return application.PublicationResult{}, nil
}

type repositoryEvidenceSourceStub struct{}

func (repositoryEvidenceSourceStub) Current(context.Context) (application.RepositoryEvidence, error) {
	return application.RepositoryEvidence{}, nil
}

type externalMappingRepositoryStub struct{}

func (externalMappingRepositoryStub) Load(context.Context) (application.ExternalMappingState, error) {
	return application.ExternalMappingState{}, nil
}

func (externalMappingRepositoryStub) Save(context.Context, application.ExternalMappingState, string) (application.ExternalMappingState, error) {
	return application.ExternalMappingState{}, nil
}

type executionWorkspaceStub struct{}

func (executionWorkspaceStub) Inspect(context.Context, application.ExternalReference) (application.ExecutionWorkspaceSnapshot, error) {
	return application.ExecutionWorkspaceSnapshot{}, nil
}

func (executionWorkspaceStub) Attach(context.Context, application.ExecutionAttachRequest) (application.ExecutionWorkItem, error) {
	return application.ExecutionWorkItem{}, nil
}

func (executionWorkspaceStub) ApplyStatus(context.Context, application.ExecutionStatusRequest) (application.ExecutionWorkItem, error) {
	return application.ExecutionWorkItem{}, nil
}

func TestIntegrationPortsAreCapabilitySized(t *testing.T) {
	t.Parallel()
	var _ application.CollaborationSource = collaborationSourceStub{}
	var _ application.PublicationTarget = publicationTargetStub{}
	var _ application.RepositoryEvidenceSource = repositoryEvidenceSourceStub{}
	var _ application.ExternalMappingRepository = externalMappingRepositoryStub{}
	var _ application.ExecutionWorkspace = executionWorkspaceStub{}

	ports := []struct {
		name    string
		typeOf  reflect.Type
		methods []string
	}{
		{"CollaborationSource", reflect.TypeOf((*application.CollaborationSource)(nil)).Elem(), []string{"Read", "Repair"}},
		{"PublicationTarget", reflect.TypeOf((*application.PublicationTarget)(nil)).Elem(), []string{"Apply", "Inspect"}},
		{"RepositoryEvidenceSource", reflect.TypeOf((*application.RepositoryEvidenceSource)(nil)).Elem(), []string{"Current"}},
		{"ExternalMappingRepository", reflect.TypeOf((*application.ExternalMappingRepository)(nil)).Elem(), []string{"Load", "Save"}},
		{"ExecutionWorkspace", reflect.TypeOf((*application.ExecutionWorkspace)(nil)).Elem(), []string{"ApplyStatus", "Attach", "Inspect"}},
	}
	for _, port := range ports {
		got := make([]string, 0, port.typeOf.NumMethod())
		for i := range port.typeOf.NumMethod() {
			got = append(got, port.typeOf.Method(i).Name)
		}
		if !slices.Equal(got, port.methods) {
			t.Errorf("%s methods = %v, want %v", port.name, got, port.methods)
		}
	}
}

func TestExternalReferenceJSONContract(t *testing.T) {
	t.Parallel()
	reference := application.ExternalReference{
		Provider:  "provider",
		Kind:      "work",
		OpaqueID:  "opaque",
		DisplayID: "display",
		URL:       "https://example.test/work/1",
		Revision:  "revision",
	}
	raw, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	want := []string{"display_id", "kind", "opaque_id", "provider", "revision", "url"}
	keys := make([]string, 0, len(got))
	for key := range got {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Fatalf("external reference keys = %v, want %v", keys, want)
	}
}

func TestIntegrationErrorClasses(t *testing.T) {
	t.Parallel()
	classes := []application.IntegrationErrorClass{
		application.IntegrationAdapterDisabled,
		application.IntegrationProviderUnavailable,
		application.IntegrationUnauthenticated,
		application.IntegrationUnauthorized,
		application.IntegrationRevisionConflict,
		application.IntegrationAmbiguousIdentity,
		application.IntegrationUnsupportedCapability,
		application.IntegrationPartialFailure,
		application.IntegrationManualRemediationRequired,
	}
	seen := make(map[application.IntegrationErrorClass]struct{}, len(classes))
	for _, class := range classes {
		if class == "" {
			t.Fatal("integration error class is empty")
		}
		if _, exists := seen[class]; exists {
			t.Fatalf("duplicate integration error class %q", class)
		}
		seen[class] = struct{}{}
	}

	cause := errors.New("provider failed")
	err := &application.IntegrationError{
		Class:     application.IntegrationProviderUnavailable,
		Provider:  "provider",
		Operation: "read",
		Err:       cause,
	}
	if !errors.Is(err, cause) {
		t.Fatal("IntegrationError does not preserve its cause")
	}
	if got := err.Error(); got != "provider_unavailable: provider failed" {
		t.Fatalf("IntegrationError.Error() = %q", got)
	}
}

func TestIntegrationContractsUseProviderNeutralIdentifiers(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("integration.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "integration.go", raw, 0)
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"github", "issue", "milestone", "label", "graphql", "nodeid", "ghargs"}
	ast.Inspect(file, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		name := strings.ToLower(identifier.Name)
		for _, fragment := range forbidden {
			if strings.Contains(name, fragment) {
				t.Errorf("provider-specific identifier %q in integration contract", identifier.Name)
			}
		}
		return true
	})
}
