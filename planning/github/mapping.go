package github

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JimmyMcBride/brain/planning"
	"github.com/JimmyMcBride/brain/planning/application"
)

const mappingLockTimeout = time.Second

func mappingError(class application.IntegrationErrorClass, operation, message string) error {
	return &application.IntegrationError{Class: class, Provider: providerName, Operation: operation, Message: message}
}

func mappingRevision(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func (a *Adapter) loadExternalMappings(ctx context.Context) (application.ExternalMappingState, error) {
	if !a.Enabled() {
		return application.ExternalMappingState{}, a.unavailable("mapping.load")
	}
	if err := ctx.Err(); err != nil {
		return application.ExternalMappingState{}, err
	}
	store, ok := a.state.(*fileStateStore)
	if !ok {
		return application.ExternalMappingState{}, a.unavailable("mapping.load")
	}
	raw, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return application.ExternalMappingState{SchemaVersion: application.IntegrationContractVersion}, nil
	}
	if err != nil {
		return application.ExternalMappingState{}, mappingError(application.IntegrationProviderUnavailable, "mapping.load", "cannot read GitHub mapping metadata")
	}
	var state githubState
	if json.Unmarshal(raw, &state) != nil {
		return application.ExternalMappingState{}, mappingError(application.IntegrationProviderUnavailable, "mapping.load", "cannot parse GitHub mapping metadata")
	}
	result, err := externalMappingsFromGitHub(normalizeGitHubState(state))
	if err != nil {
		return application.ExternalMappingState{}, err
	}
	result.Revision = mappingRevision(raw)
	return result, nil
}

