# Requirements Archive: Mibo v1

**Archived:** 2026-04-22
**Milestone:** v1
**Outcome:** ✅ All v1 requirements shipped

## v1 Requirements

### Access

- [x] **ACCS-01**: 管理员可以完成初始化配置并进入可用的媒体系统主流程
- [x] **ACCS-02**: 用户可以登录并在后续请求中保持已认证会话
- [x] **ACCS-03**: 用户可以通过稳定的 HTTP API 访问同一套媒体能力，供 Web 现在使用并为移动端、TV 端预留兼容性

### Libraries

- [x] **LIBR-01**: 管理员可以添加基于本地磁盘、NAS 或云盘接入的媒体源
- [x] **LIBR-02**: 管理员可以创建媒体库并将其绑定到指定媒体源和根路径
- [x] **LIBR-03**: 管理员可以手动触发媒体库扫描并看到任务被异步处理
- [x] **LIBR-04**: 管理员可以为媒体库配置定时刷新，使新增内容能够按计划进入系统

### Catalog

- [x] **CATA-01**: 系统可以通过 `StorageProvider` 统一读取 OpenList 提供的文件访问能力，而不把 OpenList 细节暴露给业务 API
- [x] **CATA-02**: 系统可以在扫描后把媒体文件写入可追踪的 `media_files` 目录索引
- [x] **CATA-03**: 系统可以把识别出的内容组织为稳定的电影、剧集、季、集语义结构
- [x] **CATA-04**: 用户可以看到带海报、简介和基础详情的媒体条目，而不是仅看到原始文件名
- [x] **CATA-05**: 用户可以按媒体库浏览、筛选、搜索并进入媒体详情页
- [x] **CATA-06**: 系统可以把扫描、识别、元数据匹配和 `ffprobe` 补全拆分为可重试的后台任务，而不是在单次请求中完成

### Playback

- [x] **PLAY-01**: 用户可以从媒体详情页请求播放，并获得对当前客户端可用的播放入口
- [x] **PLAY-02**: 系统在可行时优先提供直链播放，在不可行时提供明确的回退路径
- [x] **PLAY-03**: 系统可以利用 `ffprobe` 产生的媒体信息提升播放决策质量

### Progress

- [x] **PROG-01**: 用户在播放过程中上报的进度会被持久化，并在下次继续观看时可恢复
- [x] **PROG-02**: 同一用户的播放进度可以通过统一 API 在 Web、移动端和 TV 端之间同步

### Sync

- [x] **SYNC-01**: 系统具备稳定文件身份能力，使重命名、移动或重挂载不会轻易造成重复媒体或进度丢失
- [x] **SYNC-02**: 系统在全量扫描之外支持增量刷新能力，以减少日常更新成本
- [x] **SYNC-03**: 系统可以接收存储变更事件并把它们转换为安全的增量扫描或重同步任务

## Requirement Outcomes

| Requirement | Phase | Outcome |
|-------------|-------|---------|
| ACCS-01 | Phase 1 | validated |
| ACCS-02 | Phase 1 | validated |
| ACCS-03 | Phase 1 | validated |
| LIBR-01 | Phase 2 | validated |
| LIBR-02 | Phase 2 | validated |
| LIBR-03 | Phase 2 | validated |
| LIBR-04 | Phase 2 | validated |
| CATA-01 | Phase 1 | validated |
| CATA-02 | Phase 3 | validated |
| CATA-03 | Phase 3 | validated |
| CATA-04 | Phase 3 | validated |
| CATA-05 | Phase 3 | validated |
| CATA-06 | Phase 2 | validated |
| PLAY-01 | Phase 4 | validated |
| PLAY-02 | Phase 5 | validated |
| PLAY-03 | Phase 5 | validated |
| PROG-01 | Phase 4 | validated |
| PROG-02 | Phase 4 | validated |
| SYNC-01 | Phase 6 | validated |
| SYNC-02 | Phase 6 | validated |
| SYNC-03 | Phase 6 | validated |

## Notes

- v1 需求未出现 dropped 项。
- closeout 过程中修正了 `PLAY-*` 与 `SYNC-*` 的文档漂移，归档文件已反映最终真实状态。
- 下一里程碑的需求将重新定义，不继续沿用本文件。

## Deferred Beyond v1

- `EXPR-01`: 首页稳定 Continue Watching 体验增强
- `EXPR-02`: Recently Added 等家庭首页推荐面板增强
- `ACCS-04`: 更细的家庭成员隔离/访问限制
- `ACCS-05`: 远程访问和外网部署体验优化
- `PLAY-04`: 更成熟的 HLS / 转码能力
