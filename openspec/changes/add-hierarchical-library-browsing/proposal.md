## Why

Mibo currently treats a media library as a flat metadata collection once scan results are available, which makes it hard for users to browse large libraries that are intentionally organized by filesystem folders such as region, language, or collection. We need a library-aware hierarchical browsing mode so users can move from a library entry into meaningful subfolders before reaching the final metadata items they want to play.

## What Changes

- Add a hierarchical browse mode that exposes library entries as the first level in metadata browsing.
- Allow each library browse request to return immediate child folders derived from the scanned source path structure before showing media metadata items.
- Support drilling from a library root into nested folders such as `中国电影` or `欧美电影` and then into the metadata items recognized under that folder.
- Preserve existing media detail and playback flows once a user reaches a metadata item through hierarchical browsing.
- Return mixed browse results that can distinguish folder nodes from playable metadata nodes, including enough context for breadcrumb navigation and pagination.
- Keep library visibility and playback authorization behavior unchanged so users only see folders and items from libraries they are already allowed to access.

## Capabilities

### New Capabilities
- `hierarchical-library-browsing`: Browse media libraries through filesystem-derived folder levels, starting from library roots and drilling down to folders and metadata items.
- `library-folder-navigation`: Resolve folder node context, breadcrumb state, and mixed child results so clients can move deeper into a library and back out reliably.

### Modified Capabilities
- None.

## Impact

- Affected backend code in `mibo-media-server/internal` browse, library, scan, and HTTP API layers to materialize folder-aware browse responses from scanned media inventory.
- Affected frontend browse routes, metadata listing UI, and API/query wiring in `frontend/src` to render library roots, folder nodes, breadcrumbs, and final metadata item grids/lists.
- New authenticated API surface or extensions for hierarchical browse queries, folder node addressing, and mixed result payloads.
- New tests covering library root listing, nested folder traversal, mixed folder and item results, empty folder handling, authorization filtering, and playback/detail transitions from folder-derived browse results.
