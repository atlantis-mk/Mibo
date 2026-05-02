## Why

Mibo currently asks users to choose movie, show, or mixed library semantics before scanning, which exposes internal media-classification concepts and makes users solve ambiguity up front. Because the product is still in active development and persisted data compatibility is not required for this change, we can simplify the model around sources first: users add a source path, Mibo quickly detects content classes, and semantic classification happens automatically in the background.

## What Changes

- **BREAKING** Replace user-visible movie/show/mixed library creation with a source-first add flow where the user provides a storage source and path, not a video semantic library type.
- **BREAKING** Treat movie, series, season, episode, extra, and unknown video distinctions as scanner/classifier output rather than required library input.
- Add quick source probing that samples directory entries under a fixed budget to detect content classes such as video, audio, text, image, and other without running heavy metadata work.
- Add automatic source collections/views for detected content classes, with video classification using evidence and confidence to project catalog items.
- Keep deep enrichment asynchronous: inventory, classification, metadata matching, artwork, technical probing, and governance review progress independently after the source is accepted.
- Add low-confidence review surfaces so users correct exceptions after scanning instead of choosing a library type before scanning.
- Remove migration/backward-compatibility requirements for existing library type data for this development-stage rebuild.

## Capabilities

### New Capabilities
- `source-first-auto-classification`: Source-first onboarding, quick content-class probing, automatic collection creation, and post-scan review behavior.

### Modified Capabilities
- `mixed-content-library`: Replace user-selected mixed/movie/show library semantics with automatic video classification under source-first scanning.
- `library-source-policies`: Change library/source creation requirements so basic creation is source-first and does not require user-visible movie/show/mixed type selection.
- `media-graph-scanner`: Require scanner decisions to work without dedicated movie/show library type hints and to classify video semantics from source evidence.

## Impact

- Backend library/source model, create/list/update APIs, scan job payloads, and scanner classification entry points.
- Frontend setup and settings flows for adding content sources and displaying scan/probe results.
- Catalog projection and governance review behavior for ambiguous classification outcomes.
- Tests around library creation, mixed content classification, scanner decisions, and settings UI expectations.
- No data migration or compatibility shim is required; existing development data may be reset or re-created.
