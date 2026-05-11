import { Link } from '@tanstack/react-router'
import { LoaderCircleIcon, SparklesIcon } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '#/components/ui/alert'
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
import { createAuthedMiboApi, miboQueryKeys } from '#/lib/mibo-query'
import {
  getMediaCardMatchStatus,
  getMediaCardMetadataProvider,
  getMediaCardPosterUrl,
} from '#/lib/media-presentation'
import { getDisplayMatchStatus } from '#/features/media/components/standalone-media-detail-utils'
import { useQuery } from '@tanstack/react-query'

import { formatMatchStatus, formatMediaType } from './formatters'

export function MetadataGovernanceWorkspace({ token }: { token: string }) {
  const homeSectionsQuery = useQuery({
    queryKey: miboQueryKeys.metadataWorkspace(token),
    queryFn: () => createAuthedMiboApi(token).homeSections(),
  })

  return (
    <div className="space-y-4 text-foreground">
      <div className="flex flex-col gap-4 rounded-[1.75rem] border border-border/60 bg-card/80 p-5 shadow-sm backdrop-blur-sm sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-2">
          <Badge
            variant="outline"
            className="border-border/60 bg-background/70"
          >
            Phase 7
          </Badge>
          <h1 className="text-3xl font-semibold tracking-tight">
            元数据治理工作台
          </h1>
          <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
            在设置中心内定位待修正条目，进入单条目治理页执行匹配预览、人工校对和后台任务操作。
          </p>
        </div>
      </div>

      <Alert>
        <SparklesIcon className="size-4" />
        <AlertTitle>工作台入口已就位</AlertTitle>
        <AlertDescription>
          当前页面提供全局治理入口和单条目跳转，支持候选搜索、手工保存、重新匹配和元数据重抓的完整治理流程。
        </AlertDescription>
      </Alert>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
          <CardHeader className="px-5 py-5">
            <CardTitle>最近可治理条目</CardTitle>
            <CardDescription>
              按内容形态聚合最近内容，作为管理员进入治理页的全局入口。
            </CardDescription>
          </CardHeader>
          <Separator className="bg-border" />
          <CardContent className="space-y-5 px-5 py-5">
            {homeSectionsQuery.isLoading ? (
              <WorkspaceLoadingState />
            ) : homeSectionsQuery.error ? (
              <Alert>
                <AlertTitle>加载失败</AlertTitle>
                <AlertDescription>
                  {homeSectionsQuery.error.message}
                </AlertDescription>
              </Alert>
            ) : homeSectionsQuery.data?.length ? (
              homeSectionsQuery.data.map((section) => (
                <div key={section.key} className="space-y-3">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div className="text-sm font-medium text-foreground">
                        {section.title}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {section.items.length} 个最近条目
                      </div>
                    </div>
                  </div>

                  <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                    {section.items.map((rawItem) => {
                      return (
                        <Card
                          key={rawItem.id}
                          className="rounded-[1.25rem] border-border/60 bg-background/60 py-0"
                        >
                          <CardContent className="space-y-3 px-4 py-4">
                            <div className="flex items-start gap-3">
                              <div className="h-20 w-14 overflow-hidden rounded-lg bg-muted">
                                {getMediaCardPosterUrl(rawItem) ? (
                                  <img
                                    src={getMediaCardPosterUrl(rawItem)}
                                    alt={rawItem.title}
                                    className="h-full w-full object-cover"
                                  />
                                ) : null}
                              </div>
                              <div className="min-w-0 flex-1 space-y-2">
                                <div>
                                  <div className="line-clamp-2 text-sm font-medium text-foreground">
                                    {rawItem.title}
                                  </div>
                                  <div className="mt-1 text-xs text-muted-foreground">
                                    {rawItem.year ?? '年份未知'} ·{' '}
                                    {formatMediaType(rawItem.type)}
                                  </div>
                                </div>
                                <div className="flex flex-wrap gap-2">
                                  <Badge
                                    variant="outline"
                                    className="border-border/60 bg-card/70 text-[11px]"
                                  >
                                    {formatMatchStatus(
                                      getDisplayMatchStatus({
                                        governance_status:
                                          getMediaCardMatchStatus(rawItem),
                                      }),
                                    )}
                                  </Badge>
                                  {getMediaCardMetadataProvider(rawItem) ? (
                                    <Badge
                                      variant="secondary"
                                      className="text-[11px]"
                                    >
                                      {getMediaCardMetadataProvider(
                                        rawItem,
                                      ).toUpperCase()}
                                    </Badge>
                                  ) : null}
                                </div>
                              </div>
                            </div>

                            <Button asChild>
                              <Link
                                to="/settings/metadata/$id"
                                params={{ id: String(rawItem.id) }}
                              >
                                进入治理
                              </Link>
                            </Button>
                          </CardContent>
                        </Card>
                      )
                    })}
                  </div>
                </div>
              ))
            ) : (
              <div className="rounded-[1.25rem] border border-dashed border-border/70 px-4 py-10 text-center text-sm text-muted-foreground">
                当前没有可展示的条目。
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
          <CardHeader className="px-5 py-5">
            <CardTitle>治理动作说明</CardTitle>
            <CardDescription>
              区分四个动作，避免在后台治理时混淆覆盖范围。
            </CardDescription>
          </CardHeader>
          <Separator className="bg-border" />
          <CardContent className="space-y-4 px-5 py-5 text-sm text-muted-foreground">
            <GovernanceActionItem
              title="搜索候选"
              description="输入标题或年份，返回候选供管理员比对。"
            />
            <GovernanceActionItem
              title="应用候选"
              description="先看差异预览，再确认覆盖元数据。"
            />
            <GovernanceActionItem
              title="重新匹配"
              description="走后台任务，适合整条目重新识别。"
            />
            <GovernanceActionItem
              title="元数据重抓"
              description="作为独立后台任务执行，并在完成后刷新当前治理结果。"
            />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function GovernanceActionItem({
  title,
  description,
}: {
  title: string
  description: string
}) {
  return (
    <div className="rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3">
      <div className="text-sm font-medium text-foreground">{title}</div>
      <div className="mt-1 text-sm text-muted-foreground">{description}</div>
    </div>
  )
}

function WorkspaceLoadingState() {
  return (
    <div className="flex items-center gap-3 rounded-[1.25rem] border border-border/60 bg-background/60 px-4 py-6 text-sm text-muted-foreground">
      <LoaderCircleIcon className="size-4 animate-spin" />
      正在加载最近条目
    </div>
  )
}
