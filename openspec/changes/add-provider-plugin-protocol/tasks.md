## 1. Protocol Foundation

- [x] 1.1 Add backend plugin protocol types for manifest, protocol version, capabilities, operation paths, health, configuration schema, deployment kind, and endpoint metadata.
- [x] 1.2 Implement manifest validation covering required identity fields, supported protocol version, capability declarations, operation paths, and configuration schema constraints.
- [x] 1.3 Add normalized configuration schema validation and redaction helpers for strings, secrets, numbers, booleans, selects, URLs, durations, required fields, defaults, and display metadata.
- [x] 1.4 Add tests for valid manifests, invalid manifests, unsupported versions, missing capability paths, schema validation, and secret redaction.

## 2. Provider Instance Persistence

- [x] 2.1 Add additive database models or fields for plugin-backed provider instances, including deployment kind, endpoint, plugin identity, manifest snapshot, capabilities, configuration payload, health state, enabled state, and timestamps.
- [x] 2.2 Add migrations and repository/service methods for registering, updating, disabling, listing, and refreshing plugin provider instances.
- [x] 2.3 Ensure provider instance read APIs return display metadata and redacted configuration without exposing stored secrets.
- [x] 2.4 Add persistence tests for remote instances, local companion instances, manifest snapshots, disabled instances, and redacted views.

## 3. Endpoint Runtime

- [x] 3.1 Implement a plugin HTTP client for manifest fetch, health check, and capability operation calls with timeouts and structured error mapping.
- [x] 3.2 Implement provider health checks that update availability status, failure reason, cooldown state, and checked timestamp.
- [x] 3.3 Add remote endpoint registration APIs that fetch and validate manifests before enabling provider instances.
- [x] 3.4 Add minimal local companion lifecycle support that starts or locates a companion process, resolves its local endpoint, fetches its manifest, and reports lifecycle failures.
- [x] 3.5 Add runtime tests with mock remote plugin endpoints and mock local companion endpoint discovery.

## 4. Capability Routing and Compatibility

- [x] 4.1 Add capability-based provider selection that filters by enabled state, declared capability, health status, and cooldown state.
- [x] 4.2 Replace hard-coded provider selection paths where practical with capability routing while preserving existing behavior.
- [x] 4.3 Add compatibility adapters for built-in TMDB, MetaTube, local scan, local storage, and OpenList providers so they can participate in the new routing model.
- [x] 4.4 Add fallback behavior tests for disabled providers, missing capabilities, rate-limited providers, and built-in adapter execution.

## 5. Plugin Metadata Providers

- [x] 5.1 Define normalized plugin metadata search request and response contracts based on existing metadata candidate behavior.
- [x] 5.2 Define normalized plugin metadata detail request and response contracts based on existing metadata item, external ID, image, people, tag, rating, and hierarchy persistence needs.
- [x] 5.3 Integrate plugin-backed metadata search providers into metadata profiles and manual candidate search.
- [x] 5.4 Integrate plugin-backed metadata detail providers into candidate application and metadata operation execution.
- [x] 5.5 Preserve metadata source attribution for plugin-backed results in governance and metadata source views.
- [x] 5.6 Add tests for valid plugin search results, malformed search results, valid detail persistence, malformed detail rejection, fallback routing, local scan fallback, and governance attribution.

## 6. Plugin Storage Providers

- [x] 6.1 Define normalized plugin storage browse, resolve, and link request and response contracts based on the existing storage provider interface.
- [x] 6.2 Integrate plugin-backed storage providers into media source creation, update, list, browse, and delete flows.
- [x] 6.3 Integrate plugin-backed storage browse and resolve behavior into library scanning while preserving inventory, recognition, exclusion, and materialization behavior.
- [x] 6.4 Integrate plugin-backed storage link behavior with playback and probing only when the provider declares link capability.
- [x] 6.5 Add tests for browse validation, resolve failures, media source listing, scanner integration, link capability handling, authorization, and provider metadata sanitization.

## 7. Frontend Settings Experience

- [x] 7.1 Add frontend API types and query/mutation helpers for plugin manifests, plugin-backed provider instances, health state, deployment kind, capabilities, and redacted configuration.
- [x] 7.2 Build shared schema-driven configuration form components for the supported configuration schema subset.
- [x] 7.3 Update metadata provider settings to register, edit, disable, and display plugin-backed metadata provider instances alongside existing built-in providers.
- [x] 7.4 Update metadata profile editing to select provider instances by metadata search and detail capabilities.
- [x] 7.5 Update media source setup to select plugin-backed storage provider instances and render their schema-driven configuration.
- [x] 7.6 Add frontend tests for schema rendering, secret redaction display, capability filtering, disabled providers, and plugin-backed source/profile configuration.

## 8. Verification and Documentation

- [x] 8.1 Add protocol documentation for manifest fields, health responses, configuration schema, metadata capability payloads, storage capability payloads, and error mapping.
- [x] 8.2 Add example mock plugin services for at least one metadata provider and one storage provider.
- [x] 8.3 Run focused backend tests for protocol, settings, metadata, library, storage, playback, and HTTP API changes.
- [x] 8.4 Run focused frontend tests for settings and provider configuration flows.
- [x] 8.5 Run full validation with `cd mibo-media-server && go test ./...` and `cd frontend && pnpm test` when implementation is complete.
