## MODIFIED Requirements

### Requirement: Low-confidence classifications are reviewable after scanning
The system SHALL surface low-confidence or conflicting classifier decisions for review after scanning rather than requiring users to make semantic choices before scanning, and SHALL base video classification on staged filename signal extraction, indexed file signal reuse, file-role detection, candidate generation, cached directory summary evidence, confidence thresholds, and reviewable evidence.

#### Scenario: Classifier cannot confidently choose movie or series
- **WHEN** video classification evidence is ambiguous or below the configured confidence threshold
- **THEN** the system SHALL preserve the inventory facts and create a reviewable decision with evidence, confidence, candidate alternatives, and proposed action

#### Scenario: User reviews an ambiguous decision
- **WHEN** a user opens the review surface for an ambiguous source item or group
- **THEN** the UI SHALL show the proposed classification, confidence, candidate alternatives, supporting evidence, affected files, and concrete correction actions so the user can correct or accept the decision

#### Scenario: Fast classification avoids heavy work
- **WHEN** automatic video classification runs during source-first scanning
- **THEN** the system SHALL use path, filename, extension, sidecar-name, already-listed object metadata, indexed filename signals, structured filename signals, and cached directory summary evidence without running ffprobe, content hashing, external metadata searches, artwork downloads, or additional recursive source analysis in the fast path

#### Scenario: Attachment evidence avoids false semantic choices
- **WHEN** a supported video looks like a trailer, sample, PV, preview, featurette, or other non-main attachment
- **THEN** the system SHALL classify it as an attachment candidate and SHALL NOT require the user to choose movie, show, mixed, or directory semantics before scanning continues

#### Scenario: Directory context is needed for numeric filenames
- **WHEN** numeric filenames cannot be confidently classified from filename signals alone
- **THEN** the system SHALL use cached directory summary evidence when available and SHALL mark the result provisional or review-required if the cheap context remains inconclusive

#### Scenario: Indexed signals are reused
- **WHEN** automatic classification revisits unchanged files from a previous scan
- **THEN** the system SHALL reuse current indexed file signals to classify source content without requiring user input or repeated filename parsing
