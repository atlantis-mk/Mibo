## ADDED Requirements

### Requirement: Governance surfaces sanitized provider diagnostics
The governance workspace SHALL be able to surface sanitized storage-provider diagnostics for catalog items and linked assets without exposing raw provider internals or changing governance correction semantics.

#### Scenario: User reviews linked asset provider diagnostics
- **WHEN** governance presents a catalog item or descendant with linked local/provider assets
- **THEN** the workspace MUST be able to show safe source context such as storage provider name, provider-reported driver identity, available hash keys, object type hints, and provider metadata presence indicators

#### Scenario: Provider diagnostics include sensitive OpenList internals
- **WHEN** the underlying storage provider exposes signed path tokens, mount details, write/upload flags, or auth-bearing URLs
- **THEN** governance MUST NOT expose those raw values and MUST preserve existing asset-link, evidence, field-lock, and image-selection correction behavior