func externalMappingsFromGitHub(state githubState) (application.ExternalMappingState, error) {
	result := application.ExternalMappingState{SchemaVersion: application.IntegrationContractVersion}
	if state.Repo == "" && (len(state.Planning) > 0 || len(state.ProjectDecisions) > 0) {
		return result, mappingError(application.IntegrationAmbiguousIdentity, "mapping.load", "mapping metadata has no repository identity")
	}
	if state.Repo != "" && state.RepoURL != "" && state.RepoURL != "https://github.com/"+state.Repo {
		return result, mappingError(application.IntegrationAmbiguousIdentity, "mapping.load", "repository URL disagrees with mapping identity")
	}
	artifactKeys := make([]string, 0, len(state.Planning))
	for key := range state.Planning {
		artifactKeys = append(artifactKeys, key)
	}
	slices.Sort(artifactKeys)
	sources := map[string]application.ExternalReference{}
	for _, key := range artifactKeys {
		record := state.Planning[key]
		kind := planning.ArtifactKind(record.Kind)
		ref := planning.ArtifactRef{Kind: kind, ID: planning.ArtifactID(record.Slug)}
		if record.Slug != key || ref.Validate() != nil || (kind != planning.ArtifactInitiative && kind != planning.ArtifactSpec) {
			return result, mappingError(application.IntegrationAmbiguousIdentity, "mapping.load", "mapping metadata contains an invalid artifact identity")
		}
		issue, err := legacyIssueReference(state.Repo, record.IssueNumber, record.IssueURL)
		if err != nil {
			return result, err
		}
		result.ArtifactReferences = append(result.ArtifactReferences, application.ArtifactExternalReference{Artifact: ref, Reference: issue})
		if record.DiscussionNumber > 0 || record.DiscussionURL != "" {
			source, err := legacyDiscussionReference(state.Repo, record.DiscussionNumber, record.DiscussionURL)
			if err != nil {
				return result, err
			}
			sources[source.URL] = source
		}
		if record.MilestoneNumber > 0 || record.MilestoneTitle != "" {
			if record.MilestoneNumber < 1 {
				return result, mappingError(application.IntegrationAmbiguousIdentity, "mapping.load", "milestone title has no stable number")
			}
			milestoneURL := fmt.Sprintf("https://github.com/%s/milestone/%d", state.Repo, record.MilestoneNumber)
			result.GroupReferences = append(result.GroupReferences, application.ArtifactExternalReference{Artifact: ref, Reference: application.ExternalReference{Provider: providerName, Kind: "milestone", OpaqueID: milestoneURL, DisplayID: strconv.Itoa(record.MilestoneNumber), URL: milestoneURL}})
		}
	}
	for _, source := range sources {
		result.SourceReferences = append(result.SourceReferences, source)
	}
	slices.SortFunc(result.SourceReferences, func(a, b application.ExternalReference) int { return strings.Compare(a.URL, b.URL) })
	projectKeys := make([]string, 0, len(state.ProjectDecisions))
	for key := range state.ProjectDecisions {
		projectKeys = append(projectKeys, key)
	}
	slices.Sort(projectKeys)
	seenProjects := map[string]application.ExternalReference{}
	for _, key := range projectKeys {
		record := state.ProjectDecisions[key]
		if record.ProjectID == "" && record.ProjectURL == "" && record.ProjectNumber == 0 {
			continue
		}
		reference, err := legacyProjectReference(state.Repo, record)
		if err != nil {
			return result, err
		}
		if existing, ok := seenProjects[record.ProjectID]; ok {
			if existing != reference {
				return result, mappingError(application.IntegrationAmbiguousIdentity, "mapping.load", "workspace identity maps to multiple Projects")
			}
			continue
		}
		seenProjects[record.ProjectID] = reference
		result.WorkspaceReferences = append(result.WorkspaceReferences, reference)
	}
	if len(seenProjects) > 1 {
		return result, mappingError(application.IntegrationAmbiguousIdentity, "mapping.load", "mapping metadata contains multiple workspace identities")
	}
	slices.SortFunc(result.WorkspaceReferences, externalReferenceLess)
	if state.Reconciliation != nil && state.LastReconciledAt == "" {
		return result, mappingError(application.IntegrationAmbiguousIdentity, "mapping.load", "reconciliation evidence has no completion time")
	}
	if state.LastReconciledAt != "" {
		completed, err := time.Parse(time.RFC3339, state.LastReconciledAt)
		if err != nil || state.Repo == "" {
			return result, mappingError(application.IntegrationAmbiguousIdentity, "mapping.load", "reconciliation evidence is invalid")
		}
		repoURL := "https://github.com/" + state.Repo
		repository := application.RepositoryEvidence{SchemaVersion: application.IntegrationContractVersion, Repository: application.ExternalReference{Provider: providerName, Kind: "repository", OpaqueID: state.Repo, DisplayID: state.Repo, URL: repoURL}, DefaultRef: state.DefaultBranch}
		if state.Reconciliation != nil {
			repository.CurrentRef = state.Reconciliation.CurrentRef
			repository.Commit = state.Reconciliation.Commit
			if change := state.Reconciliation.PlanningChangeRequest; change != nil {
				repository.PlanningChangeRequest = &application.PlanningChangeRequestEvidence{
					Reference: application.ExternalReference{Provider: providerName, Kind: "pull_request", OpaqueID: change.OpaqueID, DisplayID: fmt.Sprintf("#%d", change.Number), URL: change.URL, Revision: state.Reconciliation.Commit},
					State:     application.ChangeRequestState(change.State), HeadRef: change.HeadRef, BaseRef: change.BaseRef, Draft: change.Draft,
				}
			}
			if err := validateStoredRepositoryEvidence(repository, "mapping.load"); err != nil {
				return result, err
			}
		}
		result.LastReconciliation = &application.ReconciliationEvidence{Repository: repository, CompletedAt: completed}
	}
	return result, nil
}

func legacyIssueReference(repo string, number int, issueURL string) (application.ExternalReference, error) {
	want := fmt.Sprintf("https://github.com/%s/issues/%d", repo, number)
	if repo == "" || number < 1 || issueURL != want {
		return application.ExternalReference{}, mappingError(application.IntegrationAmbiguousIdentity, "mapping.load", "issue mapping has no canonical stable identity")
	}
	return application.ExternalReference{Provider: providerName, Kind: "issue", OpaqueID: want, DisplayID: strconv.Itoa(number), URL: want}, nil
}

