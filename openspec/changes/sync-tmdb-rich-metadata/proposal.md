## Why

TMDB metadata matches currently populate only the catalog baseline fields, artwork, people, identities, and TV hierarchy, leaving user-visible TMDB facts such as genres, keywords, ratings, certifications, languages, status, and external IDs unused. This makes matched movies and series look incomplete even though the provider already exposes the data and Mibo already has catalog fields or tag tables that can represent the highest-value pieces.

## What Changes

- Expand TMDB movie and TV detail normalization to include rich descriptive metadata beyond the current baseline.
- Persist TMDB genres and keywords as catalog tags so detail pages, genre lists, search documents, and related-item logic can use them consistently.
- Apply TMDB community rating and region-appropriate official rating/certification to catalog items when available.
- Preserve additional TMDB external identifiers, especially IMDb IDs, as provider identities/external IDs instead of hiding them in provider payloads.
- Enrich TV series, season, and episode synchronization with the fields that Mibo can safely represent today, while avoiding broad feature modules such as reviews, recommendations, lists, and watch-provider availability.
- Keep locked-field and manual-edit governance behavior unchanged: automated TMDB enrichment MUST NOT overwrite locked user values.
- Do not introduce breaking API changes; existing catalog detail/list contracts should expose newly populated existing fields where they already exist.

## Capabilities

### New Capabilities
- `tmdb-rich-metadata`: Defines how TMDB movie, TV series, season, and episode rich metadata is normalized, persisted, and exposed through existing catalog fields, tags, identities, and provenance.

### Modified Capabilities
- `catalog-metadata-governance`: Metadata field ownership must cover newly applied rating, certification, tag, and identity outputs.
- `tv-hierarchy-metadata-completion`: TV hierarchy synchronization must preserve richer TMDB series, season, and episode detail where Mibo has stable catalog representations.

## Impact

- Backend packages affected: `internal/metadata`, `internal/catalog`, `internal/database`, and metadata tests.
- Provider API impact: TMDB detail calls will append additional endpoints such as `keywords`, `release_dates` or `content_ratings`, and `external_ids` depending on media type.
- Data impact: existing `catalog_items`, `metadata_field_states`, `tags`, `item_tags`, `catalog_external_ids`, `catalog_identities`, and `metadata_sources` should be reused before considering new tables.
- API impact: catalog responses should surface newly populated `genres`, `tags`, `community_rating`, `official_rating`, and external identity values through existing response fields.
- Test impact: movie, TV series, season, and episode metadata operation tests need regression coverage for rich fields, locked-field preservation, tag sync behavior, and source attribution.
