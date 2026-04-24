# Phase 11: Event-Driven Refresh Hardening - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段要把存储侧变更安全地转成 Mibo 内部的保守 refresh 工作流：当媒体库发生新增、更新、删除、移动等变化时，系统通过既有 `storage-events -> jobs -> worker -> targeted refresh / full sync fallback` 路径保持库内容新鲜，并用兜底 reconciliation 防止监听漏事件导致状态漂移。

本阶段聚焦“事件归一化策略 + 合并/去抖策略 + 保守删除/移动收敛 + 周期性补偿对账”，不把 listener 扩展成通用实时系统，不直接让 listener 修改 canonical 媒体数据，也不引入外部中间件、事件总线或新的存储语义来源。

</domain>

<decisions>
## Implementation Decisions

### 事件触发语义
- **D-01:** `create`、`update`、`delete` 默认归一为受影响目录的 targeted refresh，而不是直接修改媒体行或默认触发全库扫描。
- **D-02:** `move` / `rename` 在新旧路径都存在且可归一时，取共同祖先目录作为 targeted refresh root；只有路径缺失、越界、无法安全归一时才回退 full sync。
- **D-03:** listener 事件始终只是 refresh 工作触发器，不是 canonical 数据真相来源；任何最终媒体状态仍由既有扫描/对账链路收敛。

### 去重与合并窗口
- **D-04:** Phase 11 需要显式的时间窗合并策略，不能只依赖当前 job uniqueness。
- **D-05:** 合并粒度以 `library_id + normalized_root` 为主，在短时间窗口内把重复或抖动事件折叠为一次 refresh 意图。
- **D-06:** 当多个相邻/嵌套子路径在同一窗口内出现时，可以提升到更高祖先目录统一 refresh，但应保持“尽量局部、必要时再扩大”的保守策略。

### 删除与移动安全策略
- **D-07:** 删除和移动事件到来时，不在 ingest 阶段立即把内容硬判为失效；先交给 targeted refresh / partial scan / reconcile 自然收敛。
- **D-08:** 产品语义优先避免误删、误漂移和短暂存储抖动造成的错误状态，因此宁可短时间 eventual consistency，也不采用激进即时失效策略。
- **D-09:** Phase 11 应继续复用既有 provisional / missing / fallback reconciliation 语义，而不是为 listener 单独创造一套新的媒体状态机。

### Reconciliation 兜底补偿
- **D-10:** 必须保留周期性兜底 reconciliation，listener 不是绝对可靠事件流；系统需要通过后续补偿扫描把漏事件收敛回来。
- **D-11:** 兜底补偿以库级或安全范围的周期性扫描/对账任务为主，而不是要求每个 listener 事件都精确可重放。
- **D-12:** reconciliation 的职责是“恢复一致性和降低漂移”，不是把系统升级为强实时同步平台。

### 范围边界
- **D-13:** Phase 11 继续遵守既有边界：`OpenList` 只是存储边界，业务规则和事件安全策略留在 `mibo-media-server`。
- **D-14:** 本阶段不扩展到外部消息队列、分布式事件总线、通用 watcher 平台、实时推送 UI 或监听健康大盘，这些如有需要应留待后续 phase / future requirement。

### the agent's Discretion
- 合并窗口的具体时长、祖先提升规则和持久化形式可由后续 research / planning 决定，但必须满足“抗 event storm、避免重复扫描、优先局部 refresh”的锁定方向。
- reconciliation 的精确触发节奏、是否复用已有 schedule 能力、以及失败恢复的细化策略可由后续 agent 决定，但不能放弃 LIST-04 的周期兜底补偿目标。
- 事件 ingest 归一逻辑是继续留在 HTTP handler 还是下沉到独立 listener/service 层，可由后续规划决定，但对外行为必须保持上述语义。

</decisions>

<specifics>
## Specific Ideas

