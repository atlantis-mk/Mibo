## Why

Mibo now has a provider plugin protocol and can register plugin-backed provider instances, but plugin management is still embedded inside metadata and media source workflows. Administrators need a dedicated plugin center that treats plugins as first-class extensions: visible, diagnosable, configurable, lifecycle-managed, and eventually discoverable from a trusted catalog.

The broader goal is a plugin ecosystem center rather than only a provider settings panel. This should let Mibo grow from manually registered remote plugin services into locally managed companion plugins and later into catalog-driven install, upgrade, rollback, and trust workflows without redesigning the management surface each time.

## What Changes

- Add an administrator-facing settings menu and page for plugin ecosystem management.
- Centralize plugin provider instance registration, editing, enabling/disabling, health refresh, manifest inspection, and capability summaries.
- Show plugin usage relationships, including metadata profiles and media sources that depend on each plugin-backed provider instance.
- Add diagnostic views for availability status, failure reason, cooldown state, checked timestamp, endpoint, protocol version, and declared capabilities.
- Introduce a local companion plugin lifecycle model for installed companion packages, including install state, process state, resolved endpoint, logs, start/stop/restart, uninstall, and cleanup semantics.
- Define a catalog-ready model for future plugin discovery, compatibility checks, trusted source metadata, version updates, and rollback.
- Keep marketplace distribution, broad public signing infrastructure, and arbitrary plugin-provided frontend UI extensions out of the first implementation slice unless explicitly promoted by later changes.

## Phased Scope

### Phase 1: Plugin Management Center

- Add `/settings/plugins` as an admin-only settings page.
- Move or share existing remote plugin provider instance management from metadata settings into the plugin center.
- Add plugin detail panels for manifest, capabilities, configuration schema summary, health, and references.
- Keep existing metadata profile and media source plugin selection behavior intact.

### Phase 2: Local Companion Lifecycle

- Add backend models and APIs for locally installed companion plugins.
- Support install/register from a local package source or filesystem path, start, stop, restart, uninstall, and log inspection.
- Resolve local companion endpoints before provider registration or execution.
- Preserve the same provider protocol once a companion exposes its endpoint.

### Phase 3: Catalog And Updates

- Add a catalog abstraction for listing available plugins from configured sources.
- Check compatibility against Mibo version, plugin protocol version, capability support, and platform constraints.
- Support update availability, upgrade, rollback, and trust warnings.
- Require explicit administrator confirmation before running or updating third-party code.

## Capabilities

### New Capabilities

- `plugin-ecosystem-management`: Defines the settings plugin center, plugin instance management, usage visibility, diagnostics, local companion lifecycle, and catalog/update readiness.

### Modified Capabilities

- `provider-plugin-protocol`: Existing plugin provider instances remain the runtime execution model; this change adds management surfaces and lifecycle orchestration around them rather than replacing the protocol.
- `plugin-metadata-providers`: Metadata profile usage should be visible from the plugin center.
- `plugin-storage-providers`: Media source usage should be visible from the plugin center.

## Impact

- Frontend settings navigation will gain an administrator-only plugin center entry.
- Frontend plugin management should be extracted into reusable components so metadata settings can either link to the plugin center or continue rendering a scoped provider view without duplicating logic.
- Backend settings and HTTP APIs will need plugin usage summary endpoints and, for later phases, local companion lifecycle endpoints.
- Database changes are likely for local companion install records, catalog source records, plugin version records, lifecycle status, and audit-relevant timestamps.
- Tests should cover admin-only routing, plugin list/detail behavior, reference summaries, lifecycle state transitions, catalog compatibility checks, and safety confirmations.
