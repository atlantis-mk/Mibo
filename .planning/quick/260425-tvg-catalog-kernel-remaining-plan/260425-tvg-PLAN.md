# Quick Task 260425-tvg: 剧集元数据治理剩余实现计划

## Plan Summary

基于已完成的 Phase A 新 catalog kernel，后续实现按“契约与回填 → 扫描写入 → 元数据治理 → API/播放切读 → 前端迁移 → 治理 UI → 旧模型收口”的顺序推进。核心原则是先可回填、再切读、再切写、最后移除旧模型，避免一次性大爆炸重构。

## Phase 1：内核契约与迁移护栏

### 目标

确定新 catalog kernel 的最终读写契约，补齐最小生产约束，给后续阶段提供稳定边界。

### 范围

后端模型约束、DTO 契约、迁移状态、投影更新机制；不改 UI 主流程。

### 任务

- 定义后端 catalog DTO：`CatalogListItem`、`CatalogItemDetail`、`CatalogSeasonDetail`、`CatalogEpisodeDetail`、`CatalogAssetDetail`、`CatalogGovernanceWorkspace`。
- 明确 item 类型语义：`series`、`season`、`episode`、`movie`、`extra`，不再用旧前端的 `show` 作为数据库类型。
- 明确资产语义：一个 `media_asset` 表示一个可播放版本或额外资源，一个 `asset_item` 表示资产与目录项的关系。
- 增加 catalog 投影更新入口：更新 item、asset、metadata、progress 后统一刷新 `item_rollups` 与 `catalog_search_documents`。
- 给迁移过程增加系统状态或 setting：记录 `catalog_backfill_completed_at`、`catalog_read_enabled`、`legacy_cleanup_completed_at`。
- 补齐关键唯一约束与索引设计，优先覆盖 `catalog_items` 层级查询、`asset_items` item 查 asset、`inventory_files` storage path 去重、`catalog_search_documents` 搜索列表。
- 明确旧表冻结策略：迁移前旧表仍是主读，切换后旧表只做兼容回填，不再作为新写入口。

### 依赖

- 已完成 Phase A schema。
- 当前 `internal/catalog/service.go` 与 `internal/inventory/service.go` 可作为写入基础，但需要扩展查询与投影能力。

### 验证标准

- `go test ./internal/database ./internal/catalog ./internal/inventory` 通过。
- 新约束不会破坏现有测试数据库 AutoMigrate。
- 能在空库启动，能在已有旧 `media_items` / `media_files` 的库上启动。

### 风险

- GORM AutoMigrate 对复杂外键、部分唯一索引支持有限，必要时需要显式迁移 SQL。
- SQLite 与 Postgres 索引能力不同，唯一约束要避免 NULL 行为差异导致重复 season/episode。

## Phase 2：旧模型回填到新内核

### 目标

把现有 `media_items` / `media_files` 数据完整回填为 `catalog_items` / `media_assets` / `asset_items` / `inventory_files` / `asset_files`，为切读做准备。

### 范围

一次性迁移服务、可重复运行的 backfill、数据一致性报告。

### 任务

- 新增 `internal/catalog/migration.go` 或放入 `internal/library` 的迁移子流程，负责 legacy 到 catalog 的幂等回填。
- 将旧 movie `MediaItem` 映射为 `catalog_items(type=movie)`。
- 将旧 episode `MediaItem` 按 `series_title` / `external_id` / library 分组生成 `series`，按 `season_number` 生成 `season`，再生成 `episode`。
- 将旧 `MediaFile` 映射为 `inventory_files`，保持 storage path、stable identity、hash、probe 信息。
- 为每个可播放旧条目创建 `media_assets(asset_type=main)`，通过 `asset_files` 链接文件，通过 `asset_items` 链接 catalog item。
- 将旧 `poster_url`、`backdrop_url`、`logo_url` 映射到 `item_images`，并设置 selected。
- 将旧 `metadata_provider`、`external_id`、`metadata_confidence` 映射到 `catalog_external_ids` 与 `metadata_sources`。
- 将旧 `match_status` 映射为 `governance_status`：`matched`、`pending`、`unmatched`、`manual`。
- 将旧进度 `playback_progress` 暂时双写或回填到 `user_item_data`，保留旧表直到播放和 UI 完成切换。
- 生成迁移报告：总数、成功数、跳过数、冲突数、孤儿文件数、重复 episode 数。

