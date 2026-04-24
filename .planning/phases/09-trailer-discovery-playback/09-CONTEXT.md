# Phase 9: Trailer Discovery & Playback - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning

<domain>
## Phase Boundary

用户可以在 Mibo 的媒体详情体验内发现并直接播放可用预告片：系统从 TMDB 同步 trailer 元数据，在详情页中只在存在可用 trailer 时展示入口，并允许用户在不离开当前详情上下文的前提下播放该 trailer。

本阶段聚焦“TMDB trailer 元数据进入 Mibo 数据面 + 详情页内预告片入口 + 详情内播放承载”，不扩展为外部源聚合，不引入 trailer 下载、代理或转码，也不把未来定时刷新/调度能力提前并入当前范围。

</domain>

<decisions>
## Implementation Decisions

### 入口位置与详情承载
- **D-01:** 预告片正式入口不放在 Hero 主操作区，也不与正片播放按钮并列竞争首屏主 CTA。
- **D-02:** 预告片入口应放在详情内容区，并优先归入详情区/附加信息区，而不是海报区或简介区。
- **D-03:** 现有 `standalone-media-detail-hero.tsx` 中的 `预告片` 占位胶囊按钮不应继续作为正式入口落点。

### 播放体验
- **D-04:** 点击详情区入口后，预告片必须在当前详情页内以弹层播放器播放。
- **D-05:** 关闭弹层或播放结束后，用户应回到原详情上下文，不切换到独立 trailer 播放路由。
- **D-06:** Phase 9 不采用外部页面跳转，也不把预告片提升成与正片播放同等级的独立播放体验。

### 数据准备与契约
- **D-07:** trailer 视为 TMDB 元数据的一部分，应随现有 metadata match/refetch 链路同步并持久化到 Mibo 自有数据面。
- **D-08:** 详情接口应直接返回当前已知的 trailer 可用性与最终播放项，不依赖详情页现场触发 TMDB 拉取。
- **D-09:** 当没有可用 trailer 时，详情页直接隐藏入口；Phase 9 不引入“现场加载 trailer”“稍后可用”的额外详情态。

### 选片规则
- **D-10:** 当 TMDB 返回多支视频时，优先级为：`official Trailer`，其次 `non-official Trailer`，最后允许回退到 `Teaser`。
- **D-11:** 同一优先级层内不引入复杂语言排序；直接选择最靠前的可播放项作为详情页单一播放目标。
- **D-12:** 详情页应消费一个已经选定的最终 trailer 结果，而不是把多个候选直接暴露给前端做二次决策。

### 范围边界
- **D-13:** Phase 9 只覆盖 TMDB 作为 trailer 来源；未来需求 `TRLR-05` 的外部源补充不提前并入当前阶段。
- **D-14:** Phase 9 不包含 trailer 下载、本地缓存、代理转发、转码或离线保存能力。
- **D-15:** Phase 9 不新增独立 trailer 调度体系；任何后续定时刷新应叠加在既有 jobs/worker 模式上，并留待后续阶段处理。

### the agent's Discretion
- 详情区入口在 `SpecsSection` 中的具体呈现形式可由后续 agent 决定，例如信息行、按钮行或小型卡片，但必须保持“详情区附加信息入口”这一锁定方向。
- 弹层播放器的视觉样式、尺寸、遮罩、关闭交互和移动端适配细节可由后续规划决定，前提是用户不离开当前详情页上下文。
- trailer 的具体持久化建模可由后续 research / planning 决定，例如扩展 `media_items` 字段或引入 app-owned trailer cache/table；但必须由 `mibo-media-server` 持有，不能退回 request-time 临时抓取。
- 详情接口是扩展现有 `GET /api/v1/media-items/{id}` 还是补充一个紧邻详情语义的 trailer 字段组织方式，可由后续规划决定；但前端不得依赖 ad hoc 现场请求 TMDB。

