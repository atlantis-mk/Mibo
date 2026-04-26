# TV Series Metadata Governance Plan

**Date:** 2026-04-25  
**Status:** Proposed  
**Scope:** Redesign Mibo's TV-series metadata model and governance flow so shows, seasons, and episodes can be managed as first-class catalog entities instead of being inferred from episode `media_items`.

## Problem Statement

The current metadata model is adequate for movies and basic episode playback, but it is not adequate for TV-series governance.

Current state in code:

- `database.MediaItem` stores movies and episodes in one table, with optional `SeriesTitle`, `SeasonNumber`, and `EpisodeNumber` fields.
- Browse/search shows are not persistent catalog entities. `library.groupShowBrowseCandidates` builds virtual `show` cards by grouping episode rows.
- `TVSeasonMetadataCache` and `TVEpisodeMetadataCache` are TMDB/language caches, not Mibo-owned governed season/episode records.
- `MediaItem.ExternalID` stores the series-level TMDB ID for episodes, such as `tv:777`; it does not store episode-level or season-level provider IDs.
- Manual governance endpoints edit a single `MediaItem`; they cannot govern series-level identity, season metadata, missing episodes, specials, episode ordering, or locked fields across the hierarchy.
- Local series detail currently depends on available local episode rows. Provider-known but missing or unaired episodes are not durable catalog records.

This leads to concrete product gaps:

- A show has no canonical row to own title, overview, artwork, provider IDs, status, year range, or editorial locks.
- A season has no canonical row to own season poster, overview, counts, specials, or display state.
- An episode has no durable identity unless a playable file exists.
- A multi-episode file, multi-version episode, specials, extras, and date-based episodes cannot be modeled cleanly.
- Re-matching one episode duplicates series metadata work and can drift across episodes from the same series.
- Governance cannot answer “what is missing”, “what is upcoming”, or “which provider fields were overridden”.

## Emby Reference Model

Emby's public behavior provides the useful target pattern:

- Naming guidance requires TV libraries to be recognized as TV content and recommends `Series (year)/Season #/Episode` folder structure.
- Series folder and file names can include provider IDs such as `tvdb`, `tmdb`, and `imdb` tags, for example `The Vampire Diaries (2009) [tvdbid=95491]`.
- Episode naming supports multiple patterns, including `S01E01`, `1x02`, numeric episode files, date-based episodes, specials under `Season 0`/`Specials`, multi-version episodes, and multi-episode files.
- Artwork exists at multiple levels: series images, season images, and episode thumbs.
- REST APIs expose hierarchy-level operations: `/Shows/{Id}/Seasons`, `/Shows/{Id}/Episodes`, `/Shows/Missing`, and `/Shows/NextUp`.
- Returned `BaseItemDto` carries hierarchy and governance fields such as `Type`, `ParentId`, `SeriesId`, `SeasonId`, `IndexNumber`, `ParentIndexNumber`, `ProviderIds`, `People`, `Genres`, `Studios`, `UserData`, `RecursiveItemCount`, `ChildCount`, `SeasonCount`, `ParentLogoItemId`, `ParentBackdropItemId`, `LockData`, and `LockedFields`.

The important lesson is not to clone Emby wholesale. The important lesson is that a TV system needs persistent hierarchical catalog entities and separate media-file linkage.

## Emby Comparison And Adoption

Emby is better than Mibo's current model in the areas that matter most for TV metadata:

| Area | Emby approach | Mibo target decision |
|------|---------------|----------------------|
| Logical hierarchy | `BaseItemDto` has `Type`, `ParentId`, `SeriesId`, `SeasonId`, `IndexNumber`, and `ParentIndexNumber`. | Adopt this. `catalog_items` must be the single logical item graph for movie/series/season/episode/extra. |
| Provider identity | `ProviderIds` are part of item metadata. | Adopt and normalize. Use `catalog_external_ids` instead of one overloaded string field. |
| Field governance | `LockData` and `LockedFields` prevent provider refresh from overwriting curated metadata. | Adopt and make more auditable. Use `metadata_field_states` with field-level locks, provenance, and selected source. |
| User state | Emby returns `UserData` with item results for play state, favorites, resumable state, and unwatched counts. | Adopt as a projection. Add `user_item_data` and rollups instead of mixing user state into catalog rows. |
| Image inheritance | Emby exposes parent logo/backdrop/thumb IDs when child items do not have images. | Adopt. Store selected images per item and inherit parent artwork at response time. |
| TV APIs | Emby exposes `/Shows/{Id}/Seasons`, `/Shows/{Id}/Episodes`, `/Shows/Missing`, and `/Shows/NextUp`. | Adopt as thin TV convenience APIs over the generic item graph. |
| Naming support | Emby supports provider ID tags, specials, date-based episodes, multi-version episodes, and multi-episode files. | Adopt incrementally, but model all of them now through `asset_items` and `display_order`. |

