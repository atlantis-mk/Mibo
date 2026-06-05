import { useCallback, useEffect, useRef, useState } from 'react'
import {
  ChevronRightIcon,
  FolderTreeIcon,
  LoaderCircleIcon,
  RefreshCcwIcon,
  SearchIcon,
} from 'lucide-react'
import type { StorageBrowseResult } from '@/lib/mibo-api'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'

type BrowseOptions = {
  refresh?: boolean
}

type PathPickerProps = {
  browseKey: string
  browseLabel: string
  value: string
  placeholder: string
  disabled?: boolean
  ready?: boolean
  lockedMessage?: string
  selectCurrentOnBrowse?: boolean
  onValueChange: (value: string) => void
  browse:
    | ((path?: string, options?: BrowseOptions) => Promise<StorageBrowseResult>)
    | null
}

export function PathPicker({
  browseKey,
  browseLabel,
  value,
  placeholder,
  disabled = false,
  ready = true,
  lockedMessage,
  selectCurrentOnBrowse = false,
  onValueChange,
  browse,
}: PathPickerProps) {
  const [browserState, setBrowserState] = useState<StorageBrowseResult | null>(
    null
  )
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const browseRef = useRef(browse)
  const onValueChangeRef = useRef(onValueChange)
  const valueRef = useRef(value)
  const canBrowse = browse !== null

  useEffect(() => {
    browseRef.current = browse
    onValueChangeRef.current = onValueChange
    valueRef.current = value
  }, [browse, onValueChange, value])

  const loadPath = useCallback(
    async (targetPath?: string, options?: BrowseOptions) => {
      const browsePath = browseRef.current
      if (!browsePath) {
        setBrowserState(null)
        return
      }

      setIsLoading(true)
      setErrorMessage(null)
      try {
        const result = await browsePath(targetPath, options)
        setBrowserState(result)
        if (selectCurrentOnBrowse && result.current_path !== valueRef.current) {
          onValueChangeRef.current(result.current_path)
        }
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : '无法浏览路径')
      } finally {
        setIsLoading(false)
      }
    },
    [selectCurrentOnBrowse]
  )

  useEffect(() => {
    setBrowserState(null)
    setErrorMessage(null)
    if (canBrowse && ready) {
      void loadPath(value.trim() || undefined)
    }
  }, [browseKey, canBrowse, loadPath, ready, value])

  const isLocked = !ready || browse === null
  const normalizedSearchQuery = searchQuery.trim().toLocaleLowerCase()
  const visibleItems = browserState?.items.filter((item) => {
    if (!normalizedSearchQuery) return true

    return [item.name, item.path].some((value) =>
      value.toLocaleLowerCase().includes(normalizedSearchQuery)
    )
  })

  return (
    <div className='grid min-w-0 gap-3'>
      <div
        className={cn(
          'grid gap-2 sm:items-center',
          selectCurrentOnBrowse
            ? 'sm:grid-cols-[minmax(0,1fr)_auto]'
            : 'sm:grid-cols-[minmax(0,1fr)_auto_auto]'
        )}
      >
        <Input
          value={value}
          onChange={(event) => onValueChange(event.target.value)}
          placeholder={placeholder}
          disabled={disabled}
        />
        <Button
          type='button'
          variant='outline'
          disabled={disabled || isLocked || isLoading}
          onClick={() =>
            void loadPath(value.trim() || undefined, { refresh: true })
          }
        >
          {isLoading ? (
            <LoaderCircleIcon className='size-4 animate-spin' />
          ) : (
            <RefreshCcwIcon className='size-4' />
          )}
          刷新当前路径
        </Button>
        {selectCurrentOnBrowse ? null : (
          <Button
            type='button'
            variant='secondary'
            disabled={disabled || isLocked || browserState === null}
            onClick={() => {
              if (browserState) {
                onValueChange(browserState.current_path)
              }
            }}
          >
            <FolderTreeIcon className='size-4' />
            选择当前目录
          </Button>
        )}
      </div>

      <div className='min-w-0 overflow-hidden rounded-xl border border-border/70 bg-muted/20 p-3'>
        <div className='flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground'>
          <div className='min-w-0 truncate'>
            {browseLabel}
            {browserState ? `：${browserState.current_path}` : ''}
          </div>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            disabled={
              disabled ||
              isLocked ||
              isLoading ||
              browserState?.parent_path === undefined
            }
            onClick={() => void loadPath(browserState?.parent_path)}
          >
            上一级
          </Button>
        </div>

        <div className='mt-1 text-xs text-muted-foreground'>
          {browserState
            ? `可选根路径：${browserState.root_path}`
            : isLocked
              ? (lockedMessage ?? '先完成上一步，再选择路径。')
              : '正在准备路径浏览器。'}
        </div>

        {errorMessage ? (
          <div className='mt-3 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive'>
            {errorMessage}
          </div>
        ) : null}

        <div className='relative mt-3'>
          <SearchIcon className='pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground' />
          <Input
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            placeholder='搜索当前目录下的路径'
            className='pl-9'
            disabled={disabled || isLocked || browserState === null}
          />
        </div>

        <ScrollArea className='mt-3 h-80 min-w-0 overflow-hidden rounded-lg border bg-background/80'>
          <div className='grid min-w-0 gap-1 p-2'>
            {isLocked ? (
              <div className='px-3 py-6 text-sm text-muted-foreground'>
                {lockedMessage ?? '先完成上一步，再选择路径。'}
              </div>
            ) : visibleItems?.length ? (
              visibleItems.map((item) => (
                <Button
                  key={item.path}
                  type='button'
                  variant='ghost'
                  className={cn(
                    'h-auto w-full min-w-0 justify-between overflow-hidden px-3 py-2 text-left text-sm whitespace-normal',
                    value === item.path && 'bg-muted'
                  )}
                  onClick={() => void loadPath(item.path)}
                  disabled={disabled || isLoading}
                >
                  <div className='min-w-0 flex-1'>
                    <div className='truncate font-medium'>{item.name}</div>
                    <div className='truncate text-xs text-muted-foreground'>
                      {item.path}
                    </div>
                  </div>
                  <ChevronRightIcon className='size-4 shrink-0 text-muted-foreground' />
                </Button>
              ))
            ) : (
              <div className='px-3 py-6 text-sm text-muted-foreground'>
                {isLoading
                  ? '正在读取目录...'
                  : normalizedSearchQuery
                    ? '当前目录下没有匹配的子目录。'
                    : '当前目录下没有可选子目录。'}
              </div>
            )}
          </div>
        </ScrollArea>
      </div>
    </div>
  )
}
