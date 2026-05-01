import { useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  FileX2Icon,
  Loader2Icon,
  RefreshCwIcon,
  RotateCcwIcon,
  ShieldOffIcon,
} from "lucide-react"

import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import { NativeSelect, NativeSelectOption } from "#/components/ui/native-select"
import { Skeleton } from "#/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "#/components/ui/table"
import type { FilenameExclusionRule, ScanExclusion } from "#/lib/mibo-api"
import {
  createAuthedMiboApi,
  librariesQueryOptions,
  miboQueryKeys,
  scanExclusionsQueryOptions,
} from "#/lib/mibo-query"
import { cn } from "#/lib/utils"

import { ScanExclusionRulesPanel } from "./scan-exclusion-rules-panel"

type EnabledFilter = "all" | "enabled" | "disabled"

export function ScanExclusionsPanel({
  token,
  activeTab,
}: {
  token: string | null
  activeTab: "rules" | "exclusions"
}) {
  const queryClient = useQueryClient()
  const queryToken = token ?? "guest"
  const [libraryFilter, setLibraryFilter] = useState("all")
  const [enabledFilter, setEnabledFilter] = useState<EnabledFilter>("all")
  const [actionMessage, setActionMessage] = useState<string | null>(null)

  const filters = useMemo(
    () => ({
      libraryId: libraryFilter === "all" ? undefined : Number(libraryFilter),
      enabled:
        enabledFilter === "all" ? undefined : enabledFilter === "enabled",
    }),
    [enabledFilter, libraryFilter]
  )

  const exclusionsQuery = useQuery({
    ...scanExclusionsQueryOptions(queryToken, filters),
    enabled: Boolean(token),
  })
  const librariesQuery = useQuery({
    ...librariesQueryOptions(queryToken),
    enabled: Boolean(token),
  })

  const exclusions = exclusionsQuery.data?.manual_exclusions ?? []
  const filenameRules = exclusionsQuery.data?.filename_rules ?? []
  const enabledCount = exclusions.filter((item) => item.enabled).length
  const enabledRuleCount = filenameRules.filter((item) => item.enabled).length
  const disabledCount =
    exclusions.length + filenameRules.length - enabledCount - enabledRuleCount

  const invalidateExclusions = async () => {
    if (!token) return
    await queryClient.invalidateQueries({
      queryKey: miboQueryKeys.scanExclusions(queryToken, filters),
    })
  }

  const toggleMutation = useMutation({
    mutationFn: async (input: { id: number; enabled: boolean }) => {
      if (!token) throw new Error("当前未登录，无法更新扫描排除项。")
      return createAuthedMiboApi(token).setScanExclusionEnabled(
        input.id,
        input.enabled
      )
    },
    onSuccess: async (updated) => {
      setActionMessage(
        updated.enabled ? "排除项已重新启用。" : "排除项已恢复。"
      )
      await invalidateExclusions()
    },
  })

  const ruleToggleMutation = useMutation({
    mutationFn: async (input: { id: number; enabled: boolean }) => {
      if (!token) throw new Error("当前未登录，无法更新同名忽略规则。")
      return createAuthedMiboApi(token).setFilenameExclusionRuleEnabled(
        input.id,
        input.enabled
      )
    },
    onSuccess: async (updated) => {
      setActionMessage(
        updated.enabled
          ? "同名忽略规则已重新启用。"
          : "同名忽略规则已恢复。后续扫描会重新允许这些文件。"
      )
      await invalidateExclusions()
    },
  })

  const restoreMemberMutation = useMutation({
    mutationFn: async (input: { groupId: number; fileId: number }) => {
      if (!token) throw new Error("当前未登录，无法恢复文件。")
      return createAuthedMiboApi(token).restoreFilenameExclusionMatch(
        input.groupId,
        input.fileId
      )
    },
    onSuccess: async () => {
      setActionMessage("文件已单独恢复，后续扫描会重新允许该文件。")
      await invalidateExclusions()
    },
  })

  return (
    <div className="space-y-4">
      {activeTab === "rules" ? <ScanExclusionRulesPanel token={token} /> : null}

      {activeTab === "exclusions" ? (
        <>
          <section className="rounded-[1.5rem] border border-border/60 bg-card/70 p-4 shadow-sm backdrop-blur-sm">
            <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
              <div className="space-y-2">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant="outline" className="gap-1.5 bg-background/70">
                    <FileX2Icon className="size-3.5 text-amber-500" />
                    {exclusions.length + filenameRules.length} 条排除记录
                  </Badge>
                  <Badge variant="outline" className="gap-1.5 bg-background/70">
                    <ShieldOffIcon className="size-3.5 text-emerald-500" />
                    {enabledCount + enabledRuleCount} 条生效
                  </Badge>
                  <Badge variant="outline" className="gap-1.5 bg-background/70">
                    <RotateCcwIcon className="size-3.5 text-muted-foreground" />
                    {disabledCount} 条已恢复
                  </Badge>
                </div>
                <p className="text-sm text-muted-foreground">
                  管理被标记为广告或误导入的文件。恢复后不会删除历史记录，后续扫描会重新允许该文件进入导入流程。
                </p>
              </div>

              <div className="flex flex-wrap items-center gap-2">
                <NativeSelect
                  value={libraryFilter}
                  onChange={(event) =>
                    setLibraryFilter(event.currentTarget.value)
                  }
                  className="w-full sm:w-44"
                  aria-label="按媒体库筛选"
                >
                  <NativeSelectOption value="all">
                    全部媒体库
                  </NativeSelectOption>
                  {(librariesQuery.data ?? []).map((library) => (
                    <NativeSelectOption
                      key={library.id}
                      value={String(library.id)}
                    >
                      {library.name}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
                <NativeSelect
                  value={enabledFilter}
                  onChange={(event) =>
                    setEnabledFilter(event.currentTarget.value as EnabledFilter)
                  }
                  className="w-full sm:w-36"
                  aria-label="按状态筛选"
                >
                  <NativeSelectOption value="all">全部状态</NativeSelectOption>
                  <NativeSelectOption value="enabled">
                    仅生效
                  </NativeSelectOption>
                  <NativeSelectOption value="disabled">
                    仅恢复
                  </NativeSelectOption>
                </NativeSelect>
                <Button
                  variant="outline"
                  onClick={() => void invalidateExclusions()}
                  disabled={!token || exclusionsQuery.isFetching}
                >
                  <RefreshCwIcon
                    className={cn(
                      "size-4",
                      exclusionsQuery.isFetching && "animate-spin"
                    )}
                  />
                  刷新
                </Button>
              </div>
            </div>
          </section>

          {actionMessage ? (
            <div className="flex items-center gap-2 rounded-[1.1rem] border border-border bg-muted px-4 py-3 text-sm text-foreground">
              <CheckCircle2Icon className="size-4 text-muted-foreground" />
              <span>{actionMessage}</span>
            </div>
          ) : null}

          {toggleMutation.error ||
          ruleToggleMutation.error ||
          restoreMemberMutation.error ? (
            <ErrorBanner
              message={errorMessage(
                toggleMutation.error ||
                  ruleToggleMutation.error ||
                  restoreMemberMutation.error
              )}
            />
          ) : null}

          <section className="min-h-[420px] rounded-[1.5rem] border border-border/60 bg-gradient-to-br from-card/90 via-card/70 to-amber-500/5 p-5 shadow-sm backdrop-blur-sm">
            <div className="mb-5">
              <h3 className="text-base font-medium">排除项列表</h3>
              <p className="text-sm text-muted-foreground">
                自动广告规则不会出现在这里；这里只显示用户标记并持久化的扫描排除项。
              </p>
            </div>

            {exclusionsQuery.isLoading ? (
              <ExclusionSkeleton />
            ) : exclusionsQuery.isError ? (
              <ErrorState onRetry={() => void invalidateExclusions()} />
            ) : exclusions.length === 0 && filenameRules.length === 0 ? (
              <EmptyState />
            ) : (
              <div className="space-y-4">
                {filenameRules.length > 0 ? (
                  <FilenameRulesList
                    rules={filenameRules}
                    pending={
                      ruleToggleMutation.isPending ||
                      restoreMemberMutation.isPending
                    }
                    onToggle={(rule) =>
                      ruleToggleMutation.mutate({
                        id: rule.id,
                        enabled: !rule.enabled,
                      })
                    }
                    onRestoreMember={(rule, fileId) =>
                      restoreMemberMutation.mutate({
                        groupId: rule.id,
                        fileId,
                      })
                    }
                  />
                ) : null}
                {exclusions.length > 0 ? (
                  <ExclusionsTable
                    exclusions={exclusions}
                    pending={toggleMutation.isPending}
                    onToggle={(exclusion) =>
                      toggleMutation.mutate({
                        id: exclusion.id,
                        enabled: !exclusion.enabled,
                      })
                    }
                  />
                ) : null}
              </div>
            )}
          </section>
        </>
      ) : null}
    </div>
  )
}

function FilenameRulesList({
  rules,
  pending,
  onToggle,
  onRestoreMember,
}: {
  rules: FilenameExclusionRule[]
  pending: boolean
  onToggle: (rule: FilenameExclusionRule) => void
  onRestoreMember: (rule: FilenameExclusionRule, fileId: number) => void
}) {
  return (
    <div className="space-y-3">
      {rules.map((rule) => (
        <div
          key={rule.id}
          className="rounded-[1.35rem] border border-border/60 bg-background/80 p-4 shadow-sm"
        >
          <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
            <div className="space-y-1">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{rule.normalized_filename}</span>
                <Badge variant={rule.enabled ? "default" : "outline"}>
                  {rule.enabled ? "同名规则生效中" : "已恢复"}
                </Badge>
                <Badge variant="outline">{rule.affected_count} 个文件</Badge>
              </div>
              <p className="text-sm text-muted-foreground">
                所有来源 / {reasonLabel(rule.reason)}
              </p>
              <p className="text-xs text-muted-foreground">
                恢复后不会立即重建媒体，后续扫描会重新允许这些文件进入导入流程。
              </p>
            </div>
            <Button
              size="sm"
              variant={rule.enabled ? "outline" : "default"}
              disabled={pending}
              onClick={() => onToggle(rule)}
            >
              {rule.enabled ? "恢复同名规则" : "重新启用"}
            </Button>
          </div>
          <div className="mt-4 space-y-2">
            {rule.affected_files.map((file) => (
              <div
                key={file.id}
                className="flex flex-col gap-2 rounded-xl border border-border/50 bg-muted/30 px-3 py-2 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="min-w-0">
                  <div className="truncate text-sm" title={file.storage_path}>
                    {file.storage_path}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {file.restored ? "已单独恢复" : "被同名规则排除"} /{" "}
                    {file.status}
                  </div>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={pending || file.restored || !rule.enabled}
                  onClick={() => onRestoreMember(rule, file.id)}
                >
                  恢复此文件
                </Button>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function ExclusionsTable({
  exclusions,
  pending,
  onToggle,
}: {
  exclusions: ScanExclusion[]
  pending: boolean
  onToggle: (exclusion: ScanExclusion) => void
}) {
  return (
    <div className="rounded-[1.35rem] border border-border/60 bg-background/80 shadow-sm">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="min-w-64">文件</TableHead>
            <TableHead>媒体库</TableHead>
            <TableHead>原因</TableHead>
            <TableHead>存储</TableHead>
            <TableHead className="min-w-56">稳定标识</TableHead>
            <TableHead>更新时间</TableHead>
            <TableHead>状态</TableHead>
            <TableHead className="text-right">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {exclusions.map((exclusion) => (
            <TableRow key={exclusion.id}>
              <TableCell className="max-w-80">
                <div className="space-y-1">
                  <div
                    className="truncate font-medium"
                    title={exclusion.storage_path}
                  >
                    {fileNameFromPath(exclusion.storage_path)}
                  </div>
                  <div
                    className="truncate text-xs text-muted-foreground"
                    title={exclusion.storage_path || "未记录"}
                  >
                    {exclusion.storage_path || "未记录"}
                  </div>
                </div>
              </TableCell>
              <TableCell>
                {exclusion.library_name || `#${exclusion.library_id}`}
              </TableCell>
              <TableCell>{reasonLabel(exclusion.reason)}</TableCell>
              <TableCell>{exclusion.storage_provider || "未知"}</TableCell>
              <TableCell className="max-w-64 truncate font-mono text-xs">
                {exclusion.stable_identity_key || "路径回退"}
              </TableCell>
              <TableCell>{formatDateTime(exclusion.updated_at)}</TableCell>
              <TableCell>
                <Badge variant={exclusion.enabled ? "default" : "outline"}>
                  {exclusion.enabled ? "生效中" : "已恢复"}
                </Badge>
              </TableCell>
              <TableCell>
                <div className="flex justify-end">
                  <Button
                    size="sm"
                    variant={exclusion.enabled ? "outline" : "default"}
                    disabled={pending}
                    onClick={() => onToggle(exclusion)}
                  >
                    {pending ? (
                      <Loader2Icon className="size-4 animate-spin" />
                    ) : null}
                    {exclusion.enabled ? "恢复" : "重新启用"}
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function ExclusionSkeleton() {
  return (
    <div className="rounded-[1.35rem] border border-border/60 bg-background/80 p-4 shadow-sm">
      <div className="space-y-3">
        {Array.from({ length: 6 }).map((_, index) => (
          <Skeleton key={index} className="h-10 rounded-xl" />
        ))}
      </div>
    </div>
  )
}

function EmptyState() {
  return (
    <div className="flex min-h-[260px] flex-col items-center justify-center rounded-[1.35rem] border border-dashed border-border/70 bg-background/60 p-8 text-center">
      <FileX2Icon className="size-10 text-muted-foreground" />
      <h4 className="mt-4 text-base font-medium">暂无扫描排除项</h4>
      <p className="mt-2 max-w-md text-sm leading-6 text-muted-foreground">
        当你从媒体详情、资产或文件操作中标记广告/误导入文件后，它们会出现在这里。
      </p>
    </div>
  )
}

function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex min-h-[260px] flex-col items-center justify-center rounded-[1.35rem] border border-dashed border-destructive/30 bg-destructive/5 p-8 text-center">
      <AlertTriangleIcon className="size-10 text-destructive" />
      <h4 className="mt-4 text-base font-medium">无法加载扫描排除项</h4>
      <p className="mt-2 max-w-md text-sm leading-6 text-muted-foreground">
        请检查当前登录状态或稍后重试。
      </p>
      <Button className="mt-4" variant="outline" onClick={onRetry}>
        重新加载
      </Button>
    </div>
  )
}

function ErrorBanner({ message }: { message: string }) {
  return (
    <div className="flex items-start gap-3 rounded-2xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
      <AlertTriangleIcon className="mt-0.5 size-4 shrink-0" />
      <span>{message}</span>
    </div>
  )
}

function reasonLabel(reason: string) {
  switch (reason) {
    case "advertisement":
      return "广告"
    case "unwanted":
      return "不需要"
    case "duplicate":
      return "重复导入"
    case "wrong_import":
      return "误导入"
    case "other":
      return "其他"
    default:
      return reason || "未知"
  }
}

function fileNameFromPath(value: string) {
  const segments = value.split("/").filter(Boolean)
  return segments.at(-1) || value || "未知文件"
}

function formatDateTime(value?: string) {
  if (!value) return "未知"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "未知"
  return new Intl.DateTimeFormat("zh-CN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date)
}

function errorMessage(error: unknown) {
  if (error instanceof Error) return error.message
  return "操作失败，请稍后重试。"
}
