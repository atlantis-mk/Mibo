## Context

Mibo currently resolves file reads through several independent call paths. Playback emits `/api/v1/inventory-files/{id}/stream` for `local` and `openlist`, artwork handlers mix redirects, proxy fetches, and provider lookups, sidecar hydration reads directly from local files but falls back to `Get`/`Link`/`RawURL` for remote sources, and probe independently chooses a direct URL target. Those flows duplicate provider fallback rules and each interprets `storage.Object.RawURL` differently.

This is especially fragile for OpenList because provider URLs are ephemeral and must be refreshed from `provider + storagePath` at read time. Local files have the opposite problem: current stream endpoints are effectively stable forever and can be reused if leaked. The system needs a single access model that treats provider-backed files as stable locators plus short-lived runtime access grants, regardless of whether the underlying bytes come from the local filesystem or a remote signed URL.

Constraints:

- Existing inventory and resource models already use `storage_provider` and `storage_path` as stable identifiers and should remain the source of truth.
- Playback, artwork, sidecar hydration, and probe need different access behaviors but should consume a common access abstraction.
- OpenList access URLs can expire at any moment and must be refreshed on demand.
- Local file serving must gain an expiration boundary without exposing absolute paths to clients.

Stakeholders:

- Backend playback and catalog APIs
- Library scan and sidecar hydration flows
- Probe and media metadata enrichment
- Frontend playback and artwork consumers

## Goals / Non-Goals

**Goals:**

- Introduce a unified runtime access service that converts stable locators into purpose-specific, short-lived access grants.
- Remove business-layer dependency on mixed `Get`, `Link`, `RawURL`, and direct filesystem assumptions.
- Enforce expiring external-facing access URLs for both local and OpenList-backed resources.
- Keep provider-specific behavior isolated to provider adapters or the runtime access layer.
- Preserve current library/resource identity semantics and avoid storing transient access URLs as durable state.

**Non-Goals:**

- Replacing the existing inventory/resource schema with a new asset model.
- Building transcoding or download session management in this change.
- Rewriting frontend playback UX beyond consuming new signed access URLs.
- Solving long-term CDN caching strategy for volatile artwork URLs.

## Decisions

### 1. Introduce a dedicated runtime access service

Create a new backend access package that accepts a stable subject such as inventory file ID or provider locator and returns a short-lived access grant. The service will own URL signing, expiration policy, provider runtime resolution, and whether the result should be proxied, redirected, or served from disk.

Why this over reusing provider methods directly:

- Provider APIs answer storage questions, not client authorization questions.
- Centralizing TTL and signature rules avoids duplicating security policy in handlers.
- The same service can serve playback, artwork, sidecar, and probe with purpose-specific policy.

Alternatives considered:

- Expand `storage.Provider` with more fallback behavior only. Rejected because it still leaves signing, user binding, and endpoint issuance in handlers.
- Keep current handler-level logic and add helper functions. Rejected because the main problem is fragmented control of access policy, not just repeated code.

### 2. Separate stable location metadata from transient access grants

Introduce internal models that distinguish stable identity from runtime access:

- `ResourceLocator`: provider, storage path, stable identity, optional object kind
- `ResolvedObject`: provider metadata, file shape, thumbnail hints, optional local path, optional provider-supplied remote URL
- `AccessGrant`: serving mode, signed URL or local path, expiration, volatility, range support

`storage.Object.RawURL` will no longer be trusted as a durable cross-layer contract. Existing provider adapters may still populate it during migration, but access consumers must stop reading it directly.

Why:

- OpenList URLs are volatile.
- Local absolute paths must remain server-only.
- Mixed semantics in `RawURL` create implicit bugs and unsafe reuse.

Alternative considered:

- Keep `RawURL` and add a `RawURLKind` flag. Rejected because it preserves the same leaky cross-layer contract and encourages continued direct use.

### 3. External access will move to signed, purpose-bound endpoints

Introduce signed access endpoints for client-facing reads. Playback and artwork links will be issued as temporary URLs bound to a resource subject, purpose, and expiration timestamp. The signed payload should include enough context to prevent cross-purpose replay, and should optionally bind to authenticated user ID when practical.

Examples of subjects:

- inventory file playback
- inventory file subtitle
- metadata image artwork
- provider object proxy read

Why:

