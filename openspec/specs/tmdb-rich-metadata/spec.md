# tmdb-rich-metadata Specification

## Purpose
Define normalization and governed application of supported rich TMDB metadata for movies, TV series, and provider-sourced catalog tags.

## Requirements
### Requirement: TMDB movie rich metadata is normalized from detail responses
The system SHALL normalize supported TMDB movie rich metadata from detail responses and appended field-oriented endpoints before applying catalog updates.

#### Scenario: Movie detail includes rich fields
- **WHEN** a TMDB movie detail response includes genres, keywords, vote average, release-date certifications, external IDs, original language, production countries, spoken languages, status, tagline, or collection data
- **THEN** the system MUST normalize the supported catalog fields, tags, ratings, certifications, and identities without requiring a second manual metadata operation

#### Scenario: Movie detail omits optional rich fields
- **WHEN** TMDB omits optional movie rich metadata such as keywords, certifications, or external IDs
- **THEN** the system MUST still apply available baseline metadata and MUST NOT fail the metadata operation solely because optional rich fields are absent

### Requirement: TMDB TV series rich metadata is normalized from detail responses
The system SHALL normalize supported TMDB TV series rich metadata from series detail responses and appended field-oriented endpoints before applying catalog updates.

#### Scenario: Series detail includes rich fields
- **WHEN** a TMDB TV series detail response includes genres, keywords, vote average, content ratings, external IDs, original language, origin country, spoken languages, status, networks, creators, first air date, last air date, episode runtime, or production companies
- **THEN** the system MUST normalize the supported catalog fields, tags, ratings, certifications, status, dates, runtime, people, and identities without discarding the existing hierarchy output

#### Scenario: Series detail includes terminal status and last air date
- **WHEN** TMDB provides a series status and last air date for a matched TV series
- **THEN** the system MUST preserve those values where Mibo has catalog fields for series status and dates

### Requirement: TMDB genres and keywords are synchronized as catalog tags
The system SHALL persist TMDB genres and keywords as catalog tags with provider provenance so they can be displayed, searched, and used for related-item discovery.

#### Scenario: Provider returns genres and keywords
- **WHEN** TMDB metadata for a movie or TV series includes genre names and keyword names
- **THEN** the corresponding catalog item MUST link to `genre` and `keyword` tags with metadata source attribution

#### Scenario: Provider tags change on refetch
- **WHEN** a TMDB refetch returns a different set of provider genres or keywords for an item
- **THEN** the system MUST update the provider-owned genre and keyword links while preserving unrelated tag kinds such as scanner hashtags or user/local tags

#### Scenario: Detail response has duplicate tag names
- **WHEN** TMDB returns duplicate or whitespace-varied genre or keyword names
- **THEN** the system MUST store each normalized tag name at most once per item and tag kind

### Requirement: TMDB ratings and certifications populate catalog rating fields
The system SHALL apply TMDB community ratings and selected official certifications through catalog metadata field governance.

#### Scenario: Movie has vote average and regional certification
- **WHEN** a TMDB movie detail response includes `vote_average` and appended release-date certification data
- **THEN** the catalog movie MUST receive `community_rating` and `official_rating` values when those fields are not locked

#### Scenario: Series has vote average and content rating
- **WHEN** a TMDB TV series detail response includes `vote_average` and appended content-rating data
- **THEN** the catalog series MUST receive `community_rating` and `official_rating` values when those fields are not locked

#### Scenario: Rating field is locked
- **WHEN** a user has locked `community_rating` or `official_rating` and a TMDB operation returns new values
- **THEN** the operation MUST preserve the locked canonical value and record the provider value as skipped or source evidence according to field governance rules

### Requirement: TMDB external identifiers are preserved as identities
The system SHALL preserve supported TMDB-provided external identifiers as catalog identities or external IDs without conflating provider namespaces.

#### Scenario: Movie includes IMDb ID
- **WHEN** a TMDB movie detail or appended external IDs response includes an IMDb ID
- **THEN** the catalog item MUST retain that IMDb identity in addition to the primary TMDB identity

#### Scenario: TV series includes external IDs
- **WHEN** a TMDB TV series detail or appended external IDs response includes IMDb, TVDB, Wikidata, or other supported external IDs
- **THEN** the catalog series MUST retain those identities under their own provider namespace or identity type without replacing the TMDB identity

### Requirement: Unsupported TMDB feature endpoints remain out of rich metadata sync
The system SHALL NOT treat recommendations, similar titles, reviews, lists, or watch-provider availability as canonical rich metadata fields in this change.

#### Scenario: Provider response can append product feature data
- **WHEN** TMDB exposes recommendations, similar titles, reviews, lists, or watch providers for a movie or TV series
- **THEN** this metadata sync MUST ignore those endpoints unless a separate product capability defines how to store and present them
