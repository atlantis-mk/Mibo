## Context

Mibo already skips files through two mechanisms: persisted user exclusions in `scan_exclusions`, and automatic hard-coded advertisement path checks in `mibo-media-server/internal/library/scan_exclusion.go`. The settings UI currently manages only user-marked exclusions at `/settings/scan-exclusions`; automatic rules are not visible and cannot be changed without editing code.

The new design should keep the scanner conservative, preserve existing seeded advertisement behavior, and let authenticated users manage automatic rules safely from Settings. Rule changes should be read dynamically by scan decisions so the next scan uses the latest saved configuration.

## Goals / Non-Goals

**Goals:**
- Persist automatic scan exclusion rules in the backend database instead of hard-coding all advertisement markers.
- Allow authenticated users to create, read, update, delete, enable, and disable automatic rules from Settings.
- Preserve the existing automatic advertisement behavior through seeded default rules.
- Keep user-marked exclusions in `scan_exclusions` separate from automatic rules and evaluate user exclusions first.
- Apply rule changes immediately to subsequent scan decisions without process restart.
- Keep matching conservative enough to avoid substring false positives such as `Ad Astra` and `Adventure Movie`.
- Validate rules before saving so malformed or overly broad rules do not silently break scanning.

**Non-Goals:**
- Do not physically delete provider files when a rule matches.
- Do not add content-based video analysis, OCR, duration heuristics, or machine-learning classification.
- Do not require live rescanning or retroactive catalog cleanup immediately after editing a rule; changed rules affect future scans unless the user manually triggers a rescan.
- Do not merge configurable automatic rules with user-marked exclusion records; they have different lifecycle and audit semantics.

## Decisions

- Add a new `scan_exclusion_rules` database model rather than storing automatic rules in `scan_exclusions`. User exclusions target concrete files; automatic rules define reusable matching behavior and need distinct names, descriptions, enabled state, and validation.
- Support a small explicit rule type set: filename token, directory segment, and path glob or pattern. Filename token and directory segment rules should remain the primary seeded defaults because they match the current false-positive-resistant behavior.
- Scope rules globally at first, with optional library scope left out unless implementation finds an existing settings pattern that makes per-library filtering cheap. Global rules match the current hard-coded behavior and keep the first UI simpler.
- Evaluate user exclusions before automatic rules. Stable identity and path-based user exclusions are more precise and should remain the authoritative user correction mechanism.
- Seed default advertisement rules during migration or startup idempotently. The defaults should cover the current terms: `ad`, `ads`, `advert`, `adverts`, `advertisement`, `advertisements`, `commercial`, `commercials`, and `广告`, with the same token and directory-segment constraints as today.
- Read active rules dynamically during scans through the library service database handle. Avoid process-level immutable caches unless they are invalidated on CRUD writes; correctness and immediate effect are more important than caching for the expected rule count.
- Return the matched rule source in skip accounting, such as `configurable_rule:<id>` or a stable source label plus rule id/name in logs. Public API responses should avoid leaking provider credentials or signed URLs.
- Add a Settings UI section under scan exclusions rather than a separate product area. Users already expect scan ignore management there, and the panel can clearly separate “文件排除记录” from “自动规则”.
- Implement delete as hard deletion for rules unless a rule has audit requirements. Disabling remains the safe reversible path for built-in/default rules; the UI should prefer disabling seeded defaults and allow deletion only for user-created rules if that distinction is available.

## Risks / Trade-offs

- Overly broad user rules could hide valid media -> Mitigation: validate type-specific fields, preview/test a sample path in the UI, and keep default matching token-bound.
- Dynamic database reads could add overhead during large scans -> Mitigation: load enabled rules once per scan or per directory walk context while ensuring CRUD writes affect the next scan; avoid per-file database queries for rules.
- Seeded defaults might duplicate if startup/migration runs repeatedly -> Mitigation: use stable rule keys or unique names for built-in rules and idempotent upsert behavior.
- Deleting built-in rules could make support harder -> Mitigation: include an `is_system` or `source` field and prefer disabling system rules over deleting them.
- Existing hard-coded tests may become brittle -> Mitigation: rewrite tests around seeded rule records and preserve current false-positive cases.

## Migration Plan

- Add the new rule table and idempotently seed default advertisement rules.
- Change scanner automatic matching to read enabled configurable rules while preserving existing behavior under seeded data.
- Keep the old hard-coded helper only as a fallback during implementation tests if necessary, then remove it once seeded rules are covered.
- Rollback can disable configurable-rule evaluation and keep user exclusions unaffected; seeded rule records can remain inert data.

## Open Questions

- Should configurable rules be global only for the first version, or should the data model include nullable `library_id` now for later per-library overrides?
- Should system default rules be deletable, or only disableable and resettable?
- Should the UI include a path test field in the first implementation, or defer preview/testing until after core CRUD is shipped?
