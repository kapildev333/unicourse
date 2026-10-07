import { Monitor, Moon, Sun } from 'lucide-react'
import { useState } from 'react'
import { apply, read, write, type ThemeChoice } from '../lib/theme'

const options = [
  { value: 'system', label: 'System', Icon: Monitor },
  { value: 'light', label: 'Light', Icon: Sun },
  { value: 'dark', label: 'Dark', Icon: Moon },
] as const

export function ThemeSwitch() {
  const [choice, setChoice] = useState<ThemeChoice>(read)
  return (
    <div
      role="group"
      aria-label="Theme"
      className="inline-flex h-10 items-center gap-0.5 rounded border border-line-strong p-0.5 pointer-coarse:h-auto pointer-coarse:min-h-11"
    >
      {options.map(({ value, label, Icon }) => (
        <button
          key={value}
          type="button"
          aria-label={label}
          title={label}
          aria-pressed={choice === value}
          onClick={() => {
            write(value)
            apply(value)
            setChoice(value)
          }}
          className="inline-flex size-8 items-center justify-center rounded transition-colors pointer-coarse:size-11 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus not-aria-pressed:text-ink-muted not-aria-pressed:hover:bg-canvas not-aria-pressed:hover:text-ink aria-pressed:bg-accent-soft aria-pressed:text-accent-hover"
        >
          <Icon size={16} />
        </button>
      ))}
    </div>
  )
}
