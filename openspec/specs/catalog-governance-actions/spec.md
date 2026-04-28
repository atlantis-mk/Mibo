# catalog-governance-actions Specification

## Purpose
Define governance workflows for reviewing and correcting catalog hierarchy, metadata evidence, artwork, and asset-link state without corrupting unrelated catalog data.

## Requirements

### Requirement: Governance supports actionable asset-link correction
The governance workspace SHALL allow users to review and correct item-to-asset linkage when local assets are attached to the wrong hierarchy position or require manual confirmation.

#### Scenario: User repairs an incorrect episode asset link
- **WHEN** governance detects or presents an asset that is linked to an incorrect catalog descendant
- **THEN** the user MUST be able to review the linkage context and apply a corrective action without editing unrelated metadata state

### Requirement: Governance surfaces hierarchy mismatch review
The governance workflow SHALL expose mismatches between provider-derived hierarchy and local asset linkage so users can review missing, extra, or ambiguously linked descendants.

#### Scenario: Provider hierarchy conflicts with local scan structure
- **WHEN** a matched series contains provider descendants that do not line up cleanly with scanned local assets
- **THEN** the governance experience MUST surface the mismatch with enough child, availability, and linkage context for a user to decide the next corrective action

### Requirement: Governance corrections preserve independent field, evidence, and image state
The system SHALL ensure that asset-link corrections and hierarchy-review actions do not overwrite unrelated field locks, source evidence, image selections, or external identities.

#### Scenario: User corrects linkage on an item with locked fields
- **WHEN** a user performs an asset-link correction on a catalog item that already has locked fields and selected images
- **THEN** the linkage change MUST update only the intended relationship state and MUST preserve existing field locks, evidence, and image-selection decisions

### Requirement: Descendant governance remains available for seasons and episodes
The governance model SHALL support descendant season and episode workspaces with the same core evidence, artwork, and linkage concepts needed for TV-first review.

#### Scenario: User opens governance for a season or episode descendant
- **WHEN** a user opens governance for a season or episode that exists under a matched series
- **THEN** the workspace MUST expose descendant-specific evidence, artwork, and linked-asset context rather than only series-root state

### Requirement: Governance supports episode hierarchy correction
The governance workflow SHALL allow users to correct catalog episode season and episode numbering when local scan results do not match provider hierarchy.

#### Scenario: User corrects episode numbering
- **WHEN** a user identifies that a local episode is assigned to the wrong season or episode number
- **THEN** governance MUST provide a bounded correction path that updates hierarchy/index linkage and refreshes affected projections without overwriting unrelated metadata fields

#### Scenario: Correction conflicts with existing descendant
- **WHEN** the target season and episode slot already has another catalog descendant
- **THEN** governance MUST surface the conflict and require an explicit asset or hierarchy decision rather than merging records silently

### Requirement: Governance supports episode asset relinking
The governance workflow SHALL allow users to move or add linked assets between episode descendants within the current series hierarchy.

#### Scenario: Asset is linked to wrong episode
- **WHEN** an asset is linked to one episode but should belong to another episode in the same series hierarchy
- **THEN** governance MUST allow the user to move or copy the asset link to the intended descendant while preserving asset file links and stream metadata

#### Scenario: Relink target is outside the current hierarchy
- **WHEN** a requested episode asset relink targets an unrelated item outside the current series hierarchy
- **THEN** the system MUST reject or block the operation to avoid cross-library or unrelated hierarchy corruption

### Requirement: Governance supports multi-episode segment review
The governance workflow SHALL expose and correct multi-episode asset segment links.

#### Scenario: File spans multiple episodes
- **WHEN** one local file is linked to multiple episode descendants as segment or multi-episode parts
- **THEN** governance MUST show the linked episode order and segment indexes so a user can verify or correct the mapping

#### Scenario: Segment mapping is corrected
- **WHEN** a user changes a multi-episode segment mapping
- **THEN** the system MUST update only the intended asset-item relationships and MUST preserve unrelated field locks, evidence, images, and provider identities

### Requirement: Governance surfaces sanitized provider diagnostics
The governance workspace SHALL be able to surface sanitized storage-provider diagnostics for catalog items and linked assets without exposing raw provider internals or changing governance correction semantics.

#### Scenario: User reviews linked asset provider diagnostics
- **WHEN** governance presents a catalog item or descendant with linked local/provider assets
- **THEN** the workspace MUST be able to show safe source context such as storage provider name, provider-reported driver identity, available hash keys, object type hints, and provider metadata presence indicators

#### Scenario: Provider diagnostics include sensitive OpenList internals
- **WHEN** the underlying storage provider exposes signed path tokens, mount details, write/upload flags, or auth-bearing URLs
- **THEN** governance MUST NOT expose those raw values and MUST preserve existing asset-link, evidence, field-lock, and image-selection correction behavior
