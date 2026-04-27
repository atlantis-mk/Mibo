## ADDED Requirements

### Requirement: Library detail provides title-section quick index
The system SHALL provide a vertical alphabetical or character index for title-sorted library grids when enough title sections are present.

#### Scenario: Display quick index for title sort
- **WHEN** a library grid is sorted by title and contains multiple title sections
- **THEN** the page displays a quick index that represents the visible title sections

### Requirement: Quick index jumps to grid sections
The system SHALL jump to the corresponding poster-grid section when a user selects an index entry.

#### Scenario: Jump to selected section
- **WHEN** a user selects an index entry such as `M`
- **THEN** the page scrolls to the section containing titles grouped under that entry

### Requirement: Quick index handles mixed title scripts
The system SHALL group titles with Latin letters, numbers, symbols, and CJK characters into stable section keys.

#### Scenario: Group mixed titles
- **WHEN** the loaded grid contains Chinese titles, numeric titles, Latin titles, and symbol-prefixed titles
- **THEN** each item appears under a deterministic quick-index section

### Requirement: Quick index is hidden when not useful
The system SHALL hide or disable the quick index when the active sort is not title-based or when too few sections exist.

#### Scenario: Use non-title sort
- **WHEN** a user sorts the library by recent date or year
- **THEN** the quick index is hidden or shown as unavailable
