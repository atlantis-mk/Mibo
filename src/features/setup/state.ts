import type {
  SetupDatabaseInput,
  SetupDatabaseState,
  SetupStatus,
} from '@/lib/mibo-api'

export type SetupStage = 'database' | 'restart' | 'account' | 'done'

export function createSetupDatabaseForm(
  databaseState: SetupDatabaseState
): SetupDatabaseInput {
  const { draft_connection: draftConnection, defaults } = databaseState
  const driver = draftConnection.driver ?? databaseState.active_driver

  return {
    driver,
    sqlite_path: draftConnection.sqlite_path ?? defaults.sqlite_path,
    host: draftConnection.host ?? '',
    port:
      draftConnection.port ??
      (driver === 'mysql' ? defaults.mysql_port : defaults.postgres_port),
    database: draftConnection.database ?? '',
    username: draftConnection.username ?? '',
    password: '',
    ssl_mode: draftConnection.ssl_mode ?? defaults.ssl_mode,
  }
}

export function setupDatabaseFormMatchesState(
  form: SetupDatabaseInput,
  databaseState: SetupDatabaseState
): boolean {
  const active = databaseState.active_connection
  if (form.driver !== databaseState.active_driver) {
    return false
  }

  if (form.driver === 'sqlite') {
    return normalize(form.sqlite_path) === normalize(active.sqlite_path)
  }

  return (
    normalize(form.host) === normalize(active.host) &&
    Number(form.port ?? 0) === Number(active.port ?? 0) &&
    normalize(form.database) === normalize(active.database) &&
    normalize(form.username) === normalize(active.username) &&
    normalize(form.ssl_mode) === normalize(active.ssl_mode)
  )
}

export function getSetupStage(args: {
  setupStatus: SetupStatus
  databaseState: SetupDatabaseState
  hasSession: boolean
  waitingForRestart: boolean
  formMatchesActive: boolean
}): SetupStage {
  const {
    setupStatus,
    databaseState,
    hasSession,
    waitingForRestart,
    formMatchesActive,
  } = args

  if (setupStatus.can_enter_app && hasSession) {
    return 'done'
  }
  if (waitingForRestart || databaseState.restart_required) {
    return 'restart'
  }
  if (!databaseState.edit_locked && !databaseState.initialization_locked && !formMatchesActive) {
    return 'database'
  }
  if (setupStatus.has_users) {
    return 'done'
  }
  return 'account'
}

function normalize(value?: string) {
  return (value ?? '').trim()
}
