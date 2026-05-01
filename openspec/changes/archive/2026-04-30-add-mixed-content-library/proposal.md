## Why

Some users keep movies and multi-episode content in the same source tree, but current library creation forces a movie-or-show choice that biases scanning too early. A mixed content library gives these sources a simple first-pass classification path while avoiding common extras such as trailers and interviews from polluting movie or series results.

## What Changes

- Add a mixed content option when creating media libraries.
- Treat mixed libraries as scan roots that classify each direct media group after excluding known extras: `trailer`, `behind-the-scenes`, `sample`, `featurette`, `interview`, and `deleted scene`.
- For each group in a mixed library, classify groups with more than one non-extra media file as series-like TV content and groups with exactly one non-extra media file as movie content.
- Preserve existing movie and show library behavior outside the new mixed content type.

## Capabilities

### New Capabilities
- `mixed-content-library`: Defines user-facing library creation and scan classification behavior for mixed movie/TV content sources.

### Modified Capabilities
- `catalog-api-playback`: Catalog list and detail APIs must continue to expose mixed-library scan results through existing movie and series item semantics.

## Impact

- Backend library scanning and filename-extra detection in `mibo-media-server/internal/library`.
- Library creation and persistence paths that accept `Library.Type` values.
- Settings UI library form in `web/src/features/settings/components/library-form.tsx`.
- Tests around scan classification for movie, show, and mixed library roots.
