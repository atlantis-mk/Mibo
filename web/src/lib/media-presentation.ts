import type {
  CatalogEpisodeParentContext,
  CatalogEpisodeShelfItem,
  CatalogItemDetail,
  CatalogListItem,
  CatalogPersonDetail,
  CatalogSeasonDetail,
  CatalogSourceEvidence,
  CatalogExternalIdentity,
  CatalogTagDetail,
  MediaResourceDetail,
} from "#/lib/mibo-api"

export type MediaDetailView = "episode" | "series"
export type MediaCardItem = CatalogListItem
export type CatalogSeasonRail = {
  season_number: number
  name: string
  overview: string
  poster_url: string
  runtime_seconds?: number
  episodes: CatalogEpisodeRail[]
}

export type CatalogEpisodeRail = {
	metadata_item_id: number
	inventory_file_id?: number
	season_number: number
	episode_number: number
  name: string
  overview: string
  still_url: string
  air_date?: string
  runtime_seconds?: number
  availability_status: string
  progress_percent?: number
  watched?: boolean
  current?: boolean
}

export type MediaDetailPresentation = {
  id: number
  type: string
  title: string
  original_title: string
  overview: string
  year?: number
  end_year?: number
  runtime_seconds?: number
  community_rating?: number
  official_rating: string
  series_status: string
  release_date?: string
  first_air_date?: string
  last_air_date?: string
  poster_url: string
  logo_url?: string
  backdrop_url: string
  metadata_provider: string
  external_id: string
  metadata_confidence?: number
  external_identities: CatalogExternalIdentity[]
  tags: CatalogTagDetail[]
  genres: string[]
  child_summary?: CatalogItemDetail["child_summary"]
  related_items: CatalogListItem[]
  availability_status: string
  governance_status: string
  series_title_display: string
  episode_label: string
  episode_context?: CatalogEpisodeParentContext
  series_playback_target?: CatalogItemDetail["series_playback_target"]
  primary_visual_url: string
  default_season_number?: number
  same_season_episodes: CatalogEpisodeRail[]
  source_evidence: CatalogSourceEvidence[]
  cast: CatalogPersonDetail[]
  directors: CatalogPersonDetail[]
  resources: MediaResourceDetail[]
}

const seasonFolderPattern =
  /^(?:season|s)\s*0*\d+$|^第\s*[0-9一二三四五六七八九十两零]+\s*季$/i

export function parseMediaDetailView(value: unknown): MediaDetailView {
  return value === "series" ? "series" : "episode"
}

export function metadataItemDetailToPresentation(
  item: CatalogItemDetail
): MediaDetailPresentation {
  const primaryIdentity = item.external_identities?.[0]
  const seasons = item.seasons ?? []
  const episodeContext = item.episode_context
  const episodeStill =
    selectedCatalogImageUrl(item.selected_images, "still") ||
    selectedCatalogImageUrl(item.selected_images, "backdrop")
  const seriesBackdrop = selectedCatalogImageUrl(
    episodeContext?.series?.selected_images,
    "backdrop"
  )
  const seriesPoster = selectedCatalogImageUrl(
    episodeContext?.series?.selected_images,
    "poster"
  )
  const posterUrl = selectedCatalogImageUrl(item.selected_images, "poster")
  const backdropUrl = selectedCatalogImageUrl(item.selected_images, "backdrop")

  return {
    id: item.id,
    type: item.type,
    title: item.title,
    original_title: item.original_title ?? "",
    overview: item.overview ?? "",
    year: item.year,
    end_year: item.end_year,
    runtime_seconds: item.runtime_seconds,
    community_rating: item.community_rating,
    official_rating: item.official_rating ?? "",
    series_status: item.series_status ?? "",
    release_date: item.release_date,
    first_air_date: item.first_air_date,
    last_air_date: item.last_air_date,
    poster_url:
      item.type === "episode" ? episodeStill || seriesPoster : posterUrl,
    logo_url:
      selectedCatalogImageUrl(item.selected_images, "logo") || undefined,
    backdrop_url:
      item.type === "episode" ? episodeStill || seriesBackdrop : backdropUrl,
    metadata_provider: primaryIdentity?.provider ?? "",
    external_id: primaryIdentity?.external_id ?? "",
    metadata_confidence: primaryIdentity?.confidence,
    external_identities: item.external_identities ?? [],
    tags: item.tags ?? [],
    genres: item.genres ?? [],
    child_summary: item.child_summary,
    related_items: item.related_items ?? [],
    availability_status: item.availability_status,
    governance_status: item.governance_status,
    series_title_display: episodeContext?.series?.title ?? item.title,
    episode_label: formatEpisodeLabel(
      episodeContext?.season_number,
      episodeContext?.episode_number,
      episodeContext?.episode_number_end
    ),
    episode_context: episodeContext,
    series_playback_target: item.series_playback_target,
    primary_visual_url:
      item.type === "episode"
        ? episodeStill || seriesBackdrop || seriesPoster
        : posterUrl,
    default_season_number: seasons[0]?.index_number,
    same_season_episodes: catalogEpisodeShelfToRails(
      item.same_season_episodes ?? []
    ),
    source_evidence: item.source_evidence ?? [],
    cast: item.cast ?? [],
    directors: item.directors ?? [],
    resources: item.resources ?? [],
  }
}

