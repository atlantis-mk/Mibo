import { lazy, Suspense, useState } from 'react'
import { useAuthStore } from '@/stores/auth-store'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { SettingsPageInset } from './components/settings-page-inset'
import { SettingsPageShell } from './components/settings-page-shell'
import { SETTINGS_SECTIONS } from './sections'

const ConsolePage = lazy(() => import('@/features/console'))
const OperationsCenter = lazy(() => import('@/features/operations'))
const OperationsManagePage = lazy(() => import('@/features/operations/manage'))
const SecuritySettingsPanel = lazy(() =>
  import('./components/preference-panels').then((module) => ({
    default: module.SecuritySettingsPanel,
  }))
)
const UserManagementPanel = lazy(() =>
  import('./components/user-management-panel').then((module) => ({
    default: module.UserManagementPanel,
  }))
)
const RoleManagementPanel = lazy(() =>
  import('./components/role-management-panel').then((module) => ({
    default: module.RoleManagementPanel,
  }))
)
const DeviceManagementPanel = lazy(() =>
  import('./components/device-management-panel').then((module) => ({
    default: module.DeviceManagementPanel,
  }))
)
const DlnaManagementPanel = lazy(() =>
  import('./components/dlna-management-panel').then((module) => ({
    default: module.DlnaManagementPanel,
  }))
)
const LibraryManagementPanel = lazy(() =>
  import('./components/library-management-panel').then((module) => ({
    default: module.LibraryManagementPanel,
  }))
)
const ScanExclusionsPanel = lazy(() =>
  import('./components/scan-exclusions-panel').then((module) => ({
    default: module.ScanExclusionsPanel,
  }))
)
const TranscodingSettingsPanel = lazy(() =>
  import('./components/transcoding-settings-panel').then((module) => ({
    default: module.TranscodingSettingsPanel,
  }))
)
const NetworkSettingsPanel = lazy(() =>
  import('./components/network-settings-panel').then((module) => ({
    default: module.NetworkSettingsPanel,
  }))
)
const GeneralConfigPanel = lazy(() =>
  import('./components/general-config-panel').then((module) => ({
    default: module.GeneralConfigPanel,
  }))
)
const LiveTvSettingsPanel = lazy(() =>
  import('./components/live-tv-settings-panel').then((module) => ({
    default: module.LiveTvSettingsPanel,
  }))
)
const MetadataProviderSettingsPanel = lazy(() =>
  import('./components/metadata-provider-settings-panel').then((module) => ({
    default: module.MetadataProviderSettingsPanel,
  }))
)
const PluginManagementCenter = lazy(() =>
  import('@/features/plugin-management').then((module) => ({
    default: module.PluginManagementCenter,
  }))
)
const SubtitleSettingsPanel = lazy(() =>
  import('./components/subtitle-settings-panel').then((module) => ({
    default: module.SubtitleSettingsPanel,
  }))
)

export function SettingsOperationsPage() {
  return (
    <SettingsPageInset>
      <LazyPanel>
        <OperationsCenter />
      </LazyPanel>
    </SettingsPageInset>
  )
}

export function SettingsOperationsManagePage() {
  return (
    <SettingsPageInset fixedContent>
      <LazyPanel>
        <OperationsManagePage />
      </LazyPanel>
    </SettingsPageInset>
  )
}

