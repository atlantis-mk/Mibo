# Phase 17 Pattern Map

## Purpose

Concrete analogs and contracts for the Phase 17 planner/executor. Use these
patterns directly; do not explore the codebase again unless a task explicitly
sends you to a new file.

## Target Files And Closest Analogs

| Planned file | Role | Closest analog | Pattern to reuse |
|--------------|------|----------------|------------------|
| `mibo-media-server/internal/playback/profile.go` | playback request/response contracts | `mibo-media-server/internal/playback/profile.go` | keep small typed structs with explicit JSON tags and transport-neutral domain fields |
| `mibo-media-server/internal/playback/service.go` | catalog asset selection and inventory-file resolution | `mibo-media-server/internal/playback/service.go` + `mibo-media-server/internal/catalog/backfill_movies.go` | keep ranking + decision helpers in playback service; use explicit asset/item/file joins like backfill code |
| `mibo-media-server/internal/playback/service_test.go` | sqlite-backed playback behavior tests | `mibo-media-server/internal/playback/service_test.go` | temp sqlite DB + local provider registry + seeded rows, then assert decision payloads directly |
| `mibo-media-server/internal/httpapi/catalog_playback_router_test.go` | focused catalog playback route tests | `mibo-media-server/internal/httpapi/router_test.go` playback sections | authenticated request helper, `httptest` recorder, JSON envelope assertions |
| `mibo-media-server/internal/httpapi/handlers_playback.go` | thin authenticated playback handlers | existing `handlers_playback.go` | parse path/query params, delegate to service, absolutize returned URL with `buildPlaybackURL(...)` |
| `mibo-media-server/internal/httpapi/hls.go` | inventory-file keyed HLS artifact service | existing `hls.go` | keep artifact directory, locking, ffmpeg invocation, and playlist rewriting; only swap legacy file lookup for inventory-file lookup |

## Key Code Excerpts

### Playback decision contract pattern

From `mibo-media-server/internal/playback/profile.go`:

```go
type PlaybackRequest struct {
    MediaItemID      uint
    PreferredFileID  uint
    ClientProfile    ClientProfile
    AllowHLSFallback bool
}

type PlaybackDecision struct {
    Kind          string
    ClientProfile ClientProfile
    SelectedBy    string
    FallbackKind  string
    Reasons       []DecisionReason
}
```

**Use for Phase 17:** keep the same explicit `PlaybackDecision` /
`DecisionReason` surface, but replace legacy ids with `item_id`, `asset_id`,
and `inventory_file_id`.

### Playback ranking pattern

From `mibo-media-server/internal/playback/service.go`:

```go
selected, selectedBy, err := selectPlaybackFile(files, req.PreferredFileID, req.ClientProfile)
directDecision := assessDirectPlay(selected, req.ClientProfile)
```

**Use for Phase 17:** keep ranking and direct-play assessment inside the
playback service. Convert these helpers to operate on catalog asset candidates
instead of `database.MediaFile` rows.

### Catalog asset join pattern

From `mibo-media-server/internal/catalog/backfill_movies.go`:

```go
Joins("JOIN asset_items ON asset_items.asset_id = media_assets.id").
Joins("JOIN asset_files ON asset_files.asset_id = media_assets.id").
Where("asset_items.item_id = ? AND asset_items.role = ? AND asset_items.segment_index = ?", catalogItemID, inventory.AssetItemRolePrimary, 0).
Where("asset_files.file_id = ? AND asset_files.role = ? AND asset_files.part_index = ?", inventoryFileID, inventory.FileRoleSource, 0)
```

**Use for Phase 17:** follow this same relationship chain when resolving which
asset belongs to an item and which source file backs that asset.

### HTTP playback handler pattern

From `mibo-media-server/internal/httpapi/handlers_playback.go`:

```go
source, err := r.playback.GetPlaybackSource(req.Context(), playback.PlaybackRequest{ ... })
source.URL = buildPlaybackURL(req, source.URL)
writeJSON(req.Context(), w, http.StatusOK, source)
```

**Use for Phase 17:** keep handlers thin and continue absolutizing relative
playback URLs before returning them.

### HLS service pattern

From `mibo-media-server/internal/httpapi/hls.go`:

```go
func (s *hlsService) PlaylistURL(mediaFileID uint) string
func (s *hlsService) EnsurePlaylist(ctx context.Context, mediaFileID uint) (string, error)
func (s *hlsService) ArtifactPath(mediaFileID uint, name string) (string, error)
```

**Use for Phase 17:** preserve the service shape, artifact locking, and ffmpeg
workflow, but pivot all identifiers and artifact directories to
`inventoryFileID`.

## Rules To Preserve

1. **Playback selection stays in `internal/playback`, not in HTTP handlers.**
2. **Direct/HLS storage access still goes through `providers.Registry`.**
3. **Item-playback failures return decision payloads, not transport 500s.**
4. **New code must not query legacy `database.MediaFile` or
   `database.PlaybackProgress` in the main playback path.**
5. **New route tests should live in focused files instead of expanding the
   already-large shared router test unnecessarily.**
