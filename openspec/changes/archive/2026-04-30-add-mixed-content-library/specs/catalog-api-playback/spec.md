## ADDED Requirements

### Requirement: Catalog APIs expose mixed-library results as standard media items
The system SHALL expose items scanned from mixed content libraries through existing catalog movie and series response semantics without requiring clients to handle a new catalog item type.

#### Scenario: Client browses a mixed content library
- **WHEN** a client requests the item list for a mixed content library after scanning has produced movie and series catalog items
- **THEN** the API SHALL return standard catalog list items for those movies and series with the same identity, availability, artwork, and child summary fields used by dedicated movie and show libraries

#### Scenario: Client opens a mixed-library item detail
- **WHEN** a client requests detail or playback for a movie or series produced from a mixed content library
- **THEN** the API SHALL use the existing movie or series detail and playback contracts for that item type
