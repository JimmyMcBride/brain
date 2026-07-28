package modules

import (
	"fmt"
	"sort"
)

type Factory func() Module

type Registration struct {
	Descriptor Descriptor
	Factory    Factory
}

type Registry struct {
	registrations map[string]Registration
	ids           []string
}

func NewRegistry(registrations []Registration) (*Registry, error) {
	registry := &Registry{registrations: make(map[string]Registration, len(registrations))}
	for _, registration := range registrations {
		if err := registration.Descriptor.Validate(); err != nil {
			return nil, err
		}
		id := registration.Descriptor.ID
		if registration.Factory == nil {
			return nil, fmt.Errorf("module %s has no factory", id)
		}
		if _, exists := registry.registrations[id]; exists {
			return nil, fmt.Errorf("duplicate module registration %q", id)
		}
		registration.Descriptor.Capabilities = normalizedStrings(registration.Descriptor.Capabilities)
		registration.Descriptor.Permissions = normalizedStrings(registration.Descriptor.Permissions)
		registry.registrations[id] = registration
		registry.ids = append(registry.ids, id)
	}
	sort.Strings(registry.ids)
	return registry, nil
}

func (r *Registry) Lookup(id string) (Registration, bool) {
	if r == nil {
		return Registration{}, false
	}
	registration, ok := r.registrations[id]
	return registration, ok
}

func (r *Registry) IDs() []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.ids...)
}
