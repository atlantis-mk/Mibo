## Context

Mibo currently exposes operational failures mainly through job records, raw error messages, and coarse library status values such as `active`, `syncing`, and `error`. Catalog queries intentionally filter home and discovery content to active libraries, so a library can contain scanned catalog items while the home page appears empty after a later storage refresh failure. The concrete local failure that motivated this change is an OpenList/PikPak `captcha_invalid` / `captcha_token expired` error that left two libraries in `error` even though catalog items existed.

The affected areas are cross-cutting: storage adapters surface provider errors, scan and probe jobs record failures, library status gates catalog visibility, home consumes catalog and library data, settings exposes library management, and jobs expose technical history. The design should centralize diagnosis so UI surfaces do not independently parse raw job strings.

## Goals / Non-Goals

**Goals:**
- Provide stable, structured health diagnostics for media sources, libraries, jobs, and external dependencies.
- Translate known provider/runtime failures into user-facing issue summaries with recovery guidance.
- Explain home-page empty/degraded states when content exists but affected libraries are hidden by blocking health issues.
- Offer guided recovery actions such as opening OpenList, validating a source, and re-scanning affected libraries.
- Preserve raw technical details for support and debugging without making them the primary user message.

**Non-Goals:**
- Replace the jobs system or remove raw job error history.
- Build a full incident-management system with acknowledgements, paging, or historical analytics.
- Guarantee automatic repair of external services such as PikPak captcha completion; Mibo can guide and verify, but external authentication still happens outside Mibo.
- Change catalog cutover semantics or make unavailable/error libraries appear as healthy active content.

## Decisions

### 1. Introduce Domain Health Events Instead Of UI-Side Error Parsing

Backend diagnostics will normalize raw failures into stable domain reason codes such as `storage_auth_expired`, `storage_service_unreachable`, `storage_path_unavailable`, `metadata_auth_invalid`, `probe_runtime_unavailable`, and `job_failed_unknown`.

Rationale: UI strings and provider errors are unstable. A backend classification layer can inspect job kind, payload, associated source/library records, and raw errors once, then expose consistent severity, scope, and actions.

Alternative considered: let the home page and settings page scan recent jobs for keywords. This is faster initially but duplicates logic, creates inconsistent messages, and couples UI behavior to raw English/provider error strings.

### 2. Derive Health From Current State Plus Recent Failures, With Optional Persistence

The first implementation should expose a derived health view assembled from libraries, media sources, recent failed jobs, schedule runs, and known provider validation checks. If implementation reveals expensive joins or a need for explicit lifecycle control, a `health_events` table can be added to persist active events.

Rationale: the current data already records library status, job payloads, and error messages. A derived service avoids premature event lifecycle complexity while still creating a single API contract. The API contract should not depend on whether events are derived or persisted.

Alternative considered: create a persistent event ledger immediately. This gives clearer lifecycle management but risks overbuilding before the needed retention and dismissal semantics are known.

### 3. Model Health At Multiple Scopes

Health summaries should support `global`, `media_source`, `library`, `job`, and `dependency` scopes. A single root cause can affect multiple libraries, such as one OpenList/PikPak media source causing both movie and show libraries to become unavailable.

Rationale: users need to understand both the cause and the affected content. Surfacing only library-level `error` duplicates the same storage problem across many libraries, while surfacing only source-level errors hides which content disappeared.

Alternative considered: keep health only on libraries. This matches existing UI cards but makes shared external dependency failures noisy and harder to resolve once.

### 4. Use Severity And Visibility Impact Separately

Diagnostics should expose severity (`info`, `warning`, `error`, `blocking`) separately from impact fields such as `blocks_home_visibility`, `blocks_scan`, `blocks_playback`, or affected counts.

Rationale: not every error makes content disappear. Pending probes may be warnings, storage auth failures may block scans and home visibility, and metadata failures may degrade artwork without blocking playback.

Alternative considered: reuse `library.status` as the only severity. This is too coarse and conflates scan state, storage health, and catalog visibility.

### 5. Add A Health Center Route And Lightweight Global Surfaces

The Health Center should be the detailed remediation surface. The home page, sidebar, and settings cards should show concise issue indicators that link into the relevant Health Center issue.

Rationale: home should explain the symptom quickly, not become a diagnostics console. Settings should offer local context for a library or source. A central route keeps all active issues discoverable and reusable.

Alternative considered: only enhance settings. This misses the most confusing user moment: landing on an empty home page after content was scanned.

### 6. Recovery Actions Are Descriptive Contracts Backed By Existing Or New Endpoints

Health issues should include action descriptors such as `open_external_admin`, `validate_media_source`, `rescan_affected_libraries`, and `view_job`. The frontend renders these actions when supported.

Rationale: diagnostics are only useful if they answer “what now?” Action descriptors decouple issue classification from specific UI placement.

Alternative considered: hard-code buttons per reason code in the frontend. This is acceptable for the first issue but scales poorly as more provider/runtime failures are classified.

## Risks / Trade-offs

- Raw provider errors may be ambiguous → keep a fallback `job_failed_unknown` issue with technical details and add classifiers incrementally.
- Derived health can miss historical root cause after jobs are pruned → include enough recent job retention for active failures and consider persistent `health_events` if retention becomes a problem.
- Multiple failed jobs can create duplicate issues → group by reason code, scope, media source, library set, and latest failing job.
- Users may expect Mibo to fix external captcha automatically → wording must clearly state that OpenList/PikPak verification happens outside Mibo, while Mibo can validate and re-scan afterward.
- Blocking health might hide content even though stale catalog data is usable → home copy should say content is not lost and explain that unavailable libraries are hidden by current visibility rules.
- Health Center can become noisy → prioritize blocking/error issues first and collapse warning/info issues by category.

## Migration Plan

1. Add diagnostics service and API responses without changing existing job or library endpoints.
2. Add classifiers for the known OpenList/PikPak auth-expired failure and generic fallbacks.
3. Add frontend Health Center and global badges behind the new diagnostics API.
4. Update home empty/degraded state to consume health summaries when catalog-visible items are empty.
5. Extend library/settings cards to display health summaries and link to details.
6. Keep existing job list available as the technical detail fallback.

Rollback is straightforward if diagnostics are additive: hide the Health Center route/surfaces and keep existing library/job behavior. If a persistent events table is introduced, leave it unused on rollback rather than deleting data.

## Open Questions

- Should users be able to dismiss non-blocking health issues, or should issues only disappear when the underlying condition clears?
- Should home ever show stale catalog items from error libraries with a degraded banner, or continue hiding them until libraries return to active?
- What OpenList admin URL should Mibo open for provider-specific recovery: the configured OpenList base URL, a storage detail route, or a generic admin page?
- How much recent job history should diagnostics inspect if health events remain derived rather than persisted?
