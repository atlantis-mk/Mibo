# Backend Module Boundaries

本文档定义 `mibo-media-server/internal` 当前阶段的模块边界、分层、依赖方向和业务规则归属。目标不是立即大搬家，而是给后续架构收敛提供稳定的判断标准。

## 1. Target Shape

后端收敛后的目标主干是：

```text
httpapi
  -> application modules
       -> domain modules
            -> adapters / infrastructure
```

在当前代码里，对应关系是：

- `httpapi`: API adapter
- `library`, `metadata`, `catalog`, `playback`, `listener`, `health`: application + domain 混合模块，后续逐步收敛
- `database`, `providers`, `storage`, `workflow`, `ingest`, `search`, `settings`: infrastructure 或跨域支撑模块
- `app`: composition root

当前阶段不追求先拆 package，而是先把职责和依赖方向固定下来。

## 2. Module Boundaries

### httpapi

职责：

- 请求解析
- 鉴权
- HTTP 错误映射
- 响应序列化
- 组合多个 application module 完成一个接口

禁止：

- 持有业务规则真相
- 自己判断扫描策略/治理策略/投影可见性规则
- 直接操作数据库模型形成核心流程

判断标准：如果一段逻辑离开 HTTP 仍然成立，那它不应该留在 `httpapi`。

### library

职责：

- `Media Source` 与 `Library` 生命周期
- `Library Policy` 和 `Effective Library Config`
- 扫描、增量刷新、路径解析、排除规则
- materialization 编排
- 与 `workflow`、`ingest`、`providers/storage` 协同

这个模块是当前系统的媒体入口编排模块。

它拥有的核心规则：

- 一个库如何被扫描
- 一个库哪些路径生效
- 一个文件何时进入 inventory
- 哪些排除规则生效
- materialize 如何推进后续探测、匹配、投影刷新

禁止继续扩张为：

- 首页/详情读取逻辑中心
- 元数据人工治理中心
- 播放来源中心

### metadata

职责：

- `Metadata Item` 语义规则
- 外部 metadata provider 整合
- 匹配、字段应用、人工治理
- 资源与元数据的关系修正
- 治理操作审计

这个模块拥有的核心规则：

- 什么叫匹配
- 人工字段覆盖如何落地
- 资源链接/解绑/合并/拆分如何影响语义真相
- 治理操作如何记录与回放

禁止：

- 直接承接 HTTP 形状
- 承担首页/浏览读模型拼装
- 吞并扫描/库策略规则

### catalog

职责：

- `Projection` 读取
- 首页、最近添加、按库最新、浏览、详情、人物详情
- 读模型排序、筛选、展示聚合

这个模块是产品读取模型模块，不是语义真相模块。

它拥有的核心规则：

- 产品读取场景看见什么
- 以什么顺序看见
- 如何把语义实体和用户状态组织成响应

禁止：

- 主导治理写入
- 决定扫描行为
- 引入播放生成细节

### playback

职责：

- 解析 `Metadata Item` 或 `Inventory File` 的可播放来源
- 根据 client profile 选择可返回的播放信息

禁止：

- 拥有 catalog browse/home 规则
- 参与扫描和治理编排

### listener

职责：

- 接收存储变化
- 合并变化窗口
- 决定 targeted refresh 还是 full sync 意图
- 把刷新意图交给 `library`

这个模块不拥有扫描实现，只拥有“何时刷新、刷新到什么范围”的规则。

### health

职责：

- 可运营性诊断
- 系统问题汇总
- 引导 rescan/ignore 等运维动作

`health` 可以触发其他模块动作，但不应复制这些模块的业务规则。

## 3. Layering

当前建议的逻辑分层如下：

### API Adapter Layer

- `internal/httpapi`

### Application Layer

- `internal/library`
- `internal/metadata`
- `internal/catalog`
- `internal/playback`
- `internal/listener`
- `internal/health`

这里的 `Application Layer` 负责：

- use case 编排
- 事务边界
- 跨 domain/infrastructure 协作
- 将领域规则组合为外部可调用能力

### Domain Model Layer

当前尚未独立成单独 package，但领域模型已经存在，后续应继续显式化：

- `Library`
- `Library Policy`
- `Effective Library Config`
- `Inventory File`
- `Resource`
- `Metadata Item`
- `Governance Operation`
- `Projection`
- `Playback Source`

