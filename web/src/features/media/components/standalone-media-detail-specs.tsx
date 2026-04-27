import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import {
  Captions,
  ChevronLeft,
  ChevronRight,
  FileVideo,
  HardDrive,
  Volume2,
} from 'lucide-react'
import { FreeMode } from 'swiper/modules'
import { Swiper, SwiperSlide } from 'swiper/react'
import type { Swiper as SwiperType } from 'swiper/types'

import { Button } from '#/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '#/components/ui/card'
import type {
  CatalogMediaStreamSummary,
  CatalogPersonDetail,
} from '#/lib/mibo-api'
import type { CatalogDetailPresentation } from '#/lib/media-presentation'
import {
  formatProviderLabel,
  getExternalIdentityUrl,
} from '#/lib/media-presentation'

import {
  formatAudioBitDepth,
  formatAudioLayout,
  formatBitRate,
  formatBitDepth,
  formatBooleanFlag,
  formatChannelsCompact,
  formatCodecLabel,
  formatCodecLevel,
  formatAssetLabel,
  formatFileSize,
  formatFrameRate,
  formatInterlaceState,
  formatProbeStatus,
  formatSampleRate,
  formatStreamLanguage,
  formatTechnicalValue,
  findAssetFileName,
  getPrimaryCatalogAsset,
  simplifyAspectRatio,
} from './standalone-media-detail-utils'

export function PeopleSection({ item }: { item: CatalogDetailPresentation }) {
  const sections = [
    { title: item.type === 'episode' ? '本集演员' : '演员', people: item.cast },
    {
      title: item.type === 'episode' ? '本集导演' : '导演',
      people: item.directors,
    },
  ].filter((section) => section.people.length > 0)

  if (sections.length === 0) {
    return null
  }

  return (
    <section className="mt-12 space-y-6">
      <div className="space-y-2">
        <h2 className="text-[19px] font-semibold text-foreground">演职人员</h2>
        <p className="text-sm text-muted-foreground">
          {item.type === 'episode'
            ? '优先展示本集演员和导演；暂无本集人员时不会显示占位卡片。'
            : '横向滑动浏览主要演员和导演信息。'}
        </p>
      </div>
      <div className="space-y-8">
        {sections.map((section) => (
          <PeopleRail
            key={section.title}
            title={section.title}
            people={section.people}
          />
        ))}
      </div>
    </section>
  )
}

function PeopleRail({
  title,
  people,
}: {
  title: string
  people: CatalogPersonDetail[]
}) {
  const [swiper, setSwiper] = useState<SwiperType | null>(null)
  const [canScrollPrev, setCanScrollPrev] = useState(false)
  const [canScrollNext, setCanScrollNext] = useState(false)

  const updateNavigation = (instance: SwiperType) => {
    setCanScrollPrev(!instance.isBeginning)
    setCanScrollNext(!instance.isEnd)
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-[24px] font-semibold tracking-tight text-foreground">
            {title}
          </h3>
          <div className="text-sm text-muted-foreground">
            共 {people.length} 人
          </div>
        </div>
        <div className="hidden items-center gap-2 sm:flex">
          <PeopleRailArrowButton
            direction="prev"
            disabled={!canScrollPrev}
            onClick={() => swiper?.slidePrev()}
          />
          <PeopleRailArrowButton
            direction="next"
            disabled={!canScrollNext}
            onClick={() => swiper?.slideNext()}
          />
        </div>
      </div>

      <div className="relative px-0 sm:px-12">
        <Swiper
          modules={[FreeMode]}
          freeMode
          slidesPerView="auto"
          spaceBetween={18}
          onSwiper={(instance) => {
            setSwiper(instance)
            updateNavigation(instance)
          }}
          onSlideChange={updateNavigation}
          onResize={updateNavigation}
          className="w-full"
        >
          {people.map((person, index) => (
            <SwiperSlide
              key={`${title}-${person.id ?? person.name}-${index}`}
              className="!h-auto !w-[150px] sm:!w-[176px] lg:!w-[196px]"
            >
              <PersonCard person={person} />
            </SwiperSlide>
          ))}
        </Swiper>

        <div className="mt-4 flex items-center justify-end gap-2 sm:hidden">
          <PeopleRailArrowButton
            direction="prev"
            disabled={!canScrollPrev}
            onClick={() => swiper?.slidePrev()}
          />
          <PeopleRailArrowButton
            direction="next"
            disabled={!canScrollNext}
            onClick={() => swiper?.slideNext()}
          />
        </div>
      </div>
    </div>
  )
}

