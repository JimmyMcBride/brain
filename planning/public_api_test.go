package planning_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/JimmyMcBride/brain/planning"
)

const canonicalImportPath = "github.com/JimmyMcBride/brain/planning"

func TestCanonicalImportExposesPlanningDomain(t *testing.T) {
	t.Parallel()

	spec := planning.Spec{
		ID:           "public-domain-boundary",
		Title:        "Public Domain Boundary",
		Status:       planning.SpecApproved,
		Approval:     planning.Approval{State: planning.ApprovalApproved},
		Verification: []string{"go test ./..."},
	}
	if findings := planning.ValidateSpec(spec); len(findings) != 0 {
		t.Fatalf("ValidateSpec() findings = %#v", findings)
	}

	order, err := planning.OrderSpecs([]planning.Spec{spec})
	if err != nil {
		t.Fatal(err)
	}
	if want := []planning.ArtifactID{spec.ID}; !slices.Equal(order, want) {
		t.Fatalf("OrderSpecs() = %v, want %v", order, want)
	}
}

func TestPublicPlanningDomainHasOnlyStandardLibraryDependencies(t *testing.T) {
	t.Parallel()

	command := exec.Command(
		"go",
		"list",
		"-deps",
		"-f",
		"{{if not .Standard}}{{.ImportPath}}{{end}}",
		canonicalImportPath,
	)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("go list public Planning dependencies: %v", err)
	}

	var nonStandard []string
	for _, dependency := range strings.Split(string(output), "\n") {
		if dependency = strings.TrimSpace(dependency); dependency != "" {
			nonStandard = append(nonStandard, dependency)
		}
	}
	if want := []string{canonicalImportPath}; !slices.Equal(nonStandard, want) {
		t.Fatalf("non-standard dependencies = %v, want %v", nonStandard, want)
	}
}

func TestPublicPlanningDomainHasNoReplaceDirective(t *testing.T) {
	t.Parallel()

	output, err := exec.Command("go", "mod", "edit", "-json").Output()
	if err != nil {
		t.Fatalf("inspect go.mod: %v", err)
	}
	var module struct {
		Replace []json.RawMessage
	}
	if err := json.Unmarshal(output, &module); err != nil {
		t.Fatalf("decode go.mod: %v", err)
	}
	if len(module.Replace) != 0 {
		t.Fatalf("public Planning consumer boundary requires no replace directives: %s", output)
	}
}

func TestPublicPlanningExportSurface(t *testing.T) {
	t.Parallel()

	packages, err := parser.ParseDir(token.NewFileSet(), ".", func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	publicPackage, ok := packages["planning"]
	if !ok {
		t.Fatal("planning package not found")
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

	want := []string{
		"Approval", "ApprovalApproved", "ApprovalPending", "ApprovalRejected", "ApprovalState",
		"ApproveSpec", "ArtifactBrainstorm", "ArtifactID", "ArtifactID.Validate", "ArtifactInitiative",
		"ArtifactKind", "ArtifactRef", "ArtifactRef.Validate", "ArtifactRoadmap", "ArtifactSpec",
		"BeginExecution", "Brainstorm", "BuildQueue", "CompleteExecution", "CompleteSlice",
		"DomainError", "DomainError.Error", "ErrApprovalRequired", "ErrDependencyCycle", "ErrDependencyMissing",
		"ErrInvalidArtifact", "ErrInvalidExecution", "ErrInvalidIdentifier", "ErrInvalidOwnership", "ErrInvalidSlice",
		"ErrInvalidTransition", "ErrNotReady", "ErrorCode", "ErrorCodeOf", "EvaluateReadiness",
		"Evidence", "ExecutionActive", "ExecutionComplete", "ExecutionInput", "ExecutionPlan", "ExecutionState",
		"Finding", "FindingSeverity", "Initiative", "OrderSpecs", "OwnershipGitHub", "OwnershipHybrid",
		"OwnershipLocal", "OwnershipMode", "OwnershipMode.Validate", "Queue", "QueueBlocked", "QueueCurrent",
		"QueueDone", "QueueEntry", "QueueReady", "QueueState", "Readiness", "ReadinessBlocked",
		"ReadinessClarifying", "ReadinessDone", "ReadinessNeedsRefinement", "ReadinessReady", "ReadinessState",
		"ReopenSpec", "Roadmap", "RoadmapEntry", "RuntimeSlice", "SeverityError", "SeverityInfo",
		"SeverityWarning", "SliceActive", "SliceCandidate", "SliceDone", "SlicePending", "SliceState",
		"SourceReference", "Spec", "SpecApproved", "SpecDone", "SpecDraft", "SpecImplementing", "SpecStatus",
		"ValidateBrainstorm", "ValidateInitiative", "ValidateRoadmap", "ValidateSpec",
	}
	slices.Sort(want)
	if !slices.Equal(exported, want) {
		t.Fatalf("public Planning exports changed:\ngot  %v\nwant %v", exported, want)
	}
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
