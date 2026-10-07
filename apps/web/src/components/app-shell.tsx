import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { BookOpen, CalendarDays, Library, LogOut, Settings } from 'lucide-react'
import { signOut } from '../features/auth/sign-out'
import type { Me, MyMembership } from '../features/auth/use-me'
import { Button } from './button'
import { inputClass } from './form-field'
import { ThemeSwitch } from './theme-switch'

const navLink = {
  className:
    'inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus',
  activeProps: {
    className: 'bg-accent-soft font-semibold text-accent',
    'aria-current': 'page' as const,
  },
  inactiveProps: {
    className: 'font-medium text-ink-muted hover:bg-canvas hover:text-ink',
  },
}

export function AppShell(props: { me: Me; membership: MyMembership; children: React.ReactNode }) {
  const { me, membership, children } = props
  const slug = membership.university.slug
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const isAdmin = membership.role === 'admin' || membership.role === 'dept_admin'

  return (
    <>
      <header className="border-b border-line bg-surface">
        <div className="mx-auto flex min-h-14 max-w-6xl flex-wrap items-center gap-x-4 gap-y-2 lg:flex-nowrap px-8 py-2">
          <div className="flex min-w-0 items-center gap-3">
            <span className="font-bold text-2xl leading-none tracking-tight text-ink">UniCourse</span>
            <span aria-hidden="true" className="h-5 w-px bg-line" />
            <span className="min-w-0 max-w-48 truncate text-sm text-ink-muted" title={membership.university.name}>
              {membership.university.name}
            </span>
          </div>
          <nav aria-label="Main" className="flex shrink-0 flex-wrap gap-1">
            <Link to="/$slug/me" params={{ slug }} {...navLink}>
              <CalendarDays size={20} />
              My sections
            </Link>
            <Link to="/$slug/catalog" params={{ slug }} {...navLink}>
              <Library size={20} />
              Catalog
            </Link>
            {isAdmin && (
              <Link to="/$slug/courses" params={{ slug }} {...navLink}>
                <BookOpen size={20} />
                Courses
              </Link>
            )}
            {isAdmin && (
              <Link to="/$slug/admin" params={{ slug }} {...navLink}>
                <Settings size={20} />
                Admin
              </Link>
            )}
          </nav>
          <div className="ml-auto flex shrink-0 items-center gap-3">
            {me.memberships.length > 1 && (
              <select
                aria-label="University"
                className={inputClass + ' w-40 min-w-0 xl:w-56'}
                value={slug}
                onChange={(e) =>
                  void navigate({
                    to: '/$slug/me',
                    params: { slug: e.target.value },
                  })
                }
              >
                {me.memberships.map((m) => (
                  <option key={m.university.slug} value={m.university.slug}>
                    {m.university.name}
                  </option>
                ))}
              </select>
            )}
            <ThemeSwitch />
            <span className="hidden min-w-0 max-w-56 truncate text-sm text-ink-muted xl:inline" title={me.user.email}>
              {me.user.email}
            </span>
            <Button variant="secondary" size="sm" onClick={() => void signOut(queryClient)}>
              <LogOut />
              Sign out
            </Button>
          </div>
        </div>
      </header>
      <main id="main" tabIndex={-1} className="mx-auto max-w-6xl px-8 py-8">
        {children}
      </main>
    </>
  )
}