export function catalogEpisodeShelfToSeasonRails(
  item: MediaDetailPresentation
): CatalogSeasonRail[] {
  if (item.type !== "episode" || item.same_season_episodes.length === 0) {
    return []
  }
  const seasonNumber =
    item.episode_context?.season?.number ??
    item.episode_context?.season_number ??
    item.same_season_episodes[0]?.season_number ??
    0
  return [
    {
      season_number: seasonNumber,
      name: item.episode_context?.season?.title ?? `Season ${seasonNumber}`,
      overview: "",
      poster_url: selectedCatalogImageUrl(
        item.episode_context?.season?.selected_images,
        "poster"
      ),
      episodes: item.same_season_episodes,
    },
  ]
}

export function catalogSeasonsToRails(
  seasons: CatalogSeasonDetail[]
): CatalogSeasonRail[] {
  return seasons.map((season) => ({
    season_number: season.index_number ?? 0,
    name: season.title,
    overview: season.overview ?? "",
    poster_url: selectedCatalogImageUrl(season.selected_images, "poster"),
    runtime_seconds: season.runtime_seconds,
		episodes: (season.episodes ?? []).map((episode) => ({
			metadata_item_id: episode.id,
			inventory_file_id: episode.inventory_file_id,
			season_number: episode.parent_index_number ?? season.index_number ?? 0,
      episode_number: episode.index_number ?? 0,
      name: episode.title,
      overview: episode.overview ?? "",
      still_url:
        selectedCatalogImageUrl(episode.selected_images, "still") ||
        selectedCatalogImageUrl(episode.selected_images, "backdrop"),
      air_date: episode.release_date ?? episode.first_air_date,
      runtime_seconds: episode.runtime_seconds,
      availability_status: episode.availability_status,
      progress_percent: undefined,
      watched: undefined,
    })),
  }))
}

export function formatMediaRating(value?: number) {
  if (typeof value !== "number" || !Number.isFinite(value) || value <= 0) {
    return ""
  }
  return value.toFixed(value >= 10 ? 0 : 1)
}

export function formatMediaDetailYearRange(
  item: Pick<
    MediaDetailPresentation,
    | "type"
    | "year"
    | "end_year"
    | "release_date"
    | "first_air_date"
    | "last_air_date"
    | "series_status"
  >
) {
  const startYear =
    item.year ??
    yearFromDate(item.first_air_date) ??
    yearFromDate(item.release_date)
  const endYear = item.end_year ?? yearFromDate(item.last_air_date)

  if (!startYear) {
    return ""
  }
  if (item.type === "series" || item.type === "show") {
    if (endYear && endYear !== startYear) return `${startYear} - ${endYear}`
    if (isContinuingSeries(item.series_status)) return `${startYear} - 现在`
  }
  return String(startYear)
}

export function formatSeasonSummary(
  item: Pick<MediaDetailPresentation, "type" | "child_summary">
) {
  if (item.type !== "series" && item.type !== "show") return ""
  const summary = item.child_summary
  if (!summary?.child_count) return ""
  const episodeCount = Math.max(0, summary.child_count)
  const available = summary.available_count
  if (available > 0 && available !== episodeCount) {
    return `${episodeCount} 集 · ${available} 可播放`
  }
  return `${episodeCount} 集`
}

export function formatProviderLabel(provider?: string) {
  const normalized = provider?.trim().toLowerCase()
  switch (normalized) {
    case "tmdb":
      return "TMDB"
    case "imdb":
      return "IMDb"
    case "tvdb":
      return "TVDB"
    default:
      return provider?.trim() || ""
  }
}

