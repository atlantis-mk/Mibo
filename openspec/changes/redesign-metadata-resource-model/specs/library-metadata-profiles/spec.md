## MODIFIED Requirements

### Requirement: Library metadata profiles provide fetch context
Library metadata profiles SHALL provide provider and language context for metadata fetches without owning metadata identities.

#### Scenario: Library triggers metadata fetch
- **WHEN** a resource in a library links to a metadata identity needing enrichment
- **THEN** the metadata fetch uses the library's metadata strategy as context and stores the source on the metadata identity

### Requirement: Display language uses library context
The system SHALL use library metadata preferences to choose display fields from available metadata sources and field states.

#### Scenario: Same metadata in two language contexts
- **WHEN** the same metadata identity is projected into two libraries with different preferred metadata languages
- **THEN** each library projection can select display fields according to its own language preference
