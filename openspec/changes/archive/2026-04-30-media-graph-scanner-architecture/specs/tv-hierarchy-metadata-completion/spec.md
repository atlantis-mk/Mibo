## ADDED Requirements

### Requirement: TV hierarchy roots are scanner-identity stable
The system SHALL create and update TV hierarchy descendants under a stable series root derived from scanner, sidecar, provider, or manual identity rather than from per-file title inference alone.

#### Scenario: Files in one TV directory have inconsistent series titles
- **WHEN** multiple files in the same TV work directory resolve to episode slots but expose different filename title prefixes
- **THEN** the system MUST create or reuse one series root and place the episode descendants under that root

#### Scenario: Provider sync enriches scanner-created hierarchy
- **WHEN** a scanner-created series root is later matched to a TV metadata provider
- **THEN** provider season and episode metadata MUST enrich descendants under the existing series root instead of creating a second provider-only hierarchy

### Requirement: Episode metadata sync preserves local scanner slots
The system SHALL preserve local episode slots created by scanner identity when provider hierarchy sync cannot find an exact provider descendant.

#### Scenario: Provider lacks local episode slot
- **WHEN** a local episode exists under a scanner-created series but the matched provider hierarchy does not contain that season and episode number
- **THEN** the system MUST preserve the local episode and surface the mismatch for governance review instead of linking it to an unrelated provider episode or deleting it
