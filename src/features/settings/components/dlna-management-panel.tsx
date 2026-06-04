import { useEffect, useState } from 'react'
import {
  ChevronRightIcon,
  MonitorIcon,
  PlusIcon,
  ServerIcon,
  TvIcon,
} from 'lucide-react'
import { createPortal } from 'react-dom'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldTitle,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

const DLNA_SETTINGS_STORAGE_KEY = 'mibo-web-dlna-settings'

type DlnaSettings = {
  playbackEnabled: boolean
  serverEnabled: boolean
  defaultUser: string
  debugLoggingEnabled: boolean
}

type DlnaProfile = {
  id: string
  name: string
  kind: 'custom' | 'system'
  description: string
  mediaTypes: string[]
  maxStreamingQuality: string
  musicTranscodeBitrate: string
}

const defaultSettings: DlnaSettings = {
  playbackEnabled: true,
  serverEnabled: true,
  defaultUser: 'admin',
  debugLoggingEnabled: false,
}

const customProfiles: DlnaProfile[] = [
  {
    id: 'living-room-tv',
    name: '客厅电视',
    kind: 'custom',
    description: '覆盖客厅电视的字幕与图像尺寸策略。',
    mediaTypes: ['音频', '照片', '视频'],
    maxStreamingQuality: '1080p - 20 Mbps',
    musicTranscodeBitrate: '192 kbps',
  },
]

const systemProfiles: DlnaProfile[] = [
  {
    id: 'directv-hd-dvr',
    name: 'DirecTV HD-DVR',
    kind: 'system',
    description: '系统内置配置，适用于 DirecTV DVR 设备。',
    mediaTypes: ['视频', '照片'],
    maxStreamingQuality: '720p - 8 Mbps',
    musicTranscodeBitrate: '128 kbps',
  },
  {
    id: 'generic-device',
    name: 'Generic Device',
    kind: 'system',
    description: '通用 DLNA 设备配置，作为未知设备的默认回退。',
    mediaTypes: ['音频', '照片', '视频'],
    maxStreamingQuality: '原始质量',
    musicTranscodeBitrate: '192 kbps',
  },
  {
    id: 'lg-smart-tv',
    name: 'LG Smart TV',
    kind: 'system',
    description: '针对 LG WebOS 电视的直放与容器兼容策略。',
    mediaTypes: ['音频', '照片', '视频'],
    maxStreamingQuality: '4K - 80 Mbps',
    musicTranscodeBitrate: '320 kbps',
  },
  {
    id: 'samsung-smart-tv',
    name: 'Samsung Smart TV',
    kind: 'system',
    description: '针对 Samsung Smart TV 的响应头和转码策略。',
    mediaTypes: ['音频', '照片', '视频'],
    maxStreamingQuality: '4K - 60 Mbps',
    musicTranscodeBitrate: '320 kbps',
  },
  {
    id: 'sony-blu-ray-player',
    name: 'Sony Blu-ray Player',
    kind: 'system',
    description: '针对 Sony 蓝光播放器的容器和字幕能力。',
    mediaTypes: ['照片', '视频'],
    maxStreamingQuality: '1080p - 15 Mbps',
    musicTranscodeBitrate: '192 kbps',
  },
]

const allProfiles = [...customProfiles, ...systemProfiles]
const profileDetailTabs = [
  '信息',
  '直接播放',
  '转码中',
  '媒体容器',
  '编解码器',
  '响应',
]
const profileGroups = [
  '识别',
  '显示设置',
  '图像设置',
  '服务器设置',
  '字幕配置',
  'XML 设置',
]

export function DlnaManagementPanel() {
  const [activeTab, setActiveTab] = useState<'settings' | 'profiles'>(
    'settings'
  )
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(
    null
  )

  const selectedProfile = allProfiles.find(
    (profile) => profile.id === selectedProfileId
  )

  return (
    <div className='space-y-4 pb-20'>
      <Tabs
        value={activeTab}
        onValueChange={(value) => {
          setActiveTab(value as 'settings' | 'profiles')
          setSelectedProfileId(null)
        }}
      >
        <TabsList>
          <TabsTrigger value='settings'>设置</TabsTrigger>
          <TabsTrigger value='profiles'>Profiles</TabsTrigger>
        </TabsList>
      </Tabs>

      {activeTab === 'settings' ? (
        <DlnaSettingsForm formId='dlna-settings-form' />
      ) : selectedProfile ? (
        <DlnaProfileDetail
          profile={selectedProfile}
          onBack={() => setSelectedProfileId(null)}
        />
      ) : (
        <DlnaProfilesList onSelectProfile={setSelectedProfileId} />
      )}

      {createPortal(
        <div className='fixed right-6 bottom-6 z-[100] flex justify-end'>
          {activeTab === 'settings' ? (
            <Button type='submit' form='dlna-settings-form'>
              保存
            </Button>
          ) : selectedProfile ? (
            <Button
              type='button'
              onClick={() => toast.success('Profile 配置已保存')}
            >
              保存
            </Button>
          ) : null}
        </div>,
        document.body
      )}
    </div>
  )
}

