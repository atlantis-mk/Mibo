# Regression Baseline

当前回归基线覆盖后端高风险整链和前端关键状态导出，目标是把“现在系统怎么表现”固定成可重复验证的重构门禁。

## Covered Behavior

- 扫描编目：`POST /api/v1/libraries/{id}/scan` 返回 `202`，并创建 `queued` 状态的工作流运行和 `scan_library_path` 任务。
- 监听整链：`storage refresh -> workflow scan -> materialize -> projection refresh` 后，浏览、最近新增、首页分区都能看到新内容。
- 播放详情：`GET /api/v1/items/{id}/playback` 对已链接资源返回可播放源，且 `url` 指向绝对的 `inventory-files/{id}/stream` 地址。
- 元数据治理：`PUT /api/v1/items/{id}/governance/fields` 修改标题后，治理工作区和详情接口立即反映新标题。
- 投影可见性：`PUT /api/v1/items/{id}/governance/projection-visibility` 设置隐藏后，条目从库浏览结果中消失。
- 健康诊断：现有 `internal/httpapi/handlers_health_test.go` 继续约束失败任务聚合、忽略问题、媒体源验证和重扫行为。
- 前端状态：首页、健康中心、媒体展示、继续观看聚合这些关键派生状态有稳定测试基线。

## Test Entry Points

单入口：

```bash
./scripts/run-guardrails.sh
```

分项运行：

```bash
go test ./internal/httpapi -run 'Test(QueueLibraryScanBehavior|CatalogPlaybackBehavior|GovernanceBehaviorUpdatesFieldAndVisibility|Health)'
go test ./internal/library -run 'Test(ListenerRefreshToScanToProjectionEndToEnd|TargetedRefreshWorkflowKeepsScopedRootInProjectionTask|GovernanceVisibilityRemovesHomeProjectionAfterRefresh|MaterializedPlaybackStillResolvesAfterProjectionRefresh|QueueLibraryWorkflow|RunWorkflowScanLibraryPath|QueueLibraryScanWithReason)'
go test ./internal/listener
go test ./internal/playback
go test ./internal/health
```

```bash
cd web
pnpm test -- --run src/features/home/home-state.test.ts src/features/home/home-regression-state.test.ts src/features/health/health-center-state.test.ts src/lib/mibo-query.test.ts src/lib/media-presentation.test.ts src/lib/media-presentation-regression.test.ts src/features/console/ingest-diagnostics.test.ts
pnpm typecheck
```

完整后端回归：

```bash
go test ./...
```

## Maintenance Rules

- 新增扫描、播放、治理、健康相关修复时，优先先补一个面向 HTTP 或 service 公共接口的行为测试。
- 涉及 `listener -> scan -> projection`、`catalog`、`metadata governance` 的中大型重构，合并前至少跑一次 `./scripts/run-guardrails.sh`。
- 优先断言外部可观察结果：HTTP 状态码、响应体、工作流状态、浏览列表可见性、播放源可用性。
- 避免把回归测试绑死到内部 helper、临时字段排序或非关键实现细节。
