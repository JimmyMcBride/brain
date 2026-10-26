package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/JimmyMcBride/brain/internal/app"
	"github.com/JimmyMcBride/brain/internal/modules"
	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"
	planninggithub "github.com/JimmyMcBride/brain/planning/github"

	"github.com/spf13/cobra"
)

type planningIntegrationProvider interface {
	planningServiceProvider
	GitHubAdapter() (*planninggithub.Adapter, bool)
	GitHubPreflight(context.Context) (planninggithub.RepositoryStatus, error)
	GitHubConfig(bool) modules.Config
}

type planningIntegrationRun func(*app.App, *application.Service, planningIntegrationProvider, *planninggithub.Adapter, planningAuthorizer, string) error

func withPlanningIntegration(cmd *cobra.Command, loadApp appLoader, permission string, requireEnabled bool, run planningIntegrationRun) error {
	appCtx, err := loadApp()
	if err != nil {
		return err
	}
	defer appCtx.Close()
	resolution, err := appCtx.Modules.ResolveCommand(cmd.Context(), planningCommandGroup, permission)
	if err != nil {
		return err
	}
	provider, ok := resolution.Module.(planningIntegrationProvider)
	if !ok || provider.PlanningService() == nil {
		return fmt.Errorf("module %s does not provide GitHub Planning services", resolution.ModuleID)
	}
	adapter, enabled := provider.GitHubAdapter()
	if requireEnabled && (!enabled || adapter == nil) {
		return &application.IntegrationError{Class: application.IntegrationAdapterDisabled, Provider: "github", Operation: "planning.command", Message: "GitHub adapter is disabled"}
	}
	return run(appCtx, provider.PlanningService(), provider, adapter, planningAuthorizer{runtime: appCtx.Modules, moduleID: resolution.ModuleID}, resolution.ModuleID)
}

type planningGitHubConfigOutput struct {
	Action        application.MutationAction     `json:"action"`
	Enabled       bool                           `json:"enabled"`
	Repository    *application.ExternalReference `json:"repository,omitempty"`
	DefaultBranch string                         `json:"default_branch,omitempty"`
	StatePath     string                         `json:"state_path,omitempty"`
	Event         *application.Event             `json:"event,omitempty"`
}

func addPlanningGitHubCommands(planCmd *cobra.Command, loadApp appLoader) {
	githubCmd := &cobra.Command{Use: "github", Short: "Use the optional GitHub Planning adapter"}
	var confirmEnable bool
	enableCmd := &cobra.Command{
		Use: "enable", Short: "Preview or enable GitHub Planning", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return configurePlanningGitHub(cmd, loadApp, true, confirmEnable)
		},
	}
	enableCmd.Flags().BoolVar(&confirmEnable, "confirm", false, "confirm GitHub adapter enablement")
	var confirmDisable bool
	disableCmd := &cobra.Command{
		Use: "disable", Short: "Preview or disable GitHub Planning", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return configurePlanningGitHub(cmd, loadApp, false, confirmDisable)
		},
	}
	disableCmd.Flags().BoolVar(&confirmDisable, "confirm", false, "confirm GitHub adapter disablement")

	var adoptDiscussion, adoptIssues, adoptWorkspace string
	var confirmAdopt bool
	adoptCmd := &cobra.Command{
		Use: "adopt", Short: "Preview or adopt existing GitHub planning issues", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPlanningGitHubAdopt(cmd, loadApp, adoptDiscussion, adoptIssues, adoptWorkspace, confirmAdopt)
		},
	}
	adoptCmd.Flags().StringVar(&adoptDiscussion, "discussion", "", "GitHub Discussion number or URL")
	adoptCmd.Flags().StringVar(&adoptIssues, "issues", "", "comma-separated issue numbers in canonical artifact order")
	adoptCmd.Flags().StringVar(&adoptWorkspace, "workspace", "", "GitHub workspace decision: create, connect, or skip")
	adoptCmd.Flags().BoolVar(&confirmAdopt, "confirm", false, "confirm adoption and mapping writes")

	var confirmReconcile, updateVisible bool
	reconcileCmd := &cobra.Command{
		Use: "reconcile", Short: "Preview or reconcile GitHub planning state", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPlanningGitHubReconcile(cmd, loadApp, updateVisible, confirmReconcile)
		},
	}
	reconcileCmd.Flags().BoolVar(&updateVisible, "update-visible", false, "include GitHub Project membership and semantic status")
	reconcileCmd.Flags().BoolVar(&confirmReconcile, "confirm", false, "confirm reconciliation writes")

	projectCmd := &cobra.Command{Use: "project", Short: "Inspect or update the configured GitHub Project"}
	var statusIssue, statusValue string
	var confirmStatus bool
	statusCmd := &cobra.Command{
		Use: "status", Short: "Preview or update one tracked Project item status", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPlanningGitHubProjectStatus(cmd, loadApp, statusIssue, statusValue, confirmStatus)
		},
	}
	statusCmd.Flags().StringVar(&statusIssue, "issue", "", "tracked GitHub issue number")
	statusCmd.Flags().StringVar(&statusValue, "set", "", "semantic status: todo, in-progress, in-review, or done")
	statusCmd.Flags().BoolVar(&confirmStatus, "confirm", false, "confirm the status mutation")
	projectCmd.AddCommand(statusCmd)
	githubCmd.AddCommand(enableCmd, disableCmd, adoptCmd, reconcileCmd, projectCmd)
	planCmd.AddCommand(githubCmd)
}

