## MODIFIED Requirements

### Requirement: Homepage Dashboard Sections
The system SHALL render the authenticated homepage as a content discovery dashboard containing, at minimum, a top navigation shell, a recently-added hero area when displayable catalog content exists, semantic content-shape sections when grouped catalog content exists, a Continue Watching section when progress data exists, and an explanatory empty or degraded state when content is unavailable because setup, scanning, or active health issues prevent displayable content.

The homepage SHALL NOT require fetching media library lists or latest-by-library groups to render its normal discovery sections.

#### Scenario: Populated homepage
- **WHEN** an authenticated user opens the homepage and the server returns progress, recently-added content, and content-shape sections
- **THEN** the homepage displays the hero area, Continue Watching when applicable, and semantic content sections such as Movies or Series in a vertically scrollable layout
- **AND** the homepage does not render My Media library entrance cards as the primary discovery structure
- **AND** the homepage does not render Latest by Library sections

#### Scenario: Empty homepage
- **WHEN** an authenticated user opens the homepage and no displayable media content is available
- **THEN** the homepage displays an empty state that directs the user toward setup, source configuration, or scanning instead of showing broken rails
- **AND** the empty-state decision is based on displayable content and health/setup state rather than the number of media libraries

#### Scenario: Scanned content hidden by blocking health issue
- **WHEN** an authenticated user opens the homepage, no catalog items are currently displayable, and diagnostics report that affected sources have active blocking issues that impact home visibility
- **THEN** the homepage displays a health-aware empty state explaining that content was scanned or sources exist but are currently unavailable, summarizes the affected issue in user-friendly language, and provides an action to open the Health Center or relevant recovery flow

#### Scenario: Homepage has content and health warnings
- **WHEN** an authenticated user opens the homepage with some displayable catalog content and diagnostics report non-blocking or partially blocking health issues
- **THEN** the homepage continues to display available content while showing a concise degraded-state indicator that links to issue details

### Requirement: Content Shape Homepage Sections
The system SHALL group homepage discovery content into semantic sections based on user-facing content shape rather than media library membership.

Initial semantic sections SHALL include movie and series sections when matching displayable catalog items exist. Section keys SHALL be stable API values and section titles SHALL be suitable for direct homepage presentation.

#### Scenario: Movie content section
- **WHEN** displayable movie catalog items exist for the authenticated homepage feed
- **THEN** the homepage feed includes a movie section containing recent movie cards
- **AND** selecting a movie card opens the existing movie detail or playback flow

#### Scenario: Series content section
- **WHEN** displayable series or show catalog items exist for the authenticated homepage feed
- **THEN** the homepage feed includes a series section containing recent series cards
- **AND** selecting a series card opens the existing series detail flow

#### Scenario: Unsupported or unresolved scanner shapes
- **WHEN** scanned inventory exists only as unresolved, review-required, or unsupported shape output
- **THEN** the homepage does not expose raw scanner shape names as normal discovery section titles
- **AND** the system may surface the condition through health, organizing, governance, or setup-oriented UI instead of the main content rails

### Requirement: Homepage Feed API Contract
The system SHALL provide a homepage feed contract that returns semantic content sections without requiring clients to group by media library.

#### Scenario: Requesting homepage sections
- **WHEN** an authenticated client requests homepage discovery sections
- **THEN** the server returns an ordered list of sections with stable keys, display titles, and catalog list items
- **AND** each section contains only displayable, non-hidden, available catalog items

#### Scenario: No matching content for a section
- **WHEN** no displayable catalog items exist for a semantic section
- **THEN** the server omits that section or returns it empty according to the documented response contract
- **AND** the frontend does not render an empty poster rail as normal content

#### Scenario: Removing latest-by-library from homepage
- **WHEN** the homepage loads its normal data
- **THEN** the frontend does not call the homepage latest-by-library endpoint
- **AND** the frontend does not call the library list endpoint solely to decide homepage discovery content

### Requirement: Continue Watching Rail
The system SHALL render continue-watching entries as a poster rail when the user has in-progress playback records with displayable catalog items.

#### Scenario: Continue playback
- **WHEN** the user selects the continue action for an in-progress item
- **THEN** the app opens playback for that item using the stored progress position when available

#### Scenario: No continue-watching items
- **WHEN** the user has no in-progress playback records
- **THEN** the homepage omits the Continue Watching rail or shows a compact empty state without disrupting other homepage sections
