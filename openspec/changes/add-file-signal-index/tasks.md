## 1. Data Model And Repository

- [x] 1.1 Add `InventoryFileSignal` database model, AutoMigrate registration, and index repair expectations for classifier-versioned file signal rows.
- [x] 1.2 Implement file fingerprint helpers derived from storage path, basename, size, modified time, stable identity, and classifier version.
- [x] 1.3 Implement repository helpers to load reusable signals, batch upsert recomputed signals, and ignore stale classifier versions.
- [x] 1.4 Add model and repository tests for unchanged reuse, changed fingerprint invalidation, and classifier version invalidation.

## 2. Signal Extraction Integration

- [x] 2.1 Map the existing `filenameSignalModel` into persisted signal columns and evidence JSON without adding a second parser.
- [x] 2.2 Generate or reuse file signals after inventory file upsert and before content-shape profile compilation in scan/materialization flows.
- [x] 2.3 Add batch helpers so large directories can hydrate signal lookups without per-file queries.
- [x] 2.4 Preserve runtime extraction as fallback for missing signal rows during migration or cache misses.

## 3. Content Shape Reuse

- [x] 3.1 Update content-shape directory profile construction to prefer indexed file signals for visible videos.
- [x] 3.2 Keep profile aggregation behavior equivalent for episode coverage, leading numeric coverage, year density, title uniqueness, common title stem, attachment counts, and version evidence.
- [x] 3.3 Ensure content-shape plans and assignments reuse indexed signal lookups across materialization batches.
- [x] 3.4 Add tests proving a large directory reuses indexed signals and does not repeatedly parse all filenames across batches.

## 4. Low-Confidence And Cleanup Behavior

- [x] 4.1 Tighten low-confidence content-shape plan handling so uncertain plans preserve review evidence and do not silently create unrelated confirmed movie or episode semantics.
- [x] 4.2 Keep compatibility fallback only where visible local placeholders are explicitly required, with review state and evidence attached.
- [x] 4.3 Remove or bypass redundant repeated parsing paths once indexed signal coverage is available.
- [x] 4.4 Add regression tests for ambiguous movie-vs-episode directories, movie collections, movie versions, and unknown review outcomes.

## 5. Verification

- [x] 5.1 Run focused backend tests for inventory, content-shape profile/plan/assignment, scan classification, and catalog materialization.
- [x] 5.2 Run `go test ./...` from `mibo-media-server/`.
- [x] 5.3 Update OpenSpec task status after implementation and note any intentional deferred cleanup.

Note: Section 4 keeps guarded local placeholders for current catalog visibility, but fallback decisions are now recorded as review-required instead of accepted semantics.
