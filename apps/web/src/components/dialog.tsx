import { X } from 'lucide-react'
import { useEffect, useId, useRef } from 'react'
import { Button } from './button'

export function Dialog(props: {
  open: boolean
  onClose: () => void
  title: string
  description?: string
  wide?: boolean
  size?: 'md' | 'lg'
  children: React.ReactNode
}) {
  const titleId = useId()
  const descId = useId()
  const ref = useRef<HTMLDialogElement>(null)
  const trigger = useRef<Element | null>(null)
  const { open, onClose, title, description, wide, size = 'md', children } = props

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
      className={`m-auto w-[calc(100%-2rem)] max-h-[calc(100dvh-2rem)] overflow-y-auto sm:w-full ${wide ? 'max-w-4xl' : size === 'lg' ? 'max-w-2xl' : 'max-w-md'} rounded border border-line bg-surface p-6 text-ink shadow-dialog`}
    >
      <h2 id={titleId} className="pr-8 text-xl font-bold">
        {title}
      </h2>
      {description && (
        <p id={descId} className="mt-1 text-sm text-ink-muted">
          {description}
        </p>
      )}
      <div className="mt-5">{children}</div>
      {/* last in DOM so native autofocus still lands on the first control (Cancel), placed top right by CSS */}
      <Button variant="ghost" size="sm" aria-label="Close" title="Close" className="absolute top-3 right-3" onClick={() => ref.current?.close()}>
        <X aria-hidden="true" />
      </Button>
    </dialog>
  )
}

export function DialogFooter({ children }: { children: React.ReactNode }) {
  return <div className="sticky -bottom-6 -mb-6 pb-6 mt-2 flex items-center justify-end gap-2 border-t border-line bg-surface pt-4">{children}</div>
}