func configurePlanningGitHub(cmd *cobra.Command, loadApp appLoader, enabled, confirmed bool) error {
	return withPlanningIntegration(cmd, loadApp, application.PermissionPublish, false, func(appCtx *app.App, _ *application.Service, provider planningIntegrationProvider, adapter *planninggithub.Adapter, authorizer planningAuthorizer, moduleID string) error {
		_, current := provider.GitHubAdapter()
		action := application.MutationUpdate
		if current == enabled {
			action = application.MutationUnchanged
		}
		output := planningGitHubConfigOutput{Action: action, Enabled: enabled}
		if !confirmed {
			return appCtx.Output.Print(output, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "%s\tgithub enabled=%t\nPreview only. Rerun with --confirm to write.\n", action, enabled)
				return err
			})
		}
		if err := authorizer.Require(cmd.Context(), application.PermissionPublish); err != nil {
			return err
		}
		if action == application.MutationUnchanged {
			if enabled {
				status, err := adapter.Enable(cmd.Context())
				if err != nil {
					return err
				}
				output.Repository = &status.Repository
				output.DefaultBranch = status.DefaultBranch
				output.StatePath = status.StatePath
			}
			return appCtx.Output.Print(output, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "%s\tgithub enabled=%t\n", action, enabled)
				return err
			})
		}
		var status planninggithub.RepositoryStatus
		var err error
		if enabled {
			status, err = provider.GitHubPreflight(cmd.Context())
			if err != nil {
				return err
			}
		}
		if _, err := appCtx.Modules.Configure(cmd.Context(), moduleID, provider.GitHubConfig(enabled)); err != nil {
			return err
		}
		if enabled {
			resolution, err := appCtx.Modules.ResolveCommand(cmd.Context(), planningCommandGroup, application.PermissionPublish)
			if err != nil {
				return err
			}
			fresh := resolution.Module.(planningIntegrationProvider)
			adapter, _ = fresh.GitHubAdapter()
			status, err = adapter.Enable(cmd.Context())
			if err != nil {
				_, _ = appCtx.Modules.Configure(context.WithoutCancel(cmd.Context()), moduleID, provider.GitHubConfig(false))
				return err
			}
			output.Repository = &status.Repository
			output.DefaultBranch = status.DefaultBranch
			output.StatePath = status.StatePath
		}
		event := application.Event{Name: application.EventIntegrationConfigured, ModuleID: moduleID, Outcome: action, OccurredAt: time.Now().UTC()}
		if output.Repository != nil {
			event.Source = output.Repository
		}
		output.Event = &event
		if err := (planningEventSink{history: appCtx.History}).Publish(context.WithoutCancel(cmd.Context()), event); err != nil {
			return &application.IntegrationError{Class: application.IntegrationPartialFailure, Provider: "github", Operation: "integration.configure.audit", Message: "GitHub configuration changed but audit persistence failed", Err: err}
		}
		return appCtx.Output.Print(output, func(w io.Writer) error {
			if output.Repository != nil {
				_, err := fmt.Fprintf(w, "github_backend: github\nrepo: %s\ndefault_branch: %s\nstate: %s\n", output.Repository.DisplayID, output.DefaultBranch, output.StatePath)
				return err
			}
			_, err := fmt.Fprintf(w, "%s\tgithub enabled=%t\n", action, enabled)
			return err
		})
	})
}

func runPlanningDiscussionAssess(cmd *cobra.Command, loadApp appLoader, selector string) error {
	return withPlanningIntegration(cmd, loadApp, application.PermissionRead, true, func(appCtx *app.App, service *application.Service, _ planningIntegrationProvider, adapter *planninggithub.Adapter, authorizer planningAuthorizer, _ string) error {
		assessment, err := service.AssessCollaboration(cmd.Context(), adapter.CollaborationSource(), githubDiscussionReference(selector), authorizer)
		if err != nil {
			return err
		}
		return appCtx.Output.Print(assessment, func(w io.Writer) error {
			_, err := fmt.Fprintf(w, "%s\t%s\n", assessment.Decision.State, assessment.Decision.Reason)
			return err
		})
	})
}

