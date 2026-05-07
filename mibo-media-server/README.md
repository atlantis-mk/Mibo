# mibo-media-server

`mibo-media-server` 是基于 `OpenList` 的独立媒体业务服务骨架。

当前已经完成的基础能力：

- 独立 Go 模块与可启动 HTTP 服务
- 配置加载
- `SQLite / Postgres` 数据库初始化
- `StorageProvider` 抽象
- `OpenList Adapter`
- 基础数据模型自动迁移：
  - `media_sources`
  - `libraries`
  - `media_items`
  - `media_files`
  - `jobs`
- 健康检查与就绪检查
- 基础 API：
  - 媒体源管理
  - 媒体库管理
  - 作业列表
  - 系统信息
- 最小 Worker 循环与任务状态流转：
  - `queued`
  - `running`
  - `completed`
  - `failed`
- `M1` 媒体库可用能力：
  - 递归扫描媒体库目录
  - 视频文件过滤与基础分类
  - 电影识别
  - 剧集 `SxxExx / 1x02` 识别
  - `media_files` 与 `media_items` 建立关联
  - 手动重扫
  - 媒体库条目列表与详情查询
- `M2` 元数据与详情能力：
  - `TMDB` 自动匹配
  - 媒体条目元数据状态：`matched / needs_review / unmatched / skipped`
  - 海报、背景图、简介、演员、导演、类型、发布日期
  - `ffprobe` 技术信息补全
  - 视频编码、分辨率、时长、音轨、字幕轨道
  - 手动重匹配
- `M3` 播放闭环能力：
  - 媒体条目最佳播放源选择
  - 媒体文件直链获取
  - 播放前校验：文件存在、可访问、媒体信息状态
  - 直连播放响应中返回音轨、字幕轨道和基础技术信息
- `M4` 用户与进度能力：
  - 用户注册与登录
  - Bearer session token 会话
  - 每用户独立播放进度
  - 已观看 / 未观看状态
  - 继续观看
  - 最近播放
  - 单条媒体的用户进度查询

## 环境变量

- `MIBO_HTTP_ADDR`: HTTP 监听地址，默认 `:8080`
- `MIBO_DATABASE_DRIVER`: `sqlite` 或 `postgres`，默认 `sqlite`
- `MIBO_DATABASE_DSN`: 数据库连接串，默认 `data/mibo.db`
- `MIBO_OPENLIST_BASE_URL`: OpenList 地址，默认 `http://127.0.0.1:5244`
- `MIBO_OPENLIST_TOKEN`: OpenList API token
- `MIBO_OPENLIST_ROOT_PATH`: OpenList 根路径，默认 `/`
- `MIBO_OPENLIST_TIMEOUT`: OpenList 请求超时，默认 `15s`
- `MIBO_OPENLIST_INSECURE_SKIP_VERIFY`: 是否跳过 TLS 校验，默认 `false`
- `MIBO_TMDB_API_KEY`: TMDB API key，可选
- `MIBO_TMDB_BASE_URL`: TMDB API 地址，默认 `https://api.themoviedb.org/3`
- `MIBO_TMDB_IMAGE_BASE_URL`: TMDB 图片地址前缀，默认 `https://image.tmdb.org/t/p/original`
- `MIBO_TMDB_LANGUAGE`: TMDB 语言，默认 `en-US`
- `MIBO_TMDB_TIMEOUT`: TMDB 请求超时，默认 `10s`
- `MIBO_FFPROBE_ENABLED`: 是否启用 `ffprobe`，默认 `true`
- `MIBO_FFPROBE_PATH`: `ffprobe` 路径，默认 `ffprobe`
- `MIBO_FFPROBE_TIMEOUT`: `ffprobe` 超时，默认 `30s`
- `MIBO_WORKER_ENABLED`: 是否启动内置 Worker，默认 `true`
- `MIBO_WORKER_POLL_INTERVAL`: Worker 轮询间隔，默认 `2s`
- `MIBO_WORKFLOW_POLL_INTERVAL`: Workflow runner 轮询间隔，默认 `2s`
- `MIBO_WORKFLOW_LEASE_DURATION`: Workflow task 租约时长，默认 `1m`

## Workflow 调度调优

资源感知 Workflow DAG 当前用于将媒体库扫描拆成按库、按路径、按阶段的后台任务。启用后，扫描、catalog materialize、projection refresh、probe 和 metadata match 可以按资源预算调度，不同库不会因为全局串行 job 被互相阻塞。

默认建议：

- SQLite / 本地开发：保持 `db_write=1` 的保守预算，避免写锁竞争。
- Postgres / 较大部署：可提高数据库写、OpenList HTTP、ffprobe、metadata API 等资源预算。
- Workflow 调度器随 worker 默认启用；legacy jobs 仍保留用于尚未迁移的后台工作。
- 管理员可通过 `GET /api/v1/workflows`、`GET /api/v1/workflows/{id}`、`GET /api/v1/workflows/diagnostics` 查看 workflow 状态、阶段计数、资源等待和租约情况。

