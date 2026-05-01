# Media Graph Scanner Architecture Plan

本文档定义 Mibo 下一代媒体扫描、归类、元数据与 Emby-like 输出架构。

结论：在当前目标下，推荐方案是 **Media Graph + Resolver Pipeline + Identity Layer + DTO Adapter**。它比“单文件分类”和“Scanner Profile 插件化分类”更适合长期扩展，也比直接复刻 Emby/Jellyfin 内部模型更可控。

## 1. 结论

当前没有必要追求比 Media Graph 更复杂的方案。

可选但不推荐的更复杂方案包括：

- 完整事件溯源：每次扫描、解析、修正都作为事件流保存。
- 通用知识图谱数据库：把所有媒体对象、人物、文件、Provider 关系放入图数据库。
- 完整 Emby/Jellyfin 模型复刻：直接跟随其对象、API、插件体系。

这些方案理论上更强，但对 Mibo 当前阶段过重：

- 实现成本高。
- 调试复杂。
- 和现有 SQLite/GORM/Catalog/Inventory 模型割裂。
- 电影、电视剧主路径收益不明显。

推荐采用的方案：

```text
Storage Provider
  -> Inventory Scanner
  -> Media Graph Builder
  -> Resolver Pipeline
  -> Identity Reconciliation
  -> Catalog / Inventory Projection
  -> Emby-like DTO Adapter
```

它兼顾：

- 解决同目录电视剧被拆成多个 series。
- 支持电影多版本、extras、外挂字幕。
- 保留当前 Catalog/Inventory 模型投资。
- 让未来音乐、文档、图片扩展不污染视频扫描逻辑。
- 让扫描决策可解释、可回放、可治理。

## 2. 当前问题

当前扫描路径大致是：

```text
video file
  -> classifyMediaFile
  -> catalogScanArtifactFromObject
  -> writeCatalogScan
  -> queue metadata match
```

核心缺陷：

- 每个文件独立判断 `SeriesTitle`。
- `canonicalSeriesPath(seriesTitle)` 依赖解析出的标题。
- 缺少目录级归组阶段。
- 同一目录内不同命名风格会生成多个 series。
- 电影逻辑对象当前更接近“文件”，不利于多版本和 extras。
- 输出结构是 Mibo 内部 detail，不是稳定的 Emby-like 对外契约。

典型失败案例：

```text
/tv/灵笼第二季/
  灵笼 第二季.S02E01.mp4
  Incarnation.S02E02.mp4
  第03集.mp4
```

不应生成：

```text
Series: 灵笼 第二季
Series: Incarnation
Movie or bad episode: 第03集
```

应生成：

```text
Series: 灵笼第二季
  Season 2
    Episode 1
    Episode 2
    Episode 3
```

## 3. 设计原则

### 3.1 扫描只采集事实

扫描层不应急着决定“这是电影还是剧集”。它只采集：

- 文件路径。
- 文件大小。
- 修改时间。
- provider stable identity。
- hash。
- 文件扩展名与容器。
- sidecar 存在性。
- provider metadata。

业务解释交给 Resolver。

### 3.2 标题不是身份

标题是展示字段，不是唯一 key。

稳定身份应该来自：

- library id。
- scanner profile。
- 目录 group path。
- season/episode slot。
- provider external id。
- manual identity。

### 3.3 Catalog 是媒体语义，不是文件系统镜像

`CatalogItem` 表示用户看到的对象：Movie、Series、Season、Episode、Album、Track、Document。

文件路径属于 `InventoryFile`。

播放/打开入口属于 `MediaAsset`。

### 3.4 Resolver 可组合

不要写一个巨大 `classifyMediaFile`。

每个 Resolver 只负责一种证据或决策：目录形态、文件名信号、sidecar、电影归组、剧集归组、asset 关系、Provider 匹配、人工修正。

### 3.5 输出契约和内部模型解耦

内部继续使用 Mibo Catalog/Inventory。

外部通过 DTO Adapter 输出 Emby-like JSON。

