## ADDED Requirements

### Requirement: Library metadata strategies support MetaTube provider instances
The system SHALL allow library metadata strategies and reusable metadata templates to reference configured MetaTube provider instances for supported metadata stages, while preserving validation for unsupported stages.

#### Scenario: Library strategy selects MetaTube for metadata matching
- **WHEN** a library metadata strategy configures a MetaTube provider instance for supported search and detail stages
- **THEN** metadata matching and manual search for movie catalog items in that library MUST resolve the MetaTube instance from the strategy instead of falling back to global TMDB settings

#### Scenario: Library strategy rejects MetaTube hierarchy provider
- **WHEN** a library metadata strategy configures a MetaTube provider instance for the hierarchy stage
- **THEN** the strategy update MUST be rejected because MetaTube does not provide Mibo TV hierarchy semantics
