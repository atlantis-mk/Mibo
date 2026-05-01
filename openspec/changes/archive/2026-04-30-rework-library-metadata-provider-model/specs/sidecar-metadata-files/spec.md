## MODIFIED Requirements

### Requirement: Use metadata sidecar hints safely
The scanner SHALL parse supported `.nfo` and `.json` sidecars into high-confidence local metadata evidence, SHALL record that parsed evidence for reuse by the built-in local scan metadata provider, and SHALL apply any scan-time hints only through existing catalog scan governance protections.

#### Scenario: JSON metadata improves movie classification
- **WHEN** `Movie A.mkv` has a matching JSON sidecar with title and year fields
- **THEN** the scanner SHALL record parsed title and year evidence for the built-in local scan provider and MAY use those fields as local hints when creating or updating the movie catalog item

#### Scenario: NFO metadata improves episode classification
- **WHEN** an episode video has a matching NFO sidecar with series title, season number, and episode number
- **THEN** the scanner SHALL record parsed hierarchy and external-ID evidence for later local scan detail execution and SHALL use those fields as local hints when creating or updating the episode hierarchy

#### Scenario: Curated metadata is preserved
- **WHEN** a catalog item is locked, manual, matched, or needs review and local metadata evidence is applied during scan-time classification or built-in local scan refresh
- **THEN** sidecar metadata hints SHALL NOT overwrite preserved descriptive fields for that item
