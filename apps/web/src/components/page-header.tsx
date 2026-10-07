export function PageHeader(props: { title: string; description?: React.ReactNode; action?: React.ReactNode }) {
  return (
    <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 className="text-ink">{props.title}</h1>
        {props.description && <p className="mt-2 text-base text-ink-muted">{props.description}</p>}
      </div>
      {props.action}
    </div>
  )
}
