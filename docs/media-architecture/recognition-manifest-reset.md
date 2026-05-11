# Recognition Manifest Development Reset

The recognition-manifest resolver replaces scanner-side metadata identity decisions. During development, validate this architecture from a clean local catalog state instead of migrating old content-shape, path-tree, sibling-match, or weak fallback output.

## When To Reset

Reset local scan data before validating a schema or resolver change that affects any of these tables or flows:

- `recognition_manifests`, `recognition_candidates`, `recognition_evidence`, `recognition_decisions`, `recognition_conflicts`, `recognition_rules`
- `inventory_files`, `inventory_file_signals`, `resource_files`, `resources`, `resource_metadata_links`, `resource_library_links`
- metadata/resource projection tables derived from scanner materialization
- retired scanner paths such as content-shape final assignments, path-tree materialization overrides, same-metadata sibling matching, or weak title/year fallback creation

## Local Reset Procedure

1. Stop the backend and worker processes.
2. Remove local development database state under `mibo-media-server/data/`, or point the backend at a fresh SQLite DSN.
3. Start the backend with `MIBO_LOCAL_ROOT_PATH=/Users/atlan/Desktop/IdeaProjects/Mibo/demo-media` when using demo media.
4. Re-add or reuse the demo media source from setup/settings.
5. Run a full scan and let resolver/materialization work finish.
6. Rebuild projections if needed through the admin maintenance endpoints.

## Validation Checklist

After reset and rescan, verify the resolver-created graph rather than legacy scanner rows:

- same-folder movie versions materialize as one movie metadata identity with multiple resources
- sibling-folder movie versions materialize order-independently
- independent movie collections remain split
- standard TV and flat episode folders create series, season, episode, and resource links
- multi-episode files link to all episode slots
- trailers, samples, extras, and featurettes do not become primary versions
- sidecar external IDs seed metadata enrichment through resolver evidence
- hash duplicates do not override conflicting stronger identity evidence
- governance review surfaces show resolver candidates, conflicts, alternatives, and correction actions

Do not preserve disabled legacy fallback code to support old local scan state. If validation depends on old rows, reset and rebuild from the manifest resolver instead.
