# metadata-provider-instances Specification

## Purpose
TBD - created by archiving change add-library-metadata-profiles. Update Purpose after archive.
## Requirements
### Requirement: Metadata providers are managed as named runtime instances
The system SHALL manage metadata providers as named instances with provider type, enablement state, configuration payload, and operator-visible identity rather than as a single singleton provider configuration.

#### Scenario: Multiple TMDB tokens are configured independently
- **WHEN** an operator creates two TMDB provider instances with different credentials
- **THEN** the system MUST persist and expose them as distinct selectable provider instances for metadata profiles

### Requirement: Provider instances expose operational availability
The system SHALL track whether a provider instance is enabled for selection and whether it is currently unavailable due to authentication failure, rate limiting, or explicit administrative disablement.

#### Scenario: Rate-limited instance is skipped by profile execution
- **WHEN** a provider instance enters a temporary unavailable state because of provider rate limiting
- **THEN** profile execution MUST skip that instance for eligible fallback attempts until its cooldown or recovery criteria are satisfied

#### Scenario: Disabled instance cannot be selected by new profiles
- **WHEN** an operator disables a provider instance
- **THEN** the system MUST prevent newly saved profile configurations from selecting that instance as an active execution target

### Requirement: Custom provider definitions use the same instance model
The system SHALL allow future custom or non-TMDB provider definitions to be represented through the same provider-instance management model.

#### Scenario: Custom provider instance is registered
- **WHEN** an operator creates a supported custom metadata provider instance
- **THEN** the system MUST store and surface it through the same instance administration and profile-selection model used for TMDB-based instances

