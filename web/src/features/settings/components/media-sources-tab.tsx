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
import type { MediaSource } from '#/lib/mibo-api'

import { EmptyCard, InfoRow } from './settings-aside-card'

export function MediaSourcesTab({
  mediaSources,
  isLoading,
  onCreate,
  onEdit,
  onDelete,
}: {
  mediaSources: MediaSource[]
  isLoading: boolean
  onCreate: () => void
  onEdit: (source: MediaSource) => void
  onDelete: (source: MediaSource) => void
}) {
  return (
    <>
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="text-base font-medium text-foreground">媒体源</div>
          <div className="text-sm text-muted-foreground">
            管理本地目录和 OpenList 数据源。
          </div>
        </div>
        <Button onClick={onCreate}>
          <FolderPlusIcon className="size-4" />
          创建媒体源
        </Button>
      </div>

      <div className="grid gap-4 xl:grid-cols-2 2xl:grid-cols-3">
        {mediaSources.map((source) => (
          <Card
            key={source.id}
            className="rounded-[1.5rem] border-border/60 bg-card/80 py-0"
          >
            <CardHeader className="px-5 py-5">
              <CardTitle className="flex items-center justify-between gap-3 text-lg">
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
            <CardContent className="space-y-3 px-5 py-5 text-sm text-muted-foreground">
              <InfoRow label="根路径" value={source.root_path} />
              <InfoRow label="来源类型" value={source.provider} />
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
        ))}

        {!mediaSources.length && !isLoading ? (
          <EmptyCard text="还没有媒体源，点击上方按钮创建一个本地目录或 OpenList 来源。" />
        ) : null}
      </div>
    </>
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
