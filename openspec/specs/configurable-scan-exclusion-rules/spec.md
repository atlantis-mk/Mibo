# configurable-scan-exclusion-rules Specification

## Purpose
TBD - created by archiving change configure-scan-exclusion-rules. Update Purpose after archive.
## Requirements
### Requirement: Manage automatic scan exclusion rules
The system SHALL provide authenticated management operations for automatic scan exclusion rules, including listing, creating, updating, enabling, disabling, and deleting user-created rules.

#### Scenario: User lists rules
- **WHEN** an authenticated user opens scan exclusion rule settings
- **THEN** the system SHALL return the configured automatic scan exclusion rules with name, type, pattern or token value, reason, enabled state, system/user source, and timestamps

#### Scenario: User creates a rule
- **WHEN** an authenticated user creates a valid automatic scan exclusion rule
- **THEN** the system SHALL persist the rule and make it available to subsequent scan decisions without requiring a backend restart

#### Scenario: User updates a rule
- **WHEN** an authenticated user changes a rule pattern, reason, name, or enabled state
- **THEN** the system SHALL save the change and use the updated rule for subsequent scans

#### Scenario: User deletes a user-created rule
- **WHEN** an authenticated user deletes a user-created automatic rule
- **THEN** the system SHALL stop applying that rule to subsequent scans

### Requirement: Validate automatic rule definitions
The system SHALL validate automatic scan exclusion rules before saving them so malformed or dangerously broad rules are rejected.

#### Scenario: Empty rule value is rejected
- **WHEN** a user submits a rule with an empty token, segment, or path pattern
- **THEN** the system SHALL reject the request with a validation error and SHALL NOT save the rule

#### Scenario: Unsupported rule type is rejected
- **WHEN** a user submits a rule type outside the supported automatic rule type set
- **THEN** the system SHALL reject the request with a validation error

#### Scenario: Token rules remain token-bound
- **WHEN** a filename token rule is saved for `ad`
- **THEN** the scanner SHALL match explicit filename tokens such as `Movie - ad.mkv` and SHALL NOT match substring-only titles such as `Ad Astra.mkv` or `Adventure Movie.mp4`

### Requirement: Preserve seeded advertisement defaults
The system SHALL seed automatic advertisement exclusion rules that preserve the current built-in advertisement matching behavior unless a user disables or changes those rules.

#### Scenario: Existing advertisement filename remains skipped
- **WHEN** a library scan encounters `/movies/Movie A/advertisement.mp4` or `/movies/Movie A/Movie A - ad.mkv` with default rules enabled
- **THEN** the scanner SHALL skip that file and SHALL NOT create catalog, asset, inventory, metadata match, or probe work for it

#### Scenario: Existing advertisement folder remains skipped
- **WHEN** a library scan encounters a video inside a folder explicitly named `ads`, `advertisements`, `commercials`, or `广告` with default rules enabled
- **THEN** the scanner SHALL skip that video file as excluded advertisement content

#### Scenario: Disabled default rule stops applying
- **WHEN** a user disables a seeded automatic rule
- **THEN** subsequent scans SHALL stop applying that rule while preserving the rule record for later re-enabling

### Requirement: Apply rule changes to scan decisions immediately
The scanner SHALL use the latest enabled automatic rule configuration for each newly started scan without requiring process restart or code deployment.

#### Scenario: Newly enabled rule affects next scan
- **WHEN** a user enables or creates a rule that matches `promo.mkv`
- **THEN** the next scan that encounters `promo.mkv` SHALL skip it according to that rule

#### Scenario: Disabled rule no longer affects next scan
- **WHEN** a user disables the only automatic rule matching `commercial.mkv`
- **THEN** the next scan that encounters `commercial.mkv` SHALL process it normally unless another exclusion applies

### Requirement: Keep user exclusions separate and higher priority
The scanner SHALL evaluate persisted user-marked scan exclusions before configurable automatic rules and SHALL keep their management lifecycle separate.

#### Scenario: User exclusion still applies when no automatic rule matches
- **WHEN** a file has an enabled persisted user exclusion but no automatic rule matches its path
- **THEN** the scanner SHALL skip the file as a user exclusion

#### Scenario: Settings separates records from rules
- **WHEN** a user opens the scan exclusions settings area
- **THEN** the UI SHALL distinguish concrete file exclusion records from automatic configurable rules

### Requirement: Report configurable rule matches
The scanner SHALL expose scan-level visibility for files skipped by configurable automatic rules without exposing provider credentials or signed URLs.

#### Scenario: Skip reason identifies automatic rule source
- **WHEN** a scan skips a file because a configurable rule matched
- **THEN** scan results or logs SHALL distinguish that skip from a persisted user exclusion and SHOULD include a safe rule identifier or rule name where available

