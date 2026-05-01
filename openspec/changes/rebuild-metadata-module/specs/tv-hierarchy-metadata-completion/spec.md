## MODIFIED Requirements

### Requirement: TV hierarchy synchronization resolves the effective library metadata profile
The system SHALL resolve the effective library metadata strategy and metadata operation execution plan before executing rooted TV metadata synchronization for a series, season, or episode initiated action.

#### Scenario: Episode-triggered match uses the library profile
- **WHEN** a user triggers metadata matching from an episode that belongs to a library with a bound metadata profile or copied strategy
- **THEN** the system MUST resolve that library strategy and use its configured search, detail, hierarchy, local evidence, and fallback behavior while still rooting the operation at the series catalog item

#### Scenario: Automated episode scan queues series operation
- **WHEN** scanning an episode queues metadata enrichment
- **THEN** the resulting metadata operation MUST target the series root and report descendant status for the originating episode when operation results are requested

### Requirement: TV descendant identities remain provider-normalized across profile stages
The system SHALL preserve season and episode descendant identity and evidence semantics when TV metadata stages are supplied by a strategy-selected provider instance and normalized through the metadata operation pipeline.

#### Scenario: Profile-selected provider sync populates descendants
- **WHEN** a TV metadata strategy selects a provider instance that returns normalized season and episode hierarchy detail
- **THEN** the system MUST persist descendant identities, evidence, and artwork candidates for the generated or updated seasons and episodes using the same durable catalog hierarchy rules as the existing rooted sync flow

#### Scenario: Hierarchy stage records provider attempts
- **WHEN** TV hierarchy synchronization attempts one or more hierarchy-capable provider instances
- **THEN** the metadata operation evidence MUST record hierarchy-stage attempt outcomes and the provider instance that supplied descendant data

### Requirement: TV profile fallback does not bypass hierarchy mismatch safeguards
The system SHALL preserve existing hierarchy mismatch protections when a TV metadata operation falls back between configured provider instances or local evidence sources.

#### Scenario: Fallback provider lacks the local episode slot
- **WHEN** a profile falls back to another provider instance and that provider's hierarchy does not contain the local episode's expected season or episode slot
- **THEN** the system MUST preserve the local descendant, surface the mismatch for governance review, and MUST NOT silently link the episode to an unrelated provider slot

#### Scenario: Local slot mismatch appears in operation result
- **WHEN** a rooted TV metadata operation cannot match the originating episode to a provider descendant
- **THEN** the operation result MUST identify the mismatch status for the originating item while preserving the local item and its asset links
