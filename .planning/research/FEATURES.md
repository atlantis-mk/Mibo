# Feature Landscape

**Domain:** Self-hosted household media server / 家庭媒体系统
**Project:** Mibo
**Researched:** 2026-04-21
**Overall confidence:** MEDIUM-HIGH

## Executive Take

The market baseline is clear: users expect a household media server to let them add libraries, automatically organize movies and shows, browse/search cleanly, play on multiple device types, and keep watch progress in sync. Jellyfin, Plex, and Emby all present some form of this core loop as the product itself, not an advanced add-on. For Mibo, these are not differentiators; they are entry tickets.

For the next milestone, the best differentiators are not flashy “consumer streaming platform” features. They are reliability and architecture-aligned capabilities that remove pain for real self-hosters: stable ingestion from heterogeneous storage, fast incremental updates, direct-play-first playback with graceful fallback, and a client-agnostic API/progress model that works for Web/mobile/TV.

The biggest trap is copying Plex’s surface area instead of solving Mibo’s actual job. Live TV, DVR, social watch parties, ad-supported content, and broad music/photo/books ecosystems all exist in competitors, but they would distract this milestone from the project’s chosen architecture and brownfield refactor goals.

## Table Stakes

Features users expect. Missing = product feels incomplete.

| Feature | Why Expected | Complexity | Notes for Mibo |
|---------|--------------|------------|----------------|
| Initial setup + admin onboarding | Jellyfin and Plex both frame setup as a quick wizard and first-run flow. Users expect to reach a usable library quickly. | Low | Already partially present; next milestone should keep setup aligned with new architecture, not re-invent it. |
| Source and library management | Plex/Jellyfin/Emby all treat adding media libraries as a first-class action. Without this, the server is not useful. | Medium | Must support storage source selection, path validation, and library type boundaries. |
| Library scan and refresh | Users expect newly added files to appear after scan/refresh. | Medium | Manual scan is table stakes now; scheduled refresh is also expected. Event-driven updates can be deferred to differentiator tier. |
| Automatic organization into movies / series / seasons / episodes | Plex and Jellyfin both sell “automatic sorting/organization” as core value. | High | Essential for Mibo because current re-scope is explicitly moving toward stable media semantics. |
| Metadata enrichment (posters, overviews, cast/basic details) | Beautiful, metadata-rich browsing is the standard UI expectation, not a premium differentiator. | Medium | Fast-path filename parsing plus async metadata matching fits the architecture. |
| Browse, filter, and search library content | Competitors all expose home, library, details, and search flows. | Medium | Search does not need to be “smart”; it must be fast and predictable. |
| Detail pages for media items | Users expect artwork, synopsis, season/episode breakdown, runtime/basic info before pressing play. | Medium | Depends on semantic model and metadata completeness. |
| Playback that usually “just works” | Direct play, remux, or transcode fallback is the normal expectation in Jellyfin/Plex ecosystems. | High | Mibo should implement “direct link first, fallback when necessary” exactly as the architecture doc recommends. |
| Resume playback + watch progress sync | Plex explicitly highlights tracking what you watch; this is expected across devices. | Medium | Already partially present; must become stable and API-consistent across clients. |
| Multi-device access (Web now, mobile/TV-ready API) | Jellyfin/Plex/Emby all emphasize access from many device classes. | Medium | For this milestone, the table-stakes requirement is API compatibility for multiple clients, not shipping every client app. |
| Authentication and household-safe access control | Emby/Plex both expose home/managed accounts and parental controls because household usage is multi-user by default. | Medium | At minimum: authenticated access, user separation, and room for profile restrictions later. |
| Remote/local network access basics | Jellyfin docs explicitly document local-only and remote access setups. Users expect at least a workable path to access outside one browser tab. | Medium | For this milestone, support simple remote-friendly deployment; do not overbuild remote relay infrastructure. |

## Differentiators

