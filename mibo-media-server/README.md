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

### 查看媒体条目详情

```bash
curl http://127.0.0.1:8080/api/v1/media-items/1
```

### 手动重匹配元数据

```bash
curl -X POST http://127.0.0.1:8080/api/v1/media-items/1/match
```

### 获取播放源

```bash
curl http://127.0.0.1:8080/api/v1/media-items/1/playback
curl http://127.0.0.1:8080/api/v1/media-items/1/playback?file_id=1
```

### 获取文件播放直链

```bash
curl http://127.0.0.1:8080/api/v1/media-files/1/link
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

### 同步播放进度

```bash
curl -X POST http://127.0.0.1:8080/api/v1/me/progress \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "media_item_id": 1,
    "media_file_id": 1,
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

### 查看单条媒体的当前用户进度

```bash
curl http://127.0.0.1:8080/api/v1/media-items/1/progress \
  -H 'Authorization: Bearer <token>'
```

### 手动重扫媒体库

```bash
curl -X POST http://127.0.0.1:8080/api/v1/libraries/1/scan
```

### 重试失败任务

```bash
curl -X POST http://127.0.0.1:8080/api/v1/jobs/1/retry
```

## 当前边界

当前版本已经覆盖 `M0 + M1 + M2 + M3`，并补上了 `M4` 的基础用户与进度链路，但仍不包含：

- 更完整的 TV Season / Episode 元数据建模
- 手动匹配候选选择
- `TVDB` 接入
- 转码和复杂设备兼容策略
- 家庭共享的复杂权限模型
- 收藏 / 喜欢 / 更完整播放历史

这些能力可以在现有模块边界上继续向后续里程碑迭代。

当前 `sync_library` 任务会执行基础扫描：

- 递归遍历目录
- 过滤非视频文件
- 识别电影与剧集文件
- 回写 `media_items` 和 `media_files`
- 对缺失文件做软删除标记

扫描发现文件后，会继续排队后台富化任务：

- `match_media_item`
- `probe_media_file`

播放接口当前采用直连策略：

- 优先从 `media_items` 选择最佳 `media_file`
- 通过 `OpenList` 获取直链
- 返回可播放性检查结果和技术信息

当前认证与进度系统采用最小会话模式：

- 登录后返回 Bearer token
- 每个用户独立维护 `playback_progress`
- 继续观看和最近播放基于用户进度记录构建

当前仍未实现高级媒体语义：

- 多季多集结构建模
- 手动匹配候选确认
- 更复杂的置信度和冲突处理
- 多元数据源聚合
