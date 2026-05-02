## 1. Baseline Tests And Field Inventory

- [x] 1.1 Add focused tests documenting current TMDB movie detail behavior for genres, keywords, vote average, certification, and IMDb ID before enrichment
- [x] 1.2 Add focused tests documenting current TMDB TV series detail behavior for genres, keywords, vote average, content rating, status, last air date, and external IDs before enrichment
- [x] 1.3 Add focused tests documenting current TV hierarchy season and episode enrichment behavior before adding rich fields
- [x] 1.4 Confirm existing catalog fields, tag tables, identity tables, source evidence, and response contracts can carry the accepted rich metadata without new schema

## 2. TMDB Response Models And Requests

- [x] 2.1 Extend TMDB movie detail requests to append field-oriented endpoints for keywords, release dates, and external IDs
- [x] 2.2 Extend TMDB TV series detail requests to append field-oriented endpoints for keywords, content ratings, and external IDs
- [x] 2.3 Add response structs for TMDB movie keywords, movie release-date certifications, TV keywords, TV content ratings, and external IDs
- [x] 2.4 Extend season and episode response structs only for bounded hierarchy fields that are available from existing season detail calls
- [x] 2.5 Add tests proving optional appended responses can be absent without failing metadata operations

## 3. Normalization

- [x] 3.1 Extend normalized metadata detail outputs to carry community rating, official rating, series status, last air date, tags, and secondary external identities
- [x] 3.2 Normalize TMDB movie genres and keywords into distinct provider tag outputs
- [x] 3.3 Normalize TMDB TV series genres and keywords into distinct provider tag outputs
- [x] 3.4 Normalize TMDB movie release-date certifications using a deterministic region fallback policy
- [x] 3.5 Normalize TMDB TV content ratings using the same deterministic region fallback policy
- [x] 3.6 Normalize TMDB external IDs without conflating IMDb, TVDB, Wikidata, and TMDB namespaces

## 4. Field, Tag, And Identity Application

- [x] 4.1 Route `community_rating`, `official_rating`, `series_status`, and `last_air_date` through metadata field application and locked-field governance
- [x] 4.2 Implement provider tag synchronization for TMDB `genre` and `keyword` tag links with source attribution
- [x] 4.3 Ensure provider tag synchronization preserves unrelated tag kinds such as scanner hashtags and user/local tags
- [x] 4.4 Apply secondary external identities from TMDB rich metadata through catalog identity/external-ID policy
- [x] 4.5 Include rich field, tag, and identity outcomes in metadata operation applied/skipped evidence where applicable

## 5. TV Hierarchy Enrichment

- [x] 5.1 Apply series-level rich metadata in the same operation scope as TV hierarchy completion
- [x] 5.2 Preserve supported season-level rich metadata from existing season detail responses when creating or updating season descendants
- [x] 5.3 Preserve supported episode-level rich metadata from existing season detail episode payloads when creating or updating episode descendants
- [x] 5.4 Keep local episode slot preservation and provider hierarchy mismatch safeguards unchanged while enriching descendant metadata
- [x] 5.5 Add tests for series, season, and episode rich metadata source attribution and affected scope reporting

## 6. API And Projection Verification

- [x] 6.1 Verify catalog detail responses expose populated `genres`, `tags`, `community_rating`, `official_rating`, `series_status`, identities, and dates through existing contracts
- [x] 6.2 Verify catalog search projections include rich tag text after metadata operations refresh projections
- [x] 6.3 Verify related-item discovery can use provider genre/keyword tags without relying on source payload JSON
- [x] 6.4 Add regression tests that locked rich fields are skipped without losing provider source evidence

## 7. Validation

- [x] 7.1 Run focused metadata package tests covering TMDB movie rich metadata
- [x] 7.2 Run focused metadata package tests covering TMDB TV hierarchy rich metadata
- [x] 7.3 Run catalog tests covering tag display, genre derivation, identities, and projections
- [x] 7.4 Run `go test ./...` from `mibo-media-server/` and document any unrelated pre-existing failures
