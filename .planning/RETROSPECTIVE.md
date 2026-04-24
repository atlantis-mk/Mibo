# Retrospective

## Milestone: v2 — Product Discovery And Operations

**Shipped:** 2026-04-24
**Phases:** 5 | **Plans:** 18

### What Was Built

- 元数据治理、重新匹配和重抓能力，为 discovery 质量提供管理员控制面
- 原生搜索、搜索历史、共享筛选和 discovery projection 新鲜度链路
- TMDB 预告片同步、详情页入口和页面内预告片播放体验
- 产品内计划任务管理、run-now、历史记录和 worker 驱动的调度执行
- 存储事件监听到 coalesced targeted refresh 的安全链路，并由 reconciliation 兜底

### What Worked

- 继续把媒体语义写入留在 `mibo-media-server`，前端和存储入口都只通过稳定 API/worker 边界协作
- 计划任务和监听刷新都复用现有 jobs/worker 生命周期，减少了新的后台执行模型
- Phase 11 的 verifier gaps 能够通过 `--gaps` 生成精确补强计划，并在重新验证中关闭到 10/10

### What Was Inefficient

- 多个规划文档在阶段完成后仍有 stale 状态，milestone closeout 需要额外同步 `PROJECT.md`、`REQUIREMENTS.md` 和 `ROADMAP.md`
- `audit-open` 对 quick-task summary 文件名有严格约定，历史 quick task 需要补充标准 `SUMMARY.md` 才能通过 close audit
- Phase 11 初次验证才暴露并发 coalescing 与 OpenList `/` root 边界问题，说明高风险监听语义应更早纳入并发/默认配置测试

### Patterns Established

- Discovery 能力优先使用产品内 projection 与 lifecycle refresh，不引入外部搜索中间件
- 后台运营能力统一进入 schedules/jobs/worker 模型，避免为调度、监听、手动执行维护三套状态机
- 活跃监听意图通过独立 guard table 控制唯一性，历史 job rows 保持可复用和可审计

### Key Lessons

- 默认 OpenList 根路径 `/` 是必须覆盖的真实配置，路径边界测试不能只覆盖非根目录
- Listener/reconcile 这类 storm-prone 入口需要并发回归，不应只证明顺序 coalescing
- Closeout 前应运行 open artifact audit，避免 milestone 归档时才发现历史 quick-task 元数据不规范

## Milestone: v1 — MVP

**Shipped:** 2026-04-22
**Phases:** 6 | **Plans:** 13

### What Was Built

- 统一 setup/auth/app-entry 边界
- 媒体源与媒体库接入、异步扫描与任务观测
- 语义目录、首页发现流、TV season/episode 详情
- 统一播放入口与 canonical progress
- capability-aware 播放决策和明确回退路径
- 稳定身份、增量刷新和存储事件 intake

### What Worked

- 先收紧 API 边界，再向上叠加目录和播放能力，阶段依赖清晰
- 以 SUMMARY/VERIFICATION/UAT 驱动关闭阶段，能较快定位 closeout 缺口
- 让 `mibo-media-server` 保持业务核心，避免和 OpenList 边界混乱

### What Was Inefficient

- 根仓库与 `web/` 子仓库分离，提交与归档时需要双重检查
- closeout 时才发现部分 `ROADMAP.md` / `REQUIREMENTS.md` 漂移，说明阶段结束时的文档同步还不够稳定
- `audit-open` 仍将 `resolved` 的 UAT 文件视为噪音项

### Patterns Established

- 媒体产品能力统一经过 `mibo-media-server` API 边界
- 播放与目录行为通过 typed contracts 在前后端同步
- 新扫描/刷新入口只入队后台任务，不直接在边界层修改媒体数据

### Key Lessons

- library browse 页必须保持稳定浏览态，不能为了“默认详情”破坏可导航性
- closeout 前的人工验收应尽早完成，否则会把真实 UI 缺口拖到里程碑末尾
- 稳定身份与增量刷新要先保守正确，再追求更激进的自动归并

## Cross-Milestone Trends

| Milestone | Phases | Plans | Notes |
|-----------|--------|-------|-------|
| v2 | 5 | 18 | 发现、预告片、元数据治理、计划任务和监听刷新硬化全部归档 |
| v1 | 6 | 13 | 首个完整可用里程碑，建立核心媒体平台边界与播放/同步基础 |
