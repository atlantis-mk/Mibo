"use client"

import { useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'

import { HomePage } from '@/features/app/pages/home-page'
import { LibraryPage } from '@/features/app/pages/library-page'
import { MediaItemPage } from '@/features/app/pages/media-item-page'
import { MoviesPage } from '@/features/app/pages/movies-page'
import { SettingsPage } from '@/features/app/pages/settings-page'
import { ShowsPage } from '@/features/app/pages/shows-page'
import { getStoredApiBaseUrl, TOKEN_STORAGE_KEY } from '@/lib/client-config'
import { createMiboApi, type BrowseFilters } from '@/lib/mibo-api'
import type { MediaItemSearch } from '~/lib/app-route-search'

function AppShellLoading(props: { label: string }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background text-foreground">
      <div className="flex items-center gap-3 rounded-2xl border border-border/70 bg-card px-5 py-4 shadow-sm">
        <Loader2 className="size-4 animate-spin text-primary" />
        <span className="text-sm text-muted-foreground">{props.label}</span>
      </div>
    </div>
  )
}

function useLegacyAppMounted() {
  const [isMounted, setIsMounted] = useState(false)

  useEffect(() => {
    setIsMounted(true)
  }, [])

  return isMounted
}

export function LegacyHomeRoute(props: { browseFilters: BrowseFilters }) {
  const isMounted = useLegacyAppMounted()

  if (!isMounted) {
    return <AppShellLoading label="正在加载应用..." />
  }

  return <HomePage browseFilters={props.browseFilters} />
}

export function LegacyMoviesRoute(props: { browseFilters: BrowseFilters }) {
  const isMounted = useLegacyAppMounted()

  if (!isMounted) {
    return <AppShellLoading label="正在加载电影页..." />
  }

  return <MoviesPage browseFilters={props.browseFilters} />
}

export function LegacyShowsRoute(props: { browseFilters: BrowseFilters }) {
  const isMounted = useLegacyAppMounted()

  if (!isMounted) {
    return <AppShellLoading label="正在加载剧集页..." />
  }

  return <ShowsPage browseFilters={props.browseFilters} />
}

export function LegacyLibraryRoute(props: {
  browseFilters: BrowseFilters
  libraryId: number
}) {
  const isMounted = useLegacyAppMounted()

  if (!isMounted) {
    return <AppShellLoading label="正在加载媒体库..." />
  }

  return <LibraryPage browseFilters={props.browseFilters} libraryId={props.libraryId} />
}

export function LegacySettingsRoute() {
  const isMounted = useLegacyAppMounted()

  if (!isMounted) {
    return <AppShellLoading label="正在加载设置页..." />
  }

  return <SettingsPage />
}

export function LegacyMediaItemRoute(props: {
  mediaItemId: number
  search: MediaItemSearch
}) {
  const isMounted = useLegacyAppMounted()
  const [libraryId, setLibraryId] = useState<number | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)

  useEffect(() => {
    if (!isMounted) {
      return
    }

    let cancelled = false

    const loadMediaItem = async () => {
      try {
        setLoadError(null)

        const item = await createMiboApi({
          baseUrl: getStoredApiBaseUrl(),
          token: localStorage.getItem(TOKEN_STORAGE_KEY),
        }).getMediaItem(props.mediaItemId)

        if (!cancelled) {
          setLibraryId(item.library_id)
        }
      } catch (error) {
        if (!cancelled) {
          setLoadError(error instanceof Error ? error.message : '无法加载媒体详情')
        }
      }
    }

    void loadMediaItem()

    return () => {
      cancelled = true
    }
  }, [isMounted, props.mediaItemId])

  if (!isMounted) {
    return <AppShellLoading label="正在加载媒体详情..." />
  }

  if (loadError) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background text-foreground">
        <div className="rounded-2xl border border-border/70 bg-card px-5 py-4 text-sm text-muted-foreground shadow-sm">
          {loadError}
        </div>
      </div>
    )
  }

  if (libraryId === null) {
    return <AppShellLoading label="正在加载媒体详情..." />
  }

  return (
    <MediaItemPage
      browseFilters={props.search}
      libraryId={libraryId}
      mediaItemId={props.mediaItemId}
      originLibraryId={props.search.libraryId}
      originSection={props.search.from}
    />
  )
}
