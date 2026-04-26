import { FolderPlusIcon, LoaderCircleIcon, Trash2Icon } from 'lucide-react'

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
import type { Library, MediaSource } from '#/lib/mibo-api'

import { EmptyCard, InfoRow } from './settings-aside-card'

export function LibrariesTab({
  libraries,
  mediaSources,
  isLoading,
  isScanning,
  onCreate,
  onScan,
  onDelete,
}: {
  libraries: Library[]
  mediaSources: MediaSource[]
  isLoading: boolean
  isScanning: boolean
  onCreate: () => void
  onScan: (libraryId: number) => void
  onDelete: (library: Library) => void
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
      <CardHeader className="flex flex-col gap-4 px-5 py-5 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <CardTitle className="text-xl">媒体库</CardTitle>
          <CardDescription className="mt-1">
            绑定媒体源目录，控制扫描入口和库级管理操作。
          </CardDescription>
        </div>
        <Button onClick={onCreate}>
          <FolderPlusIcon className="size-4" />
          创建媒体库
        </Button>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="px-5 py-5">
        <div className="grid gap-4 xl:grid-cols-2 2xl:grid-cols-3">
          {libraries.map((library) => {
            const sourceName =
              mediaSources.find(
                (source) => source.id === library.media_source_id,
              )?.name ?? `媒体源 #${library.media_source_id}`

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
                      {library.type}
                    </Badge>
                  </CardTitle>
                  <CardDescription>绑定媒体源：{sourceName}</CardDescription>
                </CardHeader>
                <Separator className="bg-border" />
                <CardContent className="space-y-3 px-4 py-4 text-sm text-muted-foreground">
                  <InfoRow label="目录" value={library.root_path} />
                  <InfoRow label="状态" value={library.status} />
                  <InfoRow
                    label="扫描"
                    value={library.scanner_enabled ? '已启用' : '已关闭'}
                  />
                  <div className="flex justify-end gap-2 pt-2">
                    <Button
                      variant="outline"
                      className="border-border/60 bg-card/80 text-foreground hover:bg-muted hover:text-foreground"
                      disabled
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
            <EmptyCard text="还没有媒体库，点击上方按钮创建一个媒体库。" />
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}
