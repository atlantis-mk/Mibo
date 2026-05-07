import { useEffect, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import {
  CheckCircle2Icon,
  LoaderCircleIcon,
  RefreshCwIcon,
  WandSparklesIcon,
} from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert"
import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "#/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "#/components/ui/field"
import { Input } from "#/components/ui/input"
import { Separator } from "#/components/ui/separator"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "#/components/ui/select"
import type {
  CatalogGovernanceWorkspace,
  ManualSeriesRestructureResult,
  MetadataSearchCandidate,
} from "#/lib/mibo-api"
import {
  catalogGovernanceWorkspaceQueryOptions,
  createAuthedMiboApi,
  miboQueryKeys,
} from "#/lib/mibo-query"

import { CandidatePreviewCard } from "./detail-sections"
import {
  ArtworkCard,
  AssetLinksCard,
  AsyncActionsCard,
  CandidateSearchCard,
  ClassificationReviewCard,
  DraftEditorCard,
  FieldLocksCard,
  ImageCandidatesCard,
  MetadataSummaryCard,
  RelatedChildrenCard,
  SourceEvidenceCard,
} from "./detail-panels"
import { formatMatchStatus, formatMediaType } from "./formatters"

type MetadataDraft = {
  title: string
  originalTitle: string
  year: string
  overview: string
}

type AsyncActionState = {
  type: "rematch" | "refetch" | "reprobe"
  status: "queued" | "running" | "completed" | "failed"
  message: string
}

type OperationDialog =
  | "metadata"
  | "matching"
  | "actions"
  | "locks"
  | "images"
  | "assets"
  | "restructure"
  | null

type ManualSeriesEpisodeDraft = {
  assetId: number
  fileId?: number
  storagePath: string
  seasonNumber: string
  episodeNumber: string
  episodeTitle: string
}

type ManualSeriesDraft = {
  rootPath: string
  seriesTitle: string
  seasonNumber: string
  migrateMetadata: boolean
  episodes: ManualSeriesEpisodeDraft[]
}

type ManualMovieVersionsDraft = {
  action: "movie_versions" | "independent_movies"
  rootPath: string
  title: string
}

const EMPTY_DRAFT: MetadataDraft = {
  title: "",
  originalTitle: "",
  year: "",
  overview: "",
}

export function MetadataGovernanceDetail({
  token,
  itemId,
}: {
  token: string
  itemId: number
}) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const workspaceQueryKey = miboQueryKeys.catalogGovernanceWorkspace(
    token,
    itemId
  )
  const listWorkspaceQueryKey = miboQueryKeys.metadataWorkspace(token)
  const workspaceQuery = useQuery({
    ...catalogGovernanceWorkspaceQueryOptions(token, itemId),
  })

  const [draft, setDraft] = useState<MetadataDraft>(EMPTY_DRAFT)
  const [baselineDraft, setBaselineDraft] = useState<MetadataDraft>(EMPTY_DRAFT)
  const [searchTitle, setSearchTitle] = useState("")
  const [searchYear, setSearchYear] = useState("")
  const [searchIMDbId, setSearchIMDbId] = useState("")
  const [searchTMDBId, setSearchTMDBId] = useState("")
  const [searchTVDBId, setSearchTVDBId] = useState("")
  const [candidatePreview, setCandidatePreview] =
    useState<MetadataSearchCandidate | null>(null)
  const [operationDialog, setOperationDialog] = useState<OperationDialog>(null)
  const [manualSeriesDraft, setManualSeriesDraft] = useState<ManualSeriesDraft>(
    () => emptyManualSeriesDraft()
  )
  const [manualSeriesPreview, setManualSeriesPreview] =
    useState<ManualSeriesRestructureResult | null>(null)
  const [manualMovieVersionsDraft, setManualMovieVersionsDraft] =
    useState<ManualMovieVersionsDraft>(() => emptyManualMovieVersionsDraft())
  const [asyncActionState, setAsyncActionState] =
    useState<AsyncActionState | null>(null)
  const [saveSuccessMessage, setSaveSuccessMessage] = useState("")

  useEffect(() => {
    if (!workspaceQuery.data) return

    const nextDraft = buildDraftFromWorkspace(workspaceQuery.data)
    setDraft(nextDraft)
    setBaselineDraft(nextDraft)
    setSearchTitle(workspaceQuery.data.title)
    setSearchYear(
      fieldStateNumber(workspaceQuery.data, "year")
        ? String(fieldStateNumber(workspaceQuery.data, "year"))
        : ""
    )
    setSearchIMDbId("")
    setSearchTMDBId("")
    setSearchTVDBId("")
    setManualSeriesDraft(buildManualSeriesDraft(workspaceQuery.data))
    setManualMovieVersionsDraft(
      buildManualMovieVersionsDraft(workspaceQuery.data)
    )
    setManualSeriesPreview(null)
  }, [workspaceQuery.data?.item_id])

  const isDirty = JSON.stringify(draft) !== JSON.stringify(baselineDraft)

  useEffect(() => {
    if (saveSuccessMessage && isDirty) {
      setSaveSuccessMessage("")
    }
  }, [isDirty, saveSuccessMessage])

  useEffect(() => {
    if (!isDirty) return

    function handleBeforeUnload(event: BeforeUnloadEvent) {
      event.preventDefault()
      event.returnValue = ""
    }

    window.addEventListener("beforeunload", handleBeforeUnload)
    return () => window.removeEventListener("beforeunload", handleBeforeUnload)
  }, [isDirty])

  useEffect(() => {
    if (!isDirty) return

    function handleDocumentClick(event: MouseEvent) {
      const target = event.target
      if (!(target instanceof Element)) return

      const anchor = target.closest("a[href]")
      if (!(anchor instanceof HTMLAnchorElement)) return
      if (
        anchor.target === "_blank" ||
        anchor.hasAttribute("download") ||
        event.metaKey ||
        event.ctrlKey ||
        event.shiftKey ||
        event.altKey
      ) {
        return
      }

      const destination = new URL(anchor.href, window.location.href)
      const current = new URL(window.location.href)
      const isSameDocumentNavigation =
        destination.origin === current.origin &&
        destination.pathname === current.pathname &&
        destination.search === current.search &&
        destination.hash === current.hash
      if (isSameDocumentNavigation) return

      if (!window.confirm("当前有未保存修改，确认离开治理页吗？")) {
        event.preventDefault()
        event.stopPropagation()
      }
    }

    document.addEventListener("click", handleDocumentClick, true)
    return () =>
      document.removeEventListener("click", handleDocumentClick, true)
  }, [isDirty])

  const searchMutation = useMutation({
    mutationFn: () =>
      createAuthedMiboApi(token).searchCatalogItemMetadata(itemId, {
        title: searchTitle.trim() || undefined,
        year: parseOptionalNumber(searchYear),
        imdb_id: searchIMDbId.trim() || undefined,
        tmdb_id: searchTMDBId.trim() || undefined,
        tvdb_id: searchTVDBId.trim() || undefined,
      }),
  })

  const saveDraftMutation = useMutation({
    mutationFn: async () => {
      const api = createAuthedMiboApi(token)
      const updates = [
        { field_key: "title", value: draft.title.trim() },
        {
          field_key: "original_title",
          value: draft.originalTitle.trim(),
        },
        {
          field_key: "year",
          value: parseOptionalNumber(draft.year),
        },
        {
          field_key: "overview",
          value: draft.overview.trim(),
        },
      ]

      for (const update of updates) {
        if (
          update.field_key !== "title" &&
          (update.value === undefined || update.value === "")
        ) {
          continue
        }
        await api.updateCatalogGovernanceField(itemId, {
          field_key: update.field_key,
          value: update.value ?? "",
        })
      }

      return api.getCatalogGovernanceWorkspace(itemId)
    },
    onSuccess: async (workspace) => {
      const nextDraft = buildDraftFromWorkspace(workspace)
      setDraft(nextDraft)
      setBaselineDraft(nextDraft)
      setSaveSuccessMessage("草稿已保存，治理页和媒体详情将使用最新元数据。")
      queryClient.setQueryData(workspaceQueryKey, workspace)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.homeData(token),
        }),
      ])
    },
  })

  const applyCandidateMutation = useMutation({
    mutationFn: (externalId: string) =>
      createAuthedMiboApi(token).applyCatalogItemMetadataCandidate(itemId, {
        external_id: externalId,
      }),
    onSuccess: async (workspace) => {
      const nextDraft = buildDraftFromWorkspace(workspace)
      setDraft(nextDraft)
      setBaselineDraft(nextDraft)
      setCandidatePreview(null)
      setSaveSuccessMessage("候选结果已应用，当前治理草稿已同步为最新元数据。")
      queryClient.setQueryData(workspaceQueryKey, workspace)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.homeData(token),
        }),
      ])
    },
  })

  const rematchMutation = useMutation({
    mutationFn: () => createAuthedMiboApi(token).matchCatalogItem(itemId),
    onSuccess: async (workspace) => {
      const nextDraft = buildDraftFromWorkspace(workspace)
      setDraft(nextDraft)
      setBaselineDraft(nextDraft)
      setAsyncActionState({
        type: "rematch",
        status: "completed",
        message: "重新匹配已完成，治理结果已刷新。",
      })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
      ])
    },
  })

  const refetchMutation = useMutation({
    mutationFn: () =>
      createAuthedMiboApi(token).refetchCatalogItemMetadata(itemId),
    onSuccess: async (workspace) => {
      const nextDraft = buildDraftFromWorkspace(workspace)
      setDraft(nextDraft)
      setBaselineDraft(nextDraft)
      setAsyncActionState({
        type: "refetch",
        status: "completed",
        message: "元数据重抓已完成，来源证据和字段值已刷新。",
      })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
      ])
    },
  })

  const reprobeMutation = useMutation({
    mutationFn: (inventoryFileId: number) => {
      if (!inventoryFileId) {
        throw new Error("当前条目没有可重新探测的库存文件。")
      }
      return createAuthedMiboApi(token).reprobeInventoryFile(inventoryFileId)
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: workspaceQueryKey })
      setAsyncActionState({
        type: "reprobe",
        status: "completed",
        message: "重新探测已提交，资产状态会在后台刷新。",
      })
    },
  })

  const lockMutation = useMutation({
    mutationFn: ({
      fieldKey,
      nextLocked,
    }: {
      fieldKey: string
      nextLocked: boolean
    }) =>
      createAuthedMiboApi(token).updateCatalogGovernanceField(itemId, {
        field_key: fieldKey,
        value: fieldStateValue(workspaceQuery.data, fieldKey) ?? "",
        lock: nextLocked,
        lock_reason: nextLocked ? "governance ui" : "",
        force: true,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: workspaceQueryKey })
    },
  })

  const imageMutation = useMutation({
    mutationFn: ({ imageType, url }: { imageType: string; url: string }) =>
      createAuthedMiboApi(token).selectCatalogGovernanceImage(itemId, {
        image_type: imageType,
        url,
      }),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
      ])
    },
  })

  const assetLinkMutation = useMutation({
    mutationFn: async ({
      assetId,
      targetItemId,
      mode,
    }: {
      assetId: number
      targetItemId: number
      mode: "link" | "unlink"
    }) => {
      const api = createAuthedMiboApi(token)
      return mode === "link"
        ? api.linkCatalogGovernanceAsset(itemId, assetId, {
            target_item_id: targetItemId,
          })
        : api.unlinkCatalogGovernanceAsset(itemId, assetId, targetItemId)
    },
    onSuccess: async (workspace, variables) => {
      setSaveSuccessMessage(
        variables.mode === "link"
          ? "资产链接已更新，治理工作区已刷新。"
          : "资产链接已解除，治理工作区已刷新。"
      )
      queryClient.setQueryData(workspaceQueryKey, workspace)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
      ])
    },
  })

  const manualSeriesPreviewMutation = useMutation({
    mutationFn: () => {
      const workspace = workspaceQuery.data
      if (!workspace) throw new Error("治理工作区尚未加载。")
      return createAuthedMiboApi(token).previewManualSeriesRestructure(
        workspace.library_id,
        manualSeriesDraftToInput(manualSeriesDraft)
      )
    },
    onSuccess: (result) => {
      setManualSeriesPreview(result)
    },
  })

  const manualSeriesApplyMutation = useMutation({
    mutationFn: () => {
      const workspace = workspaceQuery.data
      if (!workspace) throw new Error("治理工作区尚未加载。")
      return createAuthedMiboApi(token).applyManualSeriesRestructure(
        workspace.library_id,
        manualSeriesDraftToInput(manualSeriesDraft)
      )
    },
    onSuccess: async (result) => {
      setManualSeriesPreview(result)
      setSaveSuccessMessage(
        `已重组为剧集：${(result.series?.title ?? manualSeriesDraft.seriesTitle) || "未命名剧集"}。`
      )
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.homeData(token),
        }),
      ])
    },
  })

  const movieVersionsCorrectionMutation = useMutation({
    mutationFn: () =>
      createAuthedMiboApi(token).applyCatalogGovernanceClassificationCorrection(
        itemId,
        {
          action: manualMovieVersionsDraft.action,
          root_path: manualMovieVersionsDraft.rootPath.trim() || undefined,
          title: manualMovieVersionsDraft.title.trim() || undefined,
        }
      ),
    onSuccess: async (workspace) => {
      const actionLabel =
        manualMovieVersionsDraft.action === "independent_movies"
          ? "多部电影"
          : "电影多版本"
      setSaveSuccessMessage(
        `已重组为${actionLabel}：${workspace.title || "未命名电影"}。`
      )
      queryClient.setQueryData(
        miboQueryKeys.catalogGovernanceWorkspace(token, workspace.item_id),
        workspace
      )
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, workspace.item_id),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.homeData(token),
        }),
      ])
      await navigate({
        to: "/settings/metadata/$id",
        params: { id: String(workspace.item_id) },
      })
    },
  })

  if (workspaceQuery.isLoading) {
    return (
      <div className="flex items-center gap-3 rounded-[1.5rem] border border-border/60 bg-card/80 px-5 py-4 text-foreground shadow-sm">
        <LoaderCircleIcon className="size-4 animate-spin" />
        <span className="text-sm text-muted-foreground">正在加载治理页</span>
      </div>
    )
  }

  if (workspaceQuery.error || !workspaceQuery.data) {
    return (
      <div className="rounded-[1.75rem] border border-border/60 bg-card/80 px-6 py-8 text-foreground shadow-sm">
        <div className="max-w-xl space-y-4">
          <h1 className="text-2xl font-semibold tracking-tight">
            治理页暂时不可用
          </h1>
          <p className="text-sm text-muted-foreground">
            {workspaceQuery.error?.message ?? "未找到对应治理工作区。"}
          </p>
          <Button asChild variant="outline">
            <Link to="/settings/metadata">返回治理工作台</Link>
          </Button>
        </div>
      </div>
    )
  }

  const workspace = workspaceQuery.data
  const workspaceAssets = workspace.assets ?? []
  const workspaceFieldStates = workspace.field_states ?? []
  const workspaceSourceEvidence = workspace.source_evidence ?? []
  const workspaceClassification = workspace.classification_decisions ?? []
  const workspaceSelectedImages = workspace.selected_images ?? []
  const workspaceImageCandidates = workspace.image_candidates ?? []
  const workspaceRecommendedChildren = workspace.recommended_children ?? []
  const item = buildPreviewItem(workspace)
  const activeCandidates = uniqueMetadataCandidates(searchMutation.data ?? [])
  const firstInventoryFileId = workspaceAssets.find(
    (asset) => (asset.file_ids ?? []).length > 0
  )?.file_ids[0]

  async function handleNavigateAway(
    to: "/" | "/settings/metadata" | "/media/$id"
  ) {
    if (isDirty && !window.confirm("当前有未保存修改，确认离开治理页吗？")) {
      return
    }

    if (to === "/media/$id") {
      await navigate({
        to,
        params: { id: String(itemId) },
        search: { view: undefined },
      })
      return
    }

    await navigate({ to })
  }

  return (
    <>
      <div className="space-y-4 text-foreground">
        <div className="flex flex-col gap-4 rounded-[1.75rem] border border-border/60 bg-card/80 p-5 shadow-sm backdrop-blur-sm lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge
                variant="outline"
                className="border-border/60 bg-background/70"
              >
                单条目治理
              </Badge>
              <Badge variant="secondary">
                {formatMediaType(workspace.type)}
              </Badge>
              <Badge
                variant="outline"
                className="border-border/60 bg-background/70"
              >
                {formatMatchStatus(workspace.governance_status)}
              </Badge>
            </div>
            <div>
              <h1 className="text-3xl font-semibold tracking-tight">
                {workspace.title}
              </h1>
              <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
                页面主体仅展示当前元数据、来源证据、图片、分类和资产关系；需要修改时使用右侧操作按钮打开弹窗处理。
              </p>
            </div>
          </div>

          <div className="flex flex-wrap gap-2 lg:justify-end">
            <Button
              variant="outline"
              className="border-border/60 bg-background/70"
              onClick={() => void handleNavigateAway("/settings/metadata")}
            >
              返回工作台
            </Button>
            <Button
              variant="outline"
              className="border-border/60 bg-background/70"
              onClick={() => void handleNavigateAway("/media/$id")}
            >
              查看详情页
            </Button>
            <Button onClick={() => setOperationDialog("metadata")}>操作</Button>
          </div>
        </div>

        {isDirty ? (
          <Alert>
            <WandSparklesIcon className="size-4" />
            <AlertTitle>存在未保存草稿</AlertTitle>
            <AlertDescription>
              离开当前页面前会要求确认。保存后会同步刷新治理页、媒体详情和工作台摘要。
            </AlertDescription>
          </Alert>
        ) : null}

        {saveSuccessMessage ? (
          <Alert>
            <CheckCircle2Icon className="size-4" />
            <AlertTitle>保存成功</AlertTitle>
            <AlertDescription>{saveSuccessMessage}</AlertDescription>
          </Alert>
        ) : null}

        {asyncActionState ? (
          <Alert>
            {asyncActionState.status === "failed" ? (
              <WandSparklesIcon className="size-4" />
            ) : asyncActionState.status === "completed" ? (
              <CheckCircle2Icon className="size-4" />
            ) : (
              <LoaderCircleIcon className="size-4 animate-spin" />
            )}
            <AlertTitle>{formatAsyncActionTitle(asyncActionState)}</AlertTitle>
            <AlertDescription>{asyncActionState.message}</AlertDescription>
          </Alert>
        ) : null}

        {searchMutation.error ||
        saveDraftMutation.error ||
        applyCandidateMutation.error ||
        rematchMutation.error ||
        refetchMutation.error ||
        reprobeMutation.error ||
        lockMutation.error ||
        imageMutation.error ||
        assetLinkMutation.error ||
        manualSeriesPreviewMutation.error ||
        manualSeriesApplyMutation.error ||
        movieVersionsCorrectionMutation.error ? (
          <Alert>
            <AlertTitle>操作失败</AlertTitle>
            <AlertDescription>
              {searchMutation.error?.message ||
                saveDraftMutation.error?.message ||
                applyCandidateMutation.error?.message ||
                rematchMutation.error?.message ||
                refetchMutation.error?.message ||
                reprobeMutation.error?.message ||
                lockMutation.error?.message ||
                imageMutation.error?.message ||
                assetLinkMutation.error?.message ||
                manualSeriesPreviewMutation.error?.message ||
                manualSeriesApplyMutation.error?.message ||
                movieVersionsCorrectionMutation.error?.message}
            </AlertDescription>
          </Alert>
        ) : null}

        <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
          <CardHeader className="px-5 py-5">
            <CardTitle>可用操作</CardTitle>
            <CardDescription>
              需要变更元数据、候选匹配、字段锁、图片、资产或重组结构时，从这里打开弹窗。
            </CardDescription>
          </CardHeader>
          <Separator className="bg-border" />
          <CardContent className="flex flex-wrap gap-2 px-5 py-5">
            <Button onClick={() => setOperationDialog("metadata")}>
              编辑元数据
            </Button>
            <Button
              variant="outline"
              className="border-border/60 bg-background/70"
              onClick={() => setOperationDialog("matching")}
            >
              匹配候选
            </Button>
            <Button
              variant="outline"
              className="border-border/60 bg-background/70"
              onClick={() => setOperationDialog("actions")}
            >
              后台动作
            </Button>
            <Button
              variant="outline"
              className="border-border/60 bg-background/70"
              onClick={() => setOperationDialog("locks")}
            >
              字段锁
            </Button>
            <Button
              variant="outline"
              className="border-border/60 bg-background/70"
              onClick={() => setOperationDialog("images")}
            >
              图片选择
            </Button>
            <Button
              variant="outline"
              className="border-border/60 bg-background/70"
              onClick={() => setOperationDialog("assets")}
            >
              资产链接
            </Button>
            <Button
              variant="outline"
              className="border-border/60 bg-background/70"
              onClick={() => setOperationDialog("restructure")}
            >
              手动重组
            </Button>
          </CardContent>
        </Card>

        <div className="space-y-4">
          <MetadataSummaryCard item={item} />

          <div className="grid gap-4 lg:grid-cols-2">
            <ArtworkCard
              posterUrl={item.poster_url}
              backdropUrl={item.backdrop_url}
            />
            <SourceEvidenceCard sourceEvidence={workspaceSourceEvidence} />
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <ClassificationReviewCard decisions={workspaceClassification} />
            <RelatedChildrenCard
              workspace={workspace}
              assets={workspaceAssets}
            />
          </div>
        </div>
      </div>

      <Dialog
        open={operationDialog !== null}
        onOpenChange={(open) => !open && setOperationDialog(null)}
      >
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-5xl">
          <DialogHeader>
            <DialogTitle>{operationDialogTitle(operationDialog)}</DialogTitle>
            <DialogDescription>
              操作内容集中在弹窗内处理，关闭后详情页继续保持信息展示视图。
            </DialogDescription>
          </DialogHeader>

          {operationDialog === "metadata" ? (
            <DraftEditorCard
              draft={draft}
              baselineDraft={baselineDraft}
              isDirty={isDirty}
              isPending={saveDraftMutation.isPending}
              onDraftChange={(updater) => setDraft(updater)}
              onReset={setDraft}
              onSave={() => void saveDraftMutation.mutateAsync()}
            />
          ) : null}

          {operationDialog === "matching" ? (
            <CandidateSearchCard
              searchTitle={searchTitle}
              searchYear={searchYear}
              searchIMDbId={searchIMDbId}
              searchTMDBId={searchTMDBId}
              searchTVDBId={searchTVDBId}
              isPending={searchMutation.isPending}
              isSuccess={searchMutation.isSuccess}
              activeCandidates={activeCandidates}
              onSearchTitleChange={setSearchTitle}
              onSearchYearChange={setSearchYear}
              onSearchIMDbIdChange={setSearchIMDbId}
              onSearchTMDBIdChange={setSearchTMDBId}
              onSearchTVDBIdChange={setSearchTVDBId}
              onSearch={() => void searchMutation.mutateAsync()}
              onPreview={setCandidatePreview}
            />
          ) : null}

          {operationDialog === "actions" ? (
            <AsyncActionsCard
              rematchPending={rematchMutation.isPending}
              refetchPending={refetchMutation.isPending}
              reprobePending={reprobeMutation.isPending}
              reprobeDisabled={!firstInventoryFileId}
              onRematch={() => void rematchMutation.mutateAsync()}
              onRefetch={() => void refetchMutation.mutateAsync()}
              onReprobe={() => {
                if (!firstInventoryFileId) return
                void reprobeMutation.mutateAsync(firstInventoryFileId)
              }}
            />
          ) : null}

          {operationDialog === "locks" ? (
            <FieldLocksCard
              fieldStates={workspaceFieldStates}
              isPending={lockMutation.isPending}
              onToggleLock={(fieldKey, nextLocked) => {
                void lockMutation.mutateAsync({ fieldKey, nextLocked })
              }}
            />
          ) : null}

          {operationDialog === "images" ? (
            <ImageCandidatesCard
              selectedImages={workspaceSelectedImages}
              imageCandidates={workspaceImageCandidates}
              isPending={imageMutation.isPending}
              onSelect={(imageType, url) => {
                void imageMutation.mutateAsync({ imageType, url })
              }}
            />
          ) : null}

          {operationDialog === "assets" ? (
            <AssetLinksCard
              workspaceItem={{
                id: workspace.item_id,
                title: workspace.title,
                type: workspace.type,
                availability_status: workspace.availability_status,
                governance_status: workspace.governance_status,
              }}
              relatedChildren={workspaceRecommendedChildren}
              assets={workspaceAssets}
              reprobePendingFileId={
                typeof reprobeMutation.variables === "number"
                  ? reprobeMutation.variables
                  : undefined
              }
              linkMutation={assetLinkMutation.variables}
              onReprobe={(fileId) => {
                void reprobeMutation.mutateAsync(fileId)
              }}
              onLink={(assetId, targetItemId) => {
                void assetLinkMutation.mutateAsync({
                  assetId,
                  targetItemId,
                  mode: "link",
                })
              }}
              onUnlink={(assetId, targetItemId) => {
                void assetLinkMutation.mutateAsync({
                  assetId,
                  targetItemId,
                  mode: "unlink",
                })
              }}
            />
          ) : null}

          {operationDialog === "restructure" ? (
            <div className="space-y-4">
              {isSeriesLikeWorkspace(workspace) ? (
                <ManualMovieVersionsCard
                  workspace={workspace}
                  draft={manualMovieVersionsDraft}
                  isPending={movieVersionsCorrectionMutation.isPending}
                  onDraftChange={setManualMovieVersionsDraft}
                  onApply={() => {
                    const actionLabel =
                      manualMovieVersionsDraft.action === "independent_movies"
                        ? "多部独立电影"
                        : "一部电影的多版本"
                    if (
                      !window.confirm(
                        `确认把当前剧集范围下的视频资产重组为${actionLabel}？原剧集、季和集会被标记为手动退役。`
                      )
                    ) {
                      return
                    }
                    void movieVersionsCorrectionMutation.mutateAsync()
                  }}
                />
              ) : (
                <ManualSeriesRestructureCard
                  draft={manualSeriesDraft}
                  preview={manualSeriesPreview}
                  isPreviewPending={manualSeriesPreviewMutation.isPending}
                  isApplyPending={manualSeriesApplyMutation.isPending}
                  onDraftChange={setManualSeriesDraft}
                  onPreview={() =>
                    void manualSeriesPreviewMutation.mutateAsync()
                  }
                  onApply={() => {
                    if (
                      !window.confirm(
                        "确认把该路径下已扫描资产重组为剧集？原电影条目会被标记为手动退役。"
                      )
                    ) {
                      return
                    }
                    void manualSeriesApplyMutation.mutateAsync()
                  }}
                />
              )}
            </div>
          ) : null}
        </DialogContent>
      </Dialog>

      <Dialog
        open={candidatePreview !== null}
        onOpenChange={(open) => !open && setCandidatePreview(null)}
      >
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>候选差异预览</DialogTitle>
            <DialogDescription>
              预览当前条目与候选元数据的关键差异后，再确认应用。
            </DialogDescription>
          </DialogHeader>

          {candidatePreview ? (
            <div className="grid gap-4 lg:grid-cols-2">
              <CandidatePreviewCard title="当前条目" item={item} />
              <CandidatePreviewCard
                title="候选结果"
                candidate={candidatePreview}
              />
            </div>
          ) : null}

          <DialogFooter>
            <Button variant="outline" onClick={() => setCandidatePreview(null)}>
              取消
            </Button>
            <Button
              onClick={() => {
                if (!candidatePreview) return
                void applyCandidateMutation.mutateAsync(
                  candidatePreview.external_id
                )
              }}
              disabled={!candidatePreview || applyCandidateMutation.isPending}
            >
              {applyCandidateMutation.isPending ? (
                <LoaderCircleIcon className="size-4 animate-spin" />
              ) : null}
              确认应用候选
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

function ManualMovieVersionsCard({
  workspace,
  draft,
  isPending,
  onDraftChange,
  onApply,
}: {
  workspace: CatalogGovernanceWorkspace
  draft: ManualMovieVersionsDraft
  isPending: boolean
  onDraftChange: (draft: ManualMovieVersionsDraft) => void
  onApply: () => void
}) {
  const assetCount = workspace.assets?.length ?? 0
  const fileCount =
    workspace.assets?.reduce(
      (total, asset) => total + (asset.files?.length ?? 0),
      0
    ) ?? 0
  const isIndependentMovies = draft.action === "independent_movies"

  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>手动重组为电影</CardTitle>
        <CardDescription>
          当前条目是剧集结构时，可把它下面的视频资产重新挂到一部电影，并把其余文件作为版本。
        </CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-4 px-5 py-5">
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="manual-movie-action">重组方式</FieldLabel>
            <Select
              value={draft.action}
              onValueChange={(value) =>
                onDraftChange({
                  ...draft,
                  action: value as ManualMovieVersionsDraft["action"],
                })
              }
            >
              <SelectTrigger id="manual-movie-action" className="w-full">
                <SelectValue placeholder="选择重组方式" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="movie_versions">
                  合并为一部电影的多版本
                </SelectItem>
                <SelectItem value="independent_movies">
                  拆成多部独立电影
                </SelectItem>
              </SelectContent>
            </Select>
          </Field>
          <Field>
            <FieldLabel htmlFor="manual-movie-root">电影路径</FieldLabel>
            <Input
              id="manual-movie-root"
              value={draft.rootPath}
              onChange={(event) =>
                onDraftChange({ ...draft, rootPath: event.target.value })
              }
              placeholder="/测试/pacopacomama-011919_016-FHD"
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="manual-movie-title">电影名称</FieldLabel>
            <Input
              id="manual-movie-title"
              value={draft.title}
              onChange={(event) =>
                onDraftChange({ ...draft, title: event.target.value })
              }
              placeholder="留空则使用当前标题或路径最后一段"
            />
          </Field>
        </FieldGroup>
        <div className="rounded-[1rem] border border-border/60 bg-background/60 p-4 text-sm">
          <div className="font-medium text-foreground">将执行的修复</div>
          <div className="mt-2 space-y-1 text-xs leading-5 text-muted-foreground">
            <div>
              来源：{formatMediaType(workspace.type)} · {workspace.title}
            </div>
            <div>路径：{draft.rootPath || "未填写"}</div>
            <div>
              资产：{assetCount} 个；文件：{fileCount} 个。
            </div>
            <div>
              结果：创建/复用同路径
              {isIndependentMovies
                ? "下的多部 movie，每个资产作为一部电影的主版本。"
                : " movie，第一个资产设为主版本，其余资产设为多版本。"}
            </div>
          </div>
        </div>
        <Button
          onClick={onApply}
          disabled={!draft.rootPath.trim() || isPending}
        >
          {isPending ? (
            <LoaderCircleIcon className="size-4 animate-spin" />
          ) : null}
          {isIndependentMovies ? "重组为多部电影" : "重组为电影多版本"}
        </Button>
      </CardContent>
    </Card>
  )
}

