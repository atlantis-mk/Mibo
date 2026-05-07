## MODIFIED Requirements

### Requirement: Governance acts on metadata and resource links
The governance system SHALL provide actions for metadata identities, resource-to-metadata links, and library projections.

#### Scenario: Relink resource
- **WHEN** a resource was linked to the wrong metadata identity
- **THEN** a governance action can unlink it and link it to the correct metadata identity with audit evidence

#### Scenario: Hide projection
- **WHEN** a user hides a metadata identity from one library
- **THEN** the library projection is hidden without deleting the metadata identity or resource links used by other libraries

### Requirement: Metadata merge and split
The governance system SHALL support correcting global metadata identity merges and splits.

#### Scenario: Merge duplicate metadata identities
- **WHEN** two metadata identities are confirmed to represent the same work
- **THEN** governance can merge metadata fields, external IDs, resource links, projections, and user metadata data into one identity

#### Scenario: Split incorrectly merged identity
- **WHEN** resources for different works were linked to one metadata identity
- **THEN** governance can move selected resource links to a new or existing metadata identity

### Requirement: Field locks remain metadata-scoped
The governance system SHALL lock manual metadata fields on metadata identities.

#### Scenario: Locked title resists provider update
- **WHEN** a metadata title field is manually locked
- **THEN** provider updates do not overwrite that field without explicit force
