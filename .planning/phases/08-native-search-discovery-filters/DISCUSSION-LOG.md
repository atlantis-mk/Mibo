# Phase 8 Discussion Log

**Phase:** 08 - Native Search & Discovery Filters
**Date:** 2026-04-24
**Mode:** discuss
**Status:** Decisions locked for planning

## Sources Reviewed Before Questions

- Workflow / template:
  - `/root/.config/opencode/get-shit-done/workflows/discuss-phase.md`
  - `/root/.config/opencode/get-shit-done/templates/context.md`
- Planning context:
  - `.planning/PROJECT.md`
  - `.planning/REQUIREMENTS.md`
  - `.planning/STATE.md`
  - `.planning/ROADMAP.md`
  - `.planning/phases/07-metadata-governance-matching/07-CONTEXT.md`
- Code / implementation anchors:
  - `mibo-media-server/internal/search/service.go`
  - `mibo-media-server/internal/library/query.go`
  - `mibo-media-server/internal/library/query_browse.go`
  - `mibo-media-server/internal/database/models.go`
  - `mibo-media-server/internal/progress/service.go`
  - `mibo-media-server/internal/httpapi/router.go`
  - `web/src/routes/_app.tsx`
  - `web/src/components/app-sidebar.tsx`
  - `web/src/components/search-form.tsx`
  - `web/src/features/library/index.tsx`
  - `web/src/features/home/index.tsx`
  - `web/src/lib/mibo-api.ts`
  - `web/src/lib/mibo-query.ts`

## Pre-Question Synthesis

- Phase 8 的固定边界已经明确：要做的是一个 shared native discovery contract，而不是扩大成新的外部搜索系统。
- 已有 browse contract 只覆盖 `type`、`year`、`sort`，搜索服务仍是 stub，说明真正需要锁定的是“搜索入口位置、历史记录语义、watched-state 筛选语义”等实现方向。
- 已有产品与规划文档已经提前锁定以下事实，不应重复发问：
  - 搜索/筛选必须是 app-owned、product-native。
  - 不接外部搜索中间件。
  - `FLTR-07` library filter 与 `FLTR-08` resolution filter 不属于当前 Phase 8 范围。

## Questions Asked And Answers

### 1. 搜索入口
- Question: `Phase 8 的原生搜索，主要入口要放在哪里？`
- Answer: `全局统一入口（推荐）`
- Why this mattered:
  - 当前已有全局壳层与侧边栏结构。
  - 既有 retrospective 明确 library browse 需要保持稳定浏览态，不能为了搜索入口破坏浏览体验。

### 2. 最近搜索历史
- Question: `最近搜索历史，第一版希望怎么工作？`
- Answer: `执行后自动保存（推荐）`
- Why this mattered:
  - `SRCH-08` 与 roadmap success criteria 都要求 preserved search history。
  - 需要提前锁定是“自动保存”还是“手动收藏式保存”，否则 API 与前端状态设计会分叉。

### 3. 已看状态筛选
- Question: `Phase 8 里，用户可见的“已看状态”筛选要锁定成哪种语义？`
- Answer: `三态筛选（推荐）`
- Why this mattered:
  - 当前 `progress.Service` 已经天然区分未看、观看中、已看。
  - 如果只做二态，后续 search / browse / continue watching 容易出现语义不一致。

## Locked Outcomes

- 原生搜索主入口采用全局统一入口。
- 最近搜索历史采用执行后自动保存，并支持重新打开后一键重跑。
- 已看状态筛选采用未看 / 观看中 / 已看三态。
- 搜索与 browse 必须继续收敛到同一个后端 discovery contract。

## Out-Of-Scope Reminders Captured During Discussion

- 不接 Elasticsearch、Meilisearch 或其他外部搜索中间件。
- 不提前纳入 library filter 与 resolution filter。
- 不扩展为高级 query DSL 或复杂布尔查询构建器。

## Output Produced

- `.planning/phases/08-native-search-discovery-filters/08-CONTEXT.md`
