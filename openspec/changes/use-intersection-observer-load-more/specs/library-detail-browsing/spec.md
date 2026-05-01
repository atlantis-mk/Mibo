## MODIFIED Requirements

### Requirement: Library detail supports browsing beyond the first page
The system SHALL provide incremental loading so users can browse all results returned by a library view, including libraries larger than the initial request size, and SHALL automatically request additional results when the load-more sentinel enters the library scroll viewport while more results are available.

#### Scenario: Load additional library results
- **WHEN** a library has more results than the initial page contains
- **THEN** the page exposes a way to load or navigate to additional results without changing the active filters

#### Scenario: Automatically load more near the bottom
- **WHEN** a user scrolls a browseable library view until the load-more sentinel intersects the library scroll viewport and additional results are available
- **THEN** the page requests the next result page using the same active filters and appends those results to the existing grid

#### Scenario: Avoid duplicate automatic loads
- **WHEN** the load-more sentinel intersects while a next-page request is already in progress or no additional results are available
- **THEN** the page does not start another next-page request
