# Mibo

Mibo 是一个全栈媒体系统：前端负责媒体库浏览、播放、搜索、直播电视和管理工作台，后端负责媒体源接入、媒体库扫描、元数据治理、播放访问、用户会话、任务调度与系统设置。

它不是一个单独的 Vite 页面项目。根目录是 React Web UI，`mibo-server/` 是 Go 后端服务子模块，最终可以把前端静态资源嵌入后端，构建成一个可独立运行的媒体服务器。

## 核心能力

- 媒体库：电影、剧集、剧集分季分集、媒体详情、资源选择、继续观看、最近播放。
- 播放：Web 播放页、HLS/直连播放、字幕和音轨展示、外部播放器启动偏好。
- 直播电视：IPTV 源管理、频道列表、频道分组、直播播放。
- 元数据：本地扫描证据、TMDB/插件元数据提供方、元数据匹配、治理工作台、手动搜索和候选应用。
- 用户系统：首次初始化、管理员创建、登录、PIN 登录、会话管理、用户设置、收藏和播放进度。
- 后台任务：媒体库扫描、workflow 调度、任务状态、计划任务、操作与问题诊断。
- 管理设置：媒体源、媒体库、播放、字幕、网络、插件、日志、计划任务、元数据源、设备和系统控制台。

## 技术栈

### 前端

- React 19
- TypeScript
- Vite
- TanStack Router
- TanStack Query
- Tailwind CSS 4
- Radix UI / shadcn 风格组件
- Vitest + Playwright browser runner

### 后端

- Go
- `mibo-server/` Git submodule
- SQLite / Postgres / MySQL
- OpenList 与本地/插件化存储 Provider
- Metadata / Storage / Subtitle Provider 插件协议
- 内置 worker、workflow runner、schedule service
- 可嵌入前端静态资源的单体服务

## 目录结构

```text
.
├── src/                         # React 前端源码
│   ├── routes/                  # TanStack Router 文件路由
│   ├── features/                # 按业务能力组织的页面和交互
│   ├── components/              # 共享 UI 组件
│   ├── lib/                     # API client、query key、工具函数
│   ├── hooks/                   # 共享 hooks
│   ├── stores/                  # Zustand 等状态管理
│   └── styles/                  # 全局样式和主题
├── mibo-server/                 # Go 后端服务子模块
│   ├── cmd/                     # 服务和辅助命令入口
│   ├── internal/                # 后端核心模块
│   ├── data/                    # 本地数据目录
│   └── bin/                     # 本地构建产物
├── docs/                        # 协议、迁移、设计和 rollout 文档
├── openspec/                    # OpenSpec 变更说明
├── scripts/                     # 全栈开发、构建、部署辅助脚本
├── public/                      # 前端静态资源
├── dist/                        # 前端构建产物，不要手改
└── build/                       # 组合构建产物，不要手改
```

## 本地开发

### 1. 安装依赖

```bash
pnpm install
```

如果浏览器测试依赖还没有安装：

```bash
pnpm test:browser:install
```

### 2. 初始化后端子模块

```bash
git submodule update --init --recursive
```

### 3. 启动全栈开发环境

```bash
pnpm dev:fullstack
```

这个脚本会同时启动：

- 前端：`pnpm dev`
- 后端：`MIBO_HTTP_ADDR=:8096 go run ./cmd/mibo-media-server`

Vite 会把 `/api` 代理到 `http://127.0.0.1:8096`，所以本地前端开发时通常让 `VITE_API_BASE_URL` 保持为空。

### 4. 首次初始化

打开 Vite 输出的本地地址后，如果系统还没有初始化，会进入 `/setup`。

首次初始化会完成：

- 选择或确认数据库配置
- 创建第一个管理员
- 进入媒体系统管理界面

默认数据库是 SQLite，默认后端数据文件在 `mibo-server/data/mibo.db` 或服务运行目录的 `data/mibo.db`。

## 单独启动

### 只启动前端

```bash
pnpm dev
```

开发模式下，如果没有设置 `VITE_API_BASE_URL`，前端会使用 Vite `/api` 代理。

### 只启动后端

```bash
cd mibo-server
go run ./cmd/mibo-media-server
```

后端默认监听 `:8080`。全栈脚本会改用 `:8096`，以匹配 Vite 代理。

健康检查：

```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/readyz
```

## 配置

前端常用环境变量：

```bash
# 本地开发留空，使用 Vite /api 代理。
VITE_API_BASE_URL=

# 如果启用 Clerk 路由，可填 Clerk publishable key。
VITE_CLERK_PUBLISHABLE_KEY=
```

后端常用环境变量：

```bash
MIBO_HTTP_ADDR=:8080
MIBO_DATABASE_DRIVER=sqlite
MIBO_DATABASE_DSN=data/mibo.db
MIBO_OPENLIST_BASE_URL=http://127.0.0.1:5244
MIBO_OPENLIST_TOKEN=
MIBO_OPENLIST_ROOT_PATH=/
MIBO_TMDB_API_KEY=
MIBO_FFPROBE_ENABLED=true
MIBO_FFPROBE_PATH=ffprobe
MIBO_WORKER_ENABLED=true
```

更多后端配置见 [mibo-server/README.md](mibo-server/README.md)。

## 常用命令

```bash
# 前端开发
pnpm dev

# 前后端一起开发
pnpm dev:fullstack

# 前端类型检查并构建
pnpm build

# ESLint
pnpm lint

# Prettier 检查
pnpm format:check

# 前端测试
pnpm test

# 前端覆盖率
pnpm test:coverage

# 后端测试
cd mibo-server && go test ./...

# 生成 release notes
pnpm release:notes
```

## 构建可独立运行的服务

```bash
git submodule update --init --recursive
./scripts/build-with-frontend.sh
```

