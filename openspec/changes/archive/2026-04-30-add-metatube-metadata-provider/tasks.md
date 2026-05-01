## 1. Provider Configuration Model

- [x] 1.1 Add the `metatube` metadata provider type constant and database/settings config keys for base URL, token, upstream provider filter, fallback flag, and timeout.
- [x] 1.2 Extend provider instance input/output DTOs and config resolution so MetaTube settings are saved, masked where appropriate, and returned through metadata provider APIs.
- [x] 1.3 Extend provider operational checks so configured, enabled MetaTube instances can execute and 401/403/429 failures update availability or cooldown state.
- [x] 1.4 Add backend settings tests for creating, updating, resolving, and listing MetaTube provider instances.

## 2. Stage Capability Validation

- [x] 2.1 Extend provider stage validation so MetaTube is accepted for supported movie metadata stages and rejected for hierarchy.
- [x] 2.2 Update metadata template and library strategy tests to cover valid MetaTube search/detail assignments and invalid hierarchy assignment.
- [x] 2.3 Ensure existing TMDB, TVDB configuration-only, and `local_scan` validation behavior remains unchanged.

## 3. MetaTube HTTP Client

- [x] 3.1 Add a small internal MetaTube HTTP client that applies base URL normalization, optional bearer auth, request timeout, response envelope decoding, and HTTP error mapping.
- [x] 3.2 Implement movie search request support for `/v1/movies/search` with query, optional provider filter, and fallback flag.
- [x] 3.3 Implement movie detail request support for `/v1/movies/{provider}/{id}` and tolerant decoding of optional fields.
- [x] 3.4 Add httptest-based client tests for unauthenticated requests, bearer-auth requests, search/detail decoding, not-found errors, auth failures, and rate-limit failures.

## 4. Metadata Runtime Normalization

- [x] 4.1 Introduce the minimal provider-neutral search candidate and detail normalization boundary needed for TMDB and MetaTube execution to share catalog apply behavior.
- [x] 4.2 Convert MetaTube search results into Mibo manual/automated search candidates with provider `metatube` and provider-specific external IDs.
- [x] 4.3 Convert MetaTube detail responses into normalized catalog fields, image candidate URLs, people data, genres, runtime, release date, identity, and raw payload evidence.
- [x] 4.4 Preserve existing TMDB matching, manual search, apply, hierarchy, and refetch behavior while routing MetaTube selections through the new normalization path.

## 5. Catalog Application And Refetch

- [x] 5.1 Apply normalized MetaTube detail data through catalog governance so locked fields and review states retain existing protections.
- [x] 5.2 Record MetaTube external identities with provider `metatube` and an identity key that preserves upstream provider and upstream ID.
- [x] 5.3 Record metadata source evidence with MetaTube provider-instance ID/name, upstream provider, upstream item ID, raw payload summary, and fallback summary.
- [x] 5.4 Implement MetaTube refetch using the stored MetaTube identity and current library strategy detail provider selection.
- [x] 5.5 Add catalog metadata tests for MetaTube match, manual search, apply candidate, refetch, provenance, identity separation from TMDB, and missing-identity errors.

## 6. Frontend Settings UI

- [x] 6.1 Extend frontend API types for MetaTube provider settings and provider instance input.
- [x] 6.2 Add `MetaTube` to the provider instance type selector with fields for base URL, token, default upstream provider filter, fallback flag, and timeout.
- [x] 6.3 Update provider cards and settings summaries to display MetaTube configuration state without exposing tokens.
- [x] 6.4 Ensure metadata template/profile provider selectors can include operational MetaTube instances for supported stages and surface backend validation errors for unsupported stages.

## 7. Verification

- [x] 7.1 Run focused backend tests for settings and metadata provider runtime changes.
- [x] 7.2 Run `go test ./...` from `mibo-media-server/`.
- [x] 7.3 Run frontend typecheck from `web/`.
- [x] 7.4 Manually verify that a MetaTube provider instance can be created, selected in a metadata strategy, used for search/detail, and recorded in governance evidence.
