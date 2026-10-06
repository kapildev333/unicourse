import { useEffect, useId, useRef } from 'react'

export function Dialog(props: { open: boolean; onClose: () => void; title: string; children: React.ReactNode }) {
  const titleId = useId()
  const ref = useRef<HTMLDialogElement>(null)
  const { open, onClose, title, children } = props

  useEffect(() => {
    const d = ref.current
    if (!d) return
    if (open && !d.open) d.showModal()
    if (!open && d.open) d.close()
  }, [open])

  // native <dialog> traps focus and closes on Escape
  return (
    <dialog ref={ref} onClose={onClose} aria-labelledby={titleId} className="rounded p-4">
      <h2 id={titleId} className="mb-2 text-lg font-semibold">
        {title}
      </h2>
      {children}
    </dialog>
  )
}
