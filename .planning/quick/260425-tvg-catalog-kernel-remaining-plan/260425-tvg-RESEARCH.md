# Quick Task 260425-tvg Research

## Current State

- 新 catalog kernel 已作为并行 schema 落地，包含 `catalog_items`、`catalog_external_ids`、`metadata_sources`、`metadata_field_states`、`item_images`、`media_assets`、`asset_items`、`inventory_files`、`asset_files`、`media_streams`、`user_item_data`、`item_rollups`、`catalog_search_documents`。
- 现有扫描、元数据匹配、播放、搜索、API 和前端仍主要围绕旧 `MediaItem` / `MediaFile` 工作。
- 当前风险不是 schema 缺失，而是读写路径切换顺序、回填幂等性、播放资产选择语义、治理 UI 复杂度和旧模型收口。

## Recommended Migration Strategy

- 先冻结新内核契约，再做旧数据回填。
- 后端先切写入，再切读取；API 契约稳定后再切前端。
- 播放链路单独作为后端阶段，避免与列表/详情 UI 同时迁移。
- 治理 UI 最后接入完整 field state/source/image/asset 能力。
- 旧模型删除必须放在所有主流程切换之后，并先补 consistency checker 与 repair/backfill job。

## Main Pitfalls

- 旧 episode 的 series identity 弱，按 `series_title` 盲目合并会产生误合并。
- TMDB provider ID 当前语义混杂 movie/tv/episode，需要拆 provider type。
- 多集文件和多版本文件不能再被压平到单个 media item。
- Missing/unaired episode 会让列表数量与旧系统不同，UI 需要明确 availability 状态。
- GORM AutoMigrate 对复杂外键和部分索引有限，生产级约束可能需要显式迁移 SQL。
