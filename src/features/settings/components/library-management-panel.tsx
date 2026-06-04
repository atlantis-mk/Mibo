import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FolderPlusIcon } from 'lucide-react'
import { createPortal } from 'react-dom'
import { toast } from 'sonner'
import type {
  Library,
  MediaSource,
  PluginProviderInstance,
} from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  librariesQueryOptions,
  mediaSourcesQueryOptions,
  metadataProviderInstancesQueryOptions,
  metadataProfilesQueryOptions,
  miboQueryKeys,
  operationsTasksQueryOptions,
  pluginProviderInstancesQueryOptions,
  userSettingsQueryOptions,
} from '@/lib/mibo-query'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { LibrariesTab } from './libraries-tab'
import {
  EMPTY_LIBRARY_FORM,
  libraryFormMetadataStrategyInput,
  libraryFormScanExclusionRuleInputs,
  type LibraryFormState,
} from './library-form'
import { LibraryDrawer, MediaSourceDrawer } from './library-management-drawers'
import { LibrarySettingsDrawer } from './library-settings-drawer'
import {
  buildMediaSourceDraft,
  DEFAULT_OPENLIST_BASE_URL,
  deriveLocalSourceName,
  EMPTY_SOURCE_FORM,
  type SourceFormState,
} from './media-source-form'
import { MediaSourcesTab } from './media-sources-tab'

