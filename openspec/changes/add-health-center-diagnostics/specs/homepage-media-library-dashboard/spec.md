## MODIFIED Requirements

### Requirement: Homepage Dashboard Sections
The system SHALL render the authenticated homepage as a media dashboard containing, at minimum, a top navigation shell, a recently-added hero area when displayable catalog content exists, a My Media library entrance section, a Continue Watching section when progress data exists, Latest by Library sections when recent library content exists, and an explanatory empty or degraded state when content is unavailable because setup, scanning, or active health issues prevent displayable content.

#### Scenario: Populated homepage
- **WHEN** an authenticated user opens the homepage and the server returns libraries, progress, and latest content
- **THEN** the homepage displays the hero area, My Media, Continue Watching, and Latest by Library sections in a vertically scrollable layout

#### Scenario: Empty homepage
- **WHEN** an authenticated user opens the homepage and no libraries or media content are available
- **THEN** the homepage displays an empty state that directs the user toward media-library setup or scanning instead of showing broken rails

#### Scenario: Scanned content hidden by blocking health issue
- **WHEN** an authenticated user opens the homepage, no catalog items are currently displayable, and diagnostics report that affected libraries or media sources have active blocking issues that impact home visibility
- **THEN** the homepage displays a health-aware empty state explaining that content was scanned or libraries exist but are currently unavailable, summarizes the affected issue in user-friendly language, and provides an action to open the Health Center or relevant recovery flow

#### Scenario: Homepage has content and health warnings
- **WHEN** an authenticated user opens the homepage with some displayable catalog content and diagnostics report non-blocking or partially blocking health issues
- **THEN** the homepage continues to display available content while showing a concise degraded-state indicator that links to issue details
