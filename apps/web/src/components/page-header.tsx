/** `as="h2"` is the heading of a tab inside a page that already has its h1 (admin tabs): smaller title, tighter spacing. */
export function PageHeader(props: {
  title: string
  description?: React.ReactNode
  action?: React.ReactNode
  as?: 'h1' | 'h2'
}) {
  const sub = props.as === 'h2'
  return (
    <div className={`${sub ? 'mb-6' : 'mb-8'} flex flex-wrap items-end justify-between gap-4`}>
      <div>
        {sub ? <h2 className="text-xl font-bold text-ink">{props.title}</h2> : <h1 className="text-ink">{props.title}</h1>}
        {props.description && (
          <p className={`${sub ? 'mt-1' : 'mt-2'} text-base text-ink-muted`}>{props.description}</p>
        )}
      </div>
      {props.action}
    </div>
  )
}