要求：业务规则优先围绕这些模型组织，而不是围绕 handler、SQL 片段、任务种类字符串组织。

### Infrastructure Layer

- `database`
- `providers`
- `storage`
- `workflow`
- `ingest`
- `search`
- `settings`

这些模块提供适配和支撑能力，不拥有产品业务真相。

## 4. Dependency Flow

允许的单向依赖：

```text
httpapi -> library|catalog|metadata|playback|listener|health
listener -> library
health -> library|catalog|providers
playback -> catalog|providers|database
catalog -> database|ingest
metadata -> database|inventory|search|settings|ingest
library -> database|providers|workflow|ingest|settings
```

重点禁止：

- `catalog -> httpapi`
- `metadata -> httpapi`
- `library -> httpapi`
- `storage/providers -> library/httpapi`
- `database -> domain modules`
- 在 `httpapi` 中直接组合底层 `database` 形成业务规则

收敛判断标准：依赖必须从“接口适配层”流向“应用层/领域层”，再流向“基础设施层”，不能反向回流。

## 5. Business Rule Ownership

业务规则必须集中，不再散落在多个模块重复出现。

### 集中在 library 的规则

- 扫描是否允许
- realtime monitor 是否开启
- schedule refresh 是否开启
- 文件/路径排除是否命中
- targeted refresh 范围如何裁剪
- materialize 后续任务如何排队

优先承载模型：

- `EffectiveLibraryConfig`
- `LibraryScanPolicy`
- `LibraryPath`

### 集中在 metadata 的规则

- metadata matchability
- 字段人工覆盖与锁定
- 资源链接/解绑/合并/拆分
- 治理操作证据记录
- 投影可见性治理

优先承载模型：

- `MetadataOperationResult`
- `Governance Operation`
- `Metadata Item`
- `ResourceMetadataLink`

### 集中在 catalog 的规则

- 首页 feed 选择
- recently added/latest by library/browse 排序与筛选
- 详情读取聚合
- 人物详情读模型刷新时机

优先承载模型：

- `CatalogListItem`
- `CatalogLatestByLibrarySection`
- `Projection`

### 集中在 playback 的规则

- 播放来源选择
- client profile 对来源输出的影响

## 6. Current Refactor Priorities

基于当前代码，下一阶段最值得继续收敛的是：

### 1. 收窄 library.Service

现状：

- `library.Service` 仍然直接持有 `db`、`storage`、`workflow`、`ingest`、executor 等多种能力
- 规则和基础设施访问在很多文件里交织

收敛方向：

- 继续统一通过 capability helpers 访问依赖
- 再往后把扫描、materialize、policy、source 管理缩成更深的内部 module

### 2. 把 Effective Library Config 变成真正的规则入口

现状：

- 很多扫描/刷新判断已经依赖它
- 但仍有规则散落在其他流程函数中

收敛方向：

- 所有“库现在该怎么运行”的判断尽量从它出发
- 避免 handler、listener、worker 重复读取 policy 细节

### 3. 把 metadata 治理模型继续显式化

现状：

- `metadata/service_governance.go` 已经集中了一大批治理 use case
- 但很多治理语义仍与 DB 操作和 evidence 记录混在一起

收敛方向：

- 继续把治理操作统一围绕 `Governance Operation` 和 `MetadataOperationResult` 收口
- 让治理规则先成模型，再决定如何持久化

### 4. 明确 catalog 只读角色

现状：

- `catalog` 主要承担读取和投影聚合，方向正确
- 需要继续避免把写规则和治理规则拉回 `catalog`

收敛方向：

- 保持 `catalog` 以读模型为中心
- 对外只暴露“产品读取场景”接口

## 7. Immediate Coding Rules

从现在开始，新增或重构代码应遵守：

1. 新业务判断先问“这条规则属于 library、metadata、catalog、playback 里的哪一个模块”。
2. 如果逻辑离开 HTTP 仍成立，不能留在 `httpapi`。
3. 如果逻辑决定“库怎么运行”，优先进 `library`，并围绕 `EffectiveLibraryConfig` 组织。
4. 如果逻辑决定“元数据语义是否成立”，优先进 `metadata`。
5. 如果逻辑只是“产品怎么读出来”，优先进 `catalog`。
6. 新依赖必须沿单向流动增加，不能从基础设施层反向拉业务层。
7. 新模型命名优先复用 `CONTEXT.md` 词汇，不再引入近义新名。
