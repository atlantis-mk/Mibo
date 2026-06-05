import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type Dispatch,
  type ReactNode,
  type SetStateAction,
} from 'react'
import { flushSync } from 'react-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import {
  ArrowLeft,
  ArrowRight,
  Circle,
  CheckCircle2,
  Database,
  Loader2,
  Lock,
  RefreshCcw,
  ServerCrash,
  ShieldCheck,
  Sparkles,
} from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import {
  createMiboApi,
  getApiBaseUrl,
  type SetupDatabaseInput,
  type SetupDatabaseState,
  type SetupStatus,
} from '@/lib/mibo-api'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import {
  createSetupDatabaseForm,
  setupDatabaseFormMatchesState,
} from '@/features/setup/state'

const setupApi = () => createMiboApi({ baseUrl: getApiBaseUrl() })

type WizardStep = 1 | 2 | 3

export function SetupPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const user = useAuthStore((state) => state.auth.user)
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('admin123')
  const [pin, setPIN] = useState('1234')
  const [selectedDriver, setSelectedDriver] =
    useState<SetupDatabaseInput['driver']>('sqlite')
  const [currentStep, setCurrentStep] = useState<WizardStep>(1)
  const [databaseForm, setDatabaseForm] = useState<SetupDatabaseInput>({
    driver: 'sqlite',
    sqlite_path: 'data/mibo.db',
    host: '',
    port: 5432,
    database: '',
    username: '',
    password: '',
    ssl_mode: 'disable',
  })
  const [databaseFormDirty, setDatabaseFormDirty] = useState(false)
  const [validatedFingerprint, setValidatedFingerprint] = useState<
    string | null
  >(null)
  const [isResolvingAccountStep, setIsResolvingAccountStep] = useState(false)
  const [waitingForRestart, setWaitingForRestart] = useState(false)
  const accountCheckTimeoutRef = useRef<number | null>(null)
  const hydratedFingerprintRef = useRef<string>('')

  const setupStatusQuery = useQuery({
    queryKey: ['setup', 'status'],
    queryFn: () => setupApi().getSetupStatus(),
    refetchInterval: waitingForRestart ? 1500 : 5000,
    refetchOnMount: 'always',
    retry: false,
    staleTime: 0,
  })

  const databaseStateQuery = useQuery({
    queryKey: ['setup', 'database'],
    queryFn: () => setupApi().getSetupDatabaseState(),
    refetchInterval: waitingForRestart ? 1500 : 5000,
    refetchOnMount: 'always',
    retry: false,
    staleTime: 0,
  })

  const setupStatus =
    setupStatusQuery.isFetchedAfterMount && setupStatusQuery.data
      ? setupStatusQuery.data
      : emptySetupStatus
  const databaseState =
    databaseStateQuery.isFetchedAfterMount && databaseStateQuery.data
      ? databaseStateQuery.data
      : emptySetupDatabaseState
  const formMatchesActive = setupDatabaseFormMatchesState(
    databaseForm,
    databaseState
  )
  const beginAccountCheckTransition = useCallback(() => {
    if (accountCheckTimeoutRef.current !== null) {
      window.clearTimeout(accountCheckTimeoutRef.current)
    }
    setCurrentStep(3)
    setIsResolvingAccountStep(true)
    accountCheckTimeoutRef.current = window.setTimeout(() => {
      setIsResolvingAccountStep(false)
      accountCheckTimeoutRef.current = null
    }, 500)
  }, [])
  const resolveAccountStep = useCallback(
    async (options?: { flush?: boolean; refreshDatabase?: boolean }) => {
      if (options?.flush) {
        flushSync(() => {
          beginAccountCheckTransition()
        })
      } else {
        beginAccountCheckTransition()
      }
      try {
        await Promise.all([
          delay(500),
          queryClient.refetchQueries({ queryKey: ['setup', 'status'] }),
          options?.refreshDatabase
            ? queryClient.refetchQueries({ queryKey: ['setup', 'database'] })
            : Promise.resolve(),
        ])
      } finally {
        setIsResolvingAccountStep(false)
      }
    },
    [beginAccountCheckTransition, queryClient]
  )

  useEffect(() => {
    if (!databaseStateQuery.data) return

    const fingerprint = JSON.stringify(databaseStateQuery.data.draft_connection)
    if (hydratedFingerprintRef.current === fingerprint) {
      return
    }

    const nextForm = createSetupDatabaseForm(databaseStateQuery.data)
    hydratedFingerprintRef.current = fingerprint
    setDatabaseForm(nextForm)
    setSelectedDriver(nextForm.driver)
    setDatabaseFormDirty(false)
    setValidatedFingerprint(null)
  }, [databaseStateQuery.data])

  useEffect(() => {
    return () => {
      if (accountCheckTimeoutRef.current !== null) {
        window.clearTimeout(accountCheckTimeoutRef.current)
      }
    }
  }, [])

  useEffect(() => {
    if (
      waitingForRestart &&
      databaseStateQuery.data &&
      !databaseStateQuery.data.restart_required
    ) {
      setWaitingForRestart(false)
      void resolveAccountStep({ refreshDatabase: true })
      toast.success('服务已恢复，当前数据库配置已经生效')
    }
  }, [databaseStateQuery.data, resolveAccountStep, waitingForRestart])

  useEffect(() => {
    if (!setupStatusQuery.isFetchedAfterMount) return

    if (setupStatus.has_users && currentStep !== 3) {
      beginAccountCheckTransition()
    }
  }, [
    currentStep,
    setupStatus.has_users,
    setupStatusQuery.isFetchedAfterMount,
    beginAccountCheckTransition,
  ])

  useEffect(() => {
    if (setupStatus.can_enter_app && accessToken && user) {
      void navigate({ to: '/', replace: true })
    }
  }, [accessToken, navigate, setupStatus.can_enter_app, user])

  const validateMutation = useMutation({
    mutationFn: () => setupApi().validateSetupDatabase(databaseForm),
    onSuccess: (result) => {
      setValidatedFingerprint(getDatabaseFormFingerprint(databaseForm))
      toast.success(result.message)
    },
    onError: (error) => {
      setValidatedFingerprint(null)
      toast.error(getErrorMessage(error))
    },
  })

  const draftMutation = useMutation({
    mutationFn: (input: SetupDatabaseInput) =>
      setupApi().persistSetupDatabaseDraft(input),
    onSuccess: async (result) => {
      toast.success(result.message)
      await queryClient.invalidateQueries({ queryKey: ['setup', 'database'] })
      setDatabaseForm((current) => ({
        ...current,
        driver: selectedDriver,
        port:
          selectedDriver === 'mysql'
            ? databaseState.defaults.mysql_port
            : databaseState.defaults.postgres_port,
      }))
      setValidatedFingerprint(null)
      setCurrentStep(2)
    },
    onError: (error) => {
      toast.error(getErrorMessage(error))
    },
  })

  const applyMutation = useMutation({
    mutationFn: () => setupApi().applySetupDatabase(databaseForm),
    onSuccess: async (result) => {
      toast.success(result.message)
      setDatabaseFormDirty(false)
      setValidatedFingerprint(null)

      if (result.restart_required) {
        setWaitingForRestart(true)
        return
      }

      setIsResolvingAccountStep(true)
      try {
        await resolveAccountStep({ refreshDatabase: true })
      } finally {
        setIsResolvingAccountStep(false)
      }
    },
    onError: (error) => {
      setIsResolvingAccountStep(false)
      toast.error(getErrorMessage(error))
    },
  })

  const registerMutation = useMutation({
    mutationFn: () => setupApi().registerSetupAdmin(username, password, pin),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['setup', 'status'] })
      toast.success('账号已创建，请登录')
      void navigate({
        to: '/sign-in',
        search: { redirect: undefined },
        replace: true,
      })
    },
    onError: (error) => {
      toast.error(getErrorMessage(error))
    },
  })

  const canEditDatabase =
    !databaseState.edit_locked &&
    !databaseState.initialization_locked &&
    !waitingForRestart
  const formFingerprint = getDatabaseFormFingerprint(databaseForm)
  const isValidationPassed =
    validatedFingerprint !== null && validatedFingerprint === formFingerprint
  const draftMatchesSelection =
    selectedDriver === databaseState.draft_connection.driver
  const isRefreshingSetupStatus =
    setupStatusQuery.isFetching && !setupStatusQuery.isFetchedAfterMount
  const isCheckingAccounts =
    currentStep === 3 &&
    (isResolvingAccountStep ||
      !formMatchesActive ||
      setupStatusQuery.isPending ||
      isRefreshingSetupStatus)
  const canContinueWithCurrentConfig =
    formMatchesActive && !databaseState.restart_required && !databaseFormDirty
  const canSaveAndContinue =
    canContinueWithCurrentConfig || (canEditDatabase && isValidationPassed)

  const stepOneDone = currentStep > 1
  const stepTwoDone = currentStep > 2 && !waitingForRestart

  return (
    <div className='min-h-svh bg-[radial-gradient(circle_at_top_left,_rgba(34,197,94,0.12),_transparent_38%),linear-gradient(180deg,_hsl(var(--background))_0%,_hsl(var(--muted)/0.4)_100%)] px-4 py-8 text-foreground md:px-8'>
      <div className='mx-auto flex w-full max-w-6xl flex-col gap-6'>
        <header className='space-y-3'>
          <div className='inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-sm text-emerald-700 dark:text-emerald-300'>
            <Sparkles className='size-4' />
            首次初始化
          </div>
          <h1 className='text-3xl font-semibold tracking-tight'>
            三步完成 Mibo 初始化
          </h1>
          <p className='max-w-3xl text-sm leading-7 text-muted-foreground'>
            先选择数据库，再填写连接信息，最后创建账号或直接去登录。
          </p>
        </header>

        <div className='grid gap-4 xl:grid-cols-[300px_minmax(0,1fr)]'>
          <Card className='border-emerald-500/15 bg-card/85 backdrop-blur'>
            <CardHeader>
              <CardTitle className='text-base'>初始化进度</CardTitle>
              <CardDescription>
                只有完成上一步，才能进入下一步。
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-4 text-sm'>
              <StepRow
                index='1'
                label='选择数据库'
                current={currentStep === 1}
                done={stepOneDone}
              />
              <StepRow
                index='2'
                label='填写数据库信息'
                current={currentStep === 2}
                done={stepTwoDone}
              />
              <StepRow
                index='3'
                label='创建账号'
                current={currentStep === 3}
                done={setupStatus.has_users}
              />
              <Separator />
              <StatusRow
                label='当前数据库'
                value={databaseState.active_driver}
              />
              <StatusRow
                label='数据库来源'
                value={databaseSourceLabel(databaseState.active_source)}
              />
              <StatusRow
                label='应用可进入'
                value={setupStatus.can_enter_app ? '是' : '否'}
              />
            </CardContent>
          </Card>

          <div className='space-y-6'>
            {(databaseState.edit_locked ||
              databaseState.initialization_locked) && (
              <Alert>
                <Lock className='size-4' />
                <AlertTitle>数据库配置当前为只读</AlertTitle>
                <AlertDescription>
                  {databaseState.edit_lock_reason ??
                    databaseState.initialization_lock_reason}
                </AlertDescription>
              </Alert>
            )}

            {currentStep === 1 ? (
              <Card className='border-emerald-500/15 bg-card/90 backdrop-blur'>
                <CardHeader>
                  <CardTitle className='flex items-center gap-2 text-base'>
                    <Database className='size-4' />
                    第一步：选择数据库
                  </CardTitle>
                  <CardDescription>
                    先确认本次初始化要使用哪一种数据库。
                  </CardDescription>
                </CardHeader>
                <CardContent className='space-y-6'>
                  <DriverPicker
                    value={selectedDriver}
                    disabled={!canEditDatabase}
                    onValueChange={setSelectedDriver}
                  />

                  <Alert>
                    <ShieldCheck className='size-4' />
                    <AlertTitle>下一步将填写数据库信息</AlertTitle>
                    <AlertDescription>
                      SQLite 会填写数据文件路径，Postgres 和 MySQL
                      会填写连接地址、端口和账号信息。第二步会直接按这里选择的数据库类型进行测试。
                    </AlertDescription>
                  </Alert>

                  <div className='flex justify-end'>
                    <Button
                      disabled={waitingForRestart || draftMutation.isPending}
                      onClick={() => {
                        if (!canEditDatabase || draftMatchesSelection) {
                          setDatabaseForm((current) => ({
                            ...current,
                            driver: selectedDriver,
                            port:
                              selectedDriver === 'mysql'
                                ? databaseState.defaults.mysql_port
                                : databaseState.defaults.postgres_port,
                          }))
                          setCurrentStep(2)
                          return
                        }

                        draftMutation.mutate({
                          driver: selectedDriver,
                          sqlite_path:
                            selectedDriver === 'sqlite'
                              ? databaseState.defaults.sqlite_path
                              : undefined,
                          port:
                            selectedDriver === 'mysql'
                              ? databaseState.defaults.mysql_port
                              : databaseState.defaults.postgres_port,
                          ssl_mode: databaseState.defaults.ssl_mode,
                        })
                      }}
                    >
                      {draftMutation.isPending ? (
                        <Loader2 className='size-4 animate-spin' />
                      ) : null}
                      下一步
                      {!draftMutation.isPending ? (
                        <ArrowRight className='size-4' />
                      ) : null}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ) : null}

            {currentStep === 2 ? (
              <Card className='border-emerald-500/15 bg-card/90 backdrop-blur'>
                <CardHeader>
                  <CardTitle className='flex items-center gap-2 text-base'>
                    <RefreshCcw className='size-4' />
                    第二步：填写数据库信息
                  </CardTitle>
                  <CardDescription>
                    填写连接参数后可直接测试，保存后会自动处理需要的切换流程。
                  </CardDescription>
                </CardHeader>
                <CardContent className='space-y-6'>
                  <div className='flex items-center gap-2 rounded-lg border border-emerald-500/20 bg-emerald-500/5 px-3 py-2 text-sm'>
                    <Circle
                      className={[
                        'size-3 fill-current',
                        isValidationPassed
                          ? 'text-emerald-600'
                          : 'text-muted-foreground/40',
                      ].join(' ')}
                    />
                    <span
                      className={
                        isValidationPassed
                          ? 'text-emerald-700 dark:text-emerald-300'
                          : 'text-muted-foreground'
                      }
                    >
                      {isValidationPassed
                        ? '当前填写内容已测试通过'
                        : '请先测试连接，测试通过后才能保存并继续'}
                    </span>
                  </div>

                  {databaseForm.driver === 'sqlite' ? (
                    <FieldBlock
                      label='SQLite 数据文件'
                      hint='适合单机首次体验，也适合作为默认引导选项。'
                    >
                      <Input
                        value={databaseForm.sqlite_path ?? ''}
                        disabled={!canEditDatabase}
                        onChange={(event) =>
                          updateDatabaseForm(
                            setDatabaseForm,
                            setDatabaseFormDirty,
                            'sqlite_path',
                            event.target.value
                          )
                        }
                      />
                    </FieldBlock>
                  ) : (
                    <div className='grid gap-4 md:grid-cols-2'>
                      <FieldBlock label='主机地址'>
                        <Input
                          value={databaseForm.host ?? ''}
                          disabled={!canEditDatabase}
                          onChange={(event) =>
                            updateDatabaseForm(
                              setDatabaseForm,
                              setDatabaseFormDirty,
                              'host',
                              event.target.value
                            )
                          }
                        />
                      </FieldBlock>
                      <FieldBlock label='端口'>
                        <Input
                          value={String(databaseForm.port ?? '')}
                          disabled={!canEditDatabase}
                          onChange={(event) =>
                            updateDatabaseForm(
                              setDatabaseForm,
                              setDatabaseFormDirty,
                              'port',
                              Number(event.target.value) || 0
                            )
                          }
                        />
                      </FieldBlock>
                      <FieldBlock label='数据库名'>
                        <Input
                          value={databaseForm.database ?? ''}
                          disabled={!canEditDatabase}
                          onChange={(event) =>
                            updateDatabaseForm(
                              setDatabaseForm,
                              setDatabaseFormDirty,
                              'database',
                              event.target.value
                            )
                          }
                        />
                      </FieldBlock>
                      <FieldBlock label='用户名'>
                        <Input
                          value={databaseForm.username ?? ''}
                          disabled={!canEditDatabase}
                          onChange={(event) =>
                            updateDatabaseForm(
                              setDatabaseForm,
                              setDatabaseFormDirty,
                              'username',
                              event.target.value
                            )
                          }
                        />
                      </FieldBlock>
                      <FieldBlock label='密码'>
                        <Input
                          type='password'
                          value={databaseForm.password ?? ''}
                          disabled={!canEditDatabase}
                          onChange={(event) =>
                            updateDatabaseForm(
                              setDatabaseForm,
                              setDatabaseFormDirty,
                              'password',
                              event.target.value
                            )
                          }
                        />
                      </FieldBlock>
                      <FieldBlock label='连接安全'>
                        <Select
                          value={databaseForm.ssl_mode ?? 'disable'}
                          disabled={!canEditDatabase}
                          onValueChange={(value) =>
                            updateDatabaseForm(
                              setDatabaseForm,
                              setDatabaseFormDirty,
                              'ssl_mode',
                              value
                            )
                          }
                        >
                          <SelectTrigger className='w-full'>
                            <SelectValue placeholder='选择安全模式' />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value='disable'>不启用</SelectItem>
                            <SelectItem value='require'>要求加密</SelectItem>
                            <SelectItem value='preferred'>优先加密</SelectItem>
                          </SelectContent>
                        </Select>
                      </FieldBlock>
                    </div>
                  )}

                  {waitingForRestart || databaseState.restart_required ? (
                    <Alert>
                      <Loader2 className='size-4 animate-spin' />
                      <AlertTitle>等待服务恢复</AlertTitle>
                      <AlertDescription>
                        数据库配置已保存，服务正在后台完成切换。恢复后会自动进入下一步。
                      </AlertDescription>
                    </Alert>
                  ) : null}

                  {databaseFormDirty &&
                  !formMatchesActive &&
                  !waitingForRestart ? (
                    <Alert>
                      <ServerCrash className='size-4' />
                      <AlertTitle>数据库配置尚未生效</AlertTitle>
                      <AlertDescription>
                        你已经修改了数据库信息。可以先测试连接，再直接保存继续。
                      </AlertDescription>
                    </Alert>
                  ) : null}

                  <div className='flex flex-wrap items-center justify-between gap-3'>
                    <Button
                      variant='ghost'
                      disabled={waitingForRestart}
                      onClick={() => setCurrentStep(1)}
                    >
                      <ArrowLeft className='size-4' />
                      上一步
                    </Button>

                    <div className='flex flex-wrap items-center gap-3'>
                      <Button
                        variant='outline'
                        disabled={
                          !canEditDatabase || validateMutation.isPending
                        }
                        onClick={() => validateMutation.mutate()}
                      >
                        {validateMutation.isPending ? (
                          <Loader2 className='size-4 animate-spin' />
                        ) : (
                          <CheckCircle2 className='size-4' />
                        )}
                        测试连接
                      </Button>
                      <Button
                        disabled={
                          applyMutation.isPending || !canSaveAndContinue
                        }
                        onClick={() => {
                          if (canContinueWithCurrentConfig) {
                            void resolveAccountStep({ flush: true })
                            return
                          }
                          applyMutation.mutate()
                        }}
                      >
                        {applyMutation.isPending ? (
                          <Loader2 className='size-4 animate-spin' />
                        ) : formMatchesActive ? (
                          <ArrowRight className='size-4' />
                        ) : (
                          <RefreshCcw className='size-4' />
                        )}
                        下一步
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ) : null}

            {currentStep === 3 ? (
              <Card className='border-emerald-500/15 bg-card/90 backdrop-blur'>
                <CardHeader>
                  <CardTitle className='flex items-center gap-2 text-base'>
                    <ShieldCheck className='size-4' />
                    第三步：创建账号
                  </CardTitle>
                  <CardDescription>
                    可以创建首个账号，也可以直接去登录已有账号。
                  </CardDescription>
                </CardHeader>
                <CardContent className='space-y-4'>
                  {isCheckingAccounts ? (
                    <div className='flex min-h-56 flex-col items-center justify-center gap-4 rounded-2xl border border-emerald-500/20 bg-emerald-500/5 px-6 py-10 text-center'>
                      <div className='flex size-12 items-center justify-center rounded-full bg-emerald-500/10 text-emerald-600'>
                        <Loader2 className='size-6 animate-spin' />
                      </div>
                      <div className='space-y-1'>
                        <p className='font-medium text-foreground'>
                          正在检测管理员账户
                        </p>
                        <p className='text-sm leading-6 text-muted-foreground'>
                          正在读取当前数据库中的账号状态，请稍候，完成前暂时不能操作。
                        </p>
                      </div>
                    </div>
                  ) : setupStatus.has_users ? (
                    <Alert>
                      <CheckCircle2 className='size-4' />
                      <AlertTitle>已检测到现有账号</AlertTitle>
                      <AlertDescription>
                        当前数据库里已经存在账号，可以直接进入登录页面。
                      </AlertDescription>
                    </Alert>
                  ) : (
                    <>
                      <div className='grid gap-4 md:grid-cols-3'>
                        <FieldBlock label='用户名'>
                          <Input
                            value={username}
                            onChange={(event) =>
                              setUsername(event.target.value)
                            }
                            disabled={registerMutation.isPending}
                          />
                        </FieldBlock>
                        <FieldBlock label='密码'>
                          <Input
                            value={password}
                            type='password'
                            onChange={(event) =>
                              setPassword(event.target.value)
                            }
                            disabled={registerMutation.isPending}
                          />
                        </FieldBlock>
                        <FieldBlock label='PIN'>
                          <Input
                            value={pin}
                            inputMode='numeric'
                            maxLength={4}
                            type='password'
                            onChange={(event) => setPIN(event.target.value)}
                            disabled={registerMutation.isPending}
                          />
                        </FieldBlock>
                      </div>
                      <p className='text-sm leading-6 text-muted-foreground'>
                        创建完成后会进入登录页面，不再自动登录。
                      </p>
                    </>
                  )}

                  <div className='flex flex-wrap items-center justify-between gap-3'>
                    <Button
                      variant='ghost'
                      onClick={() => setCurrentStep(2)}
                      disabled={
                        registerMutation.isPending ||
                        waitingForRestart ||
                        isCheckingAccounts
                      }
                    >
                      <ArrowLeft className='size-4' />
                      上一步
                    </Button>

                    <div className='flex flex-wrap items-center gap-3'>
                      {!setupStatus.has_users ? (
                        <Button
                          onClick={() => registerMutation.mutate()}
                          disabled={
                            registerMutation.isPending || isCheckingAccounts
                          }
                        >
                          {registerMutation.isPending ? (
                            <Loader2 className='size-4 animate-spin' />
                          ) : (
                            <ShieldCheck className='size-4' />
                          )}
                          创建账号
                        </Button>
                      ) : null}
                      <Button
                        variant={setupStatus.has_users ? 'default' : 'outline'}
                        disabled={isCheckingAccounts}
                        onClick={() =>
                          void navigate({
                            to: '/sign-in',
                            search: { redirect: undefined },
                            replace: true,
                          })
                        }
                      >
                        已有账号，去登录
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  )
}

const emptySetupStatus: SetupStatus = {
  initialized: false,
  can_enter_app: false,
  has_users: false,
  has_media_sources: false,
  has_libraries: false,
  user_count: 0,
  media_source_count: 0,
  library_count: 0,
}

const emptySetupDatabaseState: SetupDatabaseState = {
  active_driver: 'sqlite',
  active_source: 'default',
  active_connection: {
    driver: 'sqlite',
    sqlite_path: 'data/mibo.db',
    password_configured: false,
  },
  draft_connection: {
    driver: 'sqlite',
    sqlite_path: 'data/mibo.db',
    password_configured: false,
  },
  defaults: {
    sqlite_path: 'data/mibo.db',
    postgres_port: 5432,
    mysql_port: 3306,
    ssl_mode: 'disable',
  },
  edit_locked: false,
  initialization_locked: false,
  restart_required: false,
}

function DriverPicker(props: {
  value: SetupDatabaseInput['driver']
  disabled: boolean
  onValueChange: (driver: SetupDatabaseInput['driver']) => void
}) {
  return (
    <RadioGroup
      value={props.value}
      onValueChange={(value) =>
        props.onValueChange(value as SetupDatabaseInput['driver'])
      }
      className='grid gap-3 md:grid-cols-3'
    >
      {[
        {
          id: 'sqlite',
          title: 'SQLite',
          description: '默认即开即用，适合单机部署。',
        },
        {
          id: 'postgres',
          title: 'Postgres',
          description: '适合托管数据库和多环境部署。',
        },
        {
          id: 'mysql',
          title: 'MySQL',
          description: '适合已有 MySQL 基础设施的场景。',
        },
      ].map((driver) => (
        <Label
          key={driver.id}
          className='flex cursor-pointer flex-col gap-3 rounded-xl border border-border/70 bg-muted/30 p-4'
        >
          <div className='flex items-center gap-3'>
            <RadioGroupItem
              value={driver.id}
              disabled={props.disabled}
              id={driver.id}
            />
            <span className='font-medium'>{driver.title}</span>
          </div>
          <span className='text-sm leading-6 text-muted-foreground'>
            {driver.description}
          </span>
        </Label>
      ))}
    </RadioGroup>
  )
}

function FieldBlock(props: {
  label: string
  hint?: string
  children: ReactNode
}) {
  return (
    <div className='space-y-2'>
      <Label>{props.label}</Label>
      {props.children}
      {props.hint ? (
        <p className='text-xs leading-5 text-muted-foreground'>{props.hint}</p>
      ) : null}
    </div>
  )
}

function StepRow(props: {
  index: string
  label: string
  current: boolean
  done: boolean
}) {
  return (
    <div className='flex items-center justify-between gap-3'>
      <div className='flex items-center gap-3'>
        <div
          className={[
            'flex size-7 items-center justify-center rounded-full border text-xs font-semibold',
            props.done
              ? 'border-emerald-500/40 bg-emerald-500/15 text-emerald-700 dark:text-emerald-300'
              : props.current
                ? 'border-primary bg-primary/10 text-primary'
                : 'text-muted-foreground',
          ].join(' ')}
        >
          {props.index}
        </div>
        <span>{props.label}</span>
      </div>
      <span className='text-xs text-muted-foreground'>
        {props.done ? '已完成' : props.current ? '进行中' : '等待中'}
      </span>
    </div>
  )
}

function StatusRow(props: { label: string; value: string }) {
  return (
    <div className='flex items-center justify-between gap-3 text-sm'>
      <span className='text-muted-foreground'>{props.label}</span>
      <span className='font-medium'>{props.value}</span>
    </div>
  )
}

function databaseSourceLabel(source: SetupDatabaseState['active_source']) {
  switch (source) {
    case 'environment':
      return '环境变量'
    case 'bootstrap_file':
      return '引导配置文件'
    default:
      return '内置默认值'
  }
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : '请求失败，请稍后重试'
}

function delay(durationMs: number) {
  return new Promise((resolve) => window.setTimeout(resolve, durationMs))
}

function updateDatabaseForm(
  setDatabaseForm: Dispatch<SetStateAction<SetupDatabaseInput>>,
  setDatabaseFormDirty: Dispatch<SetStateAction<boolean>>,
  key: keyof SetupDatabaseInput,
  value: string | number
) {
  setDatabaseFormDirty(true)
  setDatabaseForm((current) => ({
    ...current,
    [key]: value,
  }))
}

function getDatabaseFormFingerprint(input: SetupDatabaseInput) {
  return JSON.stringify({
    driver: input.driver,
    sqlite_path: input.sqlite_path ?? '',
    host: input.host ?? '',
    port: input.port ?? 0,
    database: input.database ?? '',
    username: input.username ?? '',
    password: input.password ?? '',
    ssl_mode: input.ssl_mode ?? '',
  })
}
