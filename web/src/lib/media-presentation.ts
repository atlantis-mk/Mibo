import type {
  CatalogAssetDetail,
  CatalogItemDetail,
  CatalogListItem,
  CatalogSeasonDetail,
  CatalogSourceEvidence,
  MediaItem,
} from '#/lib/mibo-api'

export type MediaDetailView = 'episode' | 'series'
export type MediaCardItem = CatalogListItem | MediaItem
export type CatalogSeasonRail = {
  season_number: number
  name: string
  overview: string
  poster_url: string
  runtime_seconds?: number
  episodes: CatalogEpisodeRail[]
}

export type CatalogEpisodeRail = {
  item_id: number
  season_number: number
  episode_number: number
  name: string
  overview: string
  still_url: string
  air_date?: string
  runtime_seconds?: number
  availability_status: string
}

export type CatalogDetailPresentation = {
  id: number
  type: string
  title: string
  original_title: string
  overview: string
  year?: number
  runtime_seconds?: number
  release_date?: string
  first_air_date?: string
  poster_url: string
  logo_url?: string
  backdrop_url: string
  metadata_provider: string
  external_id: string
  metadata_confidence?: number
  availability_status: string
  governance_status: string
  series_title_display: string
  default_season_number?: number
  source_evidence: CatalogSourceEvidence[]
  assets: CatalogAssetDetail[]
}

const seasonFolderPattern =
  /^(?:season|s)\s*0*\d+$|^第\s*[0-9一二三四五六七八九十两零]+\s*季$/i

export function parseMediaDetailView(value: unknown): MediaDetailView {
  return value === 'series' ? 'series' : 'episode'
}

export function catalogItemDetailToPresentation(
  item: CatalogItemDetail,
): CatalogDetailPresentation {
  const primaryIdentity = item.external_identities?.[0]
  const seasons = item.seasons ?? []

  return {
    id: item.id,
    type: item.type,
    title: item.title,
    original_title: item.original_title ?? '',
    overview: item.overview ?? '',
    year: item.year,
    runtime_seconds: item.runtime_seconds,
    release_date: item.release_date,
    first_air_date: item.first_air_date,
    poster_url: selectedCatalogImageUrl(item.selected_images, 'poster'),
    logo_url:
      selectedCatalogImageUrl(item.selected_images, 'logo') || undefined,
    backdrop_url: selectedCatalogImageUrl(item.selected_images, 'backdrop'),
    metadata_provider: primaryIdentity?.provider ?? '',
    external_id: primaryIdentity?.external_id ?? '',
    metadata_confidence: primaryIdentity?.confidence,
    availability_status: item.availability_status,
    governance_status: item.governance_status,
    series_title_display: item.title,
    default_season_number: seasons[0]?.index_number,
    source_evidence: item.source_evidence ?? [],
    assets: item.assets ?? [],
  }
}

export function catalogSeasonsToRails(
  seasons: CatalogSeasonDetail[],
): CatalogSeasonRail[] {
  return seasons.map((season) => ({
    season_number: season.index_number ?? 0,
    name: season.title,
    overview: season.overview ?? '',
    poster_url: selectedCatalogImageUrl(season.selected_images, 'poster'),
    runtime_seconds: season.runtime_seconds,
    episodes: (season.episodes ?? []).map((episode) => ({
      item_id: episode.id,
      season_number: episode.parent_index_number ?? season.index_number ?? 0,
      episode_number: episode.index_number ?? 0,
      name: episode.title,
      overview: episode.overview ?? '',
      still_url:
        selectedCatalogImageUrl(episode.selected_images, 'still') ||
        selectedCatalogImageUrl(episode.selected_images, 'backdrop'),
      air_date: episode.release_date ?? episode.first_air_date,
      runtime_seconds: episode.runtime_seconds,
      availability_status: episode.availability_status,
    })),
  }))
}

export function formatMediaCardTitle(
  item: Pick<MediaCardItem, 'type' | 'title'> & {
    series_title?: string
    source_path?: string
  },
) {
  const title = getPrimarySeriesTitle(item)
  if (item.type === 'show' || item.type === 'series') {
    return title
  }

  const episodeTitle = item.title?.trim() ?? ''
  if (item.series_title?.trim() && title && title !== episodeTitle) {
    return `${title} · ${episodeTitle}`
  }

  return episodeTitle
}

export function buildPresentedCatalogItem(
  item: CatalogDetailPresentation,
  _seasons: CatalogSeasonRail[],
  _view: MediaDetailView,
) {
  return item
}

export function getPrimarySeriesTitle(
  item: Pick<MediaCardItem, 'type' | 'title'> & {
    series_title?: string
    source_path?: string
  },
) {
  if (
    item.type === 'show' ||
    item.type === 'series' ||
    item.type === 'episode'
  ) {
    const pathTitle = titleFromSourcePath(item.source_path ?? '')
    if (pathTitle) {
      return pathTitle
    }
  }

  const seriesTitle = item.series_title?.trim()
  if (seriesTitle) {
    return seriesTitle
  }

  return stripEpisodeSuffix(item.title?.trim() ?? '')
}

export function getMediaCardType(item: MediaCardItem) {
  return item.type === 'series' ? 'show' : item.type
}

export function getMediaCardPosterUrl(item: MediaCardItem) {
  return isCatalogListItem(item)
    ? selectedCatalogImageUrl(item.selected_images, 'poster')
    : item.poster_url
}

export function getMediaCardBackdropUrl(item: MediaCardItem) {
  return isCatalogListItem(item)
    ? selectedCatalogImageUrl(item.selected_images, 'backdrop')
    : item.backdrop_url
}

export function getMediaCardMetadataProvider(item: MediaCardItem) {
  return isCatalogListItem(item)
    ? (item.external_identities?.[0]?.provider ?? '')
    : item.metadata_provider
}

export function getMediaCardMatchStatus(item: MediaCardItem) {
  return isCatalogListItem(item) ? item.governance_status : item.match_status
}

export function getMediaCardAvailabilityStatus(item: MediaCardItem) {
  return isCatalogListItem(item) ? item.availability_status : item.status
}

function stripEpisodeSuffix(input: string) {
  const stripped = input.replace(
    /(?:[\s._-]+s\d{1,2}e\d{1,3})(?:[\s._-]+.*)?$/i,
    '',
  )
  return stripped.trim() || input.trim()
}

function titleFromSourcePath(sourcePath: string) {
  const segments = sourcePath
    .split(/[\\/]+/)
    .map((segment) => segment.trim())
    .filter(Boolean)
  if (segments.length < 2) {
    return ''
  }

  let index = segments.length - 2
  for (let candidate = segments.length - 2; candidate >= 0; candidate -= 1) {
    if (seasonFolderPattern.test(segments[candidate])) {
      index = candidate > 0 ? candidate - 1 : index
      break
    }
  }

  return segments[index]?.trim() || ''
}

function selectedCatalogImageUrl(
  images: { image_type: string; url: string }[] | undefined,
  imageType: string,
) {
  return (
    (images || []).find((image) => image.image_type === imageType)?.url ?? ''
  )
}

function isCatalogListItem(item: MediaCardItem): item is CatalogListItem {
  return 'selected_images' in item
}
