package planning

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var artifactIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ArtifactID is a canonical lowercase slug identifying a Planning artifact.
type ArtifactID string

// Validate reports whether the identifier is a canonical lowercase slug.
func (id ArtifactID) Validate() error {
	value := string(id)
	if value == "" || strings.TrimSpace(value) != value || !artifactIDPattern.MatchString(value) {
		return domainError(ErrInvalidIdentifier, []ArtifactID{id}, "must be a canonical lowercase slug")
	}
	return nil
}

// ArtifactKind identifies a supported Planning aggregate.
type ArtifactKind string

const (
	// ArtifactBrainstorm identifies a brainstorm.
	ArtifactBrainstorm ArtifactKind = "brainstorm"
	// ArtifactSpec identifies a spec.
	ArtifactSpec ArtifactKind = "spec"
	// ArtifactInitiative identifies an initiative.
	ArtifactInitiative ArtifactKind = "initiative"
	// ArtifactRoadmap identifies a roadmap.
	ArtifactRoadmap ArtifactKind = "roadmap"
)

func (kind ArtifactKind) valid() bool {
	switch kind {
	case ArtifactBrainstorm, ArtifactSpec, ArtifactInitiative, ArtifactRoadmap:
		return true
	default:
		return false
	}
}

// ArtifactRef identifies an artifact by kind and ID.
type ArtifactRef struct {
	Kind ArtifactKind
	ID   ArtifactID
}

// Validate reports whether the reference has a supported kind and valid ID.
func (ref ArtifactRef) Validate() error {
	if !ref.Kind.valid() {
		return domainError(ErrInvalidArtifact, []ArtifactID{ref.ID}, fmt.Sprintf("unknown artifact kind %q", ref.Kind))
	}
	return ref.ID.Validate()
}

// SourceReference records provenance for a Planning artifact.
type SourceReference struct {
	URI        string
	Revision   string
	ObservedAt time.Time
	Purpose    string
}

func validateSources(kind ArtifactKind, owner ArtifactID, sources []SourceReference) []Finding {
	var findings []Finding
	for _, source := range sources {
		if strings.TrimSpace(source.URI) == "" || strings.TrimSpace(source.Purpose) == "" {
			findings = append(findings, Finding{
				Code:     ErrInvalidArtifact,
				Severity: SeverityError,
				Message:  "source references require a URI and purpose",
				Artifact: ArtifactRef{Kind: kind, ID: owner},
			})
		}
	}
	return findings
}

// OwnershipMode identifies the supported source-of-truth arrangement.
type OwnershipMode string

const (
	// OwnershipLocal keeps Planning artifacts in the local workspace.
	OwnershipLocal OwnershipMode = "local"
	// OwnershipGitHub keeps the relevant Planning layer in GitHub.
	OwnershipGitHub OwnershipMode = "github"
	// OwnershipHybrid splits ownership explicitly between local and GitHub layers.
	OwnershipHybrid OwnershipMode = "hybrid"
)

// Validate reports whether the ownership mode is supported.
func (mode OwnershipMode) Validate() error {
	switch mode {
	case OwnershipLocal, OwnershipGitHub, OwnershipHybrid:
		return nil
	default:
		return domainError(ErrInvalidOwnership, nil, fmt.Sprintf("unsupported ownership mode %q", mode))
	}
}

// SpecStatus identifies a spec lifecycle state.
type SpecStatus string

const (
	// SpecDraft is editable and not approved for execution.
	SpecDraft SpecStatus = "draft"
	// SpecApproved is approved and ready for execution when otherwise unblocked.
	SpecApproved SpecStatus = "approved"
	// SpecImplementing has an active execution plan.
	SpecImplementing SpecStatus = "implementing"
	// SpecDone has completed execution.
	SpecDone SpecStatus = "done"
)

func (status SpecStatus) valid() bool {
	switch status {
	case SpecDraft, SpecApproved, SpecImplementing, SpecDone:
		return true
	default:
		return false
	}
}

// ApprovalState identifies the current Planning approval decision.
type ApprovalState string

const (
	// ApprovalPending means no approval decision has been made.
	ApprovalPending ApprovalState = "pending"
	// ApprovalApproved permits the approved transition or execution.
	ApprovalApproved ApprovalState = "approved"
	// ApprovalRejected records an explicit rejection.
	ApprovalRejected ApprovalState = "rejected"
)

