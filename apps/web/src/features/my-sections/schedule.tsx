import { Link } from '@tanstack/react-router'
import { useEffect, useId, useState, useSyncExternalStore } from 'react'
import type { components } from '../../api/types'
import { Badge } from '../../components/badge'
import { assignLanes, classCode, clashesFor, dayLong, dayShort, toMinutes } from '../../lib/schedule'
import { personName } from '../courses/access'
import { todayIso } from '../courses/queries'
import { useMe } from '../auth/use-me'

type Term = components['schemas']['Term']
type MySection = components['schemas']['MySection']
type Meeting = components['schemas']['Meeting']

const days = [1, 2, 3, 4, 5, 6, 7]
const hh = (h: number) => String(h).padStart(2, '0')
/** Instructor names; the viewer's own entry reads "You". */
export const instructors = (s: MySection, meId?: string) =>
  s.staff.filter((x) => x.role === 'instructor').map((x) => (x.userId === meId ? 'You' : personName(x)))

const wideQuery = '(min-width: 768px)'
function subscribeWide(cb: () => void) {
  const mq = window.matchMedia?.(wideQuery)
  mq?.addEventListener('change', cb)
  return () => mq?.removeEventListener('change', cb)
}
const wideNow = () => window.matchMedia?.(wideQuery).matches ?? true
/** True from 768 px (the week grid); false below (the agenda). Without matchMedia (jsdom) the grid. */
function useWide() {
  return useSyncExternalStore(subscribeWide, wideNow)
}

/** The current time, refreshed each minute. */
function useNow() {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 60_000)
    return () => clearInterval(t)
  }, [])
  return now
}

const byTime = (a: { s: MySection; m: Meeting }, b: { s: MySection; m: Meeting }) =>
  a.m.startsAt.localeCompare(b.m.startsAt) || a.m.endsAt.localeCompare(b.m.endsAt) || classCode(a.s).localeCompare(classCode(b.s))

const linkFocus = 'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus'

/** Phone view: one region per weekday that has meetings, one link per meeting. */
function Agenda({ slug, id, sections }: { slug: string; id: string; sections: MySection[] }) {
  return (
    <div className="flex flex-col gap-6">
      {days.map((d) => {
        const items = sections.flatMap((s) => s.meetings.filter((m) => m.weekday === d).map((m) => ({ s, m }))).sort(byTime)
        if (items.length === 0) return null
        return (
          <section key={d} aria-labelledby={`${id}-a${d}`}>
            <h3 id={`${id}-a${d}`} className="mb-2 text-base font-bold">
              {dayLong[d - 1]}
            </h3>
            <ol className="flex flex-col gap-2">
              {items.map(({ s, m }) => {
                const clash = clashesFor({ id: s.id, meetings: [m] }, sections)
                return (
                  <li key={`${s.id}-${m.id}`}>
                    <Link
                      to="/$slug/sections/$id"
                      params={{ slug, id: s.id }}
                      data-clash={clash.length > 0 || undefined}
                      className={`block rounded p-3 text-sm ${linkFocus} ${
                        clash.length
                          ? 'border-2 border-dashed border-danger bg-danger-soft'
                          : 'border border-line border-l-4 border-l-accent bg-surface hover:bg-canvas'
                      }`}
                    >
                      <span className="block">
                        <span className="font-bold">{classCode(s)}</span>{' '}
                        <span className="whitespace-nowrap text-ink-muted">
                          {m.startsAt}–{m.endsAt}
                        </span>
                      </span>
                      {clash.length > 0 && (
                        <span className="block font-bold text-danger">Clashes with {clash.map(classCode).join(', ')}</span>
                      )}
                      <span className="block">{s.course.title}</span>
                      {m.location !== '' && <span className="block text-ink-muted">{m.location}</span>}
                    </Link>
                  </li>
                )
              })}
            </ol>
          </section>
        )
      })}
    </div>
  )
}

