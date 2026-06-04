## ADDED Requirements

### Requirement: Directory snapshots are durable stage outputs
The system SHALL persist a directory snapshot for each scanned directory before inventory, signal, shape, recognition, materialization, enrichment, or projection stages consume that directory.

#### Scenario: Snapshot is created before downstream work
- **WHEN** a library scan observes a directory from a storage provider
- **THEN** the system records the directory path, root path, storage provider, child directories, normalized file summaries, visible media paths, fingerprint, scan run identity, and last observed timestamp before scheduling downstream stage work

#### Scenario: Downstream stages consume snapshots
- **WHEN** inventory sync, file signal hydration, directory shape planning, recognition unit construction, or materialization needs directory contents
- **THEN** the system reads the persisted snapshot and MUST NOT list the storage provider again for that directory during the same stage chain

### Requirement: Unchanged directories skip dependent stages
The system SHALL compare directory and stage fingerprints to skip inventory, signal, directory shape, recognition unit, and materialization work for unchanged directories.

#### Scenario: Directory fingerprint is unchanged
- **WHEN** a scan observes a directory whose normalized snapshot fingerprint matches the last successful snapshot fingerprint for the same library, provider, root path, directory path, and scanner version
- **THEN** the system marks the directory unchanged and does not schedule inventory, signal, shape, recognition, or materialization tasks for that directory unless forced refresh is requested

#### Scenario: Scan policy changes invalidate downstream stages
- **WHEN** scan policy, exclusion rules, scanner version, or signal classifier version changes for a library
- **THEN** the system treats affected directory stage fingerprints as stale even if the provider object list is unchanged

### Requirement: Inventory sync consumes snapshots in batches
The system SHALL update inventory files and sidecar facts from directory snapshots using batch operations.

#### Scenario: Changed snapshots produce inventory facts
- **WHEN** a changed directory snapshot contains video files, sidecars, or other supported file classes
- **THEN** the system batch upserts inventory file rows, records missing files for paths no longer visible in affected snapshots, and persists sidecar associations without recomputing the provider listing

#### Scenario: Inventory facts are reused by later stages
- **WHEN** file signal hydration or directory shape planning runs after inventory sync
- **THEN** the system uses inventory rows associated with the snapshot instead of reparsing provider object summaries as source-of-truth file facts

### Requirement: File signals are parsed once per signal version
The system SHALL persist reusable file signals before directory shape planning and recognition unit construction.

#### Scenario: Missing signal is hydrated
- **WHEN** an available video inventory file in a changed snapshot has no valid signal for the active classifier version
- **THEN** the system parses filename, path hints, sidecars, and local evidence once and stores the resulting signal

#### Scenario: Existing signal is reused
- **WHEN** an available video inventory file already has a valid signal for the active classifier version and unchanged signal inputs
- **THEN** the system reuses the stored signal for directory shape and recognition stages

### Requirement: Directory shape planning is authoritative
The system SHALL produce a directory content shape plan and file assignments from snapshots, inventory facts, file signals, and scan rules before recognition units are generated.

#### Scenario: Directory shape is planned
- **WHEN** a changed directory contains visible video files or recognized content hints
- **THEN** the system stores or updates a content shape plan and assignments that identify the directory as movie folder, movie versions folder, movie collection folder, season folder, episode pack, flat episode folder, attachment group, or review-required

#### Scenario: Downstream stages use planned shape
- **WHEN** recognition, materialization, metadata matching, or governance needs directory type
- **THEN** the system consumes the stored content shape plan and assignments instead of inferring directory type again from paths alone
