/** Loaded-count line above a list: "Showing 5" or, with `noun`, "Showing 5 sections" (S5-34). `total`: "Showing 50 of 120 changes" when the API gives one. No middle dots in counts (P9-L9c). */
export function ListCount({
  shown,
  noun,
  total,
  className = '',
}: {
  shown: number
  noun?: string
  total?: number
  className?: string
}) {
  return (
    <p role="status" className={`text-sm text-ink-muted ${className}`}>
      Showing {shown}
      {total !== undefined && ` of ${total}`}
      {noun && ` ${(total ?? shown) === 1 ? noun : `${noun}s`}`}
    </p>
  )
}
