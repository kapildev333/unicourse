import { useEffect, useId, useRef } from 'react'

export function Dialog(props: {
  open: boolean
  onClose: () => void
  title: string
  description?: string
  children: React.ReactNode
}) {
  const titleId = useId()
  const descId = useId()
  const ref = useRef<HTMLDialogElement>(null)
  const { open, onClose, title, description, children } = props

  useEffect(() => {
    const d = ref.current
    if (!d) return
    if (open && !d.open) d.showModal()
    if (!open && d.open) d.close()
  }, [open])

  // native <dialog> traps focus and closes on Escape
  return (
    <dialog
      ref={ref}
      onClose={onClose}
      aria-labelledby={titleId}
      aria-describedby={description ? descId : undefined}
      className="w-full max-w-md rounded-lg border border-line bg-surface p-6 text-ink shadow-xl backdrop:bg-ink/40"
    >
      <h2 id={titleId} className="text-base font-semibold">
        {title}
      </h2>
      {description && (
        <p id={descId} className="mt-1 text-sm text-ink-muted">
          {description}
        </p>
      )}
      <div className="mt-4">{children}</div>
    </dialog>
  )
}
