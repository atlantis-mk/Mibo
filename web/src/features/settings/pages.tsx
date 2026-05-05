import { Link } from "@tanstack/react-router"
import { lazy, Suspense, useState } from "react"
import { HelpCircleIcon, KeyRoundIcon } from "lucide-react"

import { Button } from "#/components/ui/button"
import { useAuthStore } from "#/stores/auth-store"

import { SettingsPageShell } from "./components/settings-page-shell"
import { SETTINGS_SECTIONS } from "./sections"

const ConsolePage = lazy(() => import("#/features/console"))
const HealthCenter = lazy(() => import("#/features/health"))
const GeneralSettingsPanel = lazy(() =>
  import("./components/preference-panels").then((module) => ({
    default: module.GeneralSettingsPanel,
  }))
)
const NotificationSettingsPanel = lazy(() =>
  import("./components/preference-panels").then((module) => ({
    default: module.NotificationSettingsPanel,
  }))
)
const SecuritySettingsPanel = lazy(() =>
  import("./components/preference-panels").then((module) => ({
    default: module.SecuritySettingsPanel,
  }))
)
const UserManagementPanel = lazy(() =>
  import("./components/user-management-panel").then((module) => ({
    default: module.UserManagementPanel,
  }))
)
const DeviceManagementPanel = lazy(() =>
  import("./components/device-management-panel").then((module) => ({
    default: module.DeviceManagementPanel,
  }))
)
const DlnaManagementPanel = lazy(() =>
  import("./components/dlna-management-panel").then((module) => ({
    default: module.DlnaManagementPanel,
  }))
)
const LibraryManagementPanel = lazy(() =>
  import("./components/library-management-panel").then((module) => ({
    default: module.LibraryManagementPanel,
  }))
)
const ScanExclusionsPanel = lazy(() =>
  import("./components/scan-exclusions-panel").then((module) => ({
    default: module.ScanExclusionsPanel,
  }))
)
const CleanupSettingsPanel = lazy(() =>
  import("./components/cleanup-settings-panel").then((module) => ({
    default: module.CleanupSettingsPanel,
  }))
)
const TranscodingSettingsPanel = lazy(() =>
  import("./components/transcoding-settings-panel").then((module) => ({
    default: module.TranscodingSettingsPanel,
  }))
)
const NetworkSettingsPanel = lazy(() =>
  import("./components/network-settings-panel").then((module) => ({
    default: module.NetworkSettingsPanel,
  }))
)
const DatabaseSettingsPanel = lazy(() =>
  import("./components/database-settings-panel").then((module) => ({
    default: module.DatabaseSettingsPanel,
  }))
)
const LiveTvSettingsPanel = lazy(() =>
  import("./components/live-tv-settings-panel").then((module) => ({
    default: module.LiveTvSettingsPanel,
  }))
)
const MetadataProviderSettingsPanel = lazy(() =>
  import("./components/metadata-provider-settings-panel").then((module) => ({
    default: module.MetadataProviderSettingsPanel,
  }))
)

export function SettingsGeneralPage() {
  return (
    <SettingsSectionPanel
      sectionKey="general"
      panel={
        <LazyPanel>
          <GeneralSettingsPanel />
        </LazyPanel>
      }
    />
  )
}

