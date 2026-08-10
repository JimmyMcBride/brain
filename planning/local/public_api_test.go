package local_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

const canonicalLocalImportPath = "github.com/JimmyMcBride/brain/planning/local"

func TestPublicPlanningLocalDependencies(t *testing.T) {
	t.Parallel()
	command := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", canonicalLocalImportPath)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("go list public Planning local dependencies: %v", err)
	}
	var got []string
	for _, dependency := range strings.Split(string(output), "\n") {
		if dependency = strings.TrimSpace(dependency); dependency != "" {
			got = append(got, dependency)
		}
	}
	slices.Sort(got)
	want := []string{
		"github.com/JimmyMcBride/brain/planning",
		"github.com/JimmyMcBride/brain/planning/application",
		canonicalLocalImportPath,
		"gopkg.in/yaml.v3",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("non-standard dependencies = %v, want %v", got, want)
	}
}

func TestPublicPlanningLocalExportSurface(t *testing.T) {
	t.Parallel()
	packages, err := parser.ParseDir(token.NewFileSet(), ".", func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	publicPackage := packages["local"]
	var got []string
	for _, file := range publicPackage.Files {
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				for _, specification := range declaration.Specs {
					typeSpec, ok := specification.(*ast.TypeSpec)
					if ok && typeSpec.Name.IsExported() {
						got = append(got, typeSpec.Name.Name)
					}
				}
			case *ast.FuncDecl:
				if !declaration.Name.IsExported() {
					continue
				}
				name := declaration.Name.Name
				if declaration.Recv != nil {
					name = "Adapter." + declaration.Name.Name
				}
				got = append(got, name)
			}
		}
	}
	slices.Sort(got)
	want := []string{
		"Adapter", "Adapter.CreateBrainstorm", "Adapter.FindBrainstorm", "Adapter.FindSpec", "Adapter.GetBrainstorm",
		"Adapter.GetSpec", "Adapter.ListBrainstorms", "Adapter.ListSpecs", "Adapter.QuerySpecs",
		"Adapter.ReadGuidedSessions", "Adapter.ReadRoadmap", "Adapter.ReplaceBrainstorm",
		"Adapter.ReplaceGuidedSessions", "Adapter.ReplaceRoadmap", "Adapter.RollbackBrainstormCreation",
		"Adapter.ReplaceSpec", "Adapter.RollbackSpecReplacement", "Adapter.Status", "Adapter.WritePromotionSpecs", "New",
	}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("public Planning local exports changed:\ngot  %v\nwant %v", got, want)
	}
}
