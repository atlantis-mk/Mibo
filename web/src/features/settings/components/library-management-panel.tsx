import { useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { CheckCircle2Icon } from "lucide-react"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "#/components/ui/alert-dialog"
import type { HealthIssue, Library, MediaSource } from "#/lib/mibo-api"
import {
  createAuthedMiboApi,
  healthIssuesQueryOptions,
  librariesQueryOptions,
  mediaSourcesQueryOptions,
  metadataProviderInstancesQueryOptions,
  metadataProfilesQueryOptions,
  miboQueryKeys,
} from "#/lib/mibo-query"

import {
  EMPTY_LIBRARY_FORM,
  libraryFormMetadataStrategyInput,
  libraryFormScanExclusionRuleInputs,
  type LibraryFormState,
} from "./library-form"
import { LibraryDrawer, MediaSourceDrawer } from "./library-management-drawers"
import { LibrariesTab } from "./libraries-tab"
import { LibrarySettingsDrawer } from "./library-settings-drawer"
import {
  buildMediaSourceDraft,
  DEFAULT_OPENLIST_BASE_URL,
  deriveLocalSourceName,
  EMPTY_SOURCE_FORM,
  type SourceFormState,
} from "./media-source-form"
import { MediaSourcesTab } from "./media-sources-tab"

export function LibraryManagementPanel({
  token,
  activeTab,
}: {
  token: string | null
  activeTab: "sources" | "libraries"
}) {
  const queryClient = useQueryClient()
  const queryToken = token ?? "guest"
  const api = useMemo(
    () => (token ? createAuthedMiboApi(token) : null),
    [token]
  )

  const [actionMessage, setActionMessage] = useState<string | null>(null)
  const [isCreateSourceOpen, setIsCreateSourceOpen] = useState(false)
  const [isCreateLibraryOpen, setIsCreateLibraryOpen] = useState(false)
  const [sourceDraft, setSourceDraft] =
    useState<SourceFormState>(EMPTY_SOURCE_FORM)
  const [libraryDraft, setLibraryDraft] =
    useState<LibraryFormState>(EMPTY_LIBRARY_FORM)
  const [editingSource, setEditingSource] = useState<MediaSource | null>(null)
  const [editingSourceDraft, setEditingSourceDraft] =
    useState<SourceFormState>(EMPTY_SOURCE_FORM)
  const [deletingSource, setDeletingSource] = useState<MediaSource | null>(null)
  const [deletingLibrary, setDeletingLibrary] = useState<Library | null>(null)
  const [editingLibrary, setEditingLibrary] = useState<Library | null>(null)

  const mediaSourcesQuery = useQuery({
    ...mediaSourcesQueryOptions(queryToken),
    enabled: !!token,
  })
  const librariesQuery = useQuery({
    ...librariesQueryOptions(queryToken),
    enabled: !!token,
  })
  const metadataProfilesQuery = useQuery({
    ...metadataProfilesQueryOptions(queryToken),
    enabled: !!token,
  })
  const metadataProviderInstancesQuery = useQuery({
    ...metadataProviderInstancesQueryOptions(queryToken),
    enabled: !!token,
  })
  const healthIssuesQuery = useQuery({
    ...healthIssuesQueryOptions(queryToken),
    enabled: !!token,
  })

  const mediaSources = mediaSourcesQuery.data ?? []
  const libraries = librariesQuery.data ?? []
  const healthIssues = healthIssuesQuery.data ?? []
  const metadataProfiles = metadataProfilesQuery.data ?? []
  const metadataProviderInstances = metadataProviderInstancesQuery.data ?? []

  async function invalidateData() {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: miboQueryKeys.mediaSources(queryToken),
      }),
      queryClient.invalidateQueries({
        queryKey: miboQueryKeys.libraries(queryToken),
      }),
    ])
  }

  const createMediaSourceMutation = useMutation({
    mutationFn: async () => {
      if (!api) throw new Error("当前未登录，无法创建媒体源。")

      return api.createMediaSource({
        provider: sourceDraft.provider,
        name:
          sourceDraft.provider === "local"
            ? deriveLocalSourceName(sourceDraft.rootPath)
            : sourceDraft.name,
        root_path: sourceDraft.rootPath,
        config:
          sourceDraft.provider === "openlist"
            ? {
                openlist: {
                  base_url: sourceDraft.baseUrl || DEFAULT_OPENLIST_BASE_URL,
                  username: sourceDraft.username || undefined,
                  password: sourceDraft.password || undefined,
                },
              }
            : undefined,
      })
    },
    onSuccess: async () => {
      setActionMessage("媒体源已创建。")
      setIsCreateSourceOpen(false)
      setSourceDraft(EMPTY_SOURCE_FORM)
      await invalidateData()
    },
    onError: (error) => {
      setActionMessage(
        error instanceof Error ? error.message : "创建媒体源失败。"
      )
    },
  })

  const updateMediaSourceMutation = useMutation({
    mutationFn: async () => {
      if (!api || !editingSource) {
        throw new Error("当前未选择要编辑的媒体源。")
      }

      return api.updateMediaSource(editingSource.id, {
        name:
          editingSource.provider === "local"
            ? deriveLocalSourceName(editingSourceDraft.rootPath)
            : editingSourceDraft.name,
        root_path: editingSourceDraft.rootPath,
        config:
          editingSource.provider === "openlist"
            ? {
                openlist: {
                  base_url:
                    editingSourceDraft.baseUrl || DEFAULT_OPENLIST_BASE_URL,
                  username: editingSourceDraft.username || undefined,
                  password: editingSourceDraft.password || undefined,
                },
              }
            : undefined,
      })
    },
    onSuccess: async () => {
      setActionMessage("媒体源已更新。")
      setEditingSource(null)
      setEditingSourceDraft(EMPTY_SOURCE_FORM)
      await invalidateData()
    },
    onError: (error) => {
      setActionMessage(
        error instanceof Error ? error.message : "更新媒体源失败。"
      )
    },
  })

  const deleteMediaSourceMutation = useMutation({
    mutationFn: async () => {
      if (!api || !deletingSource) {
        throw new Error("当前未选择要删除的媒体源。")
      }

      return api.deleteMediaSource(deletingSource.id)
    },
    onSuccess: async () => {
      setActionMessage("媒体源已删除。")
      setDeletingSource(null)
      await invalidateData()
    },
    onError: (error) => {
      setActionMessage(
        error instanceof Error ? error.message : "删除媒体源失败。"
      )
    },
  })

  const createLibraryMutation = useMutation({
    mutationFn: async () => {
      if (!api) throw new Error("当前未登录，无法添加内容来源。")

      return api.createLibrary({
        name: libraryDraft.name,
        media_source_id: Number(libraryDraft.mediaSourceId),
        root_path: libraryDraft.rootPath,
        scan: libraryDraft.scan,
        metadata: libraryDraft.metadata,
        metadata_strategy: libraryFormMetadataStrategyInput(libraryDraft),
        playback: libraryDraft.playback,
        subtitle: libraryDraft.subtitle,
        scan_exclusion_rules: libraryFormScanExclusionRuleInputs(libraryDraft),
      })
    },
    onSuccess: async () => {
      setActionMessage("内容来源已添加，Mibo 正在后台扫描。")
      setIsCreateLibraryOpen(false)
      setLibraryDraft({ ...EMPTY_LIBRARY_FORM })
      await invalidateData()
    },
    onError: (error) => {
      setActionMessage(
        error instanceof Error ? error.message : "添加内容来源失败。"
      )
    },
  })

  const deleteLibraryMutation = useMutation({
    mutationFn: async () => {
      if (!api || !deletingLibrary) {
        throw new Error("当前未选择要删除的媒体库。")
      }

      return api.deleteLibrary(deletingLibrary.id)
    },
    onSuccess: async () => {
      setActionMessage("媒体库已删除。")
      setDeletingLibrary(null)
      await invalidateData()
    },
    onError: (error) => {
      setActionMessage(
        error instanceof Error ? error.message : "删除媒体库失败。"
      )
    },
  })

  const scanLibraryMutation = useMutation({
    mutationFn: async (libraryId: number) => {
      if (!api) throw new Error("当前未登录，无法扫描媒体库。")
      return api.scanLibrary(libraryId)
    },
    onSuccess: () => {
      setActionMessage("媒体库扫描任务已提交。")
    },
    onError: (error) => {
      setActionMessage(
        error instanceof Error ? error.message : "提交扫描任务失败。"
      )
    },
  })

  return (
    <div className="space-y-4">
      {actionMessage ? (
        <div className="flex items-center gap-2 rounded-[1.1rem] border border-border bg-muted px-4 py-3 text-sm text-foreground">
          <CheckCircle2Icon className="size-4 text-muted-foreground" />
          <span>{actionMessage}</span>
        </div>
      ) : null}

      {activeTab === "sources" ? (
        <div>
          <MediaSourcesTab
            mediaSources={mediaSources}
            healthIssues={healthIssues}
            isLoading={mediaSourcesQuery.isLoading}
            onCreate={() => setIsCreateSourceOpen(true)}
            onEdit={(source) => {
              setEditingSource(source)
              setEditingSourceDraft(buildMediaSourceDraft(source))
            }}
            onDelete={setDeletingSource}
          />
        </div>
      ) : null}

      {activeTab === "libraries" ? (
        <div>
          <LibrariesTab
            libraries={libraries}
            mediaSources={mediaSources}
            healthIssues={healthIssues}
            isLoading={librariesQuery.isLoading}
            isScanning={scanLibraryMutation.isPending}
            onCreate={() => setIsCreateLibraryOpen(true)}
            onEdit={setEditingLibrary}
            onScan={(libraryId) => scanLibraryMutation.mutate(libraryId)}
            onDelete={setDeletingLibrary}
          />
        </div>
      ) : null}

      <MediaSourceDrawer
        open={isCreateSourceOpen}
        title="创建媒体源"
        description="创建本地目录或 OpenList 媒体源。"
        draft={sourceDraft}
        onChange={setSourceDraft}
        api={api}
        pending={createMediaSourceMutation.isPending}
        disabled={
          createMediaSourceMutation.isPending ||
          !sourceDraft.rootPath ||
          (sourceDraft.provider !== "local" && !sourceDraft.name) ||
          (sourceDraft.provider === "openlist" && !sourceDraft.baseUrl)
        }
        submitLabel="创建"
        onOpenChange={setIsCreateSourceOpen}
        onSubmit={() => createMediaSourceMutation.mutate()}
      />

      <MediaSourceDrawer
        open={editingSource !== null}
        title="编辑媒体源"
        description="修改媒体源名称、路径和连接信息。"
        draft={editingSourceDraft}
        onChange={setEditingSourceDraft}
        api={api}
        isEditing
        pending={updateMediaSourceMutation.isPending}
        disabled={updateMediaSourceMutation.isPending || !editingSource}
        submitLabel="保存"
        onOpenChange={(open) => {
          if (!open) {
            setEditingSource(null)
          }
        }}
        onSubmit={() => updateMediaSourceMutation.mutate()}
      />

      <LibraryDrawer
        open={isCreateLibraryOpen}
        draft={libraryDraft}
        onChange={setLibraryDraft}
        mediaSources={mediaSources}
        metadataProfiles={metadataProfiles}
        metadataProviderInstances={metadataProviderInstances}
        api={api}
        pending={createLibraryMutation.isPending}
        disabled={
          createLibraryMutation.isPending ||
          !libraryDraft.name.trim() ||
          !libraryDraft.mediaSourceId ||
          !libraryDraft.rootPath
        }
        onOpenChange={setIsCreateLibraryOpen}
        onSubmit={() => createLibraryMutation.mutate()}
      />

      <LibrarySettingsDrawer
        open={!!editingLibrary}
        library={editingLibrary}
        mediaSources={mediaSources}
        api={api}
        onOpenChange={(open) => {
          if (!open) setEditingLibrary(null)
        }}
        onSaved={invalidateData}
      />

      <AlertDialog
        open={deletingSource !== null}
        onOpenChange={(open) => !open && setDeletingSource(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除媒体源</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingSource
                ? `确认删除媒体源“${deletingSource.name}”吗？该操作不可撤销。`
                : ""}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteMediaSourceMutation.mutate()}
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={deletingLibrary !== null}
        onOpenChange={(open) => !open && setDeletingLibrary(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除媒体库</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingLibrary
                ? `确认删除媒体库“${deletingLibrary.name}”吗？该操作不可撤销。`
                : ""}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={() => deleteLibraryMutation.mutate()}>
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
