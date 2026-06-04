## Why

Media ingest currently treats several different title purposes as if they were
one field. Scan parsing, directory shape planning, recognition materialization,
metadata search, catalog display, sort keys, and resource labels each apply
slightly different cleanup rules, which makes weak identities such as a lone
year or a release-only filename easy to promote into the wrong movie.

This change makes title roles explicit end to end so directory-derived titles,
file-derived evidence, search queries, identity keys, display labels, and raw
resource labels cannot accidentally substitute for each other.

## What Changes

- Define a small set of title roles with stable responsibilities, inputs,
  fallback behavior, and persistence boundaries.
- Refactor scan parsing, content-shape planning, recognition materialization,
  metadata matching, and catalog/resource projection to consume role-specific
  title helpers instead of ad hoc cleanup functions.
- Prefer directory title evidence for single-movie, multipart-movie, and
  multi-version movie directory shapes when file names only contain metadata or
  otherwise produce weak movie identity keys.
- Keep multi-movie collection directories scoped per movie identity and avoid
  letting one directory or weak file title collapse separate movies into one
  work.
- Preserve raw file stems for resource/version labels; do not run movie-title
  cleanup over `file_title` unless a future role explicitly requires it.
- Remove obsolete compatibility wrappers and duplicate title cleanup logic after
  all call sites move to role-specific helpers.
- Add tests that cover title role boundaries, weak title fallback, directory
  shape interactions, metadata search inputs, resource file titles, and
  representative historical bug cases.

## Capabilities

### New Capabilities

- `media-title-boundaries`: Defines title roles and how scan, directory shape,
  recognition, metadata, catalog, and resource code may derive, persist, and
  fall back between them.

### Modified Capabilities

- None.

## Impact

- Affected backend packages:
  - `mibo-media-server/internal/titleclean`
  - `mibo-media-server/internal/scanparse`
  - `mibo-media-server/internal/library`
  - `mibo-media-server/internal/recognition`
  - `mibo-media-server/internal/metadata`
  - `mibo-media-server/internal/catalog`
  - `mibo-media-server/internal/database`
- Database impact: no schema change is expected, but existing persisted values
  for weak movie identities, local titles, raw work titles, search titles,
  sort titles, and resource `file_title` values may need a repair/backfill path
  for already-scanned libraries.
- API impact: no public API shape change is expected; response values become
  more consistent because each title field has a single source responsibility.
- Testing impact: backend unit and integration tests need to cover title roles
  across file parsing, directory materialization, metadata matching, and catalog
  projection.
