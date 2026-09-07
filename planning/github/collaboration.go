package github

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"github.com/JimmyMcBride/brain/planning/application"
)

const discussionQuery = `query($owner:String!, $name:String!, $number:Int!, $after:String) {
 repository(owner:$owner, name:$name) { discussion(number:$number) {
  id number url title body updatedAt
  comments(first:100, after:$after) { nodes { body } pageInfo { hasNextPage endCursor } }
 } }
}`

type discussion struct {
	ID, URL, Title, Body, UpdatedAt string
	Number                          int
	Comments                        struct {
		Nodes    []struct{ Body string }
		PageInfo struct {
			HasNextPage bool
			EndCursor   string
		}
	}
}

func providerError(class application.IntegrationErrorClass, operation, message string) error {
	return &application.IntegrationError{Class: class, Provider: providerName, Operation: operation, Message: message}
}

// Provider stderr can contain credentials or source content. Only stable error
// classes and caller cancellation escape the adapter.
func (a *Adapter) runProvider(ctx context.Context, operation string, args ...string) ([]byte, error) {
	result, err := a.runner.Run(ctx, a.projectRoot, args...)
	return providerOutput(ctx, operation, result, err)
}

func providerOutput(ctx context.Context, operation string, result RunResult, err error) ([]byte, error) {
	if err == nil {
		return result.Stdout, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	class := application.IntegrationProviderUnavailable
	var exitError *exec.ExitError
	message := strings.ToLower(string(result.Stderr))
	switch {
	case errors.As(err, &exitError) && exitError.ExitCode() == 4, strings.Contains(message, "http 401"), strings.Contains(message, "gh auth login"):
		class = application.IntegrationUnauthenticated
	case strings.Contains(message, "http 403"):
		class = application.IntegrationUnauthorized
	}
	return nil, providerError(class, operation, "provider command failed")
}

func (a *Adapter) graphql(ctx context.Context, operation string, target any, args ...string) error {
	raw, err := a.runProvider(ctx, operation, append([]string{"api", "graphql"}, args...)...)
	if err != nil {
		return err
	}
	var envelope struct{ Errors []struct{ Type string } }
	if json.Unmarshal(raw, &envelope) != nil {
		return providerError(application.IntegrationProviderUnavailable, operation, "invalid provider response")
	}
	if len(envelope.Errors) != 0 {
		class := application.IntegrationProviderUnavailable
		if envelope.Errors[0].Type == "FORBIDDEN" {
			class = application.IntegrationUnauthorized
		}
		return providerError(class, operation, "provider query failed")
	}
	if json.Unmarshal(raw, target) != nil {
		return providerError(application.IntegrationProviderUnavailable, operation, "invalid provider response")
	}
	return nil
}

func (a *Adapter) discussionLocation(ctx context.Context, ref application.ExternalReference) (string, string, int, error) {
	invalid := func() (string, string, int, error) {
		return "", "", 0, providerError(application.IntegrationAmbiguousIdentity, "collaboration.read", "expected a GitHub Discussion URL or positive display ID")
	}
	if ref.Provider != providerName || ref.Kind != "discussion" {
		return invalid()
	}
	if ref.URL != "" {
		u, err := url.Parse(ref.URL)
		if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return invalid()
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] != "discussions" {
			return invalid()
		}
		n, err := strconv.Atoi(parts[3])
		if err != nil || n < 1 {
			return invalid()
		}
		if ref.DisplayID != "" && ref.DisplayID != strconv.Itoa(n) {
			return invalid()
		}
		return parts[0], parts[1], n, nil
	}
	n, err := strconv.Atoi(ref.DisplayID)
	if err != nil || n < 1 {
		return invalid()
	}
	raw, err := a.runProvider(ctx, "collaboration.read", "repo", "view", "--json", "nameWithOwner")
	if err != nil {
		return "", "", 0, err
	}
	var repository struct{ NameWithOwner string }
	if json.Unmarshal(raw, &repository) != nil {
		return invalid()
	}
	parts := strings.Split(repository.NameWithOwner, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return invalid()
	}
	return parts[0], parts[1], n, nil
}

