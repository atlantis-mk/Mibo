# Phase 13 Pattern Map — Legacy Backfill Into Catalog Kernel

## Relevant Existing Patterns

### 1. Background work uses typed payloads and worker dispatch

**Analog files**

- `mibo-media-server/internal/library/service.go`
- `mibo-media-server/internal/library/service_libraries.go`
- `mibo-media-server/internal/worker/worker.go`
- `mibo-media-server/internal/worker/worker_catalog_test.go`

**Use for Phase 13**

- Define one exported backfill payload type in `internal/catalog`.
- Enqueue with a stable job key so duplicate active runs collapse.
- Decode typed payloads in the worker and delegate to one catalog service entrypoint.

### 2. Inventory writes are already idempotent

**Analog files**

- `mibo-media-server/internal/inventory/service.go`
- `mibo-media-server/internal/database/inventory_models.go`

**Use for Phase 13**

- Reuse `UpsertFile`, `LinkAssetToItem`, and `LinkAssetToFile` instead of hand-writing raw SQL.
- Build catalog backfill around existing uniqueness rules on storage path and asset link tuples.

### 3. Catalog domain owns canonical item/evidence persistence

**Analog files**

- `mibo-media-server/internal/catalog/service.go`
- `mibo-media-server/internal/catalog/contracts.go`

**Use for Phase 13**

- Keep legacy-to-catalog mapping in `internal/catalog`, not in HTTP or worker packages.
- Use `CreateItem`, `SetExternalID`, and `RecordMetadataSource` as the persistence boundary where
  possible.

### 4. Legacy browse grouping reveals safe fallback heuristics

**Analog files**

- `mibo-media-server/internal/library/query_browse.go`

**Use for Phase 13**

- Reuse the precedence idea (`ExternalID` before `SeriesTitle`) when choosing a series grouping key.
- Do not silently canonicalize ambiguous fallbacks; write report entries instead.

### 5. Progress migration should mirror existing user-state semantics

**Analog files**

- `mibo-media-server/internal/progress/service.go`
- `mibo-media-server/internal/database/models.go` → `PlaybackProgress`
- `mibo-media-server/internal/database/catalog_models.go` → `UserItemData`

**Use for Phase 13**

- Map `PositionSeconds`, `CompletedAt`, and `LastPlayedAt` forward into `UserItemData`.
- Resolve `asset_id` only when the legacy `media_file_id` can be mapped safely.

## Concrete Source Snippets

### Job dispatch pattern

From `internal/worker/worker.go`:

```go
case library.JobKindCatalogRefreshLibraryProjection:
    var payload catalog.LibraryProjectionRefreshPayload
    if err := decodeJobPayload(job.PayloadJSON, &payload); err != nil {
        return err
    }
    return r.catalog.RefreshLibraryProjection(ctx, payload.LibraryID, payload.RootPath)
```

### Inventory file upsert pattern

From `internal/inventory/service.go`:

```go
err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
    Columns:   []clause.Column{{Name: "storage_provider"}, {Name: "storage_path"}},
    DoUpdates: clause.AssignmentColumns([]string{"library_id", "stable_identity_key", "hashes_json", "size_bytes", "modified_at", "container", "status", "updated_at"}),
}).Create(&file).Error
```

### Catalog provider ID upsert pattern

From `internal/catalog/service.go`:

```go
err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
    Columns:   []clause.Column{{Name: "provider"}, {Name: "provider_type"}, {Name: "external_id"}},
    DoUpdates: clause.AssignmentColumns([]string{"item_id", "is_primary", "source", "confidence", "updated_at"}),
}).Create(&externalID).Error
```

### Legacy show grouping key

From `internal/library/query_browse.go`:

```go
func browseShowKey(item database.MediaItem) string {
    if externalID := strings.TrimSpace(item.ExternalID); externalID != "" {
        return "external:" + externalID
    }
    seriesTitle := strings.TrimSpace(item.SeriesTitle)
    if seriesTitle == "" {
        seriesTitle = strings.TrimSpace(item.Title)
    }
    return fmt.Sprintf("library:%d:series:%s", item.LibraryID, strings.ToLower(seriesTitle))
}
```

### Progress upsert baseline

From `internal/progress/service.go`:

```go
err := s.db.WithContext(ctx).
    Where("user_id = ? AND media_item_id = ?", userID, input.MediaItemID).
    First(&progress).Error
```

## Phase-13 Guardrails

- Do not put backfill logic in `OpenList/` or storage adapters.
- Do not run a full backfill inline in an HTTP handler.
- Do not silently merge ambiguous series identities; emit report entries.
- Do not auto-enable `catalog_read_enabled` as part of Phase 13.
- Do not create duplicate catalog items or assets when an existing row can be reused.
