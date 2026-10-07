import type { LucideIcon } from 'lucide-react'

export function EmptyState(props: { title: string; body: string; icon?: LucideIcon; action?: React.ReactNode }) {
  const Icon = props.icon
  return (
    <div className="rounded-lg border border-line bg-surface px-6 py-12 text-center">
      {Icon && <Icon size={20} className="mx-auto mb-3 text-ink-muted" />}
      <p className="text-base font-semibold text-ink">{props.title}</p>
      <p className="mx-auto mt-1 max-w-prose text-sm text-ink-muted">{props.body}</p>
      {props.action && <div className="mt-4 flex justify-center">{props.action}</div>}
    </div>
  )
}
