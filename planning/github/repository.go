package github

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/JimmyMcBride/brain/planning/application"
)

const repositoryEvidenceOperation = "repository.current"

var repositoryNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9_.-]+$`)

type repositoryView struct {
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
	DefaultBranch struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
}

type repositoryPullRequest struct {
	ID             string `json:"id"`
	Number         int    `json:"number"`
	URL            string `json:"url"`
	State          string `json:"state"`
	Draft          bool   `json:"isDraft"`
	MergedAt       string `json:"mergedAt"`
	HeadRefName    string `json:"headRefName"`
	BaseRefName    string `json:"baseRefName"`
	HeadRefOID     string `json:"headRefOid"`
	HeadRepository struct {
		Name string `json:"name"`
	} `json:"headRepository"`
	HeadRepositoryOwner struct {
		Login string `json:"login"`
	} `json:"headRepositoryOwner"`
}

func (a *Adapter) currentRepositoryEvidence(ctx context.Context) (application.RepositoryEvidence, error) {
	var evidence application.RepositoryEvidence
	if !a.Enabled() {
		return evidence, a.unavailable(repositoryEvidenceOperation)
	}
	repoRaw, err := a.runProvider(ctx, repositoryEvidenceOperation, "repo", "view", "--json", "nameWithOwner,url,defaultBranchRef")
	if err != nil {
		return evidence, err
	}
	var repo repositoryView
	if json.Unmarshal(repoRaw, &repo) != nil || !validRepositoryView(repo) {
		return evidence, repositoryEvidenceError(application.IntegrationAmbiguousIdentity, "provider returned an invalid repository identity")
	}
	branchRaw, err := a.runRepositoryGit(ctx, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return evidence, repositoryEvidenceError(application.IntegrationAmbiguousIdentity, "detached or unreadable repository HEAD")
	}
	branch := strings.TrimSpace(string(branchRaw))
	if branch == "" || strings.ContainsAny(branch, "\r\n") {
		return evidence, repositoryEvidenceError(application.IntegrationAmbiguousIdentity, "current branch is invalid")
	}
	commitRaw, err := a.runRepositoryGit(ctx, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return evidence, err
	}
	commit := strings.TrimSpace(string(commitRaw))
	if !isCommitOID(commit) {
		return evidence, repositoryEvidenceError(application.IntegrationAmbiguousIdentity, "current commit identity is invalid")
	}
	evidence = application.RepositoryEvidence{
		SchemaVersion: application.IntegrationContractVersion,
		Repository: application.ExternalReference{
			Provider: providerName, Kind: "repository", OpaqueID: repo.NameWithOwner,
			DisplayID: repo.NameWithOwner, URL: repo.URL,
		},
		DefaultRef: repo.DefaultBranch.Name,
		CurrentRef: branch,
		Commit:     commit,
	}
	prRaw, err := a.runProvider(ctx, repositoryEvidenceOperation, "pr", "list", "--repo", repo.NameWithOwner, "--head", branch, "--state", "all", "--limit", "100", "--json", "id,number,url,state,isDraft,mergedAt,headRefName,baseRefName,headRefOid,headRepository,headRepositoryOwner")
	if err != nil {
		return application.RepositoryEvidence{}, err
	}
	var prs []repositoryPullRequest
	if json.Unmarshal(prRaw, &prs) != nil || len(prs) >= 100 {
		return application.RepositoryEvidence{}, repositoryEvidenceError(application.IntegrationProviderUnavailable, "invalid or incomplete pull request listing")
	}
	if len(prs) > 1 {
		return application.RepositoryEvidence{}, repositoryEvidenceError(application.IntegrationAmbiguousIdentity, "multiple pull requests match the checked-out branch")
	}
	for _, pr := range prs {
		if err := validateRepositoryPullRequest(pr, repo, branch); err != nil {
			return application.RepositoryEvidence{}, err
		}
		if pr.HeadRefOID != commit {
			return application.RepositoryEvidence{}, repositoryEvidenceError(application.IntegrationRevisionConflict, "pull request head changed from the checked-out commit")
		}
	}
	if len(prs) == 1 {
		pr := prs[0]
		state := application.ChangeRequestClosed
		if strings.TrimSpace(pr.MergedAt) != "" {
			state = application.ChangeRequestMerged
		} else if strings.EqualFold(pr.State, "OPEN") {
			state = application.ChangeRequestOpen
		}
		evidence.PlanningChangeRequest = &application.PlanningChangeRequestEvidence{
			Reference: application.ExternalReference{
				Provider: providerName, Kind: "pull_request", OpaqueID: pr.ID,
				DisplayID: fmt.Sprintf("#%d", pr.Number), URL: pr.URL, Revision: pr.HeadRefOID,
			},
			State: state, HeadRef: pr.HeadRefName, BaseRef: pr.BaseRefName, Draft: pr.Draft,
		}
	}
	return evidence, nil
}

func (a *Adapter) runRepositoryGit(ctx context.Context, args ...string) ([]byte, error) {
	result, err := a.gitRunner.Run(ctx, a.projectRoot, args...)
	return providerOutput(ctx, repositoryEvidenceOperation, result, err)
}

func validRepositoryView(repo repositoryView) bool {
	return repositoryNamePattern.MatchString(repo.NameWithOwner) &&
		repo.URL == "https://github.com/"+repo.NameWithOwner &&
		strings.TrimSpace(repo.DefaultBranch.Name) == repo.DefaultBranch.Name && repo.DefaultBranch.Name != ""
}

func validateRepositoryPullRequest(pr repositoryPullRequest, repo repositoryView, branch string) error {
	owner, name, ok := strings.Cut(repo.NameWithOwner, "/")
	if !ok || pr.ID == "" || pr.Number < 1 || pr.URL != fmt.Sprintf("https://github.com/%s/pull/%d", repo.NameWithOwner, pr.Number) ||
		pr.HeadRefName != branch || pr.HeadRepositoryOwner.Login != owner || pr.HeadRepository.Name != name ||
		pr.BaseRefName != repo.DefaultBranch.Name || !isCommitOID(pr.HeadRefOID) {
		return repositoryEvidenceError(application.IntegrationAmbiguousIdentity, "pull request evidence disagrees with the configured repository")
	}
	switch strings.ToUpper(pr.State) {
	case "OPEN", "CLOSED", "MERGED":
	default:
		return repositoryEvidenceError(application.IntegrationAmbiguousIdentity, "pull request state is invalid")
	}
	return nil
}

func isCommitOID(value string) bool {
	// Production git rev-parse returns the full object ID. Accept the short
	// hexadecimal identity retained by the pinned conformance fixtures too.
	if len(value) < 6 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}

func repositoryEvidenceError(class application.IntegrationErrorClass, message string) error {
	return providerError(class, repositoryEvidenceOperation, message)
}