## 4. 分层架构

```text
┌──────────────────────────────────────────────┐
│ Clients / API Consumers                       │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│ DTO Adapter                                   │
│ - Mibo detail DTO                             │
│ - Emby-like media item DTO                    │
│ - Future compatibility DTO                    │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│ Domain Projection                             │
│ - CatalogItem                                 │
│ - MediaAsset                                  │
│ - InventoryFile                               │
│ - MediaStream                                 │
│ - People / Images / Tags / External IDs       │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│ Identity Reconciliation                       │
│ - scanner identities                          │
│ - provider identities                         │
│ - sidecar identities                          │
│ - manual identities                           │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│ Resolver Pipeline                             │
│ - DirectoryShapeResolver                      │
│ - FilenameSignalResolver                      │
│ - SidecarResolver                             │
│ - MovieResolver                               │
│ - SeriesResolver                              │
│ - SeasonEpisodeResolver                       │
│ - AssetResolver                               │
│ - MetadataProviderResolver                    │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│ Media Graph                                   │
│ - directory nodes                             │
│ - file nodes                                  │
│ - sidecar nodes                               │
│ - candidate work nodes                        │
│ - candidate relationship edges                │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│ Inventory Scanner                             │
│ - list provider objects                       │
│ - collect facts                               │
│ - update inventory                            │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│ Storage Provider                              │
│ - local                                       │
│ - OpenList                                    │
│ - future NAS/cloud adapters                   │
└──────────────────────────────────────────────┘
```

## 5. Core Data Concepts

### 5.1 Inventory Fact

Inventory fact 是扫描得到的物理事实。

```text
InventoryFact
  library_id
  storage_provider
  storage_path
  parent_path
  is_dir
  extension
  object_type
  stable_identity_key
  hashes_json
  size_bytes
  modified_at
  provider_meta
```

对应现有模型：

- `InventoryFile`
- `MediaSource`
- `Library`

### 5.2 Media Graph Node

Media Graph 是解析阶段的中间表达。

```text
GraphNode
  node_id
  kind = directory | file | sidecar | candidate_work | candidate_asset
  path
  attributes
  evidence
```

第一阶段可只做内存结构，不必落库。

### 5.3 Decision

每个 Resolver 输出决策。

```text
Decision
  decision_type
  target_key
  target_kind
  confidence
  evidence_refs
  reason
  warnings
```

示例：

```text
decision_type = series_group
target_key = scanner:series:/tv/灵笼第二季
target_kind = series
confidence = 0.82
reason = directory contains multiple episode-like videos
```

### 5.4 Catalog Identity

推荐新增 `catalog_identities` 表。

```text
catalog_identities
  id
  item_id
  provider
  identity_type
  identity_key
  source_path
  confidence
  evidence_json
  created_at
  updated_at
```

唯一约束建议：

```text
provider + identity_type + identity_key
```

身份示例：

```text
scanner:movie:/movies/Inception (2010)
scanner:series:/tv/Breaking Bad (2008)
scanner:season:/tv/Breaking Bad (2008):S01
scanner:episode:/tv/Breaking Bad (2008):S01:E01
tmdb:movie:27205
tmdb:tv:1396
tmdb:tv_episode:62085
imdb:title:tt1375666
manual:series:<uuid>
```

## 6. Resolver Pipeline

### 6.1 DirectoryShapeResolver

负责判断目录形态。

输入：目录节点、子目录、视频文件、sidecar。

输出：

```text
directory_shape = movie_folder | series_folder | season_folder | flat_episode_folder | mixed_folder | unknown
```

判断信号：

- 是否包含 `Season 1`、`S01`、`第一季` 等目录。
- 是否包含多个 episode-like 文件。
- 是否包含 `movie.nfo` 或 `tvshow.nfo`。
- 是否只有一个主视频文件。
- 是否存在 extras/trailer/sample 子目录。

### 6.2 FilenameSignalResolver

负责从文件名提取低层信号。

