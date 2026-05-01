# catalog-metadata-governance Specification

## Purpose
TBD - created by syncing change tvg-catalog-kernel-remaining. Update Purpose after archive.
## Requirements
### Requirement: TV metadata matching is rooted at the series catalog item
The system SHALL perform TV metadata matching at the `series` catalog item, persist the chosen provider identity on the series, and derive or update season and episode catalog items from provider season detail rather than matching each episode independently.

#### Scenario: Matching a partially scanned series expands the provider hierarchy
- **WHEN** a locally scanned series with one or more episode assets is matched successfully to a provider series
- **THEN** the system MUST persist the series external identity, create or update the corresponding season and episode catalog items from provider detail, and merge local episode assets into the matching season and episode positions without losing their asset links

### Requirement: Metadata writes preserve source evidence and field locks
The system SHALL store metadata source evidence separately from canonical field values and SHALL prevent automated metadata refreshes from overwriting fields that are explicitly locked by governance.

#### Scenario: Refetch respects locked fields
- **WHEN** a user locks a canonical field such as title or overview and a metadata refetch later retrieves new provider data
- **THEN** the system MUST update source evidence and any unlocked canonical fields but MUST NOT overwrite the locked field value

### Requirement: Governance states distinguish automatic, review, unmatched, and manual outcomes
The system SHALL assign governance status based on confidence and user actions, including matched, needs-review, unmatched, manual, and locked-equivalent states needed to explain why a catalog item is or is not automatically governed.

#### Scenario: Low-confidence match enters review instead of silent acceptance
- **WHEN** metadata matching yields ambiguous provider candidates or conflicting numbering evidence
- **THEN** the system MUST mark the item with a review-oriented governance status and retain the evidence needed for a user to confirm, reject, or lock the canonical result

### Requirement: Governance exposes image and asset relationship management
The system SHALL expose image candidates, selected images, linked assets, and asset-link context as part of the governance workspace for catalog items.

#### Scenario: Governance workspace shows images and linked assets
- **WHEN** a client requests governance details for a catalog item
- **THEN** the response MUST include canonical field states, source evidence, external identities, selected and candidate images, and linked assets with enough relationship metadata to explain playability and item-to-asset linkage

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

### Requirement: Metadata governance preserves MetaTube provenance
The system SHALL preserve MetaTube provider-instance provenance and provider-specific external identities when applying metadata from MetaTube.

#### Scenario: MetaTube metadata source is recorded
- **WHEN** metadata from a MetaTube provider instance is applied to a catalog item
- **THEN** metadata source evidence MUST identify source type provider, source name `metatube`, the executing provider instance ID and name, the upstream MetaTube provider, the upstream item ID, and the fallback summary used by execution

#### Scenario: MetaTube identity is distinct from TMDB identity
- **WHEN** a catalog item has metadata identities from both TMDB and MetaTube
- **THEN** governance reads and refetch logic MUST keep the identities distinct and MUST NOT treat a MetaTube upstream ID as a TMDB ID
