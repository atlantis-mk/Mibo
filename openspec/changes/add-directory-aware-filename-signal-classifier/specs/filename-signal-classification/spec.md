## ADDED Requirements

### Requirement: Filename signals are extracted before title cleanup
The system SHALL extract structured filename and path signals before producing cleaned title views or final media classification decisions.

#### Scenario: Dense release filename is scanned
- **WHEN** the scanner processes `Dune.Part.Two.2024.2160p.UHD.BluRay.DV.TrueHD.Atmos.7.1.x265-GROUP.mkv`
- **THEN** the filename signal output SHALL preserve title, year, quality, source, HDR, audio, codec, and release-group hints before creating the cleaned title candidate `Dune Part Two`

#### Scenario: TV release filename is scanned
- **WHEN** the scanner processes `Show.Name.S01E02.1080p.WEB-DL.DDP5.1.x264-GROUP.mkv`
- **THEN** the filename signal output SHALL preserve the series title candidate, season number, episode number, quality, source, audio, codec, and release-group hints before classification

### Requirement: Filename release hints are not authoritative technical metadata
The system SHALL treat filename-derived quality, source, codec, audio, subtitle, HDR, edition, and release-group values as release hints and evidence, not as authoritative technical metadata.

#### Scenario: Filename and probe disagree
- **WHEN** a filename contains `1080p` but later media probing reports a 2160p video stream
- **THEN** the system SHALL keep `1080p` as filename-derived evidence and SHALL use the probed stream data for authoritative technical metadata

#### Scenario: Filename contains audio channel token
- **WHEN** a filename contains `DDP5.1`
- **THEN** the system SHALL preserve it as an audio release hint and SHALL NOT write an authoritative audio layout field from that filename token alone

### Requirement: Release tokens suppress weak title and episode inference
The system SHALL use recognized release hints as anti-misclassification evidence so quality, source, codec, audio, and subtitle tokens are not treated as title words or weak episode numbers.

#### Scenario: Audio channel resembles episode number
- **WHEN** the scanner processes `Movie.Name.5.1.1080p.WEB-DL.mkv`
- **THEN** `5.1` SHALL be treated as audio evidence and SHALL suppress weak numeric episode inference from those numbers

#### Scenario: Quality token contains digits
- **WHEN** the scanner processes `Movie.Name.2160p.x265.mkv`
- **THEN** `2160p` and `x265` SHALL be excluded from title tokens and SHALL NOT create episode-number evidence

### Requirement: Directory summaries are computed from scan snapshots
The system SHALL compute per-directory summaries from already-listed scan snapshot entries and SHALL use those summaries as bounded context for classification without additional recursive storage probing.

#### Scenario: Flat numeric episode directory
- **WHEN** a scanned directory contains `01.mkv`, `02.mkv`, and `03.mkv` under a named parent directory
- **THEN** the directory summary SHALL expose consecutive numeric sequence evidence that can support episode candidates for that parent title

#### Scenario: Directory contains independent movies
- **WHEN** a scanned directory contains `Alien.1979.mkv` and `Aliens.1986.mkv`
- **THEN** the directory summary SHALL expose distinct title-year movie evidence and SHALL NOT force those siblings into one movie-version group

#### Scenario: Directory contains versions and attachments
- **WHEN** a scanned directory contains `Movie.1080p.mkv`, `Movie.2160p.mkv`, `Movie.Trailer.mkv`, and `sample.mkv`
- **THEN** the directory summary SHALL expose version-like main files and attachment role counts using the already-listed entries

### Requirement: Candidates include lightweight filename evidence
The system SHALL attach lightweight evidence summaries to movie, episode, trailer, sample, extra, and version candidates produced from filename signals and directory summaries.

#### Scenario: Movie candidate uses title and year
- **WHEN** a filename provides a cleaned title candidate and release year without episode evidence
- **THEN** the movie candidate SHALL include title and year evidence and any release hints used for version grouping

#### Scenario: Episode candidate uses marker and directory evidence
- **WHEN** a filename contains `S02E03` and the parent directory indicates season 2
- **THEN** the episode candidate SHALL include filename marker evidence and directory summary evidence

#### Scenario: Attachment candidate uses role token
- **WHEN** a filename or path segment indicates trailer, sample, preview, PV, featurette, or extra content
- **THEN** the attachment candidate SHALL include role evidence and SHALL NOT be counted as a main movie or episode file

### Requirement: Fast classification escalates only ambiguous outcomes
The system SHALL confirm high-confidence outcomes from cheap filename and directory evidence, keep medium-confidence outcomes provisional, and mark low-confidence or conflicting outcomes for review or later refinement without performing heavy work in the fast path.

#### Scenario: Strong evidence is available
- **WHEN** a candidate has strong filename evidence and no close conflicting alternative after directory summary context is applied
- **THEN** the fast classifier SHALL allow confirmed catalog projection with retained evidence

#### Scenario: Cheap evidence is insufficient
- **WHEN** filename signals and directory summary evidence cannot distinguish movie, episode, version, independent movie, or attachment semantics
- **THEN** the fast classifier SHALL preserve alternatives and produce a provisional or review-required decision instead of reading media contents or calling external providers
