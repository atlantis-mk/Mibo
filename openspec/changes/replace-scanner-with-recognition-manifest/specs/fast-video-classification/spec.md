## MODIFIED Requirements

### Requirement: Fast classifier generates candidates with evidence
The system SHALL generate one or more recognition manifest candidates for each supported video from structured filename signals and bounded directory summaries, and SHALL preserve candidate type, resource role, confidence, evidence, conflicts, and alternatives before resolver materialization.

#### Scenario: Filename has explicit episode marker
- **WHEN** a main video filename contains an explicit marker such as `S01E02`, `1x02`, `EP02`, or `第02集`
- **THEN** the classifier SHALL produce an episode candidate with season and episode evidence without requiring a user-selected TV library type

#### Scenario: Filename has movie-like evidence
- **WHEN** a main video filename or parent directory contains movie-like title and year evidence without episode markers
- **THEN** the classifier SHALL produce a movie work candidate with evidence and confidence

#### Scenario: Movie and episode candidates both exist
- **WHEN** path, filename, and sibling evidence support both movie and episode interpretations
- **THEN** the classifier SHALL preserve both alternatives in the recognition manifest and mark the decision provisional or review-required unless one candidate exceeds the configured confidence margin and no resolver conflict exists

#### Scenario: Filename release hints exist
- **WHEN** a video filename contains release hints such as quality, source, codec, audio, subtitle, edition, HDR, or release-group tokens
- **THEN** the classifier SHALL preserve those hints as variant or edition candidate evidence and SHALL NOT allow those tokens to become title words or weak episode-number evidence

### Requirement: Fast classifier uses bounded sibling grouping
The system SHALL use cached current-directory summary evidence derived from scan snapshots to produce recognition manifest evidence for episode sequences, movie version groups, independent movie files, and attachments without recursively scanning the full source for classification context.

#### Scenario: Siblings form an episode sequence
- **WHEN** likely main videos in the same directory have shared title evidence and consecutive episode numbers
- **THEN** the classifier SHALL add episode sequence evidence and compatible episode candidates to the manifest when confidence thresholds are met

#### Scenario: Siblings look like movie versions
- **WHEN** likely main videos in the same directory share a normalized title stem and differ mainly by quality, edition, cut, container, language, or release tokens
- **THEN** the classifier SHALL add one movie work candidate with multiple resource variant or edition candidates to the manifest

#### Scenario: Siblings look like independent movies
- **WHEN** likely main videos in the same directory have different title stems or year evidence and no episode sequence evidence
- **THEN** the classifier SHALL preserve independent movie candidates rather than merging them into one movie or one episode sequence

#### Scenario: Directory summary already exists
- **WHEN** multiple files in the same scanned directory require sibling context
- **THEN** the classifier SHALL reuse the cached directory summary for that scan snapshot rather than recomputing sibling evidence per file or issuing additional storage listings

### Requirement: Classification outcomes are thresholded
The system SHALL classify fast evidence into confirmed, provisional, or review-required candidate states based on configured confidence thresholds, candidate margins, and resolver conflict gates.

#### Scenario: Candidate is high confidence
- **WHEN** one candidate exceeds the fast-confirmed confidence threshold and no close conflicting candidate or resolver conflict exists
- **THEN** the system SHALL allow resolver materialization using that candidate while retaining its evidence

#### Scenario: Candidate is medium confidence
- **WHEN** a candidate is plausible but below the confirmed threshold
- **THEN** the system SHALL create a provisional manifest decision that can be validated asynchronously without blocking inventory persistence

#### Scenario: Candidate is low confidence or conflicting
- **WHEN** no candidate meets the minimum confidence threshold or movie and episode alternatives are too close
- **THEN** the system SHALL preserve inventory facts and produce a review-required resolver decision instead of silently committing a final semantic type

### Requirement: User corrections create source-scoped classification rules
The system SHALL allow confirmed user corrections to create source-scoped resolver rules that can be applied to future scans under the same source and matching path or evidence scope.

#### Scenario: User confirms sorted files are episodes
- **WHEN** a user confirms that a path group such as `/Anime/Show Season 2/*.mp4` represents sorted episodes for a named series and season
- **THEN** the system SHALL store a source-scoped resolver rule that can derive episode candidates for matching future files without requiring repeated review

#### Scenario: User confirms movie version grouping
- **WHEN** a user confirms that multiple sibling files represent versions of one movie
- **THEN** the system SHALL store a source-scoped resolver rule that can preserve the movie grouping on future scans and resyncs

#### Scenario: Rule scope is applied
- **WHEN** a learned resolver rule matches a future scan path or evidence pattern
- **THEN** the classifier SHALL include the rule as high-priority evidence and SHALL NOT apply it outside its configured source and scope
