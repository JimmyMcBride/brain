package project

import "fmt"

// Paradigm defines the vocabulary for a project management style.
type Paradigm struct {
	Name            string `json:"name" yaml:"name"`
	ContainerType   string `json:"container_type" yaml:"container_type"`
	ContainerPlural string `json:"container_plural" yaml:"container_plural"`
	ItemType        string `json:"item_type" yaml:"item_type"`
	ItemPlural      string `json:"item_plural" yaml:"item_plural"`
}

var paradigms = map[string]Paradigm{
	"epics": {
		Name:            "epics",
		ContainerType:   "epic",
		ContainerPlural: "epics",
		ItemType:        "story",
		ItemPlural:      "stories",
	},
	"milestones": {
		Name:            "milestones",
		ContainerType:   "milestone",
		ContainerPlural: "milestones",
		ItemType:        "task",
		ItemPlural:      "tasks",
	},
	"cycles": {
		Name:            "cycles",
		ContainerType:   "cycle",
		ContainerPlural: "cycles",
		ItemType:        "task",
		ItemPlural:      "tasks",
	},
}

// LookupParadigm returns the paradigm definition for the given name.
func LookupParadigm(name string) (*Paradigm, error) {
	p, ok := paradigms[name]
	if !ok {
		return nil, fmt.Errorf("unknown paradigm %q (available: epics, milestones, cycles)", name)
	}
	return &p, nil
}

// ParadigmNames returns the list of available paradigm names.
func ParadigmNames() []string {
	return []string{"epics", "milestones", "cycles"}
}
