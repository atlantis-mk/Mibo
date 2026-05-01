## 1. Scan Artifact Data

- [x] 1.1 Add scan artifact fields for image candidates with image type, URL/path, source priority, and provisional scanner provenance.
- [x] 1.2 Add scan artifact fields for sidecar-derived external identities supported by the existing metadata pipeline.
- [x] 1.3 Populate artifact image candidates from same-folder sibling artwork discovered in the existing directory snapshot.
- [x] 1.4 Populate artifact image candidates from provider-neutral storage object thumbnail URLs when available.

## 2. Catalog Scan Write Path

- [x] 2.1 Persist scan-phase image candidates as catalog item images during `writeCatalogScan` without requiring metadata or probe jobs to run first.
- [x] 2.2 Select scan-phase artwork only when the item lacks an authoritative selected image for that image type.
- [x] 2.3 Prefer sibling poster/backdrop candidates over provider thumbnail candidates for the same item and image type.
- [x] 2.4 Persist sidecar external identities with scanner provenance so later metadata enrichment can use them.

## 3. Enrichment Compatibility

- [x] 3.1 Ensure metadata matching can use scan-seeded external identities to fetch detail directly when possible.
- [x] 3.2 Ensure metadata-selected images can replace provisional scan-phase provider thumbnails according to existing selection rules.
- [x] 3.3 Ensure probe fallback artwork remains asynchronous and does not overwrite manual or remote authoritative selections.

## 4. Tests

- [x] 4.1 Add scan tests proving sibling poster and backdrop files are selected during the first scan before worker jobs complete.
- [x] 4.2 Add scan tests proving provider thumbnails fill missing poster or episode still slots as provisional artwork.
- [x] 4.3 Add tests proving existing manual or remote selected artwork is preserved during rescan.
- [x] 4.4 Add sidecar tests proving external IDs are persisted during scan and used by later metadata enrichment.
- [x] 4.5 Run focused backend tests for library scan, metadata, probe artwork, and worker behavior.
