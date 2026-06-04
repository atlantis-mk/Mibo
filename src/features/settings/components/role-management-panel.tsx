import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2Icon, PencilIcon, PlusIcon, ShieldIcon, Trash2Icon } from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import type {
  AdminRole,
  CreateAdminRoleInput,
  LibraryAccessTag,
} from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  miboQueryKeys,
} from '@/lib/mibo-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'

type RoleDraft = CreateAdminRoleInput

const EMPTY_ROLE_DRAFT: RoleDraft = {
  name: '',
  allow_library_tags: [],
  deny_library_tags: [],
}

export function RoleManagementPanel() {
  const queryClient = useQueryClient()
  const token = useAuthStore((state) => state.auth.accessToken)
  const authUser = useAuthStore((state) => state.auth.user)
  const isAdmin = authUser?.role === 'admin'
  const queryToken = token ?? 'guest'

  const rolesQuery = useQuery({
    queryKey: ['admin', 'roles', queryToken],
    queryFn: () => createAuthedMiboApi(queryToken).listAdminRoles(),
    enabled: !!token && isAdmin,
  })
  const roles: AdminRole[] = rolesQuery.data ?? []
  const accessTagsQuery = useQuery({
    queryKey: ['library-access-tags', queryToken],
    queryFn: () => createAuthedMiboApi(queryToken).listLibraryAccessTags(),
    enabled: !!token && isAdmin,
  })
  const availableAccessTags: LibraryAccessTag[] = accessTagsQuery.data ?? []

  const [draft, setDraft] = useState<RoleDraft>(EMPTY_ROLE_DRAFT)
  const [editingRole, setEditingRole] = useState<AdminRole | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<AdminRole | null>(null)
  const [isCreateOpen, setIsCreateOpen] = useState(false)

  const createRoleMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error('当前未登录。')
      return createAuthedMiboApi(token).createAdminRole(draft)
    },
    onSuccess: async () => {
      toast.success('角色已创建。')
      setDraft(EMPTY_ROLE_DRAFT)
      setIsCreateOpen(false)
      await queryClient.invalidateQueries({ queryKey: ['admin', 'roles', queryToken] })
      await queryClient.invalidateQueries({ queryKey: miboQueryKeys.adminUsers(queryToken) })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '创建角色失败。')
    },
  })

  const updateRoleMutation = useMutation({
    mutationFn: async () => {
      if (!token || !editingRole) throw new Error('当前未选择角色。')
      return createAuthedMiboApi(token).updateAdminRole(editingRole.id, {
        name: editingRole.name,
        allow_library_tags: editingRole.allow_library_tags ?? [],
        deny_library_tags: editingRole.deny_library_tags ?? [],
      })
    },
    onSuccess: async () => {
      toast.success('角色已更新。')
      setEditingRole(null)
      await queryClient.invalidateQueries({ queryKey: ['admin', 'roles', queryToken] })
      await queryClient.invalidateQueries({ queryKey: miboQueryKeys.adminUsers(queryToken) })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '更新角色失败。')
    },
  })

  const deleteRoleMutation = useMutation({
    mutationFn: async () => {
      if (!token || !deleteTarget) throw new Error('当前未选择角色。')
      return createAuthedMiboApi(token).deleteAdminRole(deleteTarget.id)
    },
    onSuccess: async () => {
      toast.success('角色已删除。')
      setDeleteTarget(null)
      await queryClient.invalidateQueries({ queryKey: ['admin', 'roles', queryToken] })
      await queryClient.invalidateQueries({ queryKey: miboQueryKeys.adminUsers(queryToken) })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '删除角色失败。')
    },
  })

  const roleCountText = useMemo(() => `${roles.length} 个角色`, [roles.length])

  return (
    <div className='space-y-4'>
      <section className='rounded-[1.5rem] border border-border/60 bg-card/80 px-5 py-4 shadow-sm backdrop-blur-sm'>
        <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
          <div className='flex items-start gap-3'>
            <div className='flex size-10 shrink-0 items-center justify-center rounded-xl border border-border/60 bg-background/70'>
              <ShieldIcon className='size-4 text-muted-foreground' />
            </div>
            <div className='min-w-0'>
              <h2 className='text-xl font-semibold tracking-tight'>角色</h2>
              <p className='mt-1 text-sm leading-6 text-muted-foreground'>
                管理角色定义，供用户分配使用。
              </p>
            </div>
          </div>
          <div className='flex items-center gap-2'>
            <div className='rounded-full border border-border/60 bg-background/70 px-3 py-1.5 text-sm text-muted-foreground'>
              {roleCountText}
            </div>
            <Button onClick={() => setIsCreateOpen(true)}>
              <PlusIcon className='size-4' />
              新建角色
            </Button>
          </div>
        </div>
      </section>

      <Card className='rounded-[1.5rem] border-border/60 bg-card/80 shadow-sm backdrop-blur-sm'>
        <CardHeader>
          <CardTitle className='flex items-center gap-2'>
            <ShieldIcon className='size-4 text-emerald-500' />
            角色列表
          </CardTitle>
          <CardDescription>编辑名称或删除未被使用的角色。</CardDescription>
        </CardHeader>
        <CardContent className='space-y-3'>
          {!isAdmin ? (
            <div className='text-sm text-muted-foreground'>当前账号无权限查看角色管理。</div>
          ) : rolesQuery.isLoading ? (
            <div className='flex items-center gap-2 text-sm text-muted-foreground'>
              <Loader2Icon className='size-4 animate-spin' />加载中...
            </div>
          ) : roles.length === 0 ? (
            <div className='text-sm text-muted-foreground'>暂无角色。</div>
          ) : (
            roles.map((role) => (
              <div key={role.id} className='flex items-center justify-between rounded-2xl border border-border/60 bg-background/70 px-4 py-3'>
                <div>
                  <div className='font-medium'>{role.name}</div>
                  <div className='text-xs text-muted-foreground'>ID #{role.id}</div>
                  <div className='mt-1 text-xs text-muted-foreground'>
                    允许: {(role.allow_library_tags ?? []).join(', ') || '未配置'}
                  </div>
                  <div className='text-xs text-muted-foreground'>
                    拒绝: {(role.deny_library_tags ?? []).join(', ') || '未配置'}
                  </div>
                </div>
                <div className='flex items-center gap-2'>
                  <Button variant='outline' size='sm' onClick={() => setEditingRole(role)}>
                    <PencilIcon className='size-4' />
                    编辑
                  </Button>
                  <Button variant='destructive' size='sm' onClick={() => setDeleteTarget(role)}>
                    <Trash2Icon className='size-4' />
                    删除
                  </Button>
                </div>
              </div>
            ))
          )}
        </CardContent>
      </Card>

      <RoleEditorDialog
        title='新建角色'
        description='创建一个新的角色名称。'
        open={isCreateOpen}
        draft={draft}
        availableAccessTags={availableAccessTags}
        onDraftChange={setDraft}
        onSubmit={() => createRoleMutation.mutate()}
        isSubmitting={createRoleMutation.isPending}
        onOpenChange={(open) => {
          setIsCreateOpen(open)
          if (!open) setDraft(EMPTY_ROLE_DRAFT)
        }}
      />

      <RoleEditorDialog
        title='编辑角色'
        description={editingRole ? `修改 ${editingRole.name} 的名称。` : ''}
        open={!!editingRole}
        draft={editingRole ?? EMPTY_ROLE_DRAFT}
        availableAccessTags={availableAccessTags}
        onDraftChange={(nextDraft) =>
          setEditingRole((current) => (current ? { ...current, ...nextDraft } : current))
        }
        onSubmit={() => updateRoleMutation.mutate()}
        isSubmitting={updateRoleMutation.isPending}
        onOpenChange={(open) => !open && setEditingRole(null)}
      />

      <ConfirmDeleteDialog
        open={!!deleteTarget}
        roleName={deleteTarget?.name ?? ''}
        isSubmitting={deleteRoleMutation.isPending}
        onConfirm={() => deleteRoleMutation.mutate()}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      />
    </div>
  )
}

