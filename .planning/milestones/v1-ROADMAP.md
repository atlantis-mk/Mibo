# Milestone v1: MVP

**Status:** ✅ SHIPPED 2026-04-22
**Phases:** 1-6
**Total Plans:** 13

## Overview

v1 将 Mibo 从可用原型推进为一个完整可用的家庭媒体系统首版：建立了稳定的登录与应用入口边界、媒体源与媒体库接入、后台扫描与任务观测、语义化媒体目录、统一播放与进度模型、能力感知的播放决策，以及稳定身份和增量刷新基础设施。

## Phases

### Phase 1: Access & Platform Boundary

**Goal**: Users can initialize Mibo, sign in, and rely on one stable media API boundary while storage implementation details stay hidden behind `mibo-media-server`.
**Depends on**: Nothing (first phase)
**Plans**: 2 plans

Plans:

- [x] 01-01: Setup/auth contract hardening
- [x] 01-02: Two-stage gate UX and boundary flow

**Details:**
建立了以 `mibo-media-server` 为中心的统一入口边界。后端锁定 setup/auth 契约，前端实现 `/setup` 硬门禁和应用内软引导，确保用户进入的是媒体产品边界而不是 OpenList 原生入口。

### Phase 2: Library & Async Sync Foundation

**Goal**: Administrators can connect storage-backed libraries and trust scans/refreshes to run asynchronously without degrading interactive requests.
**Depends on**: Phase 1
**Plans**: 3 plans

Plans:

- [x] 02-01: Async scan settings, scheduled refresh, and jobs filtering contracts
- [x] 02-02: Admin source/library flow, status badges, jobs monitoring, and refresh controls
- [x] 02-03: Authenticated admin boundary verification closure

**Details:**
交付了媒体源与媒体库管理、异步扫描排队、周期刷新、任务列表筛选与重试，并补齐了所有相关管理接口的鉴权边界，确保后台任务能力既可观测又受保护。

### Phase 3: Semantic Catalog & Discovery

**Goal**: Users can explore a durable media catalog organized as movies and shows with useful metadata and library-aware discovery.
**Depends on**: Phase 2
**Plans**: 3 plans

Plans:

- [x] 03-01: Library-aware catalog browse filters and home discovery
- [x] 03-02: Persistent TMDB TV cache and season/episode contracts
- [x] 03-03: Library-first discovery UI and season-first TV detail navigation

**Details:**
媒体内容不再只是原始文件列表，而是可持久浏览的电影/剧集语义目录。首页支持 continue watching、recently played 和按媒体库组织的最新内容；详情页支持 season-first TV 浏览；人工 UAT 已确认搜索空态、返回上下文和 TV 详情直达行为。

### Phase 4: Playback Entry & Unified Progress

**Goal**: Users can start playback from catalog surfaces and have resume state persist through one client-facing progress model.
**Depends on**: Phase 3
**Plans**: 4 plans

Plans:

- [x] 04-01: Backend playback auth and canonical progress merge semantics
- [x] 04-02: Frontend playback route intent contract and controller seam
- [x] 04-03: Home/detail/playback UI wiring for resume and restart behavior
- [x] 04-04: Manual end-to-end playback/progress verification

**Details:**
交付了统一播放入口、鉴权保护的播放源解析、canonical progress 模型、继续观看直达播放、显式从头播放以及跨端恢复语义，并通过浏览器人工验证确认行为符合预期。

### Phase 5: Playback Decision Intelligence

**Goal**: Playback becomes more reliable across device types by choosing the best available path using media facts and explicit fallback behavior.
**Depends on**: Phase 4
**Plans**: 2 plans

Plans:

- [x] 05-01: Backend explicit client-profile playback contract, probe-aware decision engine, and per-request HLS fallback
- [x] 05-02: Web typed playback contract consumption and decision-aware playback page behavior

**Details:**
播放链路升级为 capability-aware：请求显式声明 `client_profile`，后端根据媒体探测信息返回 direct / fallback / unplayable 决策，前端据此诚实地呈现可播放状态、回退链路和失败原因。

### Phase 6: Stable Identity & Incremental Refresh

**Goal**: Libraries remain accurate over time as files move, rename, or change, without relying on full rescans for every update.
**Depends on**: Phase 5
**Plans**: 4 plans

Plans:

- [x] 06-01: Stable identity evidence contract and conservative scan staging
- [x] 06-02: Conservative fallback reconciliation and ambiguity quarantine
- [x] 06-03: Targeted incremental refresh jobs and subtree-safe partial scan behavior
- [x] 06-04: Authenticated storage-event intake that enqueues safe refresh work

**Details:**
建立了稳定文件身份优先的扫描模型、探测后保守回收、歧义隔离、增量 targeted refresh，以及带鉴权和根路径约束的存储事件接入，为大库和多存储长期演进打下基础。

---

## Milestone Summary

**Key Accomplishments:**

- 建立以 `mibo-media-server` 为核心的统一产品入口和鉴权边界
- 交付媒体源/媒体库接入、异步扫描、任务可观测与重试控制
- 将原始文件浏览升级为电影/剧集语义目录和按库发现体验
- 打通统一播放入口、续播/重播语义和跨端进度同步
- 引入能力感知的播放决策，支持直链优先和明确回退路径
- 建立稳定身份、增量刷新和存储事件驱动更新基础设施

**Key Decisions:**

- 保留 OpenList 作为存储接入层，而不把媒体业务深度塞进上游
- `mibo-media-server` 继续作为媒体语义、扫描编排、播放和多端 API 的核心
- V1 维持简单部署形态，但在代码层明确 API / Worker / StorageProvider 边界
- 播放能力优先直链，HLS/转码作为兜底而不是默认路径

**Issues Resolved:**

- 完成了 Phase 2 管理接口的鉴权边界修复
- 完成了 Phase 3 的人工验收并修正了里程碑 closeout 时发现的文档漂移
- 修复了 `library/$libraryId` 自动跳到第一条详情的问题，恢复真正的媒体库浏览页

**Issues Deferred:**

- 更完整的首页推荐、家庭成员隔离、远程访问优化和更成熟的 HLS/转码能力留到下一里程碑

**Technical Debt Incurred:**

- `audit-open` 仍将 `resolved` 的 UAT 文件视为 closeout 噪音，需要后续工具侧修正
- 根仓库与 `web/` 子仓库分离，里程碑关闭时仍需跨仓库检查和提交

---

_For current project status, see .planning/ROADMAP.md_
