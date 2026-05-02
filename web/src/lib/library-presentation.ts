export function formatLibraryType(type: string | null | undefined) {
  switch ((type ?? "").trim().toLowerCase()) {
    case "auto":
    case "source":
    case "source-first":
    case "source_first":
      return "自动分类"
    case "movies":
    case "movie":
    case "films":
      return "电影库"
    case "shows":
    case "tv":
    case "tvshows":
      return "剧集库"
    case "mixed":
    case "mixed-content":
    case "mixed_content":
      return "混合内容"
    default:
      return type || "媒体库"
  }
}

export function formatSourceContentClass(type: string | null | undefined) {
  switch ((type ?? "").trim().toLowerCase()) {
    case "video":
      return "视频"
    case "audio":
      return "音乐"
    case "text":
      return "文本"
    case "image":
      return "图片"
    case "other":
      return "其他"
    default:
      return "未确定"
  }
}

export function formatProbeStatus(status: string | null | undefined) {
  switch ((status ?? "").trim().toLowerCase()) {
    case "ready":
      return "已探测"
    case "partial":
      return "部分探测"
    case "error":
      return "探测失败"
    case "pending":
      return "等待探测"
    default:
      return status || "未知"
  }
}