func runPlanningDiscussionRepair(cmd *cobra.Command, loadApp appLoader, selector string, specs []string, confirmed bool) error {
	return withPlanningIntegration(cmd, loadApp, application.PermissionRead, true, func(appCtx *app.App, service *application.Service, _ planningIntegrationProvider, adapter *planninggithub.Adapter, authorizer planningAuthorizer, _ string) error {
		input := application.CollaborationRepairInput{Source: githubDiscussionReference(selector), Specs: specs}
		preview, err := service.PreviewCollaborationRepair(cmd.Context(), adapter.CollaborationSource(), input, authorizer)
		if err != nil {
			return err
		}
		result := preview
		if confirmed && preview.Action != application.MutationUnchanged {
			input.ExpectedRevision = preview.Request.ExpectedRevision
			input.Confirmed = true
			result, err = service.RepairCollaboration(cmd.Context(), adapter.CollaborationSource(), input, authorizer, planningEventSink{history: appCtx.History})
			if err != nil {
				return err
			}
		}
		return appCtx.Output.Print(result, func(w io.Writer) error {
			if _, err := fmt.Fprintf(w, "%s\t%s\n", result.Action, result.Request.Source.URL); err != nil {
				return err
			}
			if !confirmed && result.Action != application.MutationUnchanged {
				_, err := fmt.Fprintln(w, "Preview only. Rerun with --confirm to write.")
				return err
			}
			return nil
		})
	})
}

type planningGitHubPromotionOutput struct {
	Assessment  application.CollaborationAssessment `json:"assessment"`
	Plan        application.PublicationPlan         `json:"plan"`
	Publication *application.PublicationApplyResult `json:"publication,omitempty"`
	Adoption    *application.AdoptionResult         `json:"adoption,omitempty"`
}

func runPlanningDiscussionPromote(cmd *cobra.Command, loadApp appLoader, selector, workspaceChoice string, apply, confirmed bool) error {
	if confirmed && !apply {
		return fmt.Errorf("--confirm requires --apply for GitHub promotion")
	}
	if apply && !confirmed {
		return fmt.Errorf("promotion apply requires explicit confirmation; rerun with --confirm")
	}
	return withPlanningIntegration(cmd, loadApp, application.PermissionRead, true, func(appCtx *app.App, service *application.Service, _ planningIntegrationProvider, adapter *planninggithub.Adapter, authorizer planningAuthorizer, _ string) error {
		intent, assessment, err := planningDiscussionPublicationIntent(cmd.Context(), service, adapter, selector, workspaceChoice, authorizer)
		if err != nil {
			return err
		}
		plan, err := service.PreviewPublication(cmd.Context(), adapter.PublicationTarget(), intent, authorizer)
		if err != nil {
			return err
		}
		output := planningGitHubPromotionOutput{Assessment: assessment, Plan: plan}
		if apply {
			publication, applyErr := service.ApplyPublication(cmd.Context(), adapter.PublicationTarget(), application.PublicationApplyInput{Intent: intent, ExpectedPlan: plan, Confirmed: true}, authorizer, planningEventSink{history: appCtx.History})
			output.Publication = &publication
			if applyErr != nil {
				_ = appCtx.Output.Print(output, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "partial\t%d completed action(s)\n", len(publication.Evidence.Completed))
					return err
				})
				return applyErr
			}
			candidates, err := publicationCandidates(plan, publication)
			if err != nil {
				return err
			}
			adoptIntent := application.AdoptionPreviewInput{Target: intent.Target, Source: intent.Source, Group: intent.Group, Workspace: intent.Workspace, Artifacts: intent.Artifacts, Candidates: candidates}
			adoptPlan, err := service.PreviewAdoption(cmd.Context(), adapter.PublicationTarget(), adapter.ExternalMappingRepository(), adoptIntent, authorizer)
			if err != nil {
				return err
			}
			adoption, err := service.ApplyAdoption(cmd.Context(), adapter.PublicationTarget(), adapter.ExternalMappingRepository(), application.AdoptionApplyInput{Intent: adoptIntent, ExpectedPlan: adoptPlan, Confirmed: true}, authorizer, planningEventSink{history: appCtx.History})
			output.Adoption = &adoption
			if err != nil {
				return err
			}
		}
		return appCtx.Output.Print(output, func(w io.Writer) error {
			_, err := fmt.Fprintf(w, "promotion: %d action(s)\n", len(plan.Actions))
			return err
		})
	})
}

