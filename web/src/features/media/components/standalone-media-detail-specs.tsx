import type { ReactNode } from 'react'
import { Clapperboard, FileVideo, Volume2 } from 'lucide-react'

import { Button } from '#/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '#/components/ui/card'
import type { MediaItemDetail } from '#/lib/mibo-api'

import {
  formatAudioTrackLabel,
  formatBitRate,
  formatChannels,
  formatChannelsCompact,
  formatProbeStatus,
  formatRuntime,
  formatVideoTrackLabel,
  simplifyAspectRatio,
} from './standalone-media-detail-utils'

export function SpecsSection({
  detailGroups,
  item,
  onOpenTrailer,
}: {
  detailGroups: Array<{ title: string; value: string }>
  item: MediaItemDetail
  onOpenTrailer: () => void
}) {
  const primaryFile = item.files[0]
  const audioTracks = Array.isArray(primaryFile?.audio_tracks)
    ? primaryFile.audio_tracks
    : []
  const subtitleTracks = Array.isArray(primaryFile?.subtitle_tracks)
    ? primaryFile.subtitle_tracks
    : []
  const primaryAudioTrack = audioTracks[0]
  const audioSummary = formatAudioTrackLabel(primaryAudioTrack)
  const videoSummary = formatVideoTrackLabel(primaryFile)
  return (
    <section className="mt-12 grid gap-10 xl:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)]">
      <div className="space-y-6">
        <h2 className="text-[19px] font-semibold text-foreground">其它信息</h2>
        <div className="space-y-7">
          {item.trailer ? (
            <TrailerEntryCard item={item} onOpenTrailer={onOpenTrailer} />
          ) : null}
          {detailGroups.map((group) => (
            <div key={group.title} className="space-y-2">
              <div className="text-base font-medium text-muted-foreground">
                {group.title}
              </div>
              <div className="text-[17px] leading-8 whitespace-pre-wrap text-muted-foreground/80">
                {group.value}
              </div>
            </div>
          ))}
        </div>
      </div>
      <div className="grid gap-6 self-start lg:grid-cols-2 xl:grid-cols-2">
        <InfoCard
          icon={<FileVideo className="size-4" />}
          title="视频"
          rows={[
            ['标题', videoSummary],
            ['编码器', primaryFile?.video_codec || '未知'],
            [
              '编码器标签',
              primaryFile?.container
                ? primaryFile.container.toLowerCase()
                : '未知',
            ],
            [
              '配置',
              primaryFile?.bit_rate
                ? formatBitRate(primaryFile.bit_rate)
                : '未知',
            ],
            ['等级', formatProbeStatus(primaryFile?.probe_status ?? 'pending')],
            [
              '分辨率',
              primaryFile?.width && primaryFile?.height
                ? `${primaryFile.width}x${primaryFile.height}`
                : '未知',
            ],
            [
              '宽高比',
              primaryFile?.width && primaryFile?.height
                ? simplifyAspectRatio(primaryFile.width, primaryFile.height)
                : '未知',
            ],
            [
              '时长',
              formatRuntime(
                primaryFile?.duration_seconds || item.runtime_seconds,
              ) || '未知',
            ],
          ]}
        />
        <InfoCard
          icon={<Volume2 className="size-4" />}
          title="音频"
          rows={[
            ['标题', primaryAudioTrack?.title || audioSummary],
            ['语言', primaryAudioTrack?.language || '未标注'],
            ['编码器', primaryAudioTrack?.codec || '未知'],
            [
              '编码器标签',
              primaryAudioTrack?.codec
                ? primaryAudioTrack.codec.toLowerCase()
                : '未知',
            ],
            ['配置', formatChannels(primaryAudioTrack)],
            ['布局', formatChannels(primaryAudioTrack)],
            ['频道', formatChannelsCompact(primaryAudioTrack)],
            ['字幕', `${subtitleTracks.length} 轨`],
            ['默认', primaryAudioTrack ? '是' : '否'],
          ]}
        />
      </div>
    </section>
  )
}

function TrailerEntryCard({
  item,
  onOpenTrailer,
}: {
  item: MediaItemDetail
  onOpenTrailer: () => void
}) {
  const trailer = item.trailer
  if (!trailer) {
    return null
  }

  return (
    <Card className="border-border/40 bg-card/75 text-foreground backdrop-blur-md">
      <CardHeader className="space-y-3 pb-3">
        <CardTitle className="flex items-center gap-2 text-xl font-semibold">
          <span className="rounded-full border border-border/40 bg-background/75 p-2 text-muted-foreground">
            <Clapperboard className="size-4" />
          </span>
          预告片
        </CardTitle>
        <div className="space-y-1 text-sm text-muted-foreground">
          <div className="text-base font-medium text-foreground">
            {trailer.name || '观看预告片'}
          </div>
          <div>
            {trailer.official ? '官方 Trailer' : trailer.type || 'Trailer'}
            {trailer.language ? ` · ${trailer.language.toUpperCase()}` : ''}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4 p-6 pt-0">
        {trailer.thumbnail ? (
          <button
            type="button"
            className="group relative block aspect-video w-full overflow-hidden rounded-2xl border border-border/50 bg-muted text-left"
            onClick={onOpenTrailer}
          >
            <img
              src={trailer.thumbnail}
              alt={`${item.title} trailer thumbnail`}
              className="h-full w-full object-cover transition duration-300 group-hover:scale-[1.02]"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-background/90 via-background/20 to-background/10" />
            <div className="absolute inset-x-0 bottom-0 flex items-center justify-between gap-3 p-4">
              <div className="min-w-0">
                <div className="line-clamp-1 text-sm font-medium text-white">
                  {trailer.name}
                </div>
                <div className="mt-1 text-xs text-white/75">
                  点击后在当前详情页内播放
                </div>
              </div>
              <div className="rounded-full bg-white/90 px-3 py-1 text-xs font-medium text-black">
                观看
              </div>
            </div>
          </button>
        ) : null}

        <div className="flex flex-wrap items-center gap-3">
          <Button
            size="lg"
            className="rounded-full px-6"
            onClick={onOpenTrailer}
          >
            <Clapperboard className="size-4" />
            观看预告片
          </Button>
        </div>
      </CardContent>
    </Card>
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
            <div className="text-foreground/85">{value}</div>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}
