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
The system SHALL store metadata source evidence separately from canonical field values and SHALL route metadata operation writes through field ownership policy so automated metadata refreshes cannot overwrite explicitly locked fields.

#### Scenario: Refetch respects locked fields
- **WHEN** a user locks a canonical field such as title or overview and a metadata refetch later retrieves new provider data
- **THEN** the system MUST update source evidence and any unlocked canonical fields, MUST NOT overwrite the locked field value, and MUST report the locked field as skipped in the metadata operation result

#### Scenario: Applied field records source attribution
- **WHEN** a metadata operation applies a provider or local evidence value to a canonical field
- **THEN** the corresponding field state MUST identify the source evidence or operation provenance that supplied the value when that source is available

### Requirement: Governance states distinguish automatic, review, unmatched, and manual outcomes
The system SHALL assign governance status through metadata operation decisions based on confidence, classifier decisions, match outcome, local evidence policy, metadata evidence, and user actions, including matched, needs-review, unmatched, manual, and locked-equivalent states needed to explain why a catalog item or asset relationship is or is not automatically governed.

#### Scenario: Low-confidence match enters review instead of silent acceptance
- **WHEN** metadata matching yields ambiguous provider candidates or conflicting numbering evidence
- **THEN** the system MUST mark the item with a review-oriented governance status and retain the evidence needed for a user to confirm, reject, or lock the canonical result

#### Scenario: Low-confidence classification enters review instead of silent acceptance
- **WHEN** scanner classification yields ambiguous movie, episode, attachment, version, or independent-work candidates
- **THEN** the system MUST mark the affected item, asset relationship, or candidate group with review-oriented governance status and retain classifier evidence, alternatives, confidence, and proposed actions

#### Scenario: Manual apply records manual outcome
- **WHEN** a user manually applies a metadata candidate
- **THEN** the system MUST mark the item as manually governed or equivalent and record the operation evidence needed to explain the selected candidate and field changes

#### Scenario: No candidate records unmatched outcome
- **WHEN** an automated match operation completes all eligible search or local evidence attempts without a usable candidate
- **THEN** the system MUST mark the item unmatched and retain attempt evidence explaining why no metadata was applied

### Requirement: Governance exposes image and asset relationship management
The system SHALL expose image candidates, selected images, linked assets, asset-link context, and classification decision context as part of the governance workspace for catalog items and reviewable scan groups.

#### Scenario: Governance workspace shows images and linked assets
- **WHEN** a client requests governance details for a catalog item
- **THEN** the response MUST include canonical field states, source evidence, external identities, selected and candidate images, and linked assets with enough relationship metadata to explain playability and item-to-asset linkage

#### Scenario: Governance workspace shows classification alternatives
- **WHEN** a client requests review details for an ambiguous classification decision
- **THEN** the response MUST include the candidate roles, candidate semantic types, confidence values, evidence reasons, affected files, and correction actions needed to resolve the decision

### Requirement: Governance records profile-aware metadata provenance
The system SHALL record the effective metadata profile and selected provider instance provenance for automated metadata writes so governance can explain how a library-specific strategy produced the current canonical fields.

#### Scenario: Match writes include profile provenance
- **WHEN** a metadata match applies catalog fields through a library-bound profile
- **THEN** the resulting source evidence MUST identify the effective metadata profile and the provider instance that supplied the selected metadata payload

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

### Requirement: Metadata governance preserves MetaTube provenance
The system SHALL preserve MetaTube provider-instance provenance and provider-specific external identities when applying metadata from MetaTube.

#### Scenario: MetaTube metadata source is recorded
- **WHEN** metadata from a MetaTube provider instance is applied to a catalog item
- **THEN** metadata source evidence MUST identify source type provider, source name `metatube`, the executing provider instance ID and name, the upstream MetaTube provider, the upstream item ID, and the fallback summary used by execution

#### Scenario: MetaTube identity is distinct from TMDB identity
- **WHEN** a catalog item has metadata identities from both TMDB and MetaTube
- **THEN** governance reads and refetch logic MUST keep the identities distinct and MUST NOT treat a MetaTube upstream ID as a TMDB ID

### Requirement: Governance corrections can create classification rules
The system SHALL allow user-approved classification corrections to persist source-scoped rules that future scans can use as classifier evidence.

#### Scenario: User resolves files as an episode sequence
- **WHEN** a user confirms that an ambiguous group should be treated as a named series season with sorted or explicit episode numbering
- **THEN** governance SHALL persist a source-scoped classification rule and SHALL update the affected catalog projection or asset relationships according to the confirmed decision

#### Scenario: User resolves files as movie versions
- **WHEN** a user confirms that sibling files are versions of one movie
- **THEN** governance SHALL persist a source-scoped classification rule and SHALL link the files as assets or versions of the confirmed movie work

#### Scenario: User resolves files as independent movies
- **WHEN** a user confirms that sibling files are independent movies rather than a movie version group or episode sequence
- **THEN** governance SHALL persist a source-scoped classification rule or decision record that prevents future scans from merging those files into one work

### Requirement: Governance applies rich provider fields through ownership policy
The system SHALL apply rich provider metadata fields through the same field ownership and provenance policy used for baseline metadata fields.

#### Scenario: Automated enrichment sees unlocked rich fields
- **WHEN** an automated TMDB match or refetch retrieves community rating, official rating, series status, last air date, or other supported rich canonical fields for an unlocked catalog item
- **THEN** the system MUST apply the values with metadata source attribution and include the fields in operation applied-field evidence

#### Scenario: Automated enrichment sees locked rich fields
- **WHEN** an automated TMDB match or refetch retrieves a rich field whose catalog field state is locked
- **THEN** the system MUST NOT overwrite the canonical value and MUST include the skipped field in operation evidence

### Requirement: Governance records provider tag provenance
The system SHALL preserve provenance for provider-sourced tag links so automated tag sync can be explained and bounded.

#### Scenario: TMDB sync writes tags
- **WHEN** a TMDB metadata operation links genre or keyword tags to a catalog item
- **THEN** governance evidence MUST be able to identify that those tag links came from the TMDB metadata source for that operation

#### Scenario: User tags coexist with provider tags
- **WHEN** a catalog item has user, scanner, or local tags and a TMDB operation synchronizes provider tags
- **THEN** the provider operation MUST NOT remove unrelated non-provider tag links
