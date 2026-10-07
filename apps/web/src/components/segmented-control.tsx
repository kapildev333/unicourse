import { useRef } from 'react'

/** Toggle between a few views. Selected: bg-accent-soft text-ink; others ghost. Arrow keys move the choice. */
export function SegmentedControl<T extends string>(props: {
  label: string
  options: { value: T; label: string }[]
  value: T
  onChange: (v: T) => void
}) {
  const { label, options, value, onChange } = props
  const refs = useRef<(HTMLButtonElement | null)[]>([])
  const move = (i: number, d: number) => {
    const n = (i + d + options.length) % options.length
    onChange(options[n].value)
    refs.current[n]?.focus()
  }
  return (
    <div role="group" aria-label={label} className="inline-flex gap-1 rounded border border-line-control p-1">
      {options.map((o, i) => (
        <button
          key={o.value}
          ref={(el) => {
            refs.current[i] = el
          }}
          type="button"
          aria-pressed={o.value === value}
          onClick={() => onChange(o.value)}
          onKeyDown={(e) => {
            const d = e.key === 'ArrowRight' || e.key === 'ArrowDown' ? 1 : e.key === 'ArrowLeft' || e.key === 'ArrowUp' ? -1 : 0
            if (!d) return
            e.preventDefault()
            move(i, d)
          }}
          className={`h-8 rounded px-3 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus ${
            o.value === value ? 'bg-accent-soft text-ink' : 'text-accent hover:bg-accent-soft hover:text-accent-hover'
          }`}
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}