func planningDiscussionPublicationIntent(ctx context.Context, service *application.Service, adapter *planninggithub.Adapter, selector, workspaceChoice string, authorizer planningAuthorizer) (application.PublicationPreviewInput, application.CollaborationAssessment, error) {
	ref := githubDiscussionReference(selector)
	assessment, err := service.AssessCollaboration(ctx, adapter.CollaborationSource(), ref, authorizer)
	if err != nil {
		return application.PublicationPreviewInput{}, assessment, err
	}
	if assessment.Decision.State != "ready_single_spec" && assessment.Decision.State != "ready_multi_spec" {
		return application.PublicationPreviewInput{}, assessment, fmt.Errorf("discussion is not ready for promotion: %s", assessment.Decision.Reason)
	}
	snapshot, err := adapter.CollaborationSource().Read(ctx, ref)
	if err != nil {
		return application.PublicationPreviewInput{}, assessment, err
	}
	status, err := adapter.Preflight(ctx)
	if err != nil {
		return application.PublicationPreviewInput{}, assessment, err
	}
	titles, _ := assessment.Decision.SuggestedTitles["specs"].([]string)
	if len(titles) == 0 {
		titles = []string{snapshot.Title}
	}
	artifacts := make([]application.PublicationArtifact, 0, len(titles)+1)
	var group *application.PublicationGroup
	var initiative *planning.ArtifactRef
	if len(titles) > 1 {
		ref := planning.ArtifactRef{Kind: planning.ArtifactInitiative, ID: planning.ArtifactID(planningGitHubSlug(snapshot.Title))}
		initiative = &ref
		artifacts = append(artifacts, application.PublicationArtifact{Artifact: ref, Title: snapshot.Title, Content: githubPublicationBody(snapshot.Title, snapshot.Content, snapshot.Source.URL), Readiness: planning.ReadinessReady})
		group = &application.PublicationGroup{Title: snapshot.Title}
	}
	dependencies := collaborationDependencies(assessment.Decision.DependencyGuess)
	for _, title := range titles {
		artifactRef := planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: planning.ArtifactID(planningGitHubSlug(title))}
		artifact := application.PublicationArtifact{Artifact: artifactRef, Title: title, Content: githubPublicationBody(title, snapshot.Content, snapshot.Source.URL), Readiness: planning.ReadinessReady}
		if initiative != nil {
			artifact.Group = initiative
		}
		for _, dependency := range dependencies[title] {
			artifact.Dependencies = append(artifact.Dependencies, planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: planning.ArtifactID(planningGitHubSlug(dependency))})
		}
		if len(artifact.Dependencies) > 0 {
			artifact.Readiness = planning.ReadinessBlocked
		}
		artifacts = append(artifacts, artifact)
	}
	if group != nil {
		for _, artifact := range artifacts {
			group.Members = append(group.Members, artifact.Artifact)
		}
	}
	workspace, err := planningPublicationWorkspaceDecision(ctx, adapter.ExternalMappingRepository(), workspaceChoice, snapshot.Title)
	if err != nil {
		return application.PublicationPreviewInput{}, assessment, err
	}
	return application.PublicationPreviewInput{Target: status.Repository, Source: &snapshot.Source, Artifacts: artifacts, Group: group, Workspace: workspace}, assessment, nil
}

func planningPublicationWorkspaceDecision(ctx context.Context, mappings application.ExternalMappingRepository, choice, title string) (*application.PublicationWorkspaceDecision, error) {
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "":
		return nil, nil
	case string(application.PublicationWorkspaceCreate):
		return &application.PublicationWorkspaceDecision{Choice: application.PublicationWorkspaceCreate, Title: strings.TrimSpace(title), Reason: "Coordinate execution for this publication."}, nil
	case string(application.PublicationWorkspaceSkip):
		return &application.PublicationWorkspaceDecision{Choice: application.PublicationWorkspaceSkip, Reason: "Explicitly publish without an execution workspace."}, nil
	case string(application.PublicationWorkspaceConnect):
		state, err := mappings.Load(ctx)
		if err != nil {
			return nil, err
		}
		if len(state.WorkspaceReferences) != 1 {
			return nil, fmt.Errorf("--workspace connect requires exactly one durable GitHub Project mapping")
		}
		ref := state.WorkspaceReferences[0]
		return &application.PublicationWorkspaceDecision{Choice: application.PublicationWorkspaceConnect, Reason: "Use the configured GitHub Project.", Reference: &ref}, nil
	default:
		return nil, fmt.Errorf("--workspace must be create, connect, or skip")
	}
}

