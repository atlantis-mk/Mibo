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
      <div className="flex min-h-svh items-center justify-center bg-background text-foreground">
        <div className="flex items-center gap-3 rounded-full border border-border/50 bg-card/85 px-5 py-3">
          <LoaderCircleIcon className="size-4 animate-spin" />
          <span className="text-sm text-muted-foreground">
            正在准备治理工作台
          </span>
        </div>
      </div>
    )
  }

  if (!token) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background px-6 text-foreground">
        <div className="max-w-xl space-y-4 text-center">
          <Badge variant="outline" className="border-border/60 bg-card/80">
            Metadata Governance
          </Badge>
          <h1 className="text-3xl font-semibold tracking-tight">
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
                  ? `/metadata/${mediaItemId}`
                  : '/metadata',
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