### 依赖

- Phase 1 的 DTO 和约束。
- 旧模型仍存在，且旧扫描不应在迁移中并发写入同一库。

### 验证标准

- 构造旧 movie、series、multi-season、missing metadata、multi-file 版本测试，回填后 catalog 层级正确。
- 重复执行回填不会产生重复 `catalog_items`、`media_assets`、`asset_items`。
- 回填后每个旧可播放 `MediaItem` 至少对应一个可播放 catalog item asset。
- `go test ./...` 通过。

### 风险

- 旧 episode 只有弱 `series_title` 时可能误合并，需要冲突报告进入治理队列。
- 旧 `external_id` 语义混杂 movie/tv，需要解析 provider type，不能盲目全写为 series ID。

## Phase 3：扫描器重建为 catalog 写入

### 目标

扫描结果直接写入新 catalog kernel，不再创建旧 `MediaItem` / `MediaFile`。

### 范围

`internal/library/scan*`、`internal/inventory`、worker scan jobs、probe job payload。

### 任务

- 保留现有路径分类能力，但输出新的 scan artifact：library、storage object、normalized title、series title、season number、episode number、asset hints。
- 扫描文件时先 upsert `inventory_files`，再 upsert 或复用 `media_assets`，再写 `asset_files`。
- 对 movie 文件生成或复用 `catalog_items(type=movie)`，并通过 `asset_items(role=primary)` 链接。
- 对 episode 文件生成或复用 `series`、`season`、`episode` 三级目录项，episode 先以本地证据创建 pending 状态。
- 支持多集文件：一个 asset 链接多个 episode，`asset_items.segment_index` 表示顺序，必要时记录 start/end 秒。
- 支持同一 episode 多版本：多个 `media_assets(asset_type=version|main)` 链接同一 episode，按质量、路径、probe 信息排序。
- 扫描删除时不删除 catalog 元数据，只把 `inventory_files` / `media_assets` 标记不可用，并更新 item `availability_status`。
- 扫描完成后排队 catalog projection jobs：rollup、search document、metadata matching。
- 旧 `upsertMediaItem` / `upsertMediaFile` 改为 legacy fallback，最终不再由 scan job 调用。

### 依赖

- Phase 2 的幂等 upsert 策略。
- `internal/inventory/service.go` 需要扩展查询与复用资产能力。
- `internal/library/scan_reconcile.go` 的 stable identity 逻辑需要迁移到 `inventory_files`。

### 验证标准

- 本地 demo library 扫描后只有新表增长，旧 `media_items` / `media_files` 不再新增。
- episode 文件生成 series、season、episode，`asset_items` 能从 episode 查到资产。
- 重扫、改名、移动、删除文件后资产可用状态正确。
- `go test ./internal/library -run Scan` 和 worker scan 测试更新后通过。

### 风险

- 资产复用规则过弱会导致重复版本，过强会误合并不同文件。
- 删除策略若直接软删 item，会丢失 provider metadata；必须只更新可用性。

## Phase 4：元数据引擎重建为 series 级治理

### 目标

剧集按 series 级匹配 TMDB/Provider，并由 provider 生成 season/episode catalog 目录项。

### 范围

`internal/metadata`、metadata jobs、catalog field state、source evidence、image/person/tag 写入。

### 任务

- 新增 `MatchCatalogItem(ctx, itemID)`，旧 `MatchItem(mediaItemID)` 只保留迁移期 wrapper。
- 对 `series` 执行 provider 搜索，按 title、year、library language、路径证据计算 confidence。
- 对 `episode` 的匹配改为先找到 root series，再用 series external ID 拉取 season/episode detail。
- 成功匹配 series 后写 `catalog_external_ids(provider=tmdb, provider_type=tv, external_id=...)`。
- 记录 provider payload 到 `metadata_sources`，不要只覆盖 catalog item 字段。
- 通过 `metadata_field_states` 写 canonical 字段，尊重 `is_locked`，自动字段不覆盖锁定字段。
- 拉取 TMDB series detail 后生成或更新 `season` catalog item。
- 拉取 season episodes 后生成或更新 `episode` catalog item，包括 missing/unaired/local available 状态。
- 将本地扫描生成的 episode 与 provider 生成 episode 通过 season/episode number 合并，保留 asset link。
- 写入 `item_images` 候选，按 provider、language、分辨率、类型排序，只设置默认 selected，不删除用户选择。
- 写入 people、tags、ratings、runtime、air date 等规范化字段。
- 区分治理状态：高置信自动 `matched`，低置信 `needs_review`，找不到 `unmatched`，用户确认 `manual` 或 `locked`。
- 元数据 refetch 只刷新 source evidence 和未锁字段，不破坏手动治理。

