## ADDED Requirements

### Requirement: Product flows shall not read legacy catalog ownership models
The system SHALL serve product browse, detail, hierarchy, playback, progress, favorites, search, home, and governance flows without requiring library-owned `CatalogItem` metadata, `MediaAsset` playback versions, `AssetItem` links, or `UserItemData` state.

#### Scenario: Product route opens after a fresh resource scan
- **WHEN** a fresh database is scanned and a user opens any primary product route
- **THEN** the route MUST load from metadata items, resources, resource-library links, resource-metadata links, library projections, and metadata/resource user data rather than legacy catalog ownership rows

#### Scenario: Legacy rows are absent
- **WHEN** old catalog item, asset, asset-item, or user-item rows are absent from the database
- **THEN** supported product flows MUST remain functional for scanned media that exists in the resource graph

### Requirement: Retired compatibility APIs shall be removed
The system SHALL remove API handlers and frontend wrappers that only exist for retired media-item, media-file, catalog-item-owned asset, or asset-selected playback compatibility.

#### Scenario: Frontend code references retired wrappers
- **WHEN** frontend code is typechecked after the removal
- **THEN** no imports or calls to retired media item/file or asset-governance wrappers MUST remain

#### Scenario: Backend router is inspected
- **WHEN** backend routes are registered
- **THEN** retired media item/file and asset-governance compatibility routes MUST NOT be registered

### Requirement: Legacy database models shall not be automigrated after removal
The system SHALL stop registering database models that are only needed for retired catalog read-model behavior once all live code paths have been replaced.

#### Scenario: Fresh development database starts
- **WHEN** the backend starts against an empty development database
- **THEN** it MUST create the resource-first schema needed by current product flows without recreating retired catalog read-model tables

#### Scenario: Old database is reused
- **WHEN** a developer points the backend at an old local database containing retired tables
- **THEN** the system MAY leave those tables untouched, but current product flows MUST NOT depend on them

### Requirement: Legacy read-model tests shall be replaced
The system SHALL delete or rewrite tests whose only asserted behavior is library-owned `CatalogItem`, asset-link, or `UserItemData` semantics.

#### Scenario: Test suite runs after removal
- **WHEN** `go test ./...` and frontend type/build checks run after the removal
- **THEN** tests MUST assert metadata/resource/projection behavior for equivalent product outcomes and MUST NOT require retired catalog read-model behavior
