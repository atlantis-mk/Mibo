## Context

Mibo already has the foundations needed for an operator console: authenticated app routes, a shadcn/Radix sidebar layout, settings routes, system info, library/source APIs, jobs, schedules, catalog consistency operations, playback/progress services, and catalog counts on library details. The current product navigation is media-consumer oriented, while operational controls are split across settings pages and backend endpoints.

The console should be introduced as a first-class admin surface without changing the consumer homepage behavior. It should reuse the existing TanStack Router file-route structure, `AppSidebar`, API client/query helpers, and existing backend services where possible. Backend work should be limited to dashboard-friendly read-only aggregation plus small action wrappers for operations that already exist.

## Goals / Non-Goals

**Goals:**

- Add a `/console` admin dashboard route for authenticated users.
- Present server identity, status, version, port, storage provider, database, uptime, access addresses, media counts, and module health in a compact status area.
- Present recent operational activity and playback-oriented events in a timeline.
- Provide grouped management navigation modeled after Emby-style admin consoles but adapted to Mibo concepts.
- Surface quick actions for scans, logs, catalog consistency checks, projection rebuilds, settings, and future server actions.
- Keep the first implementation incremental and compatible with the existing setup/auth gate.
- Make desktop the primary layout while preserving usable mobile behavior through the existing sidebar and responsive card stacking.

**Non-Goals:**

- Do not implement a full Emby clone or introduce every Emby setting category as a working feature.
- Do not add destructive shutdown/restart behavior until the backend has explicit lifecycle support and guardrails.
- Do not build new flows on retired legacy media routes such as `/api/v1/media-items/*` or `/api/v1/media-files/*`.
- Do not replace existing settings/library/schedule/log pages; the console should link into or summarize them.
- Do not introduce a new external UI framework or telemetry dependency.

## Decisions

### Decision 1: Add a dedicated console route instead of replacing the homepage

The dashboard will live at `/console` under the authenticated `_app` route group. The current homepage remains consumer-focused, and the sidebar gets a new high-priority `控制台` entry.

Alternatives considered:

- Replace `/` with the admin console: rejected because Mibo's existing homepage is library/media discovery oriented and should remain useful to normal playback users.
- Put the console under `/settings`: rejected because the requested design is an operational dashboard, not a settings subsection.

### Decision 2: Use the existing app shell/sidebar, then extend it with grouped admin sections

The first version should evolve `AppSidebar` rather than adding a parallel shell. The sidebar can gain grouped sections such as Mibo Web, server, media management, devices, and advanced operations, while still showing existing library shortcuts.

Alternatives considered:

- Create a completely separate admin layout: rejected for first version because it duplicates responsive sidebar, search, auth, and route guard behavior.
- Keep only the current simple sidebar: rejected because the console requires denser grouped administration entry points.

### Decision 3: Add a dashboard summary API if existing endpoints are too fragmented

Introduce a small authenticated `GET /api/v1/admin/console` endpoint when implementation confirms existing APIs cannot provide the page efficiently. The response should aggregate read-only data from config, database, storage provider, system info, libraries, jobs/schedules, and catalog counts. It should avoid complex side effects and should tolerate partial failures by returning warning statuses for individual sections when possible.

Expected shape:

- `server`: name, service, version, update status, port, uptime, database, storage provider, root path.
- `access`: local URL, LAN URL candidates if discoverable, configured remote URL if available, and remote status.
- `media`: library count, media source count, item/file/person/series/episode counts when available.
- `health`: database/storage/module status summaries.
- `activity`: recent dashboard events with type, severity, user, device, media title, message, and timestamp.
- `devices`: connected/recent device summaries when available, otherwise an empty or unsupported state.
- `quick_actions`: action descriptors with labels, route targets, API targets, disabled state, and risk level.

Alternatives considered:

- Compose the dashboard entirely from frontend calls to many existing endpoints: acceptable as an interim fallback, but it risks slow loading, duplicated error handling, and inconsistent partial states.
- Add many specialized endpoints for each card: rejected because it increases API surface before the console interaction model is proven.

### Decision 4: Treat activity as an operational timeline with graceful placeholders

The console should render real events where Mibo already records them, such as recent progress/playback records, job history, schedule history, scan events, source changes, and warnings. Missing categories should show empty or unsupported states rather than fake production data.

Alternatives considered:

- Build a new persistent audit-event subsystem first: rejected for the initial dashboard because it expands scope across many services.
- Use only static placeholder data: rejected because the console must be useful for operational work.

### Decision 5: Keep high-risk actions explicit and bounded

Quick actions should route to existing safe screens or call bounded APIs. Potentially disruptive operations such as projection rebuilds, scans, or future shutdown actions need clear labels, confirmation where appropriate, and success/failure feedback. Unsupported actions such as Premiere status, camera upload, DLNA, plugins, and server shutdown should appear as unavailable or future-capability entries rather than hidden if they are important to the requested console map.

Alternatives considered:

- Hide all unavailable entries: rejected because the requested design uses a complete management map and unavailable states help communicate roadmap boundaries.
- Implement all actions immediately: rejected because many entries represent separate features outside a console shell.

### Decision 6: Use existing Mibo visual language with Emby-inspired information architecture

The UI should use white/light-gray surfaces, green success accents, dense cards, timeline rows, and grouped sidebar navigation. It should preserve current shadcn primitives, Tailwind tokens, and Mibo branding instead of copying Emby's exact styles.

Alternatives considered:

- Exact Emby visual clone: rejected because Mibo should remain distinct and consistent with the current app.
- Current media-home visual treatment: rejected because the console needs higher information density and stronger operational affordances.

## Risks / Trade-offs

- Dashboard aggregation can become a dumping ground -> Keep the first API response focused on summary data and avoid embedding full library/job/detail payloads.
- Activity data may be incomplete at first -> Render clear empty states and use existing playback/progress/job records before adding a generalized audit log.
- Sidebar density can hurt regular media browsing -> Keep consumer routes visible, group admin routes clearly, and ensure mobile collapse still works.
- Update-status/version checks may require external network calls -> Start with local version and `unknown` or `not_configured` update status unless a safe update source already exists.
- LAN address discovery can be platform-sensitive -> Provide local API base and best-effort LAN candidates, with `unavailable` status when not discoverable.
- Quick actions can trigger expensive work -> Require confirmation for rebuild or broad scan actions and show job status after submission.
- Admin authorization may be under-specified -> Gate the route with existing auth initially and add role checks if the current auth model exposes reliable admin roles.

## Migration Plan

1. Add backend summary and activity support only where existing endpoints cannot cover the console efficiently.
2. Add typed API client/query helpers for dashboard data and quick actions.
3. Add `/console` route and sidebar entry while leaving existing routes unchanged.
4. Build the console UI behind the existing setup/auth gate.
5. Verify `pnpm typecheck` in `web/` and focused backend tests for any new handlers.
6. Rollback by removing the `/console` route/sidebar entry and any new dashboard-only backend routes; existing media and settings behavior should remain unaffected.

## Open Questions

- Should `/console` require an admin role immediately, or is the existing authenticated user gate sufficient for the current Mibo deployment model?
- What source should be authoritative for server version and update status?
- Should device tracking be backed by real sessions immediately, or should the first version only show recent playback clients from progress/playback data?
- Should log viewing be implemented as a real endpoint in this change, or should the console initially link to a placeholder/settings route until log streaming is designed?
