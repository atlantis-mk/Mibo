import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { HelpCircleIcon, KeyRoundIcon } from 'lucide-react'

import { Button } from '#/components/ui/button'
import { Tabs, TabsList, TabsTrigger } from '#/components/ui/tabs'
import ConsolePage from '#/features/console'
import { useAuthStore } from '#/stores/auth-store'

import { DatabaseSettingsPanel } from './components/database-settings-panel'
import { DeviceManagementPanel } from './components/device-management-panel'
import { DlnaManagementPanel } from './components/dlna-management-panel'
import { LibraryManagementPanel } from './components/library-management-panel'
import { LiveTvSettingsPanel } from './components/live-tv-settings-panel'
import { MetadataProviderSettingsPanel } from './components/metadata-provider-settings-panel'
import { NetworkSettingsPanel } from './components/network-settings-panel'
import {
  GeneralSettingsPanel,
  NotificationSettingsPanel,
  SecuritySettingsPanel,
} from './components/preference-panels'
import { SettingsPageShell } from './components/settings-page-shell'
import { TranscodingSettingsPanel } from './components/transcoding-settings-panel'
import { UserManagementPanel } from './components/user-management-panel'
import { SETTINGS_SECTIONS } from './sections'

export function SettingsGeneralPage() {
  return (
    <SettingsSectionPanel
      sectionKey="general"
      panel={<GeneralSettingsPanel />}
    />
  )
}

export function SettingsConsolePage() {
  const section = SETTINGS_SECTIONS.find(({ key }) => key === 'console')

  if (!section) {
    return null
  }

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
    >
      <ConsolePage embedded />
    </SettingsPageShell>
  )
}

export function SettingsUsersPage() {
  const section = SETTINGS_SECTIONS.find(({ key }) => key === 'users')

  if (!section) {
    return null
  }

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
    >
      <UserManagementPanel />
    </SettingsPageShell>
  )
}

export function SettingsDevicesPage() {
  const section = SETTINGS_SECTIONS.find(({ key }) => key === 'devices')

  if (!section) {
    return null
  }

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
      actions={
        <Button variant="ghost" size="icon" className="rounded-full">
          <span className="sr-only">设备帮助</span>
          <HelpCircleIcon className="size-4" />
        </Button>
      }
    >
      <DeviceManagementPanel />
    </SettingsPageShell>
  )
}

export function SettingsDlnaPage() {
  const section = SETTINGS_SECTIONS.find(({ key }) => key === 'dlna')

  if (!section) {
    return null
  }

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
      actions={
        <Button variant="ghost" size="icon" className="rounded-full">
          <span className="sr-only">DLNA 帮助</span>
          <HelpCircleIcon className="size-4" />
        </Button>
      }
    >
      <DlnaManagementPanel />
    </SettingsPageShell>
  )
}

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
      panel={<TranscodingSettingsPanel />}
    />
  )
}

export function SettingsNetworkPage() {
  return (
    <SettingsSectionPanel
      sectionKey="network"
      panel={<NetworkSettingsPanel />}
    />
  )
}

export function SettingsDatabasePage() {
  return (
    <SettingsSectionPanel
      sectionKey="database"
      panel={<DatabaseSettingsPanel />}
    />
  )
}

export function SettingsLiveTvPage() {
  return (
    <SettingsSectionPanel
      sectionKey="live-tv"
      panel={<LiveTvSettingsPanel />}
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
  sectionKey:
    | 'general'
    | 'network'
    | 'database'
    | 'playback'
    | 'notifications'
    | 'security'
    | 'live-tv'
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
