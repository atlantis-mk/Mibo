## MODIFIED Requirements

### Requirement: Home dashboard aggregates projections
The home dashboard SHALL aggregate library status and latest media from library metadata projections and resource availability.

#### Scenario: Latest items by library
- **WHEN** a library has recently added resources linked to metadata identities
- **THEN** the home dashboard lists latest projected metadata items for that library using projection latest-added ordering

#### Scenario: Same metadata in multiple libraries
- **WHEN** the same metadata identity is projected into multiple libraries
- **THEN** the home dashboard can show it separately per library according to each library projection

### Requirement: Home dashboard reports resource-backed health
The home dashboard SHALL report blocking or organizing states from resource/projection ingest conditions rather than library-owned catalog rows.

#### Scenario: Projection awaiting metadata
- **WHEN** a resource is linked but its metadata identity still requires metadata matching or review
- **THEN** the home dashboard can surface the affected library through projection/resource conditions
