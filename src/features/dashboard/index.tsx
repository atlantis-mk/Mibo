import { APP_NAME, APP_TAGLINE } from '@/config/app-shell'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Main } from '@/components/layout/main'

export function Dashboard() {
  return (
    <Main>
      <div className='space-y-4'>
        <div>
          <p className='text-sm font-medium'>{APP_NAME}</p>
          <p className='text-xs text-muted-foreground'>{APP_TAGLINE}</p>
        </div>

        <div>
          <h1 className='text-2xl font-bold tracking-tight'>Home</h1>
          <p className='text-sm text-muted-foreground'>
            The demo content and navigation have been removed. This shell is
            ready for product pages.
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Start Building</CardTitle>
          </CardHeader>
          <CardContent className='text-sm text-muted-foreground'>
            Use the existing routing, sidebar, theme, and layout primitives to
            add real application screens.
          </CardContent>
        </Card>
      </div>
    </Main>
  )
}
