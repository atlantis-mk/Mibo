## ADDED Requirements

### Requirement: Library detail shows accurate browse totals
The system SHALL display the server-reported total number of browse results for the active library view and active filters instead of using only the currently loaded item count.

#### Scenario: Display filtered total count
- **WHEN** a user opens a library detail page with active filters that match 79 browse results
- **THEN** the page displays copy equivalent to `共 79 项`

### Requirement: Library detail supports browsing beyond the first page
The system SHALL provide pagination or incremental loading so users can browse all results returned by a library view, including libraries larger than the initial request size.

#### Scenario: Load additional library results
- **WHEN** a library has more results than the initial page contains
- **THEN** the page exposes a way to load or navigate to additional results without changing the active filters

### Requirement: Library detail supports title sort direction
The system SHALL allow users to sort library results by title in ascending and descending order.

#### Scenario: Toggle title sort order
- **WHEN** a user selects title sort and toggles the direction control
- **THEN** the item order changes between ascending and descending title order using the same active filters

### Requirement: Library detail uses responsive poster grid layout
The system SHALL render library results as a poster-first responsive grid that remains usable on desktop and mobile screens.

#### Scenario: Browse posters on mobile
- **WHEN** a user opens the library detail page on a narrow viewport
- **THEN** poster cards remain readable, tappable, and vertically scrollable without horizontal page overflow
