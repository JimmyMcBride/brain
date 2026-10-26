package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JimmyMcBride/brain/planning/application"
)

const repositoryPreflightOperation = "repository.preflight"

// RepositoryStatus is the bounded repository identity established while
// enabling or checking the optional adapter.
type RepositoryStatus struct {
	Repository    application.ExternalReference `json:"repository"`
	DefaultBranch string                        `json:"default_branch"`
	StatePath     string                        `json:"state_path"`
	Action        application.MutationAction    `json:"action"`
}

// Preflight verifies gh authentication, repository identity, the default
// branch, and issue support without changing local or provider state.
func (a *Adapter) Preflight(ctx context.Context) (RepositoryStatus, error) {
	if !a.Enabled() {
		return RepositoryStatus{}, a.unavailable(repositoryPreflightOperation)
	}
	raw, err := a.runProvider(ctx, repositoryPreflightOperation, "repo", "view", "--json", "nameWithOwner,url,defaultBranchRef,hasIssuesEnabled")
	if err != nil {
		return RepositoryStatus{}, err
	}
	var view struct {
		NameWithOwner string `json:"nameWithOwner"`
		URL           string `json:"url"`
		DefaultBranch struct {
			Name string `json:"name"`
		} `json:"defaultBranchRef"`
		HasIssuesEnabled bool `json:"hasIssuesEnabled"`
	}
	if json.Unmarshal(raw, &view) != nil || !repositoryNamePattern.MatchString(view.NameWithOwner) ||
		view.URL != "https://github.com/"+view.NameWithOwner || view.DefaultBranch.Name == "" || !view.HasIssuesEnabled {
		return RepositoryStatus{}, providerError(application.IntegrationAmbiguousIdentity, repositoryPreflightOperation, "repository identity, default branch, or issue support is invalid")
	}
	return RepositoryStatus{
		Repository:    application.ExternalReference{Provider: providerName, Kind: "repository", OpaqueID: view.NameWithOwner, DisplayID: view.NameWithOwner, URL: view.URL},
		DefaultBranch: view.DefaultBranch.Name,
		StatePath:     filepath.Join(a.projectRoot, filepath.FromSlash(metadataRelativePath)),
	}, nil
}

// Enable preflights the provider and initializes compatible local adapter
// state. Repeated calls preserve mappings and perform no write when unchanged.
func (a *Adapter) Enable(ctx context.Context) (RepositoryStatus, error) {
	status, err := a.Preflight(ctx)
	if err != nil {
		return RepositoryStatus{}, err
	}
	state, readErr := a.state.read()
	action := application.MutationUnchanged
	if errors.Is(readErr, os.ErrNotExist) {
		state = normalizeGitHubState(githubState{})
		action = application.MutationCreate
	} else if readErr != nil {
		return RepositoryStatus{}, readErr
	}
	if state.Repo != "" && (state.Repo != status.Repository.OpaqueID || state.RepoURL != status.Repository.URL) {
		return RepositoryStatus{}, providerError(application.IntegrationAmbiguousIdentity, repositoryPreflightOperation, "adapter state belongs to a different repository")
	}
	if state.Repo != status.Repository.OpaqueID || state.RepoURL != status.Repository.URL || state.DefaultBranch != status.DefaultBranch || state.LastEnabledAt == "" {
		if action == application.MutationUnchanged {
			action = application.MutationUpdate
		}
		now := a.currentTime().Format("2006-01-02T15:04:05Z")
		state.Repo = status.Repository.OpaqueID
		state.RepoURL = status.Repository.URL
		state.DefaultBranch = status.DefaultBranch
		state.LastEnabledAt = now
		state.LastUpdatedAt = now
		if err := a.state.write(state); err != nil {
			return RepositoryStatus{}, err
		}
	}
	status.Action = action
	return status, nil
}

// Health validates provider access and the durable adapter identity without
// changing either local or remote state.
func (a *Adapter) Health(ctx context.Context) error {
	status, err := a.Preflight(ctx)
	if err != nil {
		return err
	}
	state, err := a.state.read()
	if err != nil {
		return err
	}
	if state.Repo != status.Repository.OpaqueID || state.RepoURL != status.Repository.URL || state.DefaultBranch != status.DefaultBranch {
		return fmt.Errorf("GitHub adapter state does not match the current repository")
	}
	return nil
}
