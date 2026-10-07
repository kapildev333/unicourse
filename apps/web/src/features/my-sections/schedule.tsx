import { Link } from '@tanstack/react-router'
import { useId } from 'react'
import type { components } from '../../api/types'
import { assignLanes, classCode, clashesFor, dayLong, dayShort, toMinutes } from '../../lib/schedule'
import { personName } from '../courses/access'

type Term = components['schemas']['Term']
type MySection = components['schemas']['MySection']

const days = [1, 2, 3, 4, 5, 6, 7]
const hh = (h: number) => String(h).padStart(2, '0')
const instructors = (s: MySection) => s.staff.filter((x) => x.role === 'instructor').map(personName)

export function Schedule({ slug, term, sections }: { slug: string; term: Term; sections: MySection[] }) {
  const id = useId()
  const timed = sections.filter((s) => s.meetings.length > 0)
  const untimed = sections.filter((s) => s.meetings.length === 0)
  const all = timed.flatMap((s) => s.meetings)
  const start = Math.min(480, Math.floor(Math.min(...all.map((m) => toMinutes(m.startsAt))) / 60) * 60)
  const end = Math.max(1080, Math.ceil(Math.max(...all.map((m) => toMinutes(m.endsAt))) / 60) * 60)
  const rows = end - start
  const hours = Array.from({ length: (end - start) / 60 }, (_, i) => start / 60 + i)
  return (
    <section aria-labelledby={`${id}-h`}>
      <h2 id={`${id}-h`} className="sr-only">
        Weekly schedule, {term.name}
      </h2>
      {timed.length === 0 ? (
        <p className="mb-4 text-sm text-ink-muted">None of your sections in {term.name} has meeting times yet.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-line bg-surface">
          <div className="grid min-w-[48rem] grid-cols-[3.5rem_repeat(7,minmax(0,1fr))]">
            <div aria-hidden="true">
              <div className="h-8 border-b border-line bg-canvas" />
              {hours.map((h) => (
                <div key={h} className="h-24 pr-2 text-right text-xs text-ink-muted">
                  {hh(h)}:00
                </div>
              ))}
            </div>
            {days.map((d) => {
              const items = timed
                .flatMap((s) => s.meetings.filter((m) => m.weekday === d).map((m) => ({ s, m })))
                .sort(
                  (a, b) =>
                    a.m.startsAt.localeCompare(b.m.startsAt) ||
                    a.m.endsAt.localeCompare(b.m.endsAt) ||
                    classCode(a.s).localeCompare(classCode(b.s)),
                )
              const { lane, lanes } = assignLanes(items.map((i) => i.m))
              return (
                <section key={d} aria-labelledby={`${id}-d${d}`} className="border-l border-line">
                  <h3 id={`${id}-d${d}`} className="h-8 border-b border-line bg-canvas text-center text-xs leading-8 font-bold">
                    <span aria-hidden="true">{dayShort[d - 1]}</span>
                    <span className="sr-only">{dayLong[d - 1]}</span>
                  </h3>
                  {items.length === 0 && <p className="sr-only">No meetings</p>}
                  <ol
                    className="grid"
                    style={{
                      gridTemplateRows: `repeat(${rows}, 0.1rem)`,
                      gridTemplateColumns: `repeat(${lanes}, minmax(0, 1fr))`,
                      backgroundImage: 'repeating-linear-gradient(to bottom, var(--color-line) 0 1px, transparent 1px 6rem)',
                    }}
                  >
                    {items.map(({ s, m }, i) => {
                      const clash = clashesFor({ id: s.id, meetings: [m] }, sections)
                      const names = instructors(s)
                      return (
                        <li
                          key={`${s.id}-${m.id}`}
                          className="min-h-0 p-0.5"
                          style={{
                            gridRow: `${toMinutes(m.startsAt) - start + 1} / ${toMinutes(m.endsAt) - start + 1}`,
                            gridColumn: lane[i] + 1,
                          }}
                        >
                          <Link
                            to="/$slug/sections/$id"
                            params={{ slug, id: s.id }}
                            data-clash={clash.length > 0 || undefined}
                            className={`block h-full overflow-hidden rounded p-1.5 text-xs leading-4 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus ${
                              clash.length ? 'border-2 border-dashed border-danger bg-danger-soft' : 'border border-line border-l-4 border-l-accent bg-surface hover:bg-canvas'
                            }`}
                          >
                            <span className="block">
                              <span className="font-bold">{classCode(s)}</span>{' '}
                              <span className="text-ink-muted">
                                {m.startsAt}–{m.endsAt}
                              </span>
                            </span>
                            {clash.length > 0 && (
                              <span className="block truncate font-bold text-danger">
                                Clashes with {clash.map(classCode).join(', ')}
                              </span>
                            )}
                            <span className="block truncate">{s.course.title}</span>
                            {names.length > 0 && <span className="block truncate text-ink-muted">{names.join(', ')}</span>}
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
              const names = instructors(s)
              return (
                <li key={s.id}>
                  <Link to="/$slug/sections/$id" params={{ slug, id: s.id }} className="font-medium text-accent hover:underline">
                    {classCode(s)}
                  </Link>{' '}
                  — {s.course.title}
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
