## ADDED Requirements

### Requirement: File signals are persistently indexed
The system SHALL persist structured filename and path signals for supported video inventory files using a classifier-versioned index separate from physical inventory facts.

#### Scenario: Video file is indexed
- **WHEN** the scanner records or refreshes an available supported video file
- **THEN** the system SHALL persist a file signal row containing the storage provider, storage path, classifier version, file fingerprint, parent path, basename, title candidate, year evidence, episode evidence, role evidence, release hints, and evidence JSON

#### Scenario: Inventory facts remain separate
- **WHEN** file signals are written for an inventory file
- **THEN** the system SHALL keep physical facts such as size, modified time, stable identity, container, content class, and availability in `inventory_files` rather than making `inventory_file_signals` authoritative for storage facts

### Requirement: File signal reuse is fingerprinted
The system SHALL reuse a persisted file signal only when the storage identity, classifier version, and file fingerprint match the current scan facts.

#### Scenario: File is unchanged
- **WHEN** a previously indexed file is scanned again with the same storage path, classifier version, size, modified time, stable identity evidence, and basename-derived fingerprint inputs
- **THEN** the system SHALL reuse the existing file signal without reparsing the filename

#### Scenario: File changed
- **WHEN** a previously indexed file is scanned with changed fingerprint inputs
- **THEN** the system SHALL recompute and replace the file signal before directory profiling uses it

#### Scenario: Classifier version changed
- **WHEN** the filename classifier version changes
- **THEN** the system SHALL treat existing file signals from older classifier versions as non-reusable and SHALL generate new versioned signal rows

### Requirement: Directory profiles prefer indexed signals
The system SHALL build content-shape directory profiles from current indexed file signals when available, and SHALL only use runtime filename parsing as a fallback for missing or stale signal rows.

#### Scenario: All visible files have signals
- **WHEN** a directory profile is built and all visible supported video files have current file signal rows
- **THEN** the profile SHALL aggregate episode coverage, leading numeric coverage, year density, title uniqueness, common title stem, role counts, and release/version evidence from indexed signals without reparsing filenames

#### Scenario: Some files lack signals
- **WHEN** a directory profile is built and one or more visible supported video files lack current file signal rows
- **THEN** the profile builder SHALL parse only the missing files through the existing filename signal extractor and SHALL make those signals available for persistence or future reuse

### Requirement: Signal index supports large directory fast paths
The system SHALL use indexed signals to avoid repeated filename parsing across scan batches, materialization batches, and unchanged rescans for high-cardinality directories.

#### Scenario: Large episode directory spans batches
- **WHEN** a directory with many episode files is materialized through multiple database write batches
- **THEN** the system SHALL reuse the directory's indexed file signals and compiled content-shape plan rather than rebuilding full filename evidence for every batch

#### Scenario: Directory is unchanged on rescan
- **WHEN** a directory fingerprint and its visible file signal fingerprints are unchanged on a later scan
- **THEN** the system SHALL reuse the previous profile, plan, and assignments without reparsing every filename in the directory

### Requirement: Signal evidence is diagnostic and reviewable
The system SHALL retain enough file signal evidence to explain directory-shape and file-assignment decisions during review and debugging.

#### Scenario: Ambiguous directory decision is created
- **WHEN** indexed file signals produce conflicting movie, episode, version, or collection evidence
- **THEN** the resulting content-shape decision SHALL reference signal-derived evidence and alternatives so the ambiguity can be reviewed without re-reading media contents or calling remote providers
