import type { ReactNode } from 'react'
import { FileVideo, Volume2 } from 'lucide-react'

import { Card, CardContent, CardHeader, CardTitle } from '#/components/ui/card'
import type { CatalogDetailPresentation } from '#/lib/media-presentation'

import {
  formatAssetLabel,
  formatAvailabilityStatus,
  formatProbeStatus,
  formatRuntime,
  getPrimaryCatalogAsset,
} from './standalone-media-detail-utils'

export function SpecsSection({
  detailGroups,
  item,
}: {
  detailGroups: Array<{ title: string; value: string }>
  item: CatalogDetailPresentation
}) {
  const primaryAsset = getPrimaryCatalogAsset(item)
  return (
    <section className="mt-12 grid min-w-0 gap-10 xl:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)]">
      <div className="min-w-0 space-y-6">
        <h2 className="text-[19px] font-semibold text-foreground">其它信息</h2>
        <div className="space-y-7">
          {detailGroups.map((group) => (
            <div key={group.title} className="space-y-2">
              <div className="text-base font-medium text-muted-foreground">
                {group.title}
              </div>
              <div className="text-[17px] leading-8 whitespace-pre-wrap text-muted-foreground/80 [overflow-wrap:anywhere]">
                {group.value}
              </div>
            </div>
          ))}
        </div>
      </div>
      <div className="grid min-w-0 gap-6 self-start lg:grid-cols-2 xl:grid-cols-2">
        <InfoCard
          icon={<FileVideo className="size-4" />}
          title="视频"
          rows={[
            ['标题', formatAssetLabel(primaryAsset)],
            ['资源类型', primaryAsset?.asset_type || '未知'],
            ['质量', primaryAsset?.quality_label || '未知'],
            ['版本', primaryAsset?.edition || '默认'],
            [
              '时长',
              formatRuntime(
                primaryAsset?.duration_seconds || item.runtime_seconds,
              ) || '未知',
            ],
            [
              '探测状态',
              formatProbeStatus(primaryAsset?.probe_status ?? 'pending'),
            ],
            [
              '文件数量',
              primaryAsset ? String(primaryAsset.file_ids.length) : '0',
            ],
          ]}
        />
        <InfoCard
          icon={<Volume2 className="size-4" />}
          title="资源链接"
          rows={[
            ['可用性', formatAvailabilityStatus(item.availability_status)],
            ['治理状态', item.governance_status || 'pending'],
            [
              '链接条目',
              primaryAsset ? String(primaryAsset.links.length) : '0',
            ],
            ['主提供方', item.metadata_provider || '未匹配'],
            ['外部 ID', item.external_id || '未关联'],
          ]}
        />
      </div>
    </section>
  )
}

function InfoCard({
  icon,
  title,
  rows,
}: {
  icon: ReactNode
  title: string
  rows: [string, string][]
}) {
  return (
    <Card className="border-border/40 bg-card/75 text-foreground backdrop-blur-md">
      <CardHeader className="pb-0">
        <CardTitle className="flex items-center gap-2 text-xl font-semibold">
          <span className="rounded-full border border-border/40 bg-background/75 p-2 text-muted-foreground">
            {icon}
          </span>
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 p-6">
        {rows.map(([label, value]) => (
          <div
            key={label}
            className="grid grid-cols-[72px_minmax(0,1fr)] gap-3 text-sm"
          >
            <div className="text-muted-foreground">{label}</div>
            <div className="text-foreground/85 [overflow-wrap:anywhere]">
              {value}
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}
