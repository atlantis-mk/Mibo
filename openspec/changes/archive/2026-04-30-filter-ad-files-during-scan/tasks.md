## 1. Exclusion Model And Rules

- [x] 1.1 Add a persisted scan exclusion model scoped by library, provider, stable file identity when available, normalized path fallback, reason, enabled state, and audit metadata.
- [x] 1.2 Support initial exclusion reasons including `advertisement`, `unwanted`, `duplicate`, `wrong_import`, and `other`, while exposing only the minimal reason set needed by the first UI/API.
- [x] 1.3 Add a centralized scanner helper that checks persisted scan exclusions and explicit advertisement video paths using token-bound filename and directory-segment checks.
- [x] 1.4 Cover the helper with unit tests for persisted exclusions and positive ad markers including `ad`, `ads`, `advertisement`, `commercial`, and `广告`.
- [x] 1.5 Cover disabled exclusions and false-positive cases such as `Ad Astra`, `Adventure Movie`, regular episodes, trailers, samples, and featurettes.

## 2. User Exclusion Operation

- [x] 2.1 Add a backend operation or endpoint to mark a scanned inventory file, asset, or catalog-linked media entry as excluded from future scans.
- [x] 2.2 Persist a scan exclusion when the mark operation is called, preferring stable identity and retaining normalized path fallback.
- [x] 2.3 Remove or hide the associated scanner-managed asset from normal catalog browsing without deleting the source storage file.
- [x] 2.4 Ensure exclusion records can be disabled or restored later without losing audit history.
- [x] 2.5 Add tests that marking an item as advertisement creates the exclusion and removes it from normal catalog results.

## 3. Scan Pipeline Integration

- [x] 3.1 Call the exclusion filter in `walkDirectory` immediately after confirming an object is a supported video file.
- [x] 3.2 Ensure skipped excluded files do not enter classification, catalog writing, metadata match queueing, inventory file creation, or probe queueing.
- [x] 3.3 Keep directory recursion and sibling media processing unchanged when excluded files are skipped.
- [x] 3.4 Add scan-level visibility for skipped files through `SyncResult` and scanner logging or equivalent operational output, distinguishing user exclusions from automatic filename filtering.

## 4. Verification

- [x] 4.1 Add scan integration tests proving mixed folders skip excluded files while scanning valid videos and sidecars.
- [x] 4.2 Add assertions that skipped excluded files create no catalog items, assets, inventory files, match jobs, or probe jobs.
- [x] 4.3 Add rescan tests proving user-marked scan exclusions prevent re-import.
- [x] 4.4 Run focused backend tests for scanner classification, exclusion marking operations, and catalog scan behavior.
