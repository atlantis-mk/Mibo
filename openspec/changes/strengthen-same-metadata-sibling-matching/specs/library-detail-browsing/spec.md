## MODIFIED Requirements

### Requirement: Library detail renders organizing media cards
The system SHALL render inventory-backed or resource-backed unresolved media entries as media-like cards with clear organizing state rather than as raw file-list rows, SHALL keep those entries visible until a final metadata-backed replacement is ready, and SHALL upgrade same-metadata sibling matches into the existing metadata-backed card instead of leaving duplicate organizing entries behind.

#### Scenario: Newly discovered file appears in library detail
- **WHEN** a library detail browse response includes an organizing media entry
- **THEN** the page MUST render it in the media grid using available filename-derived title, type guess, and placeholder artwork if no selected image exists
- **AND** the card MUST show copy or badges indicating that the item is still being organized

#### Scenario: Organizing card upgrades to catalog card
- **WHEN** an organizing entry is later linked to a final catalog item returned by browse APIs
- **THEN** the library detail page MUST render the catalog-backed card in place of the organizing card on refresh or query update
- **AND** it MUST NOT show both cards for the same file or work group in the same browse view

#### Scenario: New sibling resource upgrades an existing metadata card
- **WHEN** a newly scanned resource is accepted as another version of an existing metadata-backed movie or episode item
- **THEN** the browse response MUST prefer the existing metadata-backed card and version context over a new organizing card for that resource

#### Scenario: Low-confidence sibling match remains visible before metadata reuse
- **WHEN** scan or matching work has produced a guarded or review-required same-metadata candidate that does not yet have an accepted metadata link
- **THEN** the browse response MUST continue to include an organizing card for that unresolved resource
