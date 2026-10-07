import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { signOut } from '../features/auth/sign-out'
import type { Me, MyMembership } from '../features/auth/use-me'
import { Button } from './button'
import { inputClass } from './form-field'

const navBase =
  'rounded-md px-3 py-1.5 text-sm font-medium text-ink-muted hover:text-ink focus-visible:outline-2 focus-visible:outline-focus'
const navActive = { className: 'bg-accent-soft text-accent', 'aria-current': 'page' as const }

export function AppShell(props: { me: Me; membership: MyMembership; children: React.ReactNode }) {
  const { me, membership, children } = props
  const slug = membership.university.slug
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const isAdmin = membership.role === 'admin' || membership.role === 'dept_admin'

  return (
    <>
      <header className="flex h-14 items-center gap-6 border-b border-line bg-surface px-8">
        <div className="flex items-center gap-2">
          <span className="text-sm font-semibold tracking-tight">UniCourse</span>
          <span aria-hidden="true" className="text-ink-muted">·</span>
          <span className="text-sm text-ink-muted">{membership.university.name}</span>
        </div>
        <nav aria-label="Main" className="flex gap-1">
          <Link to="/$slug/me" params={{ slug }} className={navBase} activeProps={navActive}>
            My sections
          </Link>
          <Link to="/$slug/catalog" params={{ slug }} className={navBase} activeProps={navActive}>
            Catalog
          </Link>
          {isAdmin && (
            <Link to="/$slug/courses" params={{ slug }} className={navBase} activeProps={navActive}>
              Courses
            </Link>
          )}
          {isAdmin && (
            <Link to="/$slug/admin" params={{ slug }} className={navBase} activeProps={navActive}>
              Admin
            </Link>
          )}
        </nav>
        <div className="ml-auto flex items-center gap-3">
          {me.memberships.length > 1 && (
            <select
              aria-label="University"
              className={inputClass + ' w-56'}
              value={slug}
              onChange={(e) => void navigate({ to: '/$slug/me', params: { slug: e.target.value } })}
            >
              {me.memberships.map((m) => (
                <option key={m.university.slug} value={m.university.slug}>
                  {m.university.name}
                </option>
              ))}
            </select>
          )}
          <span className="text-sm text-ink-muted">{me.user.email}</span>
          <Button variant="secondary" size="sm" onClick={() => void signOut(queryClient)}>
            Sign out
          </Button>
        </div>
      </header>
      <main id="main" tabIndex={-1} className="mx-auto max-w-6xl px-8 py-6">
        {children}
      </main>
    </>
  )
}
