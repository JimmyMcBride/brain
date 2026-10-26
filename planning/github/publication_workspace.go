package github

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/JimmyMcBride/brain/planning/application"
)

const publicationProjectFields = `
  id number url title
  owner {
    __typename
    ... on User { login }
    ... on Organization { login }
  }
  repositories(first:100) {
    nodes { nameWithOwner }
    pageInfo { hasNextPage }
  }`

type publicationProject struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	URL    string `json:"url"`
	Title  string `json:"title"`
	Owner  struct {
		Typename string `json:"__typename"`
		Login    string `json:"login"`
	} `json:"owner"`
	Repositories struct {
		Nodes []struct {
			NameWithOwner string `json:"nameWithOwner"`
		} `json:"nodes"`
		PageInfo struct {
			HasNextPage bool `json:"hasNextPage"`
		} `json:"pageInfo"`
	} `json:"repositories"`
}

func (a *Adapter) publicationWorkspaceSnapshot(ctx context.Context, desired *application.PublicationWorkspaceDecision, repo string) (*application.PublicationWorkspaceDecision, error) {
	if desired == nil || desired.Choice == application.PublicationWorkspaceSkip {
		return nil, nil
	}
	var project *publicationProject
	if desired.Reference != nil {
		found, err := a.publicationProjectByReference(ctx, "publication.inspect", *desired.Reference)
		if err != nil {
			return nil, err
		}
		project = &found
	} else {
		found, err := a.publicationProjectByTitle(ctx, "publication.inspect", repo, desired.Title)
		if err != nil {
			return nil, err
		}
		project = found
	}
	if project == nil {
		return nil, nil
	}
	ref := publicationProjectReference(*project)
	return &application.PublicationWorkspaceDecision{Choice: desired.Choice, Title: project.Title, Reason: desired.Reason, Reference: &ref}, nil
}

func (a *Adapter) publicationProjectByReference(ctx context.Context, operation string, ref application.ExternalReference) (publicationProject, error) {
	var project publicationProject
	if _, _, err := publicationProjectLocation(ref); err != nil {
		return project, providerError(application.IntegrationAmbiguousIdentity, operation, "workspace reference has no canonical Project identity")
	}
	query := `query($id:ID!) { node(id:$id) { ... on ProjectV2 {` + publicationProjectFields + ` } } }`
	var response struct {
		Data struct {
			Node *publicationProject `json:"node"`
		} `json:"data"`
	}
	if err := a.graphql(ctx, operation, &response, "-f", "query="+query, "-f", "id="+ref.OpaqueID); err != nil {
		return project, err
	}
	if response.Data.Node == nil {
		return project, providerError(application.IntegrationAmbiguousIdentity, operation, "selected Project no longer exists")
	}
	project = *response.Data.Node
	if err := validatePublicationProject(project, operation); err != nil {
		return publicationProject{}, err
	}
	current := publicationProjectReference(project)
	if current.Provider != ref.Provider || current.Kind != ref.Kind || current.OpaqueID != ref.OpaqueID || current.DisplayID != ref.DisplayID || current.URL != ref.URL {
		return publicationProject{}, providerError(application.IntegrationAmbiguousIdentity, operation, "selected Project identity disagrees with provider")
	}
	return project, nil
}