func runPlanningGitHubAdopt(cmd *cobra.Command, loadApp appLoader, discussion, issues, workspaceChoice string, confirmed bool) error {
	if strings.TrimSpace(discussion) == "" || strings.TrimSpace(issues) == "" {
		return fmt.Errorf("github adopt requires --discussion and --issues")
	}
	return withPlanningIntegration(cmd, loadApp, application.PermissionRead, true, func(appCtx *app.App, service *application.Service, _ planningIntegrationProvider, adapter *planninggithub.Adapter, authorizer planningAuthorizer, _ string) error {
		publication, _, err := planningDiscussionPublicationIntent(cmd.Context(), service, adapter, discussion, workspaceChoice, authorizer)
		if err != nil {
			return err
		}
		numbers, err := parseIssueNumbers(issues)
		if err != nil || len(numbers) != len(publication.Artifacts) {
			return fmt.Errorf("--issues must provide one positive issue number for each of %d artifacts", len(publication.Artifacts))
		}
		candidates := make([]application.ArtifactExternalReference, len(numbers))
		for i, number := range numbers {
			url := fmt.Sprintf("%s/issues/%d", publication.Target.URL, number)
			candidates[i] = application.ArtifactExternalReference{Artifact: publication.Artifacts[i].Artifact, Reference: application.ExternalReference{Provider: "github", Kind: "issue", OpaqueID: url, DisplayID: strconv.Itoa(number), URL: url}}
		}
		intent := application.AdoptionPreviewInput{Target: publication.Target, Source: publication.Source, Group: publication.Group, Workspace: publication.Workspace, Artifacts: publication.Artifacts, Candidates: candidates}
		plan, err := service.PreviewAdoption(cmd.Context(), adapter.PublicationTarget(), adapter.ExternalMappingRepository(), intent, authorizer)
		if err != nil {
			return err
		}
		if !confirmed {
			return appCtx.Output.Print(plan, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "adoption: %d action(s)\nPreview only. Rerun with --confirm to write.\n", len(plan.Publication.Actions))
				return err
			})
		}
		result, err := service.ApplyAdoption(cmd.Context(), adapter.PublicationTarget(), adapter.ExternalMappingRepository(), application.AdoptionApplyInput{Intent: intent, ExpectedPlan: plan, Confirmed: true}, authorizer, planningEventSink{history: appCtx.History})
		if err != nil {
			return err
		}
		return appCtx.Output.Print(result, func(w io.Writer) error {
			_, err := fmt.Fprintf(w, "adopted: %d artifact(s)\n", len(result.Mappings.ArtifactReferences))
			return err
		})
	})
}

func runPlanningGitHubReconcile(cmd *cobra.Command, loadApp appLoader, updateVisible, confirmed bool) error {
	return withPlanningIntegration(cmd, loadApp, application.PermissionRead, true, func(appCtx *app.App, service *application.Service, _ planningIntegrationProvider, adapter *planninggithub.Adapter, authorizer planningAuthorizer, _ string) error {
		intent, err := localReconciliationIntent(cmd.Context(), service, adapter, updateVisible)
		if err != nil {
			return err
		}
		plan, err := service.PreviewReconciliation(cmd.Context(), adapter.PublicationTarget(), adapter.RepositoryEvidenceSource(), adapter.ExternalMappingRepository(), adapter.ExecutionWorkspace(), intent, authorizer)
		if err != nil {
			return err
		}
		if !confirmed {
			return appCtx.Output.Print(plan, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "reconcile: %d publication, %d attachment, %d status action(s)\nPreview only. Rerun with --confirm to write.\n", len(plan.Publication.Actions), len(plan.Attachments), len(plan.Execution))
				return err
			})
		}
		result, err := service.ApplyReconciliation(cmd.Context(), adapter.PublicationTarget(), adapter.RepositoryEvidenceSource(), adapter.ExternalMappingRepository(), adapter.ExecutionWorkspace(), application.ReconciliationApplyInput{Intent: intent, ExpectedPlan: plan, Confirmed: true}, authorizer, planningEventSink{history: appCtx.History})
		if err != nil {
			return err
		}
		return appCtx.Output.Print(result, func(w io.Writer) error {
			_, err := fmt.Fprintf(w, "reconciled: %d publication, %d execution action(s)\n", len(result.Publication.Completed), len(result.Execution))
			return err
		})
	})
}

