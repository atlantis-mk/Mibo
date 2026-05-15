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
