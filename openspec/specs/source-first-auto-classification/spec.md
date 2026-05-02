# source-first-auto-classification Specification

## Purpose
Define source-first content creation, bounded source probing, automatic content-class detection, and reviewable low-confidence classification without requiring users to choose media semantics up front.

## Requirements
### Requirement: Source creation does not require media semantics
The system SHALL allow users to add a content source by selecting a storage source and root path without choosing movie, show, mixed, video, audio, or text semantics up front.

#### Scenario: User adds a source path
- **WHEN** a user submits a valid storage source and root path from setup or settings
- **THEN** the system SHALL accept the source path without requiring a movie, show, mixed, or broad content-class selection

#### Scenario: Source name is derived when omitted
- **WHEN** a user adds a source path without entering a display name
- **THEN** the system SHALL derive a stable display name from the path or storage source metadata

### Requirement: Source probing is fast and bounded
The system SHALL run a lightweight source probe after a source path is provided and SHALL enforce probe limits for duration, object count, and traversal depth.

#### Scenario: Probe samples source contents
- **WHEN** a source path is accepted
- **THEN** the system SHALL sample directory entries using provider listing, path names, file extensions, and already-available object metadata without reading media file contents

#### Scenario: Probe budget is exhausted
- **WHEN** probing reaches its configured duration, object-count, or depth limit before fully traversing the source
- **THEN** the system SHALL stop probing and return a partial result marked as budget-limited

#### Scenario: Probe avoids heavy work
- **WHEN** source probing runs
- **THEN** the system MUST NOT run ffprobe, calculate file-content hashes, call external metadata providers, or download artwork as part of the synchronous probe

### Requirement: Source probe reports content-class distribution
The system SHALL report detected content classes from the probe using broad classes that are separate from catalog semantic item types.

#### Scenario: Probe finds mixed file extensions
- **WHEN** sampled source entries include supported video, audio, text, image, and unknown extensions
- **THEN** the probe result SHALL include counts or percentages for `video`, `audio`, `text`, `image`, and `other`

#### Scenario: Probe cannot determine a dominant class
- **WHEN** sampled source evidence does not produce a dominant content class
- **THEN** the system SHALL still accept the source and mark the probe result as mixed or uncertain instead of requiring user selection

### Requirement: Content collections are generated from detected classes
The system SHALL create or expose source-scoped content collections/views from detected content classes without requiring users to create separate semantic libraries.

#### Scenario: Video content is detected
- **WHEN** the source probe or subsequent scan detects supported video files
- **THEN** the system SHALL expose a video collection/view for that source and enqueue video inventory and classification work

#### Scenario: Non-video content is detected before deep support exists
- **WHEN** the source probe detects audio, text, image, or other files that do not yet have deep catalog support
- **THEN** the system SHALL preserve their inventory visibility or summary counts without blocking video scanning

### Requirement: Initial source feedback is available before deep enrichment
The system SHALL return source acceptance and probe feedback before full recursive scanning, technical probing, metadata matching, or artwork enrichment completes.

#### Scenario: Source is accepted
- **WHEN** the user adds a valid source path
- **THEN** the system SHALL return the accepted source and probe summary while background jobs continue asynchronously

#### Scenario: Background work progresses
- **WHEN** inventory, classification, metadata, or artwork jobs continue after source creation
- **THEN** the UI SHALL be able to show progress or partial catalog results without waiting for all jobs to complete

### Requirement: Low-confidence classifications are reviewable after scanning
The system SHALL surface low-confidence or conflicting classifier decisions for review after scanning rather than requiring users to make semantic choices before scanning, and SHALL base video classification on staged file-role detection, candidate generation, bounded sibling grouping, confidence thresholds, and reviewable evidence.

#### Scenario: Classifier cannot confidently choose movie or series
- **WHEN** video classification evidence is ambiguous or below the configured confidence threshold
- **THEN** the system SHALL preserve the inventory facts and create a reviewable decision with evidence, confidence, candidate alternatives, and proposed action

#### Scenario: User reviews an ambiguous decision
- **WHEN** a user opens the review surface for an ambiguous source item or group
- **THEN** the UI SHALL show the proposed classification, confidence, candidate alternatives, supporting evidence, affected files, and concrete correction actions so the user can correct or accept the decision

#### Scenario: Fast classification avoids heavy work
- **WHEN** automatic video classification runs during source-first scanning
- **THEN** the system SHALL use path, filename, extension, sidecar-name, already-listed object metadata, and bounded current-directory sibling evidence without running ffprobe, content hashing, external metadata searches, or artwork downloads in the fast path

#### Scenario: Attachment evidence avoids false semantic choices
- **WHEN** a supported video looks like a trailer, sample, PV, preview, featurette, or other non-main attachment
- **THEN** the system SHALL classify it as an attachment candidate and SHALL NOT require the user to choose movie, show, mixed, or directory semantics before scanning continues
