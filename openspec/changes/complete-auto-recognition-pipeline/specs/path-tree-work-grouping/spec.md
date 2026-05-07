## ADDED Requirements

### Requirement: Path tree compiler creates work groups before catalog projection
The system SHALL compile path-tree work groups from already-scanned directory snapshots, indexed file signals, content-shape profiles, and content-shape plans before catalog materialization chooses final movie, series, episode, version, attachment, or review semantics.

#### Scenario: Parent directory contains child release folders
- **WHEN** a parent directory contains multiple child directories and each child directory contains a likely main video
- **THEN** the work-group compiler SHALL evaluate the children together before each child is independently materialized as a catalog work

#### Scenario: Leaf content-shape plan exists
- **WHEN** a leaf directory already has a content-shape plan
- **THEN** the path-tree work-group compiler SHALL use the leaf plan as evidence rather than reparsing the leaf files or replacing the content-shape planner

### Requirement: Sibling movie release folders become one version group
The system SHALL group sibling one-file movie release folders into one movie work with multiple source assets when their indexed file signals share a normalized title/year work key and their differences are primarily release hints.

#### Scenario: Same movie has two release folders
- **WHEN** the source contains sibling folders for `3.Iron.2004...MiniHD/3.Iron.2004...MiniHD.mkv` and `3-Iron.2004...TAGHD/3-Iron.2004...TAGHD.mkv`
- **THEN** the scanner SHALL create or reuse one movie catalog item for `3 Iron` year `2004`
- **AND** it SHALL link both video files as separate assets or versions of that movie

#### Scenario: Release differences are technical only
- **WHEN** sibling release folders differ by quality, source, codec, audio, HDR, edition, release group, or container while sharing the same title/year work key
- **THEN** those differences SHALL be treated as version evidence rather than independent movie identity evidence

#### Scenario: Titles differ materially
- **WHEN** sibling child folders have different normalized title/year work keys and low version evidence
- **THEN** the scanner SHALL NOT merge them into one movie version group

### Requirement: Movie collections split into independent movie groups
The system SHALL split collection directories into independent movie work groups when child files or child directories have distinct movie-like title/year evidence and weak episode sequence evidence.

#### Scenario: Collection directory contains multiple movie files
- **WHEN** a directory contains `Alien.1979.mkv`, `Aliens.1986.mkv`, and `Heat.1995.mkv`
- **THEN** the scanner SHALL create separate movie work groups for each distinct title/year key
- **AND** it SHALL NOT treat the directory as one movie, one series, or one version group

#### Scenario: Collection directory contains one-file child movie folders
- **WHEN** a directory contains multiple child directories and each child has one main video with a distinct title/year work key
- **THEN** the scanner SHALL create separate movie groups for each child rather than treating the parent as one work

### Requirement: Series roots group season-like children
The system SHALL group sibling season folders, episode packs, or season-like directories under a shared series work group when their indexed signals and path evidence indicate one series.

#### Scenario: Series has season children
- **WHEN** a parent contains `Season 1` and `Season 2` child directories with episode-like videos
- **THEN** the scanner SHALL create one series root and season descendants rather than two unrelated works

#### Scenario: Series has noisy season children
- **WHEN** sibling directories include noisy release tokens but expose compatible series title and season evidence
- **THEN** the scanner SHALL normalize them under one series work group when confidence thresholds are met

### Requirement: Work groups preserve evidence and alternatives
The system SHALL persist or expose work-group shape, confidence, review state, work key, assignments, evidence, and alternatives for scanner-created groups.

#### Scenario: Work group is ambiguous
- **WHEN** movie-version, movie-collection, and episode-pack evidence are close or below threshold
- **THEN** the scanner SHALL create a review-required work-group decision with candidate alternatives and affected file paths

#### Scenario: Work group is high confidence
- **WHEN** one work-group candidate exceeds the configured confidence threshold and margin
- **THEN** the scanner SHALL materialize using that group while retaining evidence linking files to the chosen work key

### Requirement: Work group plans are reusable and invalidated safely
The system SHALL reuse work-group plans for unchanged parent scopes and SHALL invalidate or recompile them when child file signals, child directory fingerprints, scan policies, exclusion rules, classifier version, or scoped correction rules change.

#### Scenario: Parent directory is unchanged
- **WHEN** a later scan observes the same parent fingerprint, child plan fingerprints, and classifier version
- **THEN** the scanner SHALL reuse the previous work-group plan and assignments without recompiling every child directory

