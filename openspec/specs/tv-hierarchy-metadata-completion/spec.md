# tv-hierarchy-metadata-completion Specification

## Purpose
Define durable TV hierarchy metadata completion behavior for series, season, and episode descendants, including provider identity, evidence, artwork, availability, and hierarchy-specific reads.
## Requirements
### Requirement: Descendant TV catalog items retain durable provider identity and evidence
The system SHALL persist season-level and episode-level provider identities, evidence snapshots, and artwork candidates for catalog descendants that are generated or updated from a series-root metadata match.

#### Scenario: Series match populates season and episode descendants
- **WHEN** a series catalog item is matched or refreshed from a TV metadata provider
- **THEN** the system MUST ensure the corresponding season and episode catalog items retain durable descendant identities, source evidence, and artwork candidates needed by governance and descendant APIs

### Requirement: TV hierarchy queries are served from explicit catalog hierarchy reads
The system SHALL expose season, episode, and child-list reads from explicit catalog hierarchy queries instead of relying on legacy grouping behavior or compatibility-only presentation logic.

#### Scenario: Client requests a series hierarchy view
- **WHEN** a client requests children, seasons, or episodes for a catalog-backed series
- **THEN** the system MUST serve the hierarchy from durable catalog parent-child relationships and descendant state rather than reconstructing the hierarchy from legacy media rows

### Requirement: Missing and unaired TV descendants remain explicit catalog states
The system SHALL preserve missing and unaired episodes as durable catalog descendants with explicit availability semantics even when no local playable asset exists.

#### Scenario: Provider-known episode has no local asset
- **WHEN** provider metadata includes an episode that has not been scanned locally or has a future air date
- **THEN** the corresponding catalog episode MUST remain queryable with an explicit missing or unaired availability state instead of disappearing from the hierarchy

### Requirement: TV convenience reads expose hierarchy-specific operational views
The system SHALL provide convenience reads for TV-specific views such as missing episodes and next-up calculations where those views are not yet available from the current cutover.

#### Scenario: Client requests a missing or next-up series view
- **WHEN** a client requests a TV convenience view for a matched series
- **THEN** the system MUST derive the response from catalog hierarchy, descendant availability, and user progress data without depending on legacy media-item semantics

### Requirement: Episode descendants retain episode-level metadata after series sync
The system SHALL retain episode-level metadata on catalog descendants generated or updated by series-root provider sync.

#### Scenario: Series sync returns episode details
- **WHEN** a matched TV series provider response includes season episodes with names, overviews, air dates, runtimes, still images, and provider episode IDs
- **THEN** the corresponding catalog episode descendants MUST retain those values as item fields, selected image candidates, source evidence, and external identities

#### Scenario: Local episode already exists
- **WHEN** provider sync finds an existing local episode with matching season and episode numbers
- **THEN** the system MUST enrich that existing descendant instead of creating a duplicate episode for the same provider slot

### Requirement: Episode-level credits are synchronized when provider data exists
The system SHALL persist episode-level people data when the metadata provider exposes it.

#### Scenario: Provider returns episode credits
- **WHEN** a provider response includes episode directors, cast, guest stars, or equivalent credits for a catalog episode
- **THEN** the system MUST persist those people against the episode descendant with role information and avatar URLs when available

#### Scenario: Provider lacks episode credits
- **WHEN** no episode-level people data is available from the provider
- **THEN** the system MUST leave the episode people list empty rather than copying series people into descendant persistence automatically

### Requirement: Episode matching reports descendant-specific outcomes
The system SHALL report matching outcomes in terms of the episode or season where the user initiated the action.

#### Scenario: User rematches from an episode page
- **WHEN** a user triggers match or refresh from a catalog episode
- **THEN** the system MUST synchronize through the appropriate series provider identity while surfacing whether the opened episode gained or retained the expected descendant identity and artwork

