## ADDED Requirements

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
