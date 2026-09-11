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
		"AdoptionApplyInput", "AdoptionPlan", "AdoptionPreviewInput",
		"PublicationApplyInput", "PublicationApplyResult", "Service.ApplyPublication", "PermissionPublish", "EventPublicationApplied",
		"PublicationPreviewInput", "Service.PreviewPublication",
		"CollaborationRepairInput", "CollaborationRepairResult", "EventCollaborationRepaired", "PermissionCollaboration",
		"Service.AssessCollaboration", "Service.PreviewCollaborationRepair", "Service.RepairCollaboration",
		"AdoptionResult", "ArtifactExternalReference", "ChangeRequestClosed", "ChangeRequestMerged", "ChangeRequestOpen", "ChangeRequestState",
		"Authorizer", "BrainstormChallengeInput", "BrainstormDocument", "BrainstormMutationResult", "BrainstormPreview",
		"BrainstormRefinementInput", "BrainstormResult", "BrainstormUpdateInput", "CheckFinding", "CheckInput",
		"CheckReport", "CheckReport.ErrorCount", "CheckReport.HasErrors", "CheckReport.WarningCount",
		"CollaborationAssessment", "CollaborationContribution", "CollaborationRepairEvidence", "CollaborationRepairRequest",
		"CollaborationSource", "CollaborationSourceSnapshot", "CreateBrainstormInput", "ErrArtifactConflict", "ErrConfirmationRequired",
		"ErrEventSinkRequired", "ErrWorkspaceNotReadable", "ErrWorkspaceNotWritable", "Event",
		"EventBrainstormCreated", "EventBrainstormPromoted", "EventBrainstormUpdated", "EventGuidedSessionUpdated", "EventIntegrationAdopted",
		"EventRoadmapUpdated", "EventSink", "EventSpecExecutionStarted", "EventSpecUpdated", "GuidePacket", "GuidedSessionMutationInput", "GuidedSessionRecord",
		"GuidedSessionResult", "GuidedSessionState", "ExecutionAttachRequest", "ExecutionDone", "ExecutionField", "ExecutionFieldRoleStatus",
		"ExecutionInProgress", "ExecutionInReview", "ExecutionStatus", "ExecutionStatusRequest", "ExecutionTodo", "ExecutionWorkItem",
		"ExecutionWorkspace", "ExecutionWorkspaceSnapshot", "ExternalMappingRepository", "ExternalMappingState", "ExternalReference",
		"IntegrationAdapterDisabled", "IntegrationAmbiguousIdentity", "IntegrationContractVersion", "IntegrationError", "IntegrationError.Error",
		"IntegrationError.Unwrap", "IntegrationErrorClass", "IntegrationManualRemediationRequired", "IntegrationPartialFailure",
		"IntegrationProviderUnavailable", "IntegrationRevisionConflict", "IntegrationUnauthenticated", "IntegrationUnauthorized",
		"IntegrationUnsupportedCapability", "LocalPromotionDraft", "LocalPromotionInput",
		"LocalPromotionRepairInput", "LocalPromotionRepairResult", "LocalPromotionResult", "MaturityDecision",
		"MutationAction", "MutationCreate", "MutationReuse", "MutationUnchanged", "MutationUpdate", "New", "Options",
		"PermissionBrainstorm", "PermissionExecute", "PermissionRead", "PermissionRoadmap", "PermissionSpecApprove", "PermissionSpecEdit", "PlanningChangeRequestEvidence", "ProjectStatus", "PromotionSpecDraft",
		"PromotionSpecWrite", "PublicationActionEvidence", "PublicationActionKind", "PublicationApplyAction", "PublicationArtifact",
		"PublicationArtifactAction", "PublicationInspectRequest", "PublicationPlan", "PublicationRelationship",
		"PublicationRelationshipAction", "PublicationRelationshipContains", "PublicationRelationshipDependsOn",
		"PublicationRelationshipKind", "PublicationResult", "PublicationSnapshot", "PublicationTarget", "ReconciliationEvidence", "ReconciliationPlan", "ReconciliationResult",
		"Repository", "RepositoryEvidence", "RepositoryEvidenceSource", "RoadmapDocument", "RoadmapParkingInput", "RoadmapPreview", "RoadmapResult",
		"Service", "Service.AnalyzeSpec", "Service.ApplyAdoption", "Service.AssessLocalBrainstorm", "Service.BeginSpecExecution", "Service.ChallengeBrainstorm", "Service.Check",
		"Service.CreateBrainstorm", "Service.CurrentGuidePacket", "Service.CurrentGuidedSession", "Service.GetBrainstorm",
		"Service.EditSpec", "Service.GetGuidedSession", "Service.GetSpec", "Service.GuidePacketForChain", "Service.HandoffSpec", "Service.ListBrainstorms",
		"Service.ListGuidedSessions", "Service.ListSpecs", "Service.ParkRoadmap", "Service.PreviewBrainstorm",
		"Service.PreviewBrainstormChallenge", "Service.PreviewBrainstormRefinement", "Service.PreviewBrainstormUpdate",
		"Service.PreviewAdoption", "Service.PreviewLocalPromotion", "Service.PreviewLocalPromotionRepair", "Service.PreviewRoadmap", "Service.PreviewRoadmapParking", "Service.ProjectStatus",
		"Service.PreviewSpecAnalysis", "Service.PreviewSpecChecklist", "Service.PreviewSpecEdit", "Service.PreviewSpecExecution", "Service.PreviewSpecHandoff", "Service.PreviewSpecInitiative", "Service.PreviewSpecStatus",
		"Service.PromoteLocalBrainstorm", "Service.ReadRoadmap", "Service.RefineBrainstorm", "Service.ReopenGuidedSession",
		"Service.RepairLocalPromotionSource", "Service.ReviewGuidedSession", "Service.RunSpecChecklist", "Service.SetSpecInitiative", "Service.SetSpecStatus", "Service.Status", "Service.SwitchGuidedSession",
		"Service.UpdateBrainstorm", "Service.UpdateRoadmap",
		"SpecAnalysisFinding", "SpecAnalysisReport", "SpecAnalysisReport.ErrorCount", "SpecAnalysisReport.WarningCount",
		"SpecChecklistFinding", "SpecChecklistReport", "SpecChecklistReport.ErrorCount", "SpecChecklistReport.WarningCount",
		"SpecDocument", "SpecEditInput", "SpecExecutionInput", "SpecExecutionResult", "SpecExecutionSlice", "SpecExecutionView", "SpecInitiativeInput", "SpecMutationResult", "SpecQueryDocument", "SpecStatusInput", "SpecSummary", "UpdateRoadmapInput", "WorkspaceCompatible", "WorkspaceFutureSchema",
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
