# Domain Pitfalls

**Domain:** 家庭媒体系统 / household media server  
**Researched:** 2026-04-21  
**Overall confidence:** MEDIUM-HIGH

This project is **greenfield with brownfield constraints**: there is already a working codebase, but the main risk is not “can it play a file?” — it is **locking the system into the wrong boundaries while the library, clients, and performance needs grow**.

The most common failure mode in this domain is treating a household media server like a thin file browser. Mature systems converge on the opposite model: **storage is one concern; media semantics, playback policy, and user state are separate concerns**. Your architecture doc already points in that direction (`OpenList` for storage access, `mibo-media-server` for media/product logic). The pitfalls below are the places projects usually drift away from that boundary.

---

## Critical Pitfalls

### 1) Path-based identity instead of stable file identity
**What goes wrong:** The system treats `storage_id + path` as the file’s identity, so renames, moves, mount changes, reorganizations, and duplicate scans create new records instead of updating existing ones.

**Why it happens:** Path is the easiest thing to index early. Projects postpone stable identity until after scanning “works.”

**Consequences:**
- duplicate media items after library cleanup
- lost watch progress after rename/move
- bad incremental scan behavior
- impossible-to-trust deduplication and “continue watching”

**Warning signs:**
- a rename causes delete + recreate rather than move detection
- progress is attached to file rows that churn after rescans
- users report “same movie appears twice” after reorganizing folders
- scanner needs expensive full-library comparisons to detect changes

**Prevention strategy:**
- define a `StableIdentity` capability on `StorageProvider` early, even if some adapters only partially support it in V1
- store both **logical media identity** and **source file identity** separately
- treat path as mutable metadata, not canonical identity
- add move/rename reconciliation rules before shipping aggressive incremental sync
- keep progress attached to media/version identity, not raw scan row identity

**Phase to address:** **Phase 1-2** — storage boundary + scanner foundation

---

### 2) Doing full media understanding inside the scan loop
**What goes wrong:** A “scan” tries to traverse storage, parse filenames, call TMDB/TVDB, run `ffprobe`, generate artwork, and sometimes even prepare playback artifacts in one synchronous pipeline.

**Why it happens:** Early prototypes optimize for fewer concepts, not for throughput or failure isolation.

**Consequences:**
- slow scans block API capacity
- remote metadata outages stall ingestion
- a few bad files poison whole-library refreshes
- retries become non-idempotent and expensive

**Warning signs:**
- “scan library” latency scales with metadata provider latency
- worker and API share the same hot resources with no queue isolation
- one failed lookup forces full rescan reruns
- CPU spikes during scans because probe/transcode work sneaks into request paths

**Prevention strategy:**
- keep scan phase as a **fast path**: discover files, basic classification, enqueue follow-up jobs
- split jobs into `scan -> classify -> metadata match -> probe -> playback prep`
- make each stage idempotent and retryable with explicit status per item
- record partial success so the library can be browsable before metadata is perfect
- keep API reads off the scan hot path

**Phase to address:** **Phase 2** — worker orchestration and job model

---

### 3) Weak naming assumptions and no explicit metadata confidence model
**What goes wrong:** The system assumes filenames are “good enough,” then silently matches the wrong series/movie, mishandles specials, multi-episode files, alternate orders, or mixed libraries.

**Why it happens:** Filename parsing seems easy until real household libraries contain anime, specials, date-based shows, multi-part rips, mixed folders, and inconsistent renames.

**Consequences:**
- wrong posters/episodes/season mappings
- bad series grouping and broken episode order
- user distrust in the whole library
- repeated destructive rescans to “fix metadata”

**Warning signs:**
- high rate of low-similarity automatic matches
- mixed movie/show folders need special-case code everywhere
- season/special handling becomes a pile of regex exceptions
- “fix match” becomes a common support path very early

**Prevention strategy:**
- separate **filename parsing** from **metadata matching** from **canonical media modeling**
- persist a match confidence score and match source
- support explicit identifiers (TMDB/TVDB/IMDb) and manual correction flows
- treat specials, date-based shows, multi-episode files, and alternate ordering as first-class edge cases, not later hacks
- keep libraries typed; do not normalize around mixed-content roots

**Phase to address:** **Phase 3** — metadata pipeline and media semantics

**Why this is high-confidence:** Jellyfin and Plex both explicitly require disciplined folder structure, typed libraries, season naming, and metadata IDs for reliable matching; Jellyfin also discourages mixed libraries due to unreliable metadata results.

---

### 4) Leaking storage implementation details past the storage boundary
**What goes wrong:** Business logic and clients become dependent on OpenList path conventions, direct-link formats, or raw file tree semantics.

**Why it happens:** It is tempting to expose “whatever OpenList already knows” directly to move faster.

**Consequences:**
- impossible to swap/add direct local/NAS/cloud adapters cleanly
- playback APIs become storage-specific instead of media-specific
- search, progress, and permissions couple to file tree details
- every future optimization becomes a breaking change

