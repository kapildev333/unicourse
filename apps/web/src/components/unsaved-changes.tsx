import { useBlocker } from '@tanstack/react-router'
import { Button } from './button'
import { Dialog, DialogFooter } from './dialog'

/** Blocks router navigation and tab close while `when` is true; asks before discarding. */
export function UnsavedChangesGuard({ when }: { when: boolean }) {
  const blocker = useBlocker({ shouldBlockFn: () => when, withResolver: true, enableBeforeUnload: () => when })
  if (blocker.status !== 'blocked') return null
  return (
    <Dialog open title="Discard unsaved changes?" description="Your changes on this page haven't been saved." onClose={blocker.reset}>
      <DialogFooter>
        <Button variant="secondary" onClick={blocker.reset}>
          Keep editing
        </Button>
        <Button variant="danger" onClick={blocker.proceed}>
          Discard changes
        </Button>
      </DialogFooter>
    </Dialog>
  )
}
