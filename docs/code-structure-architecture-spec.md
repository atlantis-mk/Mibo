# Mibo 代码结构规范 / 架构规范

本文档定义当前 `Mibo` 仓库的代码结构约定、模块边界与演进规则。目标不是追求抽象上的完美分层，而是保证当前项目在持续迭代时仍然具备：

- 目录职责清晰
- 依赖方向稳定
- 前后端边界明确
- 新功能落点可预测
- 后续重构不破坏现有结构

## 1. 仓库级边界

仓库当前不是单一应用，而是由多个边界明确的子项目组成。

```text
Mibo/
├── web/                # 前端应用
├── mibo-media-server/  # 后端服务
├── OpenList/           # 上游项目，外部边界
├── demo-media/         # 本地演示媒体数据
├── docs/               # 项目文档
└── package.json        # 根级工具入口，仅用于本地工具
```

规则：

- 产品代码只放在 `web/` 和 `mibo-media-server/`。
- `OpenList/` 视为上游依赖边界，不在其中实现 Mibo 自有业务。
- 根目录 `package.json` 不是前端应用清单，不承载业务脚本或运行时逻辑。
- `demo-media/` 只作为本地调试数据，不作为业务代码依赖。

## 2. 总体架构

当前项目采用「前端 SPA + 后端 HTTP 服务 + 存储适配层 + 后台任务」结构：

- `web/` 负责页面、交互、路由、查询状态和播放体验。
- `mibo-media-server/` 负责认证、媒体库、元数据、播放、进度、任务调度等业务能力。
- `OpenList` 和本地文件系统通过 `StorageProvider` 抽象接入后端。
- 慢操作通过 `jobs + worker` 异步执行，不阻塞在线请求。

建议保持以下依赖方向：

```text
UI Route -> Feature -> Query/API Client -> HTTP API -> Domain Service -> Storage/DB
                                                \-> Jobs -> Worker -> Domain Service
```

禁止反向依赖：

- 前端 feature 不直接依赖具体 route 文件。
- 前端组件不直接写裸 `fetch`，统一走 `lib/mibo-api.ts` 或 `lib/mibo-query.ts`。
- 后端 `internal/httpapi` 不承载核心业务逻辑，只做请求编排、鉴权、参数解析和响应映射。
- 后端 domain service 不依赖前端概念或页面语义。

## 3. 前端规范

### 3.1 目录职责

当前前端以 `web/src/routes` + `web/src/features` 为核心。

```text
web/src/
├── routes/        # TanStack Router 文件路由
├── features/      # 业务功能模块
├── components/    # 跨 feature 复用组件
├── components/ui/ # shadcn/radix-nova 基础 UI 原子组件
├── lib/           # API 客户端、query 封装、工具函数
├── stores/        # 跨页面状态，如鉴权态
├── hooks/         # 通用 hook
└── styles.css     # 全局样式入口
```

职责要求：

- `routes/` 只负责路由声明、参数解析、页面入口装配。
- `features/` 负责某个业务场景的页面实现和场景内组件。
- `components/` 只放跨 feature 共享的产品级组件。
- `components/ui/` 只放可复用 UI 基元，不掺杂业务请求和业务状态。
- `lib/` 负责客户端边界能力，尤其是 API、query key、请求封装。
- `stores/` 只放确实需要跨页面持久化或共享的状态。

### 3.2 路由层规则

以当前 TanStack Router 文件路由为准：

- route 文件保持薄，只做 `createFileRoute(...)`、参数校验、调用 feature 组件。
- route 文件不写大段页面 JSX，不堆积业务逻辑。
- 带布局的页面通过 `_app.tsx` 这类 layout route 统一注入壳层。

当前项目中的推荐模式：

- `src/routes/_app.library.$id.tsx` 负责读 `id`，再渲染 `#/features/library`
- `src/routes/_app.media.$id.tsx` 负责读 `id`，再渲染 `#/features/media`
- `src/routes/play.$id.tsx` 负责解析搜索参数，再渲染 `#/features/play`

新增页面时，优先遵守同样模式。

### 3.3 Feature 分层规则

每个 feature 应尽量形成以下结构：

```text
features/<feature>/
├── index.tsx              # feature 页面入口
├── components/            # feature 私有组件
├── hooks/                 # feature 私有 hook（必要时）
└── utils.ts or constants.ts
```

规则：

- `index.tsx` 作为该功能默认入口，供 route 直接引用。
- 只有该功能内部使用的组件，放在 feature 自己的 `components/`。
- 如果一个交互只服务于一个页面，不要上提到全局 `components/`。
- 只有当多个 feature 复用时，才提升为共享组件或共享 hook。

### 3.4 数据获取规则

前端数据访问统一收口：

