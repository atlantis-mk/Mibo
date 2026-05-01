## MODIFIED Requirements

### Requirement: Sidecar external identity seeds metadata enrichment
The scanner SHALL persist supported sidecar external identities as scanner evidence and provider-facing external IDs so later metadata operations can fetch detail without first performing a remote search when the library strategy permits that provider.

#### Scenario: Sidecar external identity seeds metadata enrichment
- **WHEN** a matching metadata sidecar contains a supported external identity such as a TMDB or MetaTube identifier
- **THEN** the scanner MUST persist that identity on the catalog item with scanner provenance so later metadata enrichment can fetch detail without first performing a remote search

#### Scenario: Sidecar identity records local evidence source
- **WHEN** a metadata operation uses a sidecar-provided external identity to fetch provider detail
- **THEN** the operation evidence MUST identify the scanner metadata source as the seed and the provider detail source as the applied metadata source

### Requirement: Sidecar hints respect field ownership
The scanner SHALL record supported sidecar hints as local evidence and metadata operations SHALL apply those hints through catalog field ownership and governance rules.

#### Scenario: Manual field is locked
- **WHEN** a sidecar contains a title, year, overview, or provider ID for an item whose corresponding field has been locked or manually curated
- **THEN** the scanner MUST record sidecar evidence without overwriting the protected field

#### Scenario: Local apply respects locked field
- **WHEN** a local evidence metadata operation applies sidecar hints to an item with a locked title field
- **THEN** the operation MUST skip the locked title, apply eligible unlocked fields, and report the skipped title in the operation result

#### Scenario: Local-only strategy applies sidecar evidence
- **WHEN** a library strategy permits local evidence application and a scanned item has parsed sidecar metadata
- **THEN** a local metadata operation MUST be able to apply supported sidecar hints without requiring a remote provider configuration
