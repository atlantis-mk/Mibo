## MODIFIED Requirements

### Requirement: Governance states distinguish automatic, review, unmatched, and manual outcomes
The system SHALL assign governance status through metadata operation and recognition resolver decisions based on confidence, classifier decisions, resolver conflicts, match outcome, local evidence policy, metadata evidence, and user actions, including matched, needs-review, unmatched, manual, and locked-equivalent states needed to explain why a metadata item, resource relationship, or recognition candidate is or is not automatically governed.

#### Scenario: Low-confidence match enters review instead of silent acceptance
- **WHEN** metadata matching yields ambiguous provider candidates or conflicting numbering evidence
- **THEN** the system MUST mark the item with a review-oriented governance status and retain the evidence needed for a user to confirm, reject, or lock the canonical result

#### Scenario: Low-confidence classification enters review instead of silent acceptance
- **WHEN** scanner recognition yields ambiguous movie, episode, attachment, version, edition, or independent-work candidates
- **THEN** the system MUST mark the affected metadata item, resource relationship, or recognition candidate group with review-oriented governance status and retain resolver evidence, alternatives, confidence, conflicts, and proposed actions

#### Scenario: Manual apply records manual outcome
- **WHEN** a user manually applies a metadata or recognition candidate
- **THEN** the system MUST mark the item, relationship, or resolver rule as manually governed or equivalent and record the operation evidence needed to explain the selected candidate and field changes

#### Scenario: No candidate records unmatched outcome
- **WHEN** an automated match or recognition operation completes all eligible search, resolver, or local evidence attempts without a usable candidate
- **THEN** the system MUST mark the item or candidate unmatched and retain attempt evidence explaining why no metadata or relationship was applied

### Requirement: Governance exposes image and asset relationship management
The system SHALL expose image candidates, selected images, linked resources, resource-link context, and recognition resolver decision context as part of the governance workspace for metadata items and reviewable recognition groups.

#### Scenario: Governance workspace shows images and linked assets
- **WHEN** a client requests governance details for a metadata item
- **THEN** the response MUST include canonical field states, source evidence, external identities, selected and candidate images, and linked resources with enough relationship metadata to explain playability and item-to-resource linkage

#### Scenario: Governance workspace shows classification alternatives
- **WHEN** a client requests review details for an ambiguous recognition decision
- **THEN** the response MUST include the candidate roles, candidate semantic types, confidence values, conflict reasons, evidence reasons, affected files, and correction actions needed to resolve the decision

### Requirement: Governance corrections can create classification rules
The system SHALL allow user-approved recognition corrections to persist source-scoped resolver rules that future scans can use as high-priority evidence.

#### Scenario: User resolves files as an episode sequence
- **WHEN** a user confirms that an ambiguous group should be treated as a named series season with sorted or explicit episode numbering
- **THEN** governance SHALL persist a source-scoped resolver rule and SHALL update the affected metadata projection or resource relationships according to the confirmed decision

#### Scenario: User resolves files as movie versions
- **WHEN** a user confirms that sibling files are versions or editions of one movie
- **THEN** governance SHALL persist a source-scoped resolver rule and SHALL link the files as resources of the confirmed movie work through resolver materialization

#### Scenario: User resolves files as independent movies
- **WHEN** a user confirms that sibling files are independent movies rather than a movie version group or episode sequence
- **THEN** governance SHALL persist a source-scoped resolver rule or decision record that prevents future scans from merging those files into one work

## ADDED Requirements

### Requirement: Governance removes obsolete scanner decision dependencies
The system SHALL expose and modify recognition resolver decisions rather than depending on legacy content-shape, path-tree, sibling-match, or per-file catalog scan decision records as authoritative governance state.

#### Scenario: Governance loads a recognition review group
- **WHEN** a client requests review data for scanner-created ambiguity
- **THEN** the response MUST be backed by resolver manifest candidates and decisions rather than obsolete scanner-specific final decision records

#### Scenario: Cleanup removes old decision source
- **WHEN** implementation removes a legacy scanner final-decision path
- **THEN** governance tests and handlers MUST be updated to read equivalent resolver evidence or the obsolete governance surface MUST be removed