#### Scenario: Provider hierarchy does not contain the local episode slot
- **WHEN** a local episode's season or episode number cannot be found in the matched provider hierarchy
- **THEN** the system MUST preserve the local item and mark or surface the mismatch for governance review instead of silently linking it to an unrelated provider episode

### Requirement: TV hierarchy distinguishes consumer-local and operational-complete views
The system SHALL preserve complete provider-known TV hierarchy state while allowing consumer detail views to present only local playable episode descendants.

#### Scenario: Provider sync creates missing descendants
- **WHEN** provider metadata includes episodes that have no local file or have not aired yet
- **THEN** the catalog MUST keep those missing or unaired episode descendants with explicit availability state for governance and operational reads

#### Scenario: Consumer detail reads a series hierarchy
- **WHEN** the default consumer series detail view reads seasons and episodes for display
- **THEN** the hierarchy used by that view MUST exclude missing and unaired episode descendants that do not have local playable files

#### Scenario: Operational view reads complete hierarchy state
- **WHEN** a missing-episode, metadata governance, or explicit availability query reads TV hierarchy state
- **THEN** the system MUST expose the complete matching set of provider-known descendants including missing and unaired episodes

### Requirement: TV hierarchy synchronization resolves the effective library metadata profile
The system SHALL resolve the effective library metadata profile before executing rooted TV metadata synchronization for a series, season, or episode initiated action.

#### Scenario: Episode-triggered match uses the library profile
- **WHEN** a user triggers metadata matching from an episode that belongs to a library with a bound metadata profile
- **THEN** the system MUST resolve that library profile and use its configured search, detail, hierarchy, and fallback behavior while still rooting the operation at the series catalog item

### Requirement: TV descendant identities remain provider-normalized across profile stages
The system SHALL preserve season and episode descendant identity and evidence semantics even when TV metadata stages are supplied by a profile-selected provider instance rather than a single global provider configuration.

#### Scenario: Profile-selected provider sync populates descendants
- **WHEN** a TV metadata profile selects a provider instance that returns normalized season and episode hierarchy detail
- **THEN** the system MUST persist descendant identities, evidence, and artwork candidates for the generated or updated seasons and episodes using the same durable catalog hierarchy rules as the existing rooted sync flow

### Requirement: TV profile fallback does not bypass hierarchy mismatch safeguards
The system SHALL preserve existing hierarchy mismatch protections when a TV metadata profile falls back between configured provider instances.

#### Scenario: Fallback provider lacks the local episode slot
- **WHEN** a profile falls back to another provider instance and that provider's hierarchy does not contain the local episode's expected season or episode slot
- **THEN** the system MUST preserve the local descendant, surface the mismatch for governance review, and MUST NOT silently link the episode to an unrelated provider slot

### Requirement: TV hierarchy roots are scanner-identity stable
The system SHALL create and update TV hierarchy descendants under a stable series root derived from scanner, sidecar, provider, or manual identity rather than from per-file title inference alone.

#### Scenario: Files in one TV directory have inconsistent series titles
- **WHEN** multiple files in the same TV work directory resolve to episode slots but expose different filename title prefixes
- **THEN** the system MUST create or reuse one series root and place the episode descendants under that root

#### Scenario: Provider sync enriches scanner-created hierarchy
- **WHEN** a scanner-created series root is later matched to a TV metadata provider
- **THEN** provider season and episode metadata MUST enrich descendants under the existing series root instead of creating a second provider-only hierarchy

### Requirement: Episode metadata sync preserves local scanner slots
The system SHALL preserve local episode slots created by scanner identity when provider hierarchy sync cannot find an exact provider descendant.

#### Scenario: Provider lacks local episode slot
- **WHEN** a local episode exists under a scanner-created series but the matched provider hierarchy does not contain that season and episode number
- **THEN** the system MUST preserve the local episode and surface the mismatch for governance review instead of linking it to an unrelated provider episode or deleting it
