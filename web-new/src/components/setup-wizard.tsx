"use client"

import { type FormEvent, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import {
  ArrowRight,
  CheckCircle2,
  Loader2,
  Server,
  Sparkles,
  UserRound,
} from 'lucide-react'
import { toast } from 'sonner'

import {
  buildSuggestedLibraryRootPath,
  DEFAULT_LOCAL_MEDIA_ROOT_PATH,
  DEFAULT_OPENLIST_BASE_URL,
  LIBRARY_TYPE_OPTIONS,
  STORAGE_PROVIDER_OPTIONS,
} from '~/features/setup/constants'
import {
  API_BASE_STORAGE_KEY,
  defaultApiBaseUrl,
  isSetupFullyInitialized,
  needsSetupGuide,
  SETUP_STATUS_EVENT,
  TOKEN_STORAGE_KEY,
} from '~/lib/client-config'
import { ApiError, createMiboApi, DEFAULT_BROWSE_FILTERS, type MediaSource, type SetupStatus } from '~/lib/mibo-api'
import { buildBrowseRouteSearch } from '~/lib/route-search'
import { Badge } from '~/components/ui/badge'
import { Button } from '~/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '~/components/ui/card'
import { Input } from '~/components/ui/input'
import { Progress } from '~/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '~/components/ui/select'

type WizardStep = 0 | 1 | 2 | 3 | 4

export function SetupWizard() {
  const navigate = useNavigate()
  const [apiBaseUrl, setApiBaseUrl] = useState(defaultApiBaseUrl)
  const [draftApiBaseUrl, setDraftApiBaseUrl] = useState(defaultApiBaseUrl)
  const [token, setToken] = useState<string | null>(null)
  const [setupStatus, setSetupStatus] = useState<SetupStatus | null>(null)
  const [currentStep, setCurrentStep] = useState<WizardStep>(0)
  const [isLoading, setIsLoading] = useState(false)
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('admin123')
  const [sourceProvider, setSourceProvider] = useState('local')
  const [sourceName, setSourceName] = useState('Home Media')
  const [sourceRootPath, setSourceRootPath] = useState(DEFAULT_LOCAL_MEDIA_ROOT_PATH)
  const [sourceOpenListBaseUrl, setSourceOpenListBaseUrl] = useState(DEFAULT_OPENLIST_BASE_URL)
  const [sourceOpenListUsername, setSourceOpenListUsername] = useState('')
  const [sourceOpenListPassword, setSourceOpenListPassword] = useState('')
  const [libraryType, setLibraryType] = useState<string>(LIBRARY_TYPE_OPTIONS[0].value)
  const [libraryName, setLibraryName] = useState<string>(LIBRARY_TYPE_OPTIONS[0].pathSuffix)
  const [libraryRootPath, setLibraryRootPath] = useState(
    buildSuggestedLibraryRootPath(DEFAULT_LOCAL_MEDIA_ROOT_PATH, LIBRARY_TYPE_OPTIONS[0].pathSuffix)
  )
  const [mediaSourceId, setMediaSourceId] = useState<number | null>(null)
  const [mediaSources, setMediaSources] = useState<MediaSource[]>([])

  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }

    const storedApiBaseUrl = window.localStorage.getItem(API_BASE_STORAGE_KEY)
    const storedToken = window.localStorage.getItem(TOKEN_STORAGE_KEY)
    const nextApiBaseUrl = storedApiBaseUrl ?? defaultApiBaseUrl

    setApiBaseUrl(nextApiBaseUrl)
    setDraftApiBaseUrl(nextApiBaseUrl)
    setToken(storedToken)
  }, [])

  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }

    window.localStorage.setItem(API_BASE_STORAGE_KEY, apiBaseUrl)
  }, [apiBaseUrl])

  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }

    if (token) {
      window.localStorage.setItem(TOKEN_STORAGE_KEY, token)
      return
    }

    window.localStorage.removeItem(TOKEN_STORAGE_KEY)
  }, [token])

  const api = useMemo(() => createMiboApi({ baseUrl: apiBaseUrl, token }), [apiBaseUrl, token])

  const selectedLibraryType = useMemo(
    () => LIBRARY_TYPE_OPTIONS.find((option) => option.value === libraryType) ?? LIBRARY_TYPE_OPTIONS[0],
    [libraryType]
  )

  useEffect(() => {
    void refreshStatus(apiBaseUrl)
  }, [apiBaseUrl])

  useEffect(() => {
    if (!token || !(setupStatus?.has_media_sources ?? false)) {
      setMediaSources([])
      setMediaSourceId(null)
      return
    }

    let cancelled = false

    const loadMediaSources = async () => {
      try {
        const sources = await api.listMediaSources()

        if (!cancelled) {
          setMediaSources(sources)
        }
      } catch {
        if (!cancelled) {
          setMediaSources([])
        }
      }
    }

    void loadMediaSources()

    return () => {
      cancelled = true
    }
  }, [api, setupStatus?.has_media_sources, token])

  useEffect(() => {
    if (mediaSources.length === 0) {
      setMediaSourceId(null)
      return
    }

    if (mediaSourceId !== null && mediaSources.some((source) => source.id === mediaSourceId)) {
      return
    }

    const nextSource = mediaSources[0]
    setMediaSourceId(nextSource.id)
    setLibraryRootPath(buildSuggestedLibraryRootPath(nextSource.root_path, selectedLibraryType.pathSuffix))
  }, [mediaSourceId, mediaSources, selectedLibraryType.pathSuffix])

  async function refreshStatus(baseUrl = apiBaseUrl) {
    try {
      const status = await createMiboApi({ baseUrl }).getSetupStatus()

      setSetupStatus(status)
      window.dispatchEvent(new Event(SETUP_STATUS_EVENT))

      if (isSetupFullyInitialized(status)) {
        setCurrentStep(4)
        return
      }

      if (status.has_libraries) {
        setCurrentStep(4)
      } else if (status.has_media_sources) {
        setCurrentStep(3)
      } else if (status.has_users) {
        setCurrentStep(2)
      } else {
        setCurrentStep(1)
      }
    } catch (error) {
      handleApiError(error, '无法连接后端服务')
    }
  }

  function handleApiError(error: unknown, fallbackMessage: string) {
    if (error instanceof ApiError) {
      toast.error(error.message)
      return
    }

    toast.error(fallbackMessage)
  }

  async function handleRegister(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsLoading(true)

    try {
      await api.register(username, password)
      const login = await api.login(username, password)
      setToken(login.token)
      toast.success('管理员账号已创建')
      await refreshStatus()
    } catch (error) {
      handleApiError(error, '无法创建账号')
    } finally {
      setIsLoading(false)
    }
  }

  async function handleCreateSource(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsLoading(true)

    try {
      const source = await api.createMediaSource({
        provider: sourceProvider,
        name: sourceName,
        root_path: sourceRootPath,
        config:
          sourceProvider === 'openlist'
            ? {
                openlist: {
                  base_url: sourceOpenListBaseUrl,
                  username: sourceOpenListUsername || undefined,
                  password: sourceOpenListPassword || undefined,
                },
              }
            : undefined,
      })

      setMediaSources([source])
      setMediaSourceId(source.id)
      toast.success('媒体源已创建')
      await refreshStatus()
    } catch (error) {
      handleApiError(error, '无法创建媒体源')
    } finally {
      setIsLoading(false)
    }
  }

  async function handleCreateLibrary(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    if (mediaSourceId === null) {
      toast.error('请先选择媒体源')
      return
    }

    setIsLoading(true)

    try {
      await api.createLibrary({
        name: libraryName,
        type: libraryType,
        media_source_id: mediaSourceId,
        root_path: libraryRootPath,
      })

      toast.success('媒体库已创建，首次扫描已加入队列')
      await refreshStatus()
    } catch (error) {
      handleApiError(error, '无法创建媒体库')
    } finally {
      setIsLoading(false)
    }
  }

  async function handleSkipToApp() {
    await navigate({
      to: '/',
      search: buildBrowseRouteSearch(DEFAULT_BROWSE_FILTERS),
    })
  }

  const progressValue = ((currentStep + 1) / 5) * 100
  const canSkipToApp = setupStatus ? needsSetupGuide(setupStatus) || setupStatus.can_enter_app : false

  return (
    <div className="min-h-screen bg-background bg-[radial-gradient(circle_at_top_left,_rgba(99,102,241,0.18),_transparent_30%),radial-gradient(circle_at_bottom_right,_rgba(34,197,94,0.14),_transparent_28%)] px-4 py-8 text-foreground sm:px-6 lg:px-8">
      <div className="mx-auto grid min-h-[calc(100vh-4rem)] max-w-6xl gap-6 lg:grid-cols-[0.9fr_1.1fr]">
        <Card className="border-none bg-transparent shadow-none ring-0">
          <CardHeader className="px-0 pt-0">
            <Badge variant="outline" className="w-fit border-primary/30 bg-primary/5 text-primary">
              初始化向导
            </Badge>
            <CardTitle className="text-4xl font-semibold tracking-tight sm:text-5xl">
              在 `web-new/` 里完成 Mibo 的首次初始化。
            </CardTitle>
            <CardDescription className="max-w-xl text-base leading-7">
              这一版先迁移最关键的 setup 主路径，让 TanStack Start 新壳能够连接后端、创建管理员、添加媒体源并落地首个媒体库。
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 px-0">
            <Progress value={progressValue} className="h-2" />
            <div className="grid gap-3">
              {[
                { step: 0, title: '连接后端', done: true },
                { step: 1, title: '创建首个用户', done: Boolean(setupStatus?.has_users) },
                { step: 2, title: '添加媒体源（可稍后）', done: Boolean(setupStatus?.has_media_sources) },
                { step: 3, title: '创建媒体库（可稍后）', done: Boolean(setupStatus?.has_libraries) },
                { step: 4, title: '完成并进入应用', done: Boolean(setupStatus?.initialized) },
              ].map((item) => (
                <div
                  key={item.title}
                  className="flex items-center gap-3 rounded-2xl border border-border/70 bg-card/70 p-4 backdrop-blur"
                >
                  <div className="flex size-8 items-center justify-center rounded-full bg-primary/10 text-primary">
                    {item.done ? <CheckCircle2 className="size-4" /> : <span className="text-xs font-semibold">{item.step + 1}</span>}
                  </div>
                  <div>
                    <div className="font-medium">{item.title}</div>
                    <div className="text-sm text-muted-foreground">{item.done ? '已完成' : '待完成'}</div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card className="self-center border-border/70 bg-card/90 shadow-xl backdrop-blur">
          <CardHeader>
            <CardTitle>
              {currentStep === 0 && '后端地址'}
              {currentStep === 1 && '创建首个管理员'}
              {currentStep === 2 && '添加第一个媒体源'}
              {currentStep === 3 && '创建第一个媒体库'}
              {currentStep === 4 && '初始化完成'}
            </CardTitle>
            <CardDescription>
              {currentStep === 0 && '把向导指向正在运行的 mibo-media-server API。'}
              {currentStep === 1 && '创建首个账号，用来管理媒体库与播放历史。'}
              {currentStep === 2 && '选择本地目录或 OpenList 作为第一个媒体源。'}
              {currentStep === 3 && '创建媒体库并触发首次扫描。'}
              {currentStep === 4 && '新框架下的初始化主路径已经可用。'}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {currentStep === 0 && (
              <div className="space-y-4">
                <Input
                  value={draftApiBaseUrl}
                  onChange={(event) => setDraftApiBaseUrl(event.target.value)}
                  placeholder="http://127.0.0.1:8080"
                />
                <Button
                  onClick={async () => {
                    const next = draftApiBaseUrl.trim() || defaultApiBaseUrl
                    setApiBaseUrl(next)
                    await refreshStatus(next)
                    toast.success('后端地址已更新')
                  }}
                >
                  <Server className="size-4" />
                  验证后端
                </Button>
              </div>
            )}

            {currentStep === 1 && (
              <form className="grid gap-3" onSubmit={handleRegister}>
                <Input
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  placeholder="用户名"
                  autoComplete="username"
                />
                <Input
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  type="password"
                  placeholder="密码（至少 8 位）"
                  autoComplete="new-password"
                />
                <Button type="submit" disabled={isLoading}>
                  {isLoading ? <Loader2 className="size-4 animate-spin" /> : <UserRound className="size-4" />}
                  创建管理员
                </Button>
              </form>
            )}

            {currentStep === 2 && (
              <form className="grid gap-3" onSubmit={handleCreateSource}>
                <Select
                  value={sourceProvider}
                  onValueChange={(value) => {
                    const option = STORAGE_PROVIDER_OPTIONS.find((item) => item.value === value)
                    setSourceProvider(value)
                    setSourceRootPath(option?.examplePath ?? sourceRootPath)
                    setLibraryRootPath(
                      buildSuggestedLibraryRootPath(
                        option?.examplePath ?? sourceRootPath,
                        LIBRARY_TYPE_OPTIONS.find((item) => item.value === libraryType)?.pathSuffix ??
                          LIBRARY_TYPE_OPTIONS[0].pathSuffix
                      )
                    )
                  }}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择 provider" />
                  </SelectTrigger>
                  <SelectContent>
                    {STORAGE_PROVIDER_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <div className="text-xs leading-5 text-muted-foreground">
                  {STORAGE_PROVIDER_OPTIONS.find((option) => option.value === sourceProvider)?.description}
                </div>
                <Input value={sourceName} onChange={(event) => setSourceName(event.target.value)} placeholder="家庭媒体" />
                <Input
                  value={sourceRootPath}
                  onChange={(event) => setSourceRootPath(event.target.value)}
                  placeholder="/Users/atlan/Desktop/IdeaProjects/Mibo/demo-media"
                />
                {sourceProvider === 'openlist' ? (
                  <>
                    <Input
                      value={sourceOpenListBaseUrl}
                      onChange={(event) => setSourceOpenListBaseUrl(event.target.value)}
                      placeholder={DEFAULT_OPENLIST_BASE_URL}
                    />
                    <Input
                      value={sourceOpenListUsername}
                      onChange={(event) => setSourceOpenListUsername(event.target.value)}
                      placeholder="OpenList 用户名（可选）"
                    />
                    <Input
                      value={sourceOpenListPassword}
                      onChange={(event) => setSourceOpenListPassword(event.target.value)}
                      type="password"
                      placeholder="OpenList 密码（可选）"
                    />
                  </>
                ) : null}
                <Button type="submit" disabled={isLoading}>
                  {isLoading ? <Loader2 className="size-4 animate-spin" /> : <Sparkles className="size-4" />}
                  创建媒体源
                </Button>
              </form>
            )}

            {currentStep === 3 && (
              <form className="grid gap-3" onSubmit={handleCreateLibrary}>
                <Select
                  value={libraryType}
                  onValueChange={(value) => {
                    const nextType = LIBRARY_TYPE_OPTIONS.find((option) => option.value === value) ?? LIBRARY_TYPE_OPTIONS[0]
                    setLibraryType(value)
                    setLibraryName(nextType.pathSuffix)
                    const selectedSource = mediaSources.find((source) => source.id === mediaSourceId)
                    setLibraryRootPath(
                      buildSuggestedLibraryRootPath(
                        selectedSource?.root_path ?? sourceRootPath,
                        nextType.pathSuffix
                      )
                    )
                  }}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择媒体库类型" />
                  </SelectTrigger>
                  <SelectContent>
                    {LIBRARY_TYPE_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Input value={libraryName} onChange={(event) => setLibraryName(event.target.value)} placeholder="Movies" />
                <Input
                  value={libraryRootPath}
                  onChange={(event) => setLibraryRootPath(event.target.value)}
                  placeholder={buildSuggestedLibraryRootPath(sourceRootPath, selectedLibraryType.pathSuffix)}
                />
                <Button type="submit" disabled={isLoading}>
                  {isLoading ? <Loader2 className="size-4 animate-spin" /> : <ArrowRight className="size-4" />}
                  创建媒体库并完成初始化
                </Button>
              </form>
            )}

            {currentStep === 4 && (
              <div className="space-y-4">
                <div className="rounded-2xl border border-primary/20 bg-primary/5 px-4 py-3 text-sm text-muted-foreground">
                  {setupStatus?.initialized
                    ? '初始化已完成，可以进入新框架下的首页壳。'
                    : '用户已可进入应用，但还可以稍后继续补齐媒体源或媒体库。'}
                </div>
                <Button onClick={handleSkipToApp}>
                  <ArrowRight className="size-4" />
                  进入应用
                </Button>
                <Button asChild variant="outline">
                  <Link to="/">返回迁移首页</Link>
                </Button>
              </div>
            )}
          </CardContent>
          <CardFooter className="flex flex-wrap items-center justify-between gap-3 border-t border-border/60 pt-6">
            <div className="text-sm text-muted-foreground">API Base: {apiBaseUrl}</div>
            {canSkipToApp ? (
              <Button variant="ghost" onClick={handleSkipToApp}>
                稍后继续，先进入应用
              </Button>
            ) : null}
          </CardFooter>
        </Card>
      </div>
    </div>
  )
}
