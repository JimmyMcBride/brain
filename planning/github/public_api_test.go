package github_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"testing"
)

const canonicalGitHubImportPath = "github.com/JimmyMcBride/brain/planning/github"

func TestPublicGitHubAdapterDependencies(t *testing.T) {
	t.Parallel()
	command := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", canonicalGitHubImportPath)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("go list public GitHub adapter dependencies: %v", err)
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
		canonicalGitHubImportPath,
	}
	if runtime.GOOS == "windows" {
		want = append(want, "golang.org/x/sys/windows")
	}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("non-standard dependencies = %v, want %v", got, want)
	}
}

func TestPublicGitHubAdapterExportSurface(t *testing.T) {
	t.Parallel()
	packages, err := parser.ParseDir(token.NewFileSet(), ".", func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	publicPackage := packages["github"]
	var got []string
	for _, file := range publicPackage.Files {
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				for _, specification := range declaration.Specs {
					switch specification := specification.(type) {
					case *ast.TypeSpec:
						if specification.Name.IsExported() {
							got = append(got, specification.Name.Name)
						}
					case *ast.ValueSpec:
						for _, name := range specification.Names {
							if name.IsExported() {
								got = append(got, name.Name)
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
					receiver, ok := declaration.Recv.List[0].Type.(*ast.StarExpr)
					if !ok {
						continue
					}
					identifier, ok := receiver.X.(*ast.Ident)
					if !ok || identifier.Name != "Adapter" {
						continue
					}
					name = "Adapter." + declaration.Name.Name
				}
				got = append(got, name)
			}
		}
	}
	slices.Sort(got)
	want := []string{
		"Adapter", "Adapter.CollaborationSource", "Adapter.Enabled", "Adapter.ExecutionWorkspace",
		"Adapter.ExternalMappingRepository", "Adapter.PublicationTarget", "Adapter.RepositoryEvidenceSource",
		"Config", "InputRunner", "New", "Options", "RunResult", "Runner",
	}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("public GitHub adapter exports changed:\ngot  %v\nwant %v", got, want)
	}
}