输出：

```text
title_candidate
year_candidate
season_number
episode_number
episode_number_end
quality_label
edition
release_group
source_tags
is_sample
is_trailer
```

支持格式：

```text
S01E02
S01E01-E02
S01E01E02
1x02
第03集
03
Show - 03
Movie.Name.2024.2160p.WEB-DL
```

### 6.3 SidecarResolver

负责 sidecar 读取和证据生成。

支持：

- `.srt`
- `.ass`
- `.nfo`
- `.json`

层级：

```text
group sidecar:
  movie.nfo
  tvshow.nfo
  season.nfo
  metadata.json

file sidecar:
  Inception.nfo
  Inception.chi.srt
  Show.S01E01.nfo
```

字段优先级：

```text
manual locked field
  > selected provider field
  > exact file sidecar
  > group sidecar
  > resolver inference
```

### 6.4 MovieResolver

负责将电影目录归并为一个 Movie。

典型输入：

```text
/movies/Inception (2010)/
  Inception.mkv
  Inception.2160p.mkv
  trailer.mp4
  movie.nfo
```

输出：

```text
Movie item key = scanner:movie:/movies/Inception (2010)
Main asset candidate = Inception.mkv
Version asset candidate = Inception.2160p.mkv
Extra asset candidate = trailer.mp4
```

主文件判断：

```text
1. sidecar explicit main
2. filename close to directory name
3. longest duration
4. largest size
5. not sample/trailer/extra
```

### 6.5 SeriesResolver

负责将同一剧目录归到一个 Series。

典型输入：

```text
/tv/灵笼第二季/
  灵笼 第二季.S02E01.mp4
  Incarnation.S02E02.mp4
  第03集.mp4
```

输出：

```text
Series item key = scanner:series:/tv/灵笼第二季
Series title candidate = 灵笼第二季
```

关键规则：

- 同一 flat episode folder 内的 episode-like 文件优先归到同一个 series。
- 文件名中的 `Incarnation` 只能作为别名或 original title 信号，不能单独生成 series。
- 如果存在 `tvshow.nfo`，以 sidecar series title 为显示名。
- 如果目录名包含季信息，可以作为 season hint，不应强行拆 series title。

### 6.6 SeasonEpisodeResolver

负责生成 Season/Episode slot。

输入：Series decision、目录形态、文件名信号。

输出：

```text
Season key = scanner:season:<series-key>:S02
Episode key = scanner:episode:<series-key>:S02:E03
```

无季目录处理：

- 如果目录名或文件名包含季信息，使用该季。
- 如果 TV library 且无季信息，默认 Season 1。
- 如果 mixed library 且无季信息，标记 `needs_review`，但仍保持同 series group。

多集文件处理：

```text
S01E01-E02
  -> Episode S01E01
  -> Episode S01E02
  -> one MediaAsset linked to both
```

### 6.7 AssetResolver

负责生成资产关系。

输出：

```text
MediaAsset
AssetItem
AssetFile
InventoryFile
```

Asset 类型：

```text
main
version
multi_episode
extra
trailer
sample
subtitle
document
```

Link role：

```text
primary
version
multi_episode_part
extra
```

### 6.8 MetadataProviderResolver

负责外部匹配。

电影：

```text
Movie -> TMDB movie search/detail
```

电视剧：

```text
Series -> TMDB/TVDB series search/detail
Season/Episode -> follow provider hierarchy
```

原则：

- Episode 不单独用剧名搜索。
- Episode 通过 `series provider id + season + episode` 定位。
- 低置信度进入 `needs_review`，不要拆分对象。

## 7. Projection to Current Models

现有模型可以继续作为落库目标。

```text
CatalogItem
  Movie / Series / Season / Episode / future Album / Track / Document

MediaAsset
  playable/openable resource

InventoryFile
  physical file

MediaStream
  ffprobe stream

CatalogExternalID
  provider ids

ItemImage
  posters / backdrops / stills / logos

ItemPerson
  cast / director / artist / author

Tag / ItemTag
  genres / tags / studios if desired

MetadataSource
  provider and scanner evidence

MetadataFieldState
  field override and locks
```

