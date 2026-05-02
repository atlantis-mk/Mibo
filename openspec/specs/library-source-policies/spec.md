# library-source-policies Specification

## Purpose
Define source-first paths and per-source policies for scanning, metadata, playback, subtitles, and management without requiring user-selected movie, show, or mixed semantics.

## Requirements
### Requirement: Libraries support multiple source paths
The system SHALL allow a source-first content collection to contain one or more enabled source paths, with each path scoped to a media source and normalized for that source's storage provider.

#### Scenario: Create source-first collection with one default path
- **WHEN** a user creates content from a media source and root path
- **THEN** the system SHALL create the source-first collection and one enabled source path that references the selected media source and normalized root path without requiring a movie, show, or mixed type

#### Scenario: Add a second path to a source-first collection
- **WHEN** a user adds a valid path from any configured media source to an existing source-first collection
- **THEN** the system SHALL persist the path as an enabled source path and include it in future scans for that collection

#### Scenario: Reject invalid source path
- **WHEN** a user adds or updates a source path that the referenced storage provider cannot resolve
- **THEN** the system SHALL reject the change and SHALL NOT create or update the source path

### Requirement: Library path state controls traversal
The system SHALL traverse only enabled source paths when scanning, listening, reconciling, or running scheduled source jobs.

#### Scenario: Disabled path is skipped
- **WHEN** a source-first collection has one enabled path and one disabled path
- **THEN** a scan SHALL traverse the enabled path and SHALL NOT traverse the disabled path

#### Scenario: Existing library migration is not required
- **WHEN** this development-stage rebuild is applied
- **THEN** the system SHALL NOT be required to migrate existing single-root movie, show, or mixed libraries into source-first paths

### Requirement: Library APIs expose paths and compatibility fields
The system SHALL expose source paths in create, detail, list, and management APIs and SHALL NOT require legacy top-level movie, show, or mixed type compatibility fields for source-first consumers.

#### Scenario: Source detail includes paths
- **WHEN** a client requests detail for a source-first collection with multiple source paths
- **THEN** the response SHALL include all source paths with media source, root path, enabled state, probe summary, and display metadata

#### Scenario: Legacy type compatibility is not required
- **WHEN** a client lists source-first collections after this rebuild
- **THEN** each response SHALL NOT be required to include a user-selected `movies`, `shows`, or `mixed` type field

### Requirement: Libraries have scan policies
The system SHALL support per-source scan policies that control scan participation, realtime/listener behavior, scheduled refresh defaults, hidden-file handling, ignored extensions, minimum file size, sample-size ignores, configurable rule participation, and probe budgets.

#### Scenario: Scan policy defaults preserve source-first behavior
- **WHEN** a source-first collection has no explicit scan policy row
- **THEN** the effective scan policy SHALL enable probing and scanning with default budgeted source-first behavior

#### Scenario: Ignored extension is skipped
- **WHEN** a source scan policy lists `.txt` as an ignored file extension
- **THEN** scanner traversal SHALL skip matching `.txt` files for that source and SHALL continue processing supported media files

#### Scenario: Minimum file size is zero
- **WHEN** a source scan policy sets minimum file size to `0`
- **THEN** scanner traversal SHALL NOT skip files based on minimum file size

#### Scenario: File below minimum size is skipped
- **WHEN** a source scan policy sets minimum file size to `1024` bytes and a video file is smaller than `1024` bytes
- **THEN** scanner traversal SHALL skip that video file for the source

#### Scenario: Realtime policy disables listener refresh
- **WHEN** a source scan policy disables realtime/listener refresh
- **THEN** storage listener reconciliation SHALL NOT enqueue targeted refresh jobs for that source solely because of provider change events

### Requirement: Scan exclusions remain authoritative
The system SHALL apply manual scan exclusions and configured scan exclusion rules before or alongside policy-driven ignore decisions, and SHALL preserve their existing audit and reason semantics.

#### Scenario: Manual exclusion wins over policy
- **WHEN** a file matches both a manual scan exclusion and a policy ignore rule
- **THEN** the scanner SHALL skip the file and record the manual scan exclusion as the authoritative source of the skip decision

#### Scenario: Configurable exclusion rules still apply
- **WHEN** a source scan policy enables configurable exclusion rules and a path matches an enabled exclusion rule
- **THEN** the scanner SHALL skip the path using the configured exclusion rule reason

### Requirement: Libraries have metadata policies
The system SHALL support per-source metadata strategies for preferred metadata language, image language, country or region metadata, and ordered provider instances per metadata stage. The executable strategy SHALL resolve directly from source strategy state instead of legacy local metadata participation booleans, provider enablement flags, provider priority strings, or user-selected movie/show library type.

#### Scenario: Metadata language overrides global default
- **WHEN** a source metadata strategy sets preferred metadata language to `zh-CN`
- **THEN** metadata searches and refreshes for catalog items from that source SHALL use `zh-CN` as the preferred language unless a more specific operation explicitly overrides it

