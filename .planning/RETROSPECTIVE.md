# Retrospective

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
| v1 | 6 | 13 | 首个完整可用里程碑，建立核心媒体平台边界与播放/同步基础 |
