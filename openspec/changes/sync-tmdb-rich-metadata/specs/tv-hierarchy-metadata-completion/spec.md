## MODIFIED Requirements

### Requirement: Episode descendants retain episode-level metadata after series sync
The system SHALL retain episode-level metadata on catalog descendants generated or updated by series-root provider sync, including supported baseline fields and rich provider fields that are available without unbounded per-episode requests.

#### Scenario: Series sync returns episode details
- **WHEN** a matched TV series provider response includes season episodes with names, overviews, air dates, runtimes, still images, and provider episode IDs
- **THEN** the corresponding catalog episode descendants MUST retain those values as item fields, selected image candidates, source evidence, and external identities

#### Scenario: Series sync returns episode ratings or external IDs
- **WHEN** TMDB season or episode detail provides supported episode-level ratings, certifications, or external IDs within the bounded hierarchy sync
- **THEN** the corresponding catalog episode descendants MUST retain those values where Mibo has stable catalog fields or identity storage

#### Scenario: Local episode already exists
- **WHEN** provider sync finds an existing local episode with matching season and episode numbers
- **THEN** the system MUST enrich that existing descendant instead of creating a duplicate episode for the same provider slot

### Requirement: TV hierarchy synchronization preserves rich season metadata
The system SHALL preserve supported TMDB season-level rich metadata when creating or updating catalog season descendants from a series-root sync.

#### Scenario: Season detail includes supported fields
- **WHEN** TMDB season detail includes name, overview, air date, poster, season ID, external IDs, credits, or images
- **THEN** the catalog season descendant MUST retain the supported fields, artwork, identities, and source evidence without losing existing local descendant links

#### Scenario: Season detail omits optional fields
- **WHEN** TMDB season detail omits optional rich fields such as external IDs or credits
- **THEN** hierarchy synchronization MUST still create or update the season and its episodes from available baseline data

### Requirement: TV series rich metadata coexists with hierarchy completion
The system SHALL apply series-level rich metadata and descendant hierarchy updates as one metadata operation scope.

#### Scenario: Series detail and hierarchy both contain updates
- **WHEN** a TMDB TV sync returns series-level genres, keywords, ratings, certifications, status, images, people, and season or episode hierarchy data
- **THEN** the system MUST apply series fields, tags, identities, images, people, and descendant updates within the same operation result and affected scope
