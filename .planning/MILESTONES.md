# Milestones: Mibo

## v2 Product Discovery And Operations (Shipped: 2026-04-24)

**Phases completed:** 5 phases, 18 plans, 30 tasks

**Key accomplishments:**

- TMDB trailer metadata now syncs into Mibo-owned detail data, and users can open a single selected trailer from the detail info area in an in-page modal player.
- Persisted schedule rows with typed daily, weekly, and monthly recurrence plus schedule-centric run history foundations
- Scoped library scan, cleanup, and invalid-link executors ready for schedule-driven maintenance work
- Batch metadata refetch, trailer sync, and artwork refresh executors wired through existing metadata ownership paths
- Authenticated schedule CRUD, history, and run-now APIs exposed over the existing jobs-based execution path
- Due schedules now enqueue through the worker queue and feed schedule-centric run history from real job execution outcomes
- Dedicated schedules route with typed API/query contracts, list management surface, and detail-layer history shell
- Settings now advertises the schedules workspace as a lightweight summary and jump-off surface instead of hosting the full management flow
- Durable listener coalescing with 15-second debounce windows, safe ancestor promotion, and six-hour reconciliation intent per library
- Authenticated storage-event ingress now validates library boundaries and delegates create/update/delete/move refresh intent to the listener service.
- Worker dispatch now turns coalesced listener jobs into existing scan queue work and keeps active libraries covered by self-reseeding reconciliation jobs
- OpenList libraries rooted at `/` now accept valid absolute child storage events and enqueue targeted listener refresh intent.
- Durable active-intent guards now serialize concurrent listener refresh and reconcile creation without making historical job keys unique.

---

## v1 MVP

**Shipped:** 2026-04-22
**Phases:** 6
**Plans:** 13
**Tasks:** 40+
**Timeline:** 2026-04-21 → 2026-04-22
**Approx. code footprint:** 32,757 lines of Go + TypeScript
**Milestone diff:** 59 files changed, 15,879 insertions, 2 deletions

### Delivered

- 建立了统一的初始化、登录与应用入口边界
- 交付媒体源/媒体库配置、异步扫描、任务观测与重试
- 交付电影/剧集语义目录、首页发现流和 season-first TV 详情
- 打通统一播放入口、续播/重播和跨端进度语义
- 上线按客户端能力决策的 direct / fallback / unplayable 播放响应
- 建立稳定身份、增量刷新和存储事件驱动同步基础能力

### Known Deferred Items

- Known deferred items at close: 0

### Archives

- `.planning/milestones/v1-ROADMAP.md`
- `.planning/milestones/v1-REQUIREMENTS.md`
- `.planning/v1-MILESTONE-AUDIT.md`
