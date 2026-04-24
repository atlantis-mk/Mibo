# Phase 10: Scheduled Operations Control - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning

<domain>
## Phase Boundary

管理员可以在 Mibo 内管理产品原生的 recurring maintenance schedules，用于自动化执行扫描、元数据重抓、预告片同步、库清理、失效链接检查、封面刷新等后台维护动作。该能力必须叠加在既有 jobs/worker 执行模型之上，并向管理员提供启停、立即运行、下次运行时间、最近结果和运行历史。

本阶段聚焦“schedule 管理产品层 + schedule 到 job 的执行桥接 + 管理端可见性”，不引入并行调度架构，不扩展为任意 cron 平台，也不把未来更细粒度的目标选择、跨系统通知编排或保留策略提前并入当前范围。

</domain>

<decisions>
## Implementation Decisions

### 管理入口与页面承载
- **D-01:** Phase 10 的正式管理员入口采用独立的计划任务工作台，而不是把完整能力直接塞进设置页 tab。
- **D-02:** 设置页现有“通知与任务”分组可以作为 schedule 能力的辅助入口或摘要承载，但不是主工作面。
- **D-03:** 计划任务工作台应沿用现有 metadata governance 的独立 admin workspace 模式，而不是依赖当前不稳定的示例侧栏导航。

### 调度规则表达
- **D-04:** schedule 配置默认采用产品化的频率模板表达，而不是直接暴露 cron。
- **D-05:** 第一版调度表达应围绕管理员可理解的 recurring frequency 组织，例如每天、每周、每月配合时间/星期等常见参数。
- **D-06:** Phase 10 不把 cron 或其他高级规则编辑器作为正式交付内容，以保持 simple deployment 和低认知负担。

### 任务建模粒度
- **D-07:** 每条 schedule 的核心建模采用“任务类型 + 目标范围”模式。
- **D-08:** 目标范围至少需要支持全局范围与按媒体库范围，不扩展到单个媒体项级别。
- **D-09:** schedule 类型应对齐 roadmap / requirements 已锁定的维护任务集合：扫描、元数据重抓、预告片同步、库清理、失效链接检查、封面刷新；后续 planner 可以决定这些类型是单表枚举、typed payload 还是 job template，但不能脱离这组正式能力。

### 执行与反馈模型
- **D-10:** schedule 只负责定义何时触发与触发什么；实际执行必须复用既有 jobs/service/worker 链路 enqueue 对应 job，而不是再造并行执行器。
- **D-11:** 管理员点击“立即运行”时，也应走同一 schedule-to-job 桥接路径，并返回标准 job 异步反馈，而不是同步长请求。
- **D-12:** 计划任务 UI 的状态语义应继续沿用现有后台 job 模式：`queued`、`running`、`completed`、`failed` 作为核心反馈语言。

### 可见性与历史
- **D-13:** 工作台列表必须直接展示每条 schedule 的 enabled 状态、next run time 与 latest result。
- **D-14:** 运行历史采用“列表概览 + 详情层查看最近若干次运行”的组织方式，而不是把完整历史直接塞进主列表。
- **D-15:** Phase 10 的最近结果和运行历史应围绕 schedule 维度组织，让管理员能看到某条 schedule 最近一次结果与近期运行轨迹，而不是只暴露全局 jobs 列表让用户自行拼接。

### 范围边界
- **D-16:** Phase 10 继续遵守既有约束：调度与媒体业务逻辑留在 `mibo-media-server`，不下沉到 OpenList，也不把 OpenList 当作 schedule 执行宿主。
- **D-17:** 当前已有的“按设置间隔触发全库扫描”只可视为实现参考和迁移锚点，不等同于本阶段最终产品能力；Phase 10 需要形成正式的 schedule domain model 和管理员管理面。
- **D-18:** Phase 10 不扩展到 item-level schedule、自定义高级规则语言、外部通知工作流或 retention policy 等相邻但未锁定的增强能力。

### the agent's Discretion
- 计划任务工作台的具体信息架构可由后续 agent 决定，例如列表 + 右侧详情抽屉、列表 + 独立详情页、或列表 + 对话框编辑，但必须保持“独立工作台为主入口、历史在详情层展开”的锁定方向。
- 频率模板的精确字段组织可由后续 research / planning 决定，例如是否抽象为 `daily` / `weekly` / `monthly` 及其参数对象，但不能退回直接暴露 cron 文本。
- schedule 与 job history 的具体数据建模可由后续规划决定，例如单独的 run 表、schedule 与 jobs 的关联字段、latest result 快照字段等，但必须支持 `SJOB-07` / `SJOB-08` 所需的 next run、latest result、run history。
- 设置页中的“通知与任务”tab 是否展示摘要卡片、统计概览或跳转入口可由后续规划决定，但不应取代独立工作台本身。

