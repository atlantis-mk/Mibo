import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import { LoaderCircleIcon } from "lucide-react"
import { useEffect, useMemo, useState } from "react"

import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert"
import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import { StandaloneMediaDetail } from "#/features/media/components/standalone-media-detail"
import {
  buildPresentedMediaItem,
  catalogEpisodeShelfToSeasonRails,
  metadataItemDetailToPresentation,
  catalogSeasonsToRails,
  type MediaDetailView,
} from "#/lib/media-presentation"
import {
	createAuthedMiboApi,
	favoritesQueryOptions,
	homeDataQueryOptions,
	metadataItemDetailQueryOptions,
	metadataItemProgressQueryOptions,
	metadataItemResourcesQueryOptions,
	miboQueryKeys,
} from "#/lib/mibo-query"
import { useAuthStore } from "#/stores/auth-store"

export default function MediaDetail({
  itemId,
  detailView,
  episodePage,
}: {
  itemId: number
  detailView: MediaDetailView
  episodePage: number
}) {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const queryToken = token ?? "guest"
  const hasValidItemId = Number.isFinite(itemId) && itemId > 0

	const itemQuery = useQuery({
		...metadataItemDetailQueryOptions(queryToken, itemId),
		enabled: hasHydrated && !!token && hasValidItemId,
	})
	const progressQuery = useQuery({
		...metadataItemProgressQueryOptions(queryToken, itemId),
		enabled: hasHydrated && !!token && hasValidItemId,
	})
  const resourcesQuery = useQuery({
    ...metadataItemResourcesQueryOptions(queryToken, itemId),
    enabled: hasHydrated && !!token && hasValidItemId,
  })
  const favoritesQuery = useQuery({
    ...favoritesQueryOptions(queryToken),
    enabled: hasHydrated && !!token && hasValidItemId,
  })
  const detailItem = itemQuery.data
  const detailResources = itemQuery.data?.resources ?? []
  const presentationItem = itemQuery.data
    ? metadataItemDetailToPresentation(itemQuery.data)
    : null
  const detailSeasonRails = catalogSeasonsToRails(detailItem?.seasons ?? [])
  const presentedItem = itemQuery.data
    ? buildPresentedMediaItem(
        presentationItem ?? metadataItemDetailToPresentation(itemQuery.data),
        detailSeasonRails,
        detailView
      )
    : null
  const displayedSeasonRails = presentedItem
    ? presentedItem.type === "episode"
      ? catalogEpisodeShelfToSeasonRails(presentedItem)
      : detailSeasonRails
    : []
  const selectedResourceId = useMemo(() => {
    const preferredResourceId = progressQuery.data?.preferred_resource_id
    if (typeof preferredResourceId === "number") {
      return preferredResourceId
    }
    return resourcesQuery.data?.[0]?.id
  }, [progressQuery.data?.preferred_resource_id, resourcesQuery.data])
  const [selectedResourceIdState, setSelectedResourceIdState] = useState<
    number | undefined
  >(selectedResourceId)
  const [hasUserSelectedResource, setHasUserSelectedResource] = useState(false)

  useEffect(() => {
    setHasUserSelectedResource(false)
    setSelectedResourceIdState(undefined)
  }, [itemId])

  useEffect(() => {
    if (hasUserSelectedResource) {
      return
    }
    setSelectedResourceIdState(selectedResourceId)
  }, [hasUserSelectedResource, selectedResourceId])

  const reprobeMutation = useMutation({
    mutationFn: async () => {
      if (!token) {
        throw new Error("当前未登录，无法重新探测媒体文件。")
      }

      const primaryFileId = detailResources.find(
        (resource) => resource.file_ids.length > 0
      )?.file_ids[0]
      if (!primaryFileId) {
        throw new Error("当前条目没有可重新探测的媒体资源。")
      }

      return createAuthedMiboApi(token).reprobeInventoryFile(primaryFileId)
    },
    onSuccess: async () => {
		await Promise.all([
			queryClient.invalidateQueries({
				queryKey: metadataItemDetailQueryOptions(queryToken, itemId).queryKey,
			}),
        queryClient.invalidateQueries({
          queryKey: metadataItemResourcesQueryOptions(queryToken, itemId)
            .queryKey,
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
        throw new Error("当前未登录，无法更新观看进度。")
      }

      const item = itemQuery.data
      if (!item) {
        throw new Error("媒体详情尚未加载完成。")
      }

      const durationSeconds =
        progressQuery.data?.duration_seconds ?? item.runtime_seconds

      if (!durationSeconds || durationSeconds <= 0) {
        throw new Error("当前媒体缺少时长信息，暂时无法标记为看完。")
      }

		return createAuthedMiboApi(token).updateProgress({
			metadata_item_id: item.metadata_item_id,
			resource_id: selectedResourceIdState,
			position_seconds: durationSeconds,
        duration_seconds: durationSeconds,
        completed: true,
      })
    },
    onSuccess: async () => {
		await Promise.all([
			queryClient.invalidateQueries({
				queryKey: metadataItemProgressQueryOptions(queryToken, itemId)
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
        throw new Error("当前未登录，无法更新收藏。")
      }

      const api = createAuthedMiboApi(token)
		const metadataItemId = itemQuery.data?.metadata_item_id
		if (typeof metadataItemId !== "number") {
		  throw new Error("当前媒体暂不支持收藏。")
		}
      return favorite
        ? api.addFavorite(metadataItemId)
        : api.removeFavorite(metadataItemId)
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
  const preferredResourceMutation = useMutation({
    mutationFn: async (resourceId: number) => {
      if (!token) {
        throw new Error("当前未登录，无法切换播放版本。")
      }
      return createAuthedMiboApi(token).setPreferredResource({
        metadata_item_id: itemId,
        resource_id: resourceId,
      })
    },
    onMutate: async (resourceId) => {
      setHasUserSelectedResource(true)
      setSelectedResourceIdState(resourceId)
      const progressQueryKey = metadataItemProgressQueryOptions(
        queryToken,
        itemId
      ).queryKey
      await queryClient.cancelQueries({ queryKey: progressQueryKey })
      const previous = queryClient.getQueryData(progressQueryKey)
      queryClient.setQueryData(progressQueryKey, (current) => {
        if (!current || typeof current !== "object") {
          return {
            user_id: user?.id ?? 0,
            metadata_item_id: itemId,
            resource_id: resourceId,
            preferred_resource_id: resourceId,
            position_seconds: 0,
            watched: false,
          }
        }
        return {
          ...current,
          resource_id: resourceId,
          preferred_resource_id: resourceId,
        }
      })
      return { previous, progressQueryKey }
    },
    onSuccess: async (state) => {
      queryClient.setQueryData(
        metadataItemProgressQueryOptions(queryToken, itemId).queryKey,
        state
      )
      await queryClient.invalidateQueries({
        queryKey: metadataItemProgressQueryOptions(queryToken, itemId).queryKey,
      })
    },
    onError: (_error, _resourceId, context) => {
      setHasUserSelectedResource(false)
      setSelectedResourceIdState(
        typeof selectedResourceId === "number" ? selectedResourceId : resourcesQuery.data?.[0]?.id
      )
      if (context?.previous) {
        queryClient.setQueryData(context.progressQueryKey, context.previous)
      }
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
          <Button asChild size="lg">
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
            Math.round((progress.position_seconds / durationSeconds) * 100)
          )
        )
      : 0
  const mutationErrorMessage =
    reprobeMutation.error?.message ||
    markWatchedMutation.error?.message ||
    preferredResourceMutation.error?.message ||
    favoriteMutation.error?.message
  const isFavorite = Boolean(
    favoritesQuery.data?.some(
      (entry) =>
		entry.item.metadata_item_id === itemQuery.data.metadata_item_id
	    )
  )
  const itemQueryError = itemQuery.error as unknown
  const seriesEpisodesErrorMessage =
    itemQueryError &&
    typeof itemQueryError === "object" &&
    "message" in itemQueryError &&
    typeof itemQueryError.message === "string"
      ? itemQueryError.message
      : null

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
        episodePage={episodePage}
        isSeriesEpisodesLoading={itemQuery.isLoading}
        seriesEpisodesErrorMessage={seriesEpisodesErrorMessage}
        onGoBack={() => {
          if (window.history.length > 1) {
            window.history.back()
            return
          }

          void navigate({ to: "/" })
        }}
        onOpenPlaybackEntry={(options) => {
          const playbackItemId = options?.itemId ?? itemId
          void navigate({
            to: "/play/$id",
            params: { id: String(playbackItemId) },
            search: {
              fromStart: Boolean(options?.fromStart),
              resourceId: options?.resourceId,
            },
          })
        }}
        resourceChoices={resourcesQuery.data ?? []}
        resourceSummaries={detailResources}
        selectedResourceId={selectedResourceIdState}
        onSelectResource={(resourceId) => {
          void preferredResourceMutation.mutateAsync(resourceId)
        }}
        isSelectingResource={preferredResourceMutation.isPending}
        onReprobePrimaryFile={() => {
          void reprobeMutation.mutateAsync()
        }}
        isReprobePending={reprobeMutation.isPending}
        onManageMetadata={() => {
          void navigate({
            to: "/settings/metadata/$id",
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
