## MODIFIED Requirements

### Requirement: Governance states distinguish automatic, review, unmatched, and manual outcomes
The system SHALL assign governance status based on confidence, classifier decisions, metadata evidence, and user actions, including matched, needs-review, unmatched, manual, and locked-equivalent states needed to explain why a catalog item or asset relationship is or is not automatically governed.

#### Scenario: Low-confidence match enters review instead of silent acceptance
- **WHEN** metadata matching yields ambiguous provider candidates or conflicting numbering evidence
- **THEN** the system MUST mark the item with a review-oriented governance status and retain the evidence needed for a user to confirm, reject, or lock the canonical result

#### Scenario: Low-confidence classification enters review instead of silent acceptance
- **WHEN** scanner classification yields ambiguous movie, episode, attachment, version, or independent-work candidates
- **THEN** the system MUST mark the affected item, asset relationship, or candidate group with review-oriented governance status and retain classifier evidence, alternatives, confidence, and proposed actions

### Requirement: Governance exposes image and asset relationship management
The system SHALL expose image candidates, selected images, linked assets, asset-link context, and classification decision context as part of the governance workspace for catalog items and reviewable scan groups.

#### Scenario: Governance workspace shows images and linked assets
- **WHEN** a client requests governance details for a catalog item
- **THEN** the response MUST include canonical field states, source evidence, external identities, selected and candidate images, and linked assets with enough relationship metadata to explain playability and item-to-asset linkage

#### Scenario: Governance workspace shows classification alternatives
- **WHEN** a client requests review details for an ambiguous classification decision
- **THEN** the response MUST include the candidate roles, candidate semantic types, confidence values, evidence reasons, affected files, and correction actions needed to resolve the decision

## ADDED Requirements

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