function DlnaSettingsForm({ formId }: { formId: string }) {
  const [draft, setDraft] = useState<DlnaSettings>(defaultSettings)

  useEffect(() => {
    const savedSettings = window.localStorage.getItem(DLNA_SETTINGS_STORAGE_KEY)

    if (!savedSettings) {
      return
    }

    try {
      setDraft({
        ...defaultSettings,
        ...(JSON.parse(savedSettings) as Partial<DlnaSettings>),
      })
    } catch {
      window.localStorage.removeItem(DLNA_SETTINGS_STORAGE_KEY)
    }
  }, [])

  function updateDraft<Value extends keyof DlnaSettings>(
    key: Value,
    value: DlnaSettings[Value]
  ) {
    setDraft((current) => ({ ...current, [key]: value }))
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    window.localStorage.setItem(
      DLNA_SETTINGS_STORAGE_KEY,
      JSON.stringify(draft)
    )
    toast.success('DLNA 设置已保存')
  }

  return (
    <section className='space-y-4'>
      <div className='space-y-1'>
        <h3 className='text-base font-medium'>全局设置</h3>
        <p className='text-sm leading-6'>
          控制 DLNA 播放、媒体服务器发现、默认用户和调试日志。
        </p>
      </div>
      <div>
        <form id={formId} onSubmit={handleSubmit} className='space-y-6'>
          <FieldGroup>
            <DlnaSwitchField
              title='启用 DLNA 播放'
              description='允许 Mibo 检测网络中的 DLNA 设备，并把媒体推送到设备播放。'
              checked={draft.playbackEnabled}
              onCheckedChange={(checked) =>
                updateDraft('playbackEnabled', checked)
              }
            />
            <DlnaSwitchField
              title='启用 DLNA 服务器'
              description='允许服务器向局域网中的 DLNA 设备公开可浏览和可流式传输的媒体。'
              checked={draft.serverEnabled}
              onCheckedChange={(checked) =>
                updateDraft('serverEnabled', checked)
              }
            />

            <Field>
              <FieldLabel>默认用户</FieldLabel>
              <Select
                value={draft.defaultUser}
                onValueChange={(value) => updateDraft('defaultUser', value)}
              >
                <SelectTrigger className='w-full md:max-w-xl'>
                  <SelectValue placeholder='选择默认用户' />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='admin'>admin</SelectItem>
                  <SelectItem value='family'>family</SelectItem>
                  <SelectItem value='guest'>guest</SelectItem>
                </SelectContent>
              </Select>
              <FieldDescription>
                连接设备默认看到该用户可访问的媒体库。单个设备配置可以覆盖此值。
              </FieldDescription>
            </Field>

            <DlnaSwitchField
              title='启用 DLNA 调试日志'
              description='用于排查设备发现、直放和转码问题。开启后可能生成较大的日志文件。'
              checked={draft.debugLoggingEnabled}
              onCheckedChange={(checked) =>
                updateDraft('debugLoggingEnabled', checked)
              }
            />
          </FieldGroup>
        </form>
      </div>
    </section>
  )
}

function DlnaProfilesList({
  onSelectProfile,
}: {
  onSelectProfile: (profileId: string) => void
}) {
  return (
    <div className='space-y-4'>
      <section className='space-y-4'>
        <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
          <div>
            <h3 className='text-base font-medium'>自定义配置</h3>
            <p className='mt-1 text-sm leading-6'>
              创建新的自定义配置，用于新设备或覆盖系统配置。
            </p>
          </div>
          <Button
            variant='outline'
            disabled
            title='DLNA Profile 保存接口接入后启用'
          >
            <PlusIcon className='size-4' />
            新建
          </Button>
        </div>
        <div className='space-y-3'>
          {customProfiles.map((profile) => (
            <ProfileRow
              key={profile.id}
              profile={profile}
              onSelectProfile={onSelectProfile}
            />
          ))}
        </div>
      </section>

      <section className='space-y-4'>
        <div className='space-y-1'>
          <h3 className='text-base font-medium'>系统配置</h3>
          <p className='text-sm leading-6'>
            系统配置为只读模板；修改时会另存为新的自定义配置。
          </p>
        </div>
        <div className='space-y-2'>
          {systemProfiles.map((profile) => (
            <ProfileRow
              key={profile.id}
              profile={profile}
              onSelectProfile={onSelectProfile}
            />
          ))}
        </div>
      </section>
    </div>
  )
}