func legacyDiscussionReference(repo string, number int, discussionURL string) (application.ExternalReference, error) {
	want := fmt.Sprintf("https://github.com/%s/discussions/%d", repo, number)
	if repo == "" || number < 1 || discussionURL != want {
		return application.ExternalReference{}, mappingError(application.IntegrationAmbiguousIdentity, "mapping.load", "discussion mapping has no canonical stable identity")
	}
	return application.ExternalReference{Provider: providerName, Kind: "discussion", OpaqueID: want, DisplayID: strconv.Itoa(number), URL: want}, nil
}

func legacyProjectReference(repo string, record githubProjectDecision) (application.ExternalReference, error) {
	path := strings.TrimPrefix(record.ProjectURL, "https://github.com/")
	parts := strings.Split(path, "/")
	number := strconv.Itoa(record.ProjectNumber)
	repoOwner := strings.Split(repo, "/")[0]
	if record.ProjectID == "" || record.ProjectNumber < 1 || len(parts) != 4 ||
		(parts[0] != "users" && parts[0] != "orgs") || parts[1] == "" ||
		parts[2] != "projects" || parts[3] != number ||
		!strings.EqualFold(parts[1], repoOwner) || record.ProjectOwner != "" && record.ProjectOwner != parts[1] ||
		record.ProjectURL != "https://github.com/"+strings.Join(parts, "/") {
		return application.ExternalReference{}, mappingError(application.IntegrationAmbiguousIdentity, "mapping.load", "workspace mapping has no canonical stable identity")
	}
	return application.ExternalReference{Provider: providerName, Kind: "project", OpaqueID: record.ProjectID, DisplayID: number, URL: record.ProjectURL}, nil
}

func externalReferenceLess(a, b application.ExternalReference) int {
	if value := strings.Compare(a.Provider, b.Provider); value != 0 {
		return value
	}
	if value := strings.Compare(a.Kind, b.Kind); value != 0 {
		return value
	}
	return strings.Compare(a.OpaqueID, b.OpaqueID)
}

func (a *Adapter) saveExternalMappings(ctx context.Context, desired application.ExternalMappingState, expectedRevision string) (application.ExternalMappingState, error) {
	if !a.Enabled() {
		return application.ExternalMappingState{}, a.unavailable("mapping.save")
	}
	store, ok := a.state.(*fileStateStore)
	if !ok {
		return application.ExternalMappingState{}, a.unavailable("mapping.save")
	}
	if desired.SchemaVersion != application.IntegrationContractVersion || desired.Revision != "" && desired.Revision != expectedRevision {
		return application.ExternalMappingState{}, mappingError(application.IntegrationRevisionConflict, "mapping.save", "mapping state does not match expected revision")
	}
	release, err := acquireMappingLock(ctx, store.path+".lock")
	if err != nil {
		return application.ExternalMappingState{}, err
	}
	defer release()
	raw, err := os.ReadFile(store.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return application.ExternalMappingState{}, mappingError(application.IntegrationProviderUnavailable, "mapping.save", "cannot read GitHub mapping metadata")
	}
	if errors.Is(err, os.ErrNotExist) {
		raw = nil
	}
	if mappingRevision(raw) != expectedRevision {
		return application.ExternalMappingState{}, mappingError(application.IntegrationRevisionConflict, "mapping.save", "mapping metadata changed since load")
	}
	current := normalizeGitHubState(githubState{})
	if len(raw) > 0 && json.Unmarshal(raw, &current) != nil {
		return application.ExternalMappingState{}, mappingError(application.IntegrationProviderUnavailable, "mapping.save", "cannot parse GitHub mapping metadata")
	}
	current = normalizeGitHubState(current)
	next, err := mergeExternalMappings(current, desired)
	if err != nil {
		return application.ExternalMappingState{}, err
	}
	if reflect.DeepEqual(next, current) {
		result, err := externalMappingsFromGitHub(current)
		if err != nil {
			return application.ExternalMappingState{}, err
		}
		result.Revision = expectedRevision
		return result, nil
	}
	next.LastUpdatedAt = time.Now().UTC().Format(time.RFC3339)
	written, err := encodeGitHubState(next)
	if err != nil {
		return application.ExternalMappingState{}, mappingError(application.IntegrationProviderUnavailable, "mapping.save", "cannot encode GitHub mapping metadata")
	}
	latest, latestErr := os.ReadFile(store.path)
	if errors.Is(latestErr, os.ErrNotExist) {
		latest = nil
	} else if latestErr != nil {
		return application.ExternalMappingState{}, mappingError(application.IntegrationProviderUnavailable, "mapping.save", "cannot recheck GitHub mapping metadata")
	}
	if mappingRevision(latest) != expectedRevision {
		return application.ExternalMappingState{}, mappingError(application.IntegrationRevisionConflict, "mapping.save", "mapping metadata changed during save")
	}
	if err := atomicWriteFile(store.path, written, 0o644); err != nil {
		return application.ExternalMappingState{}, mappingError(application.IntegrationProviderUnavailable, "mapping.save", "cannot replace GitHub mapping metadata")
	}
	result, err := externalMappingsFromGitHub(next)
	if err != nil {
		return application.ExternalMappingState{}, err
	}
	result.Revision = mappingRevision(written)
	return result, nil
}

