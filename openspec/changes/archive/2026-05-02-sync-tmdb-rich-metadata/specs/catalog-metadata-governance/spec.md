## ADDED Requirements

### Requirement: Governance applies rich provider fields through ownership policy
The system SHALL apply rich provider metadata fields through the same field ownership and provenance policy used for baseline metadata fields.

#### Scenario: Automated enrichment sees unlocked rich fields
- **WHEN** an automated TMDB match or refetch retrieves community rating, official rating, series status, last air date, or other supported rich canonical fields for an unlocked catalog item
- **THEN** the system MUST apply the values with metadata source attribution and include the fields in operation applied-field evidence

#### Scenario: Automated enrichment sees locked rich fields
- **WHEN** an automated TMDB match or refetch retrieves a rich field whose catalog field state is locked
- **THEN** the system MUST NOT overwrite the canonical value and MUST include the skipped field in operation evidence

### Requirement: Governance records provider tag provenance
The system SHALL preserve provenance for provider-sourced tag links so automated tag sync can be explained and bounded.

#### Scenario: TMDB sync writes tags
- **WHEN** a TMDB metadata operation links genre or keyword tags to a catalog item
- **THEN** governance evidence MUST be able to identify that those tag links came from the TMDB metadata source for that operation

#### Scenario: User tags coexist with provider tags
- **WHEN** a catalog item has user, scanner, or local tags and a TMDB operation synchronizes provider tags
- **THEN** the provider operation MUST NOT remove unrelated non-provider tag links