需要新增：

```text
catalog_identities
```

可选新增：

```text
scan_decisions
```

如果不想第一阶段新增 `scan_decisions`，可以先把 decision payload 写入 `MetadataSource`。

## 8. Movie DTO

Movie 输出接近 Emby：

```json
{
  "Name": "Inception",
  "ServerId": "xxx",
  "Id": "12345",
  "Type": "Movie",
  "MediaType": "Video",
  "Path": "/movies/Inception (2010)",
  "ProductionYear": 2010,
  "PremiereDate": "2010-07-15T00:00:00.0000000Z",
  "Overview": "A thief who steals corporate secrets...",
  "CommunityRating": 8.8,
  "RunTimeTicks": 88800000000,
  "Genres": ["Action", "Science Fiction", "Thriller"],
  "Tags": [],
  "Studios": [
    {
      "Name": "Warner Bros. Pictures",
      "Id": 101
    }
  ],
  "ProviderIds": {
    "Imdb": "tt1375666",
    "Tmdb": "27205"
  },
  "People": [],
  "ImageTags": {},
  "BackdropImageTags": [],
  "MediaSources": []
}
```

推荐：Movie `Path` 使用作品目录，`MediaSources[].Path` 使用具体文件。

兼容模式可提供：Movie `Path` 使用主媒体文件路径。

## 9. Series DTO

Series：

```json
{
  "Name": "Breaking Bad",
  "Id": "100",
  "Type": "Series",
  "MediaType": "Video",
  "Path": "/tv/Breaking Bad (2008)",
  "ProductionYear": 2008,
  "PremiereDate": "2008-01-20T00:00:00.0000000Z",
  "EndDate": "2013-09-29T00:00:00.0000000Z",
  "Status": "Ended",
  "Overview": "A high school chemistry teacher diagnosed with cancer...",
  "Genres": ["Drama", "Crime"],
  "CommunityRating": 9.5,
  "OfficialRating": "TV-MA",
  "ProviderIds": {
    "Imdb": "tt0903747",
    "Tmdb": "1396",
    "Tvdb": "81189"
  },
  "People": [],
  "ImageTags": {},
  "BackdropImageTags": [],
  "RecursiveItemCount": 62,
  "ChildCount": 5
}
```

Season：

```json
{
  "Name": "Season 1",
  "Id": "101",
  "Type": "Season",
  "SeriesId": "100",
  "SeriesName": "Breaking Bad",
  "IndexNumber": 1,
  "ParentIndexNumber": null,
  "Path": "/tv/Breaking Bad (2008)/Season 1",
  "ProviderIds": {
    "Tvdb": "30272",
    "Tmdb": "3572"
  },
  "ImageTags": {
    "Primary": "season-poster-tag"
  },
  "ChildCount": 7
}
```

Episode：

```json
{
  "Name": "Pilot",
  "Id": "102",
  "Type": "Episode",
  "MediaType": "Video",
  "SeriesId": "100",
  "SeriesName": "Breaking Bad",
  "SeasonId": "101",
  "SeasonName": "Season 1",
  "IndexNumber": 1,
  "ParentIndexNumber": 1,
  "Path": "/tv/Breaking Bad (2008)/Season 1/Breaking Bad - S01E01 - Pilot.mkv",
  "ProductionYear": 2008,
  "PremiereDate": "2008-01-20T00:00:00.0000000Z",
  "Overview": "Walter White, a struggling high school chemistry teacher...",
  "RunTimeTicks": 34800000000,
  "ProviderIds": {
    "Imdb": "tt0959621",
    "Tmdb": "62085",
    "Tvdb": "349232"
  },
  "People": [],
  "ImageTags": {},
  "MediaSources": []
}
```

## 10. MediaSource DTO

`MediaAsset + InventoryFile + MediaStream` 映射为 Emby-like `MediaSource`。

