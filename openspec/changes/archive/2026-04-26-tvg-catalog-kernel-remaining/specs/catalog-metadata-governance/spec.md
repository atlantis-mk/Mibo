## ADDED Requirements

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
