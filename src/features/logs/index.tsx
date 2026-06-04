import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { getApiBaseUrl, type AdminLogFile } from '@/lib/mibo-api'
import {
  adminLogSettingsQueryOptions,
  adminLogsQueryOptions,
  createAuthedMiboApi,
  miboQueryKeys,
} from '@/lib/mibo-query'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { SettingsPageShell } from '@/features/settings/components/settings-page-shell'
import { SETTINGS_SECTIONS } from '@/features/settings/sections'
import { LogListPanel } from './components/log-list-panel'
import { LogSettingsPanel } from './components/log-settings-panel'
import { formatBytes, formatDate } from './format'

type ActiveTab = 'logs' | 'settings'

export default function LogsPage() {
  const token = useAuthStore((state) => state.auth.accessToken)
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<ActiveTab>('logs')
  const [previewLog, setPreviewLog] = useState<AdminLogFile | null>(null)
  const [previewContent, setPreviewContent] = useState<string>('')
  const section = SETTINGS_SECTIONS.find(({ key }) => key === 'logs')
  const queryToken = token ?? 'guest'
  const logsQuery = useQuery({
    ...adminLogsQueryOptions(queryToken),
    enabled: !!token,
  })
  const logSettingsQuery = useQuery({
    ...adminLogSettingsQueryOptions(queryToken),
    enabled: !!token,
  })
  const logs = logsQuery.data ?? []

  const previewMutation = useMutation({
    mutationFn: (name: string) =>
      createAuthedMiboApi(queryToken).getAdminLog(name),
    onSuccess: (data, name) => {
      setPreviewLog(logs.find((log) => log.name === name) ?? null)
      setPreviewContent(
        data.truncated
          ? `仅显示最后 ${formatBytes(data.max_bytes)}，完整文件大小 ${formatBytes(data.size_bytes)}。\n\n${data.content}`
          : data.content
      )
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) =>
      createAuthedMiboApi(queryToken).deleteAdminLog(name),
    onSuccess: async () => {
      toast.success('日志已删除')
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.adminLogs(queryToken),
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  if (!section) {
    return null
  }
  const settingsSection = section
  const navigationTabs = (
    <Tabs
      value={activeTab}
      onValueChange={(value) => setActiveTab(value as ActiveTab)}
    >
      <TabsList className='rounded-full'>
        <TabsTrigger className='rounded-full' value='logs'>
          日志
        </TabsTrigger>
        <TabsTrigger className='rounded-full' value='settings'>
          设置
        </TabsTrigger>
      </TabsList>
    </Tabs>
  )

  async function downloadLog(log: AdminLogFile) {
    if (!token) {
      return
    }

    try {
      const response = await fetch(
        `${getApiBaseUrl().replace(/\/$/, '')}/api/v1/admin/log-download/${encodeURIComponent(log.name)}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      )

      if (!response.ok) {
        throw new Error(`下载失败，状态码 ${response.status}`)
      }

      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = log.name
      anchor.click()
      window.URL.revokeObjectURL(url)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '下载日志失败')
    }
  }

  return (
    <SettingsPageShell
      icon={settingsSection.icon}
      title={settingsSection.title}
      description={settingsSection.description}
      showHeader={false}
    >
      {activeTab === 'logs' ? (
        <LogListPanel
          tabs={navigationTabs}
          logs={logs}
          isLoading={logsQuery.isLoading}
          isRefreshing={logsQuery.isFetching}
          onRefresh={() => logsQuery.refetch()}
          onPreview={(log) => previewMutation.mutate(log.name)}
          onDownload={downloadLog}
          onDelete={(log) => {
            if (window.confirm(`删除日志 ${log.name}？`)) {
              deleteMutation.mutate(log.name)
            }
          }}
        />
      ) : (
        <LogSettingsPanel
          tabs={navigationTabs}
          token={token}
          queryToken={queryToken}
          settings={logSettingsQuery.data}
          isLoading={logSettingsQuery.isLoading}
        />
      )}

      <Dialog
        open={!!previewLog || previewMutation.isPending}
        onOpenChange={(open) => {
          if (!open) {
            setPreviewLog(null)
            setPreviewContent('')
          }
        }}
      >
        <DialogContent className='max-h-[82vh] max-w-4xl overflow-hidden'>
          <DialogHeader>
            <DialogTitle>{previewLog?.name ?? '读取日志'}</DialogTitle>
            <DialogDescription>
              {previewLog
                ? `${formatDate(previewLog.modified_at)} · ${formatBytes(previewLog.size_bytes)}`
                : '正在加载日志内容'}
            </DialogDescription>
          </DialogHeader>
          <pre className='max-h-[60vh] overflow-auto rounded-2xl bg-muted/60 p-4 text-xs leading-5 text-muted-foreground'>
            {previewMutation.isPending ? '正在加载...' : previewContent}
          </pre>
        </DialogContent>
      </Dialog>
    </SettingsPageShell>
  )
}
