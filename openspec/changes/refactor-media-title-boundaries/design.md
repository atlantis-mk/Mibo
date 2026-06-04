## Context

The ingest pipeline now has several places that transform title-like text:
filename parsing, folder parsing, content-shape planning, recognition key
generation, materialization, metadata matching, resource projection, and catalog
querying. The same source text can currently flow through generic cleanup in
different ways, so a release-only filename such as `2026.2160p.WEB-DL.mkv` can
become both a weak movie identity and a user-visible title.

The target architecture is not more title fields. It is fewer, explicit title
roles with one owner each. Some roles are durable fields (`title`,
`search_title`, `sort_title`, `raw_work_title`, `file_title`); some are
transient evidence (`TitleCandidate`, folder title candidates, work keys).

## Goals / Non-Goals

**Goals:**

- Make title transformation role-specific and auditable from scan through
  catalog projection.
- Prevent weak file-derived titles from becoming movie identities when a
  directory shape gives better title evidence.
- Keep resource file labels faithful to the file stem instead of work-title
  cleanup.
- Remove duplicate cleanup wrappers and make new title use cases choose a role
  deliberately.
- Provide focused tests for role boundaries and historical failure cases.

**Non-Goals:**

- Replace provider metadata titles, user edits, or governance field policy.
- Add a new public API shape for catalog responses.
- Solve provider-specific transliteration or multilingual alias ranking.
- Rebuild the whole directory ingest pipeline beyond the title responsibilities
  needed by this change.

## Decisions

### 1. Centralize title roles in `titleclean`

Create or complete role-specific helpers in `internal/titleclean`:

- `MovieIdentityTitle`: strict canonical text for movie work identity and
  comparison. It removes release noise, normalizes separators and case, and is
  never used for display labels.
- `MovieSearchTitle`: human-readable local query text for metadata providers
  and search documents. It strips release noise but preserves readable casing.
- `MovieDisplayTitle`: local fallback display title from scanner evidence.
  Provider or user-managed titles remain authoritative once available.
- `RawWorkTitleText`: readable provenance text for local resource/work evidence.
  It can preserve years and descriptive words, but it is not a work key.
- `ResourceFileTitleFromPath`: file stem only, used for `file_title`.
- `SortTitle`: a narrowly scoped helper for local fallback sort text if current
  sort-title behavior is not already centralized in metadata field policy.

Rationale: every call site can now answer "which role am I producing?" instead
of picking an arbitrary cleanup helper.

Alternative considered: keep generic `Normalize` and document expected use at
each call site. That keeps the current ambiguity and makes future regressions
likely.

### 2. Move directory-vs-file choice into content-shape policy

Title source priority belongs with the directory shape decision, not with
low-level filename cleanup. Content-shape planning/materialization should expose
one policy table for movie-like shapes:

- single movie folder: directory title preferred when the file identity is weak;
- multipart movie folder: directory title preferred for the shared work;
- multi-version movie folder: directory title preferred for the shared work;
- movie collection folder: each movie keeps its own file, child directory, or
  catalog/external-id identity; the collection root name is not a shared movie
  title;
- series and season folders: use series-title rules, not movie-title fallback;
- extras, attachments, mixed, or review-required folders: do not create a movie
  title identity from weak evidence.

Rationale: the same filename can be valid inside one shape and dangerous inside
another. The folder shape is the strongest available context for choosing a
title source.

Alternative considered: make filename parsing smarter until it extracts every
title. This cannot reliably distinguish a real numeric title from a release-only
filename without directory context.

### 3. Treat weak identities as a first-class classification

Add a small weak-title predicate used before materialization and metadata
matching. A movie identity candidate is weak when it is empty, only a year,
only release/technical tokens, or only punctuation/separators after cleanup.
Weak candidates may be preserved as raw provenance but MUST NOT be used as the
sole movie work key for movie-folder, multipart, or multi-version scopes when
the directory has usable title evidence.

Rationale: this is the direct guardrail for the observed "different folders
became one movie" failure.

Alternative considered: special-case `2026`. That fixes one symptom but not the
class of release-only titles.

### 4. Keep persistence boundaries explicit

Materialization should write durable fields from the appropriate role:

- `metadata_items.title`: provider or user title, with local scanner display
  title only as a provisional fallback.
- `metadata_items.search_title`: local search/query role, not identity key text.
- `metadata_items.sort_title`: derived from the selected display/provider title
  by one sort-title policy.
- `resources.raw_work_title`: provenance role, usually directory or work-title
  evidence.
- `resources.file_title`: raw file stem role.
- recognition candidate evidence: source-tagged title evidence, including
  whether directory, file, sidecar, catalog ID, or fallback supplied it.

Rationale: durable fields then explain what they mean without reverse
engineering the cleanup path that produced them.

Alternative considered: persist only one title plus evidence JSON. That would
reduce fields, but catalog, metadata matching, and resource presentation need
different stable values.

### 5. Delete compatibility cleanup after migration

After all call sites use the role helpers, old wrappers such as generic
movie-folder title candidates and duplicate formatter functions should either
be removed or made private implementation details behind the role API.

Rationale: leaving old names around invites new code to bypass the role model.

Alternative considered: keep wrappers for compatibility indefinitely. This
would keep compile-time success but preserve the same maintenance debt.

## Risks / Trade-offs

- [Risk] Existing tests may encode old fallback behavior. -> Mitigation: update
  tests around named role expectations, and add regression tests for historical
  weak-title folders.
- [Risk] Some existing data already has weak `sort_key`, `title`,
  `search_title`, or `file_title` values. -> Mitigation: add a targeted
  repair/backfill path or rescan behavior that can recompute affected local
  values without overwriting provider or user-governed fields.
- [Risk] Changing identity keys can duplicate existing metadata rows during
  rescan. -> Mitigation: prefer repair/linking for weak local rows and keep
  external IDs/provider titles as stronger merge evidence.
- [Risk] Folder names can also be noisy. -> Mitigation: directory fallback only
  applies when the directory title has usable identity text and the shape policy
  allows that source.

## Migration Plan

1. Introduce the title role API and tests without changing persistence behavior.
2. Move scan, shape, recognition, metadata, and resource call sites to the role
   API in small slices.
3. Add weak-title and directory-shape regression tests before changing work-key
   fallback.
4. Remove obsolete wrappers and duplicate local formatting only after the new
   helpers cover every production use.
5. Add repair/backfill support for local scanner-derived fields that are known
   to be weak or were generated by old title normalization.

Rollback strategy: the changes are internal. If materialization behavior
regresses, revert the call-site migration while leaving the role helpers and
tests as diagnostic scaffolding.

## Open Questions

- Should the repair/backfill run automatically after scanner version changes,
  or remain an explicit maintenance operation?
- Should `sort_title` get a dedicated titleclean helper immediately, or stay in
  metadata field policy if that already provides a single owner?
