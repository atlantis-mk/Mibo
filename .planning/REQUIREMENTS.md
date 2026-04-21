# Requirements: Mibo

**Defined:** 2026-04-21
**Core Value:** 无论底层媒体文件来自本地磁盘、NAS 还是云盘，用户都能稳定地完成媒体库接入、内容浏览、播放和进度同步。

## v1 Requirements

Requirements for the current milestone. Each maps to exactly one roadmap phase.

### Access

- [x] **ACCS-01
**: 管理员可以完成初始化配置并进入可用的媒体系统主流程
- [x] **ACCS-02
**: 用户可以登录并在后续请求中保持已认证会话
- [x] **ACCS-03
**: 用户可以通过稳定的 HTTP API 访问同一套媒体能力，供 Web 现在使用并为移动端、TV 端预留兼容性

### Libraries

- [ ] **LIBR-01**: 管理员可以添加基于本地磁盘、NAS 或云盘接入的媒体源
- [ ] **LIBR-02**: 管理员可以创建媒体库并将其绑定到指定媒体源和根路径
- [ ] **LIBR-03**: 管理员可以手动触发媒体库扫描并看到任务被异步处理
- [ ] **LIBR-04**: 管理员可以为媒体库配置定时刷新，使新增内容能够按计划进入系统

### Catalog

- [x] **CATA-01
**: 系统可以通过 `StorageProvider` 统一读取 OpenList 提供的文件访问能力，而不把 OpenList 细节暴露给业务 API
- [ ] **CATA-02**: 系统可以在扫描后把媒体文件写入可追踪的 `media_files` 目录索引
- [ ] **CATA-03**: 系统可以把识别出的内容组织为稳定的电影、剧集、季、集语义结构
- [ ] **CATA-04**: 用户可以看到带海报、简介和基础详情的媒体条目，而不是仅看到原始文件名
- [ ] **CATA-05**: 用户可以按媒体库浏览、筛选、搜索并进入媒体详情页
- [ ] **CATA-06**: 系统可以把扫描、识别、元数据匹配和 `ffprobe` 补全拆分为可重试的后台任务，而不是在单次请求中完成

### Playback

- [ ] **PLAY-01**: 用户可以从媒体详情页请求播放，并获得对当前客户端可用的播放入口
- [ ] **PLAY-02**: 系统在可行时优先提供直链播放，在不可行时提供明确的回退路径
- [ ] **PLAY-03**: 系统可以利用 `ffprobe` 产生的媒体信息提升播放决策质量

### Progress

- [ ] **PROG-01**: 用户在播放过程中上报的进度会被持久化，并在下次继续观看时可恢复
- [ ] **PROG-02**: 同一用户的播放进度可以通过统一 API 在 Web、移动端和 TV 端之间同步

### Sync

- [ ] **SYNC-01**: 系统具备稳定文件身份能力，使重命名、移动或重挂载不会轻易造成重复媒体或进度丢失
- [ ] **SYNC-02**: 系统在全量扫描之外支持增量刷新能力，以减少日常更新成本
- [ ] **SYNC-03**: 系统可以接收存储变更事件并把它们转换为安全的增量扫描或重同步任务

## v2 Requirements

Deferred to a later milestone. Tracked, but not required in the current roadmap.

### Experience

- **EXPR-01**: 用户可以在首页看到稳定可用的 Continue Watching 入口
- **EXPR-02**: 用户可以在首页看到 Recently Added 等家庭媒体首页推荐面板

### Access

- **ACCS-04**: 家庭成员可以拥有更细的账户隔离或家庭级访问限制
- **ACCS-05**: 系统提供更完整的远程访问和外网部署体验优化

### Playback

- **PLAY-04**: 系统可以在更多设备场景下提供更成熟的 HLS 或转码能力

## Out of Scope

Explicit exclusions for this project initialization scope.

| Feature | Reason |
|---------|--------|
| 深度 fork 或把媒体业务深度嵌入 OpenList | 违背当前架构边界，长期维护成本过高 |
| 从第一阶段开始自研完整存储协议栈 | 现阶段复用 OpenList 更符合交付速度目标 |
| 早期微服务化拆分 | 当前阶段优先简单部署和清晰模块边界 |
| Live TV / DVR | 属于独立产品方向，会显著拉大范围 |
| 音乐、照片、书籍等广泛媒体类型扩展 | 当前阶段优先聚焦视频媒体语义 |
| Watch-party / 社交功能 | 不是家庭媒体系统当前里程碑的核心价值 |
| 永久在线的激进预转码流水线 | 复杂度和成本过高，现阶段应优先直链与按需回退 |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| ACCS-01 | Phase 1 | Pending |
| ACCS-02 | Phase 1 | Pending |
| ACCS-03 | Phase 1 | Pending |
| LIBR-01 | Phase 2 | Pending |
| LIBR-02 | Phase 2 | Pending |
| LIBR-03 | Phase 2 | Pending |
| LIBR-04 | Phase 2 | Pending |
| CATA-01 | Phase 1 | Pending |
| CATA-02 | Phase 3 | Pending |
| CATA-03 | Phase 3 | Pending |
| CATA-04 | Phase 3 | Pending |
| CATA-05 | Phase 3 | Pending |
| CATA-06 | Phase 2 | Pending |
| PLAY-01 | Phase 4 | Pending |
| PLAY-02 | Phase 5 | Pending |
| PLAY-03 | Phase 5 | Pending |
| PROG-01 | Phase 4 | Pending |
| PROG-02 | Phase 4 | Pending |
| SYNC-01 | Phase 6 | Pending |
| SYNC-02 | Phase 6 | Pending |
| SYNC-03 | Phase 6 | Pending |

**Coverage:**
- v1 requirements: 20 total
- Mapped to phases: 20
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-21*
*Last updated: 2026-04-21 after roadmap creation*
