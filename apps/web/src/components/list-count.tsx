/** Loaded-count line above a list: "Showing 5" or, with `noun`, "Showing 5 sections" (S5-34). */
export function ListCount({ shown, noun, className = '' }: { shown: number; noun?: string; className?: string }) {
  return (
    <p role="status" className={`text-sm text-ink-muted ${className}`}>
      Showing {shown}
      {noun && ` ${shown === 1 ? noun : `${noun}s`}`}
    </p>
  )
}
