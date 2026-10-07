import { useQueryClient } from '@tanstack/react-query'
import { LogOut } from 'lucide-react'
import { Link, Outlet, createFileRoute } from '@tanstack/react-router'
import { EntryLayout } from '../../components/entry-layout'
import { AppShell } from '../../components/app-shell'
import { Button } from '../../components/button'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { signOut } from '../../features/auth/sign-out'
import { useMe } from '../../features/auth/use-me'

function UniversityLayout() {
  const { slug } = Route.useParams()
  const me = useMe()
  const queryClient = useQueryClient()
  if (me.isPending)
    return (
      <div className="mx-auto max-w-336 px-6 py-8">
        <Loading />
      </div>
    )
  if (me.isError) {
    if (me.error.status === 401)
      return (
        <div className="mx-auto max-w-336 px-6 py-8">
          <Loading />
        </div>
      )
    return (
      <main id="main" tabIndex={-1} className="mx-auto max-w-336 px-6 py-8">
        <ErrorBanner onRetry={() => void me.refetch()}>{errorMessage(me.error)}</ErrorBanner>
      </main>
    )
  }
  const membership = me.data.memberships.find((m) => m.university.slug === slug)
  if (!membership) {
    return (
      <EntryLayout>
        <h1 className="text-2xl text-ink">You don't have access to this university</h1>
        <p className="mt-2 text-sm text-ink-muted">
          Your account ({me.data.user.email}) isn't a member of “{slug}”.
        </p>
        <ul className="mt-4 flex flex-col gap-1 text-sm">
          {me.data.memberships.map((m) => (
            <li key={m.university.slug}>
              <Link
                to="/$slug/me"
                params={{ slug: m.university.slug }}
                className="text-accent underline hover:text-accent-hover"
              >
                {m.university.name}
              </Link>
            </li>
          ))}
        </ul>
        <Button variant="secondary" className="mt-6" onClick={() => void signOut(queryClient)}>
          <LogOut size={16} aria-hidden="true" />
          Sign out
        </Button>
      </EntryLayout>
    )
  }
  return (
    <AppShell me={me.data} membership={membership}>
      <Outlet />
    </AppShell>
  )
}

export const Route = createFileRoute('/$slug')({ component: UniversityLayout })
