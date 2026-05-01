## Context

Mibo already has metadata provider instances, metadata templates, and per-library metadata strategies. The recent runtime model makes provider instances the executable source selection mechanism, but actual execution is still mostly TMDB-shaped: TMDB is the only editable remote provider with execution support, TVDB configuration is persisted but not used, and `local_scan` is a system-managed detail-stage provider backed by scanner evidence.

MetaTube is best treated as an HTTP metadata provider instance. Its server exposes REST endpoints for movie search, movie detail, image retrieval, provider discovery, translation, and modules. The integration should call a configured MetaTube server over HTTP using an optional bearer token. Mibo should not import `metatube-sdk-go` or reach into upstream internals.

## Goals / Non-Goals

**Goals:**

- Add `metatube` as a selectable metadata provider instance type in settings.
- Support MetaTube provider instances in library strategies and metadata templates for movie-oriented search, detail, image, and people metadata stages where Mibo can safely normalize the returned data.
- Normalize MetaTube search and detail responses into Mibo catalog candidates, canonical fields, identities, image candidates, people records, and metadata source evidence.
- Preserve provider-instance provenance, including MetaTube server identity, selected upstream MetaTube provider, external ID, and fallback behavior.
- Keep the integration HTTP-based and configurable per instance.

**Non-Goals:**

- Importing or vendoring `metatube-sdk-go` as a Go library.
- Implementing TV/series hierarchy support through MetaTube in the first version.
- Replacing TMDB as the default provider or changing existing TMDB/local scan behavior.
- Building a full MetaTube provider browser UI from `/v1/providers` in the first version.
- Supporting arbitrary MetaTube translation or review endpoints unless needed by the metadata pipeline.

## Decisions

### Decision: Add an HTTP-backed `metatube` provider type

Provider instance records will accept a new `metatube` provider type with JSON config fields for base URL, optional token, default upstream provider filter, fallback flag, and timeout. Runtime execution will call MetaTube server endpoints over HTTP and unwrap the standard `{ "data": ... }` response envelope.

Alternative considered: import `github.com/metatube-community/metatube-sdk-go` directly. This was rejected because MetaTube is already distributed as an API server, direct imports would couple Mibo to upstream provider internals and transitive dependencies, and provider instances naturally model external service configuration.

### Decision: Treat MetaTube as movie metadata, not TV hierarchy metadata

The first version will support movie-oriented search and detail flows. MetaTube can provide movie titles, summaries, release dates, actors, director, genres, studio-like fields, ratings, runtime, posters, thumbnails, and backdrops. It will not participate in `hierarchy` because Mibo's hierarchy pipeline expects series, season, and episode semantics that MetaTube's movie endpoints do not provide.

Alternative considered: allow MetaTube in every stage and let unsupported operations fail at runtime. This was rejected because existing provider strategy validation is type-driven and should reject invalid assignments before execution.

### Decision: Normalize provider output before catalog application

The metadata runtime should not extend TMDB-specific `detailResponse` as the long-term shape for MetaTube. Instead, MetaTube responses should be converted into provider-neutral search candidates and provider-neutral detail data before catalog fields, identities, images, people, and metadata source evidence are written.

Alternative considered: map MetaTube detail directly into TMDB detail structs. This was rejected because MetaTube external IDs are provider-specific strings, MetaTube image URLs are already concrete, and pretending they are TMDB IDs would corrupt identity and refetch behavior.

### Decision: Store provider-specific identities using `provider = metatube`

MetaTube external IDs will be stored under `provider = metatube`, with identity type values that preserve the upstream MetaTube provider namespace. For example, a MetaTube result from `fanza` with ID `abc123` should remain distinguishable from another upstream provider that returns the same raw ID. Metadata source payloads should include enough raw context to refetch the same detail later.

Alternative considered: store the upstream MetaTube provider name as Mibo's top-level provider. This was rejected because Mibo would lose the fact that the executable provider instance was MetaTube and would make provider-instance health, credentials, and provenance harder to reason about.

### Decision: UI starts with manual provider filter configuration

The settings UI will expose MetaTube fields directly and allow an optional default upstream provider string. It will not require live provider discovery before saving an instance. A later enhancement can call `/v1/providers` for suggestions and health diagnostics.

Alternative considered: dynamically populate provider choices from the configured MetaTube server during provider instance editing. This was deferred because it introduces token, network, and loading states into a form that should be saveable even when the server is temporarily unavailable.

## Risks / Trade-offs

- [Risk] MetaTube endpoints have no dedicated OpenAPI contract and may change response fields. -> Mitigation: keep the client tolerant of missing optional fields, cover the current contract with httptest fixtures, and preserve raw payload evidence for debugging.
- [Risk] Current metadata code is TMDB-shaped, so a minimal implementation could duplicate apply logic. -> Mitigation: introduce the smallest provider-neutral normalization boundary needed for TMDB and MetaTube to share catalog apply behavior.
- [Risk] MetaTube results target adult-video metadata and may not fit all Mibo library types. -> Mitigation: validate supported item types and document MetaTube as movie-oriented; do not enable hierarchy support in the first version.
- [Risk] Image handling can either proxy through MetaTube or store external URLs directly. -> Mitigation: begin by storing concrete image URLs from MetaTube detail/search responses as image candidates; use MetaTube image endpoints only when a result lacks direct URLs or a later design needs server-side processing.
- [Risk] Bearer token misconfiguration could mark an instance unavailable. -> Mitigation: reuse provider availability/failure state and record 401/403 as unavailable, 429 as cooldown when MetaTube returns those statuses.

## Migration Plan

1. Add the `metatube` provider type constant and config parsing without creating any default instances.
2. Extend provider instance APIs and UI types/forms so operators can create MetaTube instances explicitly.
3. Extend provider capability validation so `metatube` is accepted only for supported stages.
4. Add the MetaTube HTTP client and normalization layer behind the metadata runtime.
5. Update matching, manual search, apply, and refetch flows to branch through provider-neutral execution rather than TMDB-only paths where MetaTube participates.
6. Add tests with local httptest MetaTube fixtures before enabling the provider type as operational.

Rollback strategy: since no default MetaTube instances are created, disabling or deleting MetaTube provider instances restores previous behavior. Existing TMDB and `local_scan` strategies remain valid.

## Open Questions

- Should MetaTube's `number` field become the primary displayed title hint, an alternate title, or a separate catalog external attribute?
- Should the first UI expose MetaTube's upstream provider filter as one string, or allow multiple preferred provider names in order?
- Should MetaTube image endpoints be used as a Mibo-controlled proxy URL, or should Mibo store direct `cover_url` and `thumb_url` values returned by MetaTube?
