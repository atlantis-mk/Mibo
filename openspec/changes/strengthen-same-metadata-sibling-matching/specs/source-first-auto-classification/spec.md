## MODIFIED Requirements

### Requirement: Low-confidence classifications are reviewable after scanning
The system SHALL surface low-confidence or conflicting classifier decisions for review after scanning rather than requiring users to make semantic choices before scanning, SHALL base video classification on staged filename signal extraction, file-role detection, candidate generation, cached directory summary evidence, path-tree and work-group evidence, sidecar and supported external identity evidence, same-metadata sibling evidence, confidence thresholds, and reviewable evidence, and SHALL avoid forcing final metadata creation or same-metadata linking when only weak fallback evidence is available.

#### Scenario: Classifier cannot confidently choose movie or series
- **WHEN** video classification evidence is ambiguous or below the configured confidence threshold
- **THEN** the system SHALL preserve the inventory facts and create a reviewable decision with evidence, confidence, candidate alternatives, and proposed action

#### Scenario: User reviews an ambiguous decision
- **WHEN** a user opens the review surface for an ambiguous source item or group
- **THEN** the UI SHALL show the proposed classification, confidence, candidate alternatives, supporting evidence, affected files, and concrete correction actions so the user can correct or accept the decision

#### Scenario: Fast classification avoids heavy work
- **WHEN** automatic video classification runs during source-first scanning
- **THEN** the system SHALL use path, filename, extension, sidecar-name, already-listed object metadata, structured filename signals, cached directory summary evidence, path-tree assignments, group-level evidence, and supported sidecar or provider identity hints without running ffprobe, synchronous content hashing, external metadata searches, artwork downloads, or additional recursive source analysis in the fast path

#### Scenario: Attachment evidence avoids false semantic choices
- **WHEN** a supported video looks like a trailer, sample, PV, preview, featurette, or other non-main attachment
- **THEN** the system SHALL classify it as an attachment candidate and SHALL NOT require the user to choose movie, show, mixed, or directory semantics before scanning continues

#### Scenario: Directory context is needed for numeric filenames
- **WHEN** numeric filenames cannot be confidently classified from filename signals alone
- **THEN** the system SHALL use cached directory summary evidence when available and SHALL mark the result provisional or review-required if the cheap context remains inconclusive

#### Scenario: Weak same-metadata candidate is held back
- **WHEN** sibling matching only has weak title or title-plus-year evidence for an existing metadata identity without stronger sidecar, provider, episode tuple, or `md5` support
- **THEN** the system MUST keep the resource unresolved and reviewable instead of automatically linking it to that metadata identity

#### Scenario: Md5 can upgrade matching asynchronously
- **WHEN** a file `md5` becomes available after the fast-path scan completed
- **THEN** the system MUST be able to re-run same-metadata sibling matching without requiring a full rescan of the source tree
