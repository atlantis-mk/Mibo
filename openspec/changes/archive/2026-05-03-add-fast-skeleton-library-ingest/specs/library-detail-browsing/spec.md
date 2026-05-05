## ADDED Requirements

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
