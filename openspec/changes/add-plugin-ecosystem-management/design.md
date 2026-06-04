## Context

The provider plugin protocol change established a runtime model for plugin-backed provider instances. Those instances expose manifests, capabilities, health, configuration schema, deployment kind, endpoint metadata, and enabled state. The current frontend can register and manage remote plugin provider instances from the metadata provider settings panel, and media sources can select plugin-backed storage providers.

That solved execution, but not product ownership. Once plugins support metadata, storage, and eventually more capabilities, administrators need a place to answer operational questions:

- What plugins are installed or registered?
- Which plugins are unhealthy?
- What capabilities does a plugin provide?
- What metadata profiles or media sources depend on this plugin?
- Is this plugin remote or local?
- Can I stop, restart, update, uninstall, or roll back it safely?

## Goals / Non-Goals

**Goals:**

- Promote plugins to a first-class settings area.
- Reuse the existing provider plugin protocol and provider instance model where possible.
- Make plugin state, references, and health understandable before adding installation complexity.
- Support both remote plugin services and local companion plugins from one management center.
- Design catalog and update support so it can be implemented later without changing the core navigation model.
- Keep plugin lifecycle actions administrator-only and explicit.

**Non-Goals for the first implementation slice:**

- Building a public marketplace.
- Enabling arbitrary plugin-provided frontend UI extensions.
- Automatically installing or updating third-party code without administrator confirmation.
- Replacing built-in metadata or storage providers with external plugins.
- Supporting non-provider plugin extension points before the management shell is stable.

## Decisions

### 1. Add a dedicated plugin center under settings

Plugins should not remain hidden inside metadata settings. A `/settings/plugins` route makes plugin management discoverable and avoids coupling future storage, local companion, catalog, and update features to metadata provider screens.

Alternative considered: keep plugin management inside metadata and media source settings. This is simpler for V1, but it makes cross-capability diagnostics awkward and would require duplicate controls as plugin capabilities expand.

### 2. Separate plugin identity from provider instance usage

The provider instance remains the executable unit for metadata and storage operations. The plugin center should present those instances through a plugin-first lens: identity, deployment shape, capabilities, health, configuration, and references.

For local companion and catalog phases, an installed plugin package may produce one or more provider instances. The data model should avoid assuming a one-to-one relationship between installed plugin package and provider instance forever.

### 3. Make usage references visible before destructive actions

Disable, uninstall, update, or rollback actions can break metadata profiles and media sources. The plugin center should show references and require elevated confirmation when an action affects active usage.

Alternative considered: allow operations and rely on runtime errors. That is faster to implement but poor administrator UX and makes failures surprising.

### 4. Treat local companion lifecycle as orchestration around the same protocol

Local companions should be installed, started, stopped, logged, and resolved by Mibo, but once an endpoint is available the existing provider protocol should continue to handle manifest, health, configuration, and capability calls.

This preserves the earlier plugin protocol decision and prevents a second plugin execution API from appearing accidentally.

### 5. Design catalog support around trust and compatibility

Catalog entries should be metadata first: source, plugin ID, versions, protocol compatibility, supported capabilities, platform constraints, checksum/signature metadata, homepage, and release notes. Installation should only happen after compatibility checks and administrator confirmation.

The first catalog implementation can be a configured JSON feed or local manifest source; a public marketplace can come later.

## UI Shape

```text
Settings
└── Plugins
    ├── Overview
    │   ├── Registered / installed count
    │   ├── Unhealthy / cooling down count
    │   └── Capability distribution
    ├── Instances
    │   ├── Remote provider instances
    │   ├── Local companion provider instances
    │   └── Enable / disable / refresh health
    ├── Detail
    │   ├── Manifest
    │   ├── Capabilities
    │   ├── Configuration schema summary
    │   ├── Runtime health
    │   └── References
    ├── Local
    │   ├── Install / uninstall
    │   ├── Start / stop / restart
    │   └── Logs
    └── Catalog
        ├── Available plugins
        ├── Compatibility and trust warnings
        └── Update / rollback
```

## Data Model Sketch

Existing:

- `PluginProviderInstance`: runtime provider instance used for capability execution.

Potential additions:

- `PluginInstallation`: local installed package identity, version, source, install path, enabled state, lifecycle state, created/updated timestamps.
- `PluginRuntimeProcess`: companion process status, pid where available, resolved endpoint, last start/stop timestamps, exit reason.
- `PluginCatalogSource`: configured source name, URL/path, trust level, enabled state, last sync result.
- `PluginCatalogEntry`: plugin identity, version metadata, capabilities, compatibility, checksums/signatures, release notes.
- `PluginUsageSummary`: computed view rather than necessarily persisted; maps provider instances to metadata profiles, media sources, and future dependents.

## Risks / Trade-offs

- [Scope creep into marketplace] -> Keep Phase 1 focused on management and diagnostics, with catalog data structures designed but not necessarily implemented.
- [Running third-party code increases risk] -> Require explicit admin actions, show trust warnings, preserve source/checksum metadata, and avoid automatic execution after install unless configured.
- [Breaking active media sources or profiles] -> Surface references and require confirmation for destructive or disabling actions.
- [Duplicated plugin controls] -> Extract reusable plugin instance management components and make metadata settings link to the plugin center for full controls.
- [Local lifecycle varies by platform] -> Start with a minimal process manager contract and avoid platform-specific package managers in the first pass.

## Open Questions

- Should `/settings/metadata-sources` keep a minimal “register plugin” affordance, or should all plugin registration move to `/settings/plugins`?
- Should the first local companion install source be a local filesystem path, uploaded archive, configured directory, or catalog feed?
- Should uninstall preserve configuration by default for rollback, or remove secrets immediately unless the admin chooses to retain them?
- What trust model is acceptable for the first catalog source: unsigned local feeds, checksums only, or signatures from the beginning?
- Should plugin lifecycle actions write audit log entries in the first implementation slice?
