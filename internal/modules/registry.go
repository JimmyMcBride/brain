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
	commandOwners map[string]string
	eventOwners   map[string]string
	ids           []string
}

func NewRegistry(registrations []Registration) (*Registry, error) {
	registry := &Registry{
		registrations: make(map[string]Registration, len(registrations)),
		commandOwners: map[string]string{},
		eventOwners:   map[string]string{},
	}
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
		registration.Descriptor.Commands = normalizedStrings(registration.Descriptor.Commands)
		registration.Descriptor.Events = normalizedStrings(registration.Descriptor.Events)
		for _, command := range registration.Descriptor.Commands {
			if owner, exists := registry.commandOwners[command]; exists {
				return nil, fmt.Errorf("command %q is declared by both %s and %s", command, owner, id)
			}
			registry.commandOwners[command] = id
		}
		for _, event := range registration.Descriptor.Events {
			if owner, exists := registry.eventOwners[event]; exists {
				return nil, fmt.Errorf("event %q is declared by both %s and %s", event, owner, id)
			}
			registry.eventOwners[event] = id
		}
		registry.registrations[id] = registration
		registry.ids = append(registry.ids, id)
	}
	sort.Strings(registry.ids)
	return registry, nil
}

func (r *Registry) LookupCommand(name string) (Registration, bool) {
	if r == nil {
		return Registration{}, false
	}
	id, ok := r.commandOwners[name]
	if !ok {
		return Registration{}, false
	}
	return r.Lookup(id)
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
