import { FolderPlusIcon, LoaderCircleIcon, Trash2Icon } from "lucide-react"

import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card"
import { Separator } from "#/components/ui/separator"
import {
  healthReasonMessage,
  healthReasonTitle,
  healthSeverityClassName,
} from "#/lib/health-presentation"
import {
  formatProbeStatus,
  formatSourceContentClass,
} from "#/lib/library-presentation"
import type { HealthIssue, Library, MediaSource } from "#/lib/mibo-api"

import { EmptyCard, InfoRow } from "./settings-aside-card"

export function LibrariesTab({
  libraries,
  mediaSources,
  healthIssues,
  isLoading,
  isScanning,
  onCreate,
  onEdit,
  onScan,
  onDelete,
}: {
  libraries: Library[]
  mediaSources: MediaSource[]
  healthIssues: HealthIssue[]
  isLoading: boolean
  isScanning: boolean
  onCreate: () => void
  onEdit: (library: Library) => void
  onScan: (libraryId: number) => void
  onDelete: (library: Library) => void
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
      <CardHeader className="flex flex-col gap-4 px-5 py-5 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <CardTitle className="text-xl">内容来源</CardTitle>
          <CardDescription className="mt-1">
            绑定媒体源目录，Mibo 会探测内容类型并自动分类。
          </CardDescription>
        </div>
        <Button onClick={onCreate}>
          <FolderPlusIcon className="size-4" />
          添加来源
        </Button>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="px-5 py-5">
        <div className="grid gap-4 xl:grid-cols-2 2xl:grid-cols-3">
          {libraries.map((library) => {
            const sourceName =
              mediaSources.find(
                (source) => source.id === library.media_source_id
              )?.name ?? `媒体源 #${library.media_source_id}`
            const issue = healthIssues.find((entry) =>
              entry.affected.libraries.some((ref) => ref.id === library.id)
            )

            return (
              <Card
                key={library.id}
                className="rounded-[1.25rem] border-border/60 bg-background/60 py-0 shadow-none"
              >
                <CardHeader className="px-4 py-4">
                  <CardTitle className="flex items-center justify-between gap-3 text-base">
                    <span>{library.name}</span>
                    <Badge
                      className="border-border/60 bg-muted text-foreground"
                      variant="outline"
                    >
                      {formatSourceContentClass(
                        library.probe_summary?.dominant_class
                      )}
                    </Badge>
                  </CardTitle>
                  <CardDescription>绑定媒体源：{sourceName}</CardDescription>
                </CardHeader>
                <Separator className="bg-border" />
                <CardContent className="space-y-3 px-4 py-4 text-sm text-muted-foreground">
                  <InfoRow
                    label="目录"
                    value={`${library.paths?.filter((path) => path.enabled).length || 1} 个路径`}
                  />
                  <InfoRow label="状态" value={library.status} />
                  <InfoRow
                    label="探测"
                    value={formatProbeStatus(
                      library.probe_summary?.status || library.probe_status
                    )}
                  />
                  {library.probe_summary ? (
                    <InfoRow
                      label="样本"
                      value={`${library.probe_summary.sampled_files} 个文件 / ${library.probe_summary.sampled_dirs} 个目录${library.probe_summary.budget_limited ? " · 已达预算" : ""}`}
                    />
                  ) : null}
                  {library.collections?.length ? (
                    <div className="flex flex-wrap gap-1.5">
                      {library.collections.map((collection) => (
                        <Badge
                          key={collection.content_class}
                          variant="outline"
                          className="border-border/60 bg-muted/60 text-muted-foreground"
                        >
                          {collection.label} {collection.count}
                        </Badge>
                      ))}
                    </div>
                  ) : null}
                  {issue ? (
                    <div className="rounded-[1rem] border border-destructive/30 bg-destructive/5 p-3 text-sm">
                      <Badge
                        className={healthSeverityClassName(issue.severity)}
                        variant="outline"
                      >
                        {healthReasonTitle(issue)}
                      </Badge>
                      <div className="mt-2 leading-6 text-foreground">
                        {healthReasonMessage(issue)}
                      </div>
                    </div>
                  ) : null}
                  <InfoRow
                    label="扫描"
                    value={library.scanner_enabled ? "已启用" : "已关闭"}
                  />
                  <InfoRow
                    label="元数据 Template"
                    value={
                      library.policies?.metadata?.metadata_profile_name ||
                      "迁移默认配置"
                    }
                  />
                  <div className="flex justify-end gap-2 pt-2">
                    <Button
                      variant="outline"
                      className="border-border/60 bg-card/80 text-foreground hover:bg-muted hover:text-foreground"
                      onClick={() => onEdit(library)}
                    >
                      编辑
                    </Button>
                    <Button
                      variant="outline"
                      className="border-border/60 bg-card/80 text-foreground hover:bg-muted hover:text-foreground"
                      onClick={() => onScan(library.id)}
                    >
                      {isScanning ? (
                        <LoaderCircleIcon className="size-4 animate-spin" />
                      ) : null}
                      扫描
                    </Button>
                    <Button
                      variant="outline"
                      className="border-border/60 bg-card/80 text-foreground hover:bg-muted hover:text-foreground"
                      onClick={() => onDelete(library)}
                    >
                      <Trash2Icon className="size-4" />
                      删除
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )
          })}

          {!libraries.length && !isLoading ? (
            <EmptyCard text="还没有内容来源，点击上方按钮添加一个目录。" />
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}
