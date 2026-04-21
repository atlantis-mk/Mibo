# Mibo

## What This Is

Mibo 是一个家庭媒体系统，当前由 `web/` 前端、`mibo-media-server/` 后端和作为存储底座的 `OpenList` 组成。项目正在从“可用的媒体浏览与播放原型”演进为一个以 `mibo-media-server` 为业务核心、面向 Web、移动端和 TV 统一接入的媒体平台。

## Core Value

无论底层媒体文件来自本地磁盘、NAS 还是云盘，用户都能稳定地完成媒体库接入、内容浏览、播放和进度同步。

## Requirements

### Validated

- ✓ 用户可以完成系统初始化并进入应用主流程 — existing
- ✓ 用户可以登录并使用基于会话的鉴权访问受保护接口 — existing
- ✓ Web 客户端通过 `mibo-media-server` 的稳定 API 边界进入系统，而不是暴露 OpenList 产品入口 — Phase 1
- ✓ 应用入口已形成硬门禁 + 软门禁的双阶段 setup 体验 — Phase 1
- ✓ 用户可以配置媒体源与媒体库，并触发后台同步任务 — existing
- ✓ 系统可以扫描存储中的媒体文件并落库为媒体项与文件记录 — existing
- ✓ 系统可以为媒体项生成播放地址，并支持基础播放进度回写 — existing
- ✓ 系统已经通过 `StorageProvider` 适配本地存储与 OpenList 存储 — existing

### Active

- [ ] 在 Phase 1 之后继续以 `mibo-media-server` 作为媒体业务核心，扩展而不打破与 OpenList 的稳定边界
- [ ] 将扫描、识别、`ffprobe`、转码等慢任务继续沉淀到 Worker 路径，避免阻塞在线请求
- [ ] 完善媒体语义模型，稳定支撑 `media_items / series / seasons / episodes` 等结构
- [ ] 强化播放链路，形成“优先直链，必要时转码兜底”的统一能力
- [ ] 补齐稳定文件身份、增量扫描与事件驱动更新能力，为后续大库和多存储场景做准备
- [ ] 为 Web、移动端、TV 统一接入预留一致的媒体 API 和进度同步模型

### Out of Scope

- 深度 fork 并重度改造 `OpenList` 业务逻辑 — 会显著提高上游同步和长期维护成本
- 第一版就自研完整存储协议层与多种网盘驱动 — 复用 OpenList 更快落地
- 一开始拆成很多微服务 — 当前阶段会过早复杂化，降低交付效率
- 在没有真实瓶颈前引入全面直连适配器替代 OpenList — 优化应建立在实际热点之上

## Context

项目当前已经是一个 brownfield 系统，而不是从零开始：前端是 `web/` 下的 Vite + React SPA，后端是 `mibo-media-server/` 下的 Go 服务，二者通过 JSON HTTP 接口协作。现有代码已经覆盖初始化、登录、媒体源/媒体库管理、扫描、媒体浏览、播放和进度等基础能力，并且在 `mibo-media-server/internal/storage/` 下建立了本地与 OpenList 的统一存储适配接口。

这次初始化的目标不是重新定义一个全新产品，而是把已有实现和 `docs/media-architecture/improved-architecture.md` 中的推荐架构对齐：保留 OpenList 作为存储接入层，把媒体语义、扫描编排、播放能力和多端产品能力继续集中到 `mibo-media-server`，并沿着 API/Worker 分离、适配器优先、先跑通再优化的方向推进。

## Constraints

- **Architecture**: `OpenList` 只负责存储接入与文件访问，媒体业务核心必须留在 `mibo-media-server` — 降低耦合和长期维护成本
- **Integration**: 只能通过 `mibo-media-server/internal/storage/openlist/adapter.go` 以 HTTP 方式对接 `OpenList` — 仓库边界已明确，不能直接把业务塞进上游代码
- **Deployment**: V1 优先保持简单部署，API 与 Worker 可同进程/同镜像运行 — 先保证落地速度，再按瓶颈拆分
- **Performance**: 不预先复杂化存储直连和多服务拆分，只在真实热点出现后优化 — 避免过度设计
- **Client Surface**: API 设计需要兼容 Web、移动端、TV 统一访问 — 这是架构文档明确的目标之一
- **Data Ownership**: 文件命名空间由 OpenList 提供，媒体语义和用户状态由 `mibo-media-server` 持有 — 保持数据职责清晰

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 保留 OpenList 作为存储底座，而不是深度 fork | 复用多存储接入能力，同时避免上游同步成本失控 | — Pending |
| `mibo-media-server` 作为媒体业务核心 | 承载扫描、识别、播放、进度同步和多端 API，更符合产品演进方向 | — Pending |
| 在 `mibo-media-server` 内部坚持 `StorageProvider` 抽象 | 为未来直连 Local/NAS/Cloud 适配器保留替换空间 | — Pending |
| API 与 Worker 职责分离，但 V1 维持简单部署 | 慢任务与在线请求解耦，同时控制首版复杂度 | — Pending |
| 优先直链播放，转码只作为兜底能力 | 降低 V1 复杂度并优先满足家庭媒体播放主路径 | — Pending |

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
2. Core Value check - still the right priority?
3. Audit Out of Scope - reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-21 after Phase 1 completion*
