# Mibo

## What This Is

Mibo 是一个已经交付 v2 的家庭媒体系统，由 `web/` 前端、`mibo-media-server/` 后端和作为存储接入底座的 `OpenList` 组成。当前产品已经具备从媒体源接入、语义目录浏览、播放入口、进度同步、内容发现、预告片播放、计划任务运营到安全监听刷新的完整主路径，并继续以 `mibo-media-server` 作为业务核心向 Web、移动端和 TV 统一接入能力演进。

## Core Value

无论底层媒体文件来自本地磁盘、NAS 还是云盘，用户都能稳定地完成媒体库接入、内容浏览、播放和进度同步。

## Completed Milestone: v2 Product Discovery And Operations

**Goal:** 把 Mibo 从基础可用媒体系统推进到更完整的内容发现体验与后台运营能力，其中主线偏前台用户体验，但同时补齐管理员侧的自动化与治理能力。

**Target features:**
- 全文搜索：支持按标题 / 演员 / 导演搜索，结果区分电影和剧集，并支持高亮、排序、历史记录
- 更多筛选：类型、年份、地区、评分、是否看过、媒体库、分辨率
- 预告片观看：从 TMDB / 外部源拉取预告片链接，并在详情页直接播放
- 元数据管理：管理员可编辑标题、原始标题、年份、简介、海报、背景图、分类、演员、季集信息，支持锁定字段与重新匹配
- 扫描监听：基于存储变更自动触发刷新，而不是仅依赖手动扫描
- 计划任务管理：支持调度元数据重抓、预告片同步、库清理、失效链接检查、封面刷新等后台任务

## Current Milestone: v3 剧集元数据治理 catalog kernel 迁移

**Goal:** 把 Phase A 已落地的并行 catalog kernel 接入真实扫描、元数据、API、播放、搜索、前端和治理流程，最终从旧 `MediaItem` / `MediaFile` 主链路迁移到新内核。

**Target features:**
- 内核契约、回填与迁移护栏，让旧库可重复、安全地迁移到 `catalog_items` / `media_assets` / `asset_items`。
- 扫描器直接写入新 catalog kernel，并支持剧集层级、多集文件、多版本和删除可用性更新。
- 元数据引擎以 series 为治理根，匹配 TMDB/Provider 后生成 season/episode 目录项、证据快照、字段状态和图片候选。
- API、播放、搜索、列表、详情和进度从旧 `MediaItem` 主链路切换到 `CatalogItem` / `MediaAsset`。
- 元数据治理 UI 展示并管理字段锁、来源证据、图片选择、外部 ID 和资产链接。
- 旧模型收口，补齐外键、唯一约束、索引、投影重建和一致性检查。

## Current State

- 已 shipped `v1 MVP`，覆盖 6 个阶段、13 个计划
- 已 shipped `v2 Product Discovery And Operations`，覆盖 5 个阶段、18 个计划
- 已具备本地存储与 OpenList 存储接入边界
- 已具备电影/剧集语义目录、TV season/episode 详情和首页发现流
- 已具备统一播放入口、续播/重播、能力感知播放决策和 canonical progress
- 已具备稳定身份、增量 targeted refresh 和存储事件驱动同步基础能力
- 已完成 Phase 8：原生 discovery contract、全局搜索、搜索历史、共享筛选，以及由 projection + lifecycle refresh 保证的新鲜度验证链路
- 已完成 Phase 11：存储事件监听会安全进入 targeted refresh，并由去重合并与 reconciliation 兜底保证刷新链路稳定

## Requirements

### Validated

- ✓ 用户可以完成系统初始化并进入应用主流程 — v1
- ✓ 用户可以登录并使用基于会话的鉴权访问受保护接口 — v1
- ✓ Web 客户端通过 `mibo-media-server` 的稳定 API 边界进入系统，而不是暴露 OpenList 产品入口 — v1
- ✓ 管理员可以配置媒体源与媒体库，并把扫描/刷新作为后台任务排队执行 — v1
- ✓ 用户可以浏览电影/剧集语义目录，看到详情、海报、季集结构并进入媒体详情页 — v1
- ✓ 用户可以获得统一播放入口，并在继续观看、从头播放和跨端进度同步之间保持一致语义 — v1
- ✓ 系统会优先直链播放，并在不可直链时返回明确回退路径或不可播放原因 — v1
- ✓ 系统具备稳定文件身份、增量刷新和安全的存储事件驱动更新能力 — v1
- ✓ 用户可以通过标题、演员、导演完成产品内全文搜索，并在结果中区分电影和剧集 — Validated in Phase 8
- ✓ 用户可以通过类型、年份、地区、评分、是否看过、媒体库、分辨率等维度筛选媒体内容 — Validated in Phase 8 (current phase scope delivered FLTR-01..06; FLTR-07/08 remain future requirements)
- ✓ 用户可以在媒体详情页直接观看来自 TMDB / 外部源的预告片 — Validated in Phase 9
- ✓ 管理员可以人工编辑媒体元数据、锁定字段、重新匹配并重抓元数据 — Validated in Phase 7 (META-07 field locking remains future scope)
- ✓ 系统可以基于存储变更自动触发安全的增量刷新 — Validated in Phase 11
- ✓ 管理员可以管理后台计划任务，包括扫描、元数据、预告片和清理类调度任务 — Validated in Phase 10

