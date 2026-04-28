## ADDED Requirements

### Requirement: Sidecar subtitles are bound to media assets
The system SHALL bind discovered `.srt` and `.ass` sidecar subtitle files to the same media asset as the matched video so they can be treated as external subtitle tracks rather than evidence-only metadata.

#### Scenario: Movie subtitle sidecar is bound
- **WHEN** a movie scan discovers `Movie A.mkv` and a basename-matched `Movie A.srt`
- **THEN** the system MUST create or reuse an inventory file for the subtitle sidecar, link it to the movie asset as a subtitle attachment, and record a subtitle media stream with external disposition

#### Scenario: Episode subtitle sidecar is bound
- **WHEN** an episode scan discovers `Show S01E02.mkv` and a basename-matched `Show S01E02.ass`
- **THEN** the system MUST bind the sidecar subtitle to the episode's selected media asset without changing the episode hierarchy or classification

### Requirement: Sidecar subtitle binding is idempotent
The system SHALL reconcile scanner-managed sidecar subtitle bindings on rescan without creating duplicates or leaving stale scanner-owned subtitle attachments for the same asset.

#### Scenario: Subtitle sidecar remains present on rescan
- **WHEN** a library rescan sees the same matched subtitle sidecar for an already scanned video
- **THEN** the system MUST reuse or update the existing subtitle inventory/link/stream records instead of creating duplicate subtitle tracks

#### Scenario: Subtitle sidecar is removed before rescan
- **WHEN** a library rescan no longer sees a previously scanner-managed sidecar subtitle for the video asset
- **THEN** the system MUST remove or mark unavailable the stale scanner-managed subtitle binding while preserving the source video asset

### Requirement: Sidecar subtitle playback uses safe Mibo URLs
The system SHALL expose playable sidecar subtitle tracks through Mibo-controlled URLs or file identities without exposing raw provider signatures, mount details, or auth-bearing storage internals.

#### Scenario: OpenList subtitle sidecar is playable
- **WHEN** playback is requested for an OpenList-backed item with a bound sidecar subtitle
- **THEN** the playback response MUST include a subtitle track that can be fetched through a Mibo endpoint or safe file URL and MUST NOT include raw OpenList `sign` or mount detail values

#### Scenario: Sidecar subtitle cannot be resolved
- **WHEN** a bound sidecar subtitle file is missing or cannot be linked by the storage provider
- **THEN** the playback response MUST remain valid for the source media and MUST NOT fail the entire playback request because of the unavailable subtitle
