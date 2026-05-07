## ADDED Requirements

### Requirement: Filename token profiles support directory shape profiling
The system SHALL extract cheap filename token profiles that can be aggregated into directory shape profiles without requiring full title normalization or final semantic classification for every file.

#### Scenario: Dense numeric release filename is tokenized
- **WHEN** the scanner tokenizes `01.2160p.HD国语中字[网站].mkv` for directory shape profiling
- **THEN** the token profile SHALL expose leading numeric episode evidence, release quality evidence, subtitle/language or noise hints when recognized, and website noise evidence without treating quality or website tokens as title words

#### Scenario: Chinese episode filename is tokenized
- **WHEN** the scanner tokenizes `第001集.mkv`
- **THEN** the token profile SHALL expose explicit Chinese episode marker evidence and episode number 1

#### Scenario: SxE filename is tokenized
- **WHEN** the scanner tokenizes `Show.Name.S01E001.1080p.mkv`
- **THEN** the token profile SHALL expose series-title candidate evidence, season number 1, episode number 1, and release quality evidence

### Requirement: Token profiles suppress weak false positives
The system SHALL mark release, audio, codec, source, subtitle, website, and attachment tokens as suppression evidence so they do not become weak episode numbers or independent movie title words during directory shape profiling.

#### Scenario: Codec token resembles episode number
- **WHEN** a filename contains `x265` or `H.265`
- **THEN** the token profile SHALL treat the token as codec evidence and SHALL NOT use `265` as episode evidence

#### Scenario: Audio channel resembles decimal episode number
- **WHEN** a filename contains `5.1` or `7.1`
- **THEN** the token profile SHALL treat the token as audio evidence and SHALL suppress weak numeric episode inference from that token

### Requirement: Token profiles are reusable within a scan
The system SHALL cache filename token profiles by storage path within a scan or materialization run so directory profiling, plan compilation, and fallback classification can reuse the same parsed signals.

#### Scenario: File participates in profile and materialization
- **WHEN** a file's token profile is used to build a directory profile and later to materialize a file assignment in the same scan run
- **THEN** the system SHALL reuse the parsed token profile rather than reparsing the filename from scratch
