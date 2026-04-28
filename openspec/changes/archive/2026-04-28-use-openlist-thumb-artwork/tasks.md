## 1. Storage Contract

- [x] 1.1 Add `ThumbnailURL` to `mibo-media-server/internal/storage.Object` with JSON name `thumbnail_url`.
- [x] 1.2 Update the OpenList adapter list response parsing to read `thumb` into `ThumbnailURL`.
- [x] 1.3 Update the OpenList adapter get response parsing to read `thumb` into `ThumbnailURL`.
- [x] 1.4 Add OpenList adapter tests proving `thumb` is preserved from `/api/fs/list` and `/api/fs/get` responses.

## 2. Artwork Priority Integration

- [x] 2.1 Add a provider-thumbnail primary-artwork fallback in `internal/probe/artwork.go` after sibling artwork lookup and before ffmpeg extraction.
- [x] 2.2 Ensure provider thumbnails are never used for backdrop by default.
- [x] 2.3 Ensure non-generated selected poster artwork, including TMDB/manual remote URLs, is not overwritten by OpenList thumbnails.
- [x] 2.4 Ensure generated movie poster artwork can be replaced by an OpenList thumbnail when available, while episodes store the thumbnail as `still`.
- [x] 2.5 Keep ffmpeg extraction as the final fallback for missing poster/backdrop artwork.

## 3. Tests

- [x] 3.1 Add probe/artwork tests where an OpenList-like provider returns `ThumbnailURL` and no sibling artwork, proving movie poster uses the thumbnail and episode still uses the thumbnail.
- [x] 3.2 Add a test where sibling `cover` artwork exists and provider `ThumbnailURL` also exists, proving sibling artwork wins.
- [x] 3.3 Add a test where TMDB/non-generated poster exists and provider `ThumbnailURL` also exists, proving the existing remote poster remains selected.
- [x] 3.4 Add a test where provider `ThumbnailURL` exists but no backdrop sibling exists, proving backdrop still falls back to ffmpeg generation.
- [x] 3.5 Add a test where provider `ThumbnailURL` is blank, proving current ffmpeg fallback behavior remains unchanged.

## 4. Verification

- [x] 4.1 Run focused backend tests for OpenList storage adapter and probe artwork behavior.
- [x] 4.2 Run `go test ./...` from `mibo-media-server/`.
- [x] 4.3 Manually inspect API-selected images for OpenList-backed movie and episode items to confirm movie `poster` and episode `still` can point at provider thumbnail while `backdrop` remains explicit/generated.
