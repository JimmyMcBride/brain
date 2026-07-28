package planning

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var artifactIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type ArtifactID string

func (id ArtifactID) Validate() error {
	value := string(id)
	if value == "" || strings.TrimSpace(value) != value || !artifactIDPattern.MatchString(value) {
		return domainError(ErrInvalidIdentifier, []ArtifactID{id}, "must be a canonical lowercase slug")
	}
	return nil
}

type ArtifactKind string

const (
	ArtifactBrainstorm ArtifactKind = "brainstorm"
	ArtifactSpec       ArtifactKind = "spec"
	ArtifactInitiative ArtifactKind = "initiative"
	ArtifactRoadmap    ArtifactKind = "roadmap"
)

func (kind ArtifactKind) valid() bool {
	switch kind {
	case ArtifactBrainstorm, ArtifactSpec, ArtifactInitiative, ArtifactRoadmap:
		return true
	default:
		return false
	}
}

type ArtifactRef struct {
	Kind ArtifactKind
	ID   ArtifactID
}

func (ref ArtifactRef) Validate() error {
	if !ref.Kind.valid() {
		return domainError(ErrInvalidArtifact, []ArtifactID{ref.ID}, fmt.Sprintf("unknown artifact kind %q", ref.Kind))
	}
	return ref.ID.Validate()
}

type SourceReference struct {
	URI        string
	Revision   string
	ObservedAt time.Time
	Purpose    string
}

func validateSources(owner ArtifactID, sources []SourceReference) []Finding {
	var findings []Finding
	for _, source := range sources {
		if strings.TrimSpace(source.URI) == "" || strings.TrimSpace(source.Purpose) == "" {
			findings = append(findings, Finding{
				Code:     ErrInvalidArtifact,
				Severity: SeverityError,
				Message:  "source references require a URI and purpose",
				Artifact: ArtifactRef{ID: owner},
			})
		}
	}
	return findings
}

type OwnershipMode string

const (
	OwnershipLocal  OwnershipMode = "local"
	OwnershipGitHub OwnershipMode = "github"
	OwnershipHybrid OwnershipMode = "hybrid"
)

func (mode OwnershipMode) Validate() error {
	switch mode {
	case OwnershipLocal, OwnershipGitHub, OwnershipHybrid:
		return nil
	default:
		return domainError(ErrInvalidOwnership, nil, fmt.Sprintf("unsupported ownership mode %q", mode))
	}
}

type SpecStatus string

const (
	SpecDraft        SpecStatus = "draft"
	SpecApproved     SpecStatus = "approved"
	SpecImplementing SpecStatus = "implementing"
	SpecDone         SpecStatus = "done"
)

func (status SpecStatus) valid() bool {
	switch status {
	case SpecDraft, SpecApproved, SpecImplementing, SpecDone:
		return true
	default:
		return false
	}
}

type ApprovalState string

const (
	ApprovalPending  ApprovalState = "pending"
	ApprovalApproved ApprovalState = "approved"
	ApprovalRejected ApprovalState = "rejected"
)

type Approval struct {
	State  ApprovalState
	Reason string
}

func (approval Approval) valid() bool {
	switch approval.State {
	case ApprovalPending, ApprovalApproved, ApprovalRejected:
		return true
	default:
		return false
	}
}

type Brainstorm struct {
	ID      ArtifactID
	Title   string
	Summary string
	Sources []SourceReference
}

type Spec struct {
	ID                  ArtifactID
	Title               string
	Status              SpecStatus
	Approval            Approval
	Dependencies        []ArtifactID
	Verification        []string
	Initiative          *ArtifactID
	Sources             []SourceReference
	UnresolvedQuestions []string
}

type Initiative struct {
	ID      ArtifactID
	Title   string
	Summary string
	Specs   []ArtifactRef
	Sources []SourceReference
}

type Roadmap struct {
	ID      ArtifactID
	Title   string
	Entries []RoadmapEntry
	Sources []SourceReference
}

type RoadmapEntry struct {
	Ref  ArtifactRef
	Done bool
}

type FindingSeverity string

const (
	SeverityInfo    FindingSeverity = "info"
	SeverityWarning FindingSeverity = "warning"
	SeverityError   FindingSeverity = "error"
)

type Finding struct {
	Code     ErrorCode
	Severity FindingSeverity
	Message  string
	Artifact ArtifactRef
}

func ValidateBrainstorm(brainstorm Brainstorm) []Finding {
	findings := validateIdentity(ArtifactBrainstorm, brainstorm.ID, brainstorm.Title)
	return append(findings, validateSources(brainstorm.ID, brainstorm.Sources)...)
}