</decisions>

<specifics>
## Specific Ideas

- 独立计划任务工作台应延续 Phase 7 元数据治理工作台的产品风格：管理员到一个专门的运营面板里管理后台能力，而不是在内容详情或通用设置表单里零散操作。
- 设置页“通知与任务”目前更像占位分组，适合作为 schedule 的摘要入口，而不是承载复杂列表、运行历史和立即执行反馈的主界面。
- 调度规则要优先服务产品内管理员，因此第一版更适合做成“可理解的 recurring frequency 模板”，避免一上来引入 cron 心智负担和校验复杂度。
- schedule 的对象模型应让管理员理解为“让某类维护动作在某个范围内按固定节奏运行”，而不是把它暴露成底层 job queue 的技术配置。
- 运行可见性要体现两层：列表快速判断哪些 schedule 正常、何时下次运行；详情层再看最近几次运行结果与失败原因。

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase definition
- `.planning/ROADMAP.md` — Phase 10 的目标、依赖与 success criteria，明确这是叠加在现有 worker model 之上的 scheduled operations control。
- `.planning/REQUIREMENTS.md` — `SJOB-01` through `SJOB-08` 的锁定需求，定义了 schedule 类型、启停/立即运行/next run、latest result 与 run history。
- `.planning/PROJECT.md` — v2 对计划任务管理的产品定位，以及 `mibo-media-server` 持有媒体业务逻辑、OpenList 仅作存储边界的系统约束。
- `.planning/STATE.md` — 当前阶段状态，以及“Layer scheduled jobs on the existing jobs/worker model rather than a parallel scheduler”这一已锁定累计决策。
- `.planning/phases/07-metadata-governance-matching/07-CONTEXT.md` — 独立管理工作台、后台 job 异步反馈、app-owned governance 的先决约束。
- `.planning/phases/08-native-search-discovery-filters/08-CONTEXT.md` — typed API client + TanStack Query、全局壳层/工作台承载方式的最近前例。
- `.planning/phases/09-trailer-discovery-playback/09-CONTEXT.md` — trailer 仍依托 metadata + jobs/worker 模式，不引入独立调度体系的直接上游约束。

### Live backend anchors
- `mibo-media-server/internal/jobs/service.go` — 当前通用 job queue 服务，已提供 `Enqueue`、`EnqueueUnique`、`List`、`Retry`、`ClaimNext`、`Complete`、`Fail`，是 schedule 触发后实际执行的直接承载层。
- `mibo-media-server/internal/worker/worker.go` — 当前 worker 轮询和 dispatch 既有 job kinds，并已存在 `getRefreshInterval` / `triggerScheduledScans` 这类简化定时触发逻辑，可作为 Phase 10 调度桥接的参考锚点。
- `mibo-media-server/internal/httpapi/handlers_jobs.go` — 当前只有 jobs list / retry 能力，没有 schedule 管理 API，是 Phase 10 新增 contract 的直接对照物。
- `mibo-media-server/internal/httpapi/router.go` — 当前 jobs 相关路由注册位置，是新增 schedule routes 的入口锚点。
- `mibo-media-server/internal/database/models.go` — 当前只有 `Job` 模型，没有 schedule / schedule run 历史模型，说明 Phase 10 需要新增正式 domain persistence。
- `mibo-media-server/internal/library/enrichment.go` — 当前扫描、metadata 重抓等 enqueue 模式已经形成，是把 schedule 映射到既有维护任务时的重要执行入口。

