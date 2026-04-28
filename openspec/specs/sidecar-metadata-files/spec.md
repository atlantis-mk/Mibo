## Requirements

### Requirement: Discover same-folder sidecar files
The scanner SHALL discover supported sidecar files with `.srt`, `.ass`, `.nfo`, and `.json` extensions from the same storage folder as each scanned video file.

#### Scenario: Basename sidecar discovery
- **WHEN** a folder contains `Movie A.mkv`, `Movie A.srt`, `Movie A.ass`, and `Movie A.nfo`
- **THEN** the scanner SHALL associate those sidecar files with `Movie A.mkv`

#### Scenario: Ignore unsupported sidecars
- **WHEN** a folder contains `Movie A.mkv` and `Movie A.txt`
- **THEN** the scanner SHALL ignore `Movie A.txt` as sidecar evidence

### Requirement: Record subtitle sidecar evidence
The scanner SHALL record associated `.srt` and `.ass` sidecars as local scanner evidence for the catalog items linked to the video file and SHALL make those sidecars available for asset subtitle binding.

#### Scenario: Subtitle evidence recorded
- **WHEN** a scanned movie has matching `.srt` and `.ass` sidecars
- **THEN** the scanner metadata payload SHALL include subtitle sidecar entries with path, extension, and association source

#### Scenario: Subtitle contents not parsed for classification
- **WHEN** a scanned video has a matching subtitle sidecar
- **THEN** the scanner SHALL NOT parse subtitle dialogue text to infer title, season, or episode metadata

#### Scenario: Subtitle sidecar is available for binding
- **WHEN** the scanner records a matching `.srt` or `.ass` sidecar for a catalog asset
- **THEN** the scanner MUST pass enough sidecar path and association data to the catalog write path to bind that sidecar as an external subtitle track

### Requirement: Use metadata sidecar hints safely
The scanner SHALL parse supported `.nfo` and `.json` sidecars for high-confidence local metadata hints and apply them only through existing catalog scan governance protections.

#### Scenario: JSON metadata improves movie classification
- **WHEN** `Movie A.mkv` has a matching JSON sidecar with title and year fields
- **THEN** the scanner SHALL use those fields as local hints when creating or updating the movie catalog item

#### Scenario: NFO metadata improves episode classification
- **WHEN** an episode video has a matching NFO sidecar with series title, season number, and episode number
- **THEN** the scanner SHALL use those fields as local hints when creating or updating the episode hierarchy

#### Scenario: Curated metadata is preserved
- **WHEN** a catalog item is locked, manual, matched, or needs review
- **THEN** sidecar metadata hints SHALL NOT overwrite preserved descriptive fields for that item

### Requirement: Sidecar failures are non-fatal
The scanner SHALL continue scanning when a sidecar file is unreadable, malformed, unsupported in structure, or exceeds the allowed read limit.

#### Scenario: Malformed metadata sidecar
- **WHEN** a matching JSON or NFO sidecar cannot be parsed
- **THEN** the scanner SHALL continue processing the video and record no unsafe metadata override from that sidecar

#### Scenario: Oversized sidecar
- **WHEN** a matching sidecar exceeds the configured scanner read limit
- **THEN** the scanner SHALL skip reading that sidecar and continue scanning the video

### Requirement: Avoid ambiguous folder-level metadata matches
The scanner SHALL avoid applying folder-level sidecars to a video when the folder contains multiple plausible videos and the sidecar cannot be deterministically associated.

#### Scenario: Ambiguous folder metadata
- **WHEN** a folder contains `Movie A.mkv`, `Movie B.mkv`, and `metadata.json`
- **THEN** the scanner SHALL NOT apply `metadata.json` to either video unless deterministic association rules identify a single target

#### Scenario: Unambiguous folder metadata
- **WHEN** a folder contains one video file and `metadata.json`
- **THEN** the scanner MAY associate `metadata.json` with that video as folder-level metadata evidence
