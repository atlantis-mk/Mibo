## MODIFIED Requirements

### Requirement: TV hierarchy creates global metadata identities
TV hierarchy completion SHALL create or update global series, season, and episode metadata identities.

#### Scenario: Fetch series hierarchy
- **WHEN** metadata detail for a TV series includes seasons and episodes
- **THEN** the system creates or updates global metadata identities for the series hierarchy

### Requirement: TV projections are library-scoped
TV hierarchy completion SHALL project only relevant hierarchy nodes into a library based on resources and projection policy.

#### Scenario: Library has one episode resource
- **WHEN** a library has a resource linked to one episode of a global series
- **THEN** the library projection includes the relevant episode and required parent series/season nodes according to projection policy

#### Scenario: Global episode without library relevance
- **WHEN** a global episode exists but no resource or projection policy makes it relevant to a library
- **THEN** that episode is not shown in that library's browse results

### Requirement: Episode version links
The system SHALL represent multiple resources for the same episode as versions linked to one episode metadata identity.

#### Scenario: Two files for same episode
- **WHEN** two files identify the same series, season, and episode
- **THEN** the system links both resources to one global episode metadata identity
