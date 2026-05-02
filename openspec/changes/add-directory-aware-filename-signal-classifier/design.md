## Context

Mibo's fast scanner already separates cheap classification from slow enrichment, records decision evidence, and avoids ffprobe/provider work in the fast path. The remaining problem is that filename rules are still spread across title normalization, media classification, role detection, and sibling grouping. A token such as `DDP5.1` can be title noise, release evidence, and an anti-signal for weak numeric episode inference at the same time, but the current flow does not model that explicitly before cleanup and classification.

This change introduces a performance-first version of the broader budgeted classifier idea: a cached directory-aware filename signal classifier. It focuses on the first two cheap budget levels: per-file filename/path signal extraction and per-directory summaries built from the scan traversal snapshot. It leaves deeper source profiles, parse forests, and provider/probe validation as future refinement layers.

## Goals / Non-Goals

**Goals:**

- Extract structured filename/path signals before title cleanup and semantic classification.
- Preserve filename-derived metadata hints such as quality, source, codec, audio, subtitle, release group, role, episode, year, and path evidence.
- Build clean title views from preserved signals so cleanup does not destroy data needed for classification, debugging, or later validation.
- Avoid false episode/title inference from release tokens, especially audio channels, quality labels, codecs, and source labels.
- Generate movie, episode, trailer, sample, extra, and version candidates with lightweight evidence summaries.
- Improve accuracy with per-directory summaries computed once from scan snapshots, not by recursively probing a full source.
- Keep fast classification cheap: no media reads, ffprobe, hashes, metadata provider calls, or artwork work.
- Keep ambiguous or conflicting decisions provisional, reviewable, or eligible for later refinement.

**Non-Goals:**

- Do not introduce a full parse forest, constraint solver, or whole-source semantic inference engine in this change.
- Do not populate authoritative technical fields from filename tokens; ffprobe remains the source of truth for actual streams and dimensions.
- Do not add a frontend configuration UI for custom filename parsing rules.
- Do not require a data migration that rewrites existing catalog items solely to add filename signal snapshots.
- Do not make slow provider or probe validation part of synchronous scanning.

## Decisions

### Decision: Add a filename signal layer before cleanup and classification

The scanner will first derive a structured filename signal model from the raw path, basename, extension, and path segments. This model will separate identity signals, release hints, role hints, title tokens, cleanup evidence, and path hints. Existing regex and token rules should become signal probes that answer "what did we see?" rather than directly selecting the final semantic type.

Alternatives considered:
- Continue expanding `titleclean` and scanner regex branches independently. This is smaller initially but keeps cleanup and classification coupled and increases drift.
- Build a full grammar parser immediately. This can represent ambiguity better but is unnecessarily heavy for the performance-first target.

Rationale: filename signals are the smallest model that preserves useful metadata before cleanup while keeping implementation and runtime cost bounded.

### Decision: Treat filename-derived metadata as hints, not authoritative metadata

Quality, source, codec, audio, subtitle, HDR, edition, and release-group values from filenames will be retained as release hints and evidence. They can drive title cleanup, candidate grouping, version detection, and anti-misclassification, but they must not write authoritative technical metadata fields.

Alternatives considered:
- Use filename values as fallback technical metadata when probing is unavailable. This creates correctness conflicts for mislabeled releases and weakens the ffprobe source-of-truth boundary.
- Drop release tokens after title cleanup. This loses useful version, grouping, and debugging evidence.

Rationale: the same filename token can be useful without being trustworthy enough to become a technical fact.

### Decision: Use cached per-directory summaries for cheap context

During scan traversal, the classifier will build or reuse one summary per directory from already-listed objects. The summary can include video counts, likely main counts, attachment counts, explicit episode counts, numeric sequence evidence, title-year movie evidence, version-group evidence, common title stems, and season-directory hints. Classification for files in that directory consumes the summary instead of repeatedly listing or recursively analyzing storage.

Alternatives considered:
- Classify each file independently. This is fastest per file but misclassifies flat numeric episode folders, movie-version folders, and independent movie directories.
- Run whole-source inference synchronously. This improves global accuracy but conflicts with source-first performance and remote storage constraints.

Rationale: current-directory summary provides the highest accuracy gain per unit of cost and aligns with existing bounded sibling grouping requirements.

### Decision: Keep evidence lightweight in the fast path

The first implementation should expose compact evidence summaries rather than a full persisted evidence graph. Evidence should name the signal kind, source, value, and how it affected the candidate, such as supporting episode detection, supporting release/version grouping, being removed from title, or suppressing weak numeric episode inference.

Alternatives considered:
- Persist every token and evidence edge as graph nodes. This enables richer review tooling but adds data model and storage complexity too early.
- Only store final reason text. This is compact but weak for debugging and governance.

Rationale: lightweight evidence satisfies explainability and testing needs while preserving a path toward a richer graph later.

### Decision: Resolve conservatively with early exit and escalation

The resolver will use strong file signals and directory summaries to confirm high-confidence outcomes quickly. Medium-confidence outcomes remain provisional. Low-confidence or conflicting outcomes produce review-required decisions or are queued for later refinement if future jobs exist. The fast path should not spend more work trying to force a final answer after cheap evidence is exhausted.

Alternatives considered:
- Always keep escalating until a final answer is produced. This risks slow scans and confident wrong classifications.
- Always require review for ambiguous files. This avoids mistakes but reduces automatic classification value.

Rationale: performance is the leading constraint, and accuracy is preserved by avoiding forced guesses when cheap evidence is insufficient.

## Risks / Trade-offs

- [Risk] The filename signal model can grow into an unbounded catch-all. -> Mitigate by keeping fields grouped by identity, release, role, title, and path, and adding tests whenever a new signal kind is introduced.
- [Risk] Moving rules into signal extraction can change classification behavior unexpectedly. -> Mitigate with behavior-preserving tests first, then targeted tests for known false positives and directory layouts.
- [Risk] Directory summaries may be stale or incomplete when scans are partial. -> Mitigate by marking summaries as snapshot-derived and allowing provisional/review outcomes when required sibling evidence is incomplete.
- [Risk] Lightweight evidence may be insufficient for future UI explanations. -> Mitigate by using stable evidence kinds and values that can later be expanded into graph nodes without changing classifier semantics.
- [Risk] Release hints could be mistaken for authoritative metadata by future callers. -> Mitigate with explicit naming such as `filename_*_hints` or `release_hints` and tests proving ffprobe-derived technical metadata wins.

## Migration Plan

No production data migration is required. The new signal extraction and directory summary logic can replace internal fast-classification inputs while preserving existing catalog and inventory schemas where possible. Existing scanner decisions may receive richer evidence on future scans, but existing catalog items remain valid.

Rollback is code-level: revert the new signal pipeline and return to the existing classification path. Because raw inventory paths, original titles, and catalog facts are preserved, no irreversible data transformation is introduced.

## Open Questions

- Should filename signal snapshots be stored directly in existing scanner decision evidence, or only as derived evidence summaries for the first implementation?
- What minimum evidence fields are needed by current governance APIs to display filename-derived release hints without frontend redesign?
- Should directory summaries be built as explicit structs in the scanner pipeline or inferred lazily from existing directory snapshot maps?
