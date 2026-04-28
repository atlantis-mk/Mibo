## Context

Catalog TV metadata can create durable season and episode descendants for provider-known episodes even when no local file exists. That is useful for governance and missing-episode workflows, but the consumer series detail page currently uses the same hierarchy for its main episode shelf and its primary play affordance is rooted in the series item, which usually has no direct media asset.

The current frontend detail flow loads `GET /api/v1/items/{id}` for item detail and `GET /api/v1/series/{id}/seasons` for series shelves. Episode detail already has playable-asset-aware behavior, but series detail needs a target episode for play/continue. Backend TV helpers already expose available-only episode selection through `GetSeriesNextUp`, but the main series detail contract does not surface a playback target and the default season payload is not scoped to local episodes.

## Goals / Non-Goals

**Goals:**

- Let a series detail page primary play button open a locally available episode.
- Prefer an in-progress local episode for continue playback; otherwise choose the earliest local available episode by season and episode order.
- Make the default consumer episode shelf show only local playable episodes and season counts derived from those displayed episodes.
- Keep missing and unaired descendants queryable for governance, missing episode views, and explicit operational reads.
- Preserve episode detail playback and asset-version selection semantics.

**Non-Goals:**

- Do not remove provider-known missing or unaired catalog descendants from storage.
- Do not implement streaming across multiple episodes as a playlist.
- Do not add new external dependencies or a database migration.
- Do not change metadata matching, scan, or governance repair behavior except where tests need fixtures for local/missing descendants.

## Decisions

1. Add a series playback target to the catalog detail contract.

   The series item detail should expose a nullable playback target containing the selected episode item ID, optional asset ID, label/title, and selection reason such as `continue` or `first_available`. The frontend can keep `/play/$id` rooted at the actual episode target instead of asking the player to understand a synthetic series asset. This avoids adding playlist semantics and keeps progress writes tied to episode item IDs.

   Alternative considered: make the frontend infer the target from the loaded season rails. This is weaker because season rails may be filtered, paginated, or stale relative to progress, and it duplicates backend ordering/progress rules.

2. Teach the playback endpoint to resolve series items defensively.

   Even with a detail target, `GET /api/v1/items/{series_id}/playback` should resolve to the same selected local episode where possible and return the resolved episode `item_id` in the playback source. This keeps deep links and future clients safe if they invoke playback with a series ID directly.

   Alternative considered: require all clients to pre-resolve series playback through detail only. That leaves a known sharp edge and can still produce “no asset” decisions for a series that has local episodes.

3. Filter consumer episode shelves to local playable episodes.

   The series detail presentation should use available episodes only for its default “剧集信息” shelf and omit seasons that have no local available episodes. This can be implemented by extending `ListSeriesSeasons` with an availability mode or by applying the same filter before building frontend rails; backend filtering is preferred so counts and payload size match the consumer contract.

   Alternative considered: keep all provider descendants visible but dim unavailable cards. That conflicts with the requested behavior and makes the main detail page look incomplete or cluttered when many episodes are missing locally.

4. Keep explicit operational reads complete.

   Existing missing and availability-filtered endpoints remain the place to inspect missing/unaired descendants. This preserves the TV hierarchy metadata capability while preventing those descendants from leaking into the default consumer shelf.

   Alternative considered: delete missing descendants once no local file exists. That would break metadata governance and missing episode tracking.

## Risks / Trade-offs

- Progress selection can disagree with visual shelf ordering if one path uses different availability or sort rules -> Use a shared backend helper for available episode ordering and target selection.
- A series can have available status through rollups but no asset on any episode due to inconsistent data -> Return a clear unavailable target/playback decision and keep detail page non-crashing.
- Filtering seasons by available episodes hides missing/unaired context from the main page -> Keep explicit missing and operational views available and covered by tests.
- Adding a detail DTO field requires frontend type updates -> Make the field optional and keep movie/episode detail unchanged.