Features that set product direction apart. Not universally expected, but highly valuable for this project.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Unified storage ingestion across local / NAS / cloud via adapter boundary | Solves a real self-hosted pain: users have mixed storage, but want one media experience. Strong fit with `OpenList + StorageProvider`. | High | This is the most strategic differentiator because it matches project constraints and avoids deep OpenList coupling. |
| Stable file identity + rename/move resilience | Prevents duplicate items, lost progress, and expensive rescans when files move or storage remounts. | High | Important for brownfield cleanup and future large-library correctness. |
| Incremental and event-driven sync | Makes the system feel responsive on real household libraries without full rescans. | High | Manual + scheduled scans are table stakes; event/delta sync is where Mibo can feel materially better. |
| Fast-path scan pipeline with deferred heavy work | New media appears quickly, while metadata/ffprobe/transcode prep completes asynchronously. Better UX than “scan blocks everything.” | Medium | Strongly aligned with Worker separation in the architecture doc. |
| Direct-play-first playback policy with explicit fallback behavior | Good for home deployments where CPU is limited and storage/network capabilities vary. Users care about reliability more than “always transcode.” | Medium | This should be a deliberate product promise, not just an internal implementation detail. |
| Unified progress/history model across Web, mobile, and TV clients | Many self-hosted systems become inconsistent across clients. A single durable API contract is a real advantage. | Medium | Good milestone-level differentiator because it improves future clients without needing all clients now. |
| Operational simplicity for self-hosters | Single-image or simple deployment, minimal mandatory infra, works with SQLite/Postgres, optional Redis later. | Medium | Not flashy, but a genuine differentiator versus overbuilt stacks. |
| Household-focused “Continue Watching” and recently added surfaces | This is the smallest high-value UX differentiator to add once core library semantics are stable. | Low-Medium | Worth doing after metadata + progress are reliable; should not precede core ingestion/playback work. |

## Anti-Features

Features to explicitly NOT build for this milestone/project direction.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Deeply forking or embedding media business logic into OpenList | Conflicts with explicit architecture boundary and increases long-term maintenance cost. | Keep OpenList as storage gateway only; put media semantics in `mibo-media-server`. |
| Building a full custom storage protocol stack now | Reimplements what OpenList already provides and slows milestone delivery. | Preserve `StorageProvider` and add direct adapters only after measured hotspots. |
| Plex-style ad-supported streaming, rentals, or third-party content aggregation | Not aligned with “self-hosted household media” value proposition and creates huge scope creep. | Focus on personal media libraries only. |
| Full Live TV / DVR / tuner ecosystem | Competitors offer it, but it is a separate product area with very different UX and backend requirements. | Defer entirely until core library/playback architecture is stable. |
| Broad “everything server” scope (music, books, photos, comics, home video power features) in this milestone | Mibo architecture and current codebase are centered on video library semantics. Expanding horizontally now dilutes the rewrite. | Prioritize movies + TV semantics first; treat other media types as future expansions. |
| Watch-party / social features first | Nice-to-have but not a reason users adopt a home media server. | Get single-user and household playback/progress reliability solid first. |
| Heavy recommendation/ML discovery systems | Low immediate value versus metadata correctness, scan speed, and playback reliability. | Implement deterministic browse/search and “continue watching” first. |
| Always-on aggressive transcoding/pre-optimization pipelines | Expensive, operationally noisy, and unnecessary for V1 household deployments. | Prefer direct play/remux; transcode only when client/storage constraints demand it. |
| Enterprise multi-tenant administration | This is a household product direction, not SaaS media infrastructure. | Keep auth/user model simple and family-oriented. |

## Feature Dependencies

```text
Setup + auth → source management → library management → scan jobs
Scan jobs → media file records → media semantic model (items/series/seasons/episodes)
Media semantic model → metadata enrichment → details pages / browse surfaces / search
Media file records + storage adapter → playback link generation
Playback link generation + client capability handling → direct play / remux / transcode fallback
Playback → progress reporting → continue watching / resume playback
Stable file identity → incremental sync → reliable progress preservation across rename/move events
Unified API contract → Web/mobile/TV consistency
```

