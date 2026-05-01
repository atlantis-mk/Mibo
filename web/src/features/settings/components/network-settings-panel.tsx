import { useEffect, useRef, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { InfoIcon, LoaderCircleIcon, UploadIcon } from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '#/components/ui/alert'
import { Button } from '#/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldTitle,
} from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '#/components/ui/select'
import { Separator } from '#/components/ui/separator'
import { Switch } from '#/components/ui/switch'
import { Textarea } from '#/components/ui/textarea'
import type { NetworkSettings, NetworkSettingsInput } from '#/lib/mibo-api'
import {
  createAuthedMiboApi,
  miboQueryKeys,
  networkSettingsQueryOptions,
} from '#/lib/mibo-query'

type NetworkSettingsForm = {
  localNetworks: string
  localIpAddress: string
  localHttpPort: string
  localHttpsPort: string
  allowRemoteAccess: boolean
  remoteIpFilter: string
  remoteIpFilterMode: string
  publicHttpPort: string
  publicHttpsPort: string
  externalDomain: string
  trustProxyHeaders: boolean
  sslCertificatePath: string
  certificatePassword: string
  secureConnectionMode: string
  automaticPortMapping: boolean
  maxVideoStreams: string
  remoteStreamingBitrateLimit: string
  networkRequestProtocol: string
  clearCertificatePassword: boolean
}

const defaultNetworkSettings: NetworkSettingsForm = {
  localNetworks: '192.168.1.0/24\n10.0.0.0/8',
  localIpAddress: '',
  localHttpPort: '8096',
  localHttpsPort: '8920',
  allowRemoteAccess: true,
  remoteIpFilter: '',
  remoteIpFilterMode: 'allow',
  publicHttpPort: '8096',
  publicHttpsPort: '8920',
  externalDomain: '',
  trustProxyHeaders: false,
  sslCertificatePath: '',
  certificatePassword: '',
  secureConnectionMode: 'disabled',
  automaticPortMapping: false,
  maxVideoStreams: 'unlimited',
  remoteStreamingBitrateLimit: 'unlimited',
  networkRequestProtocol: 'auto',
  clearCertificatePassword: false,
}

