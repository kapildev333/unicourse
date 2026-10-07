import { ChevronDown, ChevronRight } from 'lucide-react'
import { useState } from 'react'

/** Native <details> with a Lucide chevron instead of the browser triangle (no rotation). */
export function Disclosure(props: { summary: React.ReactNode; children: React.ReactNode; className?: string }) {
  const [open, setOpen] = useState(false)
  const Icon = open ? ChevronDown : ChevronRight
  return (
    <details open={open} onToggle={(e) => setOpen(e.currentTarget.open)} className={props.className}>
      <summary className="flex cursor-pointer list-none items-center gap-1 text-accent hover:text-accent-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus [&::-webkit-details-marker]:hidden">
        <Icon aria-hidden="true" />
        {props.summary}
      </summary>
      {props.children}
    </details>
  )
}
