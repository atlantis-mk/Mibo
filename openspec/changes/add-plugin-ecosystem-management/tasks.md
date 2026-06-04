## 1. Plugin Center Navigation And Shell

- [x] 1.1 Add an administrator-only `/settings/plugins` section with a clear menu label, description, icon, and route.
- [x] 1.2 Add a plugin center page shell with overview, instances, detail, local lifecycle, and catalog-ready areas.
- [x] 1.3 Ensure non-admin users cannot access the plugin center route or plugin lifecycle APIs.

## 2. Shared Plugin Instance Management

- [x] 2.1 Extract existing remote plugin provider instance form, card, health badge, manifest preview, and schema-driven configuration controls into reusable plugin management components.
- [x] 2.2 Render remote plugin provider instances in the plugin center with create, edit, enable/disable, and refresh-health actions.
- [x] 2.3 Keep metadata provider settings functional while avoiding duplicate plugin management logic.
- [x] 2.4 Add frontend tests for plugin center listing, registration, editing, health refresh, and disabled states.

## 3. Plugin Usage And Diagnostics

- [x] 3.1 Add backend service methods to compute plugin provider usage across metadata profiles and media sources.
- [x] 3.2 Add an admin/settings API endpoint that returns plugin instance detail plus reference summaries.
- [x] 3.3 Display usage references before disable, uninstall, update, or rollback actions.
- [x] 3.4 Display manifest metadata, capabilities, protocol version, endpoint, deployment kind, last checked time, failure reason, and cooldown state in plugin detail.
- [x] 3.5 Add backend and frontend tests for usage summaries and destructive-action warnings.

## 4. Local Companion Lifecycle Foundation

- [x] 4.1 Add backend models for local plugin installations and companion runtime state.
- [x] 4.2 Add service methods to register/install a local companion plugin from the chosen first install source.
- [x] 4.3 Add start, stop, restart, uninstall, and log retrieval service methods for local companions.
- [x] 4.4 Add HTTP APIs for local companion lifecycle actions with administrator authorization.
- [x] 4.5 Connect local companion endpoint resolution to the existing provider plugin manifest and health workflow.
- [x] 4.6 Add tests for lifecycle state transitions, endpoint resolution failures, log retrieval, uninstall cleanup, and active-reference protection.

## 5. Catalog And Update Readiness

- [x] 5.1 Define catalog source and catalog entry models for future plugin discovery.
- [x] 5.2 Add compatibility checking for Mibo version, protocol version, platform, and declared capabilities.
- [x] 5.3 Add trust metadata fields for source, checksum/signature status, homepage, and release notes.
- [x] 5.4 Add UI placeholders or disabled states that explain catalog/update support is not yet configured when no catalog source exists.
- [x] 5.5 Add tests for compatibility warnings and trust metadata rendering.

## 6. Verification

- [x] 6.1 Run focused frontend tests for settings navigation and plugin management components.
- [x] 6.2 Run focused backend tests for settings plugin provider APIs, usage summaries, and lifecycle services.
- [x] 6.3 Run `cd frontend && pnpm lint && pnpm test` after frontend implementation.
- [x] 6.4 Run `cd mibo-media-server && go test ./...` after backend implementation.
