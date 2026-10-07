import { useEffect, useId, useRef } from 'react'

export function Dialog(props: {
  open: boolean
  onClose: () => void
  title: string
  description?: string
  wide?: boolean
  children: React.ReactNode
}) {
  const titleId = useId()
  const descId = useId()
  const ref = useRef<HTMLDialogElement>(null)
  const trigger = useRef<Element | null>(null)
  const { open, onClose, title, description, wide, children } = props

  useEffect(() => {
    const d = ref.current
    if (!d) return
    // record on the transition only: StrictMode's second run would otherwise record an element inside the dialog
    if (open && !d.open) {
      trigger.current = document.activeElement // read before showModal moves focus
      d.showModal()
    }
    if (!open && d.open) d.close()
    if (!open) return
    // conditionally mounted dialogs are removed without close(); hand focus back to the trigger either way
    return () => {
      const t = trigger.current
      if (t instanceof HTMLElement && t.isConnected) t.focus()
    }
  }, [open])

  // native <dialog> traps focus and closes on Escape
  return (
    <dialog
      ref={ref}
      onClose={onClose}
      aria-labelledby={titleId}
      aria-describedby={description ? descId : undefined}
      className={`w-full ${wide ? 'max-w-4xl' : 'max-w-md'} rounded-lg border border-line bg-surface p-6 text-ink shadow-dialog`}
    >
      <h2 id={titleId} className="text-lg font-semibold">
        {title}
      </h2>
      {description && (
        <p id={descId} className="mt-1 text-sm text-ink-muted">
          {description}
        </p>
      )}
      <div className="mt-5">{children}</div>
    </dialog>
  )
}

export function DialogFooter({ children }: { children: React.ReactNode }) {
  return <div className="mt-2 flex items-center justify-end gap-2 border-t border-line pt-4">{children}</div>
}