### Active

- [ ] 管理员可以将现有旧媒体库安全回填到新 catalog kernel，并获得冲突和一致性报告
- [ ] 扫描器可以直接生成 series / season / episode / movie catalog 项和可播放资产链接
- [ ] 剧集元数据可以按 series 级别治理，并由 provider 生成完整 season / episode 目录
- [ ] 用户和客户端可以通过新 `items`、`series`、播放、搜索和进度 API 使用 catalog 数据
- [ ] Web 前端主流程可以从旧 `MediaItem` 切换到 `CatalogItem`，并正确展示缺失/未播/可播放状态
- [ ] 管理员可以在治理 UI 中管理字段锁、来源证据、图片选择和资产链接
- [ ] 系统可以在新内核完成切换后收口旧模型，并通过生产级约束和投影检查保持一致性

### Out of Scope

- 深度 fork 并重度改造 `OpenList` 业务逻辑 — 会显著提高上游同步和长期维护成本
- 第一版就自研完整存储协议层与多种网盘驱动 — 复用 OpenList 更快落地
- 一开始拆成很多微服务 — 当前阶段会过早复杂化，降低交付效率
- 在没有真实瓶颈前引入全面直连适配器替代 OpenList — 优化应建立在实际热点之上

## Next Milestone Goals

- 当前 v3 milestone 聚焦“剧集元数据治理 catalog kernel 迁移”。
- 目标不是新增浅层 UI 功能，而是把媒体目录、元数据证据、可播放资产和用户进度统一迁移到可治理的 catalog kernel。
- 实现顺序必须保持可回填、可验证、可回滚：先契约和回填，再切扫描写入，再切元数据、API、播放和前端，最后收口旧模型。

## Context

项目现在是一个 brownfield 媒体平台：前端为 `web/` 下的 Vite + React SPA，后端为 `mibo-media-server/` 下的 Go 服务，通过 JSON HTTP API 协作。v1 之后，系统已经不只是“可用原型”，而是具备明确产品边界、可持续扩展的数据职责分层和多阶段交付验证记录的已上线基线。

## Constraints

- **Architecture**: `OpenList` 只负责存储接入与文件访问，媒体业务核心必须留在 `mibo-media-server`
- **Integration**: 只能通过 `mibo-media-server/internal/storage/openlist/adapter.go` 以 HTTP 方式对接 `OpenList`
- **Deployment**: 保持简单部署优先，按真实瓶颈再拆分
- **Performance**: 避免预先复杂化存储直连和多服务拆分
- **Client Surface**: API 设计需要兼容 Web、移动端、TV 统一访问
- **Data Ownership**: 文件命名空间由 OpenList 提供，媒体语义和用户状态由 `mibo-media-server` 持有

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 保留 OpenList 作为存储底座，而不是深度 fork | 复用多存储接入能力，同时避免上游同步成本失控 | ✓ Good |
| `mibo-media-server` 作为媒体业务核心 | 承载扫描、识别、播放、进度同步和多端 API，更符合产品演进方向 | ✓ Good |
| 在 `mibo-media-server` 内部坚持 `StorageProvider` 抽象 | 为未来直连 Local/NAS/Cloud 适配器保留替换空间 | ✓ Good |
| API 与 Worker 职责分离，但 V1 维持简单部署 | 慢任务与在线请求解耦，同时控制首版复杂度 | ✓ Good |
| 扫描与后台任务继续通过 Worker 队列执行，并通过统一任务视图暴露给管理员 | 保证请求快速返回，同时让管理员可观察、可重试地处理后台工作 | ✓ Good |
| 优先直链播放，转码只作为兜底能力 | 降低 V1 复杂度并优先满足家庭媒体播放主路径 | ✓ Good |
| 稳定身份优先于路径匹配，增量刷新必须保守不误绑定 | 保证长期媒体连续性和进度安全 | ✓ Good |
| v2 搜索与筛选先基于现有产品内数据能力实现，不接任何外部中间件 | 先验证体验和数据模型，避免过早引入部署与运维复杂度 | ✓ Good |
| 计划任务复用现有 jobs/worker 生命周期 | 统一手动、定时和监听触发的后台工作状态与失败处理 | ✓ Good |
| 监听刷新只入队 targeted/full scan，不直接修改 canonical media rows | 让存储事件与 reconciliation 共享既有扫描写入路径，降低数据漂移风险 | ✓ Good |
| 使用 `JobActiveIntent` 管理活跃监听意图唯一性 | 保留历史 job key 复用能力，同时避免并发风暴产生重复活跃刷新 | ✓ Good |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? -> Move to Out of Scope with reason
2. Requirements validated? -> Move to Validated with phase reference
3. New requirements emerged? -> Add to Active
4. Decisions to log? -> Add to Key Decisions
5. "What This Is" still accurate? -> Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check -> still the right priority?
3. Audit Out of Scope -> reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-25 after starting v3 catalog kernel migration milestone*
