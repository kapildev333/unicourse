import { useEffect, useRef } from 'react'

export function Dialog(props: { open: boolean; onClose: () => void; title: string; children: React.ReactNode }) {
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
    <dialog ref={ref} onClose={onClose} aria-labelledby="dialog-title" className="rounded p-4">
      <h2 id="dialog-title" className="mb-2 text-lg font-semibold">
        {title}
      </h2>
      {children}
    </dialog>
  )
}
