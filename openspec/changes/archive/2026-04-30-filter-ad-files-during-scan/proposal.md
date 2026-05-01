## Why

Library scans currently treat any supported video-like file as ingestible media, so advertisement clips, promotional files, download leftovers, or other unwanted videos packaged beside real media can become unwanted catalog assets. Users need a way to correct these imports, remove them from Mibo, and prevent future scans from re-importing the same files.

## What Changes

- Add scanner-level file exclusion checks for files discovered inside library folders.
- Add an operation to mark an already-scanned file or asset as excluded, with `advertisement` as the initial reason.
- When a file is excluded, remove or hide the associated scanner-managed catalog asset from normal browsing without deleting the source storage file.
- Persist a scan exclusion keyed by stable file identity when available, with path fallback, so later scans skip the same unwanted file.
- Skip excluded files before catalog item, asset, inventory file, probe, and metadata work is created for them.
- Keep conservative built-in rules for explicit advertisement files and folders while leaving ambiguous cases for user marking.
- Keep folder traversal intact so filtering one file never prevents sibling media, sidecars, or subfolders from being scanned.
- Record enough scan-level visibility to explain that a file was skipped and why.

## Capabilities

### New Capabilities
- `scan-file-exclusions`: Rules and expected behavior for ignoring excluded files during library scans, including user-marked advertisement exclusions.

### Modified Capabilities

## Impact

- Affected backend scanner code in `mibo-media-server/internal/library`, especially file classification, sync traversal, and persisted scan exclusions.
- Affected catalog and inventory behavior because skipped or user-marked excluded files must not remain visible as normal catalog media and must not create future match or probe work.
- Affected HTTP/API or admin action surface for marking a scanned file or asset as excluded for scan purposes.
- Tests should cover explicit ad filename patterns, user-marked exclusions, normal media filenames that must not be skipped, and folder traversal with mixed media and excluded files.
