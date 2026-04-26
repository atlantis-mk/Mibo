# Phase 12 Pattern Map — Catalog Kernel Contracts & Migration Guards

## Relevant Existing Patterns

### 1. API-safe typed structs live outside `internal/database`

**Analog files**

- `mibo-media-server/internal/library/service.go` → `MediaSourceView`
- `mibo-media-server/internal/settings/service.go` → `MetadataSettings`, `ScanSettings`

**Use for Phase 12**

- Create catalog DTOs as plain exported structs with `json` tags only.
- Keep `database.CatalogItem` and `database.MediaAsset` internal.

### 2. Durable settings use category/key rows with upsert helpers

**Analog files**

- `mibo-media-server/internal/settings/service.go`
- `mibo-media-server/internal/database/models.go` → `SystemSetting`

**Use for Phase 12**

- Store migration guards under a dedicated category such as `catalog_migration`.
- Reuse `clause.OnConflict` upserts and typed parse helpers.

### 3. Queueable background work uses job constants + enqueue helpers + worker dispatch

**Analog files**

- `mibo-media-server/internal/library/service.go`
- `mibo-media-server/internal/library/service_libraries.go`
- `mibo-media-server/internal/worker/worker.go`
- `mibo-media-server/internal/worker/worker_test.go`

**Use for Phase 12**

- Define new catalog projection job kinds in `internal/library`.
- Add queue helpers there.
- Add worker `case` branches with typed payload decoding.
- Cover with `RunOnce` integration tests.

### 4. Schema safety is verified through `database.Open(...)`

**Analog files**

- `mibo-media-server/internal/database/database.go`
- `mibo-media-server/internal/database/catalog_models_test.go`

**Use for Phase 12**

- Add minimum composite indexes in the migration boundary used by startup.
- Add tests proving empty DB and legacy DB still boot.

## Concrete Source Snippets

### System settings uniqueness

From `internal/database/models.go`:

```go
type SystemSetting struct {
    Category string `gorm:"size:64;not null;uniqueIndex:idx_system_setting_category_key"`
    Key      string `gorm:"size:128;not null;uniqueIndex:idx_system_setting_category_key"`
}
```

### Settings upsert pattern

From `internal/settings/service.go`:

```go
return s.db.WithContext(ctx).Clauses(clause.OnConflict{
    Columns:   []clause.Column{{Name: "category"}, {Name: "key"}},
    DoUpdates: clause.AssignmentColumns([]string{"value", "is_secret", "updated_at"}),
}).Create(&record).Error
```

### Job dispatch pattern

From `internal/worker/worker.go`:

```go
case library.JobKindReindexSearchDocument:
    var payload struct { MediaItemID uint `json:"media_item_id"` }
    if err := decodeJobPayload(job.PayloadJSON, &payload); err != nil { return err }
    return r.search.ReindexMediaItem(ctx, payload.MediaItemID)
```

### Existing catalog enum source of truth

From `internal/catalog/service.go`:

```go
const (
    ItemTypeMovie   = "movie"
    ItemTypeSeries  = "series"
    ItemTypeSeason  = "season"
    ItemTypeEpisode = "episode"

    AvailabilityAvailable    = "available"
    AvailabilityMissing      = "missing"
    AvailabilityUnaired      = "unaired"
    AvailabilityNoLocalMedia = "no_local_media"

    GovernancePending     = "pending"
    GovernanceMatched     = "matched"
    GovernanceNeedsReview = "needs_review"
    GovernanceLocked      = "locked"
    GovernanceManual      = "manual"
    GovernanceUnmatched   = "unmatched"
)
```

## Phase-12 Guardrails

- Do not add new product logic under `OpenList/`.
- Do not expose `database.*` rows directly in new catalog contracts.
- Do not create a parallel migration subsystem outside `internal/database/database.go` unless startup compatibility proves impossible.
- Do not invent a second settings persistence path when `SystemSetting` already solves the problem.