export function LibraryManagementPanel({
  token,
  activeTab,
  onActiveTabChange,
}: {
  token: string | null
  activeTab: 'sources' | 'libraries'
  onActiveTabChange: (tab: 'sources' | 'libraries') => void
}) {
  const queryClient = useQueryClient()
  const queryToken = token ?? 'guest'
  const api = useMemo(
    () => (token ? createAuthedMiboApi(token) : null),
    [token]
  )

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
  const [pendingScan, setPendingScan] = useState<{
    libraryId: number
    mode: 'full' | 'changed'
  } | null>(null)

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
  const pluginProviderInstancesQuery = useQuery({
    ...pluginProviderInstancesQueryOptions(queryToken),
    enabled: !!token,
  })
  const operationsTasksQuery = useQuery({
    ...operationsTasksQueryOptions(queryToken),
    enabled: !!token,
  })
  const userSettingsQuery = useQuery({
    ...userSettingsQueryOptions(queryToken),
    enabled: !!token,
  })
  const libraryAccessTagsQuery = useQuery({
    queryKey: miboQueryKeys.libraryAccessTags(queryToken),
    queryFn: () => createAuthedMiboApi(queryToken).listLibraryAccessTags(),
    enabled: !!token,
  })

  const mediaSources = mediaSourcesQuery.data ?? []
  const libraries = librariesQuery.data ?? []
  const operationsTasks = operationsTasksQuery.data ?? []
  const availableAccessTags = libraryAccessTagsQuery.data ?? []
  const metadataProfiles = metadataProfilesQuery.data ?? []
  const metadataProviderInstances = metadataProviderInstancesQuery.data ?? []
  const pluginProviderInstances: PluginProviderInstance[] =
    pluginProviderInstancesQuery.data ?? []
  const requireDangerousActionConfirmation =
    userSettingsQuery.data?.security.require_dangerous_action_confirmation ??
    true
  const primaryActionLabel =
    activeTab === 'sources' ? '创建媒体源' : '创建媒体库'

  async function invalidateData() {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: miboQueryKeys.mediaSources(queryToken),
      }),
      queryClient.invalidateQueries({
        queryKey: miboQueryKeys.libraries(queryToken),
      }),
      queryClient.invalidateQueries({
        queryKey: miboQueryKeys.libraryAccessTags(queryToken),
      }),
      queryClient.invalidateQueries({
        queryKey: miboQueryKeys.pluginProviderInstances(queryToken),
      }),
    ])
  }

  const createMediaSourceMutation = useMutation({
    mutationFn: async () => {
      if (!api) throw new Error('当前未登录，无法创建媒体源。')

      return api.createMediaSource({
        provider: sourceDraft.provider,
        name:
          sourceDraft.provider === 'local'
            ? deriveLocalSourceName(sourceDraft.rootPath)
            : sourceDraft.name,
        root_path: sourceDraft.rootPath,
        config:
          sourceDraft.provider === 'openlist'
            ? {
                openlist: {
                  base_url: sourceDraft.baseUrl || DEFAULT_OPENLIST_BASE_URL,
                  username: sourceDraft.username || undefined,
                  password: sourceDraft.password || undefined,
                  scan_interval: sourceDraft.scanInterval || undefined,
                },
              }
            : undefined,
      })
    },
    onSuccess: async () => {
      toast.success('媒体源已创建。')
      setIsCreateSourceOpen(false)
      setSourceDraft(EMPTY_SOURCE_FORM)
      await invalidateData()
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '创建媒体源失败。')
    },
  })

  const updateMediaSourceMutation = useMutation({
    mutationFn: async () => {
      if (!api || !editingSource) {
        throw new Error('当前未选择要编辑的媒体源。')
      }

      return api.updateMediaSource(editingSource.id, {
        name:
          editingSource.provider === 'local'
            ? deriveLocalSourceName(editingSourceDraft.rootPath)
            : editingSourceDraft.name,
        root_path: editingSourceDraft.rootPath,
        config:
          editingSource.provider === 'openlist'
            ? {
                openlist: {
                  base_url:
                    editingSourceDraft.baseUrl || DEFAULT_OPENLIST_BASE_URL,
                  username: editingSourceDraft.username || undefined,
                  password: editingSourceDraft.password || undefined,
                  scan_interval: editingSourceDraft.scanInterval || undefined,
                },
              }
            : undefined,
      })
    },
    onSuccess: async () => {
      toast.success('媒体源已更新。')
      setEditingSource(null)
      setEditingSourceDraft(EMPTY_SOURCE_FORM)
      await invalidateData()
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '更新媒体源失败。')
    },
  })

  const deleteMediaSourceMutation = useMutation({
    mutationFn: async (target?: MediaSource) => {
      const source = target ?? deletingSource
      if (!api || !source) {
        throw new Error('当前未选择要删除的媒体源。')
      }

      return api.deleteMediaSource(source.id)
    },
    onSuccess: async () => {
      toast.success('媒体源已删除。')
      setDeletingSource(null)
      await invalidateData()
    },
    onError: (error) => {
      setDeletingSource(null)
      toast.error(error instanceof Error ? error.message : '删除媒体源失败。')
    },
  })

  const createLibraryMutation = useMutation({
    mutationFn: async () => {
      if (!api) throw new Error('当前未登录，无法添加内容来源。')

      return api.createLibrary({
        name: libraryDraft.name,
        media_source_id: Number(libraryDraft.mediaSourceId),
        root_path: libraryDraft.rootPath,
        visibility_mode: libraryDraft.visibilityMode,
        access_tags: libraryDraft.accessTags,
        scan: libraryDraft.scan,
        metadata: libraryDraft.metadata,
        metadata_strategy: libraryFormMetadataStrategyInput(libraryDraft),
        playback: libraryDraft.playback,
        subtitle: libraryDraft.subtitle,
        scan_exclusion_rules: libraryFormScanExclusionRuleInputs(libraryDraft),
      })
    },
    onSuccess: async () => {
      toast.success('媒体库已创建。')
      setIsCreateLibraryOpen(false)
      setLibraryDraft({ ...EMPTY_LIBRARY_FORM })
      await invalidateData()
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '添加内容来源失败。')
    },
  })

  const deleteLibraryMutation = useMutation({
    mutationFn: async (target?: Library) => {
      const library = target ?? deletingLibrary
      if (!api || !library) {
        throw new Error('当前未选择要删除的媒体库。')
      }

      return api.deleteLibrary(library.id)
    },
    onSuccess: async () => {
      toast.success('媒体库已删除。')
      setDeletingLibrary(null)
      await invalidateData()
    },
    onError: (error) => {
      setDeletingLibrary(null)
      toast.error(error instanceof Error ? error.message : '删除媒体库失败。')
    },
  })

  const scanLibraryMutation = useMutation({
    mutationFn: async ({
      libraryId,
      mode,
    }: {
      libraryId: number
      mode: 'full' | 'changed'
    }) => {
      if (!api) throw new Error('当前未登录，无法扫描媒体库。')
      return api.scanLibrary(libraryId, mode)
    },
    onSuccess: (result) => {
      setPendingScan(null)
      toast.success(
        result.mode === 'changed'
          ? '变化扫描任务已提交。'
          : '全量扫描任务已提交。'
      )
    },
    onError: (error) => {
      setPendingScan(null)
      toast.error(error instanceof Error ? error.message : '提交扫描任务失败。')
    },
  })

  function requestDeleteSource(source: MediaSource) {
    if (requireDangerousActionConfirmation) {
      setDeletingSource(source)
      return
    }
    deleteMediaSourceMutation.mutate(source)
  }

  function requestDeleteLibrary(library: Library) {
    if (requireDangerousActionConfirmation) {
      setDeletingLibrary(library)
      return
    }
    deleteLibraryMutation.mutate(library)
  }

  function requestScanLibrary(libraryId: number, mode: 'full' | 'changed') {
    if (requireDangerousActionConfirmation) {
      setPendingScan({ libraryId, mode })
      return
    }
    scanLibraryMutation.mutate({ libraryId, mode })
  }

  return (
    <div className='space-y-4 pb-20'>
      <div className='flex justify-center'>
        <div className='inline-flex rounded-lg border border-border/60 bg-muted/30 p-1'>
          <Button
            type='button'
            onClick={() => onActiveTabChange('libraries')}
            variant={activeTab === 'libraries' ? 'outline' : 'ghost'}
            size='sm'
          >
            媒体库
          </Button>
          <Button
            type='button'
            onClick={() => onActiveTabChange('sources')}
            variant={activeTab === 'sources' ? 'outline' : 'ghost'}
            size='sm'
          >
            媒体源
          </Button>
        </div>
      </div>

      {activeTab === 'sources' ? (
        <div>
          <MediaSourcesTab
            mediaSources={mediaSources}
            pluginProviderInstances={pluginProviderInstances}
            operationsTasks={operationsTasks}
            isLoading={mediaSourcesQuery.isLoading}
            onEdit={(source) => {
              setEditingSource(source)
              setEditingSourceDraft(buildMediaSourceDraft(source))
            }}
            onDelete={requestDeleteSource}
          />
        </div>
      ) : null}

      {activeTab === 'libraries' ? (
        <div>
          <LibrariesTab
            libraries={libraries}
            mediaSources={mediaSources}
            operationsTasks={operationsTasks}
            isLoading={librariesQuery.isLoading}
            isScanning={scanLibraryMutation.isPending}
            onEdit={setEditingLibrary}
            onScan={requestScanLibrary}
            onDelete={requestDeleteLibrary}
          />
        </div>
      ) : null}

      <MediaSourceDrawer
        open={isCreateSourceOpen}
        title='创建媒体源'
        description='创建本地目录或 OpenList 媒体源。'
        draft={sourceDraft}
        onChange={setSourceDraft}
        api={api}
        pluginProviderInstances={pluginProviderInstances}
        pending={createMediaSourceMutation.isPending}
        disabled={
          createMediaSourceMutation.isPending ||
          !sourceDraft.rootPath ||
          (sourceDraft.provider !== 'local' && !sourceDraft.name) ||
          (sourceDraft.provider === 'openlist' && !sourceDraft.baseUrl)
        }
        submitLabel='创建'
        onOpenChange={setIsCreateSourceOpen}
        onSubmit={() => createMediaSourceMutation.mutate()}
      />

      <MediaSourceDrawer
        open={editingSource !== null}
        title='编辑媒体源'
        description='修改媒体源名称、路径和连接信息。'
        draft={editingSourceDraft}
        onChange={setEditingSourceDraft}
        api={api}
        pluginProviderInstances={pluginProviderInstances}
        isEditing
        pending={updateMediaSourceMutation.isPending}
        disabled={updateMediaSourceMutation.isPending || !editingSource}
        submitLabel='保存'
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
        availableAccessTags={availableAccessTags}
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
                : ''}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteMediaSourceMutation.mutate(undefined)}
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
                : ''}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteLibraryMutation.mutate(undefined)}
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={pendingScan !== null}
        onOpenChange={(open) => !open && setPendingScan(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>扫描媒体库</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingScan?.mode === 'changed'
                ? '确认提交变化扫描任务吗？'
                : '确认提交全量扫描任务吗？全量扫描可能需要较长时间。'}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (!pendingScan) return
                scanLibraryMutation.mutate(pendingScan)
              }}
            >
              扫描
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {createPortal(
        <div className='fixed right-6 bottom-6 z-[100] flex justify-end'>
          <Button
            disabled={!token}
            title={token ? undefined : '请先登录后执行创建操作'}
            onClick={() => {
              if (activeTab === 'sources') {
                setIsCreateSourceOpen(true)
                return
              }
              setIsCreateLibraryOpen(true)
            }}
          >
            <FolderPlusIcon className='size-4' />
            {primaryActionLabel}
          </Button>
        </div>,
        document.body
      )}
    </div>
  )
}