The greenfield Mibo target is better than a direct Emby clone in several areas:

| Area | Emby-style limitation | Mibo improvement |
|------|----------------------|------------------|
| Metadata provenance | API exposes selected metadata, but provider/manual/local evidence is not the main public abstraction. | Keep provider snapshots and field states so every canonical value can be audited and re-canonicalized. |
| Playback asset separation | `BaseItemDto` can carry `MediaSources`, but logical item and media-source concerns are still close together. | Split logical items, playable assets, physical files, and streams into separate tables. |
| Multi-episode and multi-version modeling | Emby supports these behaviorally through naming/media source conventions. | Model them explicitly with `media_assets` plus many-to-many `asset_items`. |
| Search/filter quality | Emby DTOs can include people/genres/studios, but Mibo's current JSON fields are weak for native filtering. | Normalize people/tags/images for reliable search documents and filters. |
| Portability | Emby is a full mature server with many implicit behaviors. | Keep the Mibo model explicit and small enough to reason about in this codebase. |

Conclusion: Emby's hierarchy, provider ID, locked field, image inheritance, user-data, and TV convenience API patterns are stronger and should be copied conceptually. The proposed Mibo design should not copy Emby's whole implementation shape; it should combine Emby's proven item graph semantics with stricter provenance and cleaner asset separation.

## Greenfield Decision

If compatibility with the current `MediaItem`-centric design is not required, the best design is not “add TV tables”. The best design is to rebuild the media core around four separate layers:

- **Catalog Graph:** logical media entities users browse and govern.
- **Metadata Evidence:** provider/manual/local facts, external IDs, provenance, field locks, and canonicalization.
- **Media Inventory:** playable assets, physical files, file parts, streams, versions, extras, and storage identity.
- **User And Query Projections:** playback progress, next-up, search documents, filters, and rollups.

This removes the current mistake where one row tries to be a movie, an episode, a virtual show card, a metadata record, and a playable file anchor at the same time.

## Core Domain Model

### `catalog_items`

Logical browsable/governable entities. This is the core table.

Fields:

- `id`
- `library_id`
- `type`: `movie`, `series`, `season`, `episode`, `extra`, `collection`
- `parent_id`: direct parent; season parent is series, episode parent is season, extra parent can be series/season/episode/movie
- `root_id`: top-level root item for fast series/collection queries
- `path`: materialized hierarchy path such as `/series:1/season:2/episode:9` for portable descendant queries without database-specific `ltree`
- `sort_key`
- `display_order`: `aired`, `dvd`, `absolute`, `manual`
- `index_number`: season number for seasons, episode number for episodes
- `index_number_end`: multi-episode ranges when a logical item spans multiple provider episodes
- `parent_index_number`: season number for episodes, mirroring Emby semantics
- `absolute_number`
- `title`
- `original_title`
- `sort_title`
- `overview`
- `release_date`
- `first_air_date`
- `last_air_date`
- `year`
- `end_year`
- `runtime_seconds`
- `community_rating`
- `official_rating`
- `series_status`: `continuing`, `ended`, `unknown` for series rows
- `availability_status`: `available`, `missing`, `unaired`, `ignored`, `no_local_media`
- `governance_status`: `pending`, `matched`, `needs_review`, `locked`, `manual`, `unmatched`
- `canonical_version`
- `last_canonicalized_at`
- timestamps and soft delete

Key rules:

- Series, seasons, and episodes exist even if no local media file exists.
- Missing and unaired episodes are catalog rows, not absent rows.
- Movies are catalog rows too, not a separate special model.
- Extras are catalog items with a parent and an extra type, not fake episodes.

### `catalog_external_ids`

Provider IDs are normalized and queryable.

Fields:

- `id`
- `item_id`
- `provider`: `tmdb`, `tvdb`, `imdb`, later `anidb`, `bangumi`
- `provider_type`: `movie`, `series`, `season`, `episode`
- `external_id`
- `is_primary`
- `source`: `path_tag`, `provider_search`, `manual`, `import`, `nfo`
- `confidence`
- timestamps

Indexes:

- unique `provider, provider_type, external_id`
- `item_id, provider`

### `metadata_sources`

Raw provider/local/manual evidence snapshots.

Fields:

- `id`
- `item_id`
- `source_type`: `provider`, `local_file`, `manual`, `nfo`
- `source_name`: `tmdb`, `tvdb`, `scan`, `admin`
- `language`
- `external_id`
- `payload_json`
- `confidence`
- `fetched_at`
- `expires_at`

This table keeps every source as evidence. It is not the display model.

### `metadata_field_states`

Per-field canonicalization and governance state.

Fields:

- `id`
- `item_id`
- `field_key`: examples `title`, `overview`, `poster`, `index_number`, `air_date`, `genres`, `cast`
- `source_id`
- `value_json`
- `is_locked`
- `lock_reason`
- `edited_by_user_id`
- `edited_at`
- timestamps

Rules:

- Canonical fields on `catalog_items` are the fast read model.
- `metadata_field_states` explains where canonical values came from and whether refresh may overwrite them.
- Manual edits write field states first, then update the canonical read model.

### `item_images`

Artwork is first-class and multi-level.

Fields:

- `id`
- `item_id`
- `image_type`: `primary`, `poster`, `backdrop`, `logo`, `thumb`, `still`, `banner`, `clearart`
- `url`
- `source_id`
- `language`
- `width`
- `height`
- `is_selected`
- `sort_order`
- timestamps

Rules:

- Series, season, episode, movie, and extra images are stored the same way.
- Response mappers can inherit parent backdrop/logo/thumb without copying parent URLs into child rows.

### `people`, `item_people`, `tags`, `item_tags`

Genres, regions, studios, cast, directors, writers, creators, and guest stars should not be stored as opaque JSON in the greenfield model.

Tables:

- `people`: normalized person identity, display name, provider refs, optional image.
- `item_people`: `item_id`, `person_id`, `role_type`, `role_name`, `character_name`, `sort_order`.
- `tags`: normalized tag with `type`: `genre`, `region`, `studio`, `keyword`, `rating`.
- `item_tags`: `item_id`, `tag_id`, `source_id`, `sort_order`.

This enables reliable search/filtering without repeatedly parsing JSON fields.

### `media_assets`

Playable logical assets. An asset can represent one file, a multi-part file set, one version, one edition, or one streamable remote object.

Fields:

- `id`
- `library_id`
- `asset_type`: `main`, `version`, `extra`, `trailer`, `sample`
- `display_name`
- `edition`
- `quality_label`
- `duration_seconds`
- `status`: `ready`, `missing`, `probing`, `error`
- `probe_status`
- `technical_summary_json`
- timestamps and soft delete

### `asset_items`

Many-to-many bridge between catalog items and playable assets.

Fields:

- `id`
- `asset_id`
- `item_id`
- `role`: `primary`, `version`, `multi_episode_part`, `extra`, `trailer`
- `segment_index`
- `start_time_seconds`
- `end_time_seconds`
- `confidence`
- `source`: `scan`, `manual`, `provider_id`, `path_rule`

Why this is the cleaner design:

- Multi-episode files link one asset to multiple episode items.
- Multi-version episodes link multiple assets to one episode item.
- Extras link to their parent item without being mistaken for episodes.
- Playback can select the best asset while metadata remains attached to the logical episode/movie.

### `media_files` and `asset_files`

Physical storage inventory.

`media_files` fields:

- `id`
- `library_id`
- `storage_provider`
- `storage_path`
- `stable_identity_key`
- `provider_hashes_json`
- `size_bytes`
- `modified_at`
- `container`
- `status`
- timestamps and soft delete

`asset_files` fields:

- `id`
- `asset_id`
- `media_file_id`
- `part_index`
- `role`: `main`, `subtitle`, `external_audio`, `chapter`, `thumbnail`, `nfo`, `artwork`

### `media_streams`

Probe result per file stream.

Fields:

- `id`
- `media_file_id`
- `stream_index`
- `type`: `video`, `audio`, `subtitle`, `attachment`
- codec, language, title, channels, width, height, bitrate, duration, flags, and raw JSON