func mergeExternalMappings(current githubState, desired application.ExternalMappingState) (githubState, error) {
	next := current
	next.Planning = map[string]githubPlanningRecord{}
	next.ProjectDecisions = make(map[string]githubProjectDecision, len(current.ProjectDecisions))
	for key, record := range current.ProjectDecisions {
		next.ProjectDecisions[key] = record
	}
	repo := current.Repo
	seenArtifacts := map[planning.ArtifactRef]bool{}
	seenIssues := map[string]bool{}
	for _, mapping := range desired.ArtifactReferences {
		if mapping.Artifact.Validate() != nil || (mapping.Artifact.Kind != planning.ArtifactInitiative && mapping.Artifact.Kind != planning.ArtifactSpec) || seenArtifacts[mapping.Artifact] {
			return next, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "artifact mappings contain an invalid or duplicate identity")
		}
		seenArtifacts[mapping.Artifact] = true
		issueRepo, number, err := mappingIssueLocation(mapping.Reference)
		if err != nil {
			return next, err
		}
		if repo == "" {
			repo = issueRepo
		}
		if repo != issueRepo || seenIssues[mapping.Reference.URL] {
			return next, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "artifact mappings cross repositories or reuse one issue")
		}
		seenIssues[mapping.Reference.URL] = true
		key := string(mapping.Artifact.ID)
		record := current.Planning[key]
		record.Slug, record.Kind, record.IssueNumber, record.IssueURL = key, string(mapping.Artifact.Kind), number, mapping.Reference.URL
		next.Planning[key] = record
	}
	if current.Repo != "" && current.Repo != repo {
		return next, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "mapping metadata belongs to another repository")
	}
	if len(desired.SourceReferences) > 1 {
		return next, mappingError(application.IntegrationUnsupportedCapability, "mapping.save", "legacy metadata cannot assign multiple sources without artifact associations")
	}
	if len(desired.SourceReferences) == 1 && len(seenArtifacts) == 0 {
		return next, mappingError(application.IntegrationUnsupportedCapability, "mapping.save", "legacy metadata cannot retain a source without an artifact mapping")
	}
	var source application.ExternalReference
	if len(desired.SourceReferences) == 1 {
		source = desired.SourceReferences[0]
		sourceRepo, number, err := mappingDiscussionLocation(source)
		if repo == "" {
			repo = sourceRepo
		}
		if err != nil || sourceRepo != repo {
			return next, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "source mapping disagrees with repository")
		}
		for key, record := range next.Planning {
			record.DiscussionNumber, record.DiscussionURL = number, source.URL
			if record.OwnershipMode == "" {
				record.OwnershipMode = "github"
			}
			if record.SourceMode == "" {
				record.SourceMode = "github"
			}
			if record.EntryMode == "" {
				record.EntryMode = "github_discussion"
			}
			next.Planning[key] = record
		}
	} else {
		for key, record := range next.Planning {
			record.DiscussionNumber, record.DiscussionURL = 0, ""
			next.Planning[key] = record
		}
	}
	next.Repo = repo
	if repo != "" {
		next.RepoURL = "https://github.com/" + repo
	}
	groups := map[planning.ArtifactRef]application.ExternalReference{}
	for _, mapping := range desired.GroupReferences {
		if !seenArtifacts[mapping.Artifact] || groups[mapping.Artifact].OpaqueID != "" {
			return next, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "group mapping has no unique artifact")
		}
		if err := validateMilestoneReference(repo, mapping.Reference); err != nil {
			return next, err
		}
		groups[mapping.Artifact] = mapping.Reference
	}
	for artifact := range seenArtifacts {
		record := next.Planning[string(artifact.ID)]
		group := groups[artifact]
		if group.OpaqueID == "" {
			record.MilestoneNumber, record.MilestoneTitle = 0, ""
		} else {
			record.MilestoneNumber, _ = strconv.Atoi(group.DisplayID)
			if old := current.Planning[string(artifact.ID)]; old.MilestoneNumber == record.MilestoneNumber {
				record.MilestoneTitle = old.MilestoneTitle
			}
		}
		next.Planning[string(artifact.ID)] = record
	}
	if err := mergeWorkspaceMappings(&next, current, desired.WorkspaceReferences); err != nil {
		return next, err
	}
	if desired.LastReconciliation == nil {
		next.LastReconciledAt = ""
		next.Reconciliation = nil
	} else {
		evidence := desired.LastReconciliation
		repository := evidence.Repository.Repository
		if evidence.CompletedAt.IsZero() || repository.Provider != providerName || repository.Kind != "repository" || repository.OpaqueID != repo || repository.URL != "https://github.com/"+repo {
			return next, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "reconciliation evidence disagrees with repository")
		}
		next.LastReconciledAt = desired.LastReconciliation.CompletedAt.UTC().Format(time.RFC3339)
		if evidence.Repository.CurrentRef != "" || evidence.Repository.Commit != "" || evidence.Repository.PlanningChangeRequest != nil {
			if err := validateStoredRepositoryEvidence(evidence.Repository, "mapping.save"); err != nil {
				return next, err
			}
			next.DefaultBranch = evidence.Repository.DefaultRef
			next.Reconciliation = reconciliationRecord(evidence.Repository)
		}
	}
	return next, nil
}

