package testmodule

import (
	"context"
	"errors"
	"sync"

	"brain/internal/modules"
)

const ID = "dev.brain.test"

type Counters struct {
	mu          sync.Mutex
	Factories   int
	Validations int
	Initializes int
	HealthCalls int
}

func (c *Counters) Snapshot() Counters {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Counters{
		Factories:   c.Factories,
		Validations: c.Validations,
		Initializes: c.Initializes,
		HealthCalls: c.HealthCalls,
	}
}

type Options struct {
	ValidationError error
	InitializeError error
	Health          modules.Health
	WriteConfig     bool
}

func Registration(counters *Counters, opts Options) modules.Registration {
	if counters == nil {
		counters = &Counters{}
	}
	return modules.Registration{
		Descriptor: modules.Descriptor{
			ID:            ID,
			Name:          "Test Module",
			Version:       "1.0.0",
			BrainAPIMajor: modules.BrainAPIMajor,
			ConfigVersion: 1,
			Capabilities:  []string{"test.inspect"},
			Permissions:   []string{"test.read"},
		},
		Factory: func() modules.Module {
			counters.mu.Lock()
			counters.Factories++
			counters.mu.Unlock()
			return &module{counters: counters, opts: opts}
		},
	}
}

type module struct {
	counters *Counters
	opts     Options
}

func (m *module) Validate(_ context.Context, _ modules.ModuleContext, config modules.Config) error {
	m.counters.mu.Lock()
	m.counters.Validations++
	m.counters.mu.Unlock()
	if m.opts.ValidationError != nil {
		return m.opts.ValidationError
	}
	if m.opts.WriteConfig {
		config["validated"] = true
	}
	if invalid, _ := config["invalid"].(bool); invalid {
		return errors.New("invalid test configuration")
	}
	return nil
}

func (m *module) Initialize(_ context.Context, _ modules.ModuleContext, _ modules.Config) error {
	m.counters.mu.Lock()
	m.counters.Initializes++
	m.counters.mu.Unlock()
	return m.opts.InitializeError
}

func (m *module) Health(_ context.Context, _ modules.ModuleContext) modules.Health {
	m.counters.mu.Lock()
	m.counters.HealthCalls++
	m.counters.mu.Unlock()
	if m.opts.Health.Status != "" {
		return m.opts.Health
	}
	return modules.Health{Status: modules.HealthHealthy}
}