func (a *Adapter) publicationProjectByTitle(ctx context.Context, operation, repo, title string) (*publicationProject, error) {
	owner := strings.Split(repo, "/")[0]
	query := `query($owner:String!) {
  repositoryOwner(login:$owner) {
    login
    ... on User { projects: projectsV2(first:100) { nodes {` + publicationProjectFields + ` } pageInfo { hasNextPage } } }
    ... on Organization { projects: projectsV2(first:100) { nodes {` + publicationProjectFields + ` } pageInfo { hasNextPage } } }
  }
}`
	var response struct {
		Data struct {
			RepositoryOwner *struct {
				Login    string `json:"login"`
				Projects struct {
					Nodes    []publicationProject `json:"nodes"`
					PageInfo struct {
						HasNextPage bool `json:"hasNextPage"`
					} `json:"pageInfo"`
				} `json:"projects"`
			} `json:"repositoryOwner"`
		} `json:"data"`
	}
	if err := a.graphql(ctx, operation, &response, "-f", "query="+query, "-f", "owner="+owner); err != nil {
		return nil, err
	}
	if response.Data.RepositoryOwner == nil || !strings.EqualFold(response.Data.RepositoryOwner.Login, owner) {
		return nil, providerError(application.IntegrationAmbiguousIdentity, operation, "repository owner could not be resolved")
	}
	if response.Data.RepositoryOwner.Projects.PageInfo.HasNextPage || len(response.Data.RepositoryOwner.Projects.Nodes) >= 100 {
		return nil, providerError(application.IntegrationProviderUnavailable, operation, "Project listing reached safety limit")
	}
	var match *publicationProject
	for i := range response.Data.RepositoryOwner.Projects.Nodes {
		candidate := &response.Data.RepositoryOwner.Projects.Nodes[i]
		if err := validatePublicationProject(*candidate, operation); err != nil {
			return nil, err
		}
		linked := false
		for _, repository := range candidate.Repositories.Nodes {
			linked = linked || strings.EqualFold(repository.NameWithOwner, repo)
		}
		if linked && strings.EqualFold(strings.TrimSpace(candidate.Title), title) {
			if match != nil {
				return nil, providerError(application.IntegrationAmbiguousIdentity, operation, "multiple repository-linked Projects match the workspace title")
			}
			copy := *candidate
			match = &copy
		}
	}
	return match, nil
}

func validatePublicationProject(project publicationProject, operation string) error {
	prefix := ""
	switch project.Owner.Typename {
	case "User":
		prefix = "users"
	case "Organization":
		prefix = "orgs"
	default:
		return providerError(application.IntegrationAmbiguousIdentity, operation, "provider returned an invalid Project owner")
	}
	wantURL := fmt.Sprintf("https://github.com/%s/%s/projects/%d", prefix, project.Owner.Login, project.Number)
	if project.ID == "" || project.Number < 1 || strings.TrimSpace(project.Title) == "" || project.Title != strings.TrimSpace(project.Title) || project.Owner.Login == "" || project.URL != wantURL || project.Repositories.PageInfo.HasNextPage || len(project.Repositories.Nodes) >= 100 {
		return providerError(application.IntegrationAmbiguousIdentity, operation, "provider returned an invalid or incomplete Project identity")
	}
	seen := map[string]bool{}
	for _, repository := range project.Repositories.Nodes {
		name := strings.ToLower(strings.TrimSpace(repository.NameWithOwner))
		if name == "" || !strings.Contains(name, "/") || seen[name] {
			return providerError(application.IntegrationAmbiguousIdentity, operation, "provider returned invalid Project repository evidence")
		}
		seen[name] = true
	}
	return nil
}

func publicationProjectLocation(ref application.ExternalReference) (string, int, error) {
	if ref.Provider != providerName || ref.Kind != "project" || ref.OpaqueID == "" || !strings.HasPrefix(ref.URL, "https://github.com/") {
		return "", 0, fmt.Errorf("invalid Project reference")
	}
	parts := strings.Split(strings.TrimPrefix(ref.URL, "https://github.com/"), "/")
	number, err := strconv.Atoi(ref.DisplayID)
	if err != nil || number < 1 || len(parts) != 4 || (parts[0] != "users" && parts[0] != "orgs") || parts[1] == "" || parts[2] != "projects" || parts[3] != strconv.Itoa(number) {
		return "", 0, fmt.Errorf("invalid Project reference")
	}
	return parts[1], number, nil
}

