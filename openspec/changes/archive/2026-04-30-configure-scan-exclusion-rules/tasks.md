## 1. Data Model And Defaults

- [x] 1.1 Add a persisted scan exclusion rule model with name, description, rule type, value or pattern, reason, enabled state, system/user source, and timestamps.
- [x] 1.2 Add indexes and uniqueness needed for stable seeded default rules and efficient enabled-rule loading.
- [x] 1.3 Seed default advertisement filename token and directory segment rules that preserve existing hard-coded behavior.
- [x] 1.4 Add validation helpers for supported rule types, required values, supported reasons, and token-bound matching constraints.

## 2. Scanner Integration

- [x] 2.1 Replace hard-coded automatic advertisement matching with configurable rule evaluation while keeping user `scan_exclusions` checks first.
- [x] 2.2 Load enabled automatic rules once per scan or equivalent scan context so rule changes apply to newly started scans without per-file database queries.
- [x] 2.3 Preserve existing false-positive avoidance for titles such as `Ad Astra`, `Adventure Movie`, regular episodes, trailers, samples, and featurettes.
- [x] 2.4 Extend skipped-file accounting or logging so configurable rule matches are distinguishable from persisted user exclusions.

## 3. Backend API

- [x] 3.1 Add authenticated endpoints to list, create, update, enable/disable, and delete scan exclusion rules.
- [x] 3.2 Ensure API responses include safe rule metadata and do not expose storage provider credentials or signed URLs.
- [x] 3.3 Reject invalid rule payloads with clear validation errors.
- [x] 3.4 Protect system default rules from unsafe deletion if the chosen design keeps seeded rules disable-only.

## 4. Settings UI

- [x] 4.1 Extend the scan exclusions settings area with a distinct automatic rules section or page.
- [x] 4.2 Add rule list, empty, loading, error, create, edit, enable/disable, and delete states using existing settings UI patterns.
- [x] 4.3 Add form validation and reason/type labels for filename token, directory segment, and path pattern rules.
- [x] 4.4 Invalidate rule queries after mutations so changes are visible immediately in the UI.

## 5. Verification

- [x] 5.1 Add backend tests for rule CRUD, validation, seeded defaults, disable behavior, and delete behavior.
- [x] 5.2 Add scanner tests proving configurable rules skip matching files and preserve normal media scanning for false-positive cases.
- [x] 5.3 Add tests proving user exclusions still take priority and remain separate from automatic rules.
- [x] 5.4 Run focused backend tests for scan exclusions and rule management.
- [x] 5.5 Run frontend typecheck after adding API client and settings UI changes.
