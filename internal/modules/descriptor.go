package modules

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const BrainAPIMajor = 1

var (
	moduleIDPattern    = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)+$`)
	semverPattern      = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
	commandNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	eventNamePattern   = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)+$`)
)

type Descriptor struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	BrainAPIMajor int      `json:"brain_api_major"`
	ConfigVersion int      `json:"config_version"`
	Capabilities  []string `json:"capabilities,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
	Commands      []string `json:"commands,omitempty"`
	Events        []string `json:"events,omitempty"`
}

func (d Descriptor) Validate() error {
	if !moduleIDPattern.MatchString(d.ID) {
		return fmt.Errorf("invalid module ID %q: expected a reverse-domain identifier", d.ID)
	}
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("module %s has an empty name", d.ID)
	}
	if !semverPattern.MatchString(d.Version) {
		return fmt.Errorf("module %s has invalid semantic version %q", d.ID, d.Version)
	}
	if d.BrainAPIMajor != BrainAPIMajor {
		return &Failure{
			Code:    FailureAPIIncompatible,
			Message: fmt.Sprintf("module %s requires Brain API major %d, supported major is %d", d.ID, d.BrainAPIMajor, BrainAPIMajor),
		}
	}
	if d.ConfigVersion < 1 {
		return fmt.Errorf("module %s has invalid config version %d", d.ID, d.ConfigVersion)
	}
	if err := validateDeclarations(d.ID, "capability", d.Capabilities); err != nil {
		return err
	}
	if err := validateDeclarations(d.ID, "permission", d.Permissions); err != nil {
		return err
	}
	if err := validateDeclarations(d.ID, "command", d.Commands); err != nil {
		return err
	}
	for _, command := range d.Commands {
		if !commandNamePattern.MatchString(command) {
			return fmt.Errorf("module %s declares invalid command %q", d.ID, command)
		}
	}
	if err := validateDeclarations(d.ID, "event", d.Events); err != nil {
		return err
	}
	for _, event := range d.Events {
		if !eventNamePattern.MatchString(event) {
			return fmt.Errorf("module %s declares invalid event %q", d.ID, event)
		}
	}
	return nil
}

func validateDeclarations(moduleID, kind string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("module %s declares an empty %s", moduleID, kind)
		}
		if trimmed != value {
			return fmt.Errorf("module %s declares %s %q with surrounding whitespace", moduleID, kind, value)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("module %s declares duplicate %s %q", moduleID, kind, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func normalizedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
