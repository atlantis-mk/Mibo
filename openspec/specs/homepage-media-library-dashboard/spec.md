# homepage-media-library-dashboard Specification

## Purpose
Define the authenticated homepage dashboard structure for media library discovery and continued viewing.

## Requirements

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

### Requirement: My Media Library Entrances
The system SHALL display each available media library as a clickable My Media card using the library name, library metadata where available, and a multi-poster collage built from representative catalog items.

#### Scenario: Entering a library
- **WHEN** the user selects a My Media library card
- **THEN** the app navigates to that library's detail route and shows the library's catalog contents

#### Scenario: Library without representative posters
- **WHEN** a library has no representative poster images
- **THEN** the library card uses a stable fallback visual while preserving the library name and navigation target

### Requirement: Latest By Library Rails
The system SHALL group recently updated or recently added catalog items by media library and render each group as a horizontal poster rail with a title that links to the full library view.

#### Scenario: Opening a latest group
- **WHEN** the user activates the title or arrow for a latest-by-library section
- **THEN** the app navigates to the corresponding library detail route

#### Scenario: Scrolling latest content
- **WHEN** a latest-by-library section contains more cards than fit in the viewport
- **THEN** the section supports horizontal scrolling without blocking the page's vertical scrolling

### Requirement: Continue Watching Rail
The system SHALL render continue-watching entries as a poster rail when the user has in-progress playback records with displayable catalog items.

#### Scenario: Continue playback
- **WHEN** the user selects the continue action for an in-progress item
- **THEN** the app opens playback for that item using the stored progress position when available

#### Scenario: No continue-watching items
- **WHEN** the user has no in-progress playback records
- **THEN** the homepage omits the Continue Watching rail or shows a compact empty state without disrupting other homepage sections
