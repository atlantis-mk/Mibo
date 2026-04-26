import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { KeyRoundIcon } from 'lucide-react'

import { Button } from '#/components/ui/button'
import { Tabs, TabsList, TabsTrigger } from '#/components/ui/tabs'
import { useAuthStore } from '#/stores/auth-store'

import { LibraryManagementPanel } from './components/library-management-panel'
import { MetadataProviderSettingsPanel } from './components/metadata-provider-settings-panel'
import {
  NotificationSettingsPanel,
  PlaybackSettingsPanel,
  SecuritySettingsPanel,
} from './components/preference-panels'
import { SettingsPageShell } from './components/settings-page-shell'
import { SETTINGS_SECTIONS } from './sections'

export function SettingsLibraryPage() {
  const token = useAuthStore((state) => state.token)
  const [activeLibraryTab, setActiveLibraryTab] = useState<
    'sources' | 'libraries'
  >('sources')
  const section = SETTINGS_SECTIONS.find(({ key }) => key === 'library')

  if (!section) {
    return null
  }

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
      actions={
        <Tabs
          value={activeLibraryTab}
          onValueChange={(value) =>
            setActiveLibraryTab(value as 'sources' | 'libraries')
          }
        >
          <TabsList>
            <TabsTrigger value="sources">媒体源</TabsTrigger>
            <TabsTrigger value="libraries">媒体库</TabsTrigger>
          </TabsList>
        </Tabs>
      }
    >
      <LibraryManagementPanel token={token} activeTab={activeLibraryTab} />
    </SettingsPageShell>
  )
}

export function SettingsPlaybackPage() {
  return (
    <SettingsSectionPanel
      sectionKey="playback"
      panel={<PlaybackSettingsPanel />}
    />
  )
}

export function SettingsNotificationsPage() {
  return (
    <SettingsSectionPanel
      sectionKey="notifications"
      panel={<NotificationSettingsPanel />}
    />
  )
}

export function SettingsSecurityPage() {
  return (
    <SettingsSectionPanel
      sectionKey="security"
      panel={<SecuritySettingsPanel />}
    />
  )
}

export function SettingsMetadataSourcesPage() {
  const token = useAuthStore((state) => state.token)
  const section = SETTINGS_SECTIONS.find(
    ({ key }) => key === 'metadata-sources',
  )

  if (!section) {
    return null
  }

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
      actions={
        <Button asChild variant="outline">
          <Link to="/settings/metadata">
            <KeyRoundIcon className="size-4" />
            打开治理工作台
          </Link>
        </Button>
      }
    >
      <MetadataProviderSettingsPanel token={token} />
    </SettingsPageShell>
  )
}

function SettingsSectionPanel({
  sectionKey,
  panel,
}: {
  sectionKey: 'playback' | 'notifications' | 'security'
  panel: React.ReactNode
}) {
  const section = SETTINGS_SECTIONS.find(({ key }) => key === sectionKey)

  if (!section) {
    return null
  }

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
    >
      {panel}
    </SettingsPageShell>
  )
}