function RoleEditorDialog({
  title,
  description,
  open,
  draft,
  availableAccessTags,
  onDraftChange,
  onSubmit,
  isSubmitting,
  onOpenChange,
}: {
  title: string
  description: string
  open: boolean
  draft: RoleDraft
  availableAccessTags: LibraryAccessTag[]
  onDraftChange: (draft: RoleDraft) => void
  onSubmit: () => void
  isSubmitting: boolean
  onOpenChange: (open: boolean) => void
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <div className='space-y-4'>
          <Input
            value={draft.name}
            onChange={(event) =>
              onDraftChange({ ...draft, name: event.target.value })
            }
            placeholder='例如 moderator'
          />
          <TagRuleSelector
            label='允许标签'
            emptyText='未配置，沿用默认开放规则。'
            tags={availableAccessTags}
            selectedTags={draft.allow_library_tags ?? []}
            onChange={(allowLibraryTags) =>
              onDraftChange({
                ...draft,
                allow_library_tags: allowLibraryTags,
              })
            }
          />
          <TagRuleSelector
            label='拒绝标签'
            emptyText='未配置，不额外拦截任何标签。'
            tags={availableAccessTags}
            selectedTags={draft.deny_library_tags ?? []}
            onChange={(denyLibraryTags) =>
              onDraftChange({
                ...draft,
                deny_library_tags: denyLibraryTags,
              })
            }
          />
          <DialogFooter>
            <Button variant='outline' onClick={() => onOpenChange(false)}>取消</Button>
            <Button onClick={onSubmit} disabled={isSubmitting}>
              {isSubmitting ? <Loader2Icon className='size-4 animate-spin' /> : null}
              保存
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function TagRuleSelector({
  label,
  emptyText,
  tags,
  selectedTags,
  onChange,
}: {
  label: string
  emptyText: string
  tags: LibraryAccessTag[]
  selectedTags: string[]
  onChange: (tags: string[]) => void
}) {
  return (
    <div className='space-y-3'>
      <div className='space-y-1'>
        <div className='text-sm font-medium'>{label}</div>
        <p className='text-sm leading-6 text-muted-foreground'>
          {selectedTags.length > 0 ? selectedTags.join(', ') : emptyText}
        </p>
      </div>
      {tags.length > 0 ? (
        <div className='flex flex-wrap gap-2'>
          {tags.map((tag) => {
            const active = selectedTags.includes(tag.name)
            return (
              <Button
                key={tag.id}
                type='button'
                variant={active ? 'default' : 'outline'}
                size='sm'
                onClick={() =>
                  onChange(
                    active
                      ? selectedTags.filter((item) => item !== tag.name)
                      : [...selectedTags, tag.name].sort()
                  )
                }
              >
                {tag.name}
              </Button>
            )
          })}
        </div>
      ) : (
        <div className='rounded-xl border border-dashed border-border/60 bg-background/60 px-3 py-2 text-sm text-muted-foreground'>
          还没有可用的访问标签。请先在媒体库设置里创建或复用标签。
        </div>
      )}
    </div>
  )
}

function ConfirmDeleteDialog({
  open,
  roleName,
  isSubmitting,
  onConfirm,
  onOpenChange,
}: {
  open: boolean
  roleName: string
  isSubmitting: boolean
  onConfirm: () => void
  onOpenChange: (open: boolean) => void
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>删除角色</DialogTitle>
          <DialogDescription>确定删除 {roleName} 吗？如果已有用户使用，会被拒绝。</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)}>取消</Button>
          <Button variant='destructive' onClick={onConfirm} disabled={isSubmitting}>
            {isSubmitting ? <Loader2Icon className='size-4 animate-spin' /> : null}
            删除
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
