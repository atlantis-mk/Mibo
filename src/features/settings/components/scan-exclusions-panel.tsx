import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  FileX2Icon,
  Loader2Icon,
  RefreshCwIcon,
  RotateCcwIcon,
  SearchIcon,
  ShieldOffIcon,
} from 'lucide-react'
import type {
  FilenameExclusionRule,
  FilenameExclusionPreview,
  Library,
  ScanExclusion,
} from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  inventoryFilesQueryOptions,
  librariesQueryOptions,
  miboQueryKeys,
  scanExclusionsQueryOptions,
} from '@/lib/mibo-query'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ScanExclusionRulesPanel } from './scan-exclusion-rules-panel'

type EnabledFilter = 'all' | 'enabled' | 'disabled'

export function ScanExclusionsPanel({
  token,
  activeTab,
}: {
  token: string | null
  activeTab: 'rules' | 'exclusions' | 'browser'
}) {
  const queryClient = useQueryClient()
  const queryToken = token ?? 'guest'
  const [libraryFilter, setLibraryFilter] = useState('all')
  const [browserLibraryId, setBrowserLibraryId] = useState<string>('')
  const [browserPage, setBrowserPage] = useState(1)
  const [browserQuery, setBrowserQuery] = useState('')
  const [enabledFilter, setEnabledFilter] = useState<EnabledFilter>('all')
  const [actionMessage, setActionMessage] = useState<string | null>(null)

  const filters = useMemo(
    () => ({
      libraryId: libraryFilter === 'all' ? undefined : Number(libraryFilter),
      enabled:
        enabledFilter === 'all' ? undefined : enabledFilter === 'enabled',
    }),
    [enabledFilter, libraryFilter]
  )

  const exclusionsQuery = useQuery({
    ...scanExclusionsQueryOptions(queryToken, filters),
    enabled: Boolean(token),
  })
  const librariesQuery = useQuery({
    ...librariesQueryOptions(queryToken),
    enabled: Boolean(token),
  })
  const browserLibrary = browserLibraryId ? Number(browserLibraryId) : undefined
  const browserFilesQuery = useQuery({
    ...inventoryFilesQueryOptions(queryToken, {
      page: browserPage,
      limit: 24,
      libraryId: browserLibrary,
      q: browserQuery,
    }),
    enabled: Boolean(token) && activeTab === 'browser',
  })

  const exclusions = exclusionsQuery.data?.manual_exclusions ?? []
  const filenameRules = exclusionsQuery.data?.filename_rules ?? []
  const enabledCount = exclusions.filter((item) => item.enabled).length
  const enabledRuleCount = filenameRules.filter((item) => item.enabled).length
  const disabledCount =
    exclusions.length + filenameRules.length - enabledCount - enabledRuleCount

  const invalidateExclusions = async () => {
    if (!token) return
    await queryClient.invalidateQueries({
      queryKey: miboQueryKeys.scanExclusions(queryToken, filters),
    })
  }

  const toggleMutation = useMutation({
    mutationFn: async (input: { id: number; enabled: boolean }) => {
      if (!token) throw new Error('当前未登录，无法更新扫描排除项。')
      return createAuthedMiboApi(token).setScanExclusionEnabled(
        input.id,
        input.enabled
      )
    },
    onSuccess: async (updated) => {
      setActionMessage(
        updated.enabled ? '排除项已重新启用。' : '排除项已恢复。'
      )
      await invalidateExclusions()
    },
  })

  const ruleToggleMutation = useMutation({
    mutationFn: async (input: { id: number; enabled: boolean }) => {
      if (!token) throw new Error('当前未登录，无法更新同名忽略规则。')
      return createAuthedMiboApi(token).setFilenameExclusionRuleEnabled(
        input.id,
        input.enabled
      )
    },
    onSuccess: async (updated) => {
      setActionMessage(
        updated.enabled
          ? '同名忽略规则已重新启用。'
          : '同名忽略规则已恢复。后续扫描会重新允许这些文件。'
      )
      await invalidateExclusions()
    },
  })

  const restoreMemberMutation = useMutation({
    mutationFn: async (input: { groupId: number; fileId: number }) => {
      if (!token) throw new Error('当前未登录，无法恢复文件。')
      return createAuthedMiboApi(token).restoreFilenameExclusionMatch(
        input.groupId,
        input.fileId
      )
    },
    onSuccess: async () => {
      setActionMessage('文件已单独恢复，后续扫描会重新允许该文件。')
      await invalidateExclusions()
    },
  })

  const unrestoreMemberMutation = useMutation({
    mutationFn: async (input: { groupId: number; fileId: number }) => {
      if (!token) throw new Error('当前未登录，无法重新排除此文件。')
      return createAuthedMiboApi(token).deleteFilenameExclusionRestore(
        input.groupId,
        input.fileId
      )
    },
    onSuccess: async () => {
      setActionMessage('文件已重新排除。')
      await invalidateExclusions()
    },
  })

  return (
    <div className='flex h-full min-h-0 flex-1 flex-col gap-4 overflow-hidden'>
      {activeTab === 'rules' ? <ScanExclusionRulesPanel token={token} /> : null}

      {activeTab === 'exclusions' ? (
        <>
          {actionMessage ? (
            <div className='flex items-center gap-2 rounded-[1.1rem] border border-border bg-muted px-4 py-3 text-sm text-foreground'>
              <CheckCircle2Icon className='size-4 text-muted-foreground' />
              <span>{actionMessage}</span>
            </div>
          ) : null}

          {toggleMutation.error ||
          ruleToggleMutation.error ||
          restoreMemberMutation.error ? (
            <ErrorBanner
              message={errorMessage(
                toggleMutation.error ||
                  ruleToggleMutation.error ||
                  restoreMemberMutation.error
              )}
            />
          ) : null}

          <section className='flex min-h-0 flex-1 flex-col'>
            <div className='mb-5 flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between'>
              <div className='space-y-2'>
                <h3 className='text-base font-medium'>排除项列表</h3>
                <div className='flex flex-wrap items-center gap-2'>
                  <Badge variant='outline' className='gap-1.5 bg-background/70'>
                    <FileX2Icon className='size-3.5 text-amber-500' />
                    {exclusions.length + filenameRules.length} 条排除记录
                  </Badge>
                  <Badge variant='outline' className='gap-1.5 bg-background/70'>
                    <ShieldOffIcon className='size-3.5 text-emerald-500' />
                    {enabledCount + enabledRuleCount} 条生效
                  </Badge>
                  <Badge variant='outline' className='gap-1.5 bg-background/70'>
                    <RotateCcwIcon className='size-3.5 text-muted-foreground' />
                    {disabledCount} 条已恢复
                  </Badge>
                </div>
              </div>

              <div className='flex flex-wrap items-center gap-2'>
                <NativeSelect
                  value={libraryFilter}
                  onChange={(event) =>
                    setLibraryFilter(event.currentTarget.value)
                  }
                  className='w-full sm:w-44'
                  aria-label='按媒体库筛选'
                >
                  <NativeSelectOption value='all'>
                    全部媒体库
                  </NativeSelectOption>
                  {(librariesQuery.data ?? []).map((library) => (
                    <NativeSelectOption
                      key={library.id}
                      value={String(library.id)}
                    >
                      {library.name}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
                <NativeSelect
                  value={enabledFilter}
                  onChange={(event) =>
                    setEnabledFilter(event.currentTarget.value as EnabledFilter)
                  }
                  className='w-full sm:w-36'
                  aria-label='按状态筛选'
                >
                  <NativeSelectOption value='all'>全部状态</NativeSelectOption>
                  <NativeSelectOption value='enabled'>
                    仅生效
                  </NativeSelectOption>
                  <NativeSelectOption value='disabled'>
                    仅恢复
                  </NativeSelectOption>
                </NativeSelect>
                <Button
                  variant='outline'
                  onClick={() => void invalidateExclusions()}
                  disabled={!token || exclusionsQuery.isFetching}
                >
                  <RefreshCwIcon
                    className={cn(
                      'size-4',
                      exclusionsQuery.isFetching && 'animate-spin'
                    )}
                  />
                  刷新
                </Button>
              </div>
            </div>

            {exclusionsQuery.isLoading ? (
              <ExclusionSkeleton />
            ) : exclusionsQuery.isError ? (
              <ErrorState onRetry={() => void invalidateExclusions()} />
            ) : exclusions.length === 0 && filenameRules.length === 0 ? (
              <EmptyState />
            ) : (
              <div className='flex min-h-0 flex-1 flex-col gap-4 overflow-hidden'>
                {filenameRules.length > 0 ? (
                  <FilenameRulesList
                    rules={filenameRules}
                    pending={
                      ruleToggleMutation.isPending ||
                      restoreMemberMutation.isPending ||
                      unrestoreMemberMutation.isPending
                    }
                    onToggle={(rule) =>
                      ruleToggleMutation.mutate({
                        id: rule.id,
                        enabled: !rule.enabled,
                      })
                    }
                    onRestoreMember={(rule, fileId) =>
                      restoreMemberMutation.mutate({
                        groupId: rule.id,
                        fileId,
                      })
                    }
                    onUnrestoreMember={(rule, fileId) =>
                      unrestoreMemberMutation.mutate({
                        groupId: rule.id,
                        fileId,
                      })
                    }
                  />
                ) : null}
                {exclusions.length > 0 ? (
                  <ExclusionsTable
                    exclusions={exclusions}
                    pending={toggleMutation.isPending}
                    onToggle={(exclusion) =>
                      toggleMutation.mutate({
                        id: exclusion.id,
                        enabled: !exclusion.enabled,
                      })
                    }
                  />
                ) : null}
              </div>
            )}
          </section>
        </>
      ) : null}

      {activeTab === 'browser' ? (
        <MediaFileBrowserSection
          token={token}
          libraries={librariesQuery.data ?? []}
          selectedLibraryId={browserLibrary ?? null}
          onLibraryChange={setBrowserLibraryId}
          files={browserFilesQuery.data?.items ?? []}
          isLoading={browserFilesQuery.isLoading}
          page={browserPage}
          total={browserFilesQuery.data?.total ?? 0}
          hasMore={browserFilesQuery.data?.has_more ?? false}
          onPageChange={setBrowserPage}
          query={browserQuery}
          onQueryChange={(value) => {
            setBrowserQuery(value)
            setBrowserPage(1)
          }}
        />
      ) : null}
    </div>
  )
}

function MediaFileBrowserSection({
  token,
  libraries,
  selectedLibraryId,
  onLibraryChange,
  files,
  isLoading,
  page,
  total,
  hasMore,
  onPageChange,
  query,
  onQueryChange,
}: {
  token: string | null
  libraries: Library[]
  selectedLibraryId: number | null
  onLibraryChange: (value: string) => void
  files: {
    id: number
    storage_path: string
    storage_provider: string
    thumbnail_url?: string
    excluded?: boolean
    exclusion_id?: number
    filename_rule_id?: number
  }[]
  isLoading: boolean
  page: number
  total: number
  hasMore: boolean
  onPageChange: (page: number) => void
  query: string
  onQueryChange: (value: string) => void
}) {
  const queryClient = useQueryClient()
  const [queryDraft, setQueryDraft] = useState(query)
  const [ignoreDialogOpen, setIgnoreDialogOpen] = useState(false)
  const [ignorePreview, setIgnorePreview] =
    useState<FilenameExclusionPreview | null>(null)
  const [selectedFileId, setSelectedFileId] = useState<number | null>(null)
  const [dialogMode, setDialogMode] = useState<'ignore' | 'restore'>('ignore')
  const [selectedRuleId, setSelectedRuleId] = useState<number | null>(null)

  useEffect(() => {
    setQueryDraft(query)
  }, [query])

  const submitQuery = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    onQueryChange(queryDraft.trim())
  }

  const previewIgnoreMutation = useMutation({
    mutationFn: async (fileId: number) => {
      if (!token) throw new Error('当前未登录，无法预览忽略影响。')
      setDialogMode('ignore')
      setSelectedFileId(fileId)
      setSelectedRuleId(null)
      return createAuthedMiboApi(token).previewInventoryFileScanExclusion(
        fileId
      )
    },
    onSuccess: (preview) => {
      setIgnorePreview(preview)
      setIgnoreDialogOpen(true)
    },
  })

  const ignoreMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error('当前未登录，无法标记忽略。')
      if (!selectedFileId)
        throw new Error('当前条目缺少文件锚点，无法标记忽略。')
      return createAuthedMiboApi(token).markInventoryFileScanExclusion(
        selectedFileId,
        'advertisement'
      )
    },
    onSuccess: async () => {
      setIgnoreDialogOpen(false)
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['settings', 'scan-exclusions'],
        }),
        queryClient.invalidateQueries({ queryKey: ['inventory-files'] }),
        queryClient.invalidateQueries({
          queryKey: ['library', 'inventory-files'],
        }),
      ])
    },
  })

  const filenameGroupMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error('当前未登录，无法标记同名忽略。')
      if (!selectedFileId)
        throw new Error('当前条目缺少文件锚点，无法标记同名忽略。')
      return createAuthedMiboApi(
        token
      ).createInventoryFileFilenameExclusionRule(
        selectedFileId,
        'advertisement'
      )
    },
    onSuccess: async () => {
      setIgnoreDialogOpen(false)
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['settings', 'scan-exclusions'],
        }),
        queryClient.invalidateQueries({ queryKey: ['inventory-files'] }),
        queryClient.invalidateQueries({
          queryKey: ['library', 'inventory-files'],
        }),
      ])
    },
  })

  const restoreMutation = useMutation({
    mutationFn: async (file: {
      id: number
      rule_id?: number
      exclusion_id?: number
    }) => {
      if (!token) throw new Error('当前未登录，无法恢复文件。')
      if (!file.rule_id) throw new Error('当前条目缺少忽略规则，无法恢复。')
      return createAuthedMiboApi(token).restoreFilenameExclusionMatch(
        file.rule_id,
        file.id
      )
    },
    onSuccess: async () => {
      setIgnoreDialogOpen(false)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['inventory-files'] }),
        queryClient.invalidateQueries({
          queryKey: ['settings', 'scan-exclusions'],
        }),
      ])
    },
  })

  const restoreRuleMutation = useMutation({
    mutationFn: async (ruleId: number) => {
      if (!token) throw new Error('当前未登录，无法恢复同名文件。')
      return createAuthedMiboApi(token).setFilenameExclusionRuleEnabled(
        ruleId,
        false
      )
    },
    onSuccess: async () => {
      setIgnoreDialogOpen(false)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['inventory-files'] }),
        queryClient.invalidateQueries({
          queryKey: ['settings', 'scan-exclusions'],
        }),
      ])
    },
  })

  const previewRestoreMutation = useMutation({
    mutationFn: async (file: { id: number; ruleId?: number }) => {
      if (!token) throw new Error('当前未登录，无法预览恢复影响。')
      if (!file.ruleId)
        throw new Error('当前条目缺少忽略规则，无法预览恢复影响。')
      setDialogMode('restore')
      setSelectedFileId(file.id)
      setSelectedRuleId(file.ruleId)
      return createAuthedMiboApi(token).previewInventoryFileScanExclusion(
        file.id
      )
    },
    onSuccess: (preview) => {
      setIgnorePreview(preview)
      setIgnoreDialogOpen(true)
    },
  })

  return (
    <section className='flex min-h-0 flex-1 flex-col overflow-hidden'>
      <div className='mb-4 flex shrink-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
        <div>
          <h3 className='text-base font-medium'>媒体文件浏览</h3>
          {token && !isLoading && files.length > 0 ? (
            <div className='mt-1 flex items-center gap-3 text-sm text-muted-foreground'>
              <span>共 {total} 个文件</span>
              <span>第 {page} 页</span>
            </div>
          ) : null}
        </div>
        <div className='flex w-full flex-col gap-2 sm:w-auto sm:flex-row'>
          <form className='flex w-full gap-2 sm:w-auto' onSubmit={submitQuery}>
            <Input
              value={queryDraft}
              onChange={(event) => setQueryDraft(event.currentTarget.value)}
              className='w-full sm:w-56'
              placeholder='搜索文件名或路径'
              aria-label='搜索文件名或路径'
            />
            <Button type='submit' variant='outline' disabled={isLoading}>
              <SearchIcon className='size-4' />
              搜索
            </Button>
          </form>
          <NativeSelect
            value={selectedLibraryId ? String(selectedLibraryId) : ''}
            onChange={(event) => onLibraryChange(event.currentTarget.value)}
            className='w-full sm:w-56'
            aria-label='选择媒体库'
          >
            <NativeSelectOption value=''>全部媒体库</NativeSelectOption>
            {libraries.map((library) => (
              <NativeSelectOption key={library.id} value={String(library.id)}>
                {library.name}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </div>
      </div>

      <div className='min-h-0 flex-1 overflow-y-auto pr-1'>
        {!token ? (
          <div className='rounded-[1.35rem] border border-dashed border-border/70 bg-background/60 px-6 py-14 text-center text-sm text-muted-foreground'>
            需要登录后才能浏览媒体文件。
          </div>
        ) : isLoading ? (
          <div className='rounded-[1.35rem] border border-border/60 bg-background/80 p-4 shadow-sm'>
            <div className='grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6'>
              {Array.from({ length: 12 }).map((_, index) => (
                <div
                  key={index}
                  className='h-[360px] rounded-[1.35rem] bg-muted/40'
                />
              ))}
            </div>
          </div>
        ) : files.length === 0 ? (
          <div className='rounded-[1.35rem] border border-dashed border-border/70 bg-background/60 px-6 py-14 text-center text-sm text-muted-foreground'>
            这个媒体库暂无文件。
          </div>
        ) : (
          <div className='space-y-4'>
            <div className='grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6'>
              {files.map((file) => (
                <InventoryFileCard
                  key={file.id}
                  file={file}
                  onEdit={() =>
                    file.excluded
                      ? previewRestoreMutation.mutate({
                          id: file.id,
                          ruleId: file.filename_rule_id ?? file.exclusion_id,
                        })
                      : previewIgnoreMutation.mutate(file.id)
                  }
                  pending={
                    previewIgnoreMutation.isPending ||
                    previewRestoreMutation.isPending
                  }
                />
              ))}
            </div>
          </div>
        )}
      </div>

      {token && !isLoading && files.length > 0 ? (
        <div className='mt-4 flex shrink-0 items-center justify-end gap-2 border-t border-border/60 pt-4'>
          <Button
            variant='outline'
            disabled={page <= 1}
            onClick={() => onPageChange(Math.max(1, page - 1))}
          >
            上一页
          </Button>
          <Button
            variant='outline'
            disabled={!hasMore}
            onClick={() => onPageChange(page + 1)}
          >
            下一页
          </Button>
        </div>
      ) : null}

      <Dialog open={ignoreDialogOpen} onOpenChange={setIgnoreDialogOpen}>
        <DialogContent className='grid max-h-[85vh] w-[calc(100vw-2rem)] max-w-2xl grid-rows-[auto_minmax(0,1fr)_auto] overflow-hidden p-0'>
          <DialogHeader>
            <div className='space-y-2 px-6 pt-6'>
              <DialogTitle>选择忽略范围</DialogTitle>
              <DialogDescription>
                {dialogMode === 'restore'
                  ? '先确认同名文件影响范围，再选择只恢复当前文件或恢复所有同名文件。'
                  : '先确认同名文件影响范围，再选择只忽略当前文件或忽略所有同名文件。'}
              </DialogDescription>
            </div>
          </DialogHeader>
          <div className='min-h-0 overflow-y-auto px-6 py-4'>
            {ignorePreview ? (
              <div className='min-w-0 space-y-3'>
                <div className='min-w-0 rounded-xl border border-border/60 bg-muted/40 p-3 text-sm'>
                  <div className='font-medium break-all'>
                    {ignorePreview.normalized_filename}
                  </div>
                  <div className='mt-1 break-all text-muted-foreground'>
                    {ignorePreview.library_name ||
                      `#${ignorePreview.library_id}`}{' '}
                    / {ignorePreview.storage_provider}，共影响{' '}
                    {ignorePreview.affected_count} 个文件
                  </div>
                </div>
                <div className='max-h-64 min-w-0 space-y-2 overflow-y-auto rounded-xl border border-border/60 p-3'>
                  {ignorePreview.affected_files.map((file) => (
                    <div
                      key={file.id}
                      className='text-xs break-all text-muted-foreground'
                      title={file.storage_path}
                    >
                      {file.storage_path}
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
          <div className='flex flex-col gap-2 border-t border-border/60 bg-muted/30 px-6 py-4 sm:flex-row sm:justify-end'>
            <Button
              variant='outline'
              className='w-full sm:w-auto'
              disabled={
                ignoreMutation.isPending ||
                filenameGroupMutation.isPending ||
                restoreMutation.isPending
              }
              onClick={() =>
                dialogMode === 'restore'
                  ? restoreMutation.mutate({
                      id: selectedFileId ?? 0,
                      rule_id: selectedRuleId ?? undefined,
                    })
                  : ignoreMutation.mutate()
              }
            >
              {dialogMode === 'restore' ? '恢复当前文件' : '仅忽略当前文件'}
            </Button>
            <Button
              variant={dialogMode === 'restore' ? 'outline' : 'destructive'}
              className='w-full sm:w-auto'
              disabled={
                ignoreMutation.isPending ||
                filenameGroupMutation.isPending ||
                restoreMutation.isPending ||
                restoreRuleMutation.isPending
              }
              onClick={() =>
                dialogMode === 'restore'
                  ? restoreRuleMutation.mutate(selectedRuleId ?? 0)
                  : filenameGroupMutation.mutate()
              }
            >
              {dialogMode === 'restore' ? '恢复同名文件' : '忽略所有同名文件'}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </section>
  )
}

function InventoryFileCard({
  file,
  onEdit,
  pending,
}: {
  file: {
    id: number
    storage_path: string
    storage_provider: string
    thumbnail_url?: string
    excluded?: boolean
    exclusion_id?: number
    filename_rule_id?: number
  }
  onEdit: () => void
  pending: boolean
}) {
  const fileName = fileNameFromPath(file.storage_path)
  return (
    <article className='overflow-hidden rounded-[1.35rem] border border-border/60 bg-background/80 shadow-sm'>
      <div className='relative aspect-[2/3] bg-muted'>
        {file.thumbnail_url ? (
          <img
            src={file.thumbnail_url}
            alt=''
            loading='lazy'
            decoding='async'
            className='h-full w-full object-cover'
          />
        ) : null}
        {file.excluded ? (
          <span className='absolute top-2 right-2 rounded-full bg-amber-500/90 px-2 py-0.5 text-[11px] font-medium text-white shadow-sm'>
            已忽略
          </span>
        ) : null}
      </div>
      <div className='space-y-1 p-3'>
        <div className='flex items-start justify-between gap-2'>
          <div className='truncate text-sm font-medium' title={fileName}>
            {fileName}
          </div>
        </div>
        <div
          className='truncate text-xs text-muted-foreground'
          title={file.storage_path}
        >
          {file.storage_path}
        </div>
        <div className='text-[11px] text-muted-foreground'>
          {file.storage_provider}
        </div>
        <Button
          size='sm'
          variant='outline'
          className='mt-2 w-full'
          disabled={pending}
          onClick={onEdit}
        >
          {file.excluded ? '恢复' : '忽略'}
        </Button>
      </div>
    </article>
  )
}

function FilenameRulesList({
  rules,
  pending,
  onToggle,
  onRestoreMember,
  onUnrestoreMember,
}: {
  rules: FilenameExclusionRule[]
  pending: boolean
  onToggle: (rule: FilenameExclusionRule) => void
  onRestoreMember: (rule: FilenameExclusionRule, fileId: number) => void
  onUnrestoreMember: (rule: FilenameExclusionRule, fileId: number) => void
}) {
  return (
    <div className='space-y-3'>
      {rules.map((rule) => (
        <div
          key={rule.id}
          className='rounded-[1.35rem] border border-border/60 bg-background/80 p-4 shadow-sm'
        >
          <div className='flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between'>
            <div className='space-y-1'>
              <div className='flex flex-wrap items-center gap-2'>
                <span className='font-medium'>{rule.normalized_filename}</span>
                <Badge variant={rule.enabled ? 'default' : 'outline'}>
                  {rule.enabled ? '同名规则生效中' : '已恢复'}
                </Badge>
                <Badge variant='outline'>{rule.affected_count} 个文件</Badge>
              </div>
              <p className='text-sm text-muted-foreground'>
                所有来源 / {reasonLabel(rule.reason)}
              </p>
              <p className='text-xs text-muted-foreground'>
                恢复后不会立即重建媒体，后续扫描会重新允许这些文件进入导入流程。
              </p>
            </div>
            <Button
              size='sm'
              variant={rule.enabled ? 'outline' : 'default'}
              disabled={pending}
              onClick={() => onToggle(rule)}
            >
              {rule.enabled ? '恢复同名规则' : '重新启用'}
            </Button>
          </div>
          <div className='mt-4 space-y-2'>
            {rule.affected_files.map((file) => (
              <div
                key={file.id}
                className='flex flex-col gap-2 rounded-xl border border-border/50 bg-muted/30 px-3 py-2 sm:flex-row sm:items-center sm:justify-between'
              >
                <div className='min-w-0'>
                  <div className='truncate text-sm' title={file.storage_path}>
                    {file.storage_path}
                  </div>
                  <div className='text-xs text-muted-foreground'>
                    {file.restored ? '已单独恢复' : '被同名规则排除'} /{' '}
                    {file.status}
                  </div>
                </div>
                {file.restored ? (
                  <Button
                    size='sm'
                    variant='outline'
                    disabled={pending}
                    onClick={() => onUnrestoreMember(rule, file.id)}
                  >
                    排除此文件
                  </Button>
                ) : (
                  <Button
                    size='sm'
                    variant='outline'
                    disabled={pending || !rule.enabled}
                    onClick={() => onRestoreMember(rule, file.id)}
                  >
                    恢复此文件
                  </Button>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function ExclusionsTable({
  exclusions,
  pending,
  onToggle,
}: {
  exclusions: ScanExclusion[]
  pending: boolean
  onToggle: (exclusion: ScanExclusion) => void
}) {
  return (
    <div className='flex min-h-0 flex-1 flex-col overflow-hidden rounded-[1.35rem] border border-border/60 bg-background/80 shadow-sm'>
      <div className='min-h-0 flex-1 overflow-auto'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className='min-w-64'>文件</TableHead>
              <TableHead>媒体库</TableHead>
              <TableHead>原因</TableHead>
              <TableHead>存储</TableHead>
              <TableHead className='min-w-56'>稳定标识</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead>状态</TableHead>
              <TableHead className='text-right'>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {exclusions.map((exclusion) => (
              <TableRow key={exclusion.id}>
                <TableCell className='max-w-80'>
                  <div className='space-y-1'>
                    <div
                      className='truncate font-medium'
                      title={exclusion.storage_path}
                    >
                      {fileNameFromPath(exclusion.storage_path)}
                    </div>
                    <div
                      className='truncate text-xs text-muted-foreground'
                      title={exclusion.storage_path || '未记录'}
                    >
                      {exclusion.storage_path || '未记录'}
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  {exclusion.library_name || `#${exclusion.library_id}`}
                </TableCell>
                <TableCell>{reasonLabel(exclusion.reason)}</TableCell>
                <TableCell>{exclusion.storage_provider || '未知'}</TableCell>
                <TableCell className='max-w-64 truncate font-mono text-xs'>
                  {exclusion.stable_identity_key || '路径回退'}
                </TableCell>
                <TableCell>{formatDateTime(exclusion.updated_at)}</TableCell>
                <TableCell>
                  <Badge variant={exclusion.enabled ? 'default' : 'outline'}>
                    {exclusion.enabled ? '生效中' : '已恢复'}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className='flex justify-end'>
                    <Button
                      size='sm'
                      variant={exclusion.enabled ? 'outline' : 'default'}
                      disabled={pending}
                      onClick={() => onToggle(exclusion)}
                    >
                      {pending ? (
                        <Loader2Icon className='size-4 animate-spin' />
                      ) : null}
                      {exclusion.enabled ? '恢复' : '重新启用'}
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}

function ExclusionSkeleton() {
  return (
    <div className='rounded-[1.35rem] border border-border/60 bg-background/80 p-4 shadow-sm'>
      <div className='space-y-3'>
        {Array.from({ length: 6 }).map((_, index) => (
          <Skeleton key={index} className='h-10 rounded-xl' />
        ))}
      </div>
    </div>
  )
}

function EmptyState() {
  return (
    <div className='flex min-h-[260px] flex-col items-center justify-center rounded-[1.35rem] border border-dashed border-border/70 bg-background/60 p-8 text-center'>
      <FileX2Icon className='size-10 text-muted-foreground' />
      <h4 className='mt-4 text-base font-medium'>暂无扫描排除项</h4>
      <p className='mt-2 max-w-md text-sm leading-6 text-muted-foreground'>
        当你从媒体详情、资产或文件操作中标记广告/误导入文件后，它们会出现在这里。
      </p>
    </div>
  )
}

function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className='flex min-h-[260px] flex-col items-center justify-center rounded-[1.35rem] border border-dashed border-destructive/30 bg-destructive/5 p-8 text-center'>
      <AlertTriangleIcon className='size-10 text-destructive' />
      <h4 className='mt-4 text-base font-medium'>无法加载扫描排除项</h4>
      <p className='mt-2 max-w-md text-sm leading-6 text-muted-foreground'>
        请检查当前登录状态或稍后重试。
      </p>
      <div className='mt-4'>
        <Button variant='outline' onClick={onRetry}>
          重新加载
        </Button>
      </div>
    </div>
  )
}

function ErrorBanner({ message }: { message: string }) {
  return (
    <div className='flex items-start gap-3 rounded-2xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive'>
      <AlertTriangleIcon className='mt-0.5 size-4 shrink-0' />
      <span>{message}</span>
    </div>
  )
}

function reasonLabel(reason: string) {
  switch (reason) {
    case 'advertisement':
      return '广告'
    case 'unwanted':
      return '不需要'
    case 'duplicate':
      return '重复导入'
    case 'wrong_import':
      return '误导入'
    case 'other':
      return '其他'
    default:
      return reason || '未知'
  }
}

function fileNameFromPath(value: string) {
  const segments = value.split('/').filter(Boolean)
  return segments[segments.length - 1] || value || '未知文件'
}

function formatDateTime(value?: string) {
  if (!value) return '未知'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '未知'
  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

function errorMessage(error: unknown) {
  if (error instanceof Error) return error.message
  return '操作失败，请稍后重试。'
}