func (a *Adapter) readDiscussion(ctx context.Context, ref application.ExternalReference) (application.CollaborationSourceSnapshot, error) {
	var snapshot application.CollaborationSourceSnapshot
	if !a.Enabled() {
		return snapshot, a.unavailable("collaboration.read")
	}
	owner, name, number, err := a.discussionLocation(ctx, ref)
	if err != nil {
		return snapshot, err
	}
	cursor := ""
	seen := map[string]bool{}
	var first *discussion
	for page := 0; page < 100; page++ {
		args := []string{"-f", "query=" + discussionQuery, "-f", "owner=" + owner, "-f", "name=" + name, "-F", "number=" + strconv.Itoa(number)}
		if cursor != "" {
			args = append(args, "-f", "after="+cursor)
		}
		var response struct {
			Data struct {
				Repository struct{ Discussion *discussion }
			}
		}
		if err := a.graphql(ctx, "collaboration.read", &response, args...); err != nil {
			return snapshot, err
		}
		item := response.Data.Repository.Discussion
		if item == nil || item.ID == "" || item.Number != number || item.URL != fmt.Sprintf("https://github.com/%s/%s/discussions/%d", owner, name, number) || (ref.OpaqueID != "" && ref.OpaqueID != item.ID) {
			return snapshot, providerError(application.IntegrationAmbiguousIdentity, "collaboration.read", "provider did not return the requested source")
		}
		if first == nil {
			first = item
			snapshot = application.CollaborationSourceSnapshot{SchemaVersion: application.IntegrationContractVersion, Source: application.ExternalReference{Provider: providerName, Kind: "discussion", OpaqueID: item.ID, DisplayID: strconv.Itoa(number), URL: item.URL}, Title: item.Title, Content: item.Body}
		} else if item.ID != first.ID || item.Title != first.Title || item.Body != first.Body || item.UpdatedAt != first.UpdatedAt {
			return application.CollaborationSourceSnapshot{}, providerError(application.IntegrationRevisionConflict, "collaboration.read", "source changed during pagination")
		}
		for _, comment := range item.Comments.Nodes {
			snapshot.Contributions = append(snapshot.Contributions, application.CollaborationContribution{Content: comment.Body})
		}
		if !item.Comments.PageInfo.HasNextPage {
			raw, _ := json.Marshal(struct {
				Snapshot  application.CollaborationSourceSnapshot
				UpdatedAt string
			}{snapshot, first.UpdatedAt})
			snapshot.Source.Revision = fmt.Sprintf("%x", sha256.Sum256(raw))
			return snapshot, nil
		}
		cursor = item.Comments.PageInfo.EndCursor
		if cursor == "" || seen[cursor] {
			break
		}
		seen[cursor] = true
	}
	return application.CollaborationSourceSnapshot{}, providerError(application.IntegrationProviderUnavailable, "collaboration.read", "incomplete provider pagination")
}

func (a *Adapter) repairDiscussion(ctx context.Context, request application.CollaborationRepairRequest) (application.CollaborationRepairEvidence, error) {
	var evidence application.CollaborationRepairEvidence
	if !a.Enabled() {
		return evidence, a.unavailable("collaboration.repair")
	}
	if request.ExpectedRevision == "" {
		return evidence, providerError(application.IntegrationRevisionConflict, "collaboration.repair", "source revision is required")
	}
	current, err := a.readDiscussion(ctx, request.Source)
	if err != nil {
		return evidence, err
	}
	if current.Source.Revision != request.ExpectedRevision {
		return evidence, providerError(application.IntegrationRevisionConflict, "collaboration.repair", "source changed since preview")
	}
	if current.Content == request.Content {
		return application.CollaborationRepairEvidence{Source: current.Source}, nil
	}
	// GitHub has no conditional updateDiscussion input. This re-read detects
	// stale previews, but cannot exclude an edit between this read and mutation.
	var response struct {
		Data struct {
			UpdateDiscussion struct{ Discussion *discussion }
		}
	}
	query := `mutation($id:ID!, $body:String!) { updateDiscussion(input:{discussionId:$id, body:$body}) { discussion { id number url title body updatedAt } } }`
	if err := a.graphql(ctx, "collaboration.repair", &response, "-f", "query="+query, "-f", "id="+current.Source.OpaqueID, "-f", "body="+request.Content); err != nil {
		var integration *application.IntegrationError
		if errors.As(err, &integration) && (integration.Class == application.IntegrationUnauthenticated || integration.Class == application.IntegrationUnauthorized) {
			return evidence, err
		}
		return evidence, &application.IntegrationError{Class: application.IntegrationPartialFailure, Provider: providerName, Operation: "collaboration.repair", Message: "mutation outcome is unknown; re-read the source before retrying", Err: err}
	}
	item := response.Data.UpdateDiscussion.Discussion
	if item == nil || item.ID != current.Source.OpaqueID || item.Body != request.Content {
		return evidence, providerError(application.IntegrationPartialFailure, "collaboration.repair", "mutation outcome requires a fresh source read")
	}
	evidence.Source = current.Source
	// A fresh read supplies a revision including contributions; do not invent one
	// from a mutation response that omits comments.
	evidence.Source.Revision = ""
	evidence.Changed = true
	return evidence, nil
}