func localReconciliationIntent(ctx context.Context, service *application.Service, adapter *planninggithub.Adapter, updateVisible bool) (application.ReconciliationPreviewInput, error) {
	status, err := adapter.Preflight(ctx)
	if err != nil {
		return application.ReconciliationPreviewInput{}, err
	}
	documents, err := service.ListSpecs(ctx)
	if err != nil {
		return application.ReconciliationPreviewInput{}, err
	}
	mappings, err := adapter.ExternalMappingRepository().Load(ctx)
	if err != nil {
		return application.ReconciliationPreviewInput{}, err
	}
	specs := make([]planning.Spec, 0, len(documents))
	for _, document := range documents {
		specs = append(specs, document.Artifact)
	}
	readinessByID, err := reconciliationReadiness(specs)
	if err != nil {
		return application.ReconciliationPreviewInput{}, err
	}
	input := application.ReconciliationPreviewInput{Publication: application.PublicationPreviewInput{Target: status.Repository}}
	for _, document := range documents {
		artifact := application.PublicationArtifact{Artifact: planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: document.Artifact.ID}, Title: document.Artifact.Title, Content: document.Body, Readiness: readinessByID[document.Artifact.ID]}
		for _, dependency := range document.Artifact.Dependencies {
			artifact.Dependencies = append(artifact.Dependencies, planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: dependency})
		}
		input.Publication.Artifacts = append(input.Publication.Artifacts, artifact)
	}
	if len(mappings.WorkspaceReferences) == 1 {
		workspace := mappings.WorkspaceReferences[0]
		input.Publication.Workspace = &application.PublicationWorkspaceDecision{Choice: application.PublicationWorkspaceConnect, Reference: &workspace}
	}
	if updateVisible {
		byArtifact := map[planning.ArtifactRef]application.ExternalReference{}
		for _, mapping := range mappings.ArtifactReferences {
			byArtifact[mapping.Artifact] = mapping.Reference
		}
		byStatus := map[planning.ArtifactRef]planning.SpecStatus{}
		for _, document := range documents {
			byStatus[planning.ArtifactRef{Kind: planning.ArtifactSpec, ID: document.Artifact.ID}] = document.Artifact.Status
		}
		for _, artifact := range input.Publication.Artifacts {
			if ref, ok := byArtifact[artifact.Artifact]; ok {
				input.Execution = append(input.Execution, application.ExecutionIntent{Artifact: artifact.Artifact, PublishedWork: ref, Status: reconciliationExecutionStatus(byStatus[artifact.Artifact])})
			}
		}
	}
	return input, nil
}

func reconciliationReadiness(specs []planning.Spec) (map[planning.ArtifactID]planning.ReadinessState, error) {
	if _, err := planning.OrderSpecs(specs); err != nil {
		return nil, err
	}
	byID := make(map[planning.ArtifactID]planning.Spec, len(specs))
	for _, spec := range specs {
		byID[spec.ID] = spec
	}
	result := make(map[planning.ArtifactID]planning.ReadinessState, len(specs))
	for _, spec := range specs {
		var blockers []planning.ArtifactID
		for _, dependency := range spec.Dependencies {
			if byID[dependency].Status != planning.SpecDone {
				blockers = append(blockers, dependency)
			}
		}
		state := planning.EvaluateReadiness(spec, planning.ValidateSpec(spec), blockers).State
		if state == planning.ReadinessClarifying {
			state = planning.ReadinessNeedsRefinement
		}
		result[spec.ID] = state
	}
	return result, nil
}

func reconciliationExecutionStatus(status planning.SpecStatus) application.ExecutionStatus {
	switch status {
	case planning.SpecImplementing:
		return application.ExecutionInProgress
	case planning.SpecDone:
		return application.ExecutionDone
	default:
		return ""
	}
}