</decisions>

<specifics>
## Specific Ideas

- 当前详情页已经分成 Hero、Cast、Specs 三段，且 `SpecsSection` 本身就是“其它信息”容器；因此预告片入口落在详情区是符合现有页面信息架构的，而不是另造一块平行内容。
- 现有 Hero 操作行里的 `预告片` 按钮更像视觉占位，不应被误当成已锁定产品方向；本阶段已经明确正式入口应后移到详情区。
- 由于 trailer 随 metadata refetch 持久化，详情页更适合消费一个简单、稳定、typed 的 trailer 结果，而不是再引入第二条“打开页面后现场拉 TMDB”的前端状态流。
- 选片规则应在后端收敛为单一最终结果，这样 `TRLR-02` 的“是否显示入口”与 `TRLR-03` 的“点击后播放什么”可以共享同一判断来源。

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase definition
- `.planning/ROADMAP.md` — Phase 9 的目标、依赖与 success criteria，明确是“详情页内发现并播放预告片”。
- `.planning/REQUIREMENTS.md` — `TRLR-01` through `TRLR-04` 的锁定需求，以及 `TRLR-05` 仍属未来范围。
- `.planning/PROJECT.md` — v2 对 trailer capability 的产品定位，以及 `OpenList -> mibo-media-server -> JSON API` 的边界约束。
- `.planning/STATE.md` — 当前阶段状态，以及 trailers 继续建立在 metadata governance / jobs-worker 模式上的累计上下文。
- `.planning/phases/07-metadata-governance-matching/07-CONTEXT.md` — metadata ownership、match/refetch 走后台任务、app-owned metadata 的先决约束。
- `.planning/phases/08-native-search-discovery-filters/08-CONTEXT.md` — 当前 frontend live package、typed API client + TanStack Query 模式、以及 discovery contract 统一的最近先例。

### Live backend anchors
- `mibo-media-server/internal/metadata/service.go` — 元数据服务的主入口，Phase 9 trailer 同步应附着在既有 metadata 语义之内。
- `mibo-media-server/internal/metadata/service_tmdb.go` — 当前 TMDB detail 请求只拉 `credits,images`；Phase 9 trailer 接入的直接扩展点在这里。
- `mibo-media-server/internal/library/enrichment.go` — `QueueMediaItemMatch` / `QueueMediaItemMetadataRefetch` 已定义现有 metadata job enqueue 模式。
- `mibo-media-server/internal/worker/worker.go` — worker 当前消费 `match_media_item` / `refetch_media_item` 等 job kind，是 trailer 同步复用后台链路的执行锚点。
- `mibo-media-server/internal/database/models.go` — 当前 `MediaItem` 没有任何 trailer 字段或 trailer relation，说明 Phase 9 需要新增 app-owned trailer 持久化建模。
- `mibo-media-server/internal/library/query.go` — `library.MediaItemDetail` 当前没有 trailer 字段，是详情 contract 需要扩展的直接位置。
- `mibo-media-server/internal/httpapi/router.go` — 当前只有 `GET /api/v1/media-items/{id}` 与 metadata/playback 路由，没有 trailer 专属 contract。