```json
{
  "Id": "source-id",
  "Path": "/movies/Inception (2010)/Inception.mkv",
  "Protocol": "File",
  "Container": "mkv",
  "Size": 12345678900,
  "Name": "1080p - HEVC",
  "IsRemote": false,
  "RunTimeTicks": 88800000000,
  "VideoType": "VideoFile",
  "DefaultAudioStreamIndex": 1,
  "DefaultSubtitleStreamIndex": 2,
  "MediaStreams": []
}
```

Stream 映射：

```text
MediaStream.StreamIndex       -> Index
MediaStream.StreamType        -> Type
MediaStream.Codec             -> Codec
MediaStream.Language          -> Language
MediaStream.Width             -> Width
MediaStream.Height            -> Height
MediaStream.BitRate           -> BitRate
MediaStream.AvgFrameRate      -> AverageFrameRate
MediaStream.RFrameRate        -> RealFrameRate
MediaStream.Profile           -> Profile
MediaStream.Level             -> Level
MediaStream.Channels          -> Channels
MediaStream.SampleRate        -> SampleRate
DispositionJSON.default       -> IsDefault
AssetFile role subtitle       -> IsExternal / Path for external subtitles
```

## 11. API Design

建议先增加 Mibo 自有 API：

```text
GET /api/v1/media/items/{id}
GET /api/v1/media/items/{id}/children
GET /api/v1/media/items/{id}/sources
GET /api/v1/media/items/{id}/playback-info
```

后续再加 Emby compatibility shim：

```text
GET /emby/Items/{id}
GET /emby/Items/{id}/Children
GET /emby/Items/{id}/PlaybackInfo
```

不要第一阶段承诺完整 Emby API，只承诺 DTO shape 接近。

## 12. Extension Design

### 12.1 Music

输入：

```text
/music/Daft Punk/Random Access Memories/01 - Give Life Back to Music.flac
```

解析：

```text
Artist = Daft Punk
Album = Random Access Memories
Track = 01 - Give Life Back to Music
Asset = flac file
Provider = embedded tags + MusicBrainz
```

Catalog：

```text
MusicArtist
  MusicAlbum
    Audio
      MediaAsset
        InventoryFile
        MediaStream
```

### 12.2 Documents

输入：

```text
/docs/Project A/Design.pdf
/docs/Project A/Spec.docx
```

Catalog：

```text
DocumentCollection
  Document
    MediaAsset
      InventoryFile
```

文档扩展字段不要硬塞到视频模型，可用 domain projection 或 metadata field。

## 13. Implementation Phases

### Phase 1: In-memory Media Graph for video

目标：先不大改数据库，只在扫描过程中引入中间图谱。

工作：

- 从 `walkDirectory` 中提取目录对象收集阶段。
- 构建 directory/file/sidecar graph。
- 新增 TV directory grouping。
- 新增 Movie folder grouping。
- 输出现有 `catalogScanArtifact` 或替代 plan。

验收：

- 同目录剧集不再拆成多个 series。
- 标准 TV 目录保持正常。
- 电影单文件保持正常。

### Phase 2: Catalog identities

目标：稳定重扫和去重。

工作：

- 新增 `catalog_identities`。
- 写入 scanner identities。
- Reconcile 优先按 identity 找 item。
- sidecar/provider/manual identity 逐步接入。

验收：

- 改文件名噪声但目录不变，不重复创建 series/movie。
- TMDB 修正标题后，scanner identity 仍保持归并稳定。

### Phase 3: Resolver pipeline extraction

目标：从大函数迁移为可解释 Resolver。

工作：

- 抽出 `FilenameSignalResolver`。
- 抽出 `DirectoryShapeResolver`。
- 抽出 `MovieResolver`。
- 抽出 `SeriesResolver`。
- 抽出 `SeasonEpisodeResolver`。
- 记录 decision reason。

验收：

- 测试覆盖每个 Resolver。
- 能输出每个 item 的 scanner evidence。