func runPlanningGitHubProjectStatus(cmd *cobra.Command, loadApp appLoader, issue, value string, confirmed bool) error {
	number, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(issue), "#"))
	if err != nil || number < 1 {
		return fmt.Errorf("github project status requires a positive --issue")
	}
	status, err := parseExecutionStatus(value)
	if err != nil {
		return err
	}
	return withPlanningIntegration(cmd, loadApp, application.PermissionRead, true, func(appCtx *app.App, service *application.Service, _ planningIntegrationProvider, adapter *planninggithub.Adapter, authorizer planningAuthorizer, _ string) error {
		mappings, err := adapter.ExternalMappingRepository().Load(cmd.Context())
		if err != nil {
			return err
		}
		if len(mappings.WorkspaceReferences) != 1 {
			return fmt.Errorf("GitHub Project mapping is missing or ambiguous")
		}
		var published *application.ArtifactExternalReference
		for i := range mappings.ArtifactReferences {
			if mappings.ArtifactReferences[i].Reference.DisplayID == strconv.Itoa(number) {
				copy := mappings.ArtifactReferences[i]
				published = &copy
			}
		}
		if published == nil {
			return fmt.Errorf("issue #%d is not mapped to a Planning artifact", number)
		}
		snapshot, err := adapter.ExecutionWorkspace().Inspect(cmd.Context(), mappings.WorkspaceReferences[0])
		if err != nil {
			return err
		}
		var item *application.ExecutionWorkItem
		for i := range snapshot.Items {
			if samePlanningExternalLocation(snapshot.Items[i].PublishedWork, published.Reference) {
				copy := snapshot.Items[i]
				item = &copy
			}
		}
		if item == nil {
			return fmt.Errorf("issue #%d is not attached to the configured GitHub Project", number)
		}
		request := application.ExecutionStatusRequest{Workspace: snapshot.Workspace, WorkItem: item.Reference, Status: status}
		plan, err := service.PreviewExecutionStatus(cmd.Context(), adapter.ExecutionWorkspace(), request, authorizer)
		if err != nil {
			return err
		}
		if !confirmed {
			return appCtx.Output.Print(plan, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "%s\tissue #%d -> %s\nPreview only. Rerun with --confirm to write.\n", plan.Action, number, status)
				return err
			})
		}
		result, err := service.ApplyExecutionStatus(cmd.Context(), adapter.ExecutionWorkspace(), application.ExecutionStatusApplyInput{ExpectedPlan: plan, Confirmed: true}, authorizer, planningEventSink{history: appCtx.History})
		if err != nil {
			return err
		}
		return appCtx.Output.Print(result, func(w io.Writer) error {
			_, err := fmt.Fprintf(w, "%s\tissue #%d -> %s\n", result.Action, number, status)
			return err
		})
	})
}

func runPlanningDiscussionGuide(cmd *cobra.Command, loadApp appLoader, discussion, stage string) error {
	if strings.TrimSpace(stage) == "" {
		return fmt.Errorf("remote guide show requires --stage")
	}
	return withPlanningIntegration(cmd, loadApp, application.PermissionRead, true, func(appCtx *app.App, service *application.Service, _ planningIntegrationProvider, adapter *planninggithub.Adapter, authorizer planningAuthorizer, _ string) error {
		assessment, err := service.AssessCollaboration(cmd.Context(), adapter.CollaborationSource(), githubDiscussionReference(discussion), authorizer)
		if err != nil {
			return err
		}
		packet := map[string]any{"schema_version": application.IntegrationContractVersion, "kind": "guide_packet", "stage": stage, "source": assessment.Source, "assessment": assessment.Decision}
		return appCtx.Output.Print(packet, func(w io.Writer) error {
			_, err := fmt.Fprintf(w, "stage: %s\nassessment: %s\n", stage, assessment.Decision.State)
			return err
		})
	})
}

func extendPlanningGitHubCheck(ctx context.Context, appCtx *app.App, report application.CheckReport) (application.CheckReport, error) {
	resolution, err := appCtx.Modules.ResolveCommand(ctx, planningCommandGroup, application.PermissionRead)
	if err != nil {
		return report, err
	}
	provider, ok := resolution.Module.(planningIntegrationProvider)
	if !ok {
		return report, nil
	}
	adapter, enabled := provider.GitHubAdapter()
	if !enabled || adapter == nil {
		return report, nil
	}
	mappings, err := adapter.ExternalMappingRepository().Load(ctx)
	if err != nil {
		return report, err
	}
	if len(mappings.WorkspaceReferences) != 1 {
		report.Findings = append(report.Findings, planningGitHubCheckFinding(
			"error", "github.project.workspace_mapping", "GitHub Planning requires exactly one durable Project mapping.",
			"Adopt or publish the configured GitHub Project before relying on execution tracking.",
		))
		return report, nil
	}
	snapshot, err := adapter.ExecutionWorkspace().Inspect(ctx, mappings.WorkspaceReferences[0])
	if err != nil {
		return report, err
	}
	supported := map[application.ExecutionStatus]bool{}
	for _, field := range snapshot.Fields {
		if field.Role != application.ExecutionFieldRoleStatus {
			continue
		}
		for _, status := range field.SupportedStatuses {
			supported[status] = true
		}
	}
	for _, status := range []application.ExecutionStatus{application.ExecutionTodo, application.ExecutionInProgress, application.ExecutionInReview, application.ExecutionDone} {
		if !supported[status] {
			report.Findings = append(report.Findings, planningGitHubCheckFinding(
				"error", "github.project.missing_status_option", fmt.Sprintf("GitHub Project Status is missing the semantic %q option.", status),
				"Restore the reviewed Todo, In Progress, In Review, and Done status options, then rerun the check.",
			))
		}
	}
	items := map[planning.ArtifactRef]application.ExecutionWorkItem{}
	for _, item := range snapshot.Items {
		items[item.Artifact] = item
	}
	for _, mapping := range mappings.ArtifactReferences {
		item, exists := items[mapping.Artifact]
		if !exists {
			report.Findings = append(report.Findings, planningGitHubCheckFinding(
				"error", "github.project.missing_item", fmt.Sprintf("Mapped %s %s is not attached to the configured GitHub Project.", mapping.Artifact.Kind, mapping.Artifact.ID),
				"Preview GitHub reconciliation and confirm the missing Project attachment.",
			))
			continue
		}
		if item.Status == "" {
			report.Findings = append(report.Findings, planningGitHubCheckFinding(
				"warn", "github.project.missing_status", fmt.Sprintf("Mapped %s %s has no semantic execution status.", mapping.Artifact.Kind, mapping.Artifact.ID),
				"Set the Project item to Todo, In Progress, In Review, or Done.",
			))
		}
	}
	return report, nil
}

