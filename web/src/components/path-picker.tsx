import { useEffect, useState } from 'react'
import {
  ChevronRightIcon,
  FolderTreeIcon,
  LoaderCircleIcon,
  RefreshCcwIcon,
} from 'lucide-react'

import { Button } from '#/components/ui/button'
import { Input } from '#/components/ui/input'
import { ScrollArea } from '#/components/ui/scroll-area'
import { cn } from '#/lib/utils'
import type {StorageBrowseResult} from '#/lib/mibo-api';

type PathPickerProps = {
  browseKey: string
  browseLabel: string
  value: string
  placeholder: string
  disabled?: boolean
  ready?: boolean
  lockedMessage?: string
  onValueChange: (value: string) => void
  browse: ((path?: string) => Promise<StorageBrowseResult>) | null
}

export function PathPicker({
  browseKey,
  browseLabel,
  value,
  placeholder,
  disabled = false,
  ready = true,
  lockedMessage,
  onValueChange,
  browse,
}: PathPickerProps) {
  const [browserState, setBrowserState] = useState<StorageBrowseResult | null>(
    null,
  )
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  async function loadPath(targetPath?: string) {
    if (!browse) {
      setBrowserState(null)
      return
    }

    setIsLoading(true)
    setErrorMessage(null)
    try {
      const result = await browse(targetPath)
      setBrowserState(result)
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '无法浏览路径')
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    setBrowserState(null)
    setErrorMessage(null)
    if (browse && ready) {
      void loadPath(value.trim() || undefined)
    }
  }, [browseKey, browse, ready])

  const isLocked = !ready || browse === null

  return (
    <div className="grid gap-3">
      <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto_auto] sm:items-center">
        <Input
          value={value}
          onChange={(event) => onValueChange(event.target.value)}
          placeholder={placeholder}
          disabled={disabled}
        />
        <Button
          type="button"
          variant="outline"
          disabled={disabled || isLocked || isLoading}
          onClick={() => void loadPath(value.trim() || undefined)}
        >
          {isLoading ? (
            <LoaderCircleIcon className="size-4 animate-spin" />
          ) : (
            <RefreshCcwIcon className="size-4" />
          )}
          浏览当前路径
        </Button>
        <Button
          type="button"
          variant="secondary"
          disabled={disabled || isLocked || browserState === null}
          onClick={() => {
            if (browserState) {
              onValueChange(browserState.current_path)
            }
          }}
        >
          <FolderTreeIcon className="size-4" />
          选择当前目录
        </Button>
      </div>

      <div className="rounded-xl border border-border/70 bg-muted/20 p-3">
        <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
          <div>
            {browseLabel}
            {browserState ? `：${browserState.current_path}` : ''}
          </div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
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

        <div className="mt-1 text-xs text-muted-foreground">
          {browserState
            ? `可选根路径：${browserState.root_path}`
            : isLocked
              ? (lockedMessage ?? '先完成上一步，再选择路径。')
              : '正在准备路径浏览器。'}
        </div>

        {errorMessage ? (
          <div className="mt-3 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive">
            {errorMessage}
          </div>
        ) : null}

        <ScrollArea className="mt-3 h-44 rounded-lg border bg-background/80">
          <div className="grid gap-1 p-2">
            {isLocked ? (
              <div className="px-3 py-6 text-sm text-muted-foreground">
                {lockedMessage ?? '先完成上一步，再选择路径。'}
              </div>
            ) : browserState?.items.length ? (
              browserState.items.map((item) => (
                <Button
                  key={item.path}
                  type="button"
                  variant="ghost"
                  className={cn(
                    'h-auto w-full justify-between px-3 py-2 text-left text-sm whitespace-normal',
                    value === item.path && 'bg-muted',
                  )}
                  onClick={() => void loadPath(item.path)}
                  disabled={disabled || isLoading}
                >
                  <div className="min-w-0">
                    <div className="truncate font-medium">{item.name}</div>
                    <div className="truncate text-xs text-muted-foreground">
                      {item.path}
                    </div>
                  </div>
                  <ChevronRightIcon className="size-4 text-muted-foreground" />
                </Button>
              ))
            ) : (
              <div className="px-3 py-6 text-sm text-muted-foreground">
                {isLoading ? '正在读取目录...' : '当前目录下没有可选子目录。'}
              </div>
            )}
          </div>
        </ScrollArea>
      </div>
    </div>
  )
}