### 依赖

- Phase 3 需要产生稳定 series/season/episode 层级。
- Phase 1 的字段锁与 source evidence 契约。
- 现有 `service_tv.go`、`service_tmdb.go` 可复用 provider client。

### 验证标准

- 对一个只有本地 S01E01 的 series，匹配后能生成完整 season/episode 目录，包括 missing episode。
- 锁定 title 后 refetch 不覆盖 title，但仍更新 source evidence。
- 同一 TMDB series 重复匹配不会重复 season/episode。
- Provider 不可用时 job 失败可重试，不留下半写脏状态。
- `go test ./internal/metadata ./internal/catalog` 通过。

### 风险

- TMDB season numbering 与本地 special season、absolute episode 可能不一致，需要治理队列暴露冲突。
- Provider 生成 missing episodes 后列表数量上升，UI 和滚动性能要同步考虑。

## Phase 5：API 切换到新内核

### 目标

现有媒体列表、详情、series seasons、治理、播放相关 API 全部从 catalog 读取。

### 范围

`internal/httpapi`、`internal/library` 查询层、`internal/search`、`internal/progress`。

### 任务

- 将 `GET /api/v1/libraries/{id}/items` 改为读取 `catalog_items` 投影，默认返回 movie 与 series，不直接平铺全部 episode。
- 新增或切换 `GET /api/v1/items/{id}` 返回 `CatalogItemDetail`，包含 selected images、assets、children summary、external IDs、progress。
- 新增或切换 `GET /api/v1/series/{id}/seasons`，从 catalog season/episode 层级读取，不再依赖 `/media-items/{id}/series-episodes`。
- 治理接口切到 catalog：`GET /api/v1/items/{id}/governance`、`PUT /api/v1/items/{id}/governance/fields`、`POST /api/v1/items/{id}/metadata/search`、`POST /api/v1/items/{id}/metadata/apply`、`POST /api/v1/items/{id}/metadata/refetch`。
- 旧 `/media-items/{id}` 路由在迁移期可以短期返回 410 或桥接到 catalog，但前端完成后应移除依赖。
- `handleHomeDiscovery`、`recently-added`、`latest-by-library` 切换到 catalog + `user_item_data`。
- `search.Service` 改为使用 `catalog_search_documents`，结果返回 CatalogItem。
- `progress.Service` 改为按 `item_id` / `asset_id` 写 `user_item_data`，不再写旧 `playback_progress`。
- API 响应不要直接暴露 GORM model，使用明确 DTO，避免旧字段泄漏。

### 依赖

- Phase 2 回填完成。
- Phase 4 metadata 至少能生成 series/season/episode 基础结构。
- 前端 Phase 7 可并行，但最终切换需要 API contract 固定。

### 验证标准

- `/api/v1/libraries/{id}/items` 在旧 demo 数据回填后能返回同等或更好的列表。
- `/api/v1/series/{id}/seasons` 返回 season 和 episode，包含 available/missing 状态。
- 治理接口能读取 field locks、source evidence、images、asset links。
- `go test ./internal/httpapi ./internal/search ./internal/progress` 通过。

### 风险

- 如果直接改变现有 `MediaItem` JSON schema，会造成前端同时大面积失败；需要同一阶段内同步前端类型切换。
- home/search/list 使用 series 聚合后数量会变化，旧测试断言需要按新产品语义更新。

## Phase 6：播放链路切换为 item -> asset/version -> URL

### 目标

播放不再从 `MediaItem` 选 `MediaFile`，而是从 catalog item 选择 asset/version，再解析 asset file 生成播放 URL。

### 范围

`internal/playback`、播放相关 handler、HLS、stream endpoint、progress。

