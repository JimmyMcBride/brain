# Module Security

## Status

Security contract for architecture and Phase 1. Process sandboxing, signing, and
distribution are future capabilities, not current claims.

Modules are executable software. A manifest is a declaration, not enforcement.
Internal official modules run in Brain's process and therefore share its OS
authority; their boundary is enforced initially through narrow interfaces,
dependency tests, permission checks, and audit behavior.

## Permission Model

Registration, enablement, and authorization are separate:

1. Module declares requested capabilities and permissions.
2. Brain validates compatibility and policy.
3. User or organization approves grants.
4. Each operation checks the grant.
5. Mutations produce attributable audit records.

Planning permissions:

```text
planning.read
planning.brainstorm
planning.spec.create
planning.spec.edit
planning.spec.approve
planning.execute
planning.admin
planning.integration.publish
```

Memory permissions remain independent:

```text
memory.read
memory.propose
memory.approve
memory.edit
```

`planning.admin` does not grant memory, secret, network, repository, or Core
administration rights.

## Agent Tools

Every tool declaration includes stable identity, input/output schema, required
permissions, mutation classification, confirmation policy, audit behavior, and
runtime availability. Brain rejects undeclared tools and schema collisions.

Read-only, reversible mutation, consequential mutation, and destructive mutation
must be distinguishable. A module cannot downgrade a Core confirmation policy.

## Data and Provenance

- Core project and user identity are authoritative.
- Modules access storage through registered, scoped adapters.
- Module identity/version is recorded on context, search, events, jobs, and audit.
- Secret values are never placed in manifests, logs, events, or context packets.
- Disabled modules cannot inject context, run jobs, or use dormant grants.
- Disabling preserves data; removal needs a separate explicit operation.

## Community Module Requirements

Before external community modules are supported, design and implement:

- publisher identity and organization allowlists
- checksums, signed releases, version pinning, and revocation
- install and upgrade review
- declared/mediated filesystem, network, and secret access
- compatibility negotiation and protocol versioning
- process health, crash containment, restart policy, and safe shutdown
- bounded resources and structured logs
- auditable permission decisions and mutations

Do not describe an external module as sandboxed until the runtime enforces the
claimed boundary. OS-level isolation and cross-platform behavior require separate
evaluation.

## Cloud and Web

Brain Cloud Core owns authentication, authorization, route mounting, tenant and
project isolation, revisions, audit, and secret resolution. Module APIs cannot
bypass those controls.

Frontend extensions add script supply-chain, data exfiltration, DOM isolation,
content security policy, and upgrade risks. Only trusted built-in web surfaces are
in the current direction. Community web sandboxing is deferred.

## Failure Policy

- incompatible module: refuse enablement with supported API range
- missing permission/config/dependency: remain disabled or unhealthy; no partial
  hidden activation
- migration failure: preserve old data and report recovery steps
- provider timeout/crash: omit contribution with attributable diagnostic; do not
  break unrelated Core workflows
- event/job retry: require idempotency key and bounded retry policy
- shutdown failure: report unhealthy state and avoid claiming clean termination
