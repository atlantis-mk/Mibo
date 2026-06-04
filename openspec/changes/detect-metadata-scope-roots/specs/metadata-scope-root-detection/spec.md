## ADDED Requirements

### Requirement: Metadata scope roots are selected by upward reduction
The system SHALL determine metadata scope roots by starting from changed leaf directory summaries, walking upward through a bounded ancestor range, and selecting the highest directory that remains complete, pure, explainable, and bounded by a classification or library boundary.

#### Scenario: Leaf cluster expands to a parent scope
- **WHEN** a leaf directory is classified and its parent contains sibling leaf summaries for the same metadata identity
- **THEN** the system MUST evaluate the parent as a candidate metadata scope root before materializing the leaf independently

#### Scenario: Boundary stops upward traversal
- **WHEN** an ancestor directory contains multiple unrelated metadata identities, matches a known source/category/library boundary, or no longer improves coverage for the dominant identity
- **THEN** the system MUST stop upward selection before that boundary and choose the best lower candidate

#### Scenario: Bounded traversal avoids unbounded scans
- **WHEN** scope detection walks ancestors from a leaf directory
- **THEN** it MUST inspect no more than the configured maximum ancestor depth or the library root, whichever comes first

### Requirement: Scope decisions describe root kind and layout separately from leaf shape
The system SHALL persist metadata scope decisions as semantic root decisions with root kind, layout, child roles, confidence, evidence, and covered files, rather than relying only on a leaf directory shape string.

#### Scenario: Versioned episode packs create a series scope
- **WHEN** sibling leaf directories are episode packs for the same series and season with highly overlapping episode sets and distinct version signatures
- **THEN** the system MUST persist a scope decision with `root_kind` of `series`, a versioned episode-pack layout, version child roles, and covered episode resources

#### Scenario: Season directories create a series scope
- **WHEN** sibling leaf or child summaries represent complementary season directories for the same series identity
- **THEN** the system MUST persist a scope decision with `root_kind` of `series` and a season-directory layout

#### Scenario: Split episode packs create one series scope
- **WHEN** sibling leaf directories represent complementary episode ranges for the same series and season without version overlap
- **THEN** the system MUST persist one series scope with a split episode-pack layout instead of separate series scopes

#### Scenario: Movie version directories create one movie scope
- **WHEN** sibling leaf directories represent the same movie identity with distinct version or edition signatures
- **THEN** the system MUST persist one movie scope with version-directory child roles

#### Scenario: Collections remain multi-identity scopes
- **WHEN** a candidate directory contains multiple clear movie identities and no single identity dominates
- **THEN** the system MUST classify the scope as a movie collection layout or review-required mixed scope according to confidence and conflict evidence

### Requirement: Scope scoring uses purity, coverage, layout, boundary, and attachments
The system SHALL score candidate scope roots using identity purity, coverage gain, layout explainability, boundary evidence, and attachment neutrality.

#### Scenario: Identity purity rejects mixed parents
- **WHEN** a parent candidate contains unrelated movie and series identities that cannot be explained as one collection or one attachment set
- **THEN** the system MUST reject that parent as a single-work scope and select a lower scope or require review

#### Scenario: Coverage gain promotes version parent
- **WHEN** a parent candidate adds sibling versions for the same episode set without adding unrelated identities
- **THEN** the system MUST prefer the parent candidate over the individual version leaf directory

#### Scenario: Directory title boundary reinforces scope root
- **WHEN** a candidate directory name matches the dominant metadata identity and its parent looks like a source, share, category, or library boundary
- **THEN** the system MUST increase confidence that the candidate directory is the metadata scope root

#### Scenario: Attachments do not reduce purity
- **WHEN** a candidate contains attachment child folders such as trailers, samples, extras, featurettes, interviews, or behind-the-scenes media beside a valid main identity
- **THEN** the system MUST exclude those attachment children from main identity purity scoring while preserving them as attachments in the scope decision

### Requirement: Scope claims prevent duplicate downstream processing
The system SHALL claim covered files for an accepted metadata scope so recognition, materialization, and directory metadata resolution run once per scope instead of once per file or leaf directory.

#### Scenario: Accepted scope suppresses leaf materialization
- **WHEN** a scope decision covers multiple leaf directories
- **THEN** the system MUST generate downstream recognition/materialization work for the scope and MUST NOT independently materialize each covered leaf as a separate metadata root

#### Scenario: Repeated file scan reuses existing claim
- **WHEN** a later scan observes a file already covered by an unchanged scope decision fingerprint
- **THEN** the system MUST reuse the scope claim and avoid enqueueing duplicate recognition/materialization work

#### Scenario: Scope change invalidates claim
- **WHEN** a covered file is added, removed, changes identity signal, or moves between child roles
- **THEN** the system MUST update or invalidate the affected scope claim and regenerate dependent recognition/materialization outputs

### Requirement: Scope decisions drive materialization and metadata resolution
The system SHALL use accepted metadata scope decisions as the authoritative input for recognition units, resource grouping, metadata item materialization, and directory metadata resolution.

#### Scenario: Versioned series materializes one hierarchy
- **WHEN** a versioned episode-pack scope is accepted for a series
- **THEN** materialization MUST create or reuse one series hierarchy and link matching episode versions as resources under the same episode metadata items

#### Scenario: Movie version scope resolves one work
- **WHEN** a movie version scope is accepted
- **THEN** directory metadata resolution MUST target one movie work and bind all version resources to that work with version or edition evidence

#### Scenario: Attachment-only orphan requires review
- **WHEN** a scope candidate contains only attachment groups and no compatible parent main scope is found
- **THEN** the system MUST produce a review-required attachment-orphan outcome and MUST NOT create speculative movie or series metadata

### Requirement: Obsolete final-root inference paths are removed after scope parity
The system SHALL remove or disable implementation paths that infer final metadata roots from single-directory leaf plans or residual directory grouping once scope root detection provides equivalent tested behavior.

#### Scenario: No downstream final root from leaf shape alone
- **WHEN** scope root detection is enabled and a scope decision exists for a leaf
- **THEN** downstream recognition and metadata resolution MUST use the scope decision instead of treating the leaf `content_shape` plan as the final metadata root

#### Scenario: Residual grouping is not a competing authority
- **WHEN** scope root detection covers movie versions, multipart movies, episode identities, and mixed review cases with tests
- **THEN** residual directory reduction MUST be removed or limited to diagnostic evidence and MUST NOT create competing materialization decisions

#### Scenario: Version bump refreshes stale derived data
- **WHEN** the scope classifier version, leaf classifier version, or scope fingerprint inputs change
- **THEN** stale scope decisions, claims, recognition units, and directory metadata resolution payloads MUST be regenerated
