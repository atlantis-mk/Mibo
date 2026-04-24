# Phase 11 Discussion Log

**Phase:** 11 - Event-Driven Refresh Hardening
**Date:** 2026-04-24
**Mode:** discuss
**Language:** Chinese
**Status:** Decisions locked for planning

## Sources Reviewed Before Questions

- Workflow / template:
  - `/root/.config/opencode/get-shit-done/workflows/discuss-phase.md`
  - `/root/.config/opencode/get-shit-done/workflows/discuss-phase-assumptions.md`
  - `/root/.config/opencode/get-shit-done/workflows/discuss-phase-power.md`
  - `/root/.config/opencode/get-shit-done/templates/context.md`
- Planning context:
  - `.planning/PROJECT.md`
  - `.planning/REQUIREMENTS.md`
  - `.planning/STATE.md`
  - `.planning/ROADMAP.md`
  - `.planning/phases/07-metadata-governance-matching/07-CONTEXT.md`
  - `.planning/phases/08-native-search-discovery-filters/08-CONTEXT.md`
  - `.planning/phases/09-trailer-discovery-playback/09-CONTEXT.md`
  - `.planning/phases/10-scheduled-operations-control/10-CONTEXT.md`
- Code / implementation anchors:
  - `mibo-media-server/internal/httpapi/router.go`
  - `mibo-media-server/internal/httpapi/handlers_storage_events.go`
  - `mibo-media-server/internal/library/service.go`
  - `mibo-media-server/internal/library/scan.go`
  - `mibo-media-server/internal/library/scan_run.go`
  - `mibo-media-server/internal/library/scan_upsert.go`
  - `mibo-media-server/internal/library/scan_reconcile.go`
  - `mibo-media-server/internal/worker/worker.go`
  - `mibo-media-server/internal/search/service.go`
  - `mibo-media-server/internal/database/models.go`
  - `mibo-media-server/internal/httpapi/router_test.go`
  - `mibo-media-server/internal/worker/worker_test.go`
  - `mibo-media-server/internal/library/identity_reconcile_test.go`

## Pre-Question Synthesis

- Phase 11 不是从零开始做 listener；代码里已经有 `/api/v1/storage-events`、targeted refresh job、partial scan 和 fallback reconciliation 的基础实现。
- 已有项目级决策已经锁定：listener 只能 enqueue refresh / reconciliation work，不能直接修改 canonical 媒体数据。
- 路线图真正还没拍板的是运行策略，而不是底层能力是否存在：
  - `create/update/delete/move` 应该多保守或多激进。
  - event storm 如何合并，避免重复扫描。
  - 删除/移动是否即时失效，还是保守收敛。
  - 漏事件后如何兜底补偿。
- 基于 REQUIREMENTS / ROADMAP / 现有代码，讨论应聚焦产品行为边界，不去提前替 planner 决定具体表结构、字段名或内部模块切分。

## Selected Gray Areas

- 事件触发语义
- 去重与合并窗口
- 删除与移动安全策略
- Reconciliation 补偿方式

## Questions Asked And Answers

### 1. 事件触发语义
- Question: `1. 事件触发语义，你更倾向哪种？`
- Options presented:
  - `保守定向刷新（推荐）`
  - `删除移动全量`
  - `你决定`
- Answer: `保守定向刷新（推荐）`
- Why this mattered:
  - 路线图要求的是 safe listener-driven refresh，而不是任何事件都拉升成全库扫描。
  - 当前代码已经具备按路径归一 targeted refresh 的基础，最自然的 v2 方向是硬化这条路径，而不是退回粗暴 full sync。

### 2. 去重与合并窗口
- Question: `2. 去重与合并窗口，你更倾向哪种？`
- Options presented:
  - `窗口合并（推荐）`
  - `仅唯一键去重`
  - `你决定`
- Answer: `窗口合并（推荐）`
- Why this mattered:
  - 当前只有 job uniqueness，还不足以抵抗 NAS/网盘抖动、批量改名和目录级重复通知。
  - LIST-03 明确要求系统对突发事件做去抖或合并，而不是仅靠偶然的唯一键命中。

### 3. 删除与移动安全策略
- Question: `3. 删除与移动的安全策略，你更倾向哪种？`
- Options presented:
  - `保守收敛（推荐）`
  - `立即失效`
  - `你决定`
- Answer: `保守收敛（推荐）`
- Why this mattered:
  - 当前 partial scan、missing 标记和 fallback reconciliation 已经形成一套偏保守的收敛链路。
  - 如果 ingest 时立即把内容硬判失效，很容易在远程存储抖动、延迟传播或 rename 乱序下误伤已有媒体状态。

### 4. Reconciliation 补偿方式
- Question: `4. Reconciliation 补偿方式，你更倾向哪种？`
- Options presented:
  - `周期兜底（推荐）`
  - `仅事件驱动`
  - `你决定`
- Answer: `周期兜底（推荐）`
- Why this mattered:
  - LIST-04 已明确要求存在兜底 reconciliation，不能把 listener 当成绝对可靠输入。
  - 既有 jobs/worker/schedule 基础已经存在，最合理的方向是复用这些后台能力做补偿，而不是追求事件流完美交付。

## Locked Outcomes

- `create/update/delete` 默认走受影响目录的 targeted refresh。
- `move/rename` 在可安全归一时走共同祖先目录 targeted refresh；只有无法归一时才回退 full sync。
- Phase 11 必须新增显式时间窗合并策略，以 `library + normalized root` 为主做去抖/合并，并允许必要时提升到更高祖先目录。
- 删除/移动事件不在 ingest 阶段立即硬判媒体失效，而是通过 targeted refresh、partial scan、missing/provisional 语义和 reconcile 保守收敛。
- 系统必须保留周期性兜底 reconciliation，以恢复漏事件并降低库状态漂移。
- listener 始终只是 refresh 触发器，不直接修改 canonical 媒体数据。

## Out-Of-Scope Reminders Captured During Discussion

- 不把 listener 扩展成通用实时系统、外部消息总线或实时推送平台。
- 不在本阶段引入 direct row mutation 的“强实时同步”设计。
- 不提前扩展监听器健康大盘、复杂告警或运维 dashboard；这些更接近 future requirement `LIST-05`。

## Output Produced

- `.planning/phases/11-event-driven-refresh-hardening/11-CONTEXT.md`