#### Scenario: New sibling conflicts with existing plan
- **WHEN** a new child directory or file conflicts with a reused movie-version, collection, or series work-group plan
- **THEN** the scanner SHALL recompile the work group or mark it review-required instead of forcing the old rule
*** Add File: openspec/changes/complete-auto-recognition-pipeline/specs/media-graph-scanner/spec.md
## MODIFIED Requirements

### Requirement: Scanner builds media graph candidates before catalog writes
The system SHALL group scanned files, sibling directories, current-directory siblings, sidecars, indexed filename-derived signals, cached directory summaries, content-shape plans, path-tree work-group plans, and learned classification rules into media graph candidates before writing catalog items, and SHALL treat directory shape as evidence rather than a final semantic type.

#### Scenario: Directory contains multiple episode-like videos
- **WHEN** a source directory contains multiple likely main videos that resolve to explicit or inferred episode slots
- **THEN** the scanner MUST create a single series candidate for that directory before projecting episode catalog descendants

#### Scenario: Movie folder contains multiple main-like files
- **WHEN** a source directory contains multiple plausible main video files for the same movie work
- **THEN** the scanner MUST create one movie candidate with multiple asset or version candidates instead of creating one movie per file

#### Scenario: Directory contains independent movies
- **WHEN** a source directory contains multiple likely main videos with distinct movie-like title or year evidence and no episode-sequence evidence
- **THEN** the scanner MUST preserve separate movie candidates instead of forcing the directory into one movie, one series, or one mixed semantic type

#### Scenario: Filename signals include release metadata
- **WHEN** scanned files include filename-derived release hints such as quality, source, codec, audio, subtitle, edition, or release group
- **THEN** the scanner MUST use those hints as candidate evidence for grouping and title cleanup without treating them as authoritative technical facts

#### Scenario: Sibling directories contain versions of one movie
- **WHEN** sibling child directories each contain one main video and those videos share a normalized title/year work key with release-hint differences
- **THEN** the scanner MUST create one movie candidate with multiple source assets before catalog projection

#### Scenario: Parent directory contains independent one-file movie folders
- **WHEN** sibling child directories each contain one main video with distinct title/year work keys
- **THEN** the scanner MUST preserve separate movie candidates instead of merging the parent into one work

### Requirement: Resolver decisions expose evidence and confidence
The system SHALL represent scanner grouping and classification as resolver decisions with candidate type, role, confidence, alternatives, filename-derived signal evidence, directory summary evidence, content-shape evidence, work-group evidence, review state, affected files, and reason text.

#### Scenario: Series candidate is inferred from a flat episode folder
- **WHEN** a resolver groups a flat source-first folder into a series candidate
- **THEN** the decision MUST include the target series identity, inferred season and episode slots when available, confidence, evidence references, alternatives considered, and a reason explaining the grouping

#### Scenario: Classification is ambiguous
- **WHEN** the scanner cannot confidently distinguish movie, episode, version, independent work, or attachment semantics
- **THEN** the resolver decision MUST preserve candidate evidence and mark the projected catalog item or relationship for governance review instead of silently creating unrelated works

#### Scenario: Attachment is detected
- **WHEN** a video file is classified as trailer, extra, sample, preview, or another non-main role
- **THEN** the resolver decision MUST expose the attachment role and evidence so catalog projection can link it to a likely parent work without treating it as a standalone movie or episode

#### Scenario: Audio token prevents episode false positive
- **WHEN** a resolver rejects weak episode inference because a numeric-looking token is classified as filename-derived audio evidence
- **THEN** the decision MUST expose that anti-misclassification evidence in its reason or evidence summary

#### Scenario: Work group overrides leaf materialization
- **WHEN** a parent work-group plan groups files from multiple child directories into one movie or series candidate
- **THEN** the resolver decision MUST expose the parent work-group evidence and the leaf alternatives that were superseded
*** Add File: openspec/changes/complete-auto-recognition-pipeline/specs/fast-video-classification/spec.md
## MODIFIED Requirements

### Requirement: Fast classifier uses bounded sibling grouping
The system SHALL use indexed file signals, cached current-directory summary evidence, content-shape plans, and bounded path-tree work-group evidence derived from already-listed snapshots to distinguish episode sequences, movie version groups, sibling-directory movie versions, independent movie files, series roots, and attachments without recursively scanning the full source for classification context.

#### Scenario: Siblings form an episode sequence
- **WHEN** likely main videos in the same directory have shared title evidence and consecutive episode numbers
- **THEN** the classifier SHALL group them as episode candidates for the same series and season when confidence thresholds are met

#### Scenario: Siblings look like movie versions
- **WHEN** likely main videos in the same directory share a normalized title stem and differ mainly by quality, edition, cut, container, language, or release tokens
- **THEN** the classifier SHALL group them as one movie candidate with multiple asset/version candidates

