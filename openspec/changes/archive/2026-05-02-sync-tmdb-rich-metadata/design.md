## Context

Mibo's metadata operation pipeline already fetches TMDB movie and TV details, normalizes provider output, applies catalog fields through governance, writes images and people, records identities, and completes TV hierarchy descendants. The current TMDB detail request appends only `credits,images,videos`, while the normalized detail model and field application path only carry baseline fields such as title, overview, dates, runtime, images, people, external IDs, and hierarchy.

TMDB exposes more detail for both movies and TV series than Mibo currently uses. Some of it already has durable Mibo representations: `community_rating`, `official_rating`, `tags`, `genres`, provider identities, metadata source payloads, series status, and descendant episode fields. Other TMDB data, such as reviews, recommendations, similar titles, lists, and watch providers, would require product-specific UI and query semantics and should not be folded into this metadata enrichment pass.

Current code paths to keep in mind:

```text
TMDB detail request
      │
      ▼
detailResponse / seasonDetailResponse / seasonEpisodeResponse
      │
      ▼
NormalizedMetadataDetail / NormalizedMetadataHierarchy
      │
      ├─ field policy ─────────▶ catalog_items + metadata_field_states
      ├─ image policy ─────────▶ item_images
      ├─ people policy ────────▶ people + item_people
      ├─ identity policy ──────▶ catalog_external_ids + catalog_identities
      └─ hierarchy policy ─────▶ seasons + episodes + descendant sources
```

## Goals / Non-Goals

**Goals:**

- Populate high-value TMDB movie and TV fields that already map naturally to Mibo's catalog model.
- Use existing catalog fields for ratings, certifications, status, dates, runtime, and identities.
- Use `tags`/`item_tags` for genres and keywords so search, display, and related-item behavior can reuse the same data.
- Preserve metadata source attribution and locked-field behavior for every newly applied canonical field.
- Keep movie, TV series, season, and episode enrichment consistent while acknowledging that TMDB exposes different append endpoints for each media type.

**Non-Goals:**

- Do not implement reviews, recommendations, similar-title shelves, watch-provider availability, lists, or collection browsing as part of this change.
- Do not add new frontend layouts beyond exposing fields that existing catalog contracts already return.
- Do not introduce a general arbitrary provider-payload browser.
- Do not replace the existing metadata pipeline or profile model.

## Decisions

### Reuse Existing Catalog Storage First

The enrichment should map to existing durable structures before adding schema:

| Metadata | Preferred storage |
|---|---|
| Genres | `tags.kind = "genre"`, `item_tags` |
| Keywords | `tags.kind = "keyword"`, `item_tags` |
| Community rating | `catalog_items.community_rating` via field policy |
| Official rating/certification | `catalog_items.official_rating` via field policy |
| TMDB/IMDb IDs | `catalog_external_ids` / `catalog_identities` |
| Original language / countries / languages / tagline | field state if stable canonical support exists, otherwise source payload until UI/storage is designed |
| Series status | `catalog_items.series_status` |

Alternative considered: storing all rich fields only in `metadata_sources.payload_json`. That preserves data but does not make it queryable, displayable through current contracts, or available to related-item/search logic. Payload-only storage is acceptable for low-priority fields but not for genres, keywords, ratings, certifications, or identities.

### Treat Tags As Provider-Sourced Metadata With Replacement Semantics Per Kind

For a TMDB operation, provider-owned `genre` and `keyword` tags should be synchronized without deleting unrelated tag kinds such as scanner `hashtag` tags. The safest mental model is:

```text
TMDB source for item X
      ├─ replace TMDB-owned genre links
      ├─ replace TMDB-owned keyword links
      └─ leave hashtag/user/local tags alone
```

The current `item_tags.source_id` column can attribute provider tag links. If source-specific replacement is too narrow because source IDs change per refetch, the implementation should define a bounded provider replacement rule for the affected tag kinds while preserving non-provider tag kinds.

Alternative considered: encode genres as `metadata_field_states["genres"]`. That conflicts with the existing catalog detail contract, where `Genres` is derived from tag details, and would split genre truth across two systems.

### Append Only Field-Oriented TMDB Endpoints

TMDB detail calls should append endpoints that directly support catalog metadata:

| Media type | Append endpoints |
|---|---|
| Movie | `credits,images,videos,keywords,release_dates,external_ids` |
| TV series | `credits,images,videos,keywords,content_ratings,external_ids` |
| TV season | `credits,images,external_ids` where useful and supported |
| TV episode | use season embedded episode details for baseline; only call episode detail append endpoints if needed for missing fields |

This keeps request count and payload size controlled. The change should not append reviews, recommendations, similar, lists, or watch providers because those are separate product features.

### Pick Certifications By Region Policy

Official rating should be selected deterministically from TMDB regional certification data. The policy should prefer a configured or language-derived region when available, then fall back to commonly useful regions, then the first non-empty certification.

```text
candidate regions
      │
      ├─ explicit preferred region, if later added
      ├─ region inferred from language tag, e.g. zh-CN → CN
      ├─ US
      └─ first non-empty certification
```

The selected value should be stored as the canonical `official_rating` and the full raw response can remain in provider source payload for audit.

### Keep Low-Priority TMDB Fields In Source Payload Until Product Semantics Exist

Fields such as `budget`, `revenue`, `production_companies`, `homepage`, `popularity`, `vote_count`, `adult`, and `belongs_to_collection` are useful but need product decisions before becoming canonical fields. This change should not add new catalog columns for them unless implementation discovers an existing field with clear semantics.

## Risks / Trade-offs

- Provider tag replacement could accidentally remove user-curated tags if kind/source boundaries are wrong → Restrict automated replacement to provider-owned `genre` and `keyword` links and preserve unrelated kinds.
- TMDB append payloads can increase latency and rate-limit exposure → Append only metadata endpoints needed for the accepted fields and keep hierarchy calls bounded to existing TV sync behavior.
- Certification rules vary by country and can surprise users → Use deterministic fallback and keep raw source payload for later policy changes.
- `community_rating` currently needs field-application support → Route it through the same governance policy as other canonical fields so locked/manual behavior remains consistent.
- TV episode detail richness may require extra per-episode calls if season embedded detail is insufficient → Prefer season detail embedded episode data first; treat per-episode calls as a later optimization only if required by specs.

## Migration Plan

- No destructive migration is required for existing catalog items.
- New metadata applies on match, manual apply, and refetch for future operations.
- Existing matched items can gain rich metadata through refetch or a later explicit backfill/rebuild task.
- Rollback should be safe by stopping new rich field writes; existing tags/ratings/certifications remain ordinary catalog metadata.

## Open Questions

- Should the preferred certification region be derived only from TMDB language, or should metadata profiles eventually expose a separate region setting?
- Should keywords use `kind = "keyword"` or an existing broader `topic` convention in places that already create topic tags manually?
- Should `original_language`, `production_countries`, and `spoken_languages` become first-class fields later, or remain source-payload-only until a UI need appears?