function PersonCard({ person }: { person: CatalogPersonDetail }) {
  const cardContent = (
    <>
      <div className="relative aspect-[2/3] overflow-hidden bg-muted">
        {person.avatar_url ? (
          <img
            src={person.avatar_url}
            alt={person.name}
            className="h-full w-full object-cover transition duration-300 group-hover:scale-[1.04]"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center bg-gradient-to-br from-muted via-muted/70 to-background text-5xl font-semibold text-muted-foreground">
            {getPersonInitial(person.name)}
          </div>
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-background/95 via-background/10 to-transparent" />
      </div>
      <div className="space-y-1 p-3.5">
        <div className="line-clamp-2 min-h-12 text-base font-medium leading-6 text-foreground">
          {person.name}
        </div>
        <div className="line-clamp-2 min-h-10 text-sm leading-5 text-muted-foreground">
          {person.role || '未标注'}
        </div>
      </div>
    </>
  )

  const className =
    'group block h-full overflow-hidden rounded-[18px] border border-border/40 bg-card/75 shadow-lg backdrop-blur-md transition hover:border-border/70 hover:bg-card/85 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary'

  if (person.id) {
    return (
      <Link
        to="/person/$id"
        params={{ id: String(person.id) }}
        aria-label={`查看 ${person.name} 的人物详情`}
        className={className}
      >
        {cardContent}
      </Link>
    )
  }

  return (
    <div className={className} aria-label={person.name}>
      {cardContent}
    </div>
  )
}

function PeopleRailArrowButton({
  direction,
  disabled,
  onClick,
}: {
  direction: 'prev' | 'next'
  disabled: boolean
  onClick: () => void
}) {
  return (
    <Button
      type="button"
      size="icon-sm"
      variant="outline"
      className="rounded-full border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground"
      onClick={onClick}
      disabled={disabled}
    >
      {direction === 'prev' ? (
        <ChevronLeft className="size-4" />
      ) : (
        <ChevronRight className="size-4" />
      )}
      <span className="sr-only">
        {direction === 'prev' ? '上一组演职人员' : '下一组演职人员'}
      </span>
    </Button>
  )
}

export function SpecsSection({
  detailGroups,
  item,
}: {
  detailGroups: Array<{ title: string; value: string }>
  item: CatalogDetailPresentation
}) {
  const primaryAsset = getPrimaryCatalogAsset(item)
  const videoStreams = (primaryAsset?.streams ?? []).filter(
    (stream) => stream.stream_type === 'video',
  )
  const audioStreams = (primaryAsset?.streams ?? []).filter(
    (stream) => stream.stream_type === 'audio',
  )
  const subtitleStreams = (primaryAsset?.streams ?? []).filter(
    (stream) => stream.stream_type === 'subtitle',
  )
  const videoRows: [string, string][] =
    videoStreams.length > 0
      ? videoStreams.flatMap(buildVideoStreamRows)
      : [['状态', '暂无视频流探测信息']]
  const audioRows: [string, string][] =
    audioStreams.length > 0
      ? audioStreams.flatMap(buildAudioStreamRows)
      : [['状态', '暂无音轨探测信息']]
  const subtitleRows: [string, string][] =
    subtitleStreams.length > 0
      ? subtitleStreams.flatMap((stream, index) =>
          buildSubtitleStreamRows(stream, index, primaryAsset?.files ?? []),
        )
      : [['字幕', '关闭 / 不可用']]
  const fileRows: [string, string][] =
    primaryAsset?.files && primaryAsset.files.length > 0
      ? primaryAsset.files.map((file, index) => [
          `${file.role || '文件'} ${index + 1}`,
          [
            file.container || '未知容器',
            formatFileSize(file.size_bytes),
            file.status,
            file.storage_path || null,
          ]
            .filter(Boolean)
            .join('\n'),
        ])
      : [
          ['资源', formatAssetLabel(primaryAsset)],
          [
            '探测状态',
            formatProbeStatus(primaryAsset?.probe_status ?? 'pending'),
          ],
        ]

  if (item.type === 'episode') {
    return (
      <section className="mt-12 space-y-8">
        <div className="space-y-2">
          <h2 className="text-[19px] font-semibold text-foreground">
            媒体信息
          </h2>
          <p className="text-sm text-muted-foreground">
            按视频、音轨、字幕和文件展示当前集的探测摘要。
          </p>
        </div>

        <TrackChoiceSummary
          audioStreams={audioStreams}
          subtitleStreams={subtitleStreams}
        />

        <div className="grid min-w-0 gap-6 md:grid-cols-2 xl:grid-cols-4">
          <InfoCard
            icon={<FileVideo className="size-4" />}
            title="视频"
            rows={videoRows}
          />
          <InfoCard
            icon={<Volume2 className="size-4" />}
            title="音轨"
            rows={audioRows}
          />
          <InfoCard
            icon={<Captions className="size-4" />}
            title="字幕"
            rows={subtitleRows}
          />
          <InfoCard
            icon={<HardDrive className="size-4" />}
            title="文件"
            rows={fileRows}
          />
        </div>
      </section>
    )
  }

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
          <ExternalIdentityLinks item={item} />
        </div>
      </div>
      <div className="min-w-0 space-y-6 self-start">
        <div className="space-y-2">
          <h2 className="text-[19px] font-semibold text-foreground">
            媒体信息
          </h2>
          <p className="text-sm text-muted-foreground">
            按视频、音轨、字幕和文件展示当前版本的探测摘要。
          </p>
        </div>

        <TrackChoiceSummary
          audioStreams={audioStreams}
          subtitleStreams={subtitleStreams}
        />

        <div className="grid min-w-0 gap-6 lg:grid-cols-2 xl:grid-cols-2">
          <InfoCard
            icon={<FileVideo className="size-4" />}
            title="视频"
            rows={videoRows}
          />
          <InfoCard
            icon={<Volume2 className="size-4" />}
            title="音轨"
            rows={audioRows}
          />
          <InfoCard
            icon={<Captions className="size-4" />}
            title="字幕"
            rows={subtitleRows}
          />
          <InfoCard
            icon={<HardDrive className="size-4" />}
            title="文件"
            rows={fileRows}
          />
        </div>
      </div>
    </section>
  )
}

function buildVideoStreamRows(
  stream: CatalogMediaStreamSummary,
  index: number,
): [string, string][] {
  const resolution =
    stream.width && stream.height ? `${stream.width}x${stream.height}` : ''
  const aspectRatio =
    stream.width && stream.height
      ? simplifyAspectRatio(stream.width, stream.height)
      : ''
  const rows: Array<[string, string]> = [
    [`视频 ${index + 1}`, stream.title || stream.codec || '视频流'],
    ['标题', formatTechnicalValue(stream.title)],
    ['编码', stream.codec || '未知编码'],
    ['档案', formatTechnicalValue(stream.profile)],
    ['级别', formatCodecLevel(stream.level, stream.codec)],
    ['分辨率', resolution],
    ['宽高比', aspectRatio],
    ['隔行扫描', formatInterlaceState(stream.field_order)],
    ['帧率', formatFrameRate(stream.avg_frame_rate, stream.r_frame_rate)],
    [
      '码率',
      stream.bit_rate && stream.bit_rate > 0
        ? formatBitRate(stream.bit_rate)
        : '',
    ],
    ['色彩空间', formatTechnicalValue(stream.color_space)],
    ['位深', formatBitDepth(stream.bit_depth)],
    ['像素格式', formatTechnicalValue(stream.pixel_format)],
    [
      '参考帧',
      stream.reference_frames && stream.reference_frames > 0
        ? String(stream.reference_frames)
        : '',
    ],
  ]

  return rows.filter(([, value]) => value.trim() !== '')
}

function buildAudioStreamRows(
  stream: CatalogMediaStreamSummary,
  index: number,
): [string, string][] {
  const language = formatStreamLanguage(stream.language)
  const codec = formatCodecLabel(stream.codec)
  const layout = formatAudioLayout(stream.channel_layout, stream.channels)
  const title = [language, codec, layout].filter(Boolean).join(' ')

  const rows: Array<[string, string]> = [
    [
      `音轨 ${index + 1}`,
      title ? `${title}${stream.default ? ' (默认)' : ''}` : '音轨',
    ],
    ['语言', language],
    ['编解码器', codec],
    ['布局', layout],
    ['频道', formatChannelsCompact({ channels: stream.channels })],
    ['采样率', formatSampleRate(stream.sample_rate)],
    ['位深度', formatAudioBitDepth(stream.bit_depth)],
    ['默认', stream.default ? '是' : '否'],
  ]

  return rows.filter(([, value]) => value.trim() !== '')
}

function buildSubtitleStreamRows(
  stream: CatalogMediaStreamSummary,
  index: number,
  files: NonNullable<CatalogDetailPresentation['assets']>[number]['files'],
): [string, string][] {
  const language = formatStreamLanguage(stream.language)
  const codec = formatCodecLabel(stream.codec)
  const title =
    formatTechnicalValue(stream.title) || language || codec || '字幕流'
  const summary = [
    title,
    codec ? `(${stream.default ? '默认 ' : ''}${codec})` : '',
  ]
    .filter(Boolean)
    .join(' ')

  const rows: Array<[string, string]> = [
    [`字幕 ${index + 1}`, summary],
    ['标题', title],
    ['语言', language],
    ['编解码器', codec],
    ['默认', formatBooleanFlag(stream.default)],
    ['强制', formatBooleanFlag(stream.forced)],
    ['听力障碍', formatBooleanFlag(stream.hearing_impaired)],
    ['外部', formatBooleanFlag(stream.external)],
    ['文件', findAssetFileName(files, stream.file_id)],
  ]

  return rows.filter(([, value]) => value.trim() !== '')
}

function TrackChoiceSummary({
  audioStreams,
  subtitleStreams,
}: {
  audioStreams: NonNullable<
    CatalogDetailPresentation['assets']
  >[number]['streams']
  subtitleStreams: NonNullable<
    CatalogDetailPresentation['assets']
  >[number]['streams']
}) {
  const defaultAudio =
    audioStreams?.find((stream) => stream.default) ?? audioStreams?.[0]
  const defaultSubtitle = subtitleStreams?.find((stream) => stream.default)

  return (
    <div className="flex flex-wrap gap-2">
      <Button
        variant="outline"
        className="rounded-full border-border/50 bg-background/75"
        disabled
      >
        音轨：
        {defaultAudio
          ? [defaultAudio.language, defaultAudio.title || defaultAudio.codec]
              .filter(Boolean)
              .join(' · ') || '默认'
          : '不可用'}
      </Button>
      <Button
        variant="outline"
        className="rounded-full border-border/50 bg-background/75"
        disabled
      >
        字幕：
        {defaultSubtitle
          ? [
              defaultSubtitle.language,
              defaultSubtitle.title || defaultSubtitle.codec,
            ]
              .filter(Boolean)
              .join(' · ') || '默认'
          : '关闭 / 不可用'}
      </Button>
    </div>
  )
}

function ExternalIdentityLinks({ item }: { item: CatalogDetailPresentation }) {
  if (item.external_identities.length === 0) {
    return (
      <div className="space-y-2">
        <div className="text-base font-medium text-muted-foreground">
          数据库链接
        </div>
        <div className="text-[17px] leading-8 text-muted-foreground/80">
          暂未关联
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      <div className="text-base font-medium text-muted-foreground">
        数据库链接
      </div>
      <div className="flex flex-wrap gap-2">
        {item.external_identities.map((identity) => {
          const href = getExternalIdentityUrl(identity)
          const label = `${formatProviderLabel(identity.provider)} ${identity.external_id}`
          if (!href) {
            return (
              <span
                key={`${identity.provider}-${identity.provider_type}-${identity.external_id}`}
                className="rounded-full border border-border/50 bg-background/75 px-3 py-1 text-sm text-muted-foreground"
              >
                {label}
              </span>
            )
          }
          return (
            <a
              key={`${identity.provider}-${identity.provider_type}-${identity.external_id}`}
              href={href}
              target="_blank"
              rel="noreferrer"
              className="rounded-full border border-border/50 bg-background/75 px-3 py-1 text-sm text-foreground underline-offset-4 transition hover:bg-accent hover:text-accent-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            >
              {label}
            </a>
          )
        })}
      </div>
    </div>
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
        {rows.map(([label, value], index) => (
          <div
            key={`${label}-${index}`}
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

function getPersonInitial(name: string) {
  const trimmed = name.trim()
  return trimmed ? trimmed.slice(0, 1).toUpperCase() : '?'
}
