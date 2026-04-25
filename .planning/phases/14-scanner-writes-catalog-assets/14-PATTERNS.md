# Phase 14 Pattern Map — Scanner Writes Catalog Assets

## Relevant Existing Patterns

### 1. Scan orchestration walks storage first, then queues downstream refreshes

**Analog files**

- `mibo-media-server/internal/library/scan_run.go`
- `mibo-media-server/internal/library/service_libraries.go`
- `mibo-media-server/internal/worker/worker_catalog_test.go`

**Use for Phase 14**

- Keep `RunSyncLibrary` / `RunTargetedRefresh` as the orchestration entrypoints.
- Preserve one library-scope `catalog_refresh_library_projection` enqueue after traversal.
- Replace legacy row writes inside the loop, not the outer scan lifecycle.

### 2. Catalog and inventory persistence is already idempotent

**Analog files**

- `mibo-media-server/internal/catalog/service.go`
- `mibo-media-server/internal/inventory/service.go`

**Use for Phase 14**

- Reuse `CreateItem`, `RecordMetadataSource`, `UpsertFile`, `CreateAsset`,
  `LinkAssetToItem`, and `LinkAssetToFile` instead of inventing raw SQL write paths.
- Express multi-episode and version behavior through `asset_items` / `asset_files`, not
  duplicate catalog rows.

### 3. Stable-identity rescans should preserve one file identity across rename/move

**Analog files**

- `mibo-media-server/internal/library/scan_upsert.go`
- `mibo-media-server/internal/library/scan_identity_test.go`

**Use for Phase 14**

- Reuse the stable-identity-first matching idea when deciding whether a reappearing file should
  reuse an existing `inventory_file` row.
- Treat path-only changes as rename/move continuity, not as a new governed media item.

### 4. Probe work already follows queue helper -> worker -> service execution

**Analog files**

- `mibo-media-server/internal/library/enrichment.go`
- `mibo-media-server/internal/worker/worker.go`
- `mibo-media-server/internal/probe/service.go`

**Use for Phase 14**

- Add a new inventory-file probe job kind instead of probing inline in scan code.
- Keep ffprobe parsing in `internal/probe`; only the payload target changes.

## Concrete Source Snippets

### Scan loop currently fans out queue work only after traversal

From `internal/library/scan_run.go`:

```go
if _, err := s.QueueLibrarySearchReindex(ctx, record.ID, rootPath); err != nil {
    return err
}
if _, err := s.QueueCatalogLibraryProjectionRefresh(ctx, record.ID, rootPath); err != nil {
    return err
}
```

### Inventory file upsert already gives an idempotent file boundary

From `internal/inventory/service.go`:

```go
err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
    Columns:   []clause.Column{{Name: "storage_provider"}, {Name: "storage_path"}},
    DoUpdates: clause.AssignmentColumns([]string{"library_id", "stable_identity_key", "hashes_json", "size_bytes", "modified_at", "container", "status", "updated_at"}),
}).Create(&file).Error
```

### Multi-episode linking is already a first-class inventory concept

From `internal/inventory/service_test.go`:

```go
if _, err := inventorySvc.LinkAssetToItem(ctx, LinkAssetItemInput{AssetID: asset.ID, ItemID: first.ID, Role: AssetItemRoleMultiEpisodePart, SegmentIndex: 1}); err != nil {
    t.Fatalf("link first episode: %v", err)
}
if _, err := inventorySvc.LinkAssetToItem(ctx, LinkAssetItemInput{AssetID: asset.ID, ItemID: second.ID, Role: AssetItemRoleMultiEpisodePart, SegmentIndex: 2}); err != nil {
    t.Fatalf("link second episode: %v", err)
}
```

### Catalog projection refresh already rebuilds availability-facing search docs

From `internal/catalog/projections.go`:

```go
docs = append(docs, database.CatalogSearchDocument{
    ItemID:             item.ID,
    LibraryID:          item.LibraryID,
    ItemType:           normalizeCatalogType(item.Type),
    Title:              strings.TrimSpace(item.Title),
    AvailabilityStatus: normalizeAvailabilityStatus(item.AvailabilityStatus),
    UpdatedAt:          updatedAt,
})
```

### Probe service already knows how to parse ffprobe into normalized fields

From `internal/probe/service.go`:

```go
updates, runtimeSeconds, err := buildProbeUpdates(parsed)
if err != nil {
    return s.markProbeError(ctx, file.ID, err)
}
```

## Phase-14 Guardrails

- Do not create new `MediaItem` / `MediaFile` rows from `RunSyncLibrary` or
  `RunTargetedRefresh`.
- Do not key episode catalog rows by raw file path.
- Do not delete governed `catalog_items` when a file disappears; only availability/status may
  change.
- Do not probe inline in the scan loop; keep probe work in queued worker execution.
- Do not write arbitrary large blobs into scanner-created `metadata_sources`; keep local evidence
  payloads compact and allowlisted.
