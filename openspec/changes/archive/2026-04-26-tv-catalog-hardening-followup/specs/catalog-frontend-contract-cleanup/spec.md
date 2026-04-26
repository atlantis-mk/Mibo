## ADDED Requirements

### Requirement: Primary frontend catalog views stop depending on legacy-shaped media adapters
The frontend SHALL use catalog-native item, detail, hierarchy, and asset contracts as the primary rendering model for search, library, home, and media detail experiences instead of normalizing catalog responses back into legacy `MediaItem`-shaped objects.

#### Scenario: Search and library render catalog-native summaries
- **WHEN** the frontend renders catalog-backed search or library results
- **THEN** it MUST consume catalog item semantics directly for type, availability, governance, and artwork without requiring legacy-only fields such as `series_title`, `source_path`, or placeholder file arrays

### Requirement: Media detail and playback entry use catalog-native hierarchy and asset semantics
The frontend SHALL render detail, hierarchy, and playback entry flows from catalog-native season, episode, progress, and asset contracts.

#### Scenario: User opens a series detail page with multiple assets
- **WHEN** a user opens a series or episode detail page that exposes catalog hierarchy and multiple linked assets
- **THEN** the UI MUST present hierarchy and playback choices from catalog-native data and MUST pass the selected asset identity through the playback flow without falling back to legacy presentation assumptions

### Requirement: Frontend empty states and actions reflect catalog availability semantics
The frontend SHALL distinguish available, missing, unaired, and unplayable catalog states in primary media views and actions.

#### Scenario: User views a provider-complete but partially local series
- **WHEN** the frontend renders a series whose catalog hierarchy includes available, missing, and unaired descendants
- **THEN** the UI MUST show those states explicitly and MUST NOT imply that every descendant has a local playable file

### Requirement: Governance entry surfaces remain catalog-native end-to-end
The frontend SHALL route from media summaries and governance worklists into catalog governance pages without converting those entry summaries into legacy media contracts first.

#### Scenario: User enters governance from a catalog summary card
- **WHEN** a user clicks into governance from a home, library, search, or governance workspace card
- **THEN** the navigation and summary rendering MUST stay rooted in catalog item identities and catalog summary data