func validateStoredRepositoryEvidence(evidence application.RepositoryEvidence, operation string) error {
	repository := evidence.Repository
	if evidence.SchemaVersion != application.IntegrationContractVersion || repository.Provider != providerName || repository.Kind != "repository" ||
		repository.OpaqueID == "" || repository.DisplayID != repository.OpaqueID || repository.URL != "https://github.com/"+repository.OpaqueID ||
		evidence.DefaultRef == "" || strings.TrimSpace(evidence.DefaultRef) != evidence.DefaultRef || evidence.CurrentRef == "" || strings.TrimSpace(evidence.CurrentRef) != evidence.CurrentRef || !isCommitOID(evidence.Commit) {
		return mappingError(application.IntegrationAmbiguousIdentity, operation, "reconciliation evidence is invalid")
	}
	if change := evidence.PlanningChangeRequest; change != nil {
		number, err := strconv.Atoi(strings.TrimPrefix(change.Reference.DisplayID, "#"))
		if err != nil || number < 1 || change.Reference.Provider != providerName || change.Reference.Kind != "pull_request" || change.Reference.OpaqueID == "" ||
			change.Reference.DisplayID != fmt.Sprintf("#%d", number) || change.Reference.URL != fmt.Sprintf("https://github.com/%s/pull/%d", repository.OpaqueID, number) ||
			change.Reference.Revision != evidence.Commit || change.HeadRef != evidence.CurrentRef || change.BaseRef != evidence.DefaultRef {
			return mappingError(application.IntegrationAmbiguousIdentity, operation, "reconciliation change-request evidence is invalid")
		}
		switch change.State {
		case application.ChangeRequestOpen, application.ChangeRequestClosed, application.ChangeRequestMerged:
		default:
			return mappingError(application.IntegrationAmbiguousIdentity, operation, "reconciliation change-request state is invalid")
		}
	}
	return nil
}

