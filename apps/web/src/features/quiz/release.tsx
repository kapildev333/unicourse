import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { Button } from '../../components/button'
import { Dialog, DialogFooter } from '../../components/dialog'
import { errorMessage } from '../../components/form-errors'
import { patchAssessment, useAssessmentRefresh, type Assessment } from './queries'

/** Hide scores at once; Release asks first because students then see keys and feedback. */
export function ReleaseButton(props: {
  slug: string
  sectionId: string
  a: Assessment
  note?: string
  variant?: 'ghost' | 'secondary'
  onDone: (message: string) => void
  onError: (message: string) => void
}) {
  const { slug, sectionId, a, note, variant = 'ghost', onDone, onError } = props
  const refresh = useAssessmentRefresh(slug, sectionId)
  const [confirm, setConfirm] = useState(false)
  const publish = !a.published
  const m = useMutation({
    mutationFn: () => patchAssessment(slug, sectionId, a.id, { published: publish }),
    onSuccess: async () => {
      await refresh()
      setConfirm(false)
      onDone(`${a.name}: scores ${publish ? 'released to students' : 'hidden from students'}.`)
    },
    onError: (e) => {
      setConfirm(false)
      onError(`Couldn't change ${a.name}: ${errorMessage(e)}`)
    },
  })
  const keys = a.kind === 'quiz'
  return (
    <>
      <Button
        variant={variant}
        size="sm"
        disabled={m.isPending}
        aria-label={`${publish ? 'Release' : 'Hide'} scores for ${a.name}`}
        onClick={() => (publish ? setConfirm(true) : m.mutate())}
      >
        {publish ? 'Release scores' : 'Hide scores'}
      </Button>
      {confirm && (
        <Dialog
          open
          title={`Release scores for ${a.name}?`}
          description={`${keys ? 'Students will see their scores, feedback and the correct answers.' : 'Students will see their scores and feedback.'}${note ? ` ${note}` : ''}`}
          onClose={() => setConfirm(false)}
        >
          <DialogFooter>
            <Button variant="secondary" onClick={() => setConfirm(false)}>
              Cancel
            </Button>
            <Button disabled={m.isPending} onClick={() => m.mutate()}>
              {m.isPending ? 'Releasing…' : 'Release scores'}
            </Button>
          </DialogFooter>
        </Dialog>
      )}
    </>
  )
}
