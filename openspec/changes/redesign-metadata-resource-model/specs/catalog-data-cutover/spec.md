## MODIFIED Requirements

### Requirement: Catalog cutover retires old ownership model
Catalog data cutover SHALL replace library-owned catalog metadata with metadata/resource/projection storage.

#### Scenario: Development data reset
- **WHEN** the new model is applied in development
- **THEN** old catalog metadata data can be reset and rebuilt by rescanning media sources

#### Scenario: Old catalog code removed
- **WHEN** new browse, search, metadata, and playback paths are implemented
- **THEN** old catalog ownership code that is no longer used is deleted rather than retained as a compatibility shim

### Requirement: Retired routes do not drive new flows
New product flows SHALL NOT depend on retired legacy media item or file routes.

#### Scenario: Frontend uses new semantics
- **WHEN** the frontend browses, opens detail, favorites, or plays media
- **THEN** it uses metadata identity and resource semantics rather than retired catalog item ownership semantics
