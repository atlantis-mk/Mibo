## MODIFIED Requirements

### Requirement: Low-confidence classifications are reviewable after scanning
The system SHALL surface low-confidence or conflicting classifier and resolver decisions for review after scanning rather than requiring users to make semantic choices before scanning, and SHALL base video recognition on staged filename signal extraction, file-role detection, recognition manifest candidates, cached directory evidence, confidence thresholds, conflict gates, and reviewable resolver evidence.

#### Scenario: Classifier cannot confidently choose movie or series
- **WHEN** video classification evidence is ambiguous or below the configured confidence threshold
- **THEN** the system SHALL preserve the inventory facts and create a reviewable resolver decision with evidence, confidence, candidate alternatives, conflict state, and proposed action

#### Scenario: User reviews an ambiguous decision
- **WHEN** a user opens the review surface for an ambiguous source item or group
- **THEN** the UI SHALL show the proposed classification, resolver decision, confidence, candidate alternatives, supporting evidence, affected files, and concrete correction actions so the user can correct or accept the decision

#### Scenario: Fast classification avoids heavy work
- **WHEN** automatic video recognition runs during source-first scanning
- **THEN** the system SHALL use path, filename, extension, sidecar-name, already-listed object metadata, structured filename signals, cached directory evidence, and resolver rules without running ffprobe, content hashing, external metadata searches, artwork downloads, or additional recursive source analysis in the fast path

#### Scenario: Attachment evidence avoids false semantic choices
- **WHEN** a supported video looks like a trailer, sample, PV, preview, featurette, or other non-main attachment
- **THEN** the system SHALL classify it as a supplemental candidate and SHALL NOT require the user to choose movie, show, mixed, or directory semantics before scanning continues

#### Scenario: Directory context is needed for numeric filenames
- **WHEN** numeric filenames cannot be confidently classified from filename signals alone
- **THEN** the system SHALL use cached directory evidence when available and SHALL mark the manifest candidate provisional or review-required if the cheap context remains inconclusive

## ADDED Requirements

### Requirement: Source-first scanning defers metadata identity to resolver
The system SHALL keep source acceptance and inventory scanning independent from final metadata identity materialization by deferring scanner-created metadata links to the recognition resolver.

#### Scenario: Source scan finds a confident movie file
- **WHEN** source-first scanning finds a video file with confident local movie evidence
- **THEN** the scan MUST record inventory facts and recognition candidates and MUST NOT bypass the resolver to create a scanner-owned metadata link directly

#### Scenario: Source scan completes before resolver enrichment
- **WHEN** source-first scanning completes inventory traversal for a source
- **THEN** source status and inventory visibility MUST be available even if recognition resolution, projection rebuilds, metadata matching, probing, or artwork enrichment continue asynchronously
