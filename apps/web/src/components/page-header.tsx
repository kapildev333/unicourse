export function PageHeader(props: { title: string; description?: string; action?: React.ReactNode }) {
  return (
    <div className="mb-6 flex flex-wrap items-end justify-between gap-4 border-b border-line pb-4">
      <div>
        <h1 className="text-2xl text-ink">{props.title}</h1>
        {props.description && <p className="mt-1 text-sm text-ink-muted">{props.description}</p>}
      </div>
      {props.action}
    </div>
  )
}
