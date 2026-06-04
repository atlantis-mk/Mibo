# Metadata Scope Roots

Mibo now separates two related directory concepts in the backend scan pipeline:

- A leaf directory shape describes the direct video files inside one innermost media folder.
- A metadata scope root describes the semantic directory that should produce one movie, series, collection, or review outcome.

For example, each `Spider-Noir (2026)/1080p彩版` style child folder can be a direct-file `episode_pack`, while the parent `Spider-Noir (2026)` is the metadata scope root with a `series/versioned_episode_packs` layout.

The scan pipeline persists leaf summaries before recognition units are built. Scope decisions then consume sibling leaf summaries, choose a bounded parent scope when purity and layout evidence allow it, and claim the covered files. Accepted scope claims suppress duplicate leaf-level recognition units. Review-required scope decisions remain visible in governance evidence and do not run automatic provider matching.

`content_shape` plans remain useful leaf evidence and fallback behavior. When an accepted metadata scope decision exists, downstream directory metadata resolution should prefer the scope decision's `root_kind`, `layout`, child roles, attachment roles, and covered files over treating a single leaf shape as the final metadata root.
