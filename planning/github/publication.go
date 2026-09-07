package github

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"
)

type publicationIssue struct {
	ID      string `json:"id"`
	NodeID  string `json:"node_id"`
	Number  int    `json:"number"`
	URL     string `json:"url"`
	HTMLURL string `json:"html_url"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	State   string `json:"state"`
	Labels  []struct {
		Name string `json:"name"`
	} `json:"labels"`
	PullRequest json.RawMessage `json:"pull_request"`
}

// gh issue list exposes a string node ID as id; REST exposes a numeric id plus
// node_id. Keep the provider's node identity in both representations.
func (issue *publicationIssue) UnmarshalJSON(raw []byte) error {
	type plainIssue publicationIssue
	var decoded plainIssue
	payload := struct {
		*plainIssue
		ID json.RawMessage `json:"id"`
	}{plainIssue: &decoded}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if len(payload.ID) > 0 && payload.ID[0] == '"' {
		if err := json.Unmarshal(payload.ID, &decoded.ID); err != nil {
			return err
		}
		if decoded.ID == "" {
			decoded.ID = decoded.NodeID
		}
		if decoded.NodeID != "" && decoded.NodeID != decoded.ID {
			return fmt.Errorf("conflicting issue node identities")
		}
		decoded.NodeID = decoded.ID
	} else {
		decoded.ID = decoded.NodeID
	}
	*issue = publicationIssue(decoded)
	return nil
}

func publicationIdentityError(message string) error {
	return providerError(application.IntegrationAmbiguousIdentity, "publication.inspect", message)
}

func (a *Adapter) inspectPublication(ctx context.Context, request application.PublicationInspectRequest) (application.PublicationSnapshot, error) {
	var empty application.PublicationSnapshot
	if !a.Enabled() {
		return empty, a.unavailable("publication.inspect")
	}
	repo := request.Target.OpaqueID
	if request.Target.Provider != providerName || request.Target.Kind != "repository" || !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9_.-]+$`).MatchString(repo) || strings.HasSuffix(repo, "/.") || strings.HasSuffix(repo, "/..") {
		return empty, publicationIdentityError("publication target must identify owner/repository")
	}
	if request.Target.URL != "" && request.Target.URL != "https://github.com/"+repo {
		return empty, publicationIdentityError("repository URL disagrees with identity")
	}
	wanted := map[planning.ArtifactRef]bool{}
	wantedSlugs := map[planning.ArtifactID]planning.ArtifactKind{}
	for _, ref := range request.Artifacts {
		if ref.Validate() != nil || (ref.Kind != planning.ArtifactInitiative && ref.Kind != planning.ArtifactSpec) || wanted[ref] {
			return empty, publicationIdentityError("invalid or duplicate artifact identity")
		}
		wanted[ref] = true
		if kind, exists := wantedSlugs[ref.ID]; exists && kind != ref.Kind {
			return empty, publicationIdentityError("metadata cannot represent multiple kinds under one slug")
		}
		wantedSlugs[ref.ID] = ref.Kind
	}
	if len(wanted) == 0 {
		return empty, publicationIdentityError("publication inspection requires artifacts")
	}
	sourceURL := ""
	if request.Source != nil {
		if request.Source.URL == "" {
			return empty, publicationIdentityError("source recovery requires a canonical source URL")
		}
		owner, name, number, err := a.discussionLocation(ctx, *request.Source)
		if err != nil {
			return empty, err
		}
		sourceURL = fmt.Sprintf("https://github.com/%s/%s/discussions/%d", owner, name, number)
	}
	state, err := a.state.read()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return empty, providerError(application.IntegrationProviderUnavailable, "publication.inspect", "cannot read adapter metadata")
	}
	if err != nil {
		state = normalizeGitHubState(githubState{})
	}
	if (state.Repo != "" && state.Repo != repo) || (state.Repo == "" && len(state.Planning) > 0) {
		return empty, publicationIdentityError("metadata belongs to a different repository")
	}
	byNumber := map[int]publicationIssue{}
	kinds := []planning.ArtifactKind{planning.ArtifactInitiative, planning.ArtifactSpec}
	for _, kind := range kinds {
		needed := false
		for ref := range wanted {
			needed = needed || ref.Kind == kind
		}
		if !needed {
			continue
		}
		raw, err := a.runProvider(ctx, "publication.inspect", "issue", "list", "--repo", repo, "--state", "all", "--limit", "1000", "--json", "id,number,url,title,body,state,labels", "--label", "plan:"+string(kind))
		if err != nil {
			return empty, err
		}
		var issues []publicationIssue
		if json.Unmarshal(raw, &issues) != nil || strings.TrimSpace(string(raw)) == "null" {
			return empty, providerError(application.IntegrationProviderUnavailable, "publication.inspect", "invalid issue listing")
		}
		if len(issues) >= 1000 {
			return empty, providerError(application.IntegrationProviderUnavailable, "publication.inspect", "issue listing reached safety limit; narrow publication scope")
		}
		for _, issue := range issues {
			if err := validatePublicationIssue(&issue, repo); err != nil {
				return empty, err
			}
			if old, exists := byNumber[issue.Number]; exists && !reflect.DeepEqual(old, issue) {
				return empty, providerError(application.IntegrationRevisionConflict, "publication.inspect", "provider changed issue evidence during listing")
			}
			byNumber[issue.Number] = issue
		}
	}
	// Metadata is authoritative for renamed or unlabelled issues. Read those
	// directly; a missing known object must not turn into a create action.
	mapped := map[planning.ArtifactRef]int{}
	for slug, record := range state.Planning {
		ref := planning.ArtifactRef{Kind: planning.ArtifactKind(record.Kind), ID: planning.ArtifactID(slug)}
		if kind, exists := wantedSlugs[ref.ID]; exists && kind != ref.Kind {
			return empty, publicationIdentityError("known slug belongs to another artifact kind")
		}
		if !wanted[ref] {
			continue
		}
		if record.Slug != slug || record.IssueNumber < 1 {
			return empty, publicationIdentityError("invalid known publication mapping")
		}
		if sourceURL != "" && record.DiscussionURL != "" && record.DiscussionURL != sourceURL {
			return empty, publicationIdentityError("known artifact belongs to another source")
		}
		mapped[ref] = record.IssueNumber
		if _, exists := byNumber[record.IssueNumber]; !exists {
			raw, err := a.runProvider(ctx, "publication.inspect", "api", "--method", "GET", fmt.Sprintf("repos/%s/issues/%d", repo, record.IssueNumber))
			if err != nil {
				return empty, err
			}
			var issue publicationIssue
			if json.Unmarshal(raw, &issue) != nil {
				return empty, providerError(application.IntegrationProviderUnavailable, "publication.inspect", "invalid mapped issue response")
			}
			if err := validatePublicationIssue(&issue, repo); err != nil {
				return empty, err
			}
			if issue.Number != record.IssueNumber {
				return empty, publicationIdentityError("provider returned another mapped issue")
			}
			byNumber[issue.Number] = issue
		}
		if record.IssueURL != "" && record.IssueURL != byNumber[record.IssueNumber].URL {
			return empty, publicationIdentityError("mapped issue URL disagrees with provider")
		}
	}
	return a.publicationSnapshot(ctx, request, repo, sourceURL, wanted, mapped, byNumber)
}

