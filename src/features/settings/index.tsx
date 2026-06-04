import { Outlet } from '@tanstack/react-router'

export default function SettingsLayout() {
  return (
    <div className='relative flex h-screen max-h-screen min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background text-foreground'>
      <div className='flex min-h-0 flex-1 flex-col overflow-hidden'>
        <main className='flex min-h-0 min-w-0 flex-1 flex-col'>
          <Outlet />
        </main>
      </div>
    </div>
  )
}
