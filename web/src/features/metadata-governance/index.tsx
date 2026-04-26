import { Link } from '@tanstack/react-router'
import { LoaderCircleIcon } from 'lucide-react'

import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import { useAuthStore } from '#/stores/auth-store'

import { MetadataGovernanceDetail } from './detail'
import { MetadataGovernanceWorkspace } from './workspace'

export default function MetadataGovernancePage({
  mediaItemId,
}: {
  mediaItemId?: number
}) {
  const token = useAuthStore((state) => state.token)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)

  if (!hasHydrated) {
    return (
      <div className="flex items-center gap-3 rounded-[1.5rem] border border-border/60 bg-card/80 px-5 py-4 text-foreground shadow-sm">
        <LoaderCircleIcon className="size-4 animate-spin" />
        <span className="text-sm text-muted-foreground">
          正在准备治理工作台
        </span>
      </div>
    )
  }

  if (!token) {
    return (
      <div className="rounded-[1.5rem] border border-border/60 bg-card/80 px-6 py-8 text-foreground shadow-sm">
        <div className="max-w-xl space-y-4">
          <Badge variant="outline" className="border-border/60 bg-card/80">
            Metadata Governance
          </Badge>
          <h1 className="text-2xl font-semibold tracking-tight">
            登录后进入元数据治理
          </h1>
          <p className="text-sm leading-7 text-muted-foreground">
            该页面需要管理员会话访问媒体详情、匹配候选和后台治理动作。
          </p>
          <Button asChild>
            <Link
              to="/login"
              search={{
                redirect: mediaItemId
                  ? `/settings/metadata/${mediaItemId}`
                  : '/settings/metadata',
              }}
            >
              前往登录
            </Link>
          </Button>
        </div>
      </div>
    )
  }

  if (
    typeof mediaItemId === 'number' &&
    Number.isFinite(mediaItemId) &&
    mediaItemId > 0
  ) {
    return <MetadataGovernanceDetail token={token} mediaItemId={mediaItemId} />
  }

  return <MetadataGovernanceWorkspace token={token} />
}
