# Quick Task 260425-tvg Context

## User Intent

基于已完成的 Phase A 新 catalog kernel，生成剩余剧集元数据治理实现计划。

## Locked Scope

- 扫描器重建：扫描结果写入 `catalog_items` / `media_assets` / `asset_items`。
- 元数据引擎重建：按 series 级别匹配 TMDB/Provider，并生成 season/episode 目录项。
- API 切换：`/items`、`/series/{id}/seasons`、治理接口等接到新内核。
- 播放链路切换：从 item 选择 asset/version，再生成播放 URL。
- 搜索/列表/详情改造：UI 从旧 `MediaItem` 切换到 `CatalogItem`。
- 元数据治理 UI 重建：字段锁、来源证据、图片选择、资产链接。
- 旧模型迁移/替换策略：从并行 schema 过渡到新内核。
- 生产级约束补强：外键、唯一约束、回填/迁移、性能索引、投影更新。

## Deliverable Contract

输出必须是可执行的分阶段实现计划，每阶段包含目标、范围、任务、依赖、验证标准和风险。

## Constraints

- Phase A 已完成基础 schema、`catalog.Service`、`inventory.Service` 与测试，但主业务链路仍在旧 `MediaItem` / `MediaFile` 上。
- 计划必须避免一次性大爆炸切换，优先支持可回填、可验证、可回滚。
- 不直接实现代码，只生成后续实现计划。
