import { FolderPlusIcon, Trash2Icon } from 'lucide-react'

import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import { Separator } from '#/components/ui/separator'
import {
  healthReasonMessage,
  healthReasonTitle,
  healthSeverityClassName,
} from '#/lib/health-presentation'
import type { HealthIssue, MediaSource } from '#/lib/mibo-api'

import { EmptyCard, InfoRow } from './settings-aside-card'

export function MediaSourcesTab({
  mediaSources,
  healthIssues,
  isLoading,
  onCreate,
  onEdit,
  onDelete,
}: {
  mediaSources: MediaSource[]
  healthIssues: HealthIssue[]
  isLoading: boolean
  onCreate: () => void
  onEdit: (source: MediaSource) => void
  onDelete: (source: MediaSource) => void
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
      <CardHeader className="flex flex-col gap-4 px-5 py-5 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <CardTitle className="text-xl">媒体源</CardTitle>
          <CardDescription className="mt-1">
            管理本地目录和 OpenList 数据源，媒体库会从这些来源中挂载目录。
          </CardDescription>
        </div>
        <Button onClick={onCreate}>
          <FolderPlusIcon className="size-4" />
          创建媒体源
        </Button>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="px-5 py-5">
        <div className="grid gap-4 xl:grid-cols-2 2xl:grid-cols-3">
          {mediaSources.map((source) => {
            const issue = healthIssues.find((entry) =>
              entry.affected.media_sources?.some((ref) => ref.id === source.id),
            )

            return (
              <Card
                key={source.id}
                className="rounded-[1.25rem] border-border/60 bg-background/60 py-0 shadow-none"
              >
              <CardHeader className="px-4 py-4">
                <CardTitle className="flex items-center justify-between gap-3 text-base">
                  <span>{source.name}</span>
                  <Badge
                    className="border-border/60 bg-muted text-foreground"
                    variant="outline"
                  >
                    {source.provider}
                  </Badge>
                </CardTitle>
                <CardDescription>
                  最近更新于 {formatDateTime(source.updated_at)}
                </CardDescription>
              </CardHeader>
              <Separator className="bg-border" />
              <CardContent className="space-y-3 px-4 py-4 text-sm text-muted-foreground">
                <InfoRow label="根路径" value={source.root_path} />
                <InfoRow label="来源类型" value={source.provider} />
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
                {source.provider === 'openlist' ? (
                  <InfoRow
                    label="OpenList 地址"
                    value={source.config?.openlist?.base_url ?? '未配置'}
                  />
                ) : null}
                <div className="flex justify-end gap-2 pt-2">
                  <Button
                    variant="outline"
                    className="border-border/60 bg-card/80 text-foreground hover:bg-muted hover:text-foreground"
                    onClick={() => onEdit(source)}
                  >
                    编辑
                  </Button>
                  <Button
                    variant="outline"
                    className="border-border/60 bg-card/80 text-foreground hover:bg-muted hover:text-foreground"
                    onClick={() => onDelete(source)}
                  >
                    <Trash2Icon className="size-4" />
                    删除
                  </Button>
                </div>
              </CardContent>
              </Card>
            )
          })}

          {!mediaSources.length && !isLoading ? (
            <EmptyCard text="还没有媒体源，点击上方按钮创建一个本地目录或 OpenList 来源。" />
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}

function formatDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}