- HTTP 类型和请求方法定义在 `web/src/lib/mibo-api.ts`
- Query key 和 `queryOptions` 定义在 `web/src/lib/mibo-query.ts`
- 页面中通过 `useQuery` / `useMutation` 组合 query options

规范要求：

- 不在 feature 组件里直接拼接 URL 和写 `fetch`。
- query key 必须稳定、显式包含关键参数，例如 `token`、`id`。
- 多页面复用的数据请求，优先抽到 `mibo-query.ts`。
- 页面只处理视图层 loading/error/empty 状态，不重复实现接口协议细节。

### 3.5 状态管理规则

前端状态分三类：

- 服务端状态：放 React Query
- 跨页面客户端状态：放 Zustand store
- 组件局部交互状态：放组件内部 state

当前已建立的典型模式：

- `stores/auth-store.ts` 保存 token、user、hydration 状态
- 媒体详情、首页、设置等服务端数据走 React Query

禁止：

- 用 Zustand 保存本应由服务端驱动的列表详情数据
- 用全局 store 替代 query cache
- 在多个组件里复制鉴权恢复逻辑

### 3.6 组件规则

组件按职责分三层：

- `components/ui/*`: 纯 UI 原子/基础复合组件
- `components/*`: 全局产品组件，如侧边栏、顶部栏、登录表单
- `features/*/components/*`: 业务页面私有组件

要求：

- UI 原子组件不直接请求接口。
- 共享产品组件可以依赖 store 或路由，但要避免绑定某个具体 feature 的私有语义。
- 页面型 feature 组件可以组合 query、mutation、导航和私有子组件。
- 优先通过 props 传值，避免无必要 context 扩散。

### 3.7 前端命名与文件风格

建议统一：

- 文件名使用 kebab-case，如 `app-top-bar.tsx`
- React 组件使用 PascalCase
- route 参数名与 URL 保持一致，如 `$id`
- query key 使用数组常量函数生成，不在页面里散写字符串

保持现有风格：

- TypeScript
- 路径别名使用 `#/*`
- 格式化交给 Prettier
- UI 基础设施沿用 shadcn/radix-nova 体系

## 4. 后端规范

### 4.1 目录职责

后端以 `cmd/ + internal/` 的 Go 服务结构为准。

```text
mibo-media-server/
├── cmd/mibo-media-server/  # 进程入口
└── internal/
    ├── app/                # 应用装配
    ├── httpapi/            # HTTP 路由与 handler
    ├── config/             # 配置加载
    ├── database/           # DB 打开与模型
    ├── auth/
    ├── library/
    ├── metadata/
    ├── playback/
    ├── progress/
    ├── settings/
    ├── jobs/
    ├── worker/
    ├── providers/
    └── storage/            # 存储抽象和适配器
```

规则：

- 可执行入口只能放在 `cmd/`。
- 业务实现只放在 `internal/`。
- 目录按领域拆分，不按技术碎片拆分出大量 `utils`、`common`、`helpers` 杂包。
- 新能力优先加到最接近的领域包，而不是创建泛化共享目录。

### 4.2 启动与装配规则

当前装配链路为：

- `cmd/mibo-media-server/main.go` 负责读取配置并启动应用
- `internal/app/app.go` 负责 database、service、router、worker 的装配

要求：

- `main.go` 保持极薄，只做启动编排。
- 依赖注入集中在 `app.New(...)`。
- 新 service 的构造应在 `internal/app/app.go` 明确接线，不做隐藏式全局单例。

### 4.3 HTTP API 层规则

`internal/httpapi/router.go` 是统一 API 边界。

这一层只负责：

- 注册路由
- 鉴权
- 解析请求
- 调用 service
- 将错误映射为 HTTP/JSON 响应

不应负责：

- 大段业务判断
- 跨多模型事务编排的核心流程
- 存储访问细节
- 后台任务核心实现

新增接口时遵守：

- 先判断该能力属于哪个 domain service
- handler 中只保留必要的输入校验和响应映射
- 业务错误尽量由 service 返回，handler 做状态码转换

### 4.4 Domain Service 规则

当前后端以 `Service` 作为领域能力主入口，例如：

- `auth.Service`
- `library.Service`
- `metadata.Service`
- `playback.Service`
- `progress.Service`
- `settings.Service`

规则：

- 一个领域包以一个 `Service` 为主入口是推荐模式。
- `service.go` 放公开能力与核心编排，拆分文件放子流程，例如 `scan.go`、`browse.go`、`query.go`。
- 对外暴露稳定输入输出结构，避免 handler 直接操作过多 DB 细节。
- 跨领域协作通过上层装配注入，不通过隐式 package 全局变量。

### 4.5 存储适配层规则

`internal/storage/provider.go` 定义稳定抽象，`internal/storage/local` 与 `internal/storage/openlist` 是具体实现。

