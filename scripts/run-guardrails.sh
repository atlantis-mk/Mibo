#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

printf '\n[guardrails] backend httpapi regressions\n'
(cd "$ROOT_DIR/mibo-server" && go test ./internal/httpapi -run 'Test(QueueLibraryScanBehavior|CatalogPlaybackBehavior|GovernanceBehaviorUpdatesFieldAndVisibility|Health)')

printf '\n[guardrails] backend library end-to-end guardrails\n'
(cd "$ROOT_DIR/mibo-server" && go test ./internal/library -run 'Test(ListenerRefreshToScanToProjectionEndToEnd|TargetedRefreshWorkflowKeepsScopedRootInProjectionTask|GovernanceVisibilityRemovesHomeProjectionAfterRefresh|MaterializedPlaybackStillResolvesAfterProjectionRefresh|QueueLibraryWorkflow|RunWorkflowScanLibraryPath|QueueLibraryScanWithReason)')

printf '\n[guardrails] backend listener\n'
(cd "$ROOT_DIR/mibo-server" && go test ./internal/listener)

printf '\n[guardrails] backend playback\n'
(cd "$ROOT_DIR/mibo-server" && go test ./internal/playback)

printf '\n[guardrails] backend health\n'
(cd "$ROOT_DIR/mibo-server" && go test ./internal/health)

printf '\n[guardrails] frontend state regressions\n'
(cd "$ROOT_DIR/web" && pnpm test -- --run src/features/home/home-state.test.ts src/features/home/home-regression-state.test.ts src/features/health/health-center-state.test.ts src/lib/mibo-query.test.ts src/lib/media-presentation.test.ts src/lib/media-presentation-regression.test.ts src/features/console/ingest-diagnostics.test.ts)

printf '\n[guardrails] frontend typecheck\n'
(cd "$ROOT_DIR/web" && pnpm typecheck)

printf '\n[guardrails] all guardrails passed\n'
