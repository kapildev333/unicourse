/** Heading row of an admin tab: the page h1 ("Admin") lives in AdminLayout, so each tab opens with an h2. */
export function TabHeader(props: { title: string; description?: React.ReactNode; action?: React.ReactNode }) {
  return (
    <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
      <div>
        <h2 className="text-xl font-bold text-ink">{props.title}</h2>
        {props.description && <p className="mt-1 text-base text-ink-muted">{props.description}</p>}
      </div>
      {props.action}
    </div>
  )
}
