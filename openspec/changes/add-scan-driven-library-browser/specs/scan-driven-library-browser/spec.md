## ADDED Requirements

### Requirement: Hierarchical browse reads projection display roots as its primary placement source
The system SHALL build hierarchical library browse nodes from projection display-root fields before attempting any browse-time folder inference.

#### Scenario: Folder tree groups items by stored display root
- **WHEN** a library browse request loads a folder level for projections with stored display-root paths
- **THEN** the system SHALL group child folders and leaf items according to those stored projection paths

#### Scenario: Leaf item appears when current path matches stored display root
- **WHEN** the current hierarchical browse path equals a projection's stored display root
- **THEN** the system SHALL return that metadata item directly as a leaf result instead of requiring another inferred folder level

### Requirement: Hierarchical browse preserves category folders that are not structural media directories
The system SHALL avoid collapsing category or collection folders into direct metadata items solely because they temporarily contain a single title.

#### Scenario: Single-title category folder remains browsable
- **WHEN** a category folder such as `动作` or `欧美经典` contains only one title beneath it and that folder is not classified as a structural child directory
- **THEN** the system SHALL keep that folder as a navigation node and SHALL NOT replace it with a direct item at the parent level

#### Scenario: Structural media directory still surfaces item directly
- **WHEN** a child folder represents a media-root directory or only contains structural child directories such as season or split-part folders
- **THEN** the system SHALL allow the matching metadata item to surface directly at that browse level

### Requirement: Hierarchical browse falls back safely when projection display roots are unavailable
The system SHALL continue to serve hierarchical browse results when projections have not yet been rebuilt with display-root fields by using the existing folder-inference fallback.

#### Scenario: Existing library remains browsable before backfill completes
- **WHEN** a hierarchical browse request encounters projection rows without stored display-root semantics
- **THEN** the system SHALL use the compatibility inference path so the library remains browsable during migration

#### Scenario: Rebuilt projection replaces fallback placement
- **WHEN** the same metadata item later gains stored display-root semantics after projection refresh
- **THEN** subsequent browse responses SHALL use the stored display path as the authoritative placement

### Requirement: Scan-driven hierarchical browse preserves existing leaf behavior
The system SHALL keep existing metadata detail, playback, visibility, and organizing-state behavior unchanged after an item is surfaced from its scan-driven display root.

#### Scenario: Direct series item opens existing detail flow
- **WHEN** a series is surfaced directly from its stored series-root directory in hierarchical browse
- **THEN** selecting that result SHALL use the same metadata detail and playback entry points as other series items

#### Scenario: Authorization still filters surfaced items
- **WHEN** a metadata item would otherwise map to the current display path but is hidden or inaccessible under existing browse rules
- **THEN** the system SHALL exclude that item from the browse response
