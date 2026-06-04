## ADDED Requirements

### Requirement: Leaf directories are classified from direct primary video siblings
The system SHALL classify the innermost video-containing directory from its direct video children and SHALL NOT use parent or sibling directory structure to decide the leaf directory shape.

#### Scenario: Direct sibling videos form one leaf cluster
- **WHEN** a directory contains multiple direct video files
- **THEN** the leaf classifier MUST build one cluster from those direct video files only and classify that cluster independently of parent directory layout

#### Scenario: Nested child videos are not pulled into the leaf cluster
- **WHEN** a directory contains child directories that also contain videos
- **THEN** the leaf classifier MUST classify each video-containing child directory as its own leaf cluster before any parent scope reduction occurs

### Requirement: Token residuals drive bottom-level shape decisions
The system SHALL use filename token residual/cancellation across primary sibling videos as a primary signal for distinguishing episode packs, movie versions, multipart movies, and movie collections.

#### Scenario: Episode residual tokens classify an episode pack
- **WHEN** sibling videos share common title/release tokens and each residual token contains a consistent season-episode marker such as `S01E01` and `S01E02`
- **THEN** the leaf classifier MUST classify the directory as `episode_pack` or `season_folder` according to folder season evidence and MUST record the residual episode evidence

#### Scenario: Version residual tokens classify movie versions
- **WHEN** sibling videos share a common movie identity and their residual tokens are only quality, source, codec, HDR, release, or edition tokens
- **THEN** the leaf classifier MUST classify the directory as `movie_versions_folder` and MUST record the version residual evidence

#### Scenario: Multipart residual tokens classify multipart movies
- **WHEN** sibling videos share common movie tokens and residual tokens form a continuous `part`, `pt`, `disc`, `disk`, or `CD` sequence
- **THEN** the leaf classifier MUST classify the directory as `multipart_movie_folder` and MUST preserve part ordering evidence

#### Scenario: Distinct title residuals classify collections
- **WHEN** sibling videos do not share a dominant work identity and residual tokens contain distinct movie title or title-year identities
- **THEN** the leaf classifier MUST classify the directory as `movie_collection_folder` when no stronger episodic or conflict evidence exists

### Requirement: Primary media controls leaf shape
The system SHALL compute leaf directory shape from primary playable videos while treating trailers, samples, previews, featurettes, extras, and behind-the-scenes videos as supplemental media.

#### Scenario: Supplemental videos do not override a main movie
- **WHEN** a directory contains one primary movie video plus trailer or extras videos
- **THEN** the leaf classifier MUST classify the main shape from the primary movie video and assign supplemental videos as attachments when possible

#### Scenario: Extras-only directory becomes attachment group
- **WHEN** a directory contains only supplemental videos or has explicit extras/trailers/sample path hints
- **THEN** the leaf classifier MUST classify the directory as `attachment_group` or review-required attachment evidence rather than as a normal movie or episode group

#### Scenario: Ambiguous specials require stronger evidence
- **WHEN** a directory or filename uses ambiguous labels such as `SP`, `OVA`, `Specials`, or `番外`
- **THEN** the leaf classifier MUST require explicit episode, season, sidecar, or duration/identity evidence before treating the media as episode content; otherwise it MUST classify the media as attachment or review-required

### Requirement: Leaf summaries are persisted and reusable
The system SHALL persist leaf classification summaries with enough evidence for upward scope detection, materialization diagnostics, and cache invalidation.

#### Scenario: Leaf summary records structural facts
- **WHEN** the system classifies a leaf directory
- **THEN** it MUST persist the leaf path, shape, dominant identity, title evidence, season set, episode set, part set, version signature, attachment roles, confidence, review state, covered file IDs or paths, residual-token evidence, classifier version, and fingerprint

#### Scenario: Unchanged leaf summaries are reused
- **WHEN** directory snapshot, inventory facts, file signals, sidecar evidence, scan policy, and leaf classifier version are unchanged
- **THEN** the system MUST reuse the persisted leaf summary instead of recomputing leaf classification

#### Scenario: Changed leaf inputs invalidate downstream decisions
- **WHEN** a leaf summary fingerprint changes
- **THEN** any metadata scope decisions, recognition units, and directory metadata resolution payloads that depend on that leaf summary MUST be regenerated or invalidated

### Requirement: Leaf conflicts remain review-safe
The system SHALL require review instead of forcing a confident leaf shape when residual evidence, sidecar hints, or parsed filename signals conflict.

#### Scenario: Episode and movie evidence conflict
- **WHEN** sibling videos have strong episode residual evidence and strong distinct movie title-year evidence
- **THEN** the leaf classifier MUST create a review-required summary with both alternatives recorded

#### Scenario: Broken multipart sequence requires review
- **WHEN** multipart residual tokens have missing, duplicated, or conflicting part numbers
- **THEN** the leaf classifier MUST classify the directory as review-required or use another safer shape only when stronger evidence supports it

#### Scenario: Sidecar conflicts are preserved
- **WHEN** NFO or sidecar shape evidence contradicts the filename-derived leaf shape
- **THEN** the leaf classifier MUST record the conflict and require review unless an explicit configured rule resolves the conflict
