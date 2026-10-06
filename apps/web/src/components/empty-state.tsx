export function EmptyState(props: { title: string; body: string; action?: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-dashed border-line-strong bg-surface px-6 py-12 text-center">
      <p className="text-sm font-semibold">{props.title}</p>
      <p className="mt-1 text-sm text-ink-muted">{props.body}</p>
      {props.action && <div className="mt-4">{props.action}</div>}
    </div>
  )
}