// Approval records a Planning approval decision and optional reason.
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

// Brainstorm is a storage-neutral discovery artifact.
type Brainstorm struct {
	ID      ArtifactID
	Title   string
	Summary string
	Sources []SourceReference
}

// Spec is a canonical storage-neutral execution contract.
type Spec struct {
	ID                  ArtifactID
	Title               string
	Status              SpecStatus
	Approval            Approval
	Dependencies        []ArtifactID
	Verification        []string
	Initiative          *ArtifactID
	ExecutionID         *ArtifactID
	Sources             []SourceReference
	UnresolvedQuestions []string
}

// Initiative groups related specs without replacing them as canonical contracts.
type Initiative struct {
	ID      ArtifactID
	Title   string
	Summary string
	Specs   []ArtifactRef
	Sources []SourceReference
}

// Roadmap is an ordered view of spec and initiative intent.
type Roadmap struct {
	ID      ArtifactID
	Title   string
	Entries []RoadmapEntry
	Sources []SourceReference
}

// RoadmapEntry references one roadmap item and its completion state.
type RoadmapEntry struct {
	Ref  ArtifactRef
	Done bool
}

// FindingSeverity identifies the impact of a domain finding.
type FindingSeverity string

const (
	// SeverityInfo reports contextual information.
	SeverityInfo FindingSeverity = "info"
	// SeverityWarning reports a non-blocking concern.
	SeverityWarning FindingSeverity = "warning"
	// SeverityError reports a domain validation failure.
	SeverityError FindingSeverity = "error"
)

// Finding describes a stable domain validation result.
type Finding struct {
	Code     ErrorCode
	Severity FindingSeverity
	Message  string
	Artifact ArtifactRef
}

// ValidateBrainstorm returns deterministic findings for a brainstorm.
func ValidateBrainstorm(brainstorm Brainstorm) []Finding {
	findings := validateIdentity(ArtifactBrainstorm, brainstorm.ID, brainstorm.Title)
	return append(findings, validateSources(ArtifactBrainstorm, brainstorm.ID, brainstorm.Sources)...)
}

// ValidateSpec returns deterministic findings for a spec.
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
	if spec.Status == SpecImplementing {
		if spec.ExecutionID == nil || spec.ExecutionID.Validate() != nil {
			findings = append(findings, invalidArtifactFinding(ArtifactSpec, spec.ID, "implementing spec requires a valid execution identifier"))
		}
	} else if spec.Status != SpecDone && spec.ExecutionID != nil {
		findings = append(findings, invalidArtifactFinding(ArtifactSpec, spec.ID, "execution identifier is only valid while implementing or done"))
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
	return append(findings, validateSources(ArtifactSpec, spec.ID, spec.Sources)...)
}

// ValidateInitiative returns deterministic findings for an initiative.
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
	return append(findings, validateSources(ArtifactInitiative, initiative.ID, initiative.Sources)...)
}

// ValidateRoadmap returns deterministic findings for a roadmap.
func ValidateRoadmap(roadmap Roadmap) []Finding {
	findings := validateIdentity(ArtifactRoadmap, roadmap.ID, roadmap.Title)
	seen := map[ArtifactRef]struct{}{}
	for _, entry := range roadmap.Entries {
		if entry.Ref.Validate() != nil || (entry.Ref.Kind != ArtifactSpec && entry.Ref.Kind != ArtifactInitiative) {
			findings = append(findings, invalidArtifactFinding(ArtifactRoadmap, roadmap.ID, "roadmap entries must reference valid specs or initiatives"))
			continue
		}
		if _, exists := seen[entry.Ref]; exists {
			findings = append(findings, invalidArtifactFinding(ArtifactRoadmap, roadmap.ID, "roadmap entry is duplicated"))
		}
		seen[entry.Ref] = struct{}{}
	}
	return append(findings, validateSources(ArtifactRoadmap, roadmap.ID, roadmap.Sources)...)
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
	if spec.ExecutionID != nil {
		executionID := *spec.ExecutionID
		copy.ExecutionID = &executionID
	}
	return copy
}
