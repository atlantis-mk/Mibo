## ADDED Requirements

### Requirement: Series detail hero plays a local episode target
The immersive media detail page SHALL use a series playback target for the primary series play action when local episodes exist.

#### Scenario: Series has a continue target
- **WHEN** a user opens a series detail page whose detail response identifies an unfinished local episode target
- **THEN** the primary action MUST be labeled as continue playback and MUST open playback for that episode item and selected asset

#### Scenario: Series has a first local episode target
- **WHEN** a user opens a series detail page with locally playable episodes and no unfinished episode target
- **THEN** the primary action MUST be labeled as play and MUST open playback for the earliest local episode target

#### Scenario: Series has no local playback target
- **WHEN** a user opens a series detail page with no locally playable episode target
- **THEN** the primary play action MUST be disabled or replaced with clear unavailable feedback without navigating to an unplayable series item

### Requirement: Series detail shelves show local episode information by default
The immersive media detail page SHALL render the default series episode shelves from local playable episode data only.

#### Scenario: Series contains unavailable provider episodes
- **WHEN** the series hierarchy contains local playable episodes as well as missing or unaired provider-known episodes
- **THEN** the default `剧集信息` shelf MUST display only local playable episode cards and MUST calculate displayed season episode counts from those cards

#### Scenario: A season has no local playable episodes
- **WHEN** a season contains only missing, unaired, or otherwise unplayable episode descendants
- **THEN** the default series detail shelf MUST omit that season rather than showing non-playable episode cards

#### Scenario: User opens an unavailable episode directly
- **WHEN** a user explicitly opens a missing or unaired episode detail page from a governance or missing-episode workflow
- **THEN** the page MUST preserve the episode detail unavailable state and MUST NOT pretend that the episode is locally playable
