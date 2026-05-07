## MODIFIED Requirements

### Requirement: Metadata operations target global metadata
Metadata match, refetch, manual apply, and local apply operations SHALL target global metadata identities.

#### Scenario: Match resource-created metadata item
- **WHEN** a resource creates or links to a metadata identity requiring enrichment
- **THEN** the metadata operation runs against the metadata identity, not a library-owned catalog row

#### Scenario: Existing external ID skips search
- **WHEN** a metadata identity already has a high-confidence external ID for the requested provider type
- **THEN** the operation can skip search and fetch detail directly for that metadata identity

### Requirement: Metadata operation records context
Metadata operations SHALL record triggering library/profile/provider/language context separately from metadata ownership.

#### Scenario: Library-triggered fetch
- **WHEN** a library scan triggers a metadata fetch
- **THEN** the stored metadata source records provider instance, metadata profile, language, and triggering context without assigning metadata ownership to the library

### Requirement: Metadata fetch deduplication
The metadata pipeline SHALL deduplicate equivalent metadata fetches for the same metadata identity, stage, language, and provider context.

#### Scenario: Two libraries trigger same fetch
- **WHEN** two libraries link resources to the same metadata identity and request the same provider/language metadata
- **THEN** the system avoids duplicate equivalent metadata fetch work