export function Schedule({ slug, term, sections }: { slug: string; term: Term; sections: MySection[] }) {
  const id = useId()
  const wide = useWide()
  const now = useNow()
  const meId = useMe().data?.user.id
  const todayDay = now.getDay() || 7
  const inTerm = term.startsOn <= todayIso(now) && todayIso(now) <= term.endsOn
  const nowMin = now.getHours() * 60 + now.getMinutes()
  const timed = sections.filter((s) => s.meetings.length > 0)
  const untimed = sections.filter((s) => s.meetings.length === 0)
  const all = timed.flatMap((s) => s.meetings)
  const start = Math.min(480, Math.floor(Math.min(...all.map((m) => toMinutes(m.startsAt))) / 60) * 60)
  const end = Math.max(1080, Math.ceil(Math.max(...all.map((m) => toMinutes(m.endsAt))) / 60) * 60)
  const rows = end - start
  const dayItems = Object.fromEntries(
    days.map((d) => [d, timed.flatMap((s) => s.meetings.filter((m) => m.weekday === d).map((m) => ({ s, m }))).sort(byTime)]),
  )
  const hours = Array.from({ length: (end - start) / 60 }, (_, i) => start / 60 + i)
  return (
    <section aria-labelledby={`${id}-h`}>
      <h2 id={`${id}-h`} className="sr-only">
        Weekly schedule, {term.name}
      </h2>
      {timed.length === 0 ? (
        <p className="mb-4 text-sm text-ink-muted">None of your sections in {term.name} has meeting times yet.</p>
      ) : !wide ? (
        <Agenda slug={slug} id={id} sections={timed} />
      ) : (
        <div className="overflow-x-auto rounded border border-line bg-surface">
          <div
            className="grid min-w-[48rem]"
            style={{
              gridTemplateColumns: `3.5rem ${days.map((d) => (d > 5 && !dayItems[d].length ? 'minmax(0,0.5fr)' : 'minmax(0,1fr)')).join(' ')}`,
            }}
          >
            <div aria-hidden="true" className="relative">
              <div className="h-8 border-b border-line bg-canvas" />
              {inTerm && nowMin >= start && nowMin < end && (
                <span
                  data-now-label
                  className="absolute right-0 z-10 -translate-y-1/2 bg-surface pr-2 text-xs text-ink-muted"
                  style={{ top: `${2 + (nowMin - start) * 0.1}rem` }}
                >
                  {hh(now.getHours())}:{hh(now.getMinutes())}
                </span>
              )}
              {hours.map((h) => (
                <div key={h} className="h-24 pr-2 text-right text-xs text-ink-muted">
                  {hh(h)}:00
                </div>
              ))}
            </div>
            {days.map((d) => {
              const items = dayItems[d]
              const today = inTerm && d === todayDay
              const { lane, lanes } = assignLanes(items.map((i) => i.m))
              return (
                <section key={d} aria-labelledby={`${id}-n${d}`} className="border-l border-line">
                  <h3
                    aria-current={today ? 'date' : undefined}
                    className="flex h-8 items-center justify-center gap-1 border-b border-line bg-canvas text-xs font-bold"
                  >
                    <span aria-hidden="true">{dayShort[d - 1]}</span>
                    <span id={`${id}-n${d}`} className="sr-only">
                      {dayLong[d - 1]}
                    </span>
                    {today && <Badge tone="accent">Today</Badge>}
                  </h3>
                  {items.length === 0 && <p className="sr-only">No meetings</p>}
                  <ol
                    className="relative grid"
                    style={{
                      gridTemplateRows: `repeat(${rows}, 0.1rem)`,
                      gridTemplateColumns: `repeat(${lanes}, minmax(0, 1fr))`,
                      backgroundImage: 'repeating-linear-gradient(to bottom, var(--color-line) 0 1px, transparent 1px 6rem)',
                    }}
                  >
                    {today && nowMin >= start && nowMin < end && (
                      <li
                        aria-hidden="true"
                        data-now-line
                        className="pointer-events-none absolute inset-x-0 border-t border-line-strong"
                        style={{ top: `${(nowMin - start) * 0.1}rem` }}
                      ></li>
                    )}
                    {items.map(({ s, m }, i) => {
                      const clash = clashesFor({ id: s.id, meetings: [m] }, sections)
                      const names = instructors(s, meId)
                      return (
                        <li
                          key={`${s.id}-${m.id}`}
                          className="relative min-h-0 p-0.5"
                          style={{
                            gridRow: `${toMinutes(m.startsAt) - start + 1} / ${toMinutes(m.endsAt) - start + 1}`,
                            gridColumn: lane[i] + 1,
                          }}
                        >
                          <Link
                            to="/$slug/sections/$id"
                            params={{ slug, id: s.id }}
                            data-clash={clash.length > 0 || undefined}
                            className={`block h-full overflow-hidden rounded p-1.5 text-xs leading-4 ${linkFocus} ${
                              clash.length
                                ? 'border-2 border-dashed border-danger bg-danger-soft'
                                : 'border border-line border-l-4 border-l-accent bg-surface hover:bg-canvas'
                            }`}
                          >
                            <span className="block">
                              <span className="font-bold">{classCode(s)}</span>{' '}
                              <span className="whitespace-nowrap text-ink-muted">
                                {m.startsAt}–{m.endsAt}
                              </span>
                            </span>
                            {clash.length > 0 && (
                              <span className="block truncate font-bold text-danger">
                                Clashes with {clash.map(classCode).join(', ')}
                              </span>
                            )}
                            <span className="block truncate">{s.course.title}</span>
                            {names.length > 0 && <span className="sr-only"> {names.join(', ')}</span>}
                            {m.location !== '' && <span className="block truncate text-ink-muted">{m.location}</span>}
                          </Link>
                        </li>
                      )
                    })}
                  </ol>
                </section>
              )
            })}
          </div>
        </div>
      )}
      {untimed.length > 0 && (
        <section aria-labelledby={`${id}-n`} className="mt-6">
          <h3 id={`${id}-n`} className="mb-2 text-sm font-bold">
            No meeting times
          </h3>
          <ul className="flex flex-col gap-1 text-sm">
            {untimed.map((s) => {
              const names = instructors(s, meId)
              return (
                <li key={s.id}>
                  <Link to="/$slug/sections/$id" params={{ slug, id: s.id }} className="font-medium text-accent hover:underline">
                    {classCode(s)}
                  </Link>
                  {' · '}
                  {s.course.title}
                  {names.length > 0 && ` · ${names.join(', ')}`}
                </li>
              )
            })}
          </ul>
        </section>
      )}
    </section>
  )
}
