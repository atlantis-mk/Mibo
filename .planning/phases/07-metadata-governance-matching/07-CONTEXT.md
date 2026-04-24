# Phase 7: Metadata Governance & Matching - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning

<domain>
## Phase Boundary

管理员可以对单个媒体条目的应用自有元数据进行人工治理，包括校正基础文本信息、替换海报/背景图、处理匹配候选，以及触发重新匹配和元数据重抓，让后续搜索、筛选和预告片等能力建立在更可靠的 Mibo 自有元数据之上。

本阶段聚焦单条目治理，不扩展为批量运营后台，也不改变既有 `OpenList -> mibo-media-server -> client` 系统边界。

</domain>

<decisions>
## Implementation Decisions

### 编辑入口
- **D-01:** 元数据治理使用独立管理页，而不是直接把完整编辑能力堆在现有媒体详情页内。
- **D-02:** 该独立管理页同时支持两个入口：从媒体详情页跳转进入当前条目的治理页，以及提供后台总入口供管理员进入治理工作流。

### 保存流程
- **D-03:** 单条目治理采用统一草稿保存模式。管理员在一个编辑会话里修改多类字段后，再统一提交保存。
- **D-04:** 管理页存在未保存改动时，离开页面必须先确认，不能静默丢弃，也不自动保存。

### 海报与演职信息
- **D-05:** 海报和背景图优先从候选结果中选择，不采用“手填图片 URL”作为主流程，也不把图片上传作为本期主路径。
- **D-06:** 演员、类型、季集基础信息整体采取“半自动为主”的治理方式。
- **D-07:** 人工编辑重点放在基础字段和必要修正上；更结构化的演职与季集内容主要通过重新匹配或元数据重抓更新，而不是做成完整内容运营后台。

### 匹配操作
- **D-08:** 匹配治理保留四个明确分开的动作：搜索候选、应用候选、重新匹配、元数据重抓。
- **D-09:** 搜索到候选后，必须先展示差异或预览，再允许管理员确认应用，不能默认直接覆盖。
- **D-10:** 重新匹配和元数据重抓采用后台任务式反馈，提交后显示已排队/处理中状态，并在完成后刷新结果。

### the agent's Discretion
- 独立管理页的具体信息架构、区块顺序、视觉布局和交互细节。
- 候选差异预览的具体呈现形式（表格、字段对比卡、分组摘要等）。
- 未保存变更提示的具体触发时机和文案。
- 后台总入口在导航中的具体位置与命名。

</decisions>

<specifics>
## Specific Ideas

- 治理能力不应只停留在当前详情页里已有的“重新匹配”按钮，而应升级成独立管理页。
- 独立管理页既要能从媒体详情进入，也要有后台总入口，兼顾“针对单条目修正”和“管理员治理工作流”。
- “半自动为主”意味着：基础文本和图片可控，但不要把演员/季集做成沉重的全量运营系统。
- 匹配类动作需要职责清晰，管理员能分辨“换候选”“重跑匹配”“重抓元数据”分别在做什么。

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase definition
- `.planning/ROADMAP.md` — Phase 7 goal, dependency, requirements mapping, and success criteria.
- `.planning/REQUIREMENTS.md` — `META-01` through `META-06`, which define the locked requirement surface for metadata editing, artwork replacement, season/episode basics, rematch, and metadata refetch.
- `.planning/PROJECT.md` — v2 milestone intent, product constraints, and the architectural rule that media business logic stays in `mibo-media-server`.
- `.planning/STATE.md` — current project state and the explicit note that metadata governance is the quality foundation for later discovery phases.

### Existing architecture and code shape
- `docs/media-architecture/improved-architecture.md` — prior architecture notes around `media_items` / `series` / `seasons` / `episodes`, useful as historical context for metadata ownership and matching flow.

### Live implementation anchors
- `mibo-media-server/internal/metadata/service.go` — existing TMDB candidate search, candidate apply, season/episode metadata lookup, and current metadata write shape.
- `mibo-media-server/internal/httpapi/router.go` — current metadata-related HTTP routes, including search/apply, rematch, metadata settings, and TV season/episode lookup endpoints.
- `mibo-media-server/internal/database/models.go` — persisted metadata fields already available on `MediaItem`, plus season/episode metadata cache tables.
- `mibo-media-server/internal/library/query.go` — current `MediaItemDetail` response shape consumed by the client.
- `mibo-media-server/internal/library/enrichment.go` — current forced rematch job enqueue behavior.
- `web-new/src/features/media/index.tsx` — current media detail page orchestration, including the existing rematch action.
- `web-new/src/features/media/components/standalone-media-detail.tsx` — current media detail presentation and action placement.
- `web-new/src/lib/mibo-api.ts` — current client contract; already exposes `rematchMediaItem`, but not manual edit or refetch APIs.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web-new/src/features/media/index.tsx`: already owns media detail data fetching, mutation wiring, and query invalidation; can be the natural source for “jump to manage” integration.
- `web-new/src/features/media/components/standalone-media-detail.tsx`: existing action row and metadata presentation can seed the future link into the dedicated governance page.
- `web-new/src/lib/mibo-api.ts`: the typed client already models media detail and rematch actions, so new governance APIs should extend this contract rather than introduce ad hoc fetch calls.
- `mibo-media-server/internal/metadata/service.go`: already contains candidate search/apply and TMDB season/episode lookup logic that planning should reuse instead of rebuilding.

### Established Patterns
- Backend long-running work is modeled as jobs handled by the worker, so rematch and metadata refetch should stay aligned with the jobs/worker flow instead of becoming synchronous heavy requests.
- Frontend state is already query-driven with React Query; governance mutations should fit the same invalidate-and-refresh pattern.
- Current metadata ownership lives on `database.MediaItem`, so manual governance should mutate app-owned metadata in Mibo rather than pushing changes into OpenList.

### Integration Points
- New admin governance UI should connect to the current media detail flow and the broader admin navigation surface in `web-new/`.
- New write APIs will extend the existing `/api/v1/media-items/{id}/...` contract family in `mibo-media-server/internal/httpapi/router.go`.
- Any refetch/rematch execution path should integrate with the existing job queue and worker dispatch model in `mibo-media-server/internal/library/enrichment.go` and related worker logic.

### Constraints
- The live frontend package is `web-new/`, while older planning/codebase docs still reference `web/`; downstream agents should trust the live package path when planning implementation work.
- There is already a rematch endpoint and manual search/apply backend support, but there is no manual metadata edit API or metadata refetch action yet.
- Image fields are stored as URLs today, so “pick from candidates” fits the current data model better than introducing a new upload/storage subsystem.

</code_context>

<deferred>
## Deferred Ideas

- 完整内容运营台式的深度演职/季集维护能力。
- 图片上传托管能力。
- 批量元数据治理或批量编辑工作流。

</deferred>

---

*Phase: 07-metadata-governance-matching*
*Context gathered: 2026-04-24*
