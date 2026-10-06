import { Button } from '../../components/button'
import { Dialog } from '../../components/dialog'
import { FormErrors, type FieldCopy } from '../../components/form-errors'

/** Confirm-and-destroy dialog; focus starts on Cancel (first focusable in DOM order). */
export function RemoveDialog(props: {
  open: boolean
  title: string
  description: string
  confirmLabel: string
  pendingLabel: string
  pending?: boolean
  error?: unknown
  copy?: FieldCopy
  onConfirm: () => void
  onClose: () => void
}) {
  const { open, title, description, confirmLabel, pendingLabel, pending, error, copy, onConfirm, onClose } = props
  return (
    <Dialog open={open} onClose={onClose} title={title} description={description}>
      <div className="flex flex-col gap-4">
        <FormErrors error={error} copy={copy} />
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="danger" disabled={pending} onClick={onConfirm}>
            {pending ? pendingLabel : confirmLabel}
          </Button>
        </div>
      </div>
    </Dialog>
  )
}
