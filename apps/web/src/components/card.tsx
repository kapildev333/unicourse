import { Link } from '@tanstack/react-router'
import { useId } from 'react'

/** A section as a card: flat cover band with the class code, bold title, muted meta lines. Render inside a <ul>. */
export function Card(props: {
  code: string
  title: string
  meta: string[]
  badge?: React.ReactNode
  note?: string // shown in danger text: the clash line
  footer?: React.ReactNode // foot link(s), outside the main link (never nest links)
  params: { slug: string; id: string }
}) {
  const { code, title, meta, badge, note, footer, params } = props
  const id = useId()
  return (
    <li className="flex flex-col overflow-hidden rounded border border-line bg-surface text-ink transition-colors hover:border-line-strong hover:shadow-card">
      <Link
        to="/$slug/sections/$id"
        params={params}
        aria-labelledby={`${id}-code ${id}-title`}
        className="flex flex-1 flex-col focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-focus"
      >
        <div className="flex h-16 items-end bg-accent-soft px-4 py-3">
          <span id={`${id}-code`} className="text-xl leading-none font-bold">
            {code}
          </span>
        </div>
        <div className="flex flex-1 flex-col gap-1 p-4">
          <h3 id={`${id}-title`} className="text-base leading-snug font-bold">
            {title}
          </h3>
          {meta.map((m, i) => (
            <p key={i} className="text-sm text-ink-muted">
              {m}
            </p>
          ))}
          {badge && <div className="mt-auto pt-3">{badge}</div>}
          {note && <p className="mt-1 text-sm font-bold text-danger">{note}</p>}
        </div>
      </Link>
      {footer && <div className="border-t border-line px-4 py-2 text-sm">{footer}</div>}
    </li>
  )
}
