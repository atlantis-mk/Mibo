# Mibo

## What This Is

Mibo 是一个已经交付 v1 的家庭媒体系统，由 `web/` 前端、`mibo-media-server/` 后端和作为存储接入底座的 `OpenList` 组成。当前产品已经具备从媒体源接入、语义目录浏览、播放入口、进度同步到增量刷新的完整主路径，并继续以 `mibo-media-server` 作为业务核心向 Web、移动端和 TV 统一接入能力演进。

## Core Value

无论底层媒体文件来自本地磁盘、NAS 还是云盘，用户都能稳定地完成媒体库接入、内容浏览、播放和进度同步。

## Current State

- 已 shipped `v1 MVP`，覆盖 6 个阶段、13 个计划
- 已具备本地存储与 OpenList 存储接入边界
- 已具备电影/剧集语义目录、TV season/episode 详情和首页发现流
- 已具备统一播放入口、续播/重播、能力感知播放决策和 canonical progress
- 已具备稳定身份、增量 targeted refresh 和存储事件驱动同步基础能力

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

### Active

- [ ] 首页继续观看、最近加入和家庭发现流的产品体验增强
- [ ] 更细粒度的家庭成员隔离和访问控制
- [ ] 更成熟的远程访问和外网部署体验
- [ ] 更完整的 HLS / 转码能力和更多设备场景兼容性
- [ ] 在保持边界稳定的前提下继续扩展多端统一媒体 API

### Out of Scope

- 深度 fork 并重度改造 `OpenList` 业务逻辑 — 会显著提高上游同步和长期维护成本
- 第一版就自研完整存储协议层与多种网盘驱动 — 复用 OpenList 更快落地
- 一开始拆成很多微服务 — 当前阶段会过早复杂化，降低交付效率
- 在没有真实瓶颈前引入全面直连适配器替代 OpenList — 优化应建立在实际热点之上

## Next Milestone Goals

- 提升首页发现体验，让 Continue Watching / Recently Added / library rails 更像产品而不是基础数据面板
- 继续打磨播放器能力，补齐更成熟的 fallback / HLS / transcoding 场景
- 提升家庭多用户能力，包括更细的账号隔离和访问控制
- 打磨远程访问、部署和长期运行稳定性体验

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

---
*Last updated: 2026-04-22 after v1 milestone completion*
