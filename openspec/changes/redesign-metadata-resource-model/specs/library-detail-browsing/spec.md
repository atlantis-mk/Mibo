## MODIFIED Requirements

### Requirement: Library detail lists projected metadata
The system SHALL list items for a library from library metadata projections rather than metadata identities owned by the library.

#### Scenario: Library detail shows projected item
- **WHEN** a metadata identity has a projection row for the requested library
- **THEN** the library detail response includes the projected metadata item with library-scoped availability and resource counts

#### Scenario: Library detail excludes unprojected global metadata
- **WHEN** a metadata identity exists globally but has no projection for the requested library
- **THEN** the library detail response excludes that metadata identity

### Requirement: Library detail exposes resource version context
The system SHALL include enough resource context for the UI to show available versions for projected metadata identities.

#### Scenario: Projected item has multiple versions
- **WHEN** a projected metadata identity has multiple resources in the requested library
- **THEN** the library detail response includes version count or resource summary for that projection
