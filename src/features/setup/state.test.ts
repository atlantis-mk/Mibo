import { describe, expect, it } from 'vitest'
import type { SetupDatabaseState, SetupStatus } from '@/lib/mibo-api'
import {
  createSetupDatabaseForm,
  getSetupStage,
  setupDatabaseFormMatchesState,
} from '@/features/setup/state'

const baseStatus: SetupStatus = {
  initialized: false,
  can_enter_app: false,
  has_users: false,
  has_media_sources: false,
  has_libraries: false,
  user_count: 0,
  media_source_count: 0,
  library_count: 0,
}

const baseDatabaseState: SetupDatabaseState = {
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

describe('setup state helpers', () => {
  it('hydrates a form from active database state', () => {
    const form = createSetupDatabaseForm(baseDatabaseState)

    expect(form.driver).toBe('sqlite')
    expect(form.sqlite_path).toBe('data/mibo.db')
    expect(form.ssl_mode).toBe('disable')
  })

  it('detects when the form matches the active runtime database', () => {
    const form = createSetupDatabaseForm(baseDatabaseState)

    expect(setupDatabaseFormMatchesState(form, baseDatabaseState)).toBe(true)
    expect(
      setupDatabaseFormMatchesState(
        { ...form, sqlite_path: 'data/alternate.db' },
        baseDatabaseState
      )
    ).toBe(false)
  })

  it('moves into restart stage while a restart is pending', () => {
    const stage = getSetupStage({
      setupStatus: baseStatus,
      databaseState: { ...baseDatabaseState, restart_required: true },
      hasSession: false,
      waitingForRestart: true,
      formMatchesActive: false,
    })

    expect(stage).toBe('restart')
  })
})
