## Context

The scanner already builds a same-folder sidecar index for each listed directory. Matching `.srt` and `.ass` files are attached to the scan artifact as `SubtitleSidecars`, then written into scanner metadata evidence for the catalog item. That evidence proves the sidecar was discovered, but the sidecar is not persisted as an inventory file, linked to the media asset, or represented as a subtitle `media_stream`, so catalog detail and playback responses still have no usable subtitle track.

Playback currently resolves a catalog item to a primary asset file and derives subtitle tracks from `media_streams` for the selected file. OpenList sidecars can be listed and linked through the storage provider like any other file, but raw provider signatures and auth-bearing URLs must remain hidden from normal catalog responses.

## Goals / Non-Goals

**Goals:**

- Persist discovered `.srt` and `.ass` sidecars as external subtitle attachments for the same media asset as the matched video.
- Surface bound sidecar subtitles in catalog asset detail and playback subtitle track responses.
- Preserve the existing scanner evidence payload so governance/debug views can still show why a subtitle was associated.
- Keep rescans idempotent: existing sidecar bindings are reused or updated, and stale scanner-managed subtitle bindings are removed when no longer discovered.
- Support local and OpenList-backed subtitle sidecars through existing storage provider link/stream mechanisms.

**Non-Goals:**

- This change does not parse subtitle dialogue text for classification, search, or metadata extraction.
- This change does not add subtitle transcoding, format conversion, OCR, or subtitle synchronization.
- This change does not expose raw OpenList `sign`, `mount_details`, or direct signed provider internals in normal responses.
- This change does not change embedded subtitle probing from ffprobe; embedded streams remain probe-owned.

## Decisions

### Bind sidecars as asset files plus external subtitle streams

Discovered subtitle sidecars should be upserted into `inventory_files`, linked to the same `media_assets` row through `asset_files`, and represented as `media_streams` with `stream_type = "subtitle"` and disposition `external = true`. This reuses the existing catalog and playback stream aggregation instead of adding a parallel subtitle-only response model.

Alternative considered: keep sidecars only in scanner evidence and make playback parse evidence payloads. That would couple playback to scanner metadata JSON and bypass existing asset/file/stream contracts.

### Use scanner-managed roles and deterministic stream indexes

Sidecar asset links should use a distinct role such as `subtitle` so they do not compete with the primary source file. Subtitle stream indexes should be deterministic per sidecar file and isolated from ffprobe source-file indexes, for example by using a stable high offset or by ordering subtitle asset files and assigning indexes within their own file rows.

Alternative considered: attach subtitle stream rows to the source video file. That is simpler for current playback aggregation, but it loses the storage path and availability state of the actual subtitle file and makes stale sidecar cleanup harder.

### Keep sidecar linkage scan-owned

The scanner should create and update sidecar inventory/link/stream rows because it already has the directory listing and deterministic association source. Probe can continue owning embedded streams for source media. On rescan, scanner-managed subtitle sidecars for the asset should be reconciled to the current matched sidecar set.

Alternative considered: let probe discover subtitles by calling storage around the source file. That would duplicate scan-sidecar indexing and add extra storage calls, especially for OpenList-backed libraries.

### Serve subtitle URLs through Mibo endpoints

Normal playback responses should not expose raw OpenList signed URLs. The first implementation should either use existing `/api/v1/inventory-files/{id}/stream` for subtitle inventory files or add a bounded subtitle-track URL that internally resolves the inventory file link. The response may include the subtitle file ID and URL, but not provider signatures.

Alternative considered: call OpenList link during playback and return the provider URL directly. That can work technically but risks leaking auth-bearing or short-lived provider internals and is inconsistent with source file streaming.

## Risks / Trade-offs

- Stale sidecar files may remain linked after removal -> Reconcile scanner-managed subtitle asset files on rescan for the selected asset.
- Existing playback track DTOs lack a subtitle URL or file identity -> Extend track DTOs minimally and keep existing fields compatible.
- OpenList sidecar link availability can differ from video availability -> Mark unavailable subtitle tracks out of playback responses or include checks only if the internal stream endpoint can serve them.
- Duplicate embedded and external subtitles can appear -> Preserve both, but mark external sidecars with `external = true` and stable titles so clients can distinguish them.

## Migration Plan

No schema migration is expected if existing `inventory_files`, `asset_files`, and `media_streams` can represent external subtitle files. Existing libraries gain playable sidecar subtitles after rescan. Rollback is code-level: scanner stops creating/updating sidecar links; existing scanner-managed sidecar rows can remain harmless or be cleaned by a follow-up maintenance task.

## Open Questions

- Should subtitle language be inferred from filename suffixes such as `.zh.srt` or remain blank in the first implementation?
- Should playback include unavailable subtitle sidecars with a status, or omit unavailable tracks until a dedicated subtitle health model exists?