### 任务

- 修改 `PlaybackRequest`：主键为 `ItemID`，可选 `AssetID`、`PreferredQuality`、`ClientProfile`、`AllowHLSFallback`。
- `GetPlaybackSource` 加载 `catalog_items`，通过 `asset_items` 查可用 `media_assets`。
- 资产排序规则：用户指定 asset 优先，其次 available、main、version quality、probe success、latest added。
- 通过 `asset_files` 找 source file，通过 `inventory_files` 解析 storage provider。
- `PlaybackSource` 返回 `item_id`、`asset_id`、`file_id`、title、quality、edition、streams、decision。
- `GET /api/v1/items/{id}/playback` 返回选择后的播放源。
- `GET /api/v1/assets/{id}/link` 返回资产级链接，底层可继续使用 inventory file stream。
- HLS 与 direct stream 改为 inventory file 或 asset file ID，不再使用旧 `media_files.id`。
- 播放进度更新携带 `item_id` 和 `asset_id`，写入 `user_item_data`。
- UI 播放页可传 `asset_id`，未传时由后端选择默认版本。

### 依赖

- Phase 3 的 asset/file/link 数据完整。
- Phase 5 的 API item ID 已稳定。
- Probe 需要迁移到 `inventory_files` / `media_streams` 或提供过渡映射。

### 验证标准

- movie item 有一个 asset 时可直接播放。
- episode item 有多个版本时能按指定 `asset_id` 播放。
- 多集 asset 能解析并返回正确 item/segment 信息。
- 文件缺失时返回不可播放 decision，而不是 500。
- `go test ./internal/playback ./internal/httpapi -run Playback` 通过。

### 风险

- HLS 现有实现绑定 `media_files`，迁移时容易遗漏 segment 缓存路径。
- 多集文件 segment start/end 若缺失，只能播放整文件，需要 UI 明示。

## Phase 7：前端类型与搜索/列表/详情切换

### 目标

UI 从旧 `MediaItem` 切换到 `CatalogItem`，列表、搜索、详情、播放入口全部使用新 API。

### 范围

`web/src/lib/mibo-api.ts`、`mibo-query.ts`、`features/media`、home/search/library routes、播放页。

### 任务

- 在 `mibo-api.ts` 新增 `CatalogItem`、`CatalogItemDetail`、`CatalogAsset`、`CatalogSeasonWithEpisodes`、`GovernanceWorkspace` 类型。
- 替换 `MediaItem` 作为页面主类型，保留 legacy 类型只用于迁移期测试或删除。
- `mibo-query.ts` query key 从 `mediaItemDetail` 切到 `catalogItemDetail`。
- 首页 latest、continue watching、search result、library items 全部渲染 CatalogItem。
- 详情页改为读取 `CatalogItemDetail`，图片来自 selected `item_images`，文件列表来自 `assets`。
- 剧集详情调用 `/series/{id}/seasons`，以 season/episode catalog item 展示 available/missing/unaired 状态。
- 播放入口从 item 选择 asset，支持版本选择、质量展示、缺失资产禁播。
- 路由可以继续使用 `/media/$id` 作为用户可见路径，但内部 ID 语义改为 catalog item ID。
- 删除或改造 `media-presentation.ts` 中依赖 `series_title`、`match_status`、`source_path` 的旧展示逻辑。

### 依赖

- Phase 5 API contract。
- Phase 6 playback source contract。

### 验证标准

- `pnpm typecheck` 通过。
- `pnpm build` 通过。
- 首页、library、search、详情、播放页手动跑通。
- 旧 `MediaItem` 类型不再被主流程 import。

### 风险

- 旧 UI 假设每个 item 都有 `source_path` 和 `files[0]`，新 series/season/missing episode 没有本地资产，需要大量空状态处理。
- route path 不改但 ID 语义改变，浏览器旧收藏链接可能失效；如果需要兼容，单独做 legacy redirect。

## Phase 8：元数据治理 UI 重建

### 目标

治理 UI 从“编辑旧 MediaItem 字段”升级为“管理 catalog item 的字段锁、来源证据、图片选择、资产链接”。

### 范围

`web/src/features/metadata-governance`、settings metadata routes、API client。

### 任务

