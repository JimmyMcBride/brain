package planning

import (
	"fmt"
	"slices"
)

// QueueState identifies a spec's computed position in an execution queue.
type QueueState string

const (
	// QueueReady means the spec may become current.
	QueueReady QueueState = "ready"
	// QueueCurrent means the spec is actively implementing.
	QueueCurrent QueueState = "current"
	// QueueBlocked means the spec cannot execute yet.
	QueueBlocked QueueState = "blocked"
	// QueueDone means the spec completed execution.
	QueueDone QueueState = "done"
)

// QueueEntry is one computed spec queue record.
type QueueEntry struct {
	SpecID  ArtifactID
	State   QueueState
	Reasons []string
}

// Queue is a deterministic ordered view over specs.
type Queue struct {
	Entries []QueueEntry
}

// BuildQueue returns a deterministic computed view over specs.
func BuildQueue(specs []Spec, current *ArtifactID) (Queue, error) {
	order, err := OrderSpecs(specs)
	if err != nil {
		return Queue{}, err
	}

	byID := make(map[ArtifactID]Spec, len(specs))
	var implementing []ArtifactID
	for _, spec := range specs {
		byID[spec.ID] = cloneSpec(spec)
		if spec.Status == SpecImplementing {
			implementing = append(implementing, spec.ID)
		}
	}
	slices.Sort(implementing)
	if len(implementing) > 1 {
		return Queue{}, domainError(ErrInvalidExecution, implementing, "only one spec can be current")
	}

	var currentID ArtifactID
	switch {
	case current != nil:
		currentID = *current
		spec, exists := byID[currentID]
		if !exists {
			return Queue{}, domainError(ErrInvalidExecution, []ArtifactID{currentID}, "current spec is not in the queue")
		}
		if spec.Status != SpecImplementing {
			return Queue{}, domainError(ErrInvalidExecution, []ArtifactID{currentID}, "current spec must be implementing")
		}
		if len(implementing) == 1 && implementing[0] != currentID {
			return Queue{}, domainError(ErrInvalidExecution, implementing, "implementing spec does not match current spec")
		}
	case len(implementing) == 1:
		currentID = implementing[0]
	}

	queue := Queue{Entries: make([]QueueEntry, 0, len(order))}
	for _, id := range order {
		spec := byID[id]
		var blockers []ArtifactID
		for _, dependency := range spec.Dependencies {
			if byID[dependency].Status != SpecDone {
				blockers = append(blockers, dependency)
			}
		}
		blockers = uniqueSortedIDs(blockers)

		if id == currentID && len(blockers) > 0 {
			return Queue{}, domainError(ErrInvalidExecution, []ArtifactID{id}, fmt.Sprintf("current spec is blocked by %v", blockers))
		}

		readiness := EvaluateReadiness(spec, ValidateSpec(spec), blockers)
		entry := QueueEntry{SpecID: id, Reasons: append([]string(nil), readiness.Reasons...)}
		switch {
		case readiness.State == ReadinessDone:
			entry.State = QueueDone
		case id == currentID:
			entry.State = QueueCurrent
		case readiness.State == ReadinessReady:
			entry.State = QueueReady
		default:
			entry.State = QueueBlocked
		}
		queue.Entries = append(queue.Entries, entry)
	}
	return queue, nil
}