#### Scenario: Ordered metadata providers drive execution
- **WHEN** a source metadata strategy configures `tmdb-primary` before `tmdb-backup` for a stage
- **THEN** metadata execution for items from that source MUST attempt those providers in the configured order instead of consulting legacy provider enablement or priority fields

#### Scenario: Local scan participation follows strategy membership
- **WHEN** a source metadata strategy omits the built-in `local_scan` provider from the detail stage
- **THEN** scanner-discovered sidecar metadata evidence MAY remain recorded for catalog history but MUST NOT be selected as an executable metadata provider during strategy-driven detail refresh for that source

### Requirement: Metadata governance overrides metadata policy
The system SHALL preserve existing catalog governance protections when applying metadata according to a source policy.

#### Scenario: Locked field is preserved
- **WHEN** a catalog item field is locked and metadata refresh runs under any source metadata policy
- **THEN** the refresh SHALL NOT overwrite the locked field value

#### Scenario: Review-needed item remains protected
- **WHEN** a catalog item requires governance review
- **THEN** automated metadata policy behavior SHALL NOT silently replace curated or review-protected descriptive fields

### Requirement: Libraries have playback policies
The system SHALL support per-source playback policies for resume enablement and resume thresholds used when recording or interpreting user progress.

#### Scenario: Resume threshold marks item complete
- **WHEN** a playback progress update crosses the source playback policy's maximum resume percentage
- **THEN** the system SHALL treat the item as completed according to that policy's threshold

#### Scenario: Short item does not record progress
- **WHEN** a media item duration is below the source playback policy's minimum resume duration
- **THEN** the system SHALL NOT store resumable progress for that playback session

### Requirement: Libraries have subtitle policies
The system SHALL support per-source subtitle policies for external sidecar subtitle binding, subtitle language preferences, strict matching, saving or exposing external subtitles, and tolerance of unavailable subtitle files.

#### Scenario: External subtitles disabled
- **WHEN** a source subtitle policy disables external sidecar subtitles
- **THEN** scanner sidecar discovery SHALL NOT bind matching subtitle files as playable external subtitle tracks for that source

#### Scenario: Preferred subtitle languages filter playback response
- **WHEN** a source subtitle policy lists preferred subtitle languages
- **THEN** playback responses for items from that source SHALL prioritize or filter subtitle tracks according to that language policy while preserving safe Mibo-controlled URLs

#### Scenario: Missing subtitle remains non-fatal
- **WHEN** a bound external subtitle file cannot be resolved during playback and the source subtitle policy tolerates unavailable subtitles
- **THEN** the playback response SHALL remain valid for the source media and SHALL NOT expose raw provider internals

### Requirement: Library management UI exposes paths and policies
The system SHALL provide source management UI controls for viewing and editing source paths, scan policy, executable metadata strategy, optional template application, playback policy, and subtitle policy without requiring users to choose movie, show, or mixed semantics during basic source creation.

#### Scenario: Basic source creation stays minimal
- **WHEN** a user creates a new source from setup or settings UI
- **THEN** the form SHALL require only storage source and root path and SHALL NOT expose movie, show, or mixed type choices by default

#### Scenario: Advanced metadata strategy is editable after creation
- **WHEN** a user opens an existing source in settings
- **THEN** the UI SHALL allow the user to review and update ordered provider instances per stage, language overrides, and reusable template application in focused sections

### Requirement: Library metadata APIs expose executable strategy state
The system SHALL expose source metadata strategy management APIs as a first-class configuration surface with ordered provider instance IDs per stage, language overrides, and optional template context.

#### Scenario: Client reads source metadata strategy
- **WHEN** a client requests metadata strategy settings for a source
- **THEN** the response MUST include the source's effective stage ordering and language overrides without requiring the client to resolve a separate runtime profile binding

#### Scenario: Client updates source metadata strategy
- **WHEN** a client saves a new source metadata strategy
- **THEN** the system MUST validate provider-stage compatibility and persist the executable strategy independently of reusable template definitions

### Requirement: Library metadata strategies support MetaTube provider instances
The system SHALL allow source metadata strategies and reusable metadata templates to reference configured MetaTube provider instances for supported metadata stages, while preserving validation for unsupported stages.

#### Scenario: Source strategy selects MetaTube for metadata matching
- **WHEN** a source metadata strategy configures a MetaTube provider instance for supported search and detail stages
- **THEN** metadata matching and manual search for movie catalog items from that source MUST resolve the MetaTube instance from the strategy instead of falling back to global TMDB settings

#### Scenario: Source strategy rejects MetaTube hierarchy provider
- **WHEN** a source metadata strategy configures a MetaTube provider instance for the hierarchy stage
- **THEN** the strategy update MUST be rejected because MetaTube does not provide Mibo TV hierarchy semantics
