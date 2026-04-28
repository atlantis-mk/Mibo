## 1. Subtitle Binding Model

- [x] 1.1 Add or confirm inventory constants for subtitle asset-file role and scanner-managed external subtitle disposition.
- [x] 1.2 Extend catalog/playback stream DTOs minimally so external subtitle tracks can carry file identity, URL, and external availability without breaking existing compact stream fields.
- [x] 1.3 Add helper behavior for deriving subtitle codec/title/language from sidecar extension and filename without parsing subtitle contents.

## 2. Scanner Persistence

- [x] 2.1 Upsert discovered `.srt` and `.ass` sidecars into `inventory_files` during catalog scan writes.
- [x] 2.2 Link sidecar inventory files to the matched `media_assets` row using a subtitle-specific `asset_files` role.
- [x] 2.3 Create or update `media_streams` rows for bound sidecar files with `stream_type = "subtitle"` and `external = true` disposition.
- [x] 2.4 Reconcile scanner-managed subtitle sidecar links on rescan so unchanged sidecars are reused and removed sidecars no longer appear as available tracks.
- [x] 2.5 Preserve existing scanner metadata evidence payload for subtitle sidecars.

## 3. Catalog And Playback Responses

- [x] 3.1 Update catalog asset detail loading to include subtitle-role asset files and their external subtitle streams in asset stream summaries.
- [x] 3.2 Update playback candidate loading to include bound sidecar subtitle files for the selected asset.
- [x] 3.3 Expose sidecar subtitle tracks in playback responses with safe Mibo-controlled URLs or file IDs, not raw OpenList provider internals.
- [x] 3.4 Ensure missing or unresolvable subtitle sidecars do not make the source media playback request fail.

## 4. Tests

- [x] 4.1 Add library scan tests proving basename `.srt` and `.ass` sidecars create subtitle inventory/link/stream records for movies and episodes.
- [x] 4.2 Add rescan tests proving sidecar subtitle binding is idempotent and stale scanner-managed subtitle bindings are removed or marked unavailable.
- [x] 4.3 Add catalog detail tests proving external sidecar subtitles appear in asset stream summaries with `external = true`.
- [x] 4.4 Add playback tests proving sidecar subtitle tracks appear with safe fetch information and do not expose raw OpenList `sign` or mount details.
- [x] 4.5 Add regression tests proving subtitle text is not parsed for classification or metadata matching.

## 5. Verification

- [x] 5.1 Run focused backend tests for library scan sidecar binding.
- [x] 5.2 Run focused backend tests for catalog detail stream summaries.
- [x] 5.3 Run focused backend tests for playback subtitle tracks.
- [x] 5.4 Run `go test ./...` from `mibo-media-server/`.
- [x] 5.5 Manually inspect an OpenList-backed item with a basename `.srt` sidecar, such as Blood Witch, and confirm playback reports the sidecar as an available external subtitle track.
