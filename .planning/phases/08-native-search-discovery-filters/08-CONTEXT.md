# Phase 8: Native Search & Discovery Filters - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning

<domain>
## Phase Boundary

用户可以在 Mibo 内通过一个原生 discovery contract 完成搜索与浏览筛选：支持按标题、原始标题、演员、导演搜索；结果中明确区分电影与剧集；展示命中高亮；提供最近搜索历史；并在搜索结果与浏览结果之间共享类型、年份、地区、评分、已看状态与排序能力。

本阶段聚焦“统一查询契约 + 原生搜索入口 + 共享筛选语义”，不扩展为外部搜索中间件接入，不引入高级查询语言，也不把未来需求 `FLTR-07`（媒体库筛选）和 `FLTR-08`（分辨率筛选）提前并入当前范围。

</domain>

<decisions>
## Implementation Decisions

### 搜索入口与界面承载
- **D-01:** Phase 8 的原生搜索采用全局统一入口，而不是只放在单个库页面内部。
- **D-02:** 全局搜索入口优先复用现有应用壳层能力，结合当前 `web/src/routes/_app.tsx` 下的侧边栏 / 壳层结构落位，而不是额外引入一套平行导航体系。
- **D-03:** 浏览页必须继续保持稳定浏览态；搜索入口增强整个产品的发现能力，但不能破坏现有 library browse 的可导航性与浏览稳定性。

### 搜索历史
- **D-04:** 最近搜索历史采用“执行后自动保存”，不要求用户额外点击保存。
- **D-05:** 用户重新打开搜索时，应能看到最近搜索历史，并支持一键重跑既有查询。
- **D-06:** 搜索历史属于 Phase 8 的正式交付内容，应作为原生搜索流程的一部分持久化，而不是仅保留页面内临时状态。

### 已看状态语义
- **D-07:** 已看状态筛选采用三态语义：`未看`、`观看中`、`已看`。
- **D-08:** 三态定义应直接对齐当前后端进度语义：`watched = false && position_seconds = 0` 视为未看，`watched = false && position_seconds > 0` 视为观看中，`watched = true` 视为已看。
- **D-09:** 搜索与浏览必须共享同一套已看状态定义，不能由不同页面分别解释 partial progress。

### 统一 discovery contract
- **D-10:** 搜索与浏览必须基于同一个后端 discovery contract，而不是分别演化出两套筛选与排序语义。
- **D-11:** 当前已存在的 browse 能力（类型、年份、排序、电影/剧集聚合）应被吸收进共享 contract，后续 search 结果与 browse 结果保持一致的媒体分组与类型区分规则。
- **D-12:** 统一排序能力至少覆盖当前已落地排序维度 `recent`、`title`、`year`、`watch_status`，并在搜索与浏览两侧保持同义。

### 范围边界
- **D-13:** 本阶段坚持产品内原生实现，继续遵守“不接 Elasticsearch / Meilisearch / 向量检索等外部中间件”的既有约束。
- **D-14:** 本阶段锁定的共享筛选范围是类型、年份、地区、评分、已看状态与排序；媒体库筛选、分辨率筛选继续留在未来需求，不提前纳入 Phase 8。
- **D-15:** 结果高亮、电影/剧集区分、最近搜索历史都属于本阶段必须规划的正式能力，而不是“有余力再做”的 UI 增强项。

### the agent's Discretion
- 全局搜索入口的具体交互形态可以由后续 agent 决定，例如侧边栏输入、命令式弹层、独立搜索页，前提是不偏离“全局统一入口”这个已锁定方向。
- 搜索结果与 browse 结果的视觉密度、卡片布局、高亮样式、空状态与加载状态由后续规划决定，但必须兼容当前 `home` / `library` 的既有视觉语言。
- 搜索历史显示条数、去重策略、清空入口和排序细节可由后续规划决定。
- 地区与评分字段的最终数据建模方式可由后续 research / planning 决定，但必须满足统一 contract，而不是让前端拼接临时逻辑。

</decisions>

<specifics>
## Specific Ideas

- 当前代码里已经有全局应用壳层、侧边栏和占位搜索框，因此第一版搜索入口应优先从全局壳层承载，而不是只在 `library` 页塞一个局部输入框。
- 既有 `library browse` 已经有电影/剧集聚合、年份筛选和排序语义，Phase 8 更像是在现有 browse contract 上升级为完整的原生 discovery contract，而不是另起一套 search-only 模型。
- 已看状态不应只暴露“已看 / 未看”二态，因为当前后端进度语义已经清楚区分 `观看中`，且该状态在产品里已有实际用户价值。
- 搜索历史应该是“重新打开即能继续找”的体验，不是收藏夹式的手动保存模型。

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase definition
- `.planning/ROADMAP.md` — Phase 8 的目标、依赖、成功标准，以及“shared discovery contract”这一阶段锚点。
- `.planning/REQUIREMENTS.md` — `SRCH-01` through `SRCH-08` 与 `FLTR-01` through `FLTR-06` 的锁定需求，同时明确 `FLTR-07` / `FLTR-08` 仍属未来范围。
- `.planning/PROJECT.md` — v2 里对原生搜索/筛选的定位，以及“不接任何外部中间件”的产品约束。
- `.planning/STATE.md` — 当前阶段状态与两个关键关注点：SQLite FTS5 readiness、watched-state 语义锁定。

