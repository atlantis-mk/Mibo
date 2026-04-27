import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { LoaderCircleIcon } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '#/components/ui/alert'
import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import { StandaloneMediaDetail } from '#/features/media/components/standalone-media-detail'
import {
  buildPresentedCatalogItem,
  catalogEpisodeShelfToSeasonRails,
  catalogItemDetailToPresentation,
  catalogSeasonsToRails,
  type MediaDetailView,
} from '#/lib/media-presentation'
import {
  catalogItemDetailQueryOptions,
  catalogItemProgressQueryOptions,
  catalogSeriesSeasonsQueryOptions,
  createAuthedMiboApi,
  favoritesQueryOptions,
  homeDataQueryOptions,
  miboQueryKeys,
} from '#/lib/mibo-query'
import { useAuthStore } from '#/stores/auth-store'

export default function MediaDetail({
  itemId,
  detailView,
}: {
  itemId: number
  detailView: MediaDetailView
}) {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const queryToken = token ?? 'guest'
  const hasValidItemId = Number.isFinite(itemId) && itemId > 0

  const itemQuery = useQuery({
    ...catalogItemDetailQueryOptions(queryToken, itemId),
    enabled: hasHydrated && !!token && hasValidItemId,
  })
  const progressQuery = useQuery({
    ...catalogItemProgressQueryOptions(queryToken, itemId),
    enabled: hasHydrated && !!token && hasValidItemId,
  })
  const favoritesQuery = useQuery({
    ...favoritesQueryOptions(queryToken),
    enabled: hasHydrated && !!token && hasValidItemId,
  })
  const detailItem = itemQuery.data
  const detailAssets = itemQuery.data?.assets ?? []
  const presentationItem = itemQuery.data
    ? catalogItemDetailToPresentation(itemQuery.data)
    : null
  const seriesEpisodesQuery = useQuery({
    ...catalogSeriesSeasonsQueryOptions(queryToken, detailItem?.id ?? 0),
    enabled: hasHydrated && !!token && detailItem?.type === 'series',
  })
  const presentedItem = itemQuery.data
    ? buildPresentedCatalogItem(
        presentationItem ?? catalogItemDetailToPresentation(itemQuery.data),
        catalogSeasonsToRails(seriesEpisodesQuery.data ?? []),
        detailView,
      )
    : null
  const displayedSeasonRails = presentedItem
    ? presentedItem.type === 'episode'
      ? catalogEpisodeShelfToSeasonRails(presentedItem)
      : catalogSeasonsToRails(seriesEpisodesQuery.data ?? [])
    : []

  const rematchMutation = useMutation({
    mutationFn: async () => {
      if (!token) {
        throw new Error('当前未登录，无法重新匹配媒体。')
      }

      return createAuthedMiboApi(token).refetchCatalogItemMetadata(itemId)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: catalogItemDetailQueryOptions(queryToken, itemId).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: homeDataQueryOptions(queryToken).queryKey,
        }),
      ])
    },
  })

  const reprobeMutation = useMutation({
    mutationFn: async () => {
      if (!token) {
        throw new Error('当前未登录，无法重新探测媒体文件。')
      }

      const primaryFileId = detailAssets.find(
        (asset) => asset.file_ids.length > 0,
      )?.file_ids[0]
      if (!primaryFileId) {
        throw new Error('当前条目没有可重新探测的媒体资产。')
      }

      return createAuthedMiboApi(token).reprobeInventoryFile(primaryFileId)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: catalogItemDetailQueryOptions(queryToken, itemId).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: homeDataQueryOptions(queryToken).queryKey,
        }),
      ])
    },
  })

  const markWatchedMutation = useMutation({
    mutationFn: async () => {
      if (!token) {
        throw new Error('当前未登录，无法更新观看进度。')
      }

      const item = itemQuery.data
      if (!item) {
        throw new Error('媒体详情尚未加载完成。')
      }

      const durationSeconds =
        progressQuery.data?.duration_seconds ?? item.runtime_seconds

      if (!durationSeconds || durationSeconds <= 0) {
        throw new Error('当前媒体缺少时长信息，暂时无法标记为看完。')
      }

      return createAuthedMiboApi(token).updateProgress({
        item_id: itemId,
        asset_id: item.assets?.[0]?.id,
        position_seconds: durationSeconds,
        duration_seconds: durationSeconds,
        completed: true,
      })
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: catalogItemProgressQueryOptions(queryToken, itemId)
            .queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: homeDataQueryOptions(queryToken).queryKey,
        }),
      ])
    },
  })
  const favoriteMutation = useMutation({
    mutationFn: async (favorite: boolean) => {
      if (!token) {
        throw new Error('当前未登录，无法更新收藏。')
      }

      const api = createAuthedMiboApi(token)
      return favorite ? api.addFavorite(itemId) : api.removeFavorite(itemId)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.favorites(queryToken),
        }),
        queryClient.invalidateQueries({
          queryKey: homeDataQueryOptions(queryToken).queryKey,
        }),
      ])
    },
  })

  if (!hasHydrated || (token && itemQuery.isLoading)) {
    return (
      <div className="flex min-h-svh w-full items-center justify-center bg-background text-foreground">
        <div className="flex items-center gap-3 rounded-full border border-border/40 bg-background/80 px-5 py-3 backdrop-blur-xl">
          <LoaderCircleIcon className="size-4 animate-spin" />
          <span className="text-sm text-muted-foreground">
            正在加载媒体详情
          </span>
        </div>
      </div>
    )
  }

  if (!token || !user) {
    return (
      <div className="flex min-h-svh w-full items-center justify-center bg-background px-6 text-foreground">
        <div className="max-w-xl space-y-4 text-center">
          <Badge
            className="border-border/60 bg-background/80"
            variant="outline"
          >
            Mibo Media
          </Badge>
          <h1 className="text-4xl font-semibold tracking-tight">
            登录后查看媒体详情
          </h1>
          <p className="text-sm leading-7 text-muted-foreground sm:text-base">
            当前详情页依赖已登录会话访问后端媒体接口。
          </p>
          <Button asChild size="lg" className="min-w-36">
            <Link to="/login" search={{ redirect: `/media/${itemId}` }}>
              前往登录
            </Link>
          </Button>
        </div>
      </div>
    )
  }

  if (!hasValidItemId) {
    return <MediaDetailError message="无效的媒体 ID。" />
  }

  if (itemQuery.error) {
    return <MediaDetailError message={itemQuery.error.message} />
  }

  if (!itemQuery.data || !presentationItem) {
    return <MediaDetailError message="未找到对应的媒体内容。" />
  }

  const progress = progressQuery.data ?? null
  const durationSeconds =
    progress?.duration_seconds ?? itemQuery.data.runtime_seconds ?? 0
  const itemProgressPercent =
    durationSeconds > 0 && progress
      ? Math.min(
          100,
          Math.max(
            0,
            Math.round((progress.position_seconds / durationSeconds) * 100),
          ),
        )
      : 0
  const mutationErrorMessage =
    rematchMutation.error?.message ||
    reprobeMutation.error?.message ||
    markWatchedMutation.error?.message ||
    favoriteMutation.error?.message
  const isFavorite = Boolean(
    favoritesQuery.data?.some((entry) => entry.item.id === itemId),
  )

  return (
    <div className="relative min-w-0 flex-1 overflow-x-hidden">
      {mutationErrorMessage ? (
        <div className="absolute inset-x-0 top-4 z-30 px-4 sm:px-6 lg:px-8">
          <div className="mx-auto max-w-4xl">
            <Alert>
              <AlertTitle>操作失败</AlertTitle>
              <AlertDescription>{mutationErrorMessage}</AlertDescription>
            </Alert>
          </div>
        </div>
      ) : null}

      <StandaloneMediaDetail
        item={presentedItem ?? presentationItem}
        itemProgressPercent={itemProgressPercent}
        progress={progress}
        seriesSeasons={displayedSeasonRails}
        isSeriesEpisodesLoading={seriesEpisodesQuery.isLoading}
        seriesEpisodesErrorMessage={seriesEpisodesQuery.error?.message ?? null}
        onGoBack={() => {
          if (window.history.length > 1) {
            window.history.back()
            return
          }

          void navigate({ to: '/' })
        }}
        onOpenPlaybackEntry={(options) => {
          void navigate({
            to: '/play/$id',
            params: { id: String(itemId) },
            search: {
              fromStart: Boolean(options?.fromStart),
              assetId: options?.assetId,
            },
          })
        }}
        onOpenAssetPlaybackEntry={(assetId) => {
          void navigate({
            to: '/play/$id',
            params: { id: String(itemId) },
            search: { fromStart: false, assetId },
          })
        }}
        assetChoices={detailAssets}
        onRematchItem={() => {
          void rematchMutation.mutateAsync()
        }}
        onReprobePrimaryFile={() => {
          void reprobeMutation.mutateAsync()
        }}
        isReprobePending={reprobeMutation.isPending}
        onManageMetadata={() => {
          void navigate({
            to: '/settings/metadata/$id',
            params: { id: String(itemId) },
          })
        }}
        onMarkWatched={() => {
          void markWatchedMutation.mutateAsync()
        }}
        isFavorite={isFavorite}
        onFavoriteToggle={(favorite) => {
          void favoriteMutation.mutateAsync(favorite)
        }}
      />
    </div>
  )
}

function MediaDetailError({ message }: { message: string }) {
  return (
    <div className="flex min-h-svh w-full items-center justify-center bg-background px-6 text-foreground">
      <div className="max-w-lg rounded-[2rem] border border-border/40 bg-card/80 p-8 text-center backdrop-blur-xl">
        <Badge className="border-border/60 bg-background/80" variant="outline">
          加载失败
        </Badge>
        <h1 className="mt-4 text-3xl font-semibold tracking-tight">
          媒体详情暂时不可用
        </h1>
        <p className="mt-3 text-sm leading-7 text-muted-foreground">
          {message}
        </p>
      </div>
    </div>
  )
}
