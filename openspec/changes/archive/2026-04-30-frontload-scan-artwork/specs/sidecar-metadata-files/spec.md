## MODIFIED Requirements

### Requirement: Use metadata sidecar hints safely
The scanner SHALL parse supported `.nfo` and `.json` sidecars for high-confidence local metadata hints, including supported external identity fields, and apply them only through existing catalog scan governance protections.

#### Scenario: JSON metadata improves movie classification
- **WHEN** `Movie A.mkv` has a matching JSON sidecar with title and year fields
- **THEN** the scanner SHALL use those fields as local hints when creating or updating the movie catalog item

#### Scenario: NFO metadata improves episode classification
- **WHEN** an episode video has a matching NFO sidecar with series title, season number, and episode number
- **THEN** the scanner SHALL use those fields as local hints when creating or updating the episode hierarchy

#### Scenario: Sidecar external identity seeds metadata enrichment
- **WHEN** a matching metadata sidecar contains a supported external identity such as a TMDB or MetaTube identifier
- **THEN** the scanner MUST persist that identity on the catalog item with scanner provenance so later metadata enrichment can fetch detail without first performing a remote search

#### Scenario: Curated metadata is preserved
- **WHEN** a catalog item is locked, manual, matched, or needs review
- **THEN** sidecar metadata hints SHALL NOT overwrite preserved descriptive fields for that item
