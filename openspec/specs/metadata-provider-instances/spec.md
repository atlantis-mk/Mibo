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
The system SHALL track whether a provider instance is enabled for selection and whether it is currently unavailable due to authentication failure, rate limiting, cooldown, explicit administrative disablement, or unsupported execution capability, and metadata operations SHALL use that availability to decide provider attempts.

#### Scenario: Rate-limited instance is skipped by profile execution
- **WHEN** a provider instance enters a temporary unavailable state because of provider rate limiting
- **THEN** metadata operation execution MUST skip that instance for eligible fallback attempts until its cooldown or recovery criteria are satisfied and MUST record the skip reason in provider attempt evidence

#### Scenario: Disabled instance cannot be selected by new profiles
- **WHEN** an operator disables a provider instance
- **THEN** the system MUST prevent newly saved profile configurations from selecting that instance as an active execution target

#### Scenario: Authentication failure marks provider unavailable
- **WHEN** a provider request returns an authentication or authorization failure
- **THEN** the system MUST record the failure reason, mark the provider instance unavailable, and allow the current operation to continue with later configured fallback providers when policy permits

#### Scenario: Unsupported provider stage is not attempted
- **WHEN** a metadata operation plan contains a provider that no longer supports the requested stage because of capability changes or migration
- **THEN** the system MUST skip that provider attempt with an unsupported capability reason instead of calling the provider

### Requirement: Custom provider definitions use the same instance model
The system SHALL allow future custom or non-TMDB provider definitions to be represented through the same provider-instance management model.

#### Scenario: Custom provider instance is registered
- **WHEN** an operator creates a supported custom metadata provider instance
- **THEN** the system MUST store and surface it through the same instance administration and profile-selection model used for TMDB-based instances
