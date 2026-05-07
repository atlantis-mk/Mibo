## MODIFIED Requirements

### Requirement: Scanner builds media graph candidates before catalog writes
The system SHALL group scanned files, sibling directories, current-directory siblings, sidecars, indexed filename-derived signals, cached directory summaries, content-shape plans, path-tree work-group plans, and learned classification rules into media graph candidates before writing catalog items, and SHALL treat directory shape as evidence rather than a final semantic type.

#### Scenario: Directory contains multiple episode-like videos
- **WHEN** a source directory contains multiple likely main videos that resolve to explicit or inferred episode slots
- **THEN** the scanner MUST create a single series candidate for that directory before projecting episode catalog descendants

#### Scenario: Movie folder contains multiple main-like files
- **WHEN** a source directory contains multiple plausible main video files for the same movie work
- **THEN** the scanner MUST create one movie candidate with multiple asset or version candidates instead of creating one movie per file

#### Scenario: Directory contains independent movies
- **WHEN** a source directory contains multiple likely main videos with distinct movie-like title or year evidence and no episode-sequence evidence
- **THEN** the scanner MUST preserve separate movie candidates instead of forcing the directory into one movie, one series, or one mixed semantic type

#### Scenario: Filename signals include release metadata
- **WHEN** scanned files include filename-derived release hints such as quality, source, codec, audio, subtitle, edition, or release group
- **THEN** the scanner MUST use those hints as candidate evidence for grouping and title cleanup without treating them as authoritative technical facts

#### Scenario: Sibling directories contain versions of one movie
- **WHEN** sibling child directories each contain one main video and those videos share a normalized title/year work key with release-hint differences
- **THEN** the scanner MUST create one movie candidate with multiple source assets before catalog projection

#### Scenario: Parent directory contains independent one-file movie folders
- **WHEN** sibling child directories each contain one main video with distinct title/year work keys
- **THEN** the scanner MUST preserve separate movie candidates instead of merging the parent into one work

### Requirement: Resolver decisions expose evidence and confidence
The system SHALL represent scanner grouping and classification as resolver decisions with candidate type, role, confidence, alternatives, filename-derived signal evidence, directory summary evidence, content-shape evidence, work-group evidence, review state, affected files, and reason text.

#### Scenario: Series candidate is inferred from a flat episode folder
- **WHEN** a resolver groups a flat source-first folder into a series candidate
- **THEN** the decision MUST include the target series identity, inferred season and episode slots when available, confidence, evidence references, alternatives considered, and a reason explaining the grouping

#### Scenario: Classification is ambiguous
- **WHEN** the scanner cannot confidently distinguish movie, episode, version, independent work, or attachment semantics
- **THEN** the resolver decision MUST preserve candidate evidence and mark the projected catalog item or relationship for governance review instead of silently creating unrelated works

#### Scenario: Attachment is detected
- **WHEN** a video file is classified as trailer, extra, sample, preview, or another non-main role
- **THEN** the resolver decision MUST expose the attachment role and evidence so catalog projection can link it to a likely parent work without treating it as a standalone movie or episode

#### Scenario: Audio token prevents episode false positive
- **WHEN** a resolver rejects weak episode inference because a numeric-looking token is classified as filename-derived audio evidence
- **THEN** the decision MUST expose that anti-misclassification evidence in its reason or evidence summary

#### Scenario: Work group overrides leaf materialization
- **WHEN** a parent work-group plan groups files from multiple child directories into one movie or series candidate
- **THEN** the resolver decision MUST expose the parent work-group evidence and the leaf alternatives that were superseded
