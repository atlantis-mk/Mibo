## MODIFIED Requirements

### Requirement: Fast classifier uses bounded sibling grouping
The system SHALL use cached current-directory summary evidence derived from scan snapshots and indexed file signals to distinguish episode sequences, movie version groups, independent movie files, and attachments without recursively scanning the full source for classification context.

#### Scenario: Siblings form an episode sequence
- **WHEN** likely main videos in the same directory have shared title evidence and consecutive episode numbers
- **THEN** the classifier SHALL group them as episode candidates for the same series and season when confidence thresholds are met

#### Scenario: Siblings look like movie versions
- **WHEN** likely main videos in the same directory share a normalized title stem and differ mainly by quality, edition, cut, container, language, or release tokens
- **THEN** the classifier SHALL group them as one movie candidate with multiple asset/version candidates

#### Scenario: Siblings look like independent movies
- **WHEN** likely main videos in the same directory have different title stems or year evidence and no episode sequence evidence
- **THEN** the classifier SHALL preserve independent movie candidates rather than merging them into one movie or one episode sequence

#### Scenario: Directory summary already exists
- **WHEN** multiple files in the same scanned directory require sibling context
- **THEN** the classifier SHALL reuse the cached directory summary for that scan snapshot rather than recomputing sibling evidence per file or issuing additional storage listings

#### Scenario: Indexed file signals already exist
- **WHEN** multiple files in the same scanned directory have current indexed file signals
- **THEN** the classifier SHALL build sibling grouping evidence from those signals and SHALL NOT reparse each filename for every materialization batch

### Requirement: Classification outcomes are thresholded
The system SHALL classify fast decisions into confirmed, provisional, or review-required states based on configured confidence thresholds and candidate margins, and SHALL avoid silently committing unrelated catalog semantics for low-confidence directory-level outcomes.

#### Scenario: Candidate is high confidence
- **WHEN** one candidate exceeds the fast-confirmed confidence threshold and no close conflicting candidate exists
- **THEN** the system SHALL allow catalog projection using that candidate while retaining its evidence

#### Scenario: Candidate is medium confidence
- **WHEN** a candidate is plausible but below the confirmed threshold
- **THEN** the system SHALL create a provisional decision that can be validated asynchronously without blocking inventory persistence

#### Scenario: Candidate is low confidence or conflicting
- **WHEN** no candidate meets the minimum confidence threshold or movie and episode alternatives are too close
- **THEN** the system SHALL preserve inventory facts and produce a review-required decision instead of silently committing a final semantic type

#### Scenario: Directory plan is uncertain
- **WHEN** a directory-level content-shape plan is below the high-confidence threshold or explicitly review-required
- **THEN** the fast classifier SHALL preserve review evidence and SHALL NOT silently project unrelated movie or episode catalog rows as if they were confirmed semantics
