# Phase 9 Discussion Log

**Phase:** 09 - Trailer Discovery & Playback
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
  - `.planning/phases/08-native-search-discovery-filters/08-CONTEXT.md`
- Code / implementation anchors:
  - `mibo-media-server/internal/metadata/service.go`
  - `mibo-media-server/internal/metadata/service_tmdb.go`
  - `mibo-media-server/internal/httpapi/router.go`
  - `mibo-media-server/internal/library/enrichment.go`
  - `mibo-media-server/internal/library/query.go`
  - `mibo-media-server/internal/database/models.go`
  - `mibo-media-server/internal/worker/worker.go`
  - `web/src/features/media/index.tsx`
  - `web/src/features/media/components/standalone-media-detail.tsx`
  - `web/src/features/media/components/standalone-media-detail-hero.tsx`
  - `web/src/features/media/components/standalone-media-detail-specs.tsx`
  - `web/src/lib/mibo-api.ts`

## Pre-Question Synthesis

- Phase 9 的固定边界已经明确：只做 TMDB trailer 同步、详情页内发现入口、详情体验内播放，不扩展为外部源聚合或播放基础设施工程。
- 现有详情页已经存在一个 Hero 区的 `预告片` 占位按钮，但它没有任何接线，也不代表正式产品方向已经被锁定。
- 后端当前没有 trailer 字段、trailer contract 或 trailer job kind；TMDB detail 请求也还没有拉 `videos`，因此真正需要锁定的是“入口放哪、怎么播放、什么时候同步、怎么选片”。
- 已有产品与规划文档已经提前锁定以下事实，不应重复发问：
  - trailer 业务仍归 `mibo-media-server`，不能塞到 OpenList。
  - 外部源属于未来需求 `TRLR-05`。
  - 下载、代理、转码不属于当前范围。

## Questions Asked And Answers

### 1. 入口位置
- Question: `当存在可用预告片时，详情页入口应该放在哪里？`
- Answer: `内容区`
- Follow-up: `在内容区里，你更希望它落在哪一块？`
- Answer: `详情区`
- Why this mattered:
  - 当前 Hero 主操作区已经拥挤，且正片播放是明确的一号 CTA。
  - 该决策直接否定了“把现有 Hero 占位按钮接上线就完工”的最短路径假设。

### 2. 播放承载
- Question: `你希望预告片在详情页里以什么方式播放？`
- Answer: `弹层播放`
- Why this mattered:
  - 需求要求“详情页内直接播放”，但实现上仍需要在弹层、内嵌、独立路由之间锁方向。
  - 该决策明确 trailer 不应复用正片独立播放路由。

### 3. 同步时机
- Question: `预告片数据应该在什么时候进入系统并可供详情页使用？`
- Answer: `随元数据同步`
- Why this mattered:
  - 现有后端已经有 metadata refetch/job/worker 模式，可直接复用。
  - 如果不先锁，后续实现会在“详情页现场取数”与“后台持久化 contract”之间分叉。

### 4. 选片规则
- Question: `当 TMDB 返回多支视频时，Phase 9 应该如何选出“可播放预告片”？`
- Answer: `Trailer优先并回退`
- Follow-up: `在多个同类候选都可用时，你希望再按什么规则定最终播放项？`
- Answer: `简单优先`
- Why this mattered:
  - `TRLR-02` 的“是否展示入口”取决于是否存在一个最终可播结果。
  - 如果不锁规则，后端落库与前端展示都会围绕“多个候选”产生歧义。

## Locked Outcomes

- 预告片入口不放 Hero，而放在详情内容区中的详情区/附加信息区。
- 点击入口后，在当前详情页内以弹层播放器播放 trailer，并返回原详情上下文。
- trailer 作为 metadata 的一部分，随现有 match/refetch 链路同步并持久化，详情页不做现场 TMDB 拉取。
- 选片规则采用 `official Trailer -> Trailer -> Teaser` 的回退顺序；同级候选直接取最靠前可播放项。

## Out-Of-Scope Reminders Captured During Discussion

- 不提前纳入外部 trailer 来源。
- 不做 trailer 下载、代理、转码或离线缓存。
- 不引入独立 trailer 播放路由或多候选选择器。

## Output Produced

- `.planning/phases/09-trailer-discovery-playback/09-CONTEXT.md`
