## Context

Library creation currently exposes movie and show library types. Backend scanning already has directory-shape logic, extra-file detection, movie grouping, and TV fallback episode classification, but those paths are selected by the library type before scanning begins.

Mixed content libraries need a conservative classification layer for source trees that contain both movie folders and multi-video TV-like groups. The catalog should still store and serve standard movie, series, season, episode, asset, and inventory entities so existing consumers do not need a new item type.

## Goals / Non-Goals

**Goals:**

- Add a user-selectable mixed content library type during library creation.
- Classify each mixed-library media group after removing known extras from the media count.
- Use one non-extra media file as the movie signal and multiple non-extra media files as the series signal.
- Keep catalog API responses compatible with existing movie and series item semantics.
- Preserve existing behavior for movie and show libraries.

**Non-Goals:**

- Add a new catalog item type for mixed content.
- Build full metadata-provider disambiguation for ambiguous mixed folders in this change.
- Add manual per-folder classification overrides in this change.
- Treat all extras as scan exclusions; known extras remain supporting media signals unless existing scanner behavior says otherwise.

## Decisions

1. Represent the new library type as a library-level type value, not a catalog item type.

   Rationale: mixed content describes scan strategy for a source root. Once classified, items are still movies or TV hierarchy nodes, which keeps detail, playback, home, and library browse APIs stable.

   Alternative considered: create a catalog `mixed` item type. This would force every consumer branch that assumes movie/series/season/episode semantics to understand a wrapper type without improving playback or browsing.

2. Apply mixed classification at the directory/group decision boundary.

   Rationale: the scanner already reasons about directory shapes and can count videos after filename signals identify extras. Keeping this logic near `resolveDirectoryShape` avoids duplicating grouping behavior in catalog writers.

   Alternative considered: classify only by filename pattern. This is too fragile for folders with clean names but no `SxxEyy` tokens, which is exactly the common mixed-library use case.

3. Count only non-extra video files for mixed movie-vs-series decisions.

   Rationale: trailers, behind-the-scenes clips, samples, featurettes, interviews, and deleted scenes often appear beside a movie and must not turn a single movie folder into a series.

   Alternative considered: count all supported videos. This would misclassify normal movie folders that include bonus content.

4. Map multi-video mixed groups to existing TV fallback behavior.

   Rationale: for the requested first version, multiple non-extra media files are considered TV-like. Existing show scanning can synthesize season/episode ordering for sorted non-extra files when explicit episode numbers are absent.

   Alternative considered: require explicit episode tokens before creating series. That would fail the requested simple rule and leave many mixed source folders unclassified.

## Risks / Trade-offs

- Ambiguous movie folders with multiple cuts or versions may be classified as series in mixed libraries. Mitigation: keep existing movie libraries unchanged and document mixed classification as simple count-based behavior for now.
- Extra detection could accidentally ignore a legitimate main title containing an extra keyword. Mitigation: reuse or extend token-bound extra matching rather than substring-only matching.
- Existing UI labels may display the raw type value. Mitigation: add a localized mixed library option and label alongside movie and show options.
- Catalog API consumers may assume each library contains only one content family. Mitigation: continue returning standard movie and series list items and avoid a new item type.

## Migration Plan

- No data migration is required for existing libraries.
- New mixed libraries persist the new library type value at creation time.
- Rollback can hide the UI option and leave existing mixed libraries readable as library records; scans can be paused or rejected for the mixed type if the backend change is reverted.

## Open Questions

- Should a future version expose per-folder manual overrides for multi-version movie folders inside mixed libraries?
