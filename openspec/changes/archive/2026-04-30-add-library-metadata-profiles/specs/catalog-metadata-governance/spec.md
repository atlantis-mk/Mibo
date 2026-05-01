## ADDED Requirements

### Requirement: Governance records profile-aware metadata provenance
The system SHALL record the effective metadata profile and selected provider instance provenance for automated metadata writes so governance can explain how a library-specific strategy produced the current canonical fields.

#### Scenario: Match writes include profile provenance
- **WHEN** a metadata match applies catalog fields through a library-bound profile
- **THEN** the resulting source evidence MUST identify the effective metadata profile and the provider instance that supplied the selected metadata payload

### Requirement: Governance retains fallback attempt outcomes
The system SHALL retain enough evidence to explain when profile execution falls back from one configured provider instance or stage source to another.

#### Scenario: Fallback provider wins the final match
- **WHEN** the first eligible provider instance in a profile cannot return a usable result and a later configured provider instance supplies the final metadata detail
- **THEN** governance evidence MUST retain the final provider instance and a summary that the operation completed through configured fallback rather than first-choice execution

### Requirement: Refetch prefers the original provider identity within profile rules
The system SHALL attempt refetch operations using the existing provider identity and provider instance provenance recorded for the catalog item before considering other profile-allowed fallbacks.

#### Scenario: Refetch preserves original provider source
- **WHEN** a catalog item already has a persisted provider identity and matching provider instance provenance from a prior automated metadata operation
- **THEN** a refetch MUST retry that provider path first and only use another profile-allowed fallback if the original path is unavailable or no longer permitted by the effective profile