### `user_item_data`

Per-user item state, equivalent to the useful subset of Emby's `UserData`.

Fields:

- `id`
- `user_id`
- `item_id`
- `asset_id`
- `playback_position_seconds`
- `played_percentage`
- `play_count`
- `is_played`
- `is_favorite`
- `last_played_at`
- `completed_at`
- timestamps

Rules:

- Episode progress is stored against both item and selected asset.
- Series and season watched states are rollups from child episodes.
- Next-up is computed from catalog order plus `user_item_data`, not from raw file timestamps.

### `item_rollups` and `search_documents`

Read-optimized projections.

`item_rollups` fields:

- `item_id`
- `child_count`
- `recursive_child_count`
- `available_episode_count`
- `missing_episode_count`
- `unaired_episode_count`
- `played_child_count`
- `in_progress_child_count`
- `latest_air_date`
- `latest_added_at`

`search_documents` should target `item_id`, not file or asset IDs, and should denormalize selected title, original title, people, tags, provider IDs, year, rating, availability, and rollup fields.

## Metadata Governance Rules

- Provider data is evidence, not truth.
- Canonical display data lives on `catalog_items` and related normalized read tables.
- Every canonical field should be traceable to a `metadata_field_states` row.
- Manual edits lock fields by default unless the user chooses “allow provider refresh”.
- Refresh never overwrites locked fields.
- Series refresh creates/updates seasons and episodes, but it does not delete governed local/manual rows without review.
- Episode availability is computed from provider air date plus asset links.
- Parent artwork is inherited at response time, not duplicated into children.

## TV-Specific Behavior On The New Core

### Scan

1. Read storage objects into `media_files`.
2. Group files into `media_assets` by version/multipart rules.
3. Parse TV paths and filenames into catalog candidates.
4. Create or resolve `series`, `season`, and `episode` catalog items.
5. Link assets to episode catalog items through `asset_items`.
6. Queue metadata matching at the series level.

### Matching

1. Resolve series using path provider tags, existing external IDs, title/year, and provider search.
2. Fetch series detail, season list, and season episode payloads.
3. Write provider snapshots to `metadata_sources`.
4. Upsert external IDs, images, people, tags, seasons, and episodes.
5. Canonicalize unlocked fields into `catalog_items` and normalized tables.
6. Mark missing/unaired/available status from `asset_items` and air dates.

### Ordering

Default TV order is aired order. The model must support future DVD/absolute/manual ordering.

Add `item_orderings` when needed:

- `id`
- `series_id`
- `order_type`: `aired`, `dvd`, `absolute`, `manual`
- `name`
- `is_default`

Add `item_order_entries` when needed:

- `ordering_id`
- `episode_id`
- `season_number`
- `episode_number`
- `absolute_number`
- `sort_order`

Do not fake alternate orders by overwriting canonical aired season/episode numbers.

## API Design

The API should expose catalog items as the primary resource.

Read APIs:

- `GET /api/v1/items?library_id=&type=&parent_id=&query=&genre=&year=&availability=&sort=&limit=`
- `GET /api/v1/items/{id}`
- `GET /api/v1/items/{id}/children?type=&availability=&order=`
- `GET /api/v1/items/{id}/assets`
- `GET /api/v1/items/{id}/images`
- `GET /api/v1/items/{id}/people`
- `GET /api/v1/items/{id}/metadata-sources`
- `GET /api/v1/items/{id}/field-states`

TV convenience APIs can be thin wrappers over item APIs:

- `GET /api/v1/series/{id}/seasons`
- `GET /api/v1/series/{id}/episodes?season=&availability=&order=`
- `GET /api/v1/series/{id}/missing`
- `GET /api/v1/series/{id}/next-up`

Governance APIs:

- `PATCH /api/v1/items/{id}/metadata`
- `PATCH /api/v1/items/{id}/field-locks`
- `POST /api/v1/items/{id}/match-candidates`
- `POST /api/v1/items/{id}/apply-match`
- `POST /api/v1/items/{id}/refresh-metadata`
- `POST /api/v1/items/{id}/images/select`
- `POST /api/v1/assets/{asset_id}/link-item`
- `DELETE /api/v1/assets/{asset_id}/link-item/{item_id}`

Playback APIs should resolve from item to asset:

- `GET /api/v1/items/{id}/playback-options`
- `GET /api/v1/assets/{asset_id}/playback-url`