function operationDialogTitle(dialog: OperationDialog) {
  switch (dialog) {
    case "metadata":
      return "编辑元数据"
    case "matching":
      return "匹配候选"
    case "actions":
      return "后台动作"
    case "locks":
      return "字段锁"
    case "images":
      return "图片选择"
    case "assets":
      return "资产链接"
    case "restructure":
      return "手动重组"
    default:
      return "操作"
  }
}

function buildDraftFromWorkspace(
  workspace: CatalogGovernanceWorkspace
): MetadataDraft {
  const year = fieldStateNumber(workspace, "year")

  return {
    title: workspace.title || "",
    originalTitle: fieldStateString(workspace, "original_title"),
    year: year ? String(year) : "",
    overview: fieldStateString(workspace, "overview"),
  }
}

function buildPreviewItem(workspace: CatalogGovernanceWorkspace) {
  return {
    id: workspace.item_id,
    library_id: workspace.library_id,
    type: workspace.type,
    title: workspace.title,
    original_title: fieldStateString(workspace, "original_title"),
    overview: fieldStateString(workspace, "overview"),
    poster_url: selectedImageUrl(workspace, "poster"),
    backdrop_url: selectedImageUrl(workspace, "backdrop"),
    year: fieldStateNumber(workspace, "year"),
    governance_status: workspace.governance_status,
    availability_status: workspace.availability_status,
    metadata_provider: workspace.external_identities?.[0]?.provider ?? "",
    external_id: workspace.external_identities?.[0]?.external_id ?? "",
  }
}

