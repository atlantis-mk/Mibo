## ADDED Requirements

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
