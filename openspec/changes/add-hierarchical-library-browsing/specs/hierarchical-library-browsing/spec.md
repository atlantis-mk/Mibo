## ADDED Requirements

### Requirement: Library browse starts from accessible library roots
The system SHALL expose a hierarchical browse entrypoint that lists the libraries accessible to the current user as the first navigation level instead of immediately returning a flat metadata result set.

#### Scenario: Show accessible libraries as first-level browse nodes
- **WHEN** an authenticated user opens hierarchical library browse without selecting a library
- **THEN** the system SHALL return one browse node per accessible library with the library identifier, display name, node kind, and a navigation target for entering that library

#### Scenario: Hide inaccessible libraries from hierarchical browse
- **WHEN** a user lacks visibility to one or more libraries under the current role scope
- **THEN** the system SHALL exclude those libraries from the first-level browse response

### Requirement: Library browse reveals filesystem-derived child folders before leaf items
The system SHALL derive browse levels from the scanned folder structure under the selected library root and SHALL return the immediate child folders for the current browse path before the user reaches leaf metadata items.

#### Scenario: Return child folders for a library root
- **WHEN** a user enters a library whose root contains child folders such as `中国电影`, `港台电影`, and `欧美电影`
- **THEN** the system SHALL return those folders as browse nodes for the next level

#### Scenario: Return child folders for a nested folder path
- **WHEN** a user opens a folder node that contains deeper subfolders
- **THEN** the system SHALL return only the immediate next-level child folders for that path rather than flattening all descendants into one response

### Requirement: Leaf browse results include metadata items that preserve existing play and detail behavior
The system SHALL return metadata item leaf nodes once the current folder path reaches content-bearing entries, and those leaf nodes SHALL remain addressable by the existing metadata detail and playback flows.

#### Scenario: Folder leaf returns recognized metadata items
- **WHEN** a folder path contains one or more recognized media items linked to metadata
- **THEN** the system SHALL return those items as browse results with the metadata identifiers required for the existing detail and playback routes

#### Scenario: Leaf selection keeps current detail experience
- **WHEN** a user selects a metadata item returned from hierarchical browse
- **THEN** the system SHALL open the same metadata detail and playback experience used for flat browse items

### Requirement: Hierarchical browse preserves browse state for mixed organized and discovered content
The system SHALL support leaf results that represent recognized metadata items and still surface discovered inventory-backed entries that have not yet been fully organized when they are present under the selected folder path.

#### Scenario: Mixed leaf results include organizing entries
- **WHEN** a folder contains both recognized metadata items and discovered inventory files awaiting organization
- **THEN** the system SHALL return both result types with enough state for the client to distinguish organized items from organizing or review-required entries

#### Scenario: Hidden or unavailable items stay excluded
- **WHEN** a metadata item or inventory-backed entry under the selected path is hidden or unavailable according to existing browse rules
- **THEN** the system SHALL exclude that entry from the hierarchical browse response