func validatePublicationIssue(issue *publicationIssue, repo string) error {
	if issue.ID == "" {
		issue.ID = issue.NodeID
	}
	if issue.HTMLURL != "" {
		issue.URL = issue.HTMLURL
	}
	if issue.ID == "" || issue.Number < 1 || issue.URL != fmt.Sprintf("https://github.com/%s/issues/%d", repo, issue.Number) || (len(issue.PullRequest) > 0 && string(issue.PullRequest) != "null") {
		return publicationIdentityError("provider returned invalid issue identity")
	}
	return nil
}

func (a *Adapter) publicationSnapshot(ctx context.Context, request application.PublicationInspectRequest, repo, sourceURL string, wanted map[planning.ArtifactRef]bool, mapped map[planning.ArtifactRef]int, issues map[int]publicationIssue) (application.PublicationSnapshot, error) {
	var refs []planning.ArtifactRef
	for ref := range wanted {
		refs = append(refs, ref)
	}
	slices.SortFunc(refs, func(a, b planning.ArtifactRef) int {
		return strings.Compare(string(a.Kind)+"/"+string(a.ID), string(b.Kind)+"/"+string(b.ID))
	})
	result := application.PublicationSnapshot{SchemaVersion: application.IntegrationContractVersion, Target: request.Target}
	result.Target.URL = "https://github.com/" + repo
	numberToIndex := map[int]int{}
	for _, ref := range refs {
		var matches []publicationIssue
		for _, issue := range issues {
			identity := mapped[ref] == issue.Number
			// Recovery requires both a canonical source link and an exact slug
			// with the semantic kind label. Title alone never establishes identity.
			if sourceURL != "" && publicationSourceLink(issue.Body, sourceURL) && publicationSlug(issue.Title) == string(ref.ID) && publicationHasLabel(issue, "plan:"+string(ref.Kind)) {
				identity = true
			}
			if identity {
				matches = append(matches, issue)
			}
		}
		if len(matches) > 1 {
			return application.PublicationSnapshot{}, publicationIdentityError("multiple issues match one publication artifact")
		}
		if len(matches) == 0 {
			continue
		}
		issue := matches[0]
		if _, duplicate := numberToIndex[issue.Number]; duplicate {
			return application.PublicationSnapshot{}, publicationIdentityError("one issue matches multiple publication artifacts")
		}
		readiness := planning.ReadinessNeedsRefinement
		if publicationHasLabel(issue, "plan:ready") {
			readiness = planning.ReadinessReady
		}
		if publicationHasLabel(issue, "plan:blocked") {
			readiness = planning.ReadinessBlocked
		}
		if strings.EqualFold(issue.State, "closed") {
			readiness = planning.ReadinessDone
		}
		raw, _ := json.Marshal(issue)
		external := application.ExternalReference{Provider: providerName, Kind: "issue", OpaqueID: issue.ID, DisplayID: strconv.Itoa(issue.Number), URL: issue.URL, Revision: fmt.Sprintf("%x", sha256.Sum256(raw))}
		numberToIndex[issue.Number] = len(result.Artifacts)
		result.Artifacts = append(result.Artifacts, application.PublicationArtifact{Artifact: ref, Title: issue.Title, Content: issue.Body, Readiness: readiness, Reference: &external})
	}
	for i := range result.Artifacts {
		artifact := &result.Artifacts[i]
		number, _ := strconv.Atoi(artifact.Reference.DisplayID)
		for _, kind := range []application.PublicationRelationshipKind{application.PublicationRelationshipContains, application.PublicationRelationshipDependsOn} {
			endpoint := "sub_issues"
			if kind == application.PublicationRelationshipDependsOn {
				endpoint = "dependencies/blocked_by"
			}
			raw, err := a.runProvider(ctx, "publication.inspect", "api", "--method", "GET", fmt.Sprintf("repos/%s/issues/%d/%s?per_page=100", repo, number, endpoint))
			if err != nil {
				return application.PublicationSnapshot{}, err
			}
			var linked []publicationIssue
			if json.Unmarshal(raw, &linked) != nil || strings.TrimSpace(string(raw)) == "null" {
				return application.PublicationSnapshot{}, providerError(application.IntegrationProviderUnavailable, "publication.inspect", "invalid relationship response")
			}
			if len(linked) >= 100 {
				return application.PublicationSnapshot{}, providerError(application.IntegrationProviderUnavailable, "publication.inspect", "relationship listing reached safety limit")
			}
			seen := map[int]bool{}
			slices.SortFunc(linked, func(a, b publicationIssue) int {
				if a.Number < b.Number {
					return -1
				}
				if a.Number > b.Number {
					return 1
				}
				return 0
			})
			for _, link := range linked {
				if err := validatePublicationIssue(&link, repo); err != nil {
					return application.PublicationSnapshot{}, err
				}
				j, found := numberToIndex[link.Number]
				if !found {
					return application.PublicationSnapshot{}, providerError(application.IntegrationManualRemediationRequired, "publication.inspect", "publication scope omits an existing relationship endpoint")
				}
				other := &result.Artifacts[j]
				if other.Reference.OpaqueID != link.ID {
					return application.PublicationSnapshot{}, publicationIdentityError("relationship identity disagrees with issue")
				}
				if seen[link.Number] {
					continue
				}
				seen[link.Number] = true
				result.Relationships = append(result.Relationships, application.PublicationRelationship{Kind: kind, Source: artifact.Artifact, Target: other.Artifact})
				if kind == application.PublicationRelationshipContains {
					if artifact.Artifact.Kind != planning.ArtifactInitiative || other.Artifact.Kind != planning.ArtifactSpec || (other.Group != nil && *other.Group != artifact.Artifact) {
						return application.PublicationSnapshot{}, publicationIdentityError("provider grouping conflicts with planning roles")
					}
					group := artifact.Artifact
					other.Group = &group
				} else {
					artifact.Dependencies = append(artifact.Dependencies, other.Artifact)
				}
			}
		}
	}
	return result, nil
}

func publicationHasLabel(issue publicationIssue, label string) bool {
	for _, item := range issue.Labels {
		if item.Name == label {
			return true
		}
	}
	return false
}

func publicationSourceLink(body, link string) bool {
	if link == "" {
		return false
	}
	for offset := 0; offset < len(body); {
		index := strings.Index(body[offset:], link)
		if index < 0 {
			return false
		}
		start := offset + index
		end := start + len(link)
		if (start == 0 || strings.ContainsRune("\t\n\f\r (<", rune(body[start-1]))) &&
			(end == len(body) || strings.ContainsRune("\t\n\f\r )]>", rune(body[end]))) {
			return true
		}
		offset = start + 1
	}
	return false
}

var publicationSlugSeparator = regexp.MustCompile(`[^a-z0-9]+`)

func publicationSlug(title string) string {
	return strings.Trim(publicationSlugSeparator.ReplaceAllString(strings.ToLower(strings.TrimSpace(title)), "-"), "-")
}
