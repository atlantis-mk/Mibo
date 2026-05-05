# library-detail-browsing Specification

## Purpose

Define the primary media-library detail browsing experience, including accurate totals, paging, sorting, and responsive poster grids.

## Requirements

### Requirement: Library detail shows accurate browse totals
The system SHALL display the server-reported total number of browse results for the active library view and active filters instead of using only the currently loaded item count.

#### Scenario: Display filtered total count
- **WHEN** a user opens a library detail page with active filters that match 79 browse results
- **THEN** the page displays copy equivalent to `共 79 项`

### Requirement: Library detail supports browsing beyond the first page
The system SHALL provide incremental loading so users can browse all results returned by a library view, including libraries larger than the initial request size, and SHALL automatically request additional results when the load-more sentinel enters the library scroll viewport while more results are available.

#### Scenario: Load additional library results
- **WHEN** a library has more results than the initial page contains
- **THEN** the page exposes a way to load or navigate to additional results without changing the active filters

#### Scenario: Automatically load more near the bottom
- **WHEN** a user scrolls a browseable library view until the load-more sentinel intersects the library scroll viewport and additional results are available
- **THEN** the page requests the next result page using the same active filters and appends those results to the existing grid

#### Scenario: Avoid duplicate automatic loads
- **WHEN** the load-more sentinel intersects while a next-page request is already in progress or no additional results are available
- **THEN** the page does not start another next-page request

### Requirement: Library detail supports title sort direction
The system SHALL allow users to sort library results by title in ascending and descending order.

#### Scenario: Toggle title sort order
- **WHEN** a user selects title sort and toggles the direction control
- **THEN** the item order changes between ascending and descending title order using the same active filters

### Requirement: Library detail uses responsive poster grid layout
The system SHALL render library results as a poster-first responsive grid that remains usable on desktop and mobile screens.

#### Scenario: Browse posters on mobile
- **WHEN** a user opens the library detail page on a narrow viewport
- **THEN** poster cards remain readable, tappable, and vertically scrollable without horizontal page overflow

### Requirement: Library detail renders organizing media cards
The system SHALL render inventory-backed discovered media entries as media-like cards with clear organizing state rather than as raw file-list rows.

#### Scenario: Newly discovered file appears in library detail
- **WHEN** a library detail browse response includes an organizing media entry
- **THEN** the page MUST render it in the media grid using available filename-derived title, type guess, and placeholder artwork if no selected image exists
- **AND** the card MUST show copy or badges indicating that the item is still being organized

#### Scenario: Organizing card upgrades to catalog card
- **WHEN** an organizing entry is later linked to a final catalog item returned by browse APIs
- **THEN** the library detail page MUST render the catalog-backed card in place of the organizing card on refresh or query update
- **AND** it MUST NOT show both cards for the same file in the same browse view

### Requirement: Organizing cards limit final-catalog-only actions
The system SHALL avoid presenting actions on organizing cards that require a stable final catalog identity unless those actions can be safely anchored to the discovered file.

#### Scenario: User opens actions for organizing card
- **WHEN** a user interacts with an organizing media card whose final catalog item is not available
- **THEN** the UI MUST either hide final-catalog-only actions or route safe actions through the file-backed anchor
- **AND** it MUST communicate that metadata and classification are still pending when an unavailable action is relevant
