# fast-video-classification Specification

## Purpose
Define fast, evidence-preserving video classification that separates cheap scan-time decisions from heavier enrichment and user review.

## Requirements
### Requirement: Fast classifier separates role from semantic type
The system SHALL classify supported video files into file roles before deciding whether main playable content is a movie or a series episode.

#### Scenario: Trailer is present beside a movie file
- **WHEN** a directory contains `Movie A.mkv` and `trailer.mkv`
- **THEN** the classifier MUST mark `trailer.mkv` as an attachment role and MUST NOT count it as a main file when deciding movie versus episode semantics

#### Scenario: Sample file is encountered
- **WHEN** a supported video filename or path segment indicates a sample, teaser, preview, PV, trailer, featurette, behind-the-scenes item, or similar attachment
- **THEN** the classifier SHALL create an attachment candidate with evidence instead of projecting it as an independent movie or episode

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

### Requirement: Classification outcomes are thresholded
The system SHALL classify fast decisions into confirmed, provisional, or review-required states based on configured confidence thresholds and candidate margins.

#### Scenario: Candidate is high confidence
- **WHEN** one candidate exceeds the fast-confirmed confidence threshold and no close conflicting candidate exists
- **THEN** the system SHALL allow catalog projection using that candidate while retaining its evidence

#### Scenario: Candidate is medium confidence
- **WHEN** a candidate is plausible but below the confirmed threshold
- **THEN** the system SHALL create a provisional decision that can be validated asynchronously without blocking inventory persistence

#### Scenario: Candidate is low confidence or conflicting
- **WHEN** no candidate meets the minimum confidence threshold or movie and episode alternatives are too close
- **THEN** the system SHALL preserve inventory facts and produce a review-required decision instead of silently committing a final semantic type

### Requirement: User corrections create source-scoped classification rules
The system SHALL allow confirmed user corrections to create source-scoped classification rules that can be applied to future scans under the same source and matching path scope.

#### Scenario: User confirms sorted files are episodes
- **WHEN** a user confirms that a path group such as `/Anime/Show Season 2/*.mp4` represents sorted episodes for a named series and season
- **THEN** the system SHALL store a source-scoped rule that can derive episode candidates for matching future files without requiring repeated review

#### Scenario: User confirms movie version grouping
- **WHEN** a user confirms that multiple sibling files represent versions of one movie
- **THEN** the system SHALL store a source-scoped rule that can preserve the movie grouping on future scans and resyncs

#### Scenario: Rule scope is applied
- **WHEN** a learned classification rule matches a future scan path
- **THEN** the classifier SHALL include the rule as evidence and SHALL NOT apply it outside its configured source and path scope
