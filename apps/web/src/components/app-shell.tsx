import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import { LogOut, Search } from 'lucide-react'
import { useEffect, useRef } from 'react'
import { signOut } from '../features/auth/sign-out'
import type { Me, MyMembership } from '../features/auth/use-me'
import { Button } from './button'
import { inputClass } from './form-field'
import { ThemeSwitch } from './theme-switch'

const navBase =
  'inline-flex min-h-11 items-center text-sm decoration-2 underline-offset-8 transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus md:min-h-10'

// Current-item rules (S5-05): Courses also covers section detail, Admin also covers the grade scale.
function NavLink(props: { to: string; slug: string; match: RegExp; children: React.ReactNode }) {
  const path = useRouterState({ select: (s) => s.location.pathname })
  const current = props.match.test(path)
  return (
    <Link
      to={props.to}
      params={{ slug: props.slug }}
      aria-current={current ? 'page' : undefined}
      className={`${navBase} ${current ? 'font-bold text-ink underline decoration-accent' : 'font-medium text-ink-muted hover:text-accent-hover'}`}
    >
      {props.children}
    </Link>
  )
}

function HeaderSearch({ slug }: { slug: string }) {
  const navigate = useNavigate()
  const input = useRef<HTMLInputElement>(null)
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement | null
      if (e.key !== '/' || e.metaKey || e.ctrlKey || e.altKey) return
      if (t && (t.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName))) return
      e.preventDefault()
      input.current?.focus()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])
  return (
    <form
      role="search"
      aria-label="Catalog"
      className="relative order-4 col-span-2 w-full max-w-xl min-w-0 md:order-none md:col-span-1 md:mx-auto"
      onSubmit={(e) => {
        e.preventDefault()
        const q = String(new FormData(e.currentTarget).get('q') ?? '').trim()
        void navigate({
          to: '/$slug/catalog',
          params: { slug },
          search: { q: q || undefined },
        })
      }}
    >
      <Search className="pointer-events-none absolute top-1/2 left-4 -translate-y-1/2 text-ink-muted" />
      <input
        ref={input}
        aria-keyshortcuts="/"
        title="Press / to search"
        name="q"
        type="search"
        maxLength={100}
        aria-label="Find a course"
        placeholder="Find a course"
        className="h-10 w-full rounded-full border border-line-strong bg-canvas pr-4 pl-11 text-sm text-ink placeholder:text-ink-muted transition-colors hover:border-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
      />
    </form>
  )
}

export function AppShell(props: { me: Me; membership: MyMembership; children: React.ReactNode }) {
  const { me, membership, children } = props
  const slug = membership.university.slug
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const name = membership.university.name
  const isAdmin = membership.role === 'admin' || membership.role === 'dept_admin'

  return (
    <>
      <header className="border-b border-line bg-surface">
        <div className="mx-auto grid min-h-18 max-w-336 grid-cols-2 items-center gap-x-6 gap-y-3 px-6 py-3 md:grid-cols-[auto_1fr_auto] md:py-0">
          <div className="contents md:flex md:min-w-0 md:items-center md:gap-6">
            <div className="order-1 flex min-w-0 items-center gap-3">
              <span className="text-2xl leading-none font-bold tracking-tight text-ink">UniCourse</span>
              {me.memberships.length === 1 && (
                <>
                  <span aria-hidden="true" className="h-6 w-px bg-line" />
                  <span className="min-w-0 max-w-48 truncate text-sm text-ink-muted" title={name}>
                    {name}
                  </span>
                </>
              )}
            </div>
            <nav
              aria-label="Main"
              className="order-3 col-span-2 flex flex-wrap items-center gap-x-4 md:col-span-1 md:shrink-0"
            >
              <NavLink to="/$slug/me" slug={slug} match={/\/me\/?$/}>
                My sections
              </NavLink>
              <NavLink to="/$slug/catalog" slug={slug} match={/\/catalog\/?$/}>
                Catalog
              </NavLink>
              {isAdmin && (
                <NavLink to="/$slug/courses" slug={slug} match={/\/(courses|sections\/[^/]+\/?$)/}>
                  Courses
                </NavLink>
              )}
              {isAdmin && (
                <NavLink to="/$slug/admin" slug={slug} match={/\/(admin|grade-scale)(\/|$)/}>
                  Admin
                </NavLink>
              )}
            </nav>
          </div>
          <HeaderSearch slug={slug} />
          <div className="order-2 flex shrink-0 items-center justify-self-end gap-3">
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
            <Button
              variant="secondary"
              title={`Signed in as ${me.user.email}`}
              onClick={() => void signOut(queryClient)}
            >
              <LogOut />
              Sign out
            </Button>
          </div>
        </div>
      </header>
      <main id="main" tabIndex={-1} className="mx-auto max-w-336 px-6 py-8">
        {children}
      </main>
    </>
  )
}