### Live frontend anchors
- `web/src/features/metadata-governance/index.tsx` — 独立 admin workspace 的现成页面模式，是 Phase 10 计划任务工作台的首要 UI 先例。
- `web/src/features/metadata-governance/detail.tsx` — 现有后台 job 触发后通过 `jobId` + `listJobs()` 轮询更新 UI 的成熟异步反馈模式，可复用于“立即运行”反馈。
- `web/src/features/settings/index.tsx` — 当前“通知与任务”tab 的占位位置，可作为 schedule 摘要入口或跳转入口，但不是主工作台。
- `web/src/routes/_app.tsx` — 当前 app 壳层只提供通用 layout，不直接决定 admin feature 信息架构；说明 Phase 10 更适合通过独立 route workspace 承载。
- `web/src/routes/_app.metadata.index.tsx` 与 `web/src/routes/_app.metadata.$id.tsx` — 现有治理工作台 route 组织先例，可为 schedule workspace 路由结构提供参考。
- `web/src/lib/mibo-api.ts` — 当前 typed API client 已有 `listJobs(...)` 契约，是扩展 schedule client contract 的直接位置。

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/features/metadata-governance/index.tsx`: 已有独立管理工作台的信息架构和页面风格，适合复用为 schedule admin workspace 的交互先例。
- `web/src/features/metadata-governance/detail.tsx`: 已有“触发后台动作 -> 记录 jobId -> 轮询 jobs -> 按状态更新 UI”的现成实现，是“立即运行 schedule”最直接的前端行为模板。
- `web/src/features/settings/index.tsx`: 现有设置页已经给出“通知与任务”分组，可承接 schedule 摘要入口，不必额外创造一套设置导航概念。
- `web/src/lib/mibo-api.ts`: 已有 typed API client 和 jobs 查询方法，新增 schedule list/create/update/toggle/run/history 等契约时应继续沿用。
- `mibo-media-server/internal/jobs/service.go`: 已有稳定的 enqueue / claim / complete / fail 生命周期，是 schedule 实际落地执行的后端核心复用点。
- `mibo-media-server/internal/worker/worker.go`: 已有 job kind dispatch 和简化定时扫描触发逻辑，说明代码库已经具备“周期性触发 -> enqueue job”的局部基础。

### Established Patterns
- 当前仓库的后台重操作统一走 job/worker，而不是在 HTTP 请求中同步做重工作；Phase 10 必须延续这一模式。
- 当前管理员复杂能力更适合放在独立工作台，而不是零散塞进详情页或通用 settings 表单；Phase 7 已给出清晰先例。
- 前端数据交互继续遵循 typed API client + TanStack Query；Phase 10 不应退回 ad hoc fetch 或手写不一致 contract。
- 当前产品真实可依赖的 admin 承载点是独立 route workspace 与 settings 页面，不是示例性质的 `AppSidebar` 导航数据。

### Integration Points
- 后端需要新增 schedule domain model，并把“何时触发、触发何种维护动作、作用于哪个范围”桥接到既有 enqueue 入口。
- 后端需要补充 schedule 管理 API，覆盖列表、创建、编辑、启停、立即运行、next run、latest result 和 run history。
- 后端需要决定如何把 schedule 与 jobs 关联，确保可以按 schedule 维度回看最近结果和近期历史，而不是只剩全局 jobs 流水。
- 前端需要新增独立 schedule workspace，并与 typed API / query keys 打通。
- 前端需要在“立即运行”动作上复用现有 job feedback 模式，让管理员看到 queued/running/completed/failed 的异步状态演进。
- 前端可以在 settings 的“通知与任务”tab 中添加摘要或跳转入口，与 schedule 工作台形成轻耦合连接。

### Constraints
- 当前后端没有任何 schedule / run history 的持久化模型，Phase 10 不是简单把现有 jobs list 改个 UI 名称即可完成。
- 当前 jobs API 只支持 list/retry，不能满足 schedule CRUD、toggle、run-now、next-run 和按 schedule 查看历史的需求。
- 当前 worker 里的定时扫描只是设置驱动的特例逻辑，不足以直接覆盖 roadmap 要求的多任务类型 schedule 管理面。
- 当前前端没有现成的 schedule 页面或组件，且 settings“通知与任务”是静态占位，因此需要真正新增产品工作流而不是接已有完整页面。

</code_context>

<deferred>
## Deferred Ideas

- 单个媒体项级别的 schedule，或更复杂的按任意筛选条件选目标范围。
- 直接暴露 cron / 高级规则语言 / power-user 调度编辑器。
- 外部通知编排、Webhooks、邮件模板中心等超出当前“任务控制”范围的通知系统增强。
- schedule retention policy、自动清理历史、长期归档策略。
- 与本 phase 无关的额外维护任务类型扩张，只保留 roadmap 已锁定的任务集合。

</deferred>

---

*Phase: 10-scheduled-operations-control*
*Context gathered: 2026-04-24*
