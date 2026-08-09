package application_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/planning/application"
	"github.com/JimmyMcBride/brain/planning/local"
)

const canonicalApplicationImportPath = "github.com/JimmyMcBride/brain/planning/application"

func TestCanonicalApplicationAndLocalImportsCompose(t *testing.T) {
	t.Parallel()
	repository := local.New(t.TempDir())
	var _ application.Repository = repository
	if service := application.New(repository, application.Options{ModuleID: "consumer"}); service == nil {
		t.Fatal("application.New() returned nil")
	}
}

func TestPublicPlanningApplicationDependencies(t *testing.T) {
	t.Parallel()
	command := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", canonicalApplicationImportPath)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("go list public Planning application dependencies: %v", err)
	}
	var got []string
	for _, dependency := range strings.Split(string(output), "\n") {
		if dependency = strings.TrimSpace(dependency); dependency != "" {
			got = append(got, dependency)
		}
	}
	slices.Sort(got)
	want := []string{"github.com/JimmyMcBride/brain/planning", canonicalApplicationImportPath}
	if !slices.Equal(got, want) {
		t.Fatalf("non-standard dependencies = %v, want %v", got, want)
	}
}

func TestPublicPlanningApplicationExportSurface(t *testing.T) {
	t.Parallel()
	got := exportedNames(t, ".", "application")
	want := []string{
		"Authorizer", "BrainstormDocument", "BrainstormPreview", "BrainstormResult", "CheckFinding",
		"CheckInput", "CheckReport", "CheckReport.ErrorCount", "CheckReport.HasErrors", "CheckReport.WarningCount",
		"CreateBrainstormInput", "ErrArtifactConflict", "ErrConfirmationRequired",
		"ErrEventSinkRequired", "ErrWorkspaceNotReadable", "ErrWorkspaceNotWritable", "Event",
		"EventBrainstormCreated", "EventRoadmapUpdated", "EventSink", "MutationAction", "MutationCreate",
		"MutationUnchanged", "MutationUpdate", "New", "Options", "PermissionBrainstorm", "PermissionRead",
		"PermissionRoadmap", "ProjectStatus", "Repository", "RoadmapDocument", "RoadmapPreview", "RoadmapResult",
		"Service", "Service.Check", "Service.CreateBrainstorm", "Service.GetBrainstorm", "Service.GetSpec",
		"Service.ListBrainstorms", "Service.ListSpecs", "Service.PreviewBrainstorm", "Service.PreviewRoadmap",
		"Service.ProjectStatus", "Service.ReadRoadmap", "Service.Status", "Service.UpdateRoadmap",
		"SpecDocument", "SpecQueryDocument", "SpecSummary", "UpdateRoadmapInput", "WorkspaceCompatible", "WorkspaceFutureSchema",
		"WorkspaceInvalid", "WorkspaceMigrationRequired", "WorkspaceMissing", "WorkspaceState", "WorkspaceStatus",
		"WorkspaceUnsupportedLegacy", "WorkspaceUnsupportedSource",
	}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("public Planning application exports changed:\ngot  %v\nwant %v", got, want)
	}
}

func exportedNames(t *testing.T, directory, packageName string) []string {
	t.Helper()
	packages, err := parser.ParseDir(token.NewFileSet(), directory, func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	publicPackage, ok := packages[packageName]
	if !ok {
		t.Fatalf("package %s not found", packageName)
	}
	var exported []string
	for _, file := range publicPackage.Files {
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				for _, specification := range declaration.Specs {
					switch specification := specification.(type) {
					case *ast.TypeSpec:
						if specification.Name.IsExported() {
							exported = append(exported, specification.Name.Name)
						}
					case *ast.ValueSpec:
						for _, name := range specification.Names {
							if name.IsExported() {
								exported = append(exported, name.Name)
							}
						}
					}
				}
			case *ast.FuncDecl:
				if !declaration.Name.IsExported() {
					continue
				}
				name := declaration.Name.Name
				if declaration.Recv != nil {
					name = receiverName(declaration.Recv.List[0].Type) + "." + name
				}
				exported = append(exported, name)
			}
		}
	}
	slices.Sort(exported)
	return exported
}

func receiverName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.StarExpr:
		return receiverName(expression.X)
	default:
		return "unknown"
	}
}
