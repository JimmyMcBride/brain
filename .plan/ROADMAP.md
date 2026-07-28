# Roadmap: brain

Created: 2026-07-27T21:38:28Z

## Overview

Planning enters Brain only after module contracts are proven. The standalone Plan
implementation remains the migration reference.

## Phase 0: Architecture and Inventory

Goal: Define reviewed product, module, domain, storage, integration, security, and
migration contracts without porting Planning code.

- [x] Define optional Planning product boundary
- [x] Inventory standalone Plan behavior and persistence
- [x] Record ADRs for module/runtime/storage/integration decisions
- [x] Define `local`/`github`/`hybrid` migration, Linear exclusion, CLI compatibility, and cloud direction
- [x] Record Brain knowledge-loop and permission boundaries

## Phase 1: Minimal Internal Module Framework

Goal: Prove registration and lifecycle with no Planning domain migration.

- [x] Approve the [Minimal Internal Module Framework](./specs/minimal-internal-module-framework.md) spec
- [x] Module descriptor and registry
- [x] Project enable/disable state
- [x] Namespaced configuration
- [x] Module permissions
- [x] Minimal lifecycle, capability declarations, and health
- [x] One test-only reference module with tests

## Phase 2–4: Local Planning and Compatibility

Goal: Extract storage-neutral Planning behavior, ship local `brain plan`, and keep
the `plan` command as a shared compatibility wrapper.

- [x] [Planning domain extraction](./specs/planning-domain-extraction.md)
- [ ] Local `.plan/` adapter and migration fixtures
- [ ] Native Planning command group and permissions/events
- [ ] Plan CLI mapping and compatibility tests

## Phase 5–7: Integrations and Knowledge Loop

Goal: Move GitHub behind adapters, keep Linear outside Brain Planning, and connect
Planning outcomes to review-first Brain memory proposals.

- [ ] GitHub adapter transition
- [ ] Reject legacy Linear workspaces with standalone Plan migration guidance
- [ ] Brain context sources and Planning context packets
- [ ] Completion-to-memory proposals

## Phase 8–11: Cloud, Hybrid, UI, Community Protocol

Goal: Add optional Brain Cloud Planning, layer-owned hybrid sync, unified web
surfaces, then external-process community modules.

## Ordering Notes

- Do not migrate Planning domain code before Phase 1 review.
- Ship local before cloud and hybrid.
- Validate official compiled modules before designing community protocol.
- Preserve confirmation, idempotency, provenance, and disabled-module behavior at
  every phase.

## Parking Lot

- external-process transport choice
- module distribution/marketplace
- community frontend sandboxing
- long-term `.plan/` relocation
- exact GitHub source-mode deprecation date
