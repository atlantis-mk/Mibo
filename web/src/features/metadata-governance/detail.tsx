import { useEffect, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import {
  CheckCircle2Icon,
  LoaderCircleIcon,
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
  DialogHeader,
  DialogTitle,
} from "#/components/ui/dialog"
import { Separator } from "#/components/ui/separator"
import type { CatalogGovernanceWorkspace } from "#/lib/mibo-api"
import {
  catalogGovernanceWorkspaceQueryOptions,
  createAuthedMiboApi,
  miboQueryKeys,
} from "#/lib/mibo-query"

import {
  ArtworkCard,
	ResourceLinksCard,
  AsyncActionsCard,
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
  | "actions"
  | "locks"
  | "images"
  | "resources"
  | null

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
  const [operationDialog, setOperationDialog] = useState<OperationDialog>(null)
  const [asyncActionState, setAsyncActionState] =
    useState<AsyncActionState | null>(null)
  const [saveSuccessMessage, setSaveSuccessMessage] = useState("")

  useEffect(() => {
    if (!workspaceQuery.data) return

    const nextDraft = buildDraftFromWorkspace(workspaceQuery.data)
    setDraft(nextDraft)
    setBaselineDraft(nextDraft)
	}, [workspaceQuery.data?.metadata_item_id])

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
        message: "重新探测已提交，资源状态会在后台刷新。",
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
  const workspaceResources = workspace.resources ?? []
  const workspaceFieldStates = workspace.field_states ?? []
  const workspaceSourceEvidence = workspace.source_evidence ?? []
  const workspaceClassification = workspace.classification_decisions ?? []
  const workspaceSelectedImages = workspace.selected_images ?? []
  const workspaceImageCandidates = workspace.image_candidates ?? []
  const workspaceRecommendedChildren = workspace.recommended_children ?? []
  const item = buildPreviewItem(workspace)
  const firstInventoryFileId = workspaceResources.find(
    (resource) => (resource.file_ids ?? []).length > 0
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
                页面主体仅展示当前元数据、来源证据、图片、分类和资源关系；需要修改时使用右侧操作按钮打开弹窗处理。
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

        {saveDraftMutation.error ||
        reprobeMutation.error ||
        lockMutation.error ||
        imageMutation.error ? (
          <Alert>
            <AlertTitle>操作失败</AlertTitle>
            <AlertDescription>
              {saveDraftMutation.error?.message ||
                reprobeMutation.error?.message ||
                lockMutation.error?.message ||
                imageMutation.error?.message}
            </AlertDescription>
          </Alert>
        ) : null}

        <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
          <CardHeader className="px-5 py-5">
            <CardTitle>可用操作</CardTitle>
            <CardDescription>
              需要变更元数据、字段锁、图片或资源关系时，从这里打开弹窗。
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
			  onClick={() => setOperationDialog("resources")}
            >
              资源链接
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
              resources={workspaceResources}
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

          {operationDialog === "actions" ? (
            <AsyncActionsCard
              reprobePending={reprobeMutation.isPending}
              reprobeDisabled={!firstInventoryFileId}
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

		  {operationDialog === "resources" ? (
			<ResourceLinksCard
				workspaceItem={{
					id: workspace.metadata_item_id,
					title: workspace.title,
					type: workspace.type,
					availability_status: workspace.availability_status,
                governance_status: workspace.governance_status,
              }}
              relatedChildren={workspaceRecommendedChildren}
              resources={workspaceResources}
              reprobePendingFileId={
                typeof reprobeMutation.variables === "number"
                  ? reprobeMutation.variables
                  : undefined
              }
              onReprobe={(fileId) => {
                void reprobeMutation.mutateAsync(fileId)
              }}
            />
          ) : null}

        </DialogContent>
      </Dialog>
    </>
  )
}

function operationDialogTitle(dialog: OperationDialog) {
  switch (dialog) {
    case "metadata":
      return "编辑元数据"
    case "actions":
      return "后台动作"
    case "locks":
      return "字段锁"
    case "images":
      return "图片选择"
	case "resources":
	  return "资源链接"
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
		id: workspace.metadata_item_id,
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
