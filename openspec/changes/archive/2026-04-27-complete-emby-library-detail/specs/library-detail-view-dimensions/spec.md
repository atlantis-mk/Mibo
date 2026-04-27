## ADDED Requirements

### Requirement: Library detail exposes view dimension tabs
The system SHALL expose library view tabs for content, recommendations, trailers, favorites, genres, tags, platforms, episodes, and folders using Chinese labels aligned with the product UI.

#### Scenario: Display library view tabs
- **WHEN** a user opens a library detail page
- **THEN** the page displays tabs equivalent to `节目`, `推荐`, `预告`, `收藏`, `类型`, `标签`, `播出平台`, `集`, and `文件夹`

### Requirement: Primary content tab shows poster grid
The system SHALL map the primary content tab to the main library poster grid for the active library.

#### Scenario: Open primary content tab
- **WHEN** a user selects the `节目` tab
- **THEN** the page displays the main library browse results as poster cards

### Requirement: Favorites tab is scoped to the current library
The system SHALL show only user-favorited items that belong to the current library when the favorites tab is selected.

#### Scenario: Open library favorites tab
- **WHEN** a user selects the `收藏` tab for a library
- **THEN** the page displays favorited items from that library and excludes favorites from other libraries

### Requirement: Unsupported dimensions have bounded empty states
The system SHALL render clear empty or coming-soon states for view dimensions whose data is not available instead of presenting broken or inert controls.

#### Scenario: Open unsupported dimension
- **WHEN** a user selects a tab whose backing data is not yet available
- **THEN** the page explains that the dimension has no data or is not connected yet

### Requirement: Library detail exposes compact filter and more actions
The system SHALL provide compact `筛选` and `更多` toolbar actions for browse controls and secondary library actions.

#### Scenario: Open filter controls
- **WHEN** a user activates the `筛选` control
- **THEN** the page reveals the available filter inputs without losing the active tab or sort state
