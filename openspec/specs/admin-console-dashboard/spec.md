# admin-console-dashboard Specification

## Purpose
Define the authenticated administrative console dashboard, including layout, navigation, server overview, operational metrics, quick actions, activity, device visibility, and resilient loading states.

## Requirements

### Requirement: Console route and layout
The system SHALL provide an authenticated admin console page with a fixed management sidebar, top page actions, and a dashboard-oriented main content area.

#### Scenario: Authenticated user opens console
- **WHEN** an authenticated user navigates to `/console`
- **THEN** the system displays the console page with the page title `控制台`, a back/navigation affordance, top-right quick entry buttons, a server status section, an activity timeline, and management entry sections

#### Scenario: Unauthenticated user opens console
- **WHEN** an unauthenticated user navigates to `/console`
- **THEN** the system applies the existing setup and authentication gates before showing console content

#### Scenario: Console is selected in sidebar
- **WHEN** the current route is `/console`
- **THEN** the sidebar highlights the `控制台` entry with the primary green active state

### Requirement: Grouped management sidebar
The system SHALL organize console navigation in a left management sidebar with grouped entries for Mibo web settings, user preferences, server management, devices, and advanced operations.

#### Scenario: Sidebar shows grouped admin entries
- **WHEN** the console sidebar is rendered
- **THEN** it includes grouped entries for console, users, media libraries, metadata, network, transcoding, database, conversions, scheduled tasks, logs, plugins, devices, downloads, camera uploads, DLNA, and advanced maintenance

#### Scenario: Unsupported entries are visible but disabled
- **WHEN** a sidebar or management entry points to a capability that is not implemented yet
- **THEN** the system displays it as unavailable or coming soon rather than linking to a broken route

### Requirement: Server overview card
The system SHALL display a server overview card that summarizes operational state using compact, high-density status fields.

#### Scenario: Server overview data is available
- **WHEN** console summary data loads successfully
- **THEN** the overview card displays server name, service status, version, update status, API port, uptime, database status, storage provider, storage root, and worker/module status where available

#### Scenario: Partial server overview data is unavailable
- **WHEN** one or more overview fields cannot be determined
- **THEN** the system displays a clear `未知` or `未配置` state for those fields without failing the entire console page

### Requirement: Access address display
The system SHALL display local, LAN, and remote access addresses with explicit availability states.

#### Scenario: Access addresses are known
- **WHEN** access information is available
- **THEN** the console displays local access, LAN access, and remote access addresses with copyable or visibly selectable URL text

#### Scenario: Remote access is not configured
- **WHEN** no remote access address is configured
- **THEN** the console displays remote access as `未配置` and provides a route or action to open network settings when available

### Requirement: Dashboard metrics
The system SHALL display concise metric cards for media library and operational status.

#### Scenario: Metrics are available
- **WHEN** console summary data includes media and operational counts
- **THEN** the console displays counts for libraries, media sources, catalog items, inventory files, movies, series, episodes, people, online or recent devices, active jobs, scan state, and warning/error count where available

#### Scenario: Metrics are loading
- **WHEN** dashboard metrics are still loading
- **THEN** the console displays skeleton or pending states that preserve the dashboard layout

### Requirement: Quick actions
The system SHALL provide quick actions for common management operations and clearly distinguish safe navigation actions from potentially expensive operations.

#### Scenario: User runs a safe quick action
- **WHEN** the user selects a safe quick action such as opening settings, opening logs, or opening media library management
- **THEN** the system navigates to the corresponding route without confirmation

#### Scenario: User runs an expensive quick action
- **WHEN** the user selects an expensive action such as scanning libraries, running a consistency check, or rebuilding catalog projections
- **THEN** the system asks for confirmation before invoking the operation and then displays success, failure, or queued-job feedback

#### Scenario: Quick action is not supported
- **WHEN** a quick action such as shutdown, Premiere status, plugins, camera upload, or DLNA is not implemented
- **THEN** the system displays the action as disabled or unavailable with a short explanatory label

### Requirement: Activity timeline
The system SHALL display recent playback and operational activity as a chronological timeline.

#### Scenario: Activity events are available
- **WHEN** recent activity data exists
- **THEN** the console displays timeline rows with an icon, severity/status style, event description, user when available, device when available, media title when available, and timestamp

#### Scenario: No activity events exist
- **WHEN** no recent activity data exists
- **THEN** the console displays an empty state explaining that recent playback, scan, and system events will appear there

#### Scenario: Activity includes warnings
- **WHEN** an activity event has warning or error severity
- **THEN** the console visually distinguishes it with warning or danger styling while preserving chronological order

### Requirement: Management entry grid
The system SHALL provide management entry cards that route users to relevant administrative areas or show unavailable states for planned capabilities.

#### Scenario: Management entries are rendered
- **WHEN** the console page loads
- **THEN** it displays entry cards for users, media libraries, live TV, network, transcoding, database, conversions, scheduled tasks, logs, plugins, devices, downloads, camera upload, DLNA, and advanced maintenance categories

#### Scenario: Implemented management entry is selected
- **WHEN** the user selects an implemented management entry
- **THEN** the system navigates to the existing route or opens the appropriate action flow

### Requirement: Device-related section
The system SHALL provide device-related console entries for connected or recent devices and planned device workflows.

#### Scenario: Device data is available
- **WHEN** connected or recent device data exists
- **THEN** the console displays device name, client type when available, user when available, current playback state when available, and last-seen time

#### Scenario: Device features are not implemented
- **WHEN** downloads, camera upload, or DLNA are not implemented
- **THEN** their entries remain visible as disabled or unavailable states without implying that they are active

### Requirement: Top-right quick entries
The system SHALL expose top-right quick entry controls for casting or playback target selection, current user, and settings.

#### Scenario: User opens top-right settings
- **WHEN** the user selects the settings quick entry
- **THEN** the system navigates to the existing settings area

#### Scenario: Casting is unavailable
- **WHEN** playback-to-device or casting is not implemented
- **THEN** the cast quick entry is visible as unavailable or disabled instead of triggering a broken workflow

### Requirement: Console visual design
The system SHALL use a light administrative visual style with green primary emphasis, compact cards, clear status colors, and responsive behavior.

#### Scenario: Console renders on desktop
- **WHEN** the console is viewed on a desktop-width viewport
- **THEN** the sidebar remains fixed or persistently available, server status spans the main content width, metric cards use a grid layout, and management entries remain information-dense

#### Scenario: Console renders on mobile
- **WHEN** the console is viewed on a mobile-width viewport
- **THEN** the sidebar uses the existing mobile behavior, cards stack vertically, action groups remain tappable, and no core server status or activity information is lost

#### Scenario: Status colors are applied
- **WHEN** a status is normal, warning, error, unavailable, or disabled
- **THEN** the console uses green, yellow, red, gray, or muted styling consistently for that status

### Requirement: Console data loading and failure handling
The system SHALL handle loading, partial failure, and full failure states without presenting stale data as healthy.

#### Scenario: Console summary request fails
- **WHEN** the console summary request fails
- **THEN** the console displays an error state with retry affordance and does not show misleading healthy statuses

#### Scenario: Console summary partially fails
- **WHEN** the backend can provide some dashboard sections but not others
- **THEN** the console renders available sections and marks failed sections with warning states and explanatory text
