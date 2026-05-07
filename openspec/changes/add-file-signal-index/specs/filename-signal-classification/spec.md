## MODIFIED Requirements

### Requirement: Filename signals are extracted before title cleanup
The system SHALL extract structured filename and path signals before producing cleaned title views or final media classification decisions, and SHALL persist reusable signal results for supported video inventory files when current file fingerprints and classifier versions are known.

#### Scenario: Dense release filename is scanned
- **WHEN** the scanner processes `Dune.Part.Two.2024.2160p.UHD.BluRay.DV.TrueHD.Atmos.7.1.x265-GROUP.mkv`
- **THEN** the filename signal output SHALL preserve title, year, quality, source, HDR, audio, codec, and release-group hints before creating the cleaned title candidate `Dune Part Two`
- **AND** the reusable file signal index SHALL be able to store those parsed hints for later directory profiling

#### Scenario: TV release filename is scanned
- **WHEN** the scanner processes `Show.Name.S01E02.1080p.WEB-DL.DDP5.1.x264-GROUP.mkv`
- **THEN** the filename signal output SHALL preserve the series title candidate, season number, episode number, quality, source, audio, codec, and release-group hints before classification
- **AND** unchanged rescans SHALL be able to reuse those indexed signals instead of reparsing the filename

### Requirement: Directory summaries are computed from scan snapshots
The system SHALL compute per-directory summaries from already-listed scan snapshot entries and SHALL use those summaries as bounded context for classification without additional recursive storage probing. When current indexed file signals are available for visible videos, the system SHALL aggregate directory summaries from those signals instead of reparsing filenames.

#### Scenario: Flat numeric episode directory
- **WHEN** a scanned directory contains `01.mkv`, `02.mkv`, and `03.mkv` under a named parent directory
- **THEN** the directory summary SHALL expose consecutive numeric sequence evidence that can support episode candidates for that parent title

#### Scenario: Directory contains independent movies
- **WHEN** a scanned directory contains `Alien.1979.mkv` and `Aliens.1986.mkv`
- **THEN** the directory summary SHALL expose distinct title-year movie evidence and SHALL NOT force those siblings into one movie-version group

#### Scenario: Directory contains versions and attachments
- **WHEN** a scanned directory contains `Movie.1080p.mkv`, `Movie.2160p.mkv`, `Movie.Trailer.mkv`, and `sample.mkv`
- **THEN** the directory summary SHALL expose version-like main files and attachment role counts using the already-listed entries

#### Scenario: Indexed signals are available
- **WHEN** the scanned directory's visible videos already have current indexed file signals
- **THEN** the directory summary SHALL use those indexed signals for sequence, title, year, role, and release-hint aggregation without additional filename parsing
