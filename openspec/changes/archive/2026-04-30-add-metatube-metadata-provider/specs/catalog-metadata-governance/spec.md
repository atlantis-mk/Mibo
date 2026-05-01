## ADDED Requirements

### Requirement: Metadata governance preserves MetaTube provenance
The system SHALL preserve MetaTube provider-instance provenance and provider-specific external identities when applying metadata from MetaTube.

#### Scenario: MetaTube metadata source is recorded
- **WHEN** metadata from a MetaTube provider instance is applied to a catalog item
- **THEN** metadata source evidence MUST identify source type provider, source name `metatube`, the executing provider instance ID and name, the upstream MetaTube provider, the upstream item ID, and the fallback summary used by execution

#### Scenario: MetaTube identity is distinct from TMDB identity
- **WHEN** a catalog item has metadata identities from both TMDB and MetaTube
- **THEN** governance reads and refetch logic MUST keep the identities distinct and MUST NOT treat a MetaTube upstream ID as a TMDB ID