### Live frontend anchors
- `web/src/features/media/index.tsx` — 当前详情页 feature 使用 React Query 拉 detail/progress，并把正片播放跳到独立 `/play/$id`；Phase 9 需要在这里接入 trailer detail contract 与弹层状态。
- `web/src/features/media/components/standalone-media-detail.tsx` — 当前详情页组合 `DetailHeroSection`、`CastSection`、`SpecsSection`，是把 trailer 入口插入详情区的页面骨架。
- `web/src/features/media/components/standalone-media-detail-hero.tsx` — 当前 Hero 操作行存在未接线的 `预告片` 占位按钮，也是本阶段明确“不再作为正式入口”的反向锚点。
- `web/src/features/media/components/standalone-media-detail-specs.tsx` — 当前“其它信息”详情区，是 Phase 9 已锁定 trailer 入口的首要承载点。
- `web/src/lib/mibo-api.ts` — 当前 `MediaItemDetail` 类型没有 trailer 字段，且尚无 trailer client contract。
- `web/src/lib/mibo-query.ts` — 当前详情页 query key / typed query 模式的扩展锚点，Phase 9 应沿用而不是退回 ad hoc fetch。

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/features/media/components/standalone-media-detail.tsx`: 已有详情页三段式布局，`SpecsSection` 可自然承接 trailer 入口，不必新造一个平行详情容器。
- `web/src/features/media/components/standalone-media-detail-specs.tsx`: 当前“其它信息”区域已经是 detailGroups + info cards 结构，适合接一个附加资料性质的 trailer 入口。
- `web/src/lib/mibo-api.ts`: 现有 typed API client 已经承载 detail/progress/discovery 等契约，Phase 9 应继续扩展 typed detail shape。
- `mibo-media-server/internal/metadata/service_tmdb.go`: 已有 TMDB request、认证、错误处理、语言参数与 `append_to_response` 模式，可直接复用到 trailer 数据抓取。
- `mibo-media-server/internal/library/enrichment.go` + `internal/worker/worker.go`: 现有 metadata match/refetch 已形成 enqueue -> worker -> metadata service 的后台链路，可承接 trailer 同步。

### Established Patterns
- 现有前端详情页数据流依赖 typed client + TanStack Query；Phase 9 不应为 trailer 单独引入页面内 ad hoc 网络流。
- 现有后端把 metadata 重抓与匹配都放进 job/worker，而不是在用户请求现场做重工作；trailer 应延续这一模式。
- 详情体验目前把正片播放视为独立播放路由，而把详情内容本身作为浏览上下文；本阶段已明确 trailer 不复用正片路由，而是在详情内弹层播放。
- app-owned metadata ownership 已在前序阶段锁定，说明 trailer 的最终可播结果应由 Mibo 持久化并对外提供，而不是把 TMDB 结果直接透传给前端自行裁决。

### Integration Points
- 后端需要把 TMDB `videos` 拉取、trailer 选片、持久化与现有 `RefetchItem` / `MatchItem` 语义打通。
- 后端需要扩展 media detail contract，让详情页能一次拿到“是否有 trailer”和“最终播放目标”的信息。
- 前端需要在 `MediaDetail` feature 中增加 trailer 弹层状态与入口渲染，但保持现有 detail query / mutation 模式不变。
- 前端需要把现有 Hero 里的 trailer 占位按钮从正式入口路径中移除或降级，避免与已锁定的详情区入口冲突。

### Constraints
- 当前 `database.MediaItem` 与 `library.MediaItemDetail` 都没有 trailer 字段，Phase 9 不是纯前端接线任务，必须覆盖后端数据建模与 API contract。
- 当前 `GET /api/v1/media-items/{id}` 没有 trailer 信息，且 router 中也没有 trailer 专属路由；下游规划不能假设现成接口存在。
- 当前 `service_tmdb.go` 只请求 `credits,images`，说明 trailer 不是已有 metadata fetch 的 incidental output，而是需要显式加入的能力。
- 当前详情页首屏的预告片胶囊按钮只是静态占位；Phase 9 已锁定正式入口迁移到详情区，因此不能简单把这个按钮接上线就算完成需求。

</code_context>

<deferred>
## Deferred Ideas

- `TRLR-05`：从 TMDB 以外的外部源补充 trailer 链接。
- trailer 下载、本地缓存、代理转发、转码或离线保存。
- 多 trailer 候选选择器、用户手动切换不同版本 trailer。
- 计划任务驱动的 trailer 周期刷新或独立 trailer scheduler。

</deferred>

---

*Phase: 09-trailer-discovery-playback*
*Context gathered: 2026-04-24*
