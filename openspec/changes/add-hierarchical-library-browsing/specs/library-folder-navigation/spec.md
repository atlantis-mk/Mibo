## ADDED Requirements

### Requirement: Folder browse nodes provide stable navigation identity
The system SHALL assign each hierarchical folder node a stable navigation identity scoped to its library and relative folder path so the client can request the node again after refresh, pagination, or reload.

#### Scenario: Re-open folder node from a stored navigation target
- **WHEN** a client requests a previously returned folder navigation target for the same library and relative path
- **THEN** the system SHALL resolve that request to the same logical folder node as long as the folder still exists in the current browse state

#### Scenario: Reject invalid folder navigation outside the library root
- **WHEN** a client requests a folder path that escapes or falls outside the selected library root
- **THEN** the system SHALL reject the request and SHALL NOT expose content outside the library boundary

### Requirement: Hierarchical browse returns breadcrumb and parent navigation context
The system SHALL return navigation metadata for the current browse node, including the current node identity, parent node identity when applicable, and breadcrumb segments from the library root to the current folder.

#### Scenario: Library root has no parent folder breadcrumb
- **WHEN** a user enters a library root node
- **THEN** the system SHALL return breadcrumb context anchored at the library and SHALL omit a parent folder node beyond that root

#### Scenario: Nested folder includes parent and breadcrumb chain
- **WHEN** a user opens a nested folder path below the library root
- **THEN** the system SHALL return the ordered breadcrumb chain and parent navigation target needed to step back one level or jump to an ancestor

### Requirement: Hierarchical browse supports paged mixed child results
The system SHALL support pagination for the current node's child results while keeping folder and item nodes consistently typed and ordered within the response contract.

#### Scenario: Paginate a folder with many child results
- **WHEN** the selected library or folder contains more child nodes than the configured page size
- **THEN** the system SHALL return a paged response with total count, limit, offset, and enough node data for the client to render the current page

#### Scenario: Preserve node type through pagination
- **WHEN** the client requests later pages for the same browse node
- **THEN** the system SHALL continue labeling each returned result as a library node, folder node, or item node so navigation and leaf actions remain correct

### Requirement: Folder navigation works with library visibility enforcement
The system SHALL apply existing library visibility rules before building hierarchical responses so navigation metadata and child counts only reflect content the current user is allowed to browse.

#### Scenario: Filter descendant navigation by accessible library scope
- **WHEN** a user browses hierarchical content with access to only a subset of libraries
- **THEN** the system SHALL build folder nodes and descendant results only from the libraries in that user's accessible scope

#### Scenario: Prevent direct access to an inaccessible library node
- **WHEN** a client requests a hierarchical browse target for a library the current user cannot access
- **THEN** the system SHALL deny the request using the same authorization model applied to other library browse operations
