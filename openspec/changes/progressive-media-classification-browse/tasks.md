## 1. Work-group Classification Core

- [x] 1.1 Add a reusable work-group classification model in `mibo-media-server/internal/library` that can represent movie, movie-version-group, series-hierarchy, and unresolved outcomes with confidence and evidence.
- [x] 1.2 Feed path-tree assignments, content-shape assignments, filename episode signals, sidecar hints, and supported external identity types into the work-group classifier before final materialization decisions are made.
- [x] 1.3 Add conflict handling so mixed movie-versus-series evidence produces a guarded or review-required result instead of a forced movie collapse.

## 2. Metadata Collapse Gating

- [x] 2.1 Route movie materialization through the new confidence gate so weak `title + year` fallback cannot directly create or reuse final movie metadata when stronger series evidence exists.
- [x] 2.2 Preserve current strong episode hierarchy materialization while making work-group acceptance the source of truth for movie-versus-series collapse.
- [x] 2.3 Ensure guarded or review-required groups keep inventory/resource facts intact and emit classification evidence without creating duplicate final metadata items.

## 3. Progressive Browse Upgrade

- [x] 3.1 Update browse assembly so unresolved inventory/resource-backed organizing entries remain visible until accepted metadata-backed entries are ready.
- [x] 3.2 Ensure accepted metadata-backed browse entries replace their unresolved predecessors instead of appearing alongside them in the same library view.
- [x] 3.3 Refresh affected projection and browse upgrade paths after accepted work-group decisions so organizing cards convert cleanly on the next query refresh.

## 4. Sidecar and Identity Promotion

- [x] 4.1 Promote parsed sidecar `media_type`, series hierarchy hints, and supported external identity provider types into primary work-group classification evidence.
- [x] 4.2 Add precedence rules so strong sidecar or provider identity evidence wins over weaker filename-only fallback or triggers review-required handling when evidence conflicts.

## 5. Regression Coverage

- [x] 5.1 Add focused library classification tests for movie version groups, season/episode directories, numeric episode names, and weak movie fallback conflicts.
- [x] 5.2 Add sidecar-driven classification tests covering movie folder metadata, tvshow metadata, and sidecar-versus-filename conflicts.
- [x] 5.3 Add browse/API regression tests proving unresolved entries stay visible before metadata collapse and are replaced once accepted metadata-backed browse results exist.