export function SettingsConsolePage() {
  return (
    <SettingsNamedPage sectionKey='console' showHeader={false}>
      <LazyPanel>
        <ConsolePage embedded scrollable={false} />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsUsersPage() {
  return (
    <SettingsPageInset>
      <LazyPanel>
        <UserManagementPanel />
      </LazyPanel>
    </SettingsPageInset>
  )
}

export function SettingsRolesPage() {
  return (
    <SettingsPageInset>
      <LazyPanel>
        <RoleManagementPanel />
      </LazyPanel>
    </SettingsPageInset>
  )
}

export function SettingsDevicesPage() {
  return (
    <SettingsPageInset>
      <LazyPanel>
        <DeviceManagementPanel />
      </LazyPanel>
    </SettingsPageInset>
  )
}

export function SettingsDlnaPage() {
  return (
    <SettingsNamedPage sectionKey='dlna' showHeader={false}>
      <LazyPanel>
        <DlnaManagementPanel />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsLibraryPage() {
  const token = useAuthStore((state) => state.auth.accessToken)
  const [activeLibraryTab, setActiveLibraryTab] = useState<
    'sources' | 'libraries'
  >('libraries')

  return (
    <SettingsPageInset>
      <LazyPanel>
        <LibraryManagementPanel
          token={token}
          activeTab={activeLibraryTab}
          onActiveTabChange={setActiveLibraryTab}
        />
      </LazyPanel>
    </SettingsPageInset>
  )
}

export function SettingsScanExclusionsPage() {
  const token = useAuthStore((state) => state.auth.accessToken)
  const [activeScanTab, setActiveScanTab] = useState<
    'rules' | 'exclusions' | 'browser'
  >('rules')

  return (
    <SettingsNamedPage
      sectionKey='scan-exclusions'
      fixedContent
      showHeader={false}
    >
      <div className='mb-4 flex flex-wrap justify-end'>
        <SegmentedControl
          value={activeScanTab}
          options={[
            { value: 'rules', label: '自动规则' },
            { value: 'exclusions', label: '排除项' },
            { value: 'browser', label: '媒体浏览' },
          ]}
          onChange={setActiveScanTab}
        />
      </div>
      <LazyPanel>
        <ScanExclusionsPanel token={token} activeTab={activeScanTab} />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsPlaybackPage() {
  const [activeTab, setActiveTab] = useState<'transcoding' | 'tone-mapping'>(
    'transcoding'
  )

  return (
    <SettingsSectionPanel
      sectionKey='playback'
      showHeader={false}
      panel={
        <>
          <div className='mb-4 flex justify-center'>
            <div
              role='tablist'
              aria-orientation='horizontal'
              className='inline-flex h-9 items-center justify-center rounded-lg bg-muted px-1.5 py-0.75 text-muted-foreground'
            >
              <button
                type='button'
                role='tab'
                aria-selected={activeTab === 'transcoding'}
                className={cn(
                  'inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-md border border-transparent px-2 py-1 text-sm font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-1 focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-50',
                  activeTab === 'transcoding'
                    ? 'bg-primary text-primary-foreground shadow-sm'
                    : 'text-muted-foreground'
                )}
                onClick={() => setActiveTab('transcoding')}
              >
                转码
              </button>
              <button
                type='button'
                role='tab'
                aria-selected={activeTab === 'tone-mapping'}
                className={cn(
                  'inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-md border border-transparent px-2 py-1 text-sm font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-1 focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-50',
                  activeTab === 'tone-mapping'
                    ? 'bg-primary text-primary-foreground shadow-sm'
                    : 'text-muted-foreground'
                )}
                onClick={() => setActiveTab('tone-mapping')}
              >
                色调映射
              </button>
            </div>
          </div>
          <LazyPanel>
            <TranscodingSettingsPanel activeTab={activeTab} />
          </LazyPanel>
        </>
      }
    />
  )
}

export function SettingsNetworkPage() {
  const token = useAuthStore((state) => state.auth.accessToken)
  return (
    <SettingsSectionPanel
      sectionKey='network'
      showHeader={false}
      panel={
        <LazyPanel>
          <NetworkSettingsPanel token={token} />
        </LazyPanel>
      }
    />
  )
}

export function SettingsGeneralPage() {
  const token = useAuthStore((state) => state.auth.accessToken)
  return (
    <SettingsSectionPanel
      sectionKey='general'
      showHeader={false}
      panel={
        <LazyPanel>
          <GeneralConfigPanel token={token} />
        </LazyPanel>
      }
    />
  )
}

export function SettingsLiveTvPage() {
  const token = useAuthStore((state) => state.auth.accessToken)
  return (
    <SettingsSectionPanel
      sectionKey='live-tv'
      showHeader={false}
      panel={
        <LazyPanel>
          <LiveTvSettingsPanel token={token} />
        </LazyPanel>
      }
    />
  )
}

export function SettingsSecurityPage() {
  return (
    <SettingsSectionPanel
      sectionKey='security'
      showHeader={false}
      panel={
        <LazyPanel>
          <SecuritySettingsPanel />
        </LazyPanel>
      }
    />
  )
}

export function SettingsMetadataSourcesPage() {
  const token = useAuthStore((state) => state.auth.accessToken)

  return (
    <SettingsNamedPage sectionKey='metadata-sources' showHeader={false}>
      <LazyPanel>
        <MetadataProviderSettingsPanel token={token} />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsPluginsPage() {
  const token = useAuthStore((state) => state.auth.accessToken)

  return (
    <SettingsNamedPage sectionKey='plugins' fixedContent showHeader={false}>
      <LazyPanel>
        <PluginManagementCenter token={token} />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsSubtitlesPage() {
  const token = useAuthStore((state) => state.auth.accessToken)

  return (
    <SettingsNamedPage sectionKey='subtitles' fixedContent showHeader={false}>
      <LazyPanel>
        <SubtitleSettingsPanel token={token} />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

function SettingsNamedPage({
  sectionKey,
  actions,
  fixedContent,
  showHeader,
  children,
}: {
  sectionKey: string
  actions?: React.ReactNode
  fixedContent?: boolean
  showHeader?: boolean
  children: React.ReactNode
}) {
  const section = SETTINGS_SECTIONS.find(({ key }) => key === sectionKey)
  if (!section) return null

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
      actions={actions}
      fixedContent={fixedContent}
      showHeader={showHeader}
    >
      {children}
    </SettingsPageShell>
  )
}

function SettingsSectionPanel({
  sectionKey,
  actions,
  showHeader,
  panel,
}: {
  sectionKey:
    | 'general'
    | 'network'
    | 'playback'
    | 'notifications'
    | 'security'
    | 'live-tv'
  actions?: React.ReactNode
  showHeader?: boolean
  panel: React.ReactNode
}) {
  const section = SETTINGS_SECTIONS.find(({ key }) => key === sectionKey)
  if (!section) return null

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
      actions={actions}
      showHeader={showHeader}
    >
      {panel}
    </SettingsPageShell>
  )
}

function LazyPanel({ children }: { children: React.ReactNode }) {
  return (
    <Suspense
      fallback={
        <div className='rounded-[1.25rem] border border-border/60 bg-card/80 px-4 py-6 text-sm text-muted-foreground shadow-sm'>
          正在加载设置面板
        </div>
      }
    >
      {children}
    </Suspense>
  )
}

function SegmentedControl<Value extends string>({
  value,
  options,
  onChange,
}: {
  value: Value
  options: Array<{ value: Value; label: string }>
  onChange: (value: Value) => void
}) {
  return (
    <div className='inline-flex rounded-lg border border-border/60 bg-muted/30 p-1'>
      {options.map((option) => (
        <Button
          key={option.value}
          type='button'
          onClick={() => onChange(option.value)}
          variant={value === option.value ? 'outline' : 'ghost'}
          size='sm'
        >
          {option.label}
        </Button>
      ))}
    </div>
  )
}