### Discovery architecture constraints
- `.planning/research/SUMMARY.md` — 项目级研究总结，明确 discovery 应保持 app-owned、SQL-native、shared query contract。
- `.planning/research/STACK.md` — 搜索/筛选基础设施建议：SQLite FTS5、Postgres `tsvector` + GIN、显式 SQL migration、`internal/search` 作为统一抽象层。
- `.planning/research/ARCHITECTURE.md` — 共享 discovery contract、search projection、过滤字段来源、API 扩展方向与迁移约束。
- `.planning/research/PITFALLS.md` — 明确警告不要让 browse/search 演化为两套 filter 语义，尤其是 watched-state 与 grouped-show 规则。

### Live backend anchors
- `mibo-media-server/internal/search/service.go` — 当前 search service 仍是 stub，表明 Phase 8 需要真正落地搜索能力而不是只接前端 UI。
- `mibo-media-server/internal/library/query.go` — 当前 browse 输入模型、类型筛选与排序枚举，是统一 discovery contract 的直接起点。
- `mibo-media-server/internal/library/query_browse.go` — 当前 browse 的过滤、排序、show 聚合与 watch-status 排序实现细节。
- `mibo-media-server/internal/progress/service.go` — 当前已看 / 观看中 / 完成判定逻辑，是三态 watched filter 的行为基础。
- `mibo-media-server/internal/database/models.go` — 当前可搜索字段、`GenresJSON` / `CastJSON` / `DirectorsJSON`、年份字段，以及地区/评分尚未显式建模的事实约束。
- `mibo-media-server/internal/httpapi/router.go` — 当前尚无 Phase 8 的搜索/历史 API 路由，只有 metadata search 路由，说明新 discovery route 需要新增。

### Live frontend anchors
- `web/src/routes/_app.tsx` — 当前应用壳层、`SidebarProvider` 与全局承载位置，可作为全局搜索入口接入点。
- `web/src/components/app-sidebar.tsx` — 当前侧边栏结构与 placeholder 搜索框位置，适合承接全局统一搜索入口。
- `web/src/components/search-form.tsx` — 现有占位搜索组件，可作为真实搜索入口的替换起点。
- `web/src/features/library/index.tsx` — 当前 browse 页面，只支持固定 recent 排序与静态网格，是共享 discovery UI 的现有基线。
- `web/src/features/home/index.tsx` — 当前首页 discovery 视觉语言，可约束结果展示不能完全偏离现有产品风格。
- `web/src/lib/mibo-api.ts` — 当前 browse API client 契约与缺失的 search/history client 方法。
- `web/src/lib/mibo-query.ts` — 当前 query key 组织方式，后续 search/discovery query key 应沿用该模式扩展。

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/routes/_app.tsx` + `web/src/components/app-sidebar.tsx`: 已有全局壳层与侧边栏，可复用为全局统一搜索入口的承载层。
- `web/src/components/search-form.tsx`: 已有占位搜索 UI，可作为 Phase 8 的真实搜索交互起点，而不是从零再造一个无关组件。
- `web/src/components/app-top-bar.tsx`: 当前 `home` / `library` 共用的顶部栏模式，可继续承载搜索结果页或筛选入口的页面级操作区。
- `web/src/lib/mibo-api.ts`: 现有 typed API client 已覆盖 browse / media / home，新增 search/history/discovery contract 时应继续沿用此模式。
- `web/src/lib/mibo-query.ts`: 现有 TanStack Query key 组织模式已经建立，Phase 8 应在此追加 search/discovery 相关 key。
- `mibo-media-server/internal/library/query.go` + `query_browse.go`: 已经实现 browse 级筛选、排序与 show 聚合，是统一 discovery contract 的后端基础。

### Established Patterns
- 现有前端数据流以 typed client + TanStack Query 为主，Phase 8 不应退回页面内 ad hoc fetch。
- 现有后端 browse 逻辑已经在服务端处理 show 聚合与 watch-status 排序，说明 grouped discovery semantics 应继续由服务端统一计算。
- `progress.Service` 的 watched 语义已经是用户级状态，而 metadata 字段大多是全局状态，因此 watched-state filter 必须被视为 user-scoped filter，而不是静态元数据字段。
- 当前搜索服务虽然已预留 `internal/search` 位置，但没有真实实现；说明后续规划需要覆盖存储、索引、API、前端接入的端到端闭环。

### Integration Points
- 后端需要在 `internal/search`、`internal/httpapi/router.go`、以及现有 browse 查询层之间建立统一 discovery contract。
- 前端需要把全局壳层搜索入口、搜索历史、搜索结果页/态，与 `library` 浏览页的共享筛选能力串起来。
- watched-state 筛选需要与 `progress` 语义对齐，并在搜索和 browse 两侧返回一致结果。

### Constraints
- 当前 `MediaItem` 模型没有显式地区和评分标量字段，这意味着 FLTR-03 / FLTR-04 很可能需要 projection 或额外字段建模，不能假设现成可查。
- 当前还没有原生搜索 API 和搜索历史 API，且 `internal/search/service.go` 只是 stub，Phase 8 不是简单前端拼装任务。
- 现有 browse 仅支持 `type`、`year`、`sort`，后续扩展必须避免把 search/browse 各自做一层参数翻译，避免 contract 分裂。

</code_context>

<deferred>
## Deferred Ideas

- `FLTR-07` 媒体库筛选。
- `FLTR-08` 分辨率筛选。
- Elasticsearch、Meilisearch、向量检索或其他外部搜索中间件。
- 高级查询语言、复杂布尔条件构造器、面向 power-user 的 query DSL。

</deferred>

---

*Phase: 08-native-search-discovery-filters*
*Context gathered: 2026-04-24*
