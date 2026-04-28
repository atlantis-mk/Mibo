## MODIFIED Requirements

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
