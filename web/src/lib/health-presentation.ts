import type { HealthIssue, HealthSeverity } from "#/lib/mibo-api"

export function healthSeverityLabel(severity: HealthSeverity) {
  if (severity === "blocking") return "需要处理"
  if (severity === "error") return "错误"
  if (severity === "warning") return "警告"
  return "提示"
}

export function healthSeverityClassName(severity: HealthSeverity) {
  if (severity === "blocking")
    return "border-destructive/40 bg-destructive/10 text-destructive"
  if (severity === "error")
    return "border-destructive/30 bg-destructive/10 text-destructive"
  if (severity === "warning")
    return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300"
  return "border-border/60 bg-muted text-muted-foreground"
}

export function healthReasonTitle(issue: HealthIssue) {
  if (issue.reason_code === "storage_auth_expired") {
    return "PikPak 登录验证已过期"
  }
  if (issue.reason_code === "job_failed_unknown") {
    return affectedLibraries(issue).length > 0
      ? "媒体库任务失败"
      : "后台任务失败"
  }
  return issue.title || "系统健康问题"
}

export function healthReasonMessage(issue: HealthIssue) {
  if (issue.reason_code === "storage_auth_expired") {
    return "OpenList/PikPak 的登录或验证码验证已过期。内容没有丢失，完成验证后可重新扫描恢复显示。"
  }
  return issue.message || "请查看技术详情或最近失败任务。"
}

export function findBlockingHomeIssue(issues: HealthIssue[]) {
  return issues.find(
    (issue) =>
      issue.severity === "blocking" && issue.impact.blocks_home_visibility
  )
}

export function affectedLibraryNames(issue: HealthIssue) {
  return affectedLibraries(issue)
    .map((library) => library.name)
    .join("、")
}

function affectedLibraries(issue: HealthIssue) {
  return issue.affected?.libraries ?? []
}