- 当前代码已经具备 `/api/v1/storage-events`、`QueueTargetedRefresh(...)`、`RunTargetedRefresh(...)`、局部 search reindex 和 fallback file reconciliation，所以 Phase 11 更像“把已有基础能力生产化硬化”，而不是从零发明 listener 体系。
- `create/update/delete/move` 的产品语义已锁定为“保守定向刷新优先，无法安全归一时才升级为 full sync fallback”。
- 对 event storm 的处理方向已锁定为“时间窗 + 路径合并”，避免 NAS/网盘抖动、批量改名或目录级通知把 worker 和 OpenList 打爆。
- 删除/移动类事件的用户体验目标不是瞬时强一致，而是避免把暂时不可见、延迟传播或路径抖动误判成永久丢失。
- LIST-04 的补偿策略已锁定为周期兜底，而不是纯 listener 驱动；即使事件流漏报，也必须能靠后续对账恢复一致性。

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase definition and locked constraints
- `.planning/ROADMAP.md` — Phase 11 的目标、success criteria 与“不做实时系统，只做 safe listener-driven refresh + reconciliation”的正式边界。
- `.planning/REQUIREMENTS.md` — `LIST-01` through `LIST-04` 的锁定需求，明确事件类型、targeted refresh、去抖/合并、兜底 reconciliation。
- `.planning/PROJECT.md` — 产品与架构层非协商约束：`OpenList` 只作存储边界，业务逻辑留在 `mibo-media-server`，不引入外部中间件。
- `.planning/STATE.md` — 当前 focus 已切到 Phase 11，且已累计锁定“scan listeners only enqueue targeted refresh and reconciliation work, never direct canonical row mutation”。
- `.planning/phases/07-metadata-governance-matching/07-CONTEXT.md` — app-owned canonical data、后台重操作走 jobs/worker、独立业务层持有规则的先决边界。
- `.planning/phases/08-native-search-discovery-filters/08-CONTEXT.md` — refresh 后相关产品投影仍应依附既有 app-owned contract，而不是退回存储侧真相直出。
- `.planning/phases/10-scheduled-operations-control/10-CONTEXT.md` — 周期性后台补偿工作应优先复用既有 jobs/worker/schedule 体系，不再造并行执行模型。

### Live backend anchors
- `mibo-media-server/internal/httpapi/router.go` — 当前已注册 `POST /api/v1/storage-events`，是 listener ingress 的现有入口。
- `mibo-media-server/internal/httpapi/handlers_storage_events.go` — 当前事件校验、越界保护、kind 归一、targeted refresh / full sync fallback 的直接实现锚点。
- `mibo-media-server/internal/library/service.go` — `JobKindTargetedRefresh`、payload 结构和 enqueue API 的定义位置。
- `mibo-media-server/internal/library/scan_run.go` — `RunTargetedRefresh` 的实际执行逻辑，说明 partial scan、library status 更新和 scoped search reindex 已存在。
- `mibo-media-server/internal/library/scan.go` — 扫描主流程与 partial/full cleanup 行为，是删除/移动最终如何收敛的核心语义来源。
- `mibo-media-server/internal/library/scan_upsert.go` — path/stable identity/provisional 处理逻辑，是移动/替换场景的直接行为基础。
- `mibo-media-server/internal/library/scan_reconcile.go` — 既有 fallback reconciliation 逻辑，是 rename / move 后保守恢复 media item 归属的重要锚点。
- `mibo-media-server/internal/worker/worker.go` — 现有 worker 已 dispatch `sync_library` 与 `targeted_refresh`，是 Phase 11 应继续复用的统一执行面。
- `mibo-media-server/internal/search/service.go` — targeted refresh 后按 rootPath scoped reindex 的现成能力，说明局部刷新已能与搜索 freshness 对齐。
- `mibo-media-server/internal/database/models.go` — 当前无 listener event inbox / coalescing / health persistence 模型，是本阶段需要补强的事实锚点。