func publicationProjectReference(project publicationProject) application.ExternalReference {
	repositories := make([]string, 0, len(project.Repositories.Nodes))
	for _, repository := range project.Repositories.Nodes {
		repositories = append(repositories, strings.ToLower(repository.NameWithOwner))
	}
	slices.Sort(repositories)
	raw, _ := json.Marshal(struct {
		ID           string
		Number       int
		URL          string
		Title        string
		OwnerType    string
		Owner        string
		Repositories []string
	}{project.ID, project.Number, project.URL, project.Title, project.Owner.Typename, project.Owner.Login, repositories})
	return application.ExternalReference{Provider: providerName, Kind: "project", OpaqueID: project.ID, DisplayID: strconv.Itoa(project.Number), URL: project.URL, Revision: fmt.Sprintf("%x", sha256.Sum256(raw))}
}

func (a *Adapter) writePublicationWorkspace(ctx context.Context, plan application.PublicationPlan, action application.PublicationApplyAction) (*publicationProject, error) {
	workspace := *action.Workspace
	if action.Action != application.MutationCreate || workspace.Choice != application.PublicationWorkspaceCreate {
		return nil, publicationApplyConflict("invalid Project create action")
	}
	if existing, err := a.publicationProjectByTitle(ctx, publicationApplyOperation, plan.Target.OpaqueID, workspace.Title); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, publicationApplyConflict("Project appeared before create")
	}
	parts := strings.Split(plan.Target.OpaqueID, "/")
	query := `query($owner:String!, $name:String!) { repositoryOwner(login:$owner) { id login } repository(owner:$owner, name:$name) { id nameWithOwner } }`
	var identities struct {
		Data struct {
			RepositoryOwner *struct {
				ID    string `json:"id"`
				Login string `json:"login"`
			} `json:"repositoryOwner"`
			Repository *struct {
				ID            string `json:"id"`
				NameWithOwner string `json:"nameWithOwner"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := a.graphql(ctx, publicationApplyOperation, &identities, "-f", "query="+query, "-f", "owner="+parts[0], "-f", "name="+parts[1]); err != nil {
		return nil, err
	}
	if identities.Data.RepositoryOwner == nil || identities.Data.Repository == nil || identities.Data.RepositoryOwner.ID == "" || identities.Data.Repository.ID == "" || !strings.EqualFold(identities.Data.RepositoryOwner.Login, parts[0]) || !strings.EqualFold(identities.Data.Repository.NameWithOwner, plan.Target.OpaqueID) {
		return nil, providerError(application.IntegrationAmbiguousIdentity, publicationApplyOperation, "repository owner identity changed before Project create")
	}
	mutation := `mutation($ownerId:ID!, $repositoryId:ID!, $title:String!) { createProjectV2(input:{ownerId:$ownerId, repositoryId:$repositoryId, title:$title}) { projectV2 {` + publicationProjectFields + ` } } }`
	payload := map[string]any{"query": mutation, "variables": map[string]string{"ownerId": identities.Data.RepositoryOwner.ID, "repositoryId": identities.Data.Repository.ID, "title": workspace.Title}}
	raw, err := a.publicationRequest(ctx, "POST", "graphql", payload)
	if err != nil {
		return nil, err
	}
	var response struct {
		Data struct {
			CreateProjectV2 struct {
				Project publicationProject `json:"projectV2"`
			} `json:"createProjectV2"`
		} `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	if json.Unmarshal(raw, &response) != nil || len(response.Errors) != 0 || validatePublicationProject(response.Data.CreateProjectV2.Project, publicationApplyOperation) != nil || response.Data.CreateProjectV2.Project.Title != workspace.Title {
		return nil, providerError(application.IntegrationPartialFailure, publicationApplyOperation, "Project mutation returned invalid evidence; inspect before retrying")
	}
	linked := false
	for _, repository := range response.Data.CreateProjectV2.Project.Repositories.Nodes {
		linked = linked || strings.EqualFold(repository.NameWithOwner, plan.Target.OpaqueID)
	}
	if !linked {
		return nil, providerError(application.IntegrationPartialFailure, publicationApplyOperation, "Project mutation omitted repository linkage; inspect before retrying")
	}
	return &response.Data.CreateProjectV2.Project, nil
}
