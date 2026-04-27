## ADDED Requirements

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
