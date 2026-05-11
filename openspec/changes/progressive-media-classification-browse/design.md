## Context

The current scan pipeline already separates `inventory_file`, `resource`, `metadata_item`, and projection read models, but movie-versus-series decisions still become final too early in the materialize path. Movie fallback can still rely on weak `title + year` evidence, while browse surfaces can only safely upgrade organizing entries after metadata collapse is complete. The change needs to preserve immediate visibility from inventory scans while making classification more conservative, more group-aware, and more heavily driven by strong local evidence such as sidecars, external IDs, and path-tree assignments.

## Goals / Non-Goals

**Goals:**
- Introduce a group-level classification phase that decides whether a set of files is a movie work, movie version group, series hierarchy, or unresolved group before final metadata collapse.
- Raise the bar for direct movie materialization so strong episode or series evidence wins over weak movie fallback.
- Keep low-confidence or conflicting groups visible through inventory/resource-backed browse entries until a stronger decision is available.
- Let sidecar metadata and supported external identities participate in the primary movie-versus-series decision instead of only decorating already-chosen metadata.

**Non-Goals:**
- Redesign the full metadata governance or merge/split toolchain.
- Replace the existing workflow DAG, projection model, or playback contracts.
- Solve every ambiguous anime, concert, or anthology edge case in one pass.

## Decisions

### Decision: Add a work-group classifier before final metadata materialization
The materialize path will produce a normalized work-group decision that identifies one of: movie, movie version group, series/season/episode, or unresolved. This classifier will consume path-tree assignments, content-shape assignments, filename episode signals, sidecar hints, and supported external IDs before any direct movie metadata fallback is attempted.

Alternatives considered:
- Keep per-file classification and only tighten `title + year` fallback. Rejected because it still makes single-file decisions without stable directory or group context.
- Delay all classification until remote metadata matching. Rejected because the system must preserve source-first behavior without requiring remote services for first visibility.

### Decision: Introduce explicit collapse confidence gates
The system will classify group decisions into high-confidence, guarded, and review-required states. Only high-confidence groups can immediately create or reuse final movie or series metadata. Guarded or conflicting groups remain inventory/resource-visible and carry review evidence forward.

Alternatives considered:
- Continue using only existing review-required decisions after metadata creation. Rejected because that still creates the wrong metadata identity too early.
- Block browse visibility until classification completes. Rejected because it regresses source-first feedback and library responsiveness.

### Decision: Treat sidecar and external identity hints as primary type evidence
Parsed sidecar metadata and supported external IDs will influence movie-versus-series decisions before final metadata collapse. Sidecar `media_type`, series title, season/episode values, and provider identity types can promote a group toward series or movie classification without waiting for later enrichment.

Alternatives considered:
- Keep sidecars as post-classification metadata decoration only. Rejected because it wastes deterministic local evidence and keeps false movie collapses in place.

### Decision: Browse upgrades replace unresolved entries instead of duplicating them
Library browse remains capable of showing organizing entries backed by inventory/resource rows, but once a final metadata-backed card is ready the browse response must replace the unresolved entry rather than showing both. This keeps the gradual `inventory -> resource -> metadata` transition without duplicate cards.

Alternatives considered:
- Hide unresolved entries as soon as any metadata placeholder exists. Rejected because weak placeholders are exactly what this change is trying to avoid treating as final.

## Risks / Trade-offs

- [More items remain unresolved longer] -> Mitigation: show explicit organizing states and promote high-confidence groups immediately.
- [Classification logic becomes more complex across library, catalog, and browse layers] -> Mitigation: concentrate the decision into a reusable work-group classifier with focused regression coverage.
- [Existing scans may produce different grouping outcomes than before] -> Mitigation: keep migration additive, preserve inventory/resource facts, and rely on projection refresh to reconcile final browse output.
- [Strong local sidecars could disagree with older catalog state] -> Mitigation: continue routing final field ownership through governance and preserve review-required outcomes when evidence conflicts.

## Migration Plan

1. Add the work-group classification output and confidence gates behind the existing materialize path.
2. Route movie metadata collapse through the new gate while preserving current episode hierarchy behavior for strong series signals.
3. Keep unresolved groups browse-visible through existing organizing entry paths.
4. Refresh projections after newly accepted group decisions so metadata-backed cards replace unresolved entries on the next browse refresh.
5. Roll back by disabling the stricter gate and falling back to current materialize behavior if browse starvation or unacceptable unresolved rates appear.

## Open Questions

- Should guarded groups materialize lightweight local placeholder metadata for governance only, or should they stay purely inventory/resource-backed until confidence improves?
- Which sidecar provider types beyond TMDB and current local formats should count as strong primary type evidence in the first implementation?
- Should user governance merge/split actions immediately create reusable path-tree classification rules in this change, or in a follow-up?