export function NetworkSettingsPanel({ token }: { token: string | null }) {
  const queryClient = useQueryClient()
  const networkQuery = useQuery({
    ...networkSettingsQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })
  const [draft, setDraft] = useState<NetworkSettingsForm>(
    defaultNetworkSettings,
  )
  const certificateInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (networkQuery.data) {
      setDraft(networkSettingsToForm(networkQuery.data))
    }
  }, [networkQuery.data])

  const saveMutation = useMutation({
    mutationFn: async (nextDraft: NetworkSettingsForm) => {
      if (!token) {
        throw new Error('当前未登录，无法保存网络设置。')
      }

      return createAuthedMiboApi(token).updateNetworkSettings(
        networkFormToInput(nextDraft),
      )
    },
    onSuccess: (settings) => {
      if (!token) {
        return
      }

      queryClient.setQueryData(miboQueryKeys.networkSettings(token), settings)
      setDraft(networkSettingsToForm(settings))
      toast.success('网络设置已保存')
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  function updateDraft<Value extends keyof NetworkSettingsForm>(
    key: Value,
    value: NetworkSettingsForm[Value],
  ) {
    setDraft((current) => ({ ...current, [key]: value }))
  }

  function handleCertificateFileChange(
    event: React.ChangeEvent<HTMLInputElement>,
  ) {
    const file = event.target.files?.[0]

    if (!file) {
      return
    }

    updateDraft('sslCertificatePath', file.name)
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    saveMutation.mutate(draft)
  }

  if (!token) {
    return (
      <Alert>
        <InfoIcon className="size-4" />
        <AlertTitle>登录后可管理网络设置</AlertTitle>
        <AlertDescription className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <span>当前页面需要管理员会话来读取和更新服务器网络配置。</span>
          <Button asChild variant="outline" className="w-fit">
            <Link to="/login" search={{ redirect: '/settings/network' }}>
              前往登录
            </Link>
          </Button>
        </AlertDescription>
      </Alert>
    )
  }

  if (networkQuery.isLoading) {
    return (
      <div className="flex items-center gap-3 rounded-[1.25rem] border border-border/60 bg-card/80 px-4 py-6 text-sm text-muted-foreground shadow-sm">
        <LoaderCircleIcon className="size-4 animate-spin" />
        正在加载网络设置
      </div>
    )
  }

  if (networkQuery.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{networkQuery.error.message}</AlertDescription>
      </Alert>
    )
  }

  const passwordConfigured =
    networkQuery.data?.certificate_password.configured ?? false

  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle className="text-xl">网络</CardTitle>
        <CardDescription>
          配置服务器在内网、外网、反向代理和安全连接场景下的访问方式。
        </CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-5 px-5 py-5">
        <div className="flex items-start gap-3 rounded-[1.15rem] border border-border/60 bg-muted/30 px-4 py-3 text-sm leading-6 text-muted-foreground">
          <InfoIcon className="mt-0.5 size-4 shrink-0" />
          <span>
            网络设置会保存到服务器。监听端口、TLS
            和部分播放限制保存后可能需要重启或后续运行时支持才会实际生效。
          </span>
        </div>

        {networkQuery.data?.effective_status ? (
          <div className="space-y-2 rounded-[1.15rem] border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm leading-6 text-amber-900 dark:text-amber-100">
            <p>{networkQuery.data.effective_status.message}</p>
            <p>自动端口映射当前为配置项，尚未执行实际 UPnP / NAT-PMP 映射。</p>
          </div>
        ) : null}

        {saveMutation.error ? (
          <Alert variant="destructive">
            <AlertTitle>保存失败</AlertTitle>
            <AlertDescription>{saveMutation.error.message}</AlertDescription>
          </Alert>
        ) : null}

        <form onSubmit={handleSubmit} className="space-y-6">
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="network-local-networks">
                局域网网络
              </FieldLabel>
              <Textarea
                id="network-local-networks"
                value={draft.localNetworks}
                onChange={(event) =>
                  updateDraft('localNetworks', event.target.value)
                }
                placeholder="192.168.1.0/24&#10;10.0.0.0/8"
                className="min-h-24 border-border/60 bg-background font-mono text-sm text-foreground placeholder:text-muted-foreground"
              />
              <FieldDescription>
                每行填写一个被视为本地访问的 IP 地址或 CIDR
                网段，用于区分本地和远程设备。
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor="network-local-ip">本地 IP 地址</FieldLabel>
              <Input
                id="network-local-ip"
                value={draft.localIpAddress}
                onChange={(event) =>
                  updateDraft('localIpAddress', event.target.value)
                }
                placeholder="留空则由服务器自动检测"
                className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
              />
              <FieldDescription>
                手动指定 Mibo
                提供给客户端使用的本地地址。多网卡或容器部署时可用于避免自动检测错误。
              </FieldDescription>
            </Field>

            <div className="grid gap-4 md:grid-cols-2">
              <NumberField
                id="network-local-http-port"
                label="本地 HTTP 端口"
                value={draft.localHttpPort}
                onChange={(value) => updateDraft('localHttpPort', value)}
                description="服务器监听的本地 HTTP 端口，Emby 默认示例为 8096。"
              />
              <NumberField
                id="network-local-https-port"
                label="本地 HTTPS 端口"
                value={draft.localHttpsPort}
                onChange={(value) => updateDraft('localHttpsPort', value)}
                description="服务器监听的本地 HTTPS 端口，Emby 默认示例为 8920。"
              />
            </div>

            <FormSwitchField
              title="允许远程访问"
              description="开启后允许外部网络连接到服务器；关闭后仅允许本地网络访问。"
              checked={draft.allowRemoteAccess}
              onCheckedChange={(checked) =>
                updateDraft('allowRemoteAccess', checked)
              }
            />

            <Field>
              <FieldLabel htmlFor="network-remote-ip-filter">
                远程 IP 地址筛选
              </FieldLabel>
              <Textarea
                id="network-remote-ip-filter"
                value={draft.remoteIpFilter}
                onChange={(event) =>
                  updateDraft('remoteIpFilter', event.target.value)
                }
                placeholder="203.0.113.10&#10;198.51.100.0/24"
                className="min-h-24 border-border/60 bg-background font-mono text-sm text-foreground placeholder:text-muted-foreground"
              />
              <FieldDescription>
                每行填写一个允许或禁止远程连接的 IP
                地址或网段，配合筛选模式生效。
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel>远程 IP 地址筛选模式</FieldLabel>
              <Select
                value={draft.remoteIpFilterMode}
                onValueChange={(value) =>
                  updateDraft('remoteIpFilterMode', value)
                }
              >
                <SelectTrigger className="w-full border-border/60 bg-background text-foreground md:max-w-md">
                  <SelectValue placeholder="选择筛选模式" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="allow">
                    白名单，仅允许列表中的地址
                  </SelectItem>
                  <SelectItem value="block">
                    黑名单，阻止列表中的地址
                  </SelectItem>
                </SelectContent>
              </Select>
              <FieldDescription>
                白名单更适合私有部署；黑名单适合临时阻断异常来源。
              </FieldDescription>
            </Field>

            <div className="grid gap-4 md:grid-cols-2">
              <NumberField
                id="network-public-http-port"
                label="公网 HTTP 端口"
                value={draft.publicHttpPort}
                onChange={(value) => updateDraft('publicHttpPort', value)}
                description="路由器或反向代理映射到本地 HTTP 端口的公网端口。"
              />
              <NumberField
                id="network-public-https-port"
                label="公网 HTTPS 端口"
                value={draft.publicHttpsPort}
                onChange={(value) => updateDraft('publicHttpsPort', value)}
                description="路由器或反向代理映射到本地 HTTPS 端口的公网端口。"
              />
            </div>

            <Field>
              <FieldLabel htmlFor="network-external-domain">外部域</FieldLabel>
              <Input
                id="network-external-domain"
                value={draft.externalDomain}
                onChange={(event) =>
                  updateDraft('externalDomain', event.target.value)
                }
                placeholder="media.example.com"
                className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
              />
              <FieldDescription>
                填写 DDNS 或自定义域名，供远程客户端生成连接地址。
              </FieldDescription>
            </Field>

            <FormSwitchField
              title="读取代理标头"
              description="启用后通过 X-Real-IP、X-Forwarded-For 等请求头识别真实客户端 IP，适合反向代理场景。"
              checked={draft.trustProxyHeaders}
              onCheckedChange={(checked) =>
                updateDraft('trustProxyHeaders', checked)
              }
            />

            <Field>
              <FieldLabel htmlFor="network-ssl-certificate">
                自定义 SSL 证书路径
              </FieldLabel>
              <div className="flex flex-col gap-3 sm:flex-row">
                <Input
                  id="network-ssl-certificate"
                  value={draft.sslCertificatePath}
                  onChange={(event) =>
                    updateDraft('sslCertificatePath', event.target.value)
                  }
                  placeholder="/config/certs/mibo.pfx"
                  className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
                />
                <input
                  ref={certificateInputRef}
                  type="file"
                  accept=".p12,.pfx,application/x-pkcs12"
                  className="hidden"
                  onChange={handleCertificateFileChange}
                />
                <Button
                  type="button"
                  variant="outline"
                  className="shrink-0"
                  onClick={() => certificateInputRef.current?.click()}
                >
                  <UploadIcon className="size-4" />
                  选择文件
                </Button>
              </div>
              <FieldDescription>
                指定包含证书和私钥的 PKCS #12
                文件。浏览器选择文件时只能读取文件名，服务器路径仍可手动填写。
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor="network-certificate-password">
                证书密码
              </FieldLabel>
              <Input
                id="network-certificate-password"
                type="password"
                value={draft.certificatePassword}
                onChange={(event) =>
                  setDraft((current) => ({
                    ...current,
                    certificatePassword: event.target.value,
                    clearCertificatePassword: false,
                  }))
                }
                placeholder={
                  passwordConfigured
                    ? '已配置，留空则保持当前密码'
                    : 'PKCS #12 证书密码'
                }
                className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
              />
              <FieldDescription>
                密码只会写入服务器，不会在读取设置时回显。
                {passwordConfigured ? ' 当前已有密码配置。' : ''}
              </FieldDescription>
              {passwordConfigured ? (
                <FormSwitchField
                  title="清除已保存的证书密码"
                  description="保存时删除服务器中已配置的证书密码。输入新密码会自动取消清除。"
                  checked={draft.clearCertificatePassword}
                  onCheckedChange={(checked) =>
                    setDraft((current) => ({
                      ...current,
                      clearCertificatePassword: checked,
                      certificatePassword: checked
                        ? ''
                        : current.certificatePassword,
                    }))
                  }
                />
              ) : null}
            </Field>

            <Field>
              <FieldLabel>安全连接模式</FieldLabel>
              <Select
                value={draft.secureConnectionMode}
                onValueChange={(value) =>
                  updateDraft('secureConnectionMode', value)
                }
              >
                <SelectTrigger className="w-full border-border/60 bg-background text-foreground md:max-w-md">
                  <SelectValue placeholder="选择安全连接模式" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="disabled">已禁用</SelectItem>
                  <SelectItem value="preferred">优先使用 HTTPS</SelectItem>
                  <SelectItem value="required">要求 HTTPS</SelectItem>
                </SelectContent>
              </Select>
              <FieldDescription>
                控制客户端是否应使用安全连接，以及是否允许回退到 HTTP。
              </FieldDescription>
            </Field>

            <FormSwitchField
              title="启用自动端口映射"
              description="保存自动端口映射偏好；实际 UPnP / NAT-PMP 映射尚未接入运行时。"
              checked={draft.automaticPortMapping}
              onCheckedChange={(checked) =>
                updateDraft('automaticPortMapping', checked)
              }
            />

            <Field>
              <FieldLabel>最大同步视频流</FieldLabel>
              <Select
                value={draft.maxVideoStreams}
                onValueChange={(value) => updateDraft('maxVideoStreams', value)}
              >
                <SelectTrigger className="w-full border-border/60 bg-background text-foreground md:max-w-md">
                  <SelectValue placeholder="选择最大同步视频流" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="unlimited">无限制</SelectItem>
                  <SelectItem value="1">1 路</SelectItem>
                  <SelectItem value="2">2 路</SelectItem>
                  <SelectItem value="4">4 路</SelectItem>
                  <SelectItem value="8">8 路</SelectItem>
                </SelectContent>
              </Select>
              <FieldDescription>
                限制并发播放会话数量，避免带宽或转码资源被单一场景耗尽。
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel>远程流传输比特率限制</FieldLabel>
              <Select
                value={draft.remoteStreamingBitrateLimit}
                onValueChange={(value) =>
                  updateDraft('remoteStreamingBitrateLimit', value)
                }
              >
                <SelectTrigger className="w-full border-border/60 bg-background text-foreground md:max-w-md">
                  <SelectValue placeholder="选择比特率限制" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="unlimited">无限制</SelectItem>
                  <SelectItem value="4mbps">4 Mbps</SelectItem>
                  <SelectItem value="8mbps">8 Mbps</SelectItem>
                  <SelectItem value="12mbps">12 Mbps</SelectItem>
                  <SelectItem value="20mbps">20 Mbps</SelectItem>
                </SelectContent>
              </Select>
              <FieldDescription>
                限制外网设备的播放码率，降低公网带宽和转码 CPU 压力。
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel>网络请求协议</FieldLabel>
              <Select
                value={draft.networkRequestProtocol}
                onValueChange={(value) =>
                  updateDraft('networkRequestProtocol', value)
                }
              >
                <SelectTrigger className="w-full border-border/60 bg-background text-foreground md:max-w-md">
                  <SelectValue placeholder="选择网络请求协议" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">自动</SelectItem>
                  <SelectItem value="ipv4">仅 IPv4</SelectItem>
                  <SelectItem value="ipv6">仅 IPv6</SelectItem>
                </SelectContent>
              </Select>
              <FieldDescription>
                控制元数据提供方和外部服务请求优先使用的网络协议，用于兼容 IPv4
                / IPv6 环境。
              </FieldDescription>
            </Field>
          </FieldGroup>

          <Button
            type="submit"
            size="lg"
            className="w-full bg-emerald-600 text-white hover:bg-emerald-700"
            disabled={saveMutation.isPending}
          >
            {saveMutation.isPending ? '正在保存...' : '保存'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

function NumberField({
  id,
  label,
  value,
  onChange,
  description,
}: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
  description: string
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input
        id={id}
        type="number"
        min="1"
        max="65535"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
      />
      <FieldDescription>{description}</FieldDescription>
    </Field>
  )
}

function FormSwitchField({
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
      orientation="horizontal"
      className="items-start rounded-[1.25rem] border border-border/60 bg-muted/30 p-3.5"
    >
      <Switch
        checked={checked}
        onCheckedChange={onCheckedChange}
        className="mt-0.5 data-[state=checked]:bg-emerald-600"
      />
      <FieldContent>
        <FieldTitle className="text-foreground">{title}</FieldTitle>
        <FieldDescription>{description}</FieldDescription>
      </FieldContent>
    </Field>
  )
}

function networkSettingsToForm(settings: NetworkSettings): NetworkSettingsForm {
  return {
    localNetworks: listToText(settings.local_networks),
    localIpAddress: settings.local_ip_address,
    localHttpPort: String(settings.local_http_port),
    localHttpsPort: String(settings.local_https_port),
    allowRemoteAccess: settings.allow_remote_access,
    remoteIpFilter: listToText(settings.remote_ip_filter),
    remoteIpFilterMode: settings.remote_ip_filter_mode,
    publicHttpPort: String(settings.public_http_port),
    publicHttpsPort: String(settings.public_https_port),
    externalDomain: settings.external_domain,
    trustProxyHeaders: settings.trust_proxy_headers,
    sslCertificatePath: settings.ssl_certificate_path,
    certificatePassword: '',
    secureConnectionMode: settings.secure_connection_mode,
    automaticPortMapping: settings.automatic_port_mapping,
    maxVideoStreams: settings.max_video_streams,
    remoteStreamingBitrateLimit: settings.remote_streaming_bitrate_limit,
    networkRequestProtocol: settings.network_request_protocol,
    clearCertificatePassword: false,
  }
}

function networkFormToInput(draft: NetworkSettingsForm): NetworkSettingsInput {
  return {
    local_networks: textToList(draft.localNetworks),
    local_ip_address: draft.localIpAddress.trim(),
    local_http_port: Number(draft.localHttpPort),
    local_https_port: Number(draft.localHttpsPort),
    allow_remote_access: draft.allowRemoteAccess,
    remote_ip_filter: textToList(draft.remoteIpFilter),
    remote_ip_filter_mode:
      draft.remoteIpFilterMode as NetworkSettingsInput['remote_ip_filter_mode'],
    public_http_port: Number(draft.publicHttpPort),
    public_https_port: Number(draft.publicHttpsPort),
    external_domain: draft.externalDomain.trim(),
    trust_proxy_headers: draft.trustProxyHeaders,
    ssl_certificate_path: draft.sslCertificatePath.trim(),
    certificate_password: draft.certificatePassword,
    clear_certificate_password: draft.clearCertificatePassword,
    secure_connection_mode:
      draft.secureConnectionMode as NetworkSettingsInput['secure_connection_mode'],
    automatic_port_mapping: draft.automaticPortMapping,
    max_video_streams:
      draft.maxVideoStreams as NetworkSettingsInput['max_video_streams'],
    remote_streaming_bitrate_limit:
      draft.remoteStreamingBitrateLimit as NetworkSettingsInput['remote_streaming_bitrate_limit'],
    network_request_protocol:
      draft.networkRequestProtocol as NetworkSettingsInput['network_request_protocol'],
  }
}

function listToText(values: string[] | null | undefined) {
  return (values ?? []).join('\n')
}

function textToList(value: string) {
  return value
    .split('\n')
    .map((entry) => entry.trim())
    .filter(Boolean)
}
