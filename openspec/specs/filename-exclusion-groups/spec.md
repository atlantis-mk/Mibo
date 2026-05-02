# filename-exclusion-groups Specification

## Purpose
TBD - created by syncing change add-filename-exclusion-groups. Update Purpose after archive.
## Requirements
### Requirement: Filename Exclusion Impact Preview
The system SHALL provide an authenticated preview for same-name exclusion impact before creating a filename exclusion rule. The preview SHALL include all sources and SHALL use the backend's normalized filename matching semantics.

#### Scenario: Preview same-name files from an item
- **WHEN** an authenticated user requests an exclusion preview for a catalog item with a linked source file
- **THEN** the system returns the normalized filename, selected-file context, total affected count, and affected file entries from all sources

#### Scenario: Preview includes other sources
- **WHEN** another library or storage source contains a file with the same normalized filename
- **THEN** the preview includes that file in the affected count and affected file entries

### Requirement: Ignore Choice Before Batch Exclusion
The system SHALL let users choose between ignoring only the selected file and ignoring all same-name files after showing the same-name impact preview.

#### Scenario: User chooses single-file ignore
- **WHEN** the user selects the single-file ignore action from the preview flow
- **THEN** the system creates or updates a file-scoped manual scan exclusion for only the selected file

#### Scenario: User chooses same-name ignore
- **WHEN** the user selects the same-name ignore action from the preview flow
- **THEN** the system creates or updates a global filename exclusion rule for the selected file's normalized filename

### Requirement: Filename Exclusion Rule Matching
The system SHALL skip future scanned video files from any source that match an enabled filename exclusion rule, unless a restore exception explicitly allows the file.

#### Scenario: Future same-name file is skipped
- **WHEN** a scan encounters a video file whose normalized filename matches an enabled filename exclusion rule
- **THEN** the system skips importing that file and records the skip source as a filename exclusion rule

#### Scenario: Different extension is not matched
- **WHEN** a scan encounters a file with the same basename but a different extension than an enabled filename exclusion rule
- **THEN** the system does not skip the file because of that filename exclusion rule

#### Scenario: Restored file overrides rule
- **WHEN** a scan encounters a file that matches an enabled filename exclusion rule and also matches a per-file restore exception
- **THEN** the system allows that file through filename rule exclusion matching

### Requirement: Existing Same-Name Files Hidden
The system SHALL hide already-scanned files that match a newly enabled filename exclusion rule without physically deleting source files.

#### Scenario: Existing matching files are removed from catalog visibility
- **WHEN** a filename exclusion rule is created or re-enabled
- **THEN** matching inventory files and linked assets from all sources are marked unavailable or missing, item links are removed, and affected catalog item availability is recalculated

#### Scenario: Source files are not deleted
- **WHEN** existing matching files are hidden by a filename exclusion rule
- **THEN** the system does not delete the underlying local or OpenList source files

### Requirement: Exclusion Management Display
The system SHALL display filename exclusion rules and their affected files in the scan exclusions management UI.

#### Scenario: Filename rule is listed
- **WHEN** a filename exclusion rule exists
- **THEN** the exclusions management UI shows the rule filename, all-source scope, reason, enabled state, affected count, and available restore actions

#### Scenario: Affected files are inspectable
- **WHEN** the user expands or opens a filename exclusion rule
- **THEN** the UI shows affected file paths and indicates which files are excluded versus individually restored

### Requirement: Single-File Restore Exception
The system SHALL allow users to restore one file from a filename exclusion rule while leaving the rule enabled for other matching files.

#### Scenario: Restore one file
- **WHEN** the user restores one affected file from a filename exclusion rule
- **THEN** the system records a per-file restore exception and allows that file to be imported on a future scan

#### Scenario: Other same-name files remain excluded
- **WHEN** one file has a restore exception and another same-name file matching the rule does not
- **THEN** future scans allow the restored file and continue skipping the other same-name file

### Requirement: Filename Rule Restore
The system SHALL allow users to restore all same-name files by disabling the filename exclusion rule.

#### Scenario: Restore rule
- **WHEN** the user restores a filename exclusion rule
- **THEN** the system disables the rule and future scans no longer skip files because of that rule

#### Scenario: Rule history remains visible
- **WHEN** a filename exclusion rule is disabled
- **THEN** the system preserves the rule record and displays it as restored or inactive rather than deleting the history
