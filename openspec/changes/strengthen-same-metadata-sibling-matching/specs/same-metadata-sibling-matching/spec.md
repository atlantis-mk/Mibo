## ADDED Requirements

### Requirement: Scanner matches sibling resources at canonical-work scope before creating duplicate metadata identities
The system SHALL run a same-metadata sibling-matching phase after work-group classification and before creating duplicate movie or episode metadata identities, and SHALL decide whether resources belong to the same canonical work, the same episode, a supplemental group, or an unresolved conflict.

#### Scenario: Same movie siblings reuse one metadata identity
- **WHEN** two accepted movie resources share one canonical movie identity according to strong local or provider evidence
- **THEN** the scanner MUST link both resources to the same movie metadata identity instead of creating duplicate movie metadata items

#### Scenario: Same episode siblings reuse one episode metadata identity
- **WHEN** two accepted series resources identify the same series, season, and episode tuple
- **THEN** the scanner MUST link both resources to the same episode metadata identity instead of creating duplicate episode metadata items

### Requirement: Sibling matching separates canonical identity from version identity
The system SHALL determine canonical work identity before it uses quality, edition, codec, or source-release traits to infer version relationships.

#### Scenario: Alternate encode becomes a version link
- **WHEN** two resources already match the same canonical movie identity and differ only by release traits such as resolution, source, codec, or edition markers
- **THEN** the scanner MUST link them as distinct versions of the same metadata identity

#### Scenario: Version traits alone do not create identity
- **WHEN** a resource only shares release traits with an existing metadata identity and lacks canonical title, provider, episode, or fingerprint support
- **THEN** the scanner MUST keep the resource unresolved instead of linking it automatically

### Requirement: File md5 strengthens cross-source sibling matching without blocking first visibility
The system SHALL treat file `md5` as strong file-level sibling evidence when it is available and SHALL NOT require synchronous hash completion before an organizing entry can appear.

#### Scenario: Same md5 across two sources reuses one metadata identity
- **WHEN** two playable files from different sources have the same `md5` and no stronger conflicting metadata identity evidence
- **THEN** the scanner MUST treat them as the same binary media for sibling matching and MUST prefer reusing the already-linked metadata identity

#### Scenario: Missing md5 does not block browse visibility
- **WHEN** a newly scanned file has no `md5` yet
- **THEN** the scanner MUST continue classification and browse visibility using non-hash evidence instead of delaying the organizing entry

### Requirement: Weak or conflicting sibling matches remain reviewable
The system SHALL keep same-title or otherwise ambiguous sibling matches unresolved when strong identity anchors are absent or conflicting.

#### Scenario: Weak same-title candidate stays unresolved
- **WHEN** a resource only matches an existing metadata identity by normalized title or weak title-plus-year evidence
- **THEN** the scanner MUST record a reviewable candidate instead of automatically linking it as the same metadata identity

#### Scenario: Conflicting identity evidence blocks auto-linking
- **WHEN** a file `md5`, sidecar identity, or provider identity conflicts with another strong identity signal for the candidate metadata item
- **THEN** the scanner MUST mark the sibling match review-required instead of silently reusing the metadata identity

### Requirement: Supplemental media is isolated from automatic primary/version matching
The system SHALL exclude samples, trailers, featurettes, previews, and other supplemental media from automatic primary or version sibling matching.

#### Scenario: Supplemental file does not merge into main movie
- **WHEN** a folder contains a main movie file and a sibling file classified as a sample or trailer
- **THEN** the scanner MUST keep the supplemental file out of the main movie primary/version link set
