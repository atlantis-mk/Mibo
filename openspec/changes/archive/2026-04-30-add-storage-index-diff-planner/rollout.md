## Rollout Defaults

- Local libraries use `fsnotify` hints plus the existing listener reconcile safety net when the worker is enabled.
- OpenList libraries use polling through the configured storage provider every five minutes with `refresh=false` by default to avoid forcing upstream cache refresh on every poll.
- Manual scans, scheduled scans, and listener reconcile remain enabled and continue to be the correctness fallback.
- Storage events posted to `POST /api/v1/storage-events` remain supported and are also recorded as storage index hints.
- Diagnostics are available at `GET /api/v1/storage-change/diagnostics` for active libraries.

## Rollback

- Stop the worker or disable automatic observer startup to stop local/OpenList background detection.
- Existing manual and scheduled scans continue to work without the observers.
- The storage index tables are passive state; leaving them in place does not affect playback or catalog reads when observers are not running.
