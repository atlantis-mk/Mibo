## ADDED Requirements

### Requirement: Title roles are explicit and non-interchangeable
The system SHALL derive scanner-managed title-like values through explicit title
roles and SHALL NOT reuse an identity, search, display, raw provenance, sort, or
file-stem title value for another role without passing through that role's
helper.

#### Scenario: Movie identity title is not used as display text
- **WHEN** scan evidence for `Movie.Title.2026.2160p.WEB-DL.mkv` is converted into a movie work key
- **THEN** the movie identity role MAY normalize the title for key comparison, but the system MUST NOT persist the normalized key text as the catalog display title

#### Scenario: Resource file title keeps the file stem
- **WHEN** a playable resource is materialized from `/library/Movie (2026)/2026.2160p.WEB-DL.SDR.DDP5.1.Atmos.H.265.mkv`
- **THEN** `file_title` MUST be `2026.2160p.WEB-DL.SDR.DDP5.1.Atmos.H.265` and MUST NOT be cleaned as a movie work title

#### Scenario: Raw work title remains provenance
- **WHEN** local resource evidence is derived from folder or file text
- **THEN** `raw_work_title` MUST keep readable provenance text and MUST NOT be used as the canonical movie work identity without applying the movie identity role

### Requirement: Directory shape controls title source selection
The system SHALL choose whether folder, file, sidecar, catalog, or fallback
title evidence is eligible according to the planned directory shape.

#### Scenario: Single movie folder can use folder title
- **WHEN** a directory is planned as a single movie folder and its direct video file has only weak release metadata as its title candidate
- **THEN** the movie work identity MUST be derived from the directory title when that directory title contains usable identity text

#### Scenario: Multipart movie folder can use folder title
- **WHEN** a directory is planned as one multipart movie with multiple direct part files
- **THEN** the system MUST derive one shared movie work identity from the directory title when file part names do not provide a stronger shared title

#### Scenario: Multi-version movie folder can use folder title
- **WHEN** a directory is planned as multiple versions or editions of the same movie
- **THEN** the system MUST derive one shared movie work identity from the directory title when file names are version or release labels

#### Scenario: Movie collection does not collapse to folder title
- **WHEN** a directory is planned as a movie collection containing multiple independent movie identities
- **THEN** each movie MUST use its own file, child directory, sidecar, catalog ID, or title-year identity and MUST NOT share the collection root folder name as one movie work identity

### Requirement: Nested directory boundaries are respected
The system SHALL apply title role and directory-shape policies per planned
directory scope: direct files follow their directory's shape, while child
directories are planned and materialized using their own shape.

#### Scenario: Child movie folder under a collection has its own title source
- **WHEN** a collection directory contains child directories that are each planned as movie folders
- **THEN** each child movie folder MUST use its own folder or file title policy and MUST NOT inherit the parent collection title as its movie work title

#### Scenario: Parent movie folder does not rewrite child directory identity
- **WHEN** a movie folder contains a child directory that is planned as extras, attachments, another movie, or review-required content
- **THEN** the child directory MUST keep its own planned behavior and MUST NOT be materialized solely by the parent movie folder title policy

### Requirement: Weak movie identities are detected before materialization
The system SHALL classify a movie identity candidate as weak when it is empty,
only a year, only release or technical tokens, or only punctuation/separators
after cleanup, and SHALL block that candidate from becoming the sole movie work
key when better allowed directory evidence exists.

#### Scenario: Release-only filename falls back to folder title
- **WHEN** `/library/Real Movie (2026)/2026.2160p.WEB-DL.mkv` is planned as a single movie folder
- **THEN** the materialized movie work key MUST be based on `Real Movie` with year `2026` instead of a year-only key

#### Scenario: Fullwidth separators are cleaned for identity
- **WHEN** `/library/废｜弛R｜升③(2026)/2026.2160p.WEB-DL.mkv` is planned as a single movie folder
- **THEN** the movie identity title MUST treat `｜` as a separator and produce usable title text from the directory name

#### Scenario: No usable title remains reviewable
- **WHEN** both the eligible file title and eligible folder title are weak for a movie-like directory
- **THEN** the system MUST avoid silently merging the directory into a shared weak movie identity and MUST mark the unit for review or use a collision-safe local fallback

### Requirement: Materialization persists titles by role
Recognition materialization SHALL write scanner-managed metadata and resource
fields from the correct title roles and SHALL preserve evidence about the source
used for each role.

#### Scenario: Local metadata item gets display and search roles
- **WHEN** a movie metadata item is created from scanner evidence before provider metadata is applied
- **THEN** `title` MUST use the local display role, `search_title` MUST use the local search role, and `sort_title` MUST be derived from the selected display/provider title by one sort policy

#### Scenario: Provider or user title is not overwritten by scanner fallback
- **WHEN** a metadata item already has provider-managed or user-governed title fields
- **THEN** scanner-derived display, search, or sort fallback values MUST NOT overwrite those governed fields unless the metadata field policy explicitly allows it

#### Scenario: Recognition evidence records title source
- **WHEN** materialization creates or updates recognition candidates, metadata items, or resources from title evidence
- **THEN** the stored evidence MUST indicate whether the chosen value came from a directory title, file title, sidecar, catalog ID, provider metadata, or fallback

### Requirement: Metadata search consumes search titles
Automatic metadata matching SHALL build provider queries and local title scores
from search-role title text and compatible provider/user titles, not from raw
file stems or movie identity key text.

#### Scenario: Search query excludes release-only file title
- **WHEN** a movie item was created from a directory title because the file name was release-only
- **THEN** automatic metadata search MUST query using the search title derived from the directory/work title and MUST NOT query using the raw `file_title`

#### Scenario: Partial title matching uses normalized search role
- **WHEN** local and remote titles are compared for automatic match scoring
- **THEN** the comparison MUST normalize both sides through the search-title policy before accepting an exact or partial title score

### Requirement: Obsolete cleanup paths are removed after migration
The system SHALL remove or make private obsolete generic title cleanup wrappers
after production call sites have moved to the explicit role helpers.

#### Scenario: Production code cannot bypass role helpers
- **WHEN** backend tests compile after the refactor
- **THEN** production scan, content-shape, recognition, metadata, and catalog code MUST NOT call old generic movie-title cleanup wrappers for role-specific persistence or identity decisions

#### Scenario: Tests document role boundaries
- **WHEN** title role tests run
- **THEN** they MUST cover identity, search, display, raw provenance, file-stem, and sort-title behavior with at least one weak filename and one localized separator case

### Requirement: Existing weak local title data can be repaired
The system SHALL provide a repair or rescan path that can recompute scanner-
derived local title fields and weak movie identities created by older title
normalization without overwriting provider-managed or user-governed metadata.

#### Scenario: Weak local movie row is repaired
- **WHEN** a previously scanned library contains a local movie item whose identity or title was derived only from a year or release tokens
- **THEN** the repair path MUST recompute its local identity and scanner-managed title fields from the role policy and eligible directory evidence

#### Scenario: Governed provider metadata is preserved during repair
- **WHEN** the repair path encounters a metadata item with provider-applied or user-locked title fields
- **THEN** it MUST preserve those governed fields and only repair scanner-managed evidence, resource labels, links, or local fallback fields that are safe to recompute
