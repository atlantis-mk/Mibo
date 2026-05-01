## ADDED Requirements

### Requirement: Libraries support multiple source paths
The system SHALL allow a media library to contain one or more enabled source paths, with each path scoped to a media source and normalized for that source's storage provider.

#### Scenario: Create library with one default path
- **WHEN** a user creates a library with a media source and root path using the existing create-library flow
- **THEN** the system SHALL create the library and one enabled library source path that references the selected media source and normalized root path

#### Scenario: Add a second path to a library
- **WHEN** a user adds a valid path from any configured media source to an existing library
- **THEN** the system SHALL persist the path as an enabled library source path and include it in future scans for that library

#### Scenario: Reject invalid source path
- **WHEN** a user adds or updates a library path that the referenced storage provider cannot resolve
- **THEN** the system SHALL reject the change and SHALL NOT create or update the library path

### Requirement: Library path state controls traversal
The system SHALL traverse only enabled library source paths when scanning, listening, reconciling, or running scheduled library jobs.

#### Scenario: Disabled path is skipped
- **WHEN** a library has one enabled path and one disabled path
- **THEN** a library scan SHALL traverse the enabled path and SHALL NOT traverse the disabled path

#### Scenario: Existing library is migrated to paths
- **WHEN** the application starts after the library-path migration with an existing single-root library
- **THEN** the system SHALL expose one enabled library source path equivalent to the library's previous media source and root path

### Requirement: Library APIs expose paths and compatibility fields
The system SHALL expose library paths in library create, detail, list, and management APIs while preserving existing top-level compatibility fields for single-root consumers.

#### Scenario: Library detail includes paths
- **WHEN** a client requests library detail for a library with multiple source paths
- **THEN** the response SHALL include all library source paths with media source, root path, enabled state, and display metadata

#### Scenario: Compatibility fields remain populated
- **WHEN** a client lists libraries after the migration
- **THEN** each library response SHALL continue to include `media_source_id` and `root_path` populated from the primary enabled path or migrated compatibility values

### Requirement: Libraries have scan policies
The system SHALL support per-library scan policies that control scan participation, realtime/listener behavior, scheduled refresh defaults, hidden-file handling, ignored extensions, minimum file size, sample-size ignores, and configurable rule participation.

#### Scenario: Scan policy defaults preserve current behavior
- **WHEN** an existing library has no explicit scan policy row
- **THEN** the effective scan policy SHALL enable scanning with behavior equivalent to the current implementation

#### Scenario: Ignored extension is skipped
- **WHEN** a library scan policy lists `.txt` as an ignored file extension
- **THEN** scanner traversal SHALL skip matching `.txt` files for that library and SHALL continue processing supported media files

#### Scenario: Minimum file size is zero
- **WHEN** a library scan policy sets minimum file size to `0`
- **THEN** scanner traversal SHALL NOT skip files based on minimum file size

#### Scenario: File below minimum size is skipped
- **WHEN** a library scan policy sets minimum file size to `1024` bytes and a video file is smaller than `1024` bytes
- **THEN** scanner traversal SHALL skip that video file for the library

#### Scenario: Realtime policy disables listener refresh
- **WHEN** a library scan policy disables realtime/listener refresh
- **THEN** storage listener reconciliation SHALL NOT enqueue targeted refresh jobs for that library solely because of provider change events

### Requirement: Scan exclusions remain authoritative
The system SHALL apply manual scan exclusions and configured scan exclusion rules before or alongside policy-driven ignore decisions, and SHALL preserve their existing audit and reason semantics.

#### Scenario: Manual exclusion wins over policy
- **WHEN** a file matches both a manual scan exclusion and a policy ignore rule
- **THEN** the scanner SHALL skip the file and record the manual scan exclusion as the authoritative source of the skip decision

#### Scenario: Configurable exclusion rules still apply
- **WHEN** a library scan policy enables configurable exclusion rules and a path matches an enabled exclusion rule
- **THEN** the scanner SHALL skip the path using the configured exclusion rule reason

### Requirement: Libraries have metadata policies
The system SHALL support per-library metadata policies for preferred metadata language, image language, country or region, local metadata participation, provider enablement, and provider priority.

#### Scenario: Metadata language overrides global default
- **WHEN** a library metadata policy sets preferred metadata language to `zh-CN`
- **THEN** metadata searches and refreshes for that library SHALL use `zh-CN` as the preferred language unless a more specific operation explicitly overrides it

#### Scenario: Disabled metadata provider is not used
- **WHEN** a library metadata policy disables TMDB
- **THEN** metadata search, match, and refresh operations for items in that library SHALL NOT call TMDB for automated provider lookup

#### Scenario: Local sidecar metadata follows policy
- **WHEN** a library metadata policy disables local metadata participation
- **THEN** scanner-discovered sidecar metadata SHALL NOT overwrite or create catalog metadata hints for that library

### Requirement: Metadata governance overrides metadata policy
The system SHALL preserve existing catalog governance protections when applying metadata according to a library policy.

#### Scenario: Locked field is preserved
- **WHEN** a catalog item field is locked and metadata refresh runs under any library metadata policy
- **THEN** the refresh SHALL NOT overwrite the locked field value

#### Scenario: Review-needed item remains protected
- **WHEN** a catalog item requires governance review
- **THEN** automated metadata policy behavior SHALL NOT silently replace curated or review-protected descriptive fields

### Requirement: Libraries have playback policies
The system SHALL support per-library playback policies for resume enablement and resume thresholds used when recording or interpreting user progress.

#### Scenario: Resume threshold marks item complete
- **WHEN** a playback progress update crosses the library playback policy's maximum resume percentage
- **THEN** the system SHALL treat the item as completed according to that policy's threshold

#### Scenario: Short item does not record progress
- **WHEN** a media item duration is below the library playback policy's minimum resume duration
- **THEN** the system SHALL NOT store resumable progress for that playback session

### Requirement: Libraries have subtitle policies
The system SHALL support per-library subtitle policies for external sidecar subtitle binding, subtitle language preferences, strict matching, saving or exposing external subtitles, and tolerance of unavailable subtitle files.

#### Scenario: External subtitles disabled
- **WHEN** a library subtitle policy disables external sidecar subtitles
- **THEN** scanner sidecar discovery SHALL NOT bind matching subtitle files as playable external subtitle tracks for that library

#### Scenario: Preferred subtitle languages filter playback response
- **WHEN** a library subtitle policy lists preferred subtitle languages
- **THEN** playback responses for items in that library SHALL prioritize or filter subtitle tracks according to that language policy while preserving safe Mibo-controlled URLs

#### Scenario: Missing subtitle remains non-fatal
- **WHEN** a bound external subtitle file cannot be resolved during playback and the library subtitle policy tolerates unavailable subtitles
- **THEN** the playback response SHALL remain valid for the source media and SHALL NOT expose raw provider internals

### Requirement: Library management UI exposes paths and policies
The system SHALL provide library management UI controls for viewing and editing source paths, scan policy, metadata policy, playback policy, and subtitle policy without requiring advanced fields during basic library creation.

#### Scenario: Basic library creation stays minimal
- **WHEN** a user creates a new library from the settings UI
- **THEN** the form SHALL require only the current essential library fields and SHALL apply default policies automatically

#### Scenario: Advanced policies are editable after creation
- **WHEN** a user opens an existing library in settings
- **THEN** the UI SHALL allow the user to review and update paths and policy groups in focused sections