export function getExternalIdentityUrl(
  identity: Pick<
    CatalogExternalIdentity,
    "provider" | "provider_type" | "external_id"
  >
) {
  const provider = identity.provider.trim().toLowerCase()
  const providerType = identity.provider_type.trim().toLowerCase()
  const externalID = identity.external_id.trim()
  const isPerson = providerType === "person" || providerType === "name"
  if (!externalID) return ""
  if (provider === "imdb") {
    return `https://www.imdb.com/${isPerson ? "name" : "title"}/${externalID}/`
  }
  if (provider === "tmdb") {
    if (isPerson) {
      const cleanID = externalID.replace(/^person:/i, "")
      return `https://www.themoviedb.org/person/${cleanID}`
    }
    const pathType =
      providerType === "tv" || providerType === "series" ? "tv" : "movie"
    const cleanID = externalID.replace(/^(movie|tv):/i, "")
    return `https://www.themoviedb.org/${pathType}/${cleanID}`
  }
  if (provider === "tvdb")
    return `https://thetvdb.com/dereferrer/series/${externalID}`
  return ""
}

export function formatMediaCardTitle(
  item: Pick<MediaCardItem, "type" | "title"> & {
    series_title?: string
    source_path?: string
  }
) {
  const title = getPrimarySeriesTitle(item)
  if (item.type === "show" || item.type === "series") {
    return title
  }

  const episodeTitle = item.title?.trim() ?? ""
  const explicitSeriesTitle = item.series_title?.trim() ?? ""
  if (
    explicitSeriesTitle &&
    title &&
    !sameMediaTitle(title, episodeTitle) &&
    !mediaTitleStartsWith(episodeTitle, title)
  ) {
    return `${title} · ${episodeTitle}`
  }

  return episodeTitle
}

export function buildPresentedMediaItem(
  item: MediaDetailPresentation,
  _seasons: CatalogSeasonRail[],
  view: MediaDetailView
) {
  if (item.type === "episode" || view === "episode") {
    const episodeTitle = item.title || item.original_title
    return {
      ...item,
      title: item.series_title_display || item.title,
      original_title: item.episode_label
        ? `${item.episode_label} - ${episodeTitle}`
        : episodeTitle,
      poster_url: item.primary_visual_url || item.poster_url,
      backdrop_url: item.backdrop_url || item.primary_visual_url,
    }
  }
  return item
}

export function getPrimarySeriesTitle(
  item: Pick<MediaCardItem, "type" | "title"> & {
    series_title?: string
    source_path?: string
  }
) {
  if (
    item.type === "show" ||
    item.type === "series" ||
    item.type === "episode"
  ) {
    const pathTitle = titleFromSourcePath(item.source_path ?? "")
    if (pathTitle) {
      return pathTitle
    }
  }

  const seriesTitle = item.series_title?.trim()
  if (seriesTitle) {
    return seriesTitle
  }

  return stripEpisodeSuffix(item.title?.trim() ?? "")
}

export function getMediaCardType(item: MediaCardItem) {
  return item.type === "series" ? "show" : item.type
}

export function getMediaCardPosterUrl(item: MediaCardItem) {
  return selectedCatalogImageUrl(item.selected_images, "poster")
}

export function getMediaCardBackdropUrl(item: MediaCardItem) {
  return selectedCatalogImageUrl(item.selected_images, "backdrop")
}

export function getMediaCardMetadataProvider(item: MediaCardItem) {
  return item.external_identities?.[0]?.provider ?? ""
}

export function getMediaCardMatchStatus(item: MediaCardItem) {
  return item.governance_status
}

export function getMediaCardAvailabilityStatus(item: MediaCardItem) {
  return item.availability_status
}

export function formatMediaCardYearRange(item: MediaCardItem) {
  if (item.organizing_summary?.message) {
    return item.organizing_summary.message
  }
  if (item.organizing) {
    if (item.maturity_state === "review_required") return "需要确认"
    return "整理中"
  }
  const startYear =
    item.year ??
    yearFromDate(item.first_air_date) ??
    yearFromDate(item.release_date)
  const endYear = yearFromDate(item.last_air_date)

  if (!startYear) {
    return "未知年份"
  }

  if (getMediaCardType(item) === "show") {
    if (endYear && endYear !== startYear) {
      return `${startYear} - ${endYear}`
    }
    if (isContinuingSeries(item.series_status)) {
      return `${startYear} - 现在`
    }
  }

  return String(startYear)
}