核心原则：

- 业务层依赖 `storage.Provider`，不依赖 OpenList 细节。
- 新存储能力优先扩展 provider 抽象和 adapter，而不是把分支判断散落在业务层。
- provider 特有配置归口到 `internal/providers/` 做归一化和注册。

这条规则是本项目最重要的架构边界之一。

### 4.6 数据库模型规则

`internal/database/models.go` 统一维护 GORM 模型。

要求：

- 模型变更必须与对应 service 行为一起设计。
- 业务状态字段要显式，例如 `status`、`match_status`、`probe_status`。
- 不把 API 展示层临时字段塞进持久化模型。
- 复杂结构继续使用 JSON 字段时，要保证对应解析逻辑只在领域内扩散。

### 4.7 异步任务规则

当前后台任务体系是：

- `internal/jobs/service.go` 负责入队与任务状态管理
- `internal/worker/worker.go` 负责轮询和执行

要求：

- 扫描、匹配、探测等慢任务走 job queue，不在请求链路中直接执行到底。
- job kind 命名保持清晰稳定，如 `sync_library`、`match_media_item`。
- 可重试逻辑、失败状态和错误信息必须可观测。
- handler 只负责入队，不负责执行任务主体。

## 5. 跨层依赖规则

允许的依赖方向：

- `routes -> features -> lib/stores/components`
- `httpapi -> domain service -> database/storage/providers`
- `worker -> jobs + domain service`
- `app -> all concrete services for wiring`

不允许的依赖方向：

- `features` 反向依赖具体 `routes`
- `components/ui` 依赖业务 store 或业务 API
- `storage` 依赖 `httpapi`
- `database` 依赖具体业务 service
- `OpenList/` 反向承载 Mibo 业务补丁作为默认实现位置

## 6. 新代码放置规则

### 6.1 新前端页面

- 路由入口：`web/src/routes/`
- 页面实现：`web/src/features/<feature>/index.tsx`
- 私有子组件：`web/src/features/<feature>/components/`

### 6.2 新前端共享组件

- 业务共享组件：`web/src/components/`
- 通用 UI 原子：`web/src/components/ui/`
- 仅某个 feature 使用：不要上提，留在对应 feature 内

### 6.3 新前端接口调用

- API 类型与方法：`web/src/lib/mibo-api.ts`
- Query key 与 options：`web/src/lib/mibo-query.ts`

### 6.4 新后端业务能力

- 领域逻辑：`mibo-media-server/internal/<domain>/`
- HTTP 暴露：`mibo-media-server/internal/httpapi/router.go`
- 需要异步执行：补充 `jobs` 和 `worker` 派发

### 6.5 新存储接入

- 抽象补充：`internal/storage/provider.go`
- 具体实现：`internal/storage/<provider>/`
- 注册和配置归一化：`internal/providers/`

## 7. 代码演进原则

### 7.1 优先小改，避免平移式重构

- 优先在已有 domain/feature 内扩展。
- 只有当一个文件或模块已经明显承载多个职责时，再拆分。
- 不为了“看起来更架构化”而提前引入更多中间层。

### 7.2 抽象必须服务当前边界

可以抽象的前提：

- 已有至少两个真实使用点
- 抽象后能稳定约束边界
- 能减少重复，而不是隐藏复杂度

本项目当前最值得保护的抽象：

- 前端 `mibo-api` / `mibo-query` 边界
- 后端 `Service` 边界
- 后端 `StorageProvider` 边界
- 后端 `jobs + worker` 异步边界

### 7.3 文档与实现保持一致

- 如果路由体系、入口文件或核心目录发生变化，应同步更新本文档。
- 规划文档若与真实代码冲突，以真实代码结构为准，并回补文档。
- 不再使用的旧结构说明应及时删除，避免误导后续开发。

## 8. 反模式清单

以下做法应避免：

- 在 `OpenList/` 内实现 Mibo 新业务
- 在 route 文件中直接写完整业务页面
- 在组件中直接拼接后端 URL 发请求
- 在 `httpapi/router.go` 中堆积大段业务编排
- 创建没有明确领域边界的 `utils` / `common` / `shared` 大杂烩目录
- 用全局状态替代服务端查询缓存
- 为单一使用点过早抽象 hook、service、adapter
- 在请求链路中同步执行大扫描、探测、匹配任务

## 9. 当前项目的落地判断标准

一个改动如果满足以下条件，就可以认为符合本项目结构规范：

- 新代码放在了正确的包或 feature 下
- 没有打破前后端、存储、异步任务的既有边界
- route、feature、query、API、service 的职责分工清楚
- 新依赖方向与本文档一致
- 没有因为局部功能而污染共享层或上游边界

如果不满足，优先调整代码落点和依赖方向，而不是继续追加说明性注释掩盖结构问题。
