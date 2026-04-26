# catalog-frontend-migration Specification

## Purpose
TBD - created by syncing change tvg-catalog-kernel-remaining. Update Purpose after archive.

## Requirements

### Requirement: Frontend media surfaces use catalog item contracts
The frontend SHALL use catalog item and catalog item detail contracts as the primary types for home, library, search, and detail experiences instead of legacy `MediaItem` and `MediaItemDetail` contracts.

#### Scenario: Library and search results render catalog items
- **WHEN** the frontend renders home sections, library grids, or search results after API cutover
- **THEN** it MUST consume catalog item data, including item type, availability, governance status, and selected images, without depending on legacy-only properties such as `series_title`, `files[0]`, or `source_path`

### Requirement: Series detail renders catalog season and episode hierarchy states
The frontend SHALL render seasons and episodes from catalog hierarchy endpoints and SHALL represent available, missing, and unaired episode states distinctly.

#### Scenario: Series page shows provider-completed episode structure
- **WHEN** a user opens a series detail page for a matched series with provider-generated episodes and partially available local assets
- **THEN** the UI MUST render seasons and episodes using the catalog hierarchy, indicate which episodes are available, missing, or unaired, and avoid implying that every episode has a playable local file

### Requirement: Playback entry supports asset-aware selection
The frontend SHALL allow playback to target the default catalog asset or a user-selected asset version when multiple versions are available.

#### Scenario: User chooses a specific asset version to play
- **WHEN** an item exposes multiple playable assets such as different qualities or editions
- **THEN** the UI MUST present those asset choices and pass the chosen asset identity through the playback request flow

### Requirement: Governance UI manages field locks, evidence, images, and asset links
The frontend SHALL provide governance workflows that show canonical fields, field locks, source evidence, image selection, external identities, and linked assets for a catalog item.

#### Scenario: User reviews and updates governance state
- **WHEN** a user opens the governance workspace for a catalog item
- **THEN** the UI MUST show field values with lock state and provenance, image candidates with current selection, and linked assets with enough status detail to explain why the item is or is not playable
