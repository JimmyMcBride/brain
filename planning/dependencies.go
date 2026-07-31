package planning

import (
	"fmt"
	"slices"
	"sort"
)

// OrderSpecs validates a dependency graph and returns dependencies before their
// dependents with lexical tie-breaking.
func OrderSpecs(specs []Spec) ([]ArtifactID, error) {
	byID := make(map[ArtifactID]Spec, len(specs))
	for _, spec := range specs {
		if err := spec.ID.Validate(); err != nil {
			return nil, err
		}
		if _, exists := byID[spec.ID]; exists {
			return nil, domainError(ErrInvalidArtifact, []ArtifactID{spec.ID}, "spec identifier is duplicated")
		}
		byID[spec.ID] = cloneSpec(spec)
	}

	var missing []ArtifactID
	var missingReasons []string
	for _, spec := range byID {
		for _, dependency := range spec.Dependencies {
			if _, exists := byID[dependency]; !exists {
				missing = append(missing, dependency)
				missingReasons = append(missingReasons, fmt.Sprintf("%s depends on missing %s", spec.ID, dependency))
			}
		}
	}
	if len(missing) > 0 {
		missing = uniqueSortedIDs(missing)
		sort.Strings(missingReasons)
		return nil, domainError(ErrDependencyMissing, missing, missingReasons...)
	}

	indegree := make(map[ArtifactID]int, len(byID))
	dependents := make(map[ArtifactID][]ArtifactID, len(byID))
	for id := range byID {
		indegree[id] = 0
	}
	for _, spec := range byID {
		seen := map[ArtifactID]struct{}{}
		for _, dependency := range spec.Dependencies {
			if _, duplicate := seen[dependency]; duplicate {
				continue
			}
			seen[dependency] = struct{}{}
			indegree[spec.ID]++
			dependents[dependency] = append(dependents[dependency], spec.ID)
		}
	}
	for id := range dependents {
		slices.Sort(dependents[id])
	}

	var ready []ArtifactID
	for id, count := range indegree {
		if count == 0 {
			ready = append(ready, id)
		}
	}
	slices.Sort(ready)

	order := make([]ArtifactID, 0, len(byID))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		for _, dependent := range dependents[id] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				ready = insertSortedID(ready, dependent)
			}
		}
	}
	if len(order) != len(byID) {
		cycles := dependencyCycleMembers(byID)
		return nil, domainError(ErrDependencyCycle, cycles, "spec dependency graph contains a cycle")
	}
	return order, nil
}

func dependencyCycleMembers(specs map[ArtifactID]Spec) []ArtifactID {
	var (
		index      int
		stack      []ArtifactID
		onStack    = map[ArtifactID]bool{}
		indices    = map[ArtifactID]int{}
		lowLink    = map[ArtifactID]int{}
		cycleNodes []ArtifactID
	)

	ids := make([]ArtifactID, 0, len(specs))
	for id := range specs {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	var visit func(ArtifactID)
	visit = func(id ArtifactID) {
		index++
		indices[id] = index
		lowLink[id] = index
		stack = append(stack, id)
		onStack[id] = true

		dependencies := append([]ArtifactID(nil), specs[id].Dependencies...)
		slices.Sort(dependencies)
		for _, dependency := range dependencies {
			if indices[dependency] == 0 {
				visit(dependency)
				lowLink[id] = min(lowLink[id], lowLink[dependency])
			} else if onStack[dependency] {
				lowLink[id] = min(lowLink[id], indices[dependency])
			}
		}
		if lowLink[id] != indices[id] {
			return
		}

		var component []ArtifactID
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == id {
				break
			}
		}
		if len(component) > 1 || hasDependency(specs[id], id) {
			cycleNodes = append(cycleNodes, component...)
		}
	}

	for _, id := range ids {
		if indices[id] == 0 {
			visit(id)
		}
	}
	return uniqueSortedIDs(cycleNodes)
}

func hasDependency(spec Spec, dependency ArtifactID) bool {
	return slices.Contains(spec.Dependencies, dependency)
}

func uniqueSortedIDs(ids []ArtifactID) []ArtifactID {
	seen := map[ArtifactID]struct{}{}
	out := make([]ArtifactID, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

func insertSortedID(ids []ArtifactID, id ArtifactID) []ArtifactID {
	index, _ := slices.BinarySearch(ids, id)
	ids = append(ids, "")
	copy(ids[index+1:], ids[index:])
	ids[index] = id
	return ids
}