#### Scenario: Siblings look like independent movies
- **WHEN** likely main videos in the same directory have different title stems or year evidence and no episode sequence evidence
- **THEN** the classifier SHALL preserve independent movie candidates rather than merging them into one movie or one episode sequence

#### Scenario: Directory summary already exists
- **WHEN** multiple files in the same scanned directory require sibling context
- **THEN** the classifier SHALL reuse the cached directory summary for that scan snapshot rather than recomputing sibling evidence per file or issuing additional storage listings

#### Scenario: Sibling directories look like movie versions
- **WHEN** bounded parent path-tree evidence shows one-file child directories with the same title/year work key and release-hint differences
- **THEN** the classifier SHALL group the child files as one movie candidate with multiple assets without calling remote metadata providers

### Requirement: Fast path avoids heavy work
The fast classifier SHALL complete using cheap storage, path, filename, sidecar-name, already-listed object metadata, indexed file signals, structured filename signals, cached directory summary evidence, content-shape plans, and bounded path-tree work-group evidence only, and SHALL NOT perform media-content reads, technical probing, hashing, external provider lookup, or artwork retrieval.

#### Scenario: Source scan classifies a video file
- **WHEN** the scanner performs fast video classification during inventory traversal
- **THEN** it SHALL use path strings, filenames, extensions, sidecar filenames, already-listed object metadata, indexed file signals, cached directory snapshots, bounded current-directory summary context, and path-tree work-group context

#### Scenario: Expensive evidence is needed
- **WHEN** a classification decision requires duration, stream metadata, hashes, TMDB, MetaTube, TVDB, or artwork evidence to become reliable
- **THEN** the fast classifier SHALL leave the decision provisional or review-required and SHALL rely on asynchronous jobs for validation
*** Add File: openspec/changes/complete-auto-recognition-pipeline/specs/source-first-auto-classification/spec.md
## MODIFIED Requirements

### Requirement: Low-confidence classifications are reviewable after scanning
The system SHALL surface low-confidence or conflicting classifier decisions for review after scanning rather than requiring users to make semantic choices before scanning, and SHALL base video classification on staged filename signal extraction, indexed file signal reuse, file-role detection, candidate generation, cached directory summary evidence, content-shape plans, path-tree work-group evidence, confidence thresholds, and reviewable evidence.

#### Scenario: Classifier cannot confidently choose movie or series
- **WHEN** video classification evidence is ambiguous or below the configured confidence threshold
- **THEN** the system SHALL preserve the inventory facts and create a reviewable decision with evidence, confidence, candidate alternatives, and proposed action

#### Scenario: User reviews an ambiguous decision
- **WHEN** a user opens the review surface for an ambiguous source item or group
- **THEN** the UI SHALL show the proposed classification, confidence, candidate alternatives, supporting evidence, affected files, and concrete correction actions so the user can correct or accept the decision

#### Scenario: Fast classification avoids heavy work
- **WHEN** automatic video classification runs during source-first scanning
- **THEN** the system SHALL use path, filename, extension, sidecar-name, already-listed object metadata, indexed filename signals, structured filename signals, cached directory summary evidence, and bounded path-tree work-group evidence without running ffprobe, content hashing, external metadata searches, artwork downloads, or additional recursive source analysis in the fast path

#### Scenario: Attachment evidence avoids false semantic choices
- **WHEN** a supported video looks like a trailer, sample, PV, preview, featurette, or other non-main attachment
- **THEN** the system SHALL classify it as an attachment candidate and SHALL NOT require the user to choose movie, show, mixed, or directory semantics before scanning continues

#### Scenario: Directory context is needed for numeric filenames
- **WHEN** numeric filenames cannot be confidently classified from filename signals alone
- **THEN** the system SHALL use cached directory summary evidence when available and SHALL mark the result provisional or review-required if the cheap context remains inconclusive

#### Scenario: Work-group context resolves sibling release folders
- **WHEN** sibling release folders can be confidently grouped as one movie version work group from local evidence
- **THEN** the system SHALL apply the grouping automatically without prompting the user before scan completion
*** Add File: openspec/changes/complete-auto-recognition-pipeline/specs/metadata-operation-pipeline/spec.md
## MODIFIED Requirements

### Requirement: Metadata operations execute through a unified pipeline
The system SHALL execute automated match, refetch, manual candidate apply, and local evidence apply through a shared metadata operation pipeline that resolves target item, library strategy, execution plan, provider attempts, metadata decision, field application, and projection refresh as one operation. Automated scanner-triggered matching SHALL be queued per recognized movie or series work group rather than per source file or per version asset.