## Frontend Product Model

- Library and search render `items`, not `media_items`.
- Series detail renders children from `items/{series_id}/children`.
- Season rows are real items with artwork and counts.
- Episode rows show `available`, `missing`, `unaired`, and `ignored` states.
- Governance UI edits item metadata, field locks, selected artwork, external IDs, and asset links.
- Playback starts from an item, then chooses an asset/version.

## Implementation Plan

### Phase A: New Catalog Kernel

Goal: Introduce the replacement schema and domain services.

Tasks:

- Add `catalog_items`, `catalog_external_ids`, `metadata_sources`, `metadata_field_states`, `item_images`, `people`, `item_people`, `tags`, and `item_tags`.
- Add `media_assets`, `asset_items`, `media_files`, `asset_files`, and `media_streams`.
- Add catalog service methods for create/update/query hierarchy and canonicalization.
- Add inventory service methods for scan upsert, asset grouping, and stream probe storage.

Validation:

- `cd mibo-media-server && go test ./internal/catalog ./internal/inventory`

### Phase B: Scanner Rebuild

Goal: Make scan produce files, assets, and catalog links instead of mixed media-item rows.

Tasks:

- Replace TV scan classification with candidate extraction into series/season/episode catalog items.
- Support Emby-style ID tags in folders/files for `tmdb`, `tvdb`, and `imdb`.
- Support specials season 0, common SxxExx/1x02 forms, multi-version naming, and basic multi-episode linking.
- Write unresolved candidates as `needs_review`, not malformed catalog rows.

Validation:

- `cd mibo-media-server && go test ./internal/library -run 'Test.*Scan|Test.*TV|Test.*Asset'`

### Phase C: Metadata Engine Rebuild

Goal: Match and refresh at the correct logical level.

Tasks:

- Implement series-first matching.
- Store provider payloads in `metadata_sources`.
- Canonicalize provider/manual/local evidence into `catalog_items`, images, people, and tags.
- Respect `metadata_field_states.is_locked`.
- Compute episode availability and series/season counts.

Validation:

- `cd mibo-media-server && go test ./internal/metadata ./internal/catalog`

### Phase D: Query, Search, And Playback Contracts

Goal: Move user-facing reads to the new item/asset model.

Tasks:

- Add item, child, people, image, asset, missing, and next-up APIs.
- Build search documents from catalog items and normalized people/tags.
- Make playback resolve item to selected asset and asset to storage link.
- Move progress to item/asset identity rather than old media item identity.

Validation:

- `cd mibo-media-server && go test ./internal/httpapi ./internal/search ./internal/playback ./internal/progress`

### Phase E: Governance UI Rebuild

Goal: Make metadata governance operate on items and fields.

Tasks:

- Add item-based API client types and query helpers.
- Build series detail, season/episode hierarchy, item governance panels, field locks, image selection, and asset-link correction UI.
- Show source provenance and current lock state for governed fields.

Validation:

- `cd web && pnpm typecheck`
- `cd web && pnpm build`

## Non-Goals

- Do not clone all Emby behavior or every API field.
- Do not implement full NFO import/export in the first rebuild.
- Do not implement every local artwork filename convention in the first rebuild.
- Do not build plugins or third-party metadata provider abstraction beyond the providers needed now.
- Do not modify OpenList; it remains storage access only.

## Why This Is Better Than The Incremental Plan

- It removes the current overloaded `MediaItem` concept instead of preserving it.
- It supports movies, series, seasons, episodes, extras, collections, missing episodes, and multi-version assets with one model.
- It treats metadata as governed evidence plus canonical output, not a provider cache copied into rows.
- It makes search/filtering reliable by normalizing people, tags, images, and provider IDs.
- It keeps playback and metadata separate, which is required for missing/unaired episodes and multi-episode files.
- It gives Mibo a long-term media-server core rather than a patch around the current TV limitations.

## Acceptance Criteria

- A series, season, or episode can exist without a local file.
- Missing, unaired, ignored, and available episodes are explicit states.
- A media asset can link to one item, many items, or be one of many versions of an item.
- Provider, local, and manual metadata are traceable as evidence.
- Field-level locks prevent refresh overwrites.
- Series refresh can update seasons and episodes without breaking manual governance.
- Search, filters, next-up, and playback operate from item/asset identity rather than mixed media-item rows.