func planningGitHubCheckFinding(severity, rule, message, suggestion string) application.CheckFinding {
	return application.CheckFinding{
		Severity: severity, Rule: rule, ArtifactType: "integration", ArtifactPath: ".plan/.meta/github.json",
		ArtifactTitle: "GitHub Planning", Section: "GitHub Project", Message: message, Suggestion: suggestion,
	}
}

func githubDiscussionReference(selector string) application.ExternalReference {
	selector = strings.TrimSpace(selector)
	ref := application.ExternalReference{Provider: "github", Kind: "discussion"}
	if strings.HasPrefix(selector, "https://") {
		ref.URL = selector
		parts := strings.Split(strings.TrimSuffix(selector, "/"), "/")
		if len(parts) > 0 {
			ref.DisplayID = parts[len(parts)-1]
		}
	} else {
		ref.DisplayID = strings.TrimPrefix(selector, "#")
	}
	return ref
}

func githubPublicationBody(title, source, sourceURL string) string {
	return "# " + title + "\n\n" + strings.TrimSpace(source) + "\n\n## Source\n" + sourceURL + "\n"
}

func planningGitHubSlug(value string) string {
	var result strings.Builder
	separator := false
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			if separator && result.Len() > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(char)
			separator = false
		} else {
			separator = true
		}
	}
	return strings.Trim(result.String(), "-")
}

func collaborationDependencies(guesses []map[string]any) map[string][]string {
	result := map[string][]string{}
	for _, guess := range guesses {
		title, _ := guess["spec"].(string)
		switch values := guess["blocked_by"].(type) {
		case []string:
			result[title] = append([]string(nil), values...)
		case []any:
			for _, value := range values {
				if dependency, ok := value.(string); ok {
					result[title] = append(result[title], dependency)
				}
			}
		}
	}
	return result
}

func publicationCandidates(plan application.PublicationPlan, result application.PublicationApplyResult) ([]application.ArtifactExternalReference, error) {
	candidates := []application.ArtifactExternalReference{}
	for _, completed := range result.Evidence.Completed {
		if completed.Action.Kind != application.PublicationArtifactAction || completed.Action.Artifact == nil {
			continue
		}
		var issue *application.ExternalReference
		for i := range completed.References {
			if completed.References[i].Kind != "issue" {
				continue
			}
			if issue != nil {
				return nil, fmt.Errorf("publication returned multiple issue identities for one artifact")
			}
			ref := completed.References[i]
			issue = &ref
		}
		if issue != nil {
			candidates = append(candidates, application.ArtifactExternalReference{Artifact: completed.Action.Artifact.Artifact, Reference: *issue})
		}
	}
	artifacts := 0
	for _, action := range plan.Actions {
		if action.Kind == application.PublicationArtifactAction {
			artifacts++
		}
	}
	if len(candidates) != artifacts {
		return nil, fmt.Errorf("publication did not return one stable identity per artifact")
	}
	return candidates, nil
}

func parseIssueNumbers(value string) ([]int, error) {
	var result []int
	for _, item := range strings.Split(value, ",") {
		number, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(item, "#")))
		if err != nil || number < 1 {
			return nil, errors.New("issue numbers must be positive integers")
		}
		result = append(result, number)
	}
	return result, nil
}

func parseExecutionStatus(value string) (application.ExecutionStatus, error) {
	switch strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_") {
	case "todo":
		return application.ExecutionTodo, nil
	case "in_progress":
		return application.ExecutionInProgress, nil
	case "in_review":
		return application.ExecutionInReview, nil
	case "done":
		return application.ExecutionDone, nil
	default:
		return "", fmt.Errorf("--set must be todo, in-progress, in-review, or done")
	}
}

func samePlanningExternalLocation(left, right application.ExternalReference) bool {
	return left.Provider == right.Provider && left.Kind == right.Kind && left.DisplayID == right.DisplayID && left.URL == right.URL
}
