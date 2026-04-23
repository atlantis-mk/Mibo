# Requirements: Mibo

**Defined:** 2026-04-23
**Core Value:** 无论底层媒体文件来自本地磁盘、NAS 还是云盘，用户都能稳定地完成媒体库接入、内容浏览、播放和进度同步。

## v2 Requirements

### Search

- [ ] **SRCH-01**: 用户可以按标题搜索媒体内容
- [ ] **SRCH-02**: 用户可以按原始标题搜索媒体内容
- [ ] **SRCH-03**: 用户可以按演员搜索媒体内容
- [ ] **SRCH-04**: 用户可以按导演搜索媒体内容
- [ ] **SRCH-05**: 搜索结果会明确区分电影和剧集
- [ ] **SRCH-06**: 搜索结果会高亮命中的关键词
- [ ] **SRCH-07**: 用户可以对搜索结果进行排序
- [ ] **SRCH-08**: 系统会为用户保留最近搜索历史

### Filters

- [ ] **FLTR-01**: 用户可以按类型筛选媒体内容
- [ ] **FLTR-02**: 用户可以按年份筛选媒体内容
- [ ] **FLTR-03**: 用户可以按地区筛选媒体内容
- [ ] **FLTR-04**: 用户可以按评分筛选媒体内容
- [ ] **FLTR-05**: 用户可以按已看 / 未看状态筛选媒体内容
- [ ] **FLTR-06**: 用户可以在搜索和浏览结果中使用统一的排序能力

### Trailers

- [ ] **TRLR-01**: 系统可以从 TMDB 同步媒体条目的预告片元数据
- [ ] **TRLR-02**: 当存在可用预告片时，媒体详情页会展示“观看预告片”入口
- [ ] **TRLR-03**: 用户可以在媒体详情页内直接播放预告片
- [ ] **TRLR-04**: 当没有可用预告片时，系统会优雅隐藏预告片入口

### Metadata Management

- [ ] **META-01**: 管理员可以编辑标题、原始标题、年份和简介
- [ ] **META-02**: 管理员可以编辑海报和背景图
- [ ] **META-03**: 管理员可以编辑分类和演员等人物信息
- [ ] **META-04**: 管理员可以编辑剧集的季集基础信息
- [ ] **META-05**: 管理员可以对媒体条目发起重新匹配
- [ ] **META-06**: 管理员可以对媒体条目发起元数据重抓

### Scan Listeners

- [ ] **LIST-01**: 系统可以监听存储中的新增、更新、删除和移动类变更
- [ ] **LIST-02**: 监听到的存储变更会被归一为 targeted refresh 任务
- [ ] **LIST-03**: 系统会对突发存储事件进行去抖或合并，避免重复刷新
- [ ] **LIST-04**: 系统保留兜底 reconciliation / 对账机制，防止监听漏事件导致状态漂移

### Scheduled Jobs

- [ ] **SJOB-01**: 管理员可以创建和管理扫描类计划任务
- [ ] **SJOB-02**: 管理员可以创建和管理元数据重抓类计划任务
- [ ] **SJOB-03**: 管理员可以创建和管理预告片同步类计划任务
- [ ] **SJOB-04**: 管理员可以创建和管理库清理类计划任务
- [ ] **SJOB-05**: 管理员可以创建和管理失效链接检查类计划任务
- [ ] **SJOB-06**: 管理员可以创建和管理封面刷新类计划任务
- [ ] **SJOB-07**: 管理员可以启停计划任务、手动执行任务，并查看下次运行时间
- [ ] **SJOB-08**: 管理员可以查看计划任务最近运行结果和历史记录

## Future Requirements

### Filters

- **FLTR-07**: 用户可以按媒体库筛选媒体内容
- **FLTR-08**: 用户可以按分辨率筛选媒体内容

### Trailers

- **TRLR-05**: 系统可以从 TMDB 以外的外部源补充预告片链接

### Metadata Management

- **META-07**: 管理员可以锁定指定元数据字段，防止后续刷新覆盖

### Scan Listeners

- **LIST-05**: 管理员可以查看监听器启停状态与运行健康度

## Out of Scope

| Feature | Reason |
|---------|--------|
| 外部搜索中间件（如 Elasticsearch / Meilisearch） | 本 milestone 明确不引入外部中间件 |
| 语义 / 向量搜索 | 先把确定性搜索和筛选做好 |
| 预告片下载、代理或转码 | v2 只管理预告片引用与播放入口 |
| 大规模批量元数据编辑 | 先把单条目治理能力做好 |
| 高级查询语言或复杂布尔筛选器 | 先交付稳定、直接的产品体验 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| SRCH-01 | Unmapped | Pending |
| SRCH-02 | Unmapped | Pending |
| SRCH-03 | Unmapped | Pending |
| SRCH-04 | Unmapped | Pending |
| SRCH-05 | Unmapped | Pending |
| SRCH-06 | Unmapped | Pending |
| SRCH-07 | Unmapped | Pending |
| SRCH-08 | Unmapped | Pending |
| FLTR-01 | Unmapped | Pending |
| FLTR-02 | Unmapped | Pending |
| FLTR-03 | Unmapped | Pending |
| FLTR-04 | Unmapped | Pending |
| FLTR-05 | Unmapped | Pending |
| FLTR-06 | Unmapped | Pending |
| TRLR-01 | Unmapped | Pending |
| TRLR-02 | Unmapped | Pending |
| TRLR-03 | Unmapped | Pending |
| TRLR-04 | Unmapped | Pending |
| META-01 | Unmapped | Pending |
| META-02 | Unmapped | Pending |
| META-03 | Unmapped | Pending |
| META-04 | Unmapped | Pending |
| META-05 | Unmapped | Pending |
| META-06 | Unmapped | Pending |
| LIST-01 | Unmapped | Pending |
| LIST-02 | Unmapped | Pending |
| LIST-03 | Unmapped | Pending |
| LIST-04 | Unmapped | Pending |
| SJOB-01 | Unmapped | Pending |
| SJOB-02 | Unmapped | Pending |
| SJOB-03 | Unmapped | Pending |
| SJOB-04 | Unmapped | Pending |
| SJOB-05 | Unmapped | Pending |
| SJOB-06 | Unmapped | Pending |
| SJOB-07 | Unmapped | Pending |
| SJOB-08 | Unmapped | Pending |

**Coverage:**
- v2 requirements: 36 total
- Mapped to phases: 0
- Unmapped: 36 ⚠

---
*Requirements defined: 2026-04-23*
*Last updated: 2026-04-23 after initial milestone v2 definition*
