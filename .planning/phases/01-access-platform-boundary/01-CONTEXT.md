# Phase 1: Access & Platform Boundary - Context

**Gathered:** 2026-04-21
**Status:** Ready for planning

<domain>
## Phase Boundary

为 Mibo 建立稳定的访问入口与平台边界：用户可以完成初始化、登录并通过统一的媒体 API 进入系统；客户端看到的是 `mibo-media-server` 提供的媒体中心化边界，而不是底层 OpenList 或存储实现细节。

</domain>

<decisions>
## Implementation Decisions

### 初始化入口
- **D-01:** 初始化前采用强门禁。未完成初始化条件时，用户不能直接进入主应用功能区，路由层继续负责把未完成初始化的访问导向 `/setup`。
- **D-02:** 允许进入应用的最低条件是已创建管理员账号，不要求在同一次 setup 中强制完成媒体源和媒体库创建。
- **D-03:** 当系统已经有管理员账号、但还没有媒体源或媒体库时，用户进入应用后的首屏应回到配置引导，而不是直接落到媒体首页空壳。
- **D-04:** Phase 1 继续保留分步向导式 setup 体验，不改成自由跳转的设置中心。

### 访问边界
- **D-05:** 客户端入口和登录后体验要体现“双阶段门禁”：无账号时强制走 setup；有账号但媒体配置未完成时允许进入应用边界，但继续落到配置引导。
- **D-06:** 下游规划必须把初始化状态判断、路由重定向、setup 完成事件和首屏落点视为同一条用户流，避免前端路由、setup 状态和后端 setup 语义各自漂移。

### the agent's Discretion
- setup 向导内部每一步的具体视觉样式、文案细节和进度呈现
- “配置引导首屏”在首页空状态、专门引导页或 settings 容器内的具体实现方式
- 登录后返回路径、loading 态和错误态的具体交互细节

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and product constraints
- `.planning/ROADMAP.md` — Phase 1 goal, success criteria, and fixed scope boundary
- `.planning/REQUIREMENTS.md` — Phase 1 requirements `ACCS-01`, `ACCS-02`, `ACCS-03`, `CATA-01`
- `.planning/PROJECT.md` — project-level architecture constraints and product direction

### Architecture and boundary rules
- `docs/media-architecture/improved-architecture.md` — keeps `OpenList` at the storage edge and `mibo-media-server` as the media/business core
- `AGENTS.md` — repo-specific rule that setup/auth flow changes must keep `web/src/router.tsx`, `web/src/components/setup-wizard.tsx`, and `web/src/lib/client-config.ts` aligned

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/components/setup-wizard.tsx`: 现有分步向导，可直接作为 Phase 1 的主要 UI 基础
- `web/src/router.tsx`: 已有 setup 状态检查和 `/setup` 路由门禁逻辑，可在此演进“双阶段门禁”
- `web/src/lib/mibo-api.ts`: 前端统一 API 边界，适合继续承接稳定的客户端 HTTP 契约
- `web/src/lib/client-config.ts`: 已集中管理 token、API base URL 和 setup status event
- `mibo-media-server/internal/auth/service.go`: 已有用户名密码登录、会话 token、30 天 session TTL 的后端基础
- `mibo-media-server/internal/storage/provider.go`: 已有存储抽象，是保持客户端不感知 OpenList 细节的基础边界

### Established Patterns
- 前端通过 `createMiboApi(...)` 统一请求后端，而不是到处直接 `fetch`
- 路由层已经承担初始化门禁职责，setup 完成通过事件刷新状态
- 后端 HTTP 响应统一使用 envelope，并通过 `router.go` 集中注册客户端 API
- 存储能力通过 `StorageProvider` 和 provider registry 暴露，业务层不应直接耦合 OpenList 细节

### Integration Points
- `web/src/router.tsx`：初始化检查、登录后落点、未完成配置时的首屏引导
- `web/src/components/setup-wizard.tsx`：setup 完成条件、步骤顺序和阶段切换
- `web/src/lib/client-config.ts`：setup 状态广播、本地 token 与 API 地址持久化
- `mibo-media-server/internal/httpapi/router.go`：`/api/v1/setup/status`、`/api/v1/auth/*` 以及客户端可见 API 形状
- `mibo-media-server/internal/auth/service.go`：认证状态与 session 持续性

</code_context>

<specifics>
## Specific Ideas

- 用户首次进入时应感受到清晰、明确的 setup 流，而不是散开的后台配置页
- “允许进入应用”不等于“允许看到一个空首页”，而是要把用户继续导向配置引导
- Phase 1 要体现 `mibo-media-server` 是客户端唯一稳定边界，不能让客户端心智落到 OpenList 或存储测试接口上

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-access-platform-boundary*
*Context gathered: 2026-04-21*
