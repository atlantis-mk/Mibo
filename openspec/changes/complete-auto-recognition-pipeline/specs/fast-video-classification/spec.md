## MODIFIED Requirements

### Requirement: Fast classifier uses bounded sibling grouping
The system SHALL use indexed file signals, cached current-directory summary evidence, content-shape plans, and bounded path-tree work-group evidence derived from already-listed snapshots to distinguish episode sequences, movie version groups, sibling-directory movie versions, independent movie files, series roots, and attachments without recursively scanning the full source for classification context.

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

#### Scenario: Sibling directories look like movie versions
- **WHEN** bounded parent path-tree evidence shows one-file child directories with the same title/year work key and release-hint differences
- **THEN** the classifier SHALL group the child files as one movie candidate with multiple assets without calling remote metadata providers

### Requirement: Fast path avoids heavy work
The fast classifier SHALL complete using cheap storage, path, filename, sidecar-name, already-listed object metadata, indexed file signals, structured filename signals, cached directory summary evidence, content-shape plans, and bounded path-tree work-group evidence only, and SHALL NOT perform media-content reads, technical probing, hashing, external provider lookup, or artwork retrieval.

#### Scenario: Source scan classifies a video file
- **WHEN** the scanner performs fast video classification during inventory traversal
- **THEN** it SHALL use path strings, filenames, extensions, sidecar filenames, already-listed object metadata, indexed file signals, cached directory snapshots, bounded current-directory summary context, and path-tree work-group context

#### Scenario: Expensive evidence is needed
- **WHEN** a classification decision requires duration, stream metadata, hashes, TMDB, MetaTube, TVDB, or artwork evidence to become reliable
- **THEN** the fast classifier SHALL leave the decision provisional or review-required and SHALL rely on asynchronous jobs for validation