- 治理详情页读取 `GET /api/v1/items/{id}/governance`。
- 页面分区：canonical 字段、字段锁、来源证据、图片候选、外部 ID、资产链接、匹配/重抓操作。
- 每个字段显示当前值、来源、confidence、锁定状态、最近编辑用户/时间。
- 支持锁定/解锁字段，保存时调用 field-level API，不再一次性覆盖整条 item。
- 来源证据面板展示 `metadata_sources` payload 摘要，支持按 provider/language/source time 筛选。
- 图片选择面板展示 `item_images`，支持 poster/backdrop/logo/still 选择，选择后只更新 selected，不删除候选。
- 资产链接面板展示 `asset_items` 和 `media_assets`，支持查看文件、质量、probe 状态、重新探测、解除错误链接。
- Series 治理页支持对子 season/episode 的生成状态、缺失状态、冲突状态做批量 review。
- 候选搜索/应用改为 series-first，episode 页面提示“匹配来源继承自 series”。

### 依赖

- Phase 4 governance 数据完整。
- Phase 5 governance API。
- Phase 7 前端 CatalogItem 类型。

### 验证标准

- 字段锁定后 refetch 不覆盖 UI 中该字段。
- 图片选择刷新详情页后生效。
- 资产链接状态能解释一个 episode 为什么可播或不可播。
- `pnpm typecheck` 与 `pnpm build` 通过。

### 风险

- 治理 UI 容易过度复杂，第一版应只做字段锁、证据、图片、资产四个核心面板。
- Payload JSON 直接展示会不可读，需要摘要化，原始 JSON 可放折叠区。

## Phase 9：旧模型替换与生产级补强

### 目标

完成从并行 schema 到新内核的收口，补齐外键、迁移、索引、性能与清理策略。

### 范围

数据库约束、旧代码删除、投影一致性、性能测试。

### 任务

- 确认所有新写路径不再写 `media_items` / `media_files` / `playback_progress` / `search_documents`。
- 删除或隔离旧 `MediaItem` query、metadata match、search index、playback select 代码。
- 保留只读 legacy migration 命令一段时间，普通运行路径不再调用。
- 增加外键：catalog child parent、external IDs item、metadata source item、field state item/source、images item/source、asset items item/asset、asset files asset/file、user item data item/asset。
- 增加唯一约束：external provider identity、metadata field item/field、asset item role segment、asset file part、inventory storage path；selected image 每类唯一优先通过业务逻辑保证。
- 增加性能索引：`catalog_items(library_id,type,availability_status,sort_key)`、`catalog_items(parent_id,parent_index_number,index_number)`、`catalog_items(root_id,type,parent_index_number,index_number)`、`asset_items(item_id,role)`、`asset_files(asset_id,part_index)`、`inventory_files(library_id,status,storage_path)`、`catalog_search_documents(library_id,item_type,availability_status,title)`。
- 增加 projection consistency checker：扫描 item/assets/metadata 后检测 rollup、availability、search document 是否滞后。
- 增加 backfill command 或 admin job：重建 `item_rollups`、重建 `catalog_search_documents`、重算 availability。
- 更新测试基线，删除旧 MediaItem 断言。
- 更新 README 或 agent notes 中新内核运行与排障说明。

### 依赖

- Phase 5、6、7 主流程已切换。
- 迁移报告显示生产数据可完整映射。

### 验证标准

- 全量 `go test ./...` 通过。
- `web/ pnpm typecheck` 与 `pnpm build` 通过。
- 空库启动、旧库迁移启动、新库重复启动都通过。
- 大库列表、series seasons、search、governance workspace 查询都有可接受延迟。
- 旧表清理前有备份和迁移完成标记。

### 风险

- 外键在已有脏数据上可能无法添加，需要先跑 consistency repair。
- 删除旧模型前若仍有隐藏测试或前端路径依赖，会造成运行时 404 或空列表。

## 推荐执行顺序

1. Phase 1 到 Phase 2 先做，保证可回滚、可验证。
2. Phase 3 与 Phase 4 组成后端写入内核重建。
3. Phase 5 与 Phase 6 完成后端读和播放切换。
4. Phase 7 与 Phase 8 完成前端切换和治理体验。
5. Phase 9 最后收口旧模型和生产约束。