function ManualSeriesRestructureCard({
  draft,
  preview,
  isPreviewPending,
  isApplyPending,
  onDraftChange,
  onPreview,
  onApply,
}: {
  draft: ManualSeriesDraft
  preview: ManualSeriesRestructureResult | null
  isPreviewPending: boolean
  isApplyPending: boolean
  onDraftChange: (draft: ManualSeriesDraft) => void
  onPreview: () => void
  onApply: () => void
}) {
  const hasAssets = draft.episodes.length > 0

  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>手动重组为剧集</CardTitle>
        <CardDescription>
          按路径把已扫描为电影的资产重新挂到剧集/季/集层级，不修改自动扫描规则。
        </CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-4 px-5 py-5">
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="manual-series-root">剧集根路径</FieldLabel>
            <Input
              id="manual-series-root"
              value={draft.rootPath}
              onChange={(event) =>
                onDraftChange({ ...draft, rootPath: event.target.value })
              }
              placeholder="/我的收藏/10-30(1)/MP4"
            />
          </Field>
          <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_160px]">
            <Field>
              <FieldLabel htmlFor="manual-series-title">剧名</FieldLabel>
              <Input
                id="manual-series-title"
                value={draft.seriesTitle}
                onChange={(event) =>
                  onDraftChange({ ...draft, seriesTitle: event.target.value })
                }
                placeholder="留空则取根路径最后一段"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="manual-series-season">统一季号</FieldLabel>
              <Input
                id="manual-series-season"
                inputMode="numeric"
                value={draft.seasonNumber}
                onChange={(event) =>
                  onDraftChange({ ...draft, seasonNumber: event.target.value })
                }
                placeholder="自动"
              />
            </Field>
          </div>
        </FieldGroup>

        <label className="flex items-start gap-3 rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3 text-sm">
          <input
            type="checkbox"
            className="mt-1 size-4 rounded border-border"
            checked={draft.migrateMetadata}
            onChange={(event) =>
              onDraftChange({
                ...draft,
                migrateMetadata: event.target.checked,
              })
            }
          />
          <span>
            <span className="font-medium text-foreground">
              迁移封面和元数据
            </span>
            <span className="mt-1 block text-xs leading-5 text-muted-foreground">
              将原 movie
              的选中图片、基础字段、标签、人物和播放进度迁移到新剧集/分集；不会复制外部电影
              ID。
            </span>
          </span>
        </label>

        <div className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="text-sm font-medium text-foreground">集映射</div>
              <div className="text-xs text-muted-foreground">
                默认从最底层文件名提取集名，可逐条覆盖季号、集号和集名。
              </div>
            </div>
            <Badge variant="outline">{draft.episodes.length} 个资产</Badge>
          </div>
          {hasAssets ? (
            <div className="space-y-2">
              {draft.episodes.map((episode, index) => (
                <div
                  key={`${episode.assetId}-${episode.storagePath}`}
                  className="rounded-[1rem] border border-border/60 bg-background/60 p-3"
                >
                  <div className="mb-3 line-clamp-2 text-xs text-muted-foreground">
                    {episode.storagePath}
                  </div>
                  <div className="grid gap-3 md:grid-cols-[90px_90px_minmax(0,1fr)]">
                    <Input
                      inputMode="numeric"
                      value={episode.seasonNumber}
                      onChange={(event) =>
                        onDraftChange(
                          updateManualSeriesEpisode(draft, index, {
                            seasonNumber: event.target.value,
                          })
                        )
                      }
                      placeholder="季"
                    />
                    <Input
                      inputMode="numeric"
                      value={episode.episodeNumber}
                      onChange={(event) =>
                        onDraftChange(
                          updateManualSeriesEpisode(draft, index, {
                            episodeNumber: event.target.value,
                          })
                        )
                      }
                      placeholder="集"
                    />
                    <Input
                      value={episode.episodeTitle}
                      onChange={(event) =>
                        onDraftChange(
                          updateManualSeriesEpisode(draft, index, {
                            episodeTitle: event.target.value,
                          })
                        )
                      }
                      placeholder="集名"
                    />
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="rounded-[1rem] border border-dashed border-border/70 px-4 py-6 text-sm text-muted-foreground">
              当前治理条目没有可用于重组的资产文件。
            </div>
          )}
        </div>

        {preview ? (
          <div className="rounded-[1rem] border border-border/60 bg-background/60 p-4">
            <div className="text-sm font-medium text-foreground">预览结果</div>
            <div className="mt-2 space-y-1 text-xs text-muted-foreground">
              {preview.mappings.map((mapping) => (
                <div key={`${mapping.asset_id}-${mapping.storage_path}`}>
                  S{String(mapping.season_number).padStart(2, "0")}E
                  {String(mapping.episode_number).padStart(2, "0")} ·{" "}
                  {mapping.episode_title} · 资产 {mapping.asset_id}
                </div>
              ))}
            </div>
            {preview.warnings?.length ? (
              <div className="mt-3 text-xs text-amber-600 dark:text-amber-300">
                {preview.warnings.join("；")}
              </div>
            ) : null}
          </div>
        ) : null}

        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            className="border-border/60 bg-background/70"
            onClick={onPreview}
            disabled={!draft.rootPath.trim() || !hasAssets || isPreviewPending}
          >
            {isPreviewPending ? (
              <LoaderCircleIcon className="size-4 animate-spin" />
            ) : (
              <RefreshCwIcon className="size-4" />
            )}
            预览重组
          </Button>
          <Button
            onClick={onApply}
            disabled={!draft.rootPath.trim() || !hasAssets || isApplyPending}
          >
            {isApplyPending ? (
              <LoaderCircleIcon className="size-4 animate-spin" />
            ) : null}
            应用重组
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function emptyManualSeriesDraft(): ManualSeriesDraft {
  return {
    rootPath: "",
    seriesTitle: "",
    seasonNumber: "",
    migrateMetadata: true,
    episodes: [],
  }
}

function emptyManualMovieVersionsDraft(): ManualMovieVersionsDraft {
  return {
    action: "movie_versions",
    rootPath: "",
    title: "",
  }
}

function buildManualSeriesDraft(
  workspace: CatalogGovernanceWorkspace
): ManualSeriesDraft {
  const episodes = (workspace.assets ?? []).flatMap((asset) =>
    (asset.files ?? [])
      .filter((file) => file.storage_path)
      .map((file) => ({
        assetId: asset.id,
        fileId: file.file_id,
        storagePath: file.storage_path ?? "",
        seasonNumber: "",
        episodeNumber: inferNumberFromPath(file.storage_path ?? ""),
        episodeTitle: titleFromStoragePath(file.storage_path ?? ""),
      }))
  )
  const firstPath = episodes[0]?.storagePath ?? ""
  const rootPath = inferManualSeriesRootPath(firstPath)

  return {
    rootPath,
    seriesTitle: rootPath ? lastPathSegment(rootPath) : workspace.title,
    seasonNumber: "",
    migrateMetadata: true,
    episodes,
  }
}

function buildManualMovieVersionsDraft(
  workspace: CatalogGovernanceWorkspace
): ManualMovieVersionsDraft {
  const rootPath = inferMovieVersionRootPath(workspace)

  return {
    action: "movie_versions",
    rootPath,
    title: workspace.title || (rootPath ? lastPathSegment(rootPath) : ""),
  }
}

function isSeriesLikeWorkspace(workspace: CatalogGovernanceWorkspace) {
  return ["series", "season", "episode"].includes(workspace.type)
}

function workspaceStoragePaths(workspace: CatalogGovernanceWorkspace) {
  const paths: string[] = []

  for (const asset of workspace.assets ?? []) {
    for (const file of asset.files ?? []) {
      if (file.storage_path) paths.push(file.storage_path)
    }
  }

  for (const child of workspace.recommended_children ?? []) {
    if (child.storage_path) paths.push(child.storage_path)
  }

  for (const decision of workspace.classification_decisions ?? []) {
    if (decision.source_path) paths.push(decision.source_path)
    for (const affectedFile of decision.affected_files ?? []) {
      if (affectedFile) paths.push(affectedFile)
    }
  }

  return paths
}

function inferMovieVersionRootPath(workspace: CatalogGovernanceWorkspace) {
  const fileDirectories = workspaceStoragePaths(workspace)
    .map((path) => pathSegments(path).slice(0, -1))
    .filter((segments) => segments.length > 0)
  const commonDirectory = commonPathSegments(fileDirectories)

  if (!commonDirectory.length) return ""
  return `/${commonDirectory.join("/")}`
}

function manualSeriesDraftToInput(draft: ManualSeriesDraft) {
  return {
    root_path: draft.rootPath.trim(),
    series_title: draft.seriesTitle.trim() || undefined,
    season_number: parseOptionalNumber(draft.seasonNumber),
    migrate_metadata: draft.migrateMetadata,
    episode_mappings: draft.episodes.map((episode) => ({
      asset_id: episode.assetId,
      file_id: episode.fileId,
      storage_path: episode.storagePath,
      season_number: parseOptionalNumber(episode.seasonNumber),
      episode_number: parseOptionalNumber(episode.episodeNumber),
      episode_title: episode.episodeTitle.trim() || undefined,
    })),
  }
}

function updateManualSeriesEpisode(
  draft: ManualSeriesDraft,
  index: number,
  updates: Partial<ManualSeriesEpisodeDraft>
): ManualSeriesDraft {
  return {
    ...draft,
    episodes: draft.episodes.map((episode, currentIndex) =>
      currentIndex === index ? { ...episode, ...updates } : episode
    ),
  }
}

function inferManualSeriesRootPath(storagePath: string) {
  const segments = pathSegments(storagePath)
  if (segments.length <= 1) return ""
  if (segments.length >= 4) return `/${segments.slice(0, -3).join("/")}`
  return `/${segments.slice(0, -1).join("/")}`
}

function inferNumberFromPath(storagePath: string) {
  const segments = pathSegments(storagePath)
  for (let index = segments.length - 1; index >= 0; index -= 1) {
    const value = leadingNumber(stripExtension(segments[index]))
    if (value) return String(value)
  }
  return ""
}

function titleFromStoragePath(storagePath: string) {
  const last = lastPathSegment(storagePath)
  return stripExtension(last)
}

function lastPathSegment(value: string) {
  const segments = pathSegments(value)
  return segments.at(-1) ?? ""
}

function pathSegments(value: string) {
  return value.split("/").filter(Boolean)
}

function commonPathSegments(paths: string[][]) {
  const firstPath = paths[0] ?? []
  const common: string[] = []

  for (let index = 0; index < firstPath.length; index += 1) {
    const segment = firstPath[index]
    if (!paths.every((path) => path[index] === segment)) break
    common.push(segment)
  }

  return common
}

function stripExtension(value: string) {
  return value.replace(/\.[^.]+$/, "")
}

function leadingNumber(value: string) {
  const match = value.trim().match(/^(\d+)/)
  if (!match) return undefined
  const parsed = Number(match[1])
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
}

function selectedImageUrl(
  workspace: CatalogGovernanceWorkspace,
  imageType: string
) {
  return (
    (workspace.selected_images || []).find(
      (image) => image.image_type === imageType
    )?.url || ""
  )
}

function fieldStateString(
  workspace: CatalogGovernanceWorkspace,
  fieldKey: string
) {
  const value = fieldStateValue(workspace, fieldKey)
  return typeof value === "string" ? value : ""
}

function fieldStateNumber(
  workspace: CatalogGovernanceWorkspace,
  fieldKey: string
) {
  const value = fieldStateValue(workspace, fieldKey)
  return typeof value === "number" ? value : undefined
}

function fieldStateValue(
  workspace: CatalogGovernanceWorkspace | undefined,
  fieldKey: string
) {
  return (workspace?.field_states ?? []).find(
    (field) => field.field_key === fieldKey
  )?.value
}

function parseOptionalNumber(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return undefined

  const parsed = Number(trimmed)
  return Number.isFinite(parsed) ? parsed : undefined
}

function uniqueMetadataCandidates(candidates: MetadataSearchCandidate[]) {
  const seen = new Set<string>()
  const result: MetadataSearchCandidate[] = []

  for (const candidate of candidates) {
    const key = metadataCandidateIdentity(candidate)
    if (seen.has(key)) continue

    seen.add(key)
    result.push(candidate)
  }

  return result
}

function metadataCandidateIdentity(candidate: MetadataSearchCandidate) {
  return `${candidate.provider.trim().toLowerCase()}-${candidate.external_id.trim()}`
}

function describeAsyncAction(type: AsyncActionState["type"]) {
  switch (type) {
    case "rematch":
      return "重新匹配"
    case "refetch":
      return "元数据重抓"
    case "reprobe":
      return "重新探测"
    default:
      return "后台动作"
  }
}

function formatAsyncActionTitle(state: AsyncActionState) {
  const action = describeAsyncAction(state.type)

  switch (state.status) {
    case "queued":
      return `${action}已排队`
    case "running":
      return `${action}处理中`
    case "completed":
      return `${action}已完成`
    case "failed":
      return `${action}失败`
    default:
      return action
  }
}
