import { redirect } from '@tanstack/react-router'
import { getAppSession, isAppSessionUnavailable } from '@/lib/app-session'

export async function redirectIfSetupComplete() {
  const { setup } = await loadSetupState()

  if (setup.can_enter_app) {
    throw redirect({ to: '/sign-in', search: { redirect: undefined } })
  }
}

export async function requireSetupComplete() {
  const { setup } = await loadSetupState()

  if (!setup.can_enter_app) {
    throw redirect({ to: '/setup' })
  }
}

export async function redirectIfSetupMissing() {
  const { setup } = await loadSetupState()

  if (!setup.can_enter_app) {
    throw redirect({ to: '/setup' })
  }
}

async function loadSetupState() {
  try {
    return await getAppSession()
  } catch (error) {
    if (isAppSessionUnavailable(error)) {
      throw redirect({ to: '/503' })
    }

    throw error
  }
}