export function SettingsHealthPage() {
  return (
    <SettingsNamedPage sectionKey="health">
      <LazyPanel>
        <HealthCenter />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsConsolePage() {
  const section = SETTINGS_SECTIONS.find(({ key }) => key === "console")
  if (!section) return null

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
    >
      <LazyPanel>
        <ConsolePage embedded />
      </LazyPanel>
    </SettingsPageShell>
  )
}

export function SettingsUsersPage() {
  return (
    <SettingsNamedPage sectionKey="users">
      <LazyPanel>
        <UserManagementPanel />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsDevicesPage() {
  return (
    <SettingsNamedPage
      sectionKey="devices"
      actions={
        <Button variant="ghost" size="icon">
          <span className="sr-only">设备帮助</span>
          <HelpCircleIcon className="size-4" />
        </Button>
      }
    >
      <LazyPanel>
        <DeviceManagementPanel />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsDlnaPage() {
  return (
    <SettingsNamedPage
      sectionKey="dlna"
      actions={
        <Button variant="ghost" size="icon">
          <span className="sr-only">DLNA 帮助</span>
          <HelpCircleIcon className="size-4" />
        </Button>
      }
    >
      <LazyPanel>
        <DlnaManagementPanel />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsLibraryPage() {
  const token = useAuthStore((state) => state.token)
  const [activeLibraryTab, setActiveLibraryTab] = useState<
    "sources" | "libraries"
  >("libraries")

  return (
    <SettingsNamedPage
      sectionKey="library"
      actions={
        <SegmentedControl
          value={activeLibraryTab}
          options={[
            { value: "libraries", label: "媒体库" },
            { value: "sources", label: "媒体源" },
          ]}
          onChange={setActiveLibraryTab}
        />
      }
    >
      <LazyPanel>
        <LibraryManagementPanel token={token} activeTab={activeLibraryTab} />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsScanExclusionsPage() {
  const token = useAuthStore((state) => state.token)
  const [activeScanTab, setActiveScanTab] = useState<"rules" | "exclusions">(
    "rules"
  )

  return (
    <SettingsNamedPage
      sectionKey="scan-exclusions"
      actions={
        <SegmentedControl
          value={activeScanTab}
          options={[
            { value: "rules", label: "自动规则" },
            { value: "exclusions", label: "排除项" },
          ]}
          onChange={setActiveScanTab}
        />
      }
    >
      <LazyPanel>
        <ScanExclusionsPanel token={token} activeTab={activeScanTab} />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsCleanupPage() {
  const token = useAuthStore((state) => state.token)
  return (
    <SettingsNamedPage sectionKey="cleanup">
      <LazyPanel>
        <CleanupSettingsPanel token={token} />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

export function SettingsPlaybackPage() {
  return (
    <SettingsSectionPanel
      sectionKey="playback"
      panel={
        <LazyPanel>
          <TranscodingSettingsPanel />
        </LazyPanel>
      }
    />
  )
}

export function SettingsNetworkPage() {
  const token = useAuthStore((state) => state.token)
  return (
    <SettingsSectionPanel
      sectionKey="network"
      panel={
        <LazyPanel>
          <NetworkSettingsPanel token={token} />
        </LazyPanel>
      }
    />
  )
}

export function SettingsDatabasePage() {
  return (
    <SettingsSectionPanel
      sectionKey="database"
      panel={
        <LazyPanel>
          <DatabaseSettingsPanel />
        </LazyPanel>
      }
    />
  )
}

export function SettingsLiveTvPage() {
  return (
    <SettingsSectionPanel
      sectionKey="live-tv"
      panel={
        <LazyPanel>
          <LiveTvSettingsPanel />
        </LazyPanel>
      }
    />
  )
}

export function SettingsNotificationsPage() {
  return (
    <SettingsSectionPanel
      sectionKey="notifications"
      panel={
        <LazyPanel>
          <NotificationSettingsPanel />
        </LazyPanel>
      }
    />
  )
}

export function SettingsSecurityPage() {
  return (
    <SettingsSectionPanel
      sectionKey="security"
      panel={
        <LazyPanel>
          <SecuritySettingsPanel />
        </LazyPanel>
      }
    />
  )
}

export function SettingsMetadataSourcesPage() {
  const token = useAuthStore((state) => state.token)

  return (
    <SettingsNamedPage
      sectionKey="metadata-sources"
      actions={
        <Button asChild variant="outline">
          <Link to="/settings/metadata">
            <KeyRoundIcon className="size-4" />
            打开治理工作台
          </Link>
        </Button>
      }
    >
      <LazyPanel>
        <MetadataProviderSettingsPanel token={token} />
      </LazyPanel>
    </SettingsNamedPage>
  )
}

function SettingsNamedPage({
  sectionKey,
  actions,
  children,
}: {
  sectionKey: string
  actions?: React.ReactNode
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
    >
      {children}
    </SettingsPageShell>
  )
}

function SettingsSectionPanel({
  sectionKey,
  panel,
}: {
  sectionKey:
    | "general"
    | "network"
    | "database"
    | "playback"
    | "notifications"
    | "security"
    | "live-tv"
  panel: React.ReactNode
}) {
  const section = SETTINGS_SECTIONS.find(({ key }) => key === sectionKey)
  if (!section) return null

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

function LazyPanel({ children }: { children: React.ReactNode }) {
  return (
    <Suspense
      fallback={
        <div className="rounded-[1.25rem] border border-border/60 bg-card/80 px-4 py-6 text-sm text-muted-foreground shadow-sm">
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
    <div className="inline-flex rounded-lg border border-border/60 bg-muted/30 p-1">
      {options.map((option) => (
        <Button
          key={option.value}
          type="button"
          onClick={() => onChange(option.value)}
          variant={value === option.value ? "outline" : "ghost"}
          size="sm"
        >
          {option.label}
        </Button>
      ))}
    </div>
  )
}
