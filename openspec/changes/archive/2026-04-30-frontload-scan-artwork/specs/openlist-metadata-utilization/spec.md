## MODIFIED Requirements

### Requirement: OpenList adapter captures safe object metadata
The OpenList storage adapter SHALL parse safe object metadata from `/api/fs/list` and `/api/fs/get` responses into provider-neutral storage objects, including creation time, object type, provider identity, hash information, raw media URL, thumbnail URL, and sanitized optional metadata indicators, and scan-time consumers SHALL be able to use thumbnail URL as fallback artwork metadata without depending on OpenList-specific structs.

#### Scenario: List response includes additional object metadata
- **WHEN** OpenList `/api/fs/list` returns object fields such as `created`, `type`, `provider`, `hash_info`, and `thumb`
- **THEN** Mibo MUST preserve the safe fields in `storage.Object` without requiring OpenList-specific response structs outside the OpenList adapter

#### Scenario: Missing optional metadata remains compatible
- **WHEN** OpenList omits optional metadata fields or another storage provider does not support them
- **THEN** Mibo MUST continue listing, scanning, probing, and playback behavior using existing fallback logic

#### Scenario: Thumbnail metadata is available for scan artwork preselection
- **WHEN** OpenList exposes a thumbnail URL for a scanned media object
- **THEN** Mibo MUST make that thumbnail URL available through provider-neutral storage metadata so the scan phase can use it as provisional artwork when no better artwork exists

### Requirement: Provider metadata use is non-authoritative
The system SHALL NOT allow OpenList metadata to overwrite user-selected, TMDB-selected, manually governed, or otherwise authoritative catalog state unless an explicit governance action requests that change, and any scan-phase artwork derived from OpenList metadata MUST remain provisional fallback artwork.

#### Scenario: Provider metadata differs from selected catalog state
- **WHEN** OpenList metadata contains provider names, hashes, related files, type hints, timestamps, or thumbnail URLs that differ from existing selected metadata or artwork
- **THEN** Mibo MUST preserve existing selected catalog state and use provider metadata only for discovery, diagnostics, or fallback behavior

#### Scenario: Remote metadata replaces provider thumbnail
- **WHEN** a catalog item selected an OpenList thumbnail as provisional artwork and later receives a higher-authority remote metadata image
- **THEN** Mibo MUST allow the higher-authority image to replace the provider thumbnail according to existing catalog image selection rules