脚本会：

1. 构建前端到 `dist/`
2. 复制静态资源到 `mibo-server/internal/webui/dist`
3. 构建后端二进制到 `build/mibo-server`

运行构建产物：

```bash
./build/mibo-server
```

如需自定义输出路径：

```bash
MIBO_OUTPUT=/tmp/mibo-media-server ./scripts/build-with-frontend.sh
```

## Docker

仓库包含多阶段 `Dockerfile`：

- `node:22-alpine` 构建前端
- `golang:1.24-alpine` 构建后端
- `alpine:3.22` 作为运行镜像

运行镜像默认：

- 服务端口：`18081`
- 数据目录：`/data`
- 媒体目录：`/media`
- SQLite DSN：`/data/mibo.db`

示例：

```bash
docker build -t mibo .
docker run --rm \
  -p 18081:18081 \
  -v "$PWD/mibo-server/data:/data" \
  -v "$PWD/demo-media:/media:ro" \
  mibo
```

## 后端模块速览

`mibo-server/internal/` 按业务边界组织，常见模块包括：

- `httpapi`：API 路由、请求处理、setup guard、管理控制台接口。
- `auth`：用户注册、登录、session token、PIN 和会话管理。
- `settings`：系统设置、用户设置、插件 Provider 配置。
- `database`：GORM 模型、迁移和数据库约束。
- `library`：媒体库扫描、目录管线、识别、资源物化。
- `catalog`：首页、媒体条目、层级浏览、搜索和用户媒体状态查询。
- `metadata`：元数据匹配、候选、治理、local evidence。
- `providers` / `providerplugin`：存储和元数据 Provider 抽象与插件协议。
- `live_tv`：直播源、频道、分组和刷新。
- `progress`：播放进度、已观看状态、继续观看。
- `workflow` / `schedule`：后台 workflow、资源预算、计划任务。
- `access`：播放访问签名与资源访问控制。

## 前端模块速览

`src/features/` 里按产品能力拆分：

- `home`：首页推荐、继续观看和状态整理。
- `library` / `library-browser`：媒体库列表和层级浏览。
- `media`：媒体详情、剧集资源选择、收藏、进度更新、播放入口。
- `play`：播放器页面、字幕标签、外部播放器启动。
- `live-tv`：直播频道列表和播放。
- `search`：全局搜索。
- `metadata-governance`：元数据治理工作台和详情修正。
- `operations`：操作问题、元数据 review dialog。
- `settings`：设置页分区和系统配置 UI。
- `plugin-management`：插件 Provider 管理。
- `setup`：首次初始化流程。

## API 与播放访问

前端通过 [src/lib/mibo-api.ts](src/lib/mibo-api.ts) 访问后端，接口统一走 `/api/v1`。

播放相关接口会返回可访问的播放 URL、字幕、音轨和技术信息。访问控制由后端负责，前端只消费后端返回的资源和签名地址，不自行推导后端访问规则。

## 开发约定

- 前端格式：Prettier，2 空格、80 列、单引号、无分号。
- 前端导入：优先使用 `@/...` 或 `#/...` 路径别名。
- React 组件使用 `PascalCase`，hooks 使用 `use*` 命名。
- 测试文件尽量靠近被测代码，常见后缀为 `*.test.ts` 和 `*.test.tsx`。
- 后端代码必须保持 `gofmt` 干净。
- 后端测试放在对应包内，文件名使用 `*_test.go`。
- 派生字段只能有一个所有者。字段进入 pipeline artifact 后，下游只消费该字段，不重新从原始输入推导同一含义。
- 不要手动修改 `dist/`、`build/`、`mibo-server/bin/` 或 `mibo-server/internal/webui/dist/`。

## 提交与 PR

提交信息建议使用 Conventional Commit：

```text
feat: expand media catalog playback
fix: keep playback progress idempotent
test: cover metadata governance fallback
```

PR 建议包含：

- 简短摘要
- 影响范围：`frontend`、`mibo-server` 或 `both`
- 测试证据
- UI 变更的截图或录屏
- 配置、迁移、媒体扫描、元数据或嵌入式 Web 资源相关影响说明

## 排障

### 前端请求不到后端

- 使用 `pnpm dev:fullstack` 时确认后端运行在 `:8096`。
- 单独运行前端时确认 `vite.config.ts` 的 `/api` 代理目标可访问。
- 如果设置了 `VITE_API_BASE_URL`，确认它没有尾部路径错误，例如应为 `http://127.0.0.1:8080`，而不是 `http://127.0.0.1:8080/api`。

### 首次启动进入 setup

这是正常行为。系统没有管理员时会要求完成初始化。若显式设置了数据库环境变量，setup 会把数据库配置视为部署环境管理。

### ffprobe 信息为空

- 确认系统安装了 `ffprobe` 或 Docker 镜像内包含 `ffmpeg`。
- 确认 `MIBO_FFPROBE_ENABLED=true`。
- 确认 `MIBO_FFPROBE_PATH` 指向可执行文件。

### 后端子模块不存在

```bash
git submodule update --init --recursive
```

### 浏览器测试失败

```bash
pnpm test:browser:install
pnpm test
```

## 相关文档

- [Provider Plugin Protocol](docs/provider-plugin-protocol.md)
- [Library Visibility Rollout](docs/library-visibility-rollout.md)
- [Metadata Scope Roots](docs/metadata-scope-roots.md)
- [Backend README](mibo-server/README.md)

## License

This project is source-available, but not open source.

The code is licensed under the PolyForm Noncommercial License 1.0.0.

Non-commercial use is allowed. Commercial use is not permitted without prior written permission from the copyright holder.

For commercial licensing, contact: atlanxg@gmail.com.
