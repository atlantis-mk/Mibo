## MODIFIED Requirements

### Requirement: Governance states distinguish automatic, review, unmatched, and manual outcomes
The system SHALL assign governance status through metadata operation decisions and scanner recognition decisions based on confidence, classifier decisions, work-group decisions, match outcome, local evidence policy, metadata evidence, and user actions, including matched, needs-review, unmatched, manual, and locked-equivalent states needed to explain why a catalog item, work group, or asset relationship is or is not automatically governed.

#### Scenario: Low-confidence match enters review instead of silent acceptance
- **WHEN** metadata matching yields ambiguous provider candidates or conflicting numbering evidence
- **THEN** the system MUST mark the item with a review-oriented governance status and retain the evidence needed for a user to confirm, reject, or lock the canonical result

#### Scenario: Low-confidence classification enters review instead of silent acceptance
- **WHEN** scanner classification yields ambiguous movie, episode, attachment, version, or independent-work candidates
- **THEN** the system MUST mark the affected item, asset relationship, or candidate group with review-oriented governance status and retain classifier evidence, alternatives, confidence, and proposed actions

#### Scenario: Low-confidence work group enters review
- **WHEN** path-tree work-group recognition yields ambiguous movie-version, movie-collection, series, or episode-pack candidates
- **THEN** the system MUST mark the affected work group or guarded placeholder with review-oriented governance status and retain work-group evidence, alternatives, confidence, affected files, and proposed correction actions

#### Scenario: Manual apply records manual outcome
- **WHEN** a user manually applies a metadata candidate
- **THEN** the system MUST mark the item as manually governed or equivalent and record the operation evidence needed to explain the selected candidate and field changes

#### Scenario: No candidate records unmatched outcome
- **WHEN** an automated match operation completes all eligible search or local evidence attempts without a usable candidate
- **THEN** the system MUST mark the item unmatched and retain attempt evidence explaining why no metadata was applied

### Requirement: Governance corrections can create classification rules
The system SHALL allow user-approved classification corrections to persist source-scoped rules that future scans can use as classifier or work-group evidence.

#### Scenario: User resolves files as an episode sequence
- **WHEN** a user confirms that an ambiguous group should be treated as a named series season with sorted or explicit episode numbering
- **THEN** governance SHALL persist a source-scoped classification rule and SHALL update the affected catalog projection or asset relationships according to the confirmed decision

#### Scenario: User resolves files as movie versions
- **WHEN** a user confirms that sibling files or sibling directories are versions of one movie
- **THEN** governance SHALL persist a source-scoped classification rule and SHALL link the files as assets or versions of the confirmed movie work

#### Scenario: User resolves files as independent movies
- **WHEN** a user confirms that sibling files or sibling directories are independent movies rather than a movie version group or episode sequence
- **THEN** governance SHALL persist a source-scoped classification rule or decision record that prevents future scans from merging those files into one work

#### Scenario: User resolves parent as movie collection
- **WHEN** a user confirms that a parent directory is a movie collection
- **THEN** governance SHALL persist a source-scoped work-group rule that future scans use to split children by title/year work key
