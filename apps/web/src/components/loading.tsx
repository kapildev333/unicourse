export function Loading({ label = 'Loading…' }: { label?: string }) {
  return (
    <p role="status" className="py-12 text-center text-sm text-ink-muted">
      {label}
    </p>
  )
}
