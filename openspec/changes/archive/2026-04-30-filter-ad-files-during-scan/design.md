## Context

Library scans currently walk every directory object, ignore non-video extensions, and send each supported video file through classification, catalog writing, metadata matching, and probe queueing. Advertisement clips, promotional videos, samples that should not be imported, and other unwanted files can be stored beside real media, but users cannot currently teach Mibo that a scanned file should be ignored in future scans.

The implementation should stay primarily in the backend scanner path under `mibo-media-server/internal/library`, with a bounded catalog or admin operation for marking scanned files as scan exclusions. Source files must not be physically deleted from storage by default.

## Goals / Non-Goals

**Goals:**
- Skip excluded files before catalog items, assets, inventory rows, match jobs, or probe jobs are created.
- Allow a user/admin operation to mark an already-scanned file or asset as excluded from future scans.
- Persist user-marked scan exclusions so the same file is skipped on later scans.
- Remove or hide user-marked excluded assets from normal catalog browsing after they are marked.
- Use conservative filename and path rules that identify explicit ad markers without matching arbitrary title substrings.
- Preserve recursive traversal and sidecar discovery for legitimate media in the same folder.
- Expose scan-level visibility for skipped files through result counters and logs or decisions, including skip reason.
- Cover the behavior with focused scanner tests.

**Non-Goals:**
- Physically delete excluded files from OpenList, local disk, or any upstream storage provider.
- Add per-library user configuration, UI controls, or provider-specific remote deletion.
- Analyze video duration, perceptual content, OCR, or audio to detect unwanted files.
- Automatically treat trailers, samples, featurettes, or other intentional extras as exclusions unless they carry explicit ad markers or the user marks them.

## Decisions

- Apply exclusion checks immediately after `isVideoFile` in directory walking. This prevents unwanted catalog, inventory, match, and probe work while keeping traversal and non-video sidecar indexing unchanged.
- Implement the filter as a scanner classification helper rather than inside catalog writing. Catalog writers should continue to assume they receive intentional scan artifacts, and the filter can be unit-tested against storage paths without constructing catalog state.
- Add a persisted `scan_exclusions` style record. The exclusion should store library scope, storage provider, stable identity when present, normalized path as fallback, reason, enabled/disabled state, and audit metadata such as creation time.
- Start with exclusion reasons such as `advertisement`, `unwanted`, `duplicate`, `wrong_import`, and `other`, while the initial UI/API can expose only `advertisement` if that is the immediate product need.
- When a scanned asset or inventory file is marked as excluded, create or update the exclusion and remove the scanner-managed asset from normal catalog availability. Prefer soft removal or scanner-managed unlinking over hard deletes so the operation is reversible and does not affect provider files.
- Check persisted exclusions before automatic filename rules. Stable identity should win over path matching to survive provider renames; path fallback handles providers or files without stable identities.
- Use token-bound filename and path-segment checks only for explicit ad indicators such as `ad`, `ads`, `advert`, `advertisement`, `commercial`, and `广告`. Substring matches are intentionally avoided so titles like `Ad Astra` or `Adventure Movie` are not skipped.
- Treat dedicated advertisement folders as stronger evidence than arbitrary parent names, but still only skip video files inside those folders. The scanner must continue recursing through the folder so any non-ad subfolders or later legitimate files are reachable.
- Keep ambiguous promotional or suspicious files out of automatic skip rules for now; users can mark them after import, and future work can add a review queue if needed.
- Add skipped-file counts to `SyncResult` and include debug or info logging where the service already supports it. This gives operational visibility with reason labels such as `user_exclusion` and `explicit_ad_rule`.

## Risks / Trade-offs

- False positives from aggressive ad terms -> Mitigation: require exact normalized tokens or explicit ad directory names rather than substring matching.
- False negatives for provider-specific naming conventions -> Mitigation: keep the helper centralized and covered by tests so additional explicit patterns can be added safely later.
- Accidental user marking hides a valid file -> Mitigation: use soft removal and keep an audit record so a later unmark/restore operation can be added without provider file loss.
- Path fallback can skip a different file that reuses the same path -> Mitigation: prefer stable identity when available and scope path fallback to library and provider.
- Generic exclusion reasons can broaden scope -> Mitigation: implement the generic data model but expose only the minimal marking path needed for advertisements first.
- Users may expect configurable rules -> Mitigation: start with a conservative built-in filter and defer settings until there is real demand and examples.