func reconciliationRecord(evidence application.RepositoryEvidence) *githubReconciliationRecord {
	record := &githubReconciliationRecord{CurrentRef: evidence.CurrentRef, Commit: evidence.Commit}
	if change := evidence.PlanningChangeRequest; change != nil {
		number, _ := strconv.Atoi(strings.TrimPrefix(change.Reference.DisplayID, "#"))
		record.PlanningChangeRequest = &githubChangeRequestRecord{
			OpaqueID: change.Reference.OpaqueID, Number: number, URL: change.Reference.URL, State: string(change.State),
			HeadRef: change.HeadRef, BaseRef: change.BaseRef, Draft: change.Draft,
		}
	}
	return record
}

func mergeWorkspaceMappings(next *githubState, current githubState, desired []application.ExternalReference) error {
	if len(desired) > 1 {
		return mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "mapping metadata supports one canonical workspace")
	}
	if len(desired) == 0 {
		for key, record := range next.ProjectDecisions {
			if record.ProjectID == "" && record.ProjectURL == "" && record.ProjectNumber == 0 {
				continue
			}
			record.ProjectOwner, record.ProjectNumber, record.ProjectID, record.ProjectURL = "", 0, "", ""
			next.ProjectDecisions[key] = record
		}
		return nil
	}
	owner, number, err := mappingProjectLocation(next.Repo, desired[0])
	if err != nil {
		return err
	}
	updated := false
	for key, record := range next.ProjectDecisions {
		if record.ProjectID == "" && record.ProjectURL == "" && record.ProjectNumber == 0 {
			continue
		}
		record.ProjectOwner, record.ProjectNumber, record.ProjectID, record.ProjectURL = owner, number, desired[0].OpaqueID, desired[0].URL
		next.ProjectDecisions[key] = record
		updated = true
	}
	if updated {
		return nil
	}
	key, record, err := workspaceDecisionRecord(*next, current)
	if err != nil {
		return err
	}
	record.ProjectOwner, record.ProjectNumber, record.ProjectID, record.ProjectURL = owner, number, desired[0].OpaqueID, desired[0].URL
	next.ProjectDecisions[key] = record
	return nil
}

func mappingProjectLocation(repo string, ref application.ExternalReference) (string, int, error) {
	if ref.Provider != providerName || ref.Kind != "project" || ref.OpaqueID == "" {
		return "", 0, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "expected a GitHub Project mapping")
	}
	parts := strings.Split(strings.TrimPrefix(ref.URL, "https://github.com/"), "/")
	number, err := strconv.Atoi(ref.DisplayID)
	repoOwner := strings.Split(repo, "/")[0]
	if err != nil || number < 1 || ref.DisplayID != strconv.Itoa(number) || len(parts) != 4 ||
		(parts[0] != "users" && parts[0] != "orgs") || parts[1] == "" || !strings.EqualFold(parts[1], repoOwner) ||
		parts[2] != "projects" || parts[3] != ref.DisplayID || ref.URL != "https://github.com/"+strings.Join(parts, "/") {
		return "", 0, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "workspace mapping is not canonical for the repository owner")
	}
	return parts[1], number, nil
}

