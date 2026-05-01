## 1. Data Model And Migration

- [x] 1.1 Add backend database models for library paths, scan policies, metadata policies, playback policies, and subtitle policies.
- [x] 1.2 Add migrations/auto-migration coverage that creates the new tables and indexes.
- [x] 1.3 Backfill one enabled library path for each existing library from `media_source_id` and `root_path`.
- [x] 1.4 Add default policy creation/resolution so libraries without explicit policy rows preserve current behavior.
- [x] 1.5 Add tests for existing-library migration, default policy resolution, and compatibility field preservation.

## 2. Library API And Service Layer

- [x] 2.1 Add typed DTOs for library paths and policy groups in backend API responses and frontend API types.
- [x] 2.2 Extend library create/detail/list responses to include paths and policies while retaining `media_source_id` and `root_path`.
- [x] 2.3 Add service methods and HTTP handlers to add, update, enable/disable, and list library paths.
- [x] 2.4 Add service methods and HTTP handlers to read and update scan, metadata, playback, and subtitle policies.
- [x] 2.5 Add validation that each library path resolves through its referenced media source provider before persistence.
- [x] 2.6 Add API tests for path management, policy updates, invalid path rejection, and compatibility responses.

## 3. Effective Library Configuration

- [x] 3.1 Implement a library resolver that returns the library, enabled paths, provider handles, and effective policies.
- [x] 3.2 Update library scan queue/run paths to use enabled library paths instead of directly traversing `Library.RootPath`.
- [x] 3.3 Update targeted refresh scoping so requested roots are matched against enabled library paths.
- [x] 3.4 Update scheduled scan, cleanup, and invalid-link jobs to resolve libraries through the new configuration helper.
- [x] 3.5 Add focused tests proving disabled paths are skipped and multi-path libraries scan all enabled paths.

## 4. Scan And Listener Policies

- [x] 4.1 Apply scan policy settings for scanner enabled state, hidden files, ignored extensions, sample-size threshold, and configurable exclusion-rule participation.
- [x] 4.2 Preserve manual scan exclusion precedence over policy-driven ignore decisions.
- [x] 4.3 Update scan summaries to account for policy-driven skipped files where useful.
- [x] 4.4 Update listener/openlist reconciliation to skip targeted refresh enqueueing when realtime/listener policy is disabled.
- [x] 4.5 Add tests for extension ignores, manual exclusion precedence, configurable-rule behavior, and realtime policy disablement.

## 5. Metadata Policy Integration

- [x] 5.1 Resolve preferred metadata language, image language, country/region, provider enablement, local metadata participation, and provider priority from library metadata policy.
- [x] 5.2 Update metadata search, match, refetch, and scanner metadata sidecar flows to use effective metadata policy.
- [x] 5.3 Ensure disabled providers are not called for automated library-scoped metadata operations.
- [x] 5.4 Preserve existing field locks, manual edits, and review-needed governance protections during policy-driven refresh.
- [x] 5.5 Add tests for language override, disabled provider behavior, local sidecar disablement, and locked-field preservation.

## 6. Playback And Subtitle Policies

- [x] 6.1 Apply playback policy thresholds when recording or interpreting user item progress.
- [x] 6.2 Apply subtitle policy during scanner sidecar binding so disabled external subtitles are not bound as playable tracks.
- [x] 6.3 Apply subtitle language preferences and unavailable-subtitle tolerance when building playback responses.
- [x] 6.4 Preserve safe Mibo-controlled subtitle URLs and avoid exposing provider internals.
- [x] 6.5 Add tests for resume thresholds, short-duration progress suppression, external subtitle disablement, subtitle language preference, and missing subtitle tolerance.

## 7. Frontend Management UI

- [x] 7.1 Update frontend API types and client methods for library paths and policy groups.
- [x] 7.2 Keep the basic create-library drawer minimal while applying default policies automatically.
- [x] 7.3 Add library settings sections for source paths, scan policy, metadata policy, playback policy, and subtitle policy.
- [x] 7.4 Add UI validation and feedback for invalid paths and failed policy updates.
- [x] 7.5 Verify the management UI works on desktop and mobile layouts.

## 8. Verification

- [x] 8.1 Run backend focused tests for library, scan exclusion, listener, metadata, playback, and subtitle flows.
- [x] 8.2 Run `go test ./...` from `mibo-media-server/`.
- [x] 8.3 Run frontend typecheck from `web/`.
- [x] 8.4 Manually verify an existing single-root library still lists, scans, and plays media after migration.
- [x] 8.5 Manually verify a multi-path library scans enabled paths, skips disabled paths, and shows combined catalog contents.
