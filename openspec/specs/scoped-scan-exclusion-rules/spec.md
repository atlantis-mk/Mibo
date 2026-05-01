# scoped-scan-exclusion-rules Specification

## Purpose
TBD - created by archiving change scope-scan-exclusion-rules. Update Purpose after archive.
## Requirements
### Requirement: Global and library-scoped scan exclusion rules
The system SHALL allow configurable scan exclusion rules to be either global or scoped to a single media library.

#### Scenario: Existing rules remain global
- **WHEN** the system reads an existing scan exclusion rule without a library scope
- **THEN** the rule is treated as global and remains eligible for every library scan

#### Scenario: User creates a library-scoped rule
- **WHEN** a user creates a configurable scan exclusion rule and selects a media library scope
- **THEN** the system stores the rule with that library as its owner

#### Scenario: User creates a global rule
- **WHEN** a user creates a configurable scan exclusion rule without selecting a media library scope
- **THEN** the system stores the rule as global

### Requirement: Scoped rule matching during scans
The system SHALL apply enabled global configurable rules and enabled configurable rules scoped to the library being scanned.

#### Scenario: Scan applies global and matching library rules
- **WHEN** a scan runs for a media library with configurable exclusion rules enabled
- **THEN** the scan evaluates enabled global rules and enabled rules scoped to that media library

#### Scenario: Scan ignores rules from other libraries
- **WHEN** a scan runs for one media library
- **THEN** the scan does not evaluate configurable rules scoped to a different media library

#### Scenario: Library policy disables configurable rules
- **WHEN** a scan runs for a media library whose scan policy disables configurable exclusion rules
- **THEN** the scan does not evaluate configurable global rules or configurable library-scoped rules

### Requirement: Scoped rule lifecycle
The system SHALL maintain scoped scan exclusion rules with the lifecycle of their owning media library.

#### Scenario: Deleting a library removes scoped rules
- **WHEN** a media library is deleted
- **THEN** the system deletes configurable scan exclusion rules scoped to that media library

#### Scenario: Deleting a library preserves global rules
- **WHEN** a media library is deleted
- **THEN** the system preserves configurable scan exclusion rules without a library scope

#### Scenario: Deleting a library preserves other library rules
- **WHEN** a media library is deleted
- **THEN** the system preserves configurable scan exclusion rules scoped to other media libraries

### Requirement: Scoped rule management API and UI
The system SHALL expose rule scope through scan exclusion rule management APIs and settings UI.

#### Scenario: API returns rule scope
- **WHEN** a client lists configurable scan exclusion rules
- **THEN** each rule includes whether it is global or scoped to a media library

#### Scenario: API validates scoped rule library
- **WHEN** a client creates or updates a rule with a media library scope that does not reference an existing library
- **THEN** the system rejects the request with a validation error

#### Scenario: UI displays and edits rule scope
- **WHEN** a user manages configurable scan exclusion rules in settings
- **THEN** the UI allows choosing global scope or a specific media library and displays the chosen scope in the rule list

### Requirement: Scoped rule uniqueness
The system SHALL prevent duplicate configurable scan exclusion rules within the same scope while allowing equivalent rules in different scopes.

#### Scenario: Duplicate rule in same scope is rejected
- **WHEN** a user creates a configurable scan exclusion rule with the same normalized type and value as an existing rule in the same scope
- **THEN** the system rejects the duplicate rule

#### Scenario: Equivalent rules in different scopes are allowed
- **WHEN** a global rule and a library-scoped rule use the same normalized type and value
- **THEN** the system allows both rules because their scopes are different

