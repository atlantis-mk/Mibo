## ADDED Requirements

### Requirement: Catalog detail exposes immersive presentation metadata
The system SHALL expose catalog-backed media detail metadata needed by the immersive detail page as typed item detail fields.

#### Scenario: Client requests item detail for presentation
- **WHEN** a client requests a catalog item detail
- **THEN** the response MUST include available user-facing metadata such as community rating, official rating, year and end year or air-date range, series status, child summary, selected images, external identities, and displayable tags or genres

#### Scenario: Metadata is unavailable
- **WHEN** optional presentation metadata is unavailable for an item
- **THEN** the response MUST remain valid and omit or return empty values for optional fields without failing the detail request

### Requirement: Catalog detail exposes related media candidates
The system SHALL expose deterministic related media candidates for use by detail-page related shelves.

#### Scenario: Related media can be derived
- **WHEN** related items can be derived from catalog hierarchy, same-library relationships, shared tags, or other catalog-backed criteria
- **THEN** the response or companion catalog query MUST return candidates as catalog list items with selected images, year data, availability, and child summary fields

#### Scenario: No related media can be derived
- **WHEN** no related items are available for a detail item
- **THEN** the response or companion query MUST return an empty related list rather than synthetic placeholder items

### Requirement: Episode detail supports progress-aware shelves
The system SHALL make enough episode identity and user progress data available for detail episode shelves to display watched or in-progress state.

#### Scenario: Client renders a series episode shelf
- **WHEN** a client renders a series season hierarchy for a signed-in user
- **THEN** each playable episode MUST have stable catalog item identity and enough progress state or progress lookup support to render watched and in-progress indicators
