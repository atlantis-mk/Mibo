# Provider Plugin Protocol

`add-provider-plugin-protocol` introduces a small HTTP protocol that lets Mibo register remote or local-companion provider instances for metadata and storage work without baking each integration into the server.

## Manifest

Every plugin must expose `GET /manifest`. Mibo validates and stores the returned document before enabling the instance.

Required fields:

- `id`: stable plugin identity, for example `io.mibo.plugin.tmdb`
- `name`: display name shown in settings
- `version`: plugin implementation version
- `protocol_version`: currently `1.0`
- `health.path`: relative health endpoint such as `/health`

Optional fields:

- `description`
- `homepage_url`
- `capabilities`
- `configuration_schema`

Capabilities are declared as:

```json
{
  "capability": "metadata.search",
  "endpoint": {
    "path": "/metadata/search",
    "method": "POST",
    "timeout": "10s",
    "authenticated": false
  }
}
```

Supported capability values:

- `metadata.search`
- `metadata.detail`
- `storage.browse`
- `storage.resolve`
- `storage.link`

## Health

The manifest’s `health.path` is polled by Mibo to refresh availability and cooldown state.

Response shape:

```json
{
  "status": "ok",
  "message": "ready",
  "failure_reason": "",
  "cooldown_until": "2026-05-27T10:00:00Z",
  "checked_at": "2026-05-27T09:59:30Z"
}
```

Supported `status` values:

- `ok`
- `degraded`
- `unavailable`

Mibo stores:

- availability status
- failure reason
- cooldown timestamp
- last checked timestamp

## Configuration Schema

Plugins can declare admin-managed configuration fields in `configuration_schema.fields`.

Supported field types:

- `string`
- `secret`
- `number`
- `boolean`
- `select`
- `url`
- `duration`

Each field supports:

- `key`
- `type`
- `required`
- `default`
- `display.label`
- `display.description`
- `display.help_text`
- `display.placeholder`
- `options` for `select`
- `minimum` and `maximum` for `number`

Secrets are stored server-side and returned to the frontend in redacted form. Editing a plugin instance keeps existing secret values unless the user supplies a replacement.

## Metadata Payloads

### `metadata.search`

Request:

```json
{
  "item_type": "movie",
  "query": "Spirited Away",
  "year_hint": 2001,
  "external_id_hints": [
    {
      "provider": "tmdb",
      "provider_type": "tmdb",
      "external_id": "129"
    }
  ],
  "preferred_language": "zh-CN",
  "library_context": {
    "library_id": 2,
    "library_type": "movie"
  }
}
```

Response:

```json
{
  "candidates": [
    {
      "provider": "mock-metadata",
      "media_type": "movie",
      "external_id": "spirited-away-2001",
      "title": "Spirited Away",
      "original_title": "千と千尋の神隠し",
      "poster_url": "https://example.com/poster.jpg",
      "release_date": "2001-07-20",
      "year": 2001,
      "confidence": 0.98,
      "reason_summary": "title and year matched"
    }
  ]
}
```

### `metadata.detail`

Request:

```json
{
  "provider": "mock-metadata",
  "media_type": "movie",
  "external_id": "spirited-away-2001",
  "preferred_language": "zh-CN"
}
```

Response:

```json
{
  "detail": {
    "provider": "mock-metadata",
    "provider_type": "mock",
    "external_id": "spirited-away-2001",
    "title": "Spirited Away",
    "original_title": "千と千尋の神隠し",
    "overview": "A young girl enters a world of spirits.",
    "release_date": "2001-07-20",
    "year": 2001,
    "runtime_seconds": 7500,
    "community_rating": 8.6,
    "tags": [
      {
        "kind": "genre",
        "name": "Animation"
      }
    ],
    "images": [
      {
        "image_type": "poster",
        "url": "https://example.com/poster.jpg",
        "selected": true
      }
    ],
    "external_ids": [
      {
        "provider": "mock-metadata",
        "provider_type": "mock",
        "external_id": "spirited-away-2001",
        "is_primary": true
      }
    ]
  }
}
```

## Storage Payloads

### `storage.browse`

Request:

```json
{
  "path": "/library",
  "refresh": false
}
```

Response:

```json
{
  "objects": [
    {
      "name": "Movies",
      "path": "/library/Movies",
      "is_dir": true,
      "provider": "mock-storage"
    }
  ]
}
```

### `storage.resolve`

Request:

```json
{
  "path": "/library/Movies/Spirited Away (2001)/movie.mkv"
}
```

Response:

```json
{
  "provider": "mock-storage",
  "path": "/library/Movies/Spirited Away (2001)/movie.mkv",
  "object": {
    "name": "movie.mkv",
    "path": "/library/Movies/Spirited Away (2001)/movie.mkv",
    "is_dir": false,
    "size": 734003200,
    "raw_url": "https://storage.example.com/raw/movie.mkv",
    "stable_identity": "mock-storage:spirited-away-2001",
    "provider": "mock-storage"
  },
  "capabilities": {
    "can_browse": true,
    "can_resolve": true,
    "can_link": true
  }
}
```

### `storage.link`

Request:

```json
{
  "path": "/library/Movies/Spirited Away (2001)/movie.mkv"
}
```

Response:

```json
{
  "url": "https://storage.example.com/signed/movie.mkv?token=demo",
  "expires_at": "2026-05-27T11:00:00Z",
  "content_type": "video/x-matroska"
}
```

## Error Mapping

Plugin HTTP failures are normalized before surfacing into Mibo routing and health logic.

Status mapping:

- `401` or `403` -> `unauthorized`
- `404` -> `not_found`
- `429` -> `rate_limited`
- `502`, `503`, `504` -> `unavailable`
- other `5xx` -> `upstream_error`
- other non-2xx or invalid JSON -> `invalid_payload`
- transport failures and timeouts -> `unavailable`

For `429`, Mibo also parses the `Retry-After` header into a cooldown duration when possible.

## Example Mock Plugins

This repository includes two runnable mock services:

- `cd /Users/atlan/Desktop/IdeaProjects/Mibo/mibo-server && go run ./cmd/mock-metadata-plugin`
- `cd /Users/atlan/Desktop/IdeaProjects/Mibo/mibo-server && go run ./cmd/mock-storage-plugin`

They expose real manifests and capability endpoints so you can exercise registration, schema rendering, health refresh, metadata profile selection, media source setup, scanning, and playback link flows end to end.