**Warning signs:**
- client routes or payloads contain raw storage paths as primary keys
- frontend can navigate source trees more naturally than media entities
- service code outside the adapter layer knows OpenList-specific response shapes
- adding a second provider requires touching business and API contracts

**Prevention strategy:**
- keep `StorageProvider` narrow and capability-based (`List`, `Get`, `Link`, `ResolveStorage`, `Capabilities`, later `StableIdentity`, `DeltaScan`, `BatchStat`)
- expose media-facing APIs around `library / media_item / version / stream`, not storage paths
- confine raw file-path knowledge to scan/admin tooling
- make adapter outputs normalized before they enter business logic
- treat OpenList as a provider, not as the product model

**Phase to address:** **Phase 1** — architecture boundary hardening

---

### 5) Assuming “playback = return a URL”
**What goes wrong:** Playback is implemented as direct-link handoff only, without capability negotiation for codec/container/subtitles/range/HDR/audio, and without a robust fallback path.

**Why it happens:** Direct links work in the happy path, especially on one browser and one test file.

**Consequences:**
- playback works on Web but fails on TV/mobile
- subtitles unexpectedly force heavy transcodes
- HDR/HEVC/container mismatches create black screens or buffering
- progress/reporting breaks because the server is bypassed too much

**Warning signs:**
- support issues are highly client-specific
- subtitles cause sudden CPU spikes
- same file direct-plays on one client and fails on another with no explanation
- no server-side record of why a playback decision was made

**Prevention strategy:**
- model playback as a **decision service**, not a link service
- evaluate per-client capabilities: container, video codec/profile, audio codec/channels, subtitle mode, HDR/SDR, seek/range support
- prefer `direct play -> remux/direct stream -> transcode fallback`
- log the decision reason for every playback session
- keep progress sync separate from whether the final stream was direct or transcoded

**Phase to address:** **Phase 4** — playback service

**Why this is high-confidence:** Plex and Jellyfin both document that playback outcomes differ by client capability, container, codecs, and subtitle behavior; subtitle burn-in is a particularly expensive transcode trigger.

---

### 6) Progress sync modeled as naive last-write-wins
**What goes wrong:** Every client posts raw position updates, and the server overwrites resume state without session semantics, monotonicity rules, or conflict handling.

**Why it happens:** Progress looks like “just save seconds watched” until multiple devices and scrub behavior appear.

**Consequences:**
- progress jumps backward after pausing on another device
- “continue watching” becomes noisy or wrong
- rewatches and partial watches are indistinguishable
- live/direct-play/proxy paths produce inconsistent reporting

**Warning signs:**
- the same title’s progress oscillates during dual-device usage
- seek events overwrite meaningful completion state
- resume state is keyed to transient playback URLs
- no distinction between heartbeat, seek, stop, and completion

**Prevention strategy:**
- model **playback sessions** separately from **canonical resume state**
- accept monotonic progress updates within a session, with explicit rules for seek/backtrack
- store `position`, `duration`, `updated_at`, `completed_at`, `last_client`, and session identifiers
- define server-side merge rules for concurrent clients
- key progress to media/version identity, never to temporary links

**Phase to address:** **Phase 4** — progress and multi-client sync

---

## Moderate Pitfalls

### 7) No backpressure or bounded concurrency in scanning/probing
**What goes wrong:** The worker floods storage, metadata providers, DB, and CPU with unbounded parallel work.

**Consequences:** NAS/cloud latency explodes, OpenList becomes the bottleneck, and the app feels slow even though “background jobs” were supposed to isolate the load.

**Warning signs:**
- scan speed gets worse as concurrency increases
- OpenList HTTP latency rises sharply during scans
- DB write contention during library refresh
- worker queues grow while throughput falls

**Prevention strategy:**
- per-stage concurrency limits
- per-library scheduling and cancellation
- queue metrics: backlog, age, retries, median stage duration
- separate rate limits for storage traversal, metadata lookup, and probe jobs

**Phase to address:** **Phase 2**, then tune again in **Phase 5**

---

### 8) Shipping incremental sync before proving the correctness model
**What goes wrong:** Projects jump to filesystem events/webhooks/cursors before they have trustworthy full-scan reconciliation, stable identity, or idempotent jobs.

**Consequences:** silent drift between storage and DB, phantom deletions, and impossible-to-debug “missing episode” reports.

**Warning signs:**
- event handling code includes lots of “if missing, do full resync” fallbacks
- the team cannot explain what happens after rename, partial upload, failed probe, or duplicate event delivery
- manual full scan is still the only trustworthy recovery path

**Prevention strategy:**
- first make full scan + reconciliation correct and repeatable
- define event semantics explicitly: create/update/delete/rename are advisory, not truth
- keep periodic reconciliation even after event-driven sync arrives
- treat partial uploads and eventually consistent backends as normal cases

**Phase to address:** **Phase 5** — incremental/event-driven sync

---

### 9) Premature “performance architecture” instead of measured evolution
**What goes wrong:** Teams rewrite for microservices, direct-storage adapters, or heavyweight caches before they have evidence about the actual hotspot.

**Consequences:** more deployment complexity, more failure modes, little real user benefit.

