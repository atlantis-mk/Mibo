## Context

Mibo currently supports metadata providers and media sources through built-in Go implementations. Storage providers already use an interface-like boundary, but the registry and source configuration are hard-coded for `local` and `openlist`. Metadata provider settings already expose provider instances and profiles, but execution still dispatches by fixed provider type such as `tmdb`, `metatube`, `tvdb`, and `local_scan`.

The desired plugin model is an external service protocol with two deployment shapes: a remote service managed outside Mibo and a local companion process managed by Mibo. Both shapes should expose the same provider protocol so the core application, frontend, and plugin authors do not need separate integration models.

## Goals / Non-Goals

**Goals:**

- Define a shared provider plugin protocol for discovery, health, configuration schema, and capability execution.
- Treat remote plugin services and local companion plugins as the same kind of callable provider endpoint after registration.
- Let provider instances declare capabilities such as metadata search, metadata detail, storage browse, storage resolve, and storage link.
- Make provider configuration schema-driven so new plugins can be configured without hard-coded provider-specific frontend forms.
- Preserve existing built-in provider behavior through compatibility adapters while plugin-backed providers are introduced.
- Keep the first version small enough to implement safely across backend settings, metadata, library, storage, and frontend settings surfaces.

**Non-Goals:**

- Building a plugin marketplace or public catalog.
- Implementing automatic plugin installation, upgrade, or rollback for third-party packages beyond local companion lifecycle support.
- Allowing arbitrary plugin-provided frontend UI extensions.
- Defining a full signing and trust distribution system.
- Replacing existing built-in providers in the first implementation pass.
- Supporting asynchronous worker-style plugin jobs in the first protocol version.

## Decisions

### 1. Use one protocol with two deployment shapes

Mibo will define a single provider protocol that can be served by a remote endpoint or by a local companion process. The core runtime will normalize both shapes into a provider instance with an endpoint, manifest snapshot, configuration, capabilities, health status, and enabled state.

Alternative considered: separate remote and local plugin models. This would let each shape optimize independently, but it would force duplicate protocol support in core routing, tests, frontend configuration, and plugin documentation. A single protocol keeps plugin authors and Mibo core aligned.

### 2. Treat local companion as endpoint management, not a second plugin API

A local companion plugin will be started and monitored by Mibo, but once it exposes its local endpoint, Mibo will call it through the same manifest, health, and capability endpoints as a remote service. Companion lifecycle code should handle process start, stop, logs, endpoint discovery, and restart policy only.

Alternative considered: stdio-only local plugins. This can work well for command-style integrations, but storage browsing and metadata lookup benefit from request/response HTTP semantics, health checks, and the same protocol surface as remote services.

### 3. Use manifest-driven discovery

Each plugin endpoint will expose a manifest containing plugin identity, protocol version, display metadata, supported capabilities, endpoint paths, configuration schema, authentication requirements, and operational hints. Mibo will validate and snapshot the manifest when registering or refreshing a provider instance.

Alternative considered: administrators manually entering capability endpoints and fields. That would make the first version simpler, but it would keep the frontend and backend coupled to provider-specific assumptions and make third-party plugins hard to configure consistently.

### 4. Use shared manifest envelope with capability-specific contracts

Manifest, health, configuration schema, authentication metadata, and instance state will be shared across plugin types. Capability payloads will remain separate contracts for metadata and storage operations.

Alternative considered: one universal request/response format for every plugin call. That would reduce the number of structs but make the protocol vague. Capability-specific contracts keep results testable and map more cleanly onto existing metadata and storage services.

### 5. Require plugins to return normalized capability results

Plugin responses should use Mibo-defined normalized results, such as metadata search candidates, metadata detail documents, and storage object summaries. Core validation will reject malformed or unsupported results before persistence or playback use.

Alternative considered: plugins return raw upstream payloads and Mibo normalizes them. That would keep plugins thinner, but it would move third-party provider knowledge into Mibo core and recreate the hard-coded provider problem this change is meant to solve.

### 6. Preserve built-in providers through adapters

Existing TMDB, MetaTube, local scan, local storage, and OpenList behavior should remain available. The first implementation can expose them through adapter registrations that satisfy the same internal provider instance and capability routing model, even if they do not immediately run as external plugin services.

Alternative considered: migrate all built-ins to external services immediately. That would prove the protocol quickly, but it creates unnecessary release and deployment risk.

### 7. Keep plugin configuration schema constrained

The first version should support a practical subset of JSON Schema or a documented lightweight schema sufficient for strings, secrets, numbers, booleans, selects, URLs, durations, required fields, defaults, help text, and secret redaction. The backend remains authoritative for validation and redaction.

Alternative considered: accepting arbitrary JSON Schema with advanced composition. That increases frontend rendering complexity and test surface before Mibo has real plugin ecosystem pressure.

## Risks / Trade-offs

- [Protocol too broad in the first version] -> Limit V1 to discovery, health, metadata search/detail, and storage browse/resolve/link.
- [Dynamic schemas create inconsistent settings UX] -> Use a constrained schema subset and shared form components, with provider-specific built-in forms retained where needed during migration.
- [Plugin response quality varies] -> Validate normalized responses, record provider failures, and surface health/cooldown state in settings.
- [Local companion management expands scope] -> Keep companion work focused on process lifecycle and endpoint discovery; defer installation marketplace and upgrades.
- [Secrets leak through manifests or config views] -> Store secrets server-side, redact all public configuration views, and avoid echoing secret values back to the frontend.
- [Existing records become hard to migrate] -> Use additive fields and compatibility adapters so existing provider instances and media sources continue to work.

## Migration Plan

1. Add plugin protocol models, validation, and storage for manifest snapshots, capabilities, endpoint metadata, deployment kind, health state, and configuration payloads.
2. Add compatibility adapters for existing built-in metadata and storage providers so current flows can use the new routing layer without changing user behavior.
3. Add remote endpoint registration and manifest refresh APIs.
4. Add local companion endpoint management with a minimal process lifecycle and endpoint discovery contract.
5. Update metadata profile and media source configuration flows to consume capability-aware provider instances.
6. Update frontend settings to render schema-driven plugin configuration where plugin manifests are present, while preserving existing specialized forms during migration.
7. Roll out plugin-backed providers behind additive APIs and keep rollback simple by allowing administrators to disable plugin instances and continue using built-in providers.

## Open Questions

- Should V1 use a constrained JSON Schema subset or a custom Mibo field schema for configuration rendering?
- Should plugin-to-core authentication start with static bearer tokens only, or also support signed requests from the beginning?
- Should local companion endpoint discovery use a localhost HTTP port file, a Unix socket, or both?
- How much of the existing `MetadataProviderInstance` and `MediaSource` tables should be evolved directly versus introducing a more general `PluginProviderInstance` table?
- Should plugin protocol version compatibility be strict exact match in V1, or allow minor-version compatibility?
