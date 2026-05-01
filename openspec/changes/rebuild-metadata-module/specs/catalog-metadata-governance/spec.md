## MODIFIED Requirements

### Requirement: Metadata writes preserve source evidence and field locks
The system SHALL store metadata source evidence separately from canonical field values and SHALL route metadata operation writes through field ownership policy so automated metadata refreshes cannot overwrite explicitly locked fields.

#### Scenario: Refetch respects locked fields
- **WHEN** a user locks a canonical field such as title or overview and a metadata refetch later retrieves new provider data
- **THEN** the system MUST update source evidence and any unlocked canonical fields, MUST NOT overwrite the locked field value, and MUST report the locked field as skipped in the metadata operation result

#### Scenario: Applied field records source attribution
- **WHEN** a metadata operation applies a provider or local evidence value to a canonical field
- **THEN** the corresponding field state MUST identify the source evidence or operation provenance that supplied the value when that source is available

### Requirement: Governance states distinguish automatic, review, unmatched, and manual outcomes
The system SHALL assign governance status through metadata operation decisions based on confidence, match outcome, local evidence policy, and user actions, including matched, needs-review, unmatched, manual, and locked-equivalent states needed to explain why a catalog item is or is not automatically governed.

#### Scenario: Low-confidence match enters review instead of silent acceptance
- **WHEN** metadata matching yields ambiguous provider candidates or conflicting numbering evidence
- **THEN** the system MUST mark the item with a review-oriented governance status and retain the evidence needed for a user to confirm, reject, or lock the canonical result

#### Scenario: Manual apply records manual outcome
- **WHEN** a user manually applies a metadata candidate
- **THEN** the system MUST mark the item as manually governed or equivalent and record the operation evidence needed to explain the selected candidate and field changes

#### Scenario: No candidate records unmatched outcome
- **WHEN** an automated match operation completes all eligible search or local evidence attempts without a usable candidate
- **THEN** the system MUST mark the item unmatched and retain attempt evidence explaining why no metadata was applied

### Requirement: Governance retains fallback attempt outcomes
The system SHALL retain enough operation evidence to explain when metadata execution attempts, skips, fails, or falls back between configured provider instances or local evidence sources.

#### Scenario: Fallback provider wins the final match
- **WHEN** the first eligible provider instance in a profile cannot return a usable result and a later configured provider instance supplies the final metadata detail
- **THEN** governance evidence MUST retain all relevant attempt outcomes, the final provider instance, and a summary that the operation completed through configured fallback rather than first-choice execution

#### Scenario: Local evidence seeds remote detail
- **WHEN** sidecar evidence supplies a provider external ID and the operation uses that ID to fetch remote detail without search
- **THEN** governance evidence MUST record both the scanner evidence source and the provider detail source that produced the canonical fields

### Requirement: Refetch prefers the original provider identity within profile rules
The system SHALL attempt refetch operations using the existing provider identity and provider instance provenance recorded for the catalog item before considering other profile-allowed fallbacks, and SHALL report the selected path through the unified metadata operation result.

#### Scenario: Refetch preserves original provider source
- **WHEN** a catalog item already has a persisted provider identity and matching provider instance provenance from a prior automated metadata operation
- **THEN** a refetch MUST retry that provider path first and only use another profile-allowed fallback if the original path is unavailable or no longer permitted by the effective profile

#### Scenario: Refetch reports missing identity
- **WHEN** a refetch operation cannot find an applicable provider identity or local evidence source for the target item
- **THEN** the system MUST fail or skip the operation with a clear result status instead of performing an unrelated search silently
