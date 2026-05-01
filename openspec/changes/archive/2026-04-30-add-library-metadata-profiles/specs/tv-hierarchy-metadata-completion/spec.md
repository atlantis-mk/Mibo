## ADDED Requirements

### Requirement: TV hierarchy synchronization resolves the effective library metadata profile
The system SHALL resolve the effective library metadata profile before executing rooted TV metadata synchronization for a series, season, or episode initiated action.

#### Scenario: Episode-triggered match uses the library profile
- **WHEN** a user triggers metadata matching from an episode that belongs to a library with a bound metadata profile
- **THEN** the system MUST resolve that library profile and use its configured search, detail, hierarchy, and fallback behavior while still rooting the operation at the series catalog item

### Requirement: TV descendant identities remain provider-normalized across profile stages
The system SHALL preserve season and episode descendant identity and evidence semantics even when TV metadata stages are supplied by a profile-selected provider instance rather than a single global provider configuration.

#### Scenario: Profile-selected provider sync populates descendants
- **WHEN** a TV metadata profile selects a provider instance that returns normalized season and episode hierarchy detail
- **THEN** the system MUST persist descendant identities, evidence, and artwork candidates for the generated or updated seasons and episodes using the same durable catalog hierarchy rules as the existing rooted sync flow

### Requirement: TV profile fallback does not bypass hierarchy mismatch safeguards
The system SHALL preserve existing hierarchy mismatch protections when a TV metadata profile falls back between configured provider instances.

#### Scenario: Fallback provider lacks the local episode slot
- **WHEN** a profile falls back to another provider instance and that provider's hierarchy does not contain the local episode's expected season or episode slot
- **THEN** the system MUST preserve the local descendant, surface the mismatch for governance review, and MUST NOT silently link the episode to an unrelated provider slot
