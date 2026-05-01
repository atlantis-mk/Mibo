## MODIFIED Requirements

### Requirement: Libraries have metadata policies
The system SHALL support per-library metadata strategies for preferred metadata language, image language, country or region metadata, and ordered provider instances per metadata stage. The executable strategy SHALL resolve directly from library strategy state instead of legacy local metadata participation booleans, provider enablement flags, or provider priority strings.

#### Scenario: Metadata language overrides global default
- **WHEN** a library metadata strategy sets preferred metadata language to `zh-CN`
- **THEN** metadata searches and refreshes for that library SHALL use `zh-CN` as the preferred language unless a more specific operation explicitly overrides it

#### Scenario: Ordered metadata providers drive execution
- **WHEN** a library metadata strategy configures `tmdb-primary` before `tmdb-backup` for a stage
- **THEN** metadata execution for items in that library MUST attempt those providers in the configured order instead of consulting legacy provider enablement or priority fields

#### Scenario: Local scan participation follows strategy membership
- **WHEN** a library metadata strategy omits the built-in `local_scan` provider from the detail stage
- **THEN** scanner-discovered sidecar metadata evidence MAY remain recorded for catalog history but MUST NOT be selected as an executable metadata provider during strategy-driven detail refresh for that library

### Requirement: Library management UI exposes paths and policies
The system SHALL provide library management UI controls for viewing and editing source paths, scan policy, executable metadata strategy, optional template application, playback policy, and subtitle policy without requiring synthetic local-only profiles during basic library creation.

#### Scenario: Basic library creation stays minimal
- **WHEN** a user creates a new library from the settings UI
- **THEN** the form SHALL require only the current essential library fields and SHALL allow a simple metadata source choice or default strategy without exposing per-stage strategy editing by default

#### Scenario: Advanced metadata strategy is editable after creation
- **WHEN** a user opens an existing library in settings
- **THEN** the UI SHALL allow the user to review and update ordered provider instances per stage, language overrides, and reusable template application in focused sections

## ADDED Requirements

### Requirement: Library metadata APIs expose executable strategy state
The system SHALL expose library metadata strategy management APIs as a first-class configuration surface with ordered provider instance IDs per stage, language overrides, and optional template context.

#### Scenario: Client reads library metadata strategy
- **WHEN** a client requests metadata strategy settings for a library
- **THEN** the response MUST include the library's effective stage ordering and language overrides without requiring the client to resolve a separate runtime profile binding

#### Scenario: Client updates library metadata strategy
- **WHEN** a client saves a new library metadata strategy
- **THEN** the system MUST validate provider-stage compatibility and persist the executable strategy independently of reusable template definitions
