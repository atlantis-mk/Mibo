## ADDED Requirements

### Requirement: OpenList adapter captures safe object metadata
The OpenList storage adapter SHALL parse safe object metadata from `/api/fs/list` and `/api/fs/get` responses into provider-neutral storage objects, including creation time, object type, provider identity, hash information, raw media URL, thumbnail URL, and sanitized optional metadata indicators.

#### Scenario: List response includes additional object metadata
- **WHEN** OpenList `/api/fs/list` returns object fields such as `created`, `type`, `provider`, `hash_info`, and `thumb`
- **THEN** Mibo MUST preserve the safe fields in `storage.Object` without requiring OpenList-specific response structs outside the OpenList adapter

#### Scenario: Missing optional metadata remains compatible
- **WHEN** OpenList omits optional metadata fields or another storage provider does not support them
- **THEN** Mibo MUST continue listing, scanning, probing, and playback behavior using existing fallback logic

### Requirement: Related files accelerate sibling discovery
The system SHALL use `/api/fs/get` related-file metadata as an optimization for finding sibling artwork and future sidecar candidates before issuing individual candidate `Get` requests.

#### Scenario: Related metadata includes matching sibling artwork
- **WHEN** a media object's related files include a supported sibling poster or backdrop candidate
- **THEN** Mibo MUST use that related object for sibling discovery before probing candidate paths individually

#### Scenario: Related metadata is absent or incomplete
- **WHEN** related metadata is absent, empty, stale, or does not include the requested candidate
- **THEN** Mibo MUST fall back to the existing direct candidate path lookup behavior

### Requirement: OpenList object type is auxiliary
The system SHALL treat OpenList object `type` as an auxiliary diagnostic or classification hint rather than the source of truth for media scan inclusion.

#### Scenario: Object type conflicts with Mibo extension classification
- **WHEN** OpenList `type` and Mibo's extension-based classification disagree
- **THEN** Mibo MUST keep using its existing media classification rules for scan inclusion and MAY retain the OpenList type as diagnostic metadata

### Requirement: Sensitive OpenList metadata is filtered from normal APIs
The system SHALL NOT expose raw OpenList `sign`, `mount_details`, write/upload capability flags, or auth-bearing provider internals through normal catalog, item, playback, or frontend response contracts.

#### Scenario: OpenList response contains sensitive provider details
- **WHEN** OpenList returns `sign`, `mount_details`, write flags, upload tools, or other provider-internal details
- **THEN** Mibo MUST either ignore them or expose only sanitized admin/debug summaries that do not include raw credentials, signed tokens, or storage internals

### Requirement: Provider metadata use is non-authoritative
The system SHALL NOT allow OpenList metadata to overwrite user-selected, TMDB-selected, manually governed, or otherwise authoritative catalog state unless an explicit governance action requests that change.

#### Scenario: Provider metadata differs from selected catalog state
- **WHEN** OpenList metadata contains provider names, hashes, related files, type hints, or timestamps that differ from existing selected metadata or artwork
- **THEN** Mibo MUST preserve existing selected catalog state and use provider metadata only for discovery, diagnostics, or fallback behavior