## MVP Recommendation For The Next Milestone

Prioritize:

1. **Reliable library ingestion and semantic modeling**
   - Includes: source/library management, scan jobs, `media_items / series / seasons / episodes`, metadata enrichment.
   - Why: everything user-facing depends on this.

2. **Playback reliability over playback breadth**
   - Includes: playback link generation, direct-play-first behavior, basic fallback path, stable detail pages.
   - Why: users forgive missing premium features; they do not forgive pressing play and failing.

3. **Progress sync + household home surfaces**
   - Includes: resume, continue watching, recently added.
   - Why: this turns a file browser into a real household media product.

4. **Incremental sync as the first true differentiator**
   - Includes: stable identity groundwork, delta/event-driven refresh path.
   - Why: this most directly improves day-2 operations for real users and aligns with architecture goals.

Defer:

- **Live TV / DVR**: separate product track, not a milestone extension.
- **Music / photos / books expansion**: too broad for current domain refocus.
- **Watch-party/social features**: lower leverage than reliable playback/progress.
- **Custom direct storage adapters beyond OpenList**: only after profiling shows real need.

## Recommendation Summary

For this project direction, treat **library onboarding, scan/organization, metadata, browse/search, playback, and progress sync** as baseline requirements. Put milestone energy into **stable media semantics, worker-based ingestion, direct-play-first playback, and incremental sync**. Those are the features that best translate the improved architecture into visible user value.

In short:

- **Table stakes:** make personal video libraries usable and dependable.
- **Differentiators:** make heterogeneous storage and ongoing updates feel effortless.
- **Anti-features:** anything that turns the milestone into “build all of Plex.”

## Sources

### Project Context

- `/Users/atlan/Desktop/IdeaProjects/Mibo/.planning/PROJECT.md` — project scope, constraints, active requirements. **Confidence: HIGH**
- `/Users/atlan/Desktop/IdeaProjects/Mibo/docs/media-architecture/improved-architecture.md` — target architecture and evolution order. **Confidence: HIGH**

### Competitor / Ecosystem Signals

- Jellyfin homepage — media types, clients, SyncPlay, privacy/self-hosting positioning: https://jellyfin.org **Confidence: HIGH**
- Jellyfin Quick Start — setup wizard, add media, optional remote access: https://jellyfin.org/docs/general/quick-start/ **Confidence: HIGH**
- Jellyfin Networking — local vs remote access expectations for self-hosted deployments: https://jellyfin.org/docs/general/post-install/networking/ **Confidence: HIGH**
- Jellyfin Transcoding — direct play/remux/transcode model and hardware implications: https://jellyfin.org/docs/general/post-install/transcoding/ **Confidence: HIGH**
- Plex “What is Plex?” — auto cataloging, artwork/info, device coverage, remote access, watch tracking, DVR expectations: https://support.plex.tv/articles/200288286-what-is-plex/ (Last modified Jan 7, 2025) **Confidence: HIGH**
- Plex Features index — feature surface for remote access, managed households, watch together, downloads, music, DVR, intros/credits, webhooks: https://support.plex.tv/articles/categories/features/ **Confidence: HIGH**
- Plex “Your Media” — automatic organization and device streaming positioning: https://www.plex.tv/your-media/ **Confidence: MEDIUM** (marketing page, but still official)
- Emby homepage/about surface — remote access, live TV, parental controls, DLNA, startup wizard, multi-device positioning: https://emby.media **Confidence: MEDIUM** (official marketing surface, less precise than docs)

## Confidence Notes

- **HIGH confidence:** core table stakes around library setup, organization, playback, progress, and multi-device access. All major competitors present these as baseline capabilities.
- **MEDIUM confidence:** exact cutoff between “table stakes” and “differentiator” for remote access and household controls varies by target user sophistication; recommendation here is tuned to Mibo’s current architecture and milestone scope.
- **LOW confidence:** none of the core recommendations rely on unverified third-party commentary.
