## MODIFIED Requirements

### Requirement: TV hierarchy roots are scanner-identity stable
The system SHALL create and update TV hierarchy descendants under a stable series root derived from media graph identity, sidecar identity, provider identity, or manual identity rather than from per-file title inference alone.

#### Scenario: Files in one TV directory have inconsistent series titles
- **WHEN** multiple files in the same TV work directory resolve to episode slots but expose different filename title prefixes
- **THEN** the system MUST create or reuse one series root from the media graph group identity and place the episode descendants under that root

#### Scenario: Provider sync enriches scanner-created hierarchy
- **WHEN** a scanner-created series root is later matched to a TV metadata provider
- **THEN** provider season and episode metadata MUST enrich descendants under the existing series root instead of creating a second provider-only hierarchy

#### Scenario: Graph decision creates TV hierarchy
- **WHEN** media graph recognition accepts a series, season, and episode decision for local files
- **THEN** the system MUST materialize non-orphan series, season, and episode metadata with parent and root relationships before projection refresh

## ADDED Requirements

### Requirement: TV hierarchy projection follows graph resource links
The system SHALL expose locally scanned TV content only when the media graph materializer has linked playable resources to episode descendants and refreshed projections for the full ancestor hierarchy.

#### Scenario: Episode resource is linked
- **WHEN** an episode resource is linked to an episode metadata item by graph materialization
- **THEN** projection refresh MUST include the episode, its season ancestor, and its series root for the affected library

#### Scenario: Episode candidate lacks accepted graph hierarchy
- **WHEN** a parser emits an episode-like signal but no graph decision accepts a series hierarchy for it
- **THEN** the system MUST NOT publish an orphan episode as a normal TV catalog item
- **AND** it MUST preserve review state or inventory-backed visibility for governance