#### Scenario: Automated match uses unified pipeline
- **WHEN** a queued catalog item match job runs for a pending catalog item
- **THEN** the system MUST execute a metadata operation of type `match` and return or persist a result that includes the target item, execution plan summary, provider attempts, selected candidate when present, applied fields, skipped fields, resulting governance status, and affected catalog item IDs

#### Scenario: Refetch uses unified pipeline
- **WHEN** a user refetches metadata for an item that already has provider identity
- **THEN** the system MUST execute a metadata operation of type `refetch` using the same execution plan, provider attempt, field application, and projection refresh semantics as automated matching

#### Scenario: Manual candidate apply uses unified pipeline
- **WHEN** a user applies a selected metadata candidate to a catalog item
- **THEN** the system MUST execute a metadata operation of type `manual_apply` and MUST record manual operation status, provider provenance, applied fields, skipped locked fields, and affected catalog item IDs

#### Scenario: Movie version work group is scanned
- **WHEN** the scanner materializes one movie item with multiple version assets
- **THEN** the metadata pipeline SHALL queue at most one automated match for the movie work group and SHALL NOT queue separate remote searches for each version file

#### Scenario: Series work group is scanned
- **WHEN** the scanner materializes a series with season or episode descendants
- **THEN** the metadata pipeline SHALL queue matching for the series root and SHALL NOT match each episode independently in the fast path
*** Add File: openspec/changes/complete-auto-recognition-pipeline/specs/catalog-metadata-governance/spec.md
## MODIFIED Requirements

### Requirement: Governance states distinguish automatic, review, unmatched, and manual outcomes
The system SHALL assign governance status through metadata operation decisions and scanner recognition decisions based on confidence, classifier decisions, work-group decisions, match outcome, local evidence policy, metadata evidence, and user actions, including matched, needs-review, unmatched, manual, and locked-equivalent states needed to explain why a catalog item, work group, or asset relationship is or is not automatically governed.

#### Scenario: Low-confidence match enters review instead of silent acceptance
- **WHEN** metadata matching yields ambiguous provider candidates or conflicting numbering evidence
- **THEN** the system MUST mark the item with a review-oriented governance status and retain the evidence needed for a user to confirm, reject, or lock the canonical result

#### Scenario: Low-confidence classification enters review instead of silent acceptance
- **WHEN** scanner classification yields ambiguous movie, episode, attachment, version, or independent-work candidates
- **THEN** the system MUST mark the affected item, asset relationship, or candidate group with review-oriented governance status and retain classifier evidence, alternatives, confidence, and proposed actions

#### Scenario: Low-confidence work group enters review
- **WHEN** path-tree work-group recognition yields ambiguous movie-version, movie-collection, series, or episode-pack candidates
- **THEN** the system MUST mark the affected work group or guarded placeholder with review-oriented governance status and retain work-group evidence, alternatives, confidence, affected files, and proposed correction actions

#### Scenario: Manual apply records manual outcome
- **WHEN** a user manually applies a metadata candidate
- **THEN** the system MUST mark the item as manually governed or equivalent and record the operation evidence needed to explain the selected candidate and field changes

#### Scenario: No candidate records unmatched outcome
- **WHEN** an automated match operation completes all eligible search or local evidence attempts without a usable candidate
- **THEN** the system MUST mark the item unmatched and retain attempt evidence explaining why no metadata was applied

### Requirement: Governance corrections can create classification rules
The system SHALL allow user-approved classification corrections to persist source-scoped rules that future scans can use as classifier or work-group evidence.

#### Scenario: User resolves files as an episode sequence
- **WHEN** a user confirms that an ambiguous group should be treated as a named series season with sorted or explicit episode numbering
- **THEN** governance SHALL persist a source-scoped classification rule and SHALL update the affected catalog projection or asset relationships according to the confirmed decision

#### Scenario: User resolves files as movie versions
- **WHEN** a user confirms that sibling files or sibling directories are versions of one movie
- **THEN** governance SHALL persist a source-scoped classification rule and SHALL link the files as assets or versions of the confirmed movie work

#### Scenario: User resolves files as independent movies
- **WHEN** a user confirms that sibling files or sibling directories are independent movies rather than a movie version group or episode sequence
- **THEN** governance SHALL persist a source-scoped classification rule or decision record that prevents future scans from merging those files into one work

#### Scenario: User resolves parent as movie collection
- **WHEN** a user confirms that a parent directory is a movie collection
- **THEN** governance SHALL persist a source-scoped work-group rule that future scans use to split children by title/year work key