- Local file links need expiry like OpenList.
- Short-lived URLs align both providers behind one security model.
- Handlers can re-resolve the current provider access path at request time, ensuring OpenList URLs are refreshed.

Alternatives considered:

- Keep `/api/v1/inventory-files/{id}/stream` permanently valid and rely on session auth only. Rejected because copied URLs remain abusable and cannot be scoped or time-limited.
- Return raw OpenList links to clients directly. Rejected because URLs expire and leak provider-specific access details.

### 4. Purpose-specific grant policy will determine serving mode

Each access purpose will have a policy that decides whether to proxy, redirect, or serve local files directly after verification.

Initial policy:

- `playback`: always issue a signed Mibo URL; handler resolves provider access at request time and proxies or serves locally
- `artwork`: always issue a signed Mibo URL; handler may internally redirect or proxy after verification, but clients only see Mibo-signed links
- `metadata`: internal-only use through access service; no external URL issuance
- `probe`: internal-only use through access service; prefer direct provider URL when possible for ffprobe efficiency

Why:

- Playback and artwork need client-safe expiring links.
- Sidecar and probe do not need public URLs and should avoid extra endpoint exposure.

Alternative considered:

- Expose direct provider URLs for artwork if stable. Rejected for this change to keep one client model and avoid split semantics.

### 5. Provider runtime resolution remains provider-specific but is hidden behind grants

The access service will ask each provider for runtime access on demand:

- `local`: resolve absolute filesystem path and return a local serving grant
- `openlist`: refresh current file access through `Link` first, then `Get` fallback, and mark results volatile

Thumbnail and sidecar reads follow the same rule: resolve from locator first, then request runtime access for the current purpose.

Why:

- OpenList requires real-time refresh from stable locator.
- Local requires no provider URL generation but still needs unified authorization and endpoint issuance.

Alternative considered:

- Cache OpenList grants centrally. Rejected for the initial change because expiry semantics vary and stale-grant risk is higher than the cost of runtime resolution.

## Risks / Trade-offs

- [Risk] Signed playback URLs may expire mid-session for long-running streams. → Mitigation: issue sufficiently long playback TTLs and allow the client to refresh playback source before expiry; handlers can also honor active byte-range requests until connection close.
- [Risk] Access service becomes a new central dependency for several modules. → Mitigation: keep the service narrowly scoped, add provider-focused tests, and migrate callers incrementally.
- [Risk] Moving away from direct `RawURL` usage may break hidden assumptions in existing code. → Mitigation: add compile-time wrappers and targeted regression tests for playback, artwork, sidecar, and probe paths.
- [Risk] Artwork fetch latency may increase because volatile provider URLs are resolved at request time. → Mitigation: keep artwork TTL short but non-zero and allow handler-level internal refresh only when needed.
- [Risk] Signed endpoints introduce clock-skew sensitivity. → Mitigation: use small verification tolerance and centralize timestamp handling in one signer/verifier implementation.

## Migration Plan

1. Introduce the access domain models and signer/verifier implementation without changing existing handlers.
2. Build provider runtime resolution helpers for `local` and `openlist`.
3. Add signed access endpoints and wire playback URL issuance to them while keeping current `/stream` handlers as internal implementation paths during transition.
4. Migrate artwork serving to signed access URLs.
5. Refactor sidecar hydration and probe to use internal access grants instead of ad hoc provider fallback logic.
6. Remove direct caller reliance on `storage.Object.RawURL` for new code paths and narrow its use to provider-internal migration compatibility.
7. Once all callers are migrated, deprecate permanent external playback/artwork URLs and keep the old endpoints only if needed for internal compatibility.

Rollback:

- Revert URL issuance to the existing `/api/v1/inventory-files/{id}/stream` and artwork routes while leaving the access service unused.
- Because stable locator fields remain unchanged, rollback does not require data migration.

## Open Questions

- Should signed access URLs be bound to user ID only, or also to session ID / device context for stricter replay prevention?
- What TTL defaults should be used for playback, artwork, and subtitles, and should they be configurable per deployment?
- Should OpenList thumbnail URLs be stored as advisory metadata only, or should the system add a separate persisted thumbnail locator concept in a follow-up change?
- Do we want to fully replace `/api/v1/inventory-files/{id}/stream`, or keep it as a verified internal target behind signed URL issuance for backward compatibility?
