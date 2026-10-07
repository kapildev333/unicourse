export function Loading({ label = 'Loading…' }: { label?: string }) {
  return (
    <div role="status" className="flex flex-col gap-3 py-6">
      <span className="sr-only">{label}</span>
      <span aria-hidden="true" className="ui-skeleton w-3/4" />
      <span aria-hidden="true" className="ui-skeleton w-full" />
      <span aria-hidden="true" className="ui-skeleton w-5/6" />
    </div>
  )
}