func workspaceDecisionRecord(state githubState, current githubState) (string, githubProjectDecision, error) {
	keys := make([]string, 0, len(state.ProjectDecisions))
	for key, record := range state.ProjectDecisions {
		if record.ProjectID == "" && record.ProjectURL == "" && record.ProjectNumber == 0 {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	if len(keys) == 1 {
		record := state.ProjectDecisions[keys[0]]
		decision := strings.ToLower(strings.TrimSpace(record.Decision))
		if decision == "skip" || decision != "" && decision != "create" && decision != "connect" {
			return "", githubProjectDecision{}, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "workspace identity conflicts with the existing project decision")
		}
		if record.Slug == "" {
			record.Slug = keys[0]
		}
		if record.Decision == "" {
			record.Decision = "connect"
		}
		return keys[0], record, nil
	}
	initiative := ""
	first := ""
	specs := 0
	for key, record := range state.Planning {
		if first == "" || key < first {
			first = key
		}
		if record.Kind == string(planning.ArtifactInitiative) {
			if initiative != "" {
				return "", githubProjectDecision{}, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "workspace mapping has multiple initiative owners")
			}
			initiative = key
		} else if record.Kind == string(planning.ArtifactSpec) {
			specs++
		}
	}
	key := initiative
	if key == "" {
		key = first
	}
	if key == "" || len(keys) > 1 {
		return "", githubProjectDecision{}, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "workspace mapping has no unique project-decision owner")
	}
	record := current.ProjectDecisions[key]
	if record.Slug == "" {
		record.Slug = key
	}
	if record.Decision == "" {
		// A durable selected identity is represented as a connection. Creation is
		// provider history; the mapping contract records the current relationship.
		record.Decision = "connect"
	}
	if initiative != "" && record.InitiativeSlug == "" {
		record.InitiativeSlug = initiative
	}
	if record.SpecCount == 0 {
		record.SpecCount = specs
	}
	if planningRecord := state.Planning[key]; planningRecord.MilestoneNumber > 0 {
		record.MilestoneNumber = planningRecord.MilestoneNumber
		if record.MilestoneTitle == "" {
			record.MilestoneTitle = planningRecord.MilestoneTitle
		}
	}
	return key, record, nil
}

func mappingIssueLocation(ref application.ExternalReference) (string, int, error) {
	if ref.Provider != providerName || ref.Kind != "issue" {
		return "", 0, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "expected a GitHub issue mapping")
	}
	parts := strings.Split(strings.TrimPrefix(ref.URL, "https://github.com/"), "/")
	if len(parts) != 4 || parts[2] != "issues" {
		return "", 0, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "issue mapping URL is not canonical")
	}
	number, err := strconv.Atoi(parts[3])
	repo := parts[0] + "/" + parts[1]
	if err != nil || number < 1 || ref.DisplayID != strconv.Itoa(number) || ref.URL != fmt.Sprintf("https://github.com/%s/issues/%d", repo, number) || ref.OpaqueID == "" {
		return "", 0, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "issue mapping identity is inconsistent")
	}
	return repo, number, nil
}

func mappingDiscussionLocation(ref application.ExternalReference) (string, int, error) {
	if ref.Provider != providerName || ref.Kind != "discussion" {
		return "", 0, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "expected a GitHub Discussion mapping")
	}
	parts := strings.Split(strings.TrimPrefix(ref.URL, "https://github.com/"), "/")
	if len(parts) != 4 || parts[2] != "discussions" {
		return "", 0, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "Discussion mapping URL is not canonical")
	}
	number, err := strconv.Atoi(parts[3])
	repo := parts[0] + "/" + parts[1]
	if err != nil || number < 1 || ref.DisplayID != strconv.Itoa(number) || ref.URL != fmt.Sprintf("https://github.com/%s/discussions/%d", repo, number) || ref.OpaqueID == "" {
		return "", 0, mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "Discussion mapping identity is inconsistent")
	}
	return repo, number, nil
}

func validateMilestoneReference(repo string, ref application.ExternalReference) error {
	number, err := strconv.Atoi(ref.DisplayID)
	want := fmt.Sprintf("https://github.com/%s/milestone/%d", repo, number)
	if ref.Provider != providerName || ref.Kind != "milestone" || number < 1 || err != nil || ref.DisplayID != strconv.Itoa(number) || ref.URL != want || ref.OpaqueID == "" {
		return mappingError(application.IntegrationAmbiguousIdentity, "mapping.save", "milestone mapping identity is inconsistent")
	}
	return nil
}

func acquireMappingLock(ctx context.Context, path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, mappingError(application.IntegrationProviderUnavailable, "mapping.save", "cannot create mapping metadata directory")
	}
	deadline := time.Now().Add(mappingLockTimeout)
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			return func() { _ = file.Close(); _ = os.Remove(path) }, nil
		}
		if !isMappingLockContention(err) {
			return nil, mappingError(application.IntegrationProviderUnavailable, "mapping.save", "cannot acquire mapping metadata lock")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, mappingError(application.IntegrationRevisionConflict, "mapping.save", "mapping metadata lock timed out")
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