function ProfileRow({
  profile,
  onSelectProfile,
}: {
  profile: DlnaProfile
  onSelectProfile: (profileId: string) => void
}) {
  const Icon = profile.kind === 'custom' ? TvIcon : MonitorIcon

  return (
    <Button
      type='button'
      onClick={() => onSelectProfile(profile.id)}
      variant='outline'
    >
      <span className='flex size-10 shrink-0 items-center justify-center'>
        <Icon className='size-4' />
      </span>
      <span className='min-w-0 flex-1'>
        <span className='flex items-center gap-2'>
          <span className='truncate text-sm font-medium'>{profile.name}</span>
          <Badge variant='secondary' className='shrink-0'>
            {profile.kind === 'custom' ? '自定义' : '系统'}
          </Badge>
        </span>
        <span className='mt-1 line-clamp-1 text-xs'>{profile.description}</span>
      </span>
      <ChevronRightIcon className='size-4 shrink-0' />
    </Button>
  )
}

function DlnaProfileDetail({
  profile,
  onBack,
}: {
  profile: DlnaProfile
  onBack: () => void
}) {
  return (
    <section className='space-y-4'>
      <div className='space-y-4'>
        <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
          <div>
            <h3 className='text-base font-medium'>配置信息</h3>
            <p className='mt-1 text-sm leading-6'>{profile.description}</p>
          </div>
          <Button variant='outline' onClick={onBack}>
            返回 Profiles
          </Button>
        </div>
        <Tabs value='信息'>
          <TabsList className='h-auto flex-wrap justify-start'>
            {profileDetailTabs.map((tab) => (
              <TabsTrigger key={tab} value={tab} disabled={tab !== '信息'}>
                {tab}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>
      <div className='space-y-6'>
        <div className='grid gap-4 md:grid-cols-2'>
          <Field>
            <FieldLabel htmlFor='dlna-profile-name'>名称</FieldLabel>
            <Input
              id='dlna-profile-name'
              defaultValue={profile.name}
              readOnly={profile.kind === 'system'}
            />
            <FieldDescription>
              {profile.kind === 'system'
                ? '系统配置为只读，保存修改时将创建自定义副本。'
                : '用于在设备识别和配置列表中展示。'}
            </FieldDescription>
          </Field>

          <Field>
            <FieldLabel>用户媒体库</FieldLabel>
            <Select defaultValue='admin'>
              <SelectTrigger className='w-full'>
                <SelectValue placeholder='选择用户媒体库' />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='admin'>admin 的媒体库</SelectItem>
                <SelectItem value='family'>family 的媒体库</SelectItem>
                <SelectItem value='guest'>guest 的媒体库</SelectItem>
              </SelectContent>
            </Select>
            <FieldDescription>
              决定该设备浏览时可见的媒体库范围。
            </FieldDescription>
          </Field>

          <Field>
            <FieldLabel>支持的媒体类型</FieldLabel>
            <div className='flex flex-wrap gap-2'>
              {['音频', '照片', '视频'].map((type) => (
                <Badge
                  key={type}
                  variant={
                    profile.mediaTypes.includes(type) ? 'default' : 'secondary'
                  }
                >
                  {type}
                </Badge>
              ))}
            </div>
            <FieldDescription>设备可浏览和播放的媒体分类。</FieldDescription>
          </Field>

          <Field>
            <FieldLabel>最大流传输质量</FieldLabel>
            <Input defaultValue={profile.maxStreamingQuality} />
            <FieldDescription>超过该质量时会触发降级或转码。</FieldDescription>
          </Field>

          <Field>
            <FieldLabel>音乐转码的比特率</FieldLabel>
            <Input defaultValue={profile.musicTranscodeBitrate} />
            <FieldDescription>音乐转码输出的目标音频码率。</FieldDescription>
          </Field>
        </div>

        <div className='space-y-2'>
          {profileGroups.map((group) => (
            <Button key={group} type='button' variant='outline'>
              <span className='flex size-9 items-center justify-center'>
                <ServerIcon className='size-4' />
              </span>
              <span className='flex-1 text-sm font-medium'>{group}</span>
              <ChevronRightIcon className='size-4' />
            </Button>
          ))}
        </div>
      </div>
    </section>
  )
}

function DlnaSwitchField({
  title,
  description,
  checked,
  onCheckedChange,
}: {
  title: string
  description: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <Field
      orientation='horizontal'
      className='items-start rounded-[1.25rem] border p-3.5'
    >
      <Switch
        checked={checked}
        onCheckedChange={onCheckedChange}
        className='mt-0.5'
      />
      <FieldContent>
        <FieldTitle>{title}</FieldTitle>
        <FieldDescription>{description}</FieldDescription>
      </FieldContent>
    </Field>
  )
}
