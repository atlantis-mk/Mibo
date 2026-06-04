## 1. Access Domain Foundation

- [x] 1.1 Introduce stable locator and runtime access domain types for provider-backed file reads (for example resource locator, resolved object, access grant, and access purpose).
- [x] 1.2 Add a shared access service interface and implementation that signs, verifies, and issues short-lived grants for playback and artwork access.
- [x] 1.3 Add provider runtime resolution methods for local and OpenList so they can translate a stable locator into a current access mode without exposing provider-specific fallback logic to callers.

## 2. Signed Access Endpoints

- [x] 2.1 Add signed access endpoints and verification middleware/helpers for inventory-file playback and metadata artwork reads.
- [x] 2.2 Update playback URL issuance so local and OpenList inventory files return signed temporary access URLs instead of permanent external-facing stream URLs.
- [x] 2.3 Update artwork URL issuance so client-visible artwork links are signed temporary access URLs instead of direct provider URLs or permanent internal routes.

## 3. Caller Migration

- [x] 3.1 Refactor inventory playback handlers to consume the shared access service and serve local files or refreshed OpenList content after grant verification.
- [x] 3.2 Refactor artwork serving to use the shared access service for local files, OpenList-backed images, and volatile thumbnail refresh behavior.
- [x] 3.3 Refactor sidecar hydration and sidecar metadata reading to request purpose-specific runtime access from the shared service instead of using ad hoc `Get`/`Link`/`RawURL` fallback logic.
- [x] 3.4 Refactor probe target resolution to obtain runtime access through the shared service instead of duplicating provider fallback behavior.

## 4. Regression Coverage and Cleanup

- [x] 4.1 Add tests covering signed grant verification, expiration handling, and purpose binding for local and OpenList-backed playback/artwork access.
- [x] 4.2 Add regression tests covering OpenList runtime URL refresh, local path non-exposure, and migrated sidecar/probe access flows.
- [x] 4.3 Remove or narrow direct business-layer dependence on `storage.Object.RawURL` so new code paths rely on the shared runtime access service instead of mixed URL/path semantics.
