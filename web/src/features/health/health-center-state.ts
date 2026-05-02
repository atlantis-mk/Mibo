import type { HealthIssue } from "#/lib/mibo-api"

export type HealthCenterPendingState = {
  validatePending: boolean
  rescanPending: boolean
  ignorePending: boolean
}

export function getHealthCenterState(
  issues: HealthIssue[],
  pending: HealthCenterPendingState
) {
  const blockingIssues = issues.filter((issue) => issue.severity === "blocking")
  const otherIssues = issues.filter((issue) => issue.severity !== "blocking")

  return {
    activeIssueCount: issues.length,
    blockingIssues,
    otherIssues,
    isEmpty: issues.length === 0,
    hasBlockingIssues: blockingIssues.length > 0,
    hasOtherIssues: otherIssues.length > 0,
    actionLoading:
      pending.validatePending || pending.rescanPending || pending.ignorePending,
  }
}
