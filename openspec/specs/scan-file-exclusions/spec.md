# scan-file-exclusions Specification

## Purpose
TBD - created by archiving change filter-ad-files-during-scan. Update Purpose after archive.
## Requirements
### Requirement: Skip excluded video files
The scanner SHALL skip supported video files that are identified by persisted scan exclusions or conservative, token-bound automatic rules before creating catalog, asset, inventory, metadata match, or probe work for those files.

#### Scenario: Advertisement filename is skipped
- **WHEN** a library scan encounters `/movies/Movie A/advertisement.mp4` or `/movies/Movie A/Movie A - ad.mkv`
- **THEN** the scanner SHALL skip that file and SHALL NOT create catalog items, media assets, inventory files, metadata match jobs, or probe jobs for it

#### Scenario: Advertisement folder video is skipped
- **WHEN** a library scan encounters a video inside a folder explicitly named `ads`, `advertisements`, `commercials`, or `广告`
- **THEN** the scanner SHALL skip that video file as excluded advertisement content

#### Scenario: User-marked exclusion is skipped on rescan
- **WHEN** a user previously marked a scanned file as excluded from scans
- **THEN** later scans SHALL skip the same file using its stable identity when available or its scoped path fallback when stable identity is unavailable

### Requirement: Mark scanned files as scan exclusions
The system SHALL provide an operation for marking an already-scanned file, asset, or catalog-linked media entry as excluded from future scans, with `advertisement` as a supported reason.

#### Scenario: User marks scanned file as advertisement
- **WHEN** a user marks a scanned media asset as an advertisement
- **THEN** the system SHALL persist a scan exclusion for that source file with reason `advertisement` and SHALL remove or hide the associated scanner-managed asset from normal catalog browsing

#### Scenario: Marking does not delete source storage
- **WHEN** a user marks a scanned file as excluded from scans
- **THEN** the system SHALL NOT physically delete the source file from OpenList, local disk, or any other storage provider

#### Scenario: Marking avoids future work
- **WHEN** a user-marked excluded file is encountered by a later scan
- **THEN** the scanner SHALL NOT queue metadata match jobs or inventory probe jobs for that file

### Requirement: Preserve normal media scanning
The scanner SHALL NOT skip supported video files unless they match a persisted scan exclusion or explicit automatic exclusion indicators, and it MUST avoid substring-only matches that would misclassify legitimate titles.

#### Scenario: Title containing ad-like substring remains scannable
- **WHEN** a library scan encounters files such as `/movies/Ad Astra/Ad Astra.mkv`, `/movies/Adventure Movie/Adventure Movie.mp4`, or `/shows/Show/Season 01/Show.S01E01.mkv`
- **THEN** the scanner SHALL process those files using the normal media classification and catalog write path

#### Scenario: Intentional extras remain scannable
- **WHEN** a movie folder contains `trailer.mkv`, `sample.mp4`, or `featurette.mkv` without explicit advertisement markers and without a persisted exclusion
- **THEN** the scanner SHALL keep applying existing movie extra classification instead of treating those files as excluded content

### Requirement: Continue folder traversal after skipped exclusions
The scanner SHALL continue scanning sibling files and child directories when one or more excluded files are skipped.

#### Scenario: Mixed folder scan continues
- **WHEN** a folder contains `advertisement.mp4`, `Movie A.mkv`, `Movie A.srt`, and a child folder containing another valid video
- **THEN** the scanner SHALL skip only the excluded file and SHALL still scan the valid video files and associated sidecars

### Requirement: Report skipped excluded files
The scanner SHALL provide scan-level visibility that files were skipped without exposing provider credentials or raw signed URLs.

#### Scenario: Skipped count is visible
- **WHEN** a scan skips two excluded files
- **THEN** the scan result or scanner logs SHALL report that two files were skipped

#### Scenario: Skip reason distinguishes user exclusions
- **WHEN** a scan skips a file because it was previously marked as excluded
- **THEN** the scan result or scanner logs SHALL distinguish the skip reason from automatic filename-based filtering

### Requirement: Preserve reversible exclusion records
The system SHALL persist scan exclusions in a reversible form so accidental marks can be disabled or restored by future operations without requiring provider file recovery.

#### Scenario: Exclusion record can be disabled
- **WHEN** an exclusion record is disabled or otherwise marked inactive
- **THEN** later scans SHALL stop applying that exclusion while preserving its audit history

