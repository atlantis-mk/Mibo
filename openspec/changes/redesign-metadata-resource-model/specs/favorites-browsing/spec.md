## MODIFIED Requirements

### Requirement: Favorites target metadata identities
The favorites system SHALL store favorites against global metadata identities rather than library-owned catalog items.

#### Scenario: Favorite visible through another library
- **WHEN** a user favorites a metadata identity through one library and later browses favorites through another library containing a linked resource
- **THEN** the favorite resolves to that metadata identity with the available library/resource context

#### Scenario: Favorite hidden from library
- **WHEN** a metadata identity is favorited but hidden or unprojected in a requested library
- **THEN** library-filtered favorites omit it while global favorites can still include it if accessible elsewhere