### Behavioral tests and prior evidence
- `mibo-media-server/internal/httpapi/router_test.go` — 当前 storage event 路由行为测试，覆盖鉴权、path escaping、防越界和 unsupported kind fallback。
- `mibo-media-server/internal/worker/worker_test.go` — 当前 worker 对 targeted refresh 的唯一键、执行顺序和 scoped subtree scan 的行为证据。
- `mibo-media-server/internal/library/identity_reconcile_test.go` — 现有 fallback reconciliation 如何在 rename / ambiguity 场景下保守恢复或隔离风险。
- `.planning/research/ARCHITECTURE.md` — 已提前指出可新增 listener service、coalescing、apply_storage_event_refresh 等方向，但必须复用现有 refresh pipeline。
- `.planning/research/PITFALLS.md` — 已明确 listener 不能直接改 canonical rows，且 event storm 需要 path coalescing / debounce / library-level throttles。
- `.planning/research/STACK.md` — 已明确不要新引库重做 listener 基础设施，并强调即使有事件流仍要保留 reconciliation scan。

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `mibo-media-server/internal/httpapi/handlers_storage_events.go`: 已有事件 payload 校验、library/source 校验、路径归一与 fallback 分支，适合作为 Phase 11 ingress 强化的直接起点。
- `mibo-media-server/internal/library/service.go`: 已有 targeted refresh job shape 和 enqueue 接口，可继续承载 coalesced refresh work。
- `mibo-media-server/internal/library/scan_run.go`: 已有 subtree-scoped partial scan + scoped search reindex，说明“局部刷新而非全库重扫”已具备执行基础。
- `mibo-media-server/internal/library/scan_reconcile.go`: 已有 conservative fallback reconciliation，可复用于 move/rename 后的最终收敛。
- `mibo-media-server/internal/worker/worker.go`: 已有统一 job dispatch 面，Phase 11 不需要额外创造执行通道。
- Phase 10 schedule 基础: 如果 planner 认为合适，可复用已有 schedule / worker 能力承载周期性 reconciliation，而不是再造一个独立定时器体系。

### Established Patterns
- 后台重工作统一走 job/worker，而不是在 HTTP 请求里同步完成；listener ingress 也必须延续这个模式。
- `OpenList` 只通过适配层提供存储读能力，不承载业务级媒体状态判断。
- canonical 媒体数据由扫描/匹配/对账链路维护，listener 只是触发刷新和恢复一致性的线索输入。
- partial scan 的删除清理只在 scoped root 内生效，这天然支持“先局部保守收敛，再在必要时扩大范围”的 Phase 11 方向。

### Integration Points
- 事件 ingress 层需要补强去抖/合并与安全归一逻辑，并把最终 refresh 意图映射到既有 job queue。
- library / worker 层需要承接 coalesced refresh work，保证同库同路径短时间内不会反复扫描。
- 若采用周期性兜底 reconciliation，需和现有 schedule / worker / job history 体系对齐，避免形成并行后台系统。
- 删除/移动场景需要继续依赖扫描缺失判定、provisional file 和 fallback reconciliation 的现有语义，不应在 listener 层旁路这些规则。

### Constraints
- 当前系统还没有显式的 listener event inbox、coalescing window persistence 或 listener health state；Phase 11 不能假设这些设施已经存在。
- 当前 `handlers_storage_events.go` 已经有一套基础 kind 映射，后续设计应在其上硬化，而不是推翻成全新架构。
- 当前 targeted refresh 只保证 scoped subtree freshness，不等于全库绝对一致；因此 LIST-04 的补偿扫描不是可选项。

</code_context>

<deferred>
## Deferred Ideas

- 监听器健康度面板、offset/lag 可视化、告警和运维 dashboard；这更接近 future requirement `LIST-05`。
- 外部消息总线、分布式事件流、通用 webhook 平台或实时 UI 推送。
- 把 listener 扩展为直接增删改 canonical rows 的“强实时同步”体系。
- 与本阶段目标无关的 watcher/provider 扩张，例如为了本地文件系统单独重做完整监听框架。

</deferred>

---

*Phase: 11-event-driven-refresh-hardening*
*Context gathered: 2026-04-24*
