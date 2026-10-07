/** Loaded-count line above a list: "Showing 5" or, with `noun`, "Showing 5 sections" (S5-34). `more`: another page exists and the API gives no total, so "Showing 50 or more changes". No middle dots in counts (P9-L9c). */
export function ListCount({
  shown,
  noun,
  more,
  className = '',
}: {
  shown: number
  noun?: string
  more?: boolean
  className?: string
}) {
  return (
    <p role="status" className={`text-sm text-ink-muted ${className}`}>
      Showing {shown}
      {more && ' or more'}
      {noun && ` ${shown === 1 ? noun : `${noun}s`}`}
    </p>
  )
}