export function getMediaCardBadgeCount(item: MediaCardItem) {
  const summary = item.child_summary
  if (!summary) {
    return null
  }

  const unwatchedCount = Math.max(
    0,
    summary.available_count - summary.played_count
  )
  if (unwatchedCount > 0) return unwatchedCount
  if (summary.in_progress_count > 0) return summary.in_progress_count
  if (summary.available_count > 0) return summary.available_count
  if (summary.child_count > 0) return summary.child_count
  return null
}

export function isMediaCardPlayable(item: MediaCardItem) {
  return item.availability_status === "available"
}

export function getMediaCardOrganizingLabel(item: MediaCardItem) {
  switch (item.organizing_summary?.state) {
    case "failed":
      return "整理失败"
    case "review_required":
      return "待确认"
    case "partial_ready":
      return "部分就绪"
    case "ready":
      return "就绪"
    default:
      return item.maturity_state === "review_required" ? "待确认" : "整理中"
  }
}

export function blocksMediaCardCatalogActions(item: MediaCardItem) {
  return Boolean(item.source_kind === "inventory_file" || item.organizing)
}

function stripEpisodeSuffix(input: string) {
  const stripped = input.replace(
    /(?:[\s._-]+s\d{1,2}e\d{1,3})(?:[\s._-]+.*)?$/i,
    ""
  )
  return stripped.trim() || input.trim()
}

function sameMediaTitle(left: string, right: string) {
  return normalizeMediaTitle(left) === normalizeMediaTitle(right)
}

function mediaTitleStartsWith(title: string, prefix: string) {
  const normalizedTitle = normalizeMediaTitle(title)
  const normalizedPrefix = normalizeMediaTitle(prefix)
  return Boolean(
    normalizedPrefix &&
    (normalizedTitle === normalizedPrefix ||
      normalizedTitle.startsWith(`${normalizedPrefix} `))
  )
}

function normalizeMediaTitle(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[._:：·-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim()
}

function titleFromSourcePath(sourcePath: string) {
  const segments = sourcePath
    .split(/[\\/]+/)
    .map((segment) => segment.trim())
    .filter(Boolean)
  if (segments.length < 2) {
    return ""
  }

  let index = segments.length - 2
  for (let candidate = segments.length - 2; candidate >= 0; candidate -= 1) {
    if (seasonFolderPattern.test(segments[candidate])) {
      index = candidate > 0 ? candidate - 1 : index
      break
    }
  }

  return segments[index]?.trim() || ""
}

function yearFromDate(value?: string) {
  if (!value || value.length < 4) return null
  const year = Number.parseInt(value.slice(0, 4), 10)
  return Number.isFinite(year) ? year : null
}

function isContinuingSeries(status?: string) {
  const normalized = status?.trim().toLowerCase()
  return normalized === "continuing" || normalized === "returning series"
}

function selectedCatalogImageUrl(
  images: { image_type: string; url: string }[] | undefined,
  imageType: string
) {
  return (
    (images || []).find((image) => image.image_type === imageType)?.url ?? ""
  )
}

function catalogEpisodeShelfToRails(
  episodes: CatalogEpisodeShelfItem[]
): CatalogEpisodeRail[] {
	return episodes.map((episode) => ({
	  metadata_item_id: episode.id,
	  inventory_file_id: episode.inventory_file_id,
    season_number: episode.season_number ?? 0,
    episode_number: episode.episode_number ?? 0,
    name: episode.title,
    overview: episode.overview ?? "",
    still_url:
      selectedCatalogImageUrl(episode.selected_images, "still") ||
      selectedCatalogImageUrl(episode.selected_images, "backdrop"),
    air_date: episode.release_date ?? episode.first_air_date,
    runtime_seconds: episode.runtime_seconds,
    availability_status: episode.availability_status,
    progress_percent: episode.progress?.played_percentage,
    watched: episode.progress?.watched,
    current: episode.current,
  }))
}

function formatEpisodeLabel(
  seasonNumber?: number,
  episodeNumber?: number,
  episodeNumberEnd?: number
) {
  if (typeof seasonNumber !== "number" && typeof episodeNumber !== "number") {
    return ""
  }
  const season = typeof seasonNumber === "number" ? `S${seasonNumber}` : ""
  const episode =
    typeof episodeNumber === "number"
      ? `E${episodeNumber}${typeof episodeNumberEnd === "number" && episodeNumberEnd !== episodeNumber ? `-E${episodeNumberEnd}` : ""}`
      : ""
  return [season, episode].filter(Boolean).join(":")
}