func ValidateSpec(spec Spec) []Finding {
	findings := validateIdentity(ArtifactSpec, spec.ID, spec.Title)
	if !spec.Status.valid() {
		findings = append(findings, invalidArtifactFinding(ArtifactSpec, spec.ID, "unknown spec status"))
	}
	if !spec.Approval.valid() {
		findings = append(findings, invalidArtifactFinding(ArtifactSpec, spec.ID, "unknown approval state"))
	}
	if spec.Status != SpecDraft && spec.Approval.State != ApprovalApproved {
		findings = append(findings, Finding{
			Code:     ErrApprovalRequired,
			Severity: SeverityError,
			Message:  "non-draft specs require Planning approval",
			Artifact: ArtifactRef{Kind: ArtifactSpec, ID: spec.ID},
		})
	}
	if len(trimmedStrings(spec.Verification)) == 0 {
		findings = append(findings, invalidArtifactFinding(ArtifactSpec, spec.ID, "verification is required"))
	}

	seenDependencies := map[ArtifactID]struct{}{}
	for _, dependency := range spec.Dependencies {
		switch {
		case dependency.Validate() != nil:
			findings = append(findings, invalidArtifactFinding(ArtifactSpec, spec.ID, "dependency has an invalid identifier"))
		case dependency == spec.ID:
			findings = append(findings, invalidArtifactFinding(ArtifactSpec, spec.ID, "spec cannot depend on itself"))
		default:
			if _, exists := seenDependencies[dependency]; exists {
				findings = append(findings, invalidArtifactFinding(ArtifactSpec, spec.ID, "dependency is duplicated"))
			}
			seenDependencies[dependency] = struct{}{}
		}
	}
	if spec.Initiative != nil {
		if err := spec.Initiative.Validate(); err != nil {
			findings = append(findings, invalidArtifactFinding(ArtifactSpec, spec.ID, "initiative has an invalid identifier"))
		}
	}
	return append(findings, validateSources(spec.ID, spec.Sources)...)
}

func ValidateInitiative(initiative Initiative) []Finding {
	findings := validateIdentity(ArtifactInitiative, initiative.ID, initiative.Title)
	seen := map[ArtifactID]struct{}{}
	for _, ref := range initiative.Specs {
		if ref.Kind != ArtifactSpec || ref.Validate() != nil {
			findings = append(findings, invalidArtifactFinding(ArtifactInitiative, initiative.ID, "initiative entries must reference valid specs"))
			continue
		}
		if _, exists := seen[ref.ID]; exists {
			findings = append(findings, invalidArtifactFinding(ArtifactInitiative, initiative.ID, "spec reference is duplicated"))
		}
		seen[ref.ID] = struct{}{}
	}
	return append(findings, validateSources(initiative.ID, initiative.Sources)...)
}

func ValidateRoadmap(roadmap Roadmap) []Finding {
	findings := validateIdentity(ArtifactRoadmap, roadmap.ID, roadmap.Title)
	for _, entry := range roadmap.Entries {
		if entry.Ref.Validate() != nil || (entry.Ref.Kind != ArtifactSpec && entry.Ref.Kind != ArtifactInitiative) {
			findings = append(findings, invalidArtifactFinding(ArtifactRoadmap, roadmap.ID, "roadmap entries must reference valid specs or initiatives"))
		}
	}
	return append(findings, validateSources(roadmap.ID, roadmap.Sources)...)
}

func validateIdentity(kind ArtifactKind, id ArtifactID, title string) []Finding {
	var findings []Finding
	if err := id.Validate(); err != nil {
		findings = append(findings, Finding{
			Code:     ErrInvalidIdentifier,
			Severity: SeverityError,
			Message:  "artifact identifier must be a canonical lowercase slug",
			Artifact: ArtifactRef{Kind: kind, ID: id},
		})
	}
	if strings.TrimSpace(title) == "" {
		findings = append(findings, invalidArtifactFinding(kind, id, "title is required"))
	}
	return findings
}

func invalidArtifactFinding(kind ArtifactKind, id ArtifactID, message string) Finding {
	return Finding{
		Code:     ErrInvalidArtifact,
		Severity: SeverityError,
		Message:  message,
		Artifact: ArtifactRef{Kind: kind, ID: id},
	}
}

func trimmedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func hasErrorFindings(findings []Finding) bool {
	for _, finding := range findings {
		if finding.Severity == SeverityError {
			return true
		}
	}
	return false
}

func cloneSpec(spec Spec) Spec {
	copy := spec
	copy.Dependencies = append([]ArtifactID(nil), spec.Dependencies...)
	copy.Verification = append([]string(nil), spec.Verification...)
	copy.Sources = append([]SourceReference(nil), spec.Sources...)
	copy.UnresolvedQuestions = append([]string(nil), spec.UnresolvedQuestions...)
	if spec.Initiative != nil {
		initiative := *spec.Initiative
		copy.Initiative = &initiative
	}
	return copy
}
