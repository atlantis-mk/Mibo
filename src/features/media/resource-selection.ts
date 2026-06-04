export function resolveActiveResourceMetadataItemId({
  itemType,
  itemId,
  selectedEpisodeMetadataItemId,
  seriesPlaybackTargetEpisodeId,
}: {
  itemType?: string
  itemId: number
  selectedEpisodeMetadataItemId?: number
  seriesPlaybackTargetEpisodeId?: number
}) {
  if (itemType === 'series') {
    return selectedEpisodeMetadataItemId ?? seriesPlaybackTargetEpisodeId ?? 0
  }
  return itemId
}