**Warning signs:**
- architecture discussions are about future scale rather than current bottlenecks
- no timing breakdown for scan/list/playback/probe paths
- proposal to bypass OpenList everywhere before hotspot profiling exists

**Prevention strategy:**
- add observability first: scan stage timings, provider latency, playback decision metrics, transcode counts, cache hit rate
- isolate Worker from API before introducing more services
- only add direct adapters for proven hotspots (for example local-path heavy scans)
- treat Redis/caching/queue extraction as response to evidence, not as V1 defaults

**Phase to address:** **Phase 5** — performance evolution

---

## Minor Pitfalls

### 10) Treating multi-client support as a UI concern rather than an API contract
**What goes wrong:** Web works first, then mobile/TV are forced to emulate Web assumptions.

**Consequences:** awkward API shapes, playback regressions on constrained clients, and a permanent bias toward browser behavior.

**Warning signs:**
- API responses expose web-player details instead of media/playback semantics
- TV/mobile clients need custom exceptions for common flows
- no client capability declaration in playback negotiation

**Prevention strategy:**
- define stable media and playback contracts before expanding clients
- require explicit client capability input for playback decisions
- keep transport/UI concerns out of media identity and progress models

**Phase to address:** **Phase 4**

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Phase 1: Storage boundary | OpenList leaks into API/business model | Normalize everything behind `StorageProvider`; media APIs must not use raw storage paths as identities |
| Phase 2: Scanner/jobs | Scan tries to do metadata/probe/transcode inline | Split fast-path discovery from slow-path enrichment jobs |
| Phase 2: Scanner correctness | Path-based identity breaks rename/move handling | Introduce stable file identity and reconciliation rules before aggressive delta sync |
| Phase 3: Metadata | Wrong matches become “normal” | Add confidence scores, manual fix flows, typed libraries, explicit IDs |
| Phase 4: Playback | Playback treated as URL generation | Build capability-aware playback decision engine with direct/remux/transcode fallback |
| Phase 4: Progress | Concurrent clients overwrite each other | Separate session telemetry from canonical resume state; add merge rules |
| Phase 5: Incremental sync | Event-driven updates drift from reality | Keep periodic reconciliation and idempotent event handlers |
| Phase 5: Performance | Premature rewrite/microservices | Measure first, then isolate Worker, then optimize hotspots |

---

## What This Means For Roadmap Planning

1. **Do not start with “better playback UX” before the storage and identity model are stable.** Playback quality is downstream of correct media/file semantics.
2. **Treat scanning as orchestration, not enrichment.** Fast discovery first; expensive understanding later.
3. **Make metadata confidence explicit.** Silent wrong matches are worse than incomplete metadata.
4. **Define playback and progress as first-class backend services.** They are not thin wrappers over storage URLs.
5. **Delay performance complexity until metrics justify it.** But add observability early so the trigger is obvious.

---

## Sources

### Project-specific
- `.planning/PROJECT.md` — active requirements and constraints around OpenList boundary, Worker separation, stable identity, playback, and multi-client API. **Confidence: HIGH**
- `docs/media-architecture/improved-architecture.md` — recommended architecture, data ownership, scan/playback flows, and known risks. **Confidence: HIGH**

### Official ecosystem references
- Jellyfin — Libraries: mixed library type is discouraged because of unreliable metadata results. https://jellyfin.org/docs/general/server/libraries/ **Confidence: HIGH**
- Jellyfin — Movies naming and metadata provider IDs improve matching reliability. https://jellyfin.org/docs/general/server/media/movies/ **Confidence: HIGH**
- Jellyfin — TV shows naming, season structure, specials, and multi-part caveats. https://jellyfin.org/docs/general/server/media/shows/ **Confidence: HIGH**
- Jellyfin — Client codec/container/subtitle compatibility varies substantially; subtitles can force expensive transcodes. https://jellyfin.org/docs/general/clients/codec-support/ **Confidence: HIGH**
- Jellyfin — Hardware acceleration is partial on some platforms; SSD/RAM cache and correct ffmpeg builds matter. https://jellyfin.org/docs/general/post-install/transcoding/hardware-acceleration/ **Confidence: HIGH**
- Plex — TV organization, year/IDs, episode ordering, specials, multi-episode and split-file caveats. Last modified 2025-10-08. https://support.plex.tv/articles/naming-and-organizing-your-tv-show-files/ **Confidence: HIGH**
- Plex — Direct Play vs Direct Stream vs Transcode overview, including subtitle-triggered full transcode. Last modified 2021-07-04 (older but still aligned with current Jellyfin guidance). https://support.plex.tv/articles/200430303-streaming-overview/ **Confidence: MEDIUM**

## Confidence Notes

- **Highest-confidence pitfalls:** storage boundary leakage, path-vs-identity problems, scan/enrichment coupling, metadata naming/mixed-library issues, client capability mismatch in playback.
- **Medium-confidence area:** exact best-practice details for cross-client progress conflict resolution are less explicitly documented in public official docs; recommendations here are based on common server design patterns plus the project’s own multi-client requirement.