## 启动

```bash
go run ./cmd/mibo-media-server
```

## API

### 健康检查

```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/readyz
```

### 创建媒体源

```bash
curl -X POST http://127.0.0.1:8080/api/v1/media-sources \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Home Media",
    "root_path": "/movies"
  }'
```

### 创建媒体库

```bash
curl -X POST http://127.0.0.1:8080/api/v1/libraries \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Movies",
    "type": "movies",
    "media_source_id": 1,
    "root_path": "/movies"
  }'
```

### 查看系统信息

```bash
curl http://127.0.0.1:8080/api/v1/system/info
```

### 查看媒体库详情

```bash
curl http://127.0.0.1:8080/api/v1/libraries/1
```

### 列出媒体条目

```bash
curl http://127.0.0.1:8080/api/v1/libraries/1/items
curl http://127.0.0.1:8080/api/v1/libraries/1/items?type=movie
```

### 查看 catalog 条目详情

```bash
curl http://127.0.0.1:8080/api/v1/items/1
```

### 手动重匹配 catalog 元数据

```bash
curl -X POST http://127.0.0.1:8080/api/v1/items/1/match
```

### 获取 catalog 播放源

```bash
curl http://127.0.0.1:8080/api/v1/items/1/playback?client_profile=web
curl http://127.0.0.1:8080/api/v1/items/1/playback?client_profile=web\&asset_id=10
```

### 获取资产与库存文件直链

```bash
curl http://127.0.0.1:8080/api/v1/assets/10/link
curl http://127.0.0.1:8080/api/v1/inventory-files/20/stream
```

### 注册与登录

```bash
curl -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123"}'

curl -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123"}'
```

### 查看当前用户

```bash
curl http://127.0.0.1:8080/api/v1/me \
  -H 'Authorization: Bearer <token>'
```

### 同步 catalog 播放进度

```bash
curl -X POST http://127.0.0.1:8080/api/v1/me/progress \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "item_id": 1,
    "asset_id": 10,
    "position_seconds": 180
  }'
```

### 查看继续观看和最近播放

```bash
curl http://127.0.0.1:8080/api/v1/me/continue-watching \
  -H 'Authorization: Bearer <token>'

curl http://127.0.0.1:8080/api/v1/me/recently-played \
  -H 'Authorization: Bearer <token>'
```

### 查看单条 catalog 条目的当前用户进度

```bash
curl http://127.0.0.1:8080/api/v1/items/1/progress \
  -H 'Authorization: Bearer <token>'
```

### catalog governance 资产纠错

当前治理工作区支持在“当前条目及其后代”范围内修正资产链接：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/items/1/governance/assets/10/links \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"target_item_id": 12}'

curl -X DELETE http://127.0.0.1:8080/api/v1/items/1/governance/assets/10/links/12 \
  -H 'Authorization: Bearer <token>'
```

这些操作只更新 `asset_items` 关系，不会覆盖字段锁、来源证据或图片选择状态。

### 手动重扫媒体库

```bash
curl -X POST http://127.0.0.1:8080/api/v1/libraries/1/scan
```

### 重试失败任务

```bash
curl -X POST http://127.0.0.1:8080/api/v1/jobs/1/retry
```

## Catalog 运行与恢复

当前 `sync_library` 主路径已经切到 catalog kernel：

- 递归遍历目录并过滤非视频文件。
- Upsert `inventory_files`、`media_assets`、`asset_files` 与 `catalog_items` 层级，而不是继续把新扫描结果写回 legacy `media_items` / `media_files`。
- 对缺失文件做软删除标记，并刷新 catalog availability、rollups 与 search documents。

扫描后的补充任务也以 catalog 实体为主：

- 探测任务围绕 `inventory_files` 更新技术信息与关联 `media_assets`。
- 元数据治理围绕 catalog item 身份执行匹配、重抓、字段锁与图片选择。

播放链路当前采用 catalog item -> asset -> inventory file 解析：

- 优先解析所选或默认 `media_asset`。
- 通过存储提供方获取直链，必要时回退到 `/api/v1/inventory-files/:id/hls/*`。
- 继续观看与最近播放基于 catalog item / asset 进度记录构建。

推荐恢复流程：

1. 重新扫描对应媒体库，刷新 inventory、asset 与 catalog 投影。
2. 如需修正匹配结果或补全详情，在元数据治理工作台中执行重新匹配或重抓。
3. 如果问题来自存储变更，优先重新扫描具体媒体库而不是依赖一次性迁移工具。

## 当前边界

当前版本仍不包含：

- `TVDB` 接入
- 转码和更复杂的设备兼容策略
- 家庭共享的复杂权限模型
- 收藏 / 喜欢 / 更完整播放历史

这些能力可以在现有模块边界上继续向后续里程碑迭代。
