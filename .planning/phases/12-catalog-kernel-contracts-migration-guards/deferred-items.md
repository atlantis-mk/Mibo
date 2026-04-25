## Deferred Items

- 2026-04-25 — `go test ./internal/httpapi -run TestReadyz -count=1` is currently blocked by an unrelated compile error in `mibo-media-server/internal/library/scan_classify.go` vs `mibo-media-server/internal/library/query_series_grouping.go` (`normalizeSeriesGroupingTitle` redeclared). This plan did not modify those files, so the failure was left untouched per scope boundary rules.
