## ADDED Requirements

### Requirement: Library tags
The system MUST allow administrators to create library access tags, reuse existing library access tags, and assign multiple tags to a library.

#### Scenario: Assign existing tags to a library
- **WHEN** an administrator selects one or more existing library access tags for a library
- **THEN** the system SHALL persist the tag assignments for that library

#### Scenario: Create and assign a new tag
- **WHEN** an administrator creates a new library access tag while editing a library
- **THEN** the system SHALL persist the new tag and bind it to the selected library

#### Scenario: Unlabeled library remains accessible
- **WHEN** a library has no assigned library access tags
- **THEN** the system SHALL treat that library as accessible to all authenticated users unless another library-level restriction applies

### Requirement: Role-based library visibility rules
The system MUST allow roles to declare allowed and denied library access tags and MUST evaluate denied tags before allowed tags.

#### Scenario: Allow access by matching tag
- **WHEN** a user's effective role rules include an allow rule that matches at least one tag assigned to a library and no deny rule matches that library
- **THEN** the system SHALL treat the library as accessible to that user

#### Scenario: Deny access by matching tag
- **WHEN** a user's effective role rules include a deny rule that matches at least one tag assigned to a library
- **THEN** the system SHALL treat the library as inaccessible to that user even if another role allows the same tag or library

#### Scenario: No allow rules means default open
- **WHEN** a user's effective role rules contain no allow rules
- **THEN** the system SHALL allow access to every library that does not match a deny rule

### Requirement: Content visibility follows accessible libraries
The system MUST derive content visibility from the set of libraries accessible to the user and MUST apply that derived scope consistently across content read paths.

#### Scenario: Hide content that exists only in inaccessible libraries
- **WHEN** a metadata item is available only through libraries that are inaccessible to the user
- **THEN** the system SHALL exclude that item from browse, home, search, favorites, continue watching, and recently played results

#### Scenario: Show content available in at least one accessible library
- **WHEN** a metadata item exists in both accessible and inaccessible libraries
- **THEN** the system SHALL continue to show the item to the user because it remains available through an accessible library

#### Scenario: Restrict item resources to accessible libraries
- **WHEN** the user opens item details for content available through both accessible and inaccessible libraries
- **THEN** the system SHALL return only resources, file links, and playback choices sourced from accessible libraries

### Requirement: Playback and direct resource access respect library visibility
The system MUST prevent playback or direct media resource access from inaccessible libraries even when the user knows the item identifier or playback route.

#### Scenario: Block playback from inaccessible library resources
- **WHEN** the selected playback candidate belongs only to a library that is inaccessible to the user
- **THEN** the system SHALL deny playback for that candidate

#### Scenario: Allow playback from accessible library resources
- **WHEN** the selected playback candidate belongs to an accessible library
- **THEN** the system SHALL permit playback subject to the existing playback authorization checks
