import { LoaderCircleIcon } from 'lucide-react'
import type {
  LibraryAccessTag,
  MediaSource,
  MetadataProfile,
  MetadataProviderInstance,
  PluginProviderInstance,
} from '@/lib/mibo-api'
import type { createAuthedMiboApi } from '@/lib/mibo-query'
import { Button } from '@/components/ui/button'
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from '@/components/ui/drawer'
import { ScrollArea } from '@/components/ui/scroll-area'
import { LibraryForm, type LibraryFormState } from './library-form'
import { SourceForm, type SourceFormState } from './media-source-form'

const DRAWER_CLASS_NAME =
  'h-[100svh] max-h-[100svh] data-[vaul-drawer-direction=right]:w-[720px] data-[vaul-drawer-direction=right]:max-w-[720px] data-[vaul-drawer-direction=right]:sm:max-w-[720px] max-sm:data-[vaul-drawer-direction=right]:w-full max-sm:data-[vaul-drawer-direction=right]:max-w-[100vw]'

export function MediaSourceDrawer({
  open,
  title,
  description,
  draft,
  onChange,
  api,
  pluginProviderInstances,
  isEditing = false,
  pending,
  disabled,
  submitLabel,
  onOpenChange,
  onSubmit,
}: {
  open: boolean
  title: string
  description: string
  draft: SourceFormState
  onChange: (nextDraft: SourceFormState) => void
  api: ReturnType<typeof createAuthedMiboApi> | null
  pluginProviderInstances: PluginProviderInstance[]
  isEditing?: boolean
  pending: boolean
  disabled: boolean
  submitLabel: string
  onOpenChange: (open: boolean) => void
  onSubmit: () => void
}) {
  return (
    <Drawer direction='right' open={open} onOpenChange={onOpenChange}>
      <DrawerContent className={DRAWER_CLASS_NAME}>
        <DrawerHeader className='border-b border-border/70 text-left'>
          <DrawerTitle>{title}</DrawerTitle>
          <DrawerDescription className='text-sm'>
            {description}
          </DrawerDescription>
        </DrawerHeader>
        <ScrollArea className='min-h-0 flex-1'>
          <div className='grid gap-5 px-4 py-4'>
            <SourceForm
              draft={draft}
              onChange={onChange}
              api={api}
              pluginProviderInstances={pluginProviderInstances}
              isEditing={isEditing}
            />
          </div>
        </ScrollArea>
        <DrawerFooter className='shrink-0 border-t border-border/70 bg-background/95 pb-[calc(env(safe-area-inset-bottom)+1rem)]'>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button onClick={onSubmit} disabled={disabled}>
            {pending ? (
              <LoaderCircleIcon className='size-4 animate-spin' />
            ) : null}
            {submitLabel}
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  )
}

export function LibraryDrawer({
  open,
  draft,
  onChange,
  mediaSources,
  availableAccessTags,
  metadataProfiles,
  metadataProviderInstances,
  api,
  pending,
  disabled,
  onOpenChange,
  onSubmit,
}: {
  open: boolean
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
  mediaSources: MediaSource[]
  availableAccessTags: LibraryAccessTag[]
  metadataProfiles: MetadataProfile[]
  metadataProviderInstances: MetadataProviderInstance[]
  api: ReturnType<typeof createAuthedMiboApi> | null
  pending: boolean
  disabled: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: () => void
}) {
  return (
    <Drawer direction='right' open={open} onOpenChange={onOpenChange}>
      <DrawerContent className={DRAWER_CLASS_NAME}>
        <DrawerHeader className='border-b border-border/70 text-left'>
          <DrawerTitle>添加内容来源</DrawerTitle>
          <DrawerDescription className='text-sm'>
            选择媒体源与目录。
          </DrawerDescription>
        </DrawerHeader>
        <ScrollArea className='min-h-0 flex-1'>
          <div className='grid gap-5 px-4 py-4'>
            <LibraryForm
              draft={draft}
              onChange={onChange}
              mediaSources={mediaSources}
              availableAccessTags={availableAccessTags}
              metadataProfiles={metadataProfiles}
              metadataProviderInstances={metadataProviderInstances}
              api={api}
            />
          </div>
        </ScrollArea>
        <DrawerFooter className='shrink-0 border-t border-border/70 bg-background/95 pb-[calc(env(safe-area-inset-bottom)+1rem)]'>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button onClick={onSubmit} disabled={disabled}>
            {pending ? (
              <LoaderCircleIcon className='size-4 animate-spin' />
            ) : null}
            添加来源
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  )
}
