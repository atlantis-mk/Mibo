## MODIFIED Requirements

### Requirement: Fast classifier generates candidates with evidence
The system SHALL generate one or more classification candidates for each supported video from structured filename signals and bounded directory summaries, and SHALL preserve candidate type, role, confidence, evidence, and alternatives before final catalog projection.

#### Scenario: Filename has explicit episode marker
- **WHEN** a main video filename contains an explicit marker such as `S01E02`, `1x02`, `EP02`, or `第02集`
- **THEN** the classifier SHALL produce an episode candidate with season and episode evidence without requiring a user-selected TV library type

#### Scenario: Filename has movie-like evidence
- **WHEN** a main video filename or parent directory contains movie-like title and year evidence without episode markers
- **THEN** the classifier SHALL produce a movie candidate with evidence and confidence

#### Scenario: Movie and episode candidates both exist
- **WHEN** path, filename, and sibling evidence support both movie and episode interpretations
- **THEN** the classifier SHALL preserve both alternatives and mark the decision provisional or review-required unless one candidate exceeds the configured confidence margin

#### Scenario: Filename release hints exist
- **WHEN** a video filename contains release hints such as quality, source, codec, audio, subtitle, edition, HDR, or release-group tokens
- **THEN** the classifier SHALL preserve those hints as candidate evidence and SHALL NOT allow those tokens to become title words or weak episode-number evidence

### Requirement: Fast classifier uses bounded sibling grouping
The system SHALL use cached current-directory summary evidence derived from scan snapshots to distinguish episode sequences, movie version groups, independent movie files, and attachments without recursively scanning the full source for classification context.

#### Scenario: Siblings form an episode sequence
- **WHEN** likely main videos in the same directory have shared title evidence and consecutive episode numbers
- **THEN** the classifier SHALL group them as episode candidates for the same series and season when confidence thresholds are met

#### Scenario: Siblings look like movie versions
- **WHEN** likely main videos in the same directory share a normalized title stem and differ mainly by quality, edition, cut, container, language, or release tokens
- **THEN** the classifier SHALL group them as one movie candidate with multiple asset/version candidates

#### Scenario: Siblings look like independent movies
- **WHEN** likely main videos in the same directory have different title stems or year evidence and no episode sequence evidence
- **THEN** the classifier SHALL preserve independent movie candidates rather than merging them into one movie or one episode sequence

#### Scenario: Directory summary already exists
- **WHEN** multiple files in the same scanned directory require sibling context
- **THEN** the classifier SHALL reuse the cached directory summary for that scan snapshot rather than recomputing sibling evidence per file or issuing additional storage listings

### Requirement: Fast path avoids heavy work
The fast classifier SHALL complete using cheap storage, path, filename, sidecar-name, already-listed object metadata, structured filename signals, and cached directory summary evidence only, and SHALL NOT perform media-content reads, technical probing, hashing, external provider lookup, or artwork retrieval.

#### Scenario: Source scan classifies a video file
- **WHEN** the scanner performs fast video classification during inventory traversal
- **THEN** it SHALL use path strings, filenames, extensions, sidecar filenames, already-listed object metadata, structured filename signals, cached directory snapshots, and bounded current-directory summary context

#### Scenario: Expensive evidence is needed
- **WHEN** a classification decision requires duration, stream metadata, hashes, TMDB, MetaTube, TVDB, or artwork evidence to become reliable
- **THEN** the fast classifier SHALL leave the decision provisional or review-required and SHALL rely on asynchronous jobs for validation
