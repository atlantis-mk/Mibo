## ADDED Requirements

### Requirement: Scanner produces recognition manifests before metadata materialization
The system SHALL build a recognition manifest from inventory facts, file signals, sidecars, path context, hash evidence, and learned rules before creating or linking metadata items for scanned video media.

#### Scenario: Scan discovers supported videos
- **WHEN** a scan discovers supported video files under a source path
- **THEN** the system MUST persist inventory facts and build recognition manifest candidates before creating `MetadataItem` or `ResourceMetadataLink` records for those files

#### Scenario: Manifest contains complete candidate evidence
- **WHEN** a recognition manifest is built
- **THEN** each candidate MUST include stable candidate keys, affected inventory files, candidate type, source scope, evidence, alternatives, confidence, and conflict state when applicable

### Requirement: Resolver owns final identity decisions
The system SHALL route all scanner-created metadata identity decisions through one deterministic identity resolver and SHALL NOT let filename, content-shape, path-tree, sidecar, or sibling match helpers directly create or link final metadata identities.

#### Scenario: Evidence provider finds a movie version group
- **WHEN** a directory or path-tree evidence provider detects files that look like movie versions
- **THEN** it MUST add evidence to the manifest and the resolver MUST decide whether to materialize one metadata identity with multiple resources

#### Scenario: Old direct-link path would create metadata
- **WHEN** scan logic has only local classification evidence for a file
- **THEN** it MUST create or update a resolver candidate rather than directly calling metadata creation or resource-to-metadata linking helpers

### Requirement: Resolver separates work identity from resource variants
The resolver SHALL distinguish canonical work identity, episode identity, resource variants, edition or cut identity, duplicate binary evidence, and supplemental resource roles.

#### Scenario: Same movie has two encodes
- **WHEN** two files share accepted movie work identity and differ by quality, source, codec, audio, subtitle, HDR, container, or release-group traits
- **THEN** the resolver MUST materialize one movie metadata identity with separate playable resource links carrying variant evidence

#### Scenario: Same movie has different cuts
- **WHEN** files share accepted movie work identity but include edition or cut evidence such as theatrical, extended, director cut, unrated, remaster, or special edition
- **THEN** the resolver MUST preserve edition evidence separately from encode variant evidence so governance and playback selection can distinguish the cut

#### Scenario: Supplemental video is detected
- **WHEN** a file is recognized as trailer, sample, extra, featurette, preview, behind-the-scenes, or another supplemental role
- **THEN** the resolver MUST NOT treat it as a primary or encode version resource and MUST materialize it only as a supplemental relationship when a parent identity is accepted

### Requirement: Resolver applies explicit acceptance and conflict gates
The resolver SHALL accept automatic identity links only when a supported identity gate is satisfied and no blocking conflict is present.

#### Scenario: External identities agree
- **WHEN** candidates for the same metadata type share the same supported provider identity from sidecar or provider evidence
- **THEN** the resolver MUST accept them as the same metadata identity unless a higher-priority manual split rule or blocking conflict applies

#### Scenario: Movie title year and variant evidence agree
- **WHEN** movie candidates share normalized title and year, have compatible work context, and differ primarily by variant or edition traits
- **THEN** the resolver MAY accept one metadata identity with multiple resources when no external identity, year, or type conflict exists

#### Scenario: Identity evidence conflicts
- **WHEN** candidate evidence contains conflicting provider identities, incompatible years, incompatible media types, or incompatible episode tuples
- **THEN** the resolver MUST block automatic merge and persist a review-required decision with alternatives and conflict reasons

### Requirement: Resolver materialization is idempotent
The system SHALL materialize accepted resolver decisions into metadata items, resources, resource files, resource metadata links, library links, and projection refresh inputs using stable resolver keys so reruns do not create duplicates.

#### Scenario: Resolver reruns after classifier version changes
- **WHEN** a manifest is rebuilt for files that were already materialized
- **THEN** the materializer MUST update or reuse existing resolver-owned metadata/resource graph records instead of creating duplicate metadata identities for the same accepted candidate

#### Scenario: Async hash or probe evidence arrives
- **WHEN** asynchronous enrichment adds hash or probe evidence for an inventory file
- **THEN** the system MUST rerun resolution only for affected candidates and refresh projections for affected metadata/resource IDs

### Requirement: Manual corrections become resolver rules
The system SHALL persist user-approved recognition corrections as resolver rules scoped to source, path, file, candidate, or metadata identity context and SHALL apply those rules before automatic heuristic evidence.

#### Scenario: User confirms movie versions
- **WHEN** a user confirms that multiple files represent versions or editions of one movie work
- **THEN** the system MUST persist a resolver rule and future scans in scope MUST use it as high-priority evidence for the same grouping

#### Scenario: User splits a bad merge candidate
- **WHEN** a user rejects a proposed same-work grouping
- **THEN** the system MUST persist a split or independent-work resolver rule that prevents future automatic merging for the scoped evidence pattern

### Requirement: Replaced recognition engines are removed or demoted to evidence providers
The implementation SHALL remove old final-decision scanner code paths or rewrite them as evidence providers so the codebase has one scanner identity architecture.

#### Scenario: Legacy helper performs final metadata linking
- **WHEN** a helper from content-shape, path-tree, same-metadata sibling matching, or catalog scan linking directly creates metadata identities or resource metadata links from scan classification
- **THEN** that helper MUST be deleted, rewritten to emit manifest evidence, or moved behind the resolver materializer

#### Scenario: Obsolete tests validate old behavior
- **WHEN** tests assert scan-order-dependent metadata creation or legacy matcher-specific final decisions
- **THEN** those tests MUST be removed or replaced with resolver manifest, conflict, and materialization tests