### Phase 4: Emby-like DTO adapter

目标：输出稳定媒体对象。

工作：

- 新增 DTO types。
- 映射 Movie/Series/Season/Episode。
- 映射 MediaSources/MediaStreams。
- 新增 `/api/v1/media/items/{id}`。

验收：

- Movie DTO 满足示例字段主路径。
- Episode DTO 包含 Series/Season context 和 MediaSources。
- `RunTimeTicks = seconds * 10000000`。

### Phase 5: Provider metadata hierarchy

目标：让外部元数据跟随 root object。

工作：

- Movie root 匹配 TMDB movie。
- Series root 匹配 TMDB/TVDB series。
- Season/Episode 跟随 provider hierarchy。
- 低置信度只标记治理，不拆分 catalog。

验收：

- 打开 Episode 时能看到 provider episode metadata。
- Series 修正后 descendant episode identity 能同步。

### Phase 6: Music and documents

目标：验证架构扩展性。

工作：

- Music directory/tag resolver。
- Document file resolver。
- 对应 DTO adapter。

验收：

- 不修改视频 Resolver 即可增加音乐/文档基础扫描。

## 14. Test Matrix

### TV flat folder

```text
/tv/灵笼第二季/灵笼 第二季.S02E01.mp4
/tv/灵笼第二季/Incarnation.S02E02.mp4
/tv/灵笼第二季/第03集.mp4
```

期望：

```text
1 Series
1 Season
3 Episodes
3 Assets
0 duplicate series
```

### TV standard folder

```text
/tv/Breaking Bad (2008)/Season 1/Breaking Bad - S01E01 - Pilot.mkv
```

期望：

```text
Series -> Season 1 -> Episode 1 -> Asset -> File -> Streams
```

### TV multi-episode file

```text
/tv/Show/Season 1/Show.S01E01-E02.mkv
```

期望：

```text
1 Asset linked to E01 and E02
role = multi_episode_part
segment_index = 1 and 2
```

### Movie multi-version

```text
/movies/Inception (2010)/Inception.1080p.mkv
/movies/Inception (2010)/Inception.2160p.mkv
```

期望：

```text
1 Movie
2 MediaSources
0 duplicate movie
```

### Movie extras

```text
/movies/Inception (2010)/Inception.mkv
/movies/Inception (2010)/trailer.mp4
/movies/Inception (2010)/behind-the-scenes.mkv
```

期望：

```text
1 Movie
1 main asset
2 extra assets
```

### Rename stability

步骤：

```text
scan file with noisy filename
rename file to cleaner filename under same group
rescan
```

期望：

```text
No duplicate CatalogItem
Identity reused
Metadata preserved unless scanner-owned field
```

## 15. Open Questions

需要实现前确认：

- Movie DTO 的 `Path` 默认使用作品目录，还是主文件路径。
- 是否第一阶段就新增 `catalog_identities`，还是先用 `MetadataSource` 过渡。
- Mixed library 是否第一阶段启用，还是先只支持 explicit movies/tv libraries。
- 是否需要完整 Emby compatibility endpoints，还是先提供 Mibo `/api/v1/media/*`。

推荐默认答案：

- Movie `Path` 使用作品目录，兼容模式可返回主文件路径。
- 第一阶段可以先不新增 `scan_decisions`，但建议尽早新增 `catalog_identities`。
- Mixed library 延后。
- 先做 Mibo API，后做 Emby shim。

## 16. Final Recommendation

采用：

```text
Media Graph + Resolver Pipeline + Identity Layer + DTO Adapter
```

不要继续强化单文件正则。

不要直接复刻 Emby 内部模型。

不要一开始引入完整事件溯源或图数据库。

第一阶段以最小可落地方式实现：

```text
in-memory graph
  + TV directory grouping
  + Movie folder grouping
  + scanner identity evidence
  + current Catalog/Inventory write model
```

这样能快速修复当前痛点，同时保留长期演进空间。
