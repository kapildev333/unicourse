import { Plus } from 'lucide-react'
import { useEffect, useRef } from 'react'
import type { components } from '../../api/types'
import { Button } from '../../components/button'
import { FormField, inputClass } from '../../components/form-field'
import { dayLong } from '../../lib/schedule'

type MeetingInput = components['schemas']['MeetingInput']
/** key: React key and error mapping, never sent. */
export type MeetingRow = Required<MeetingInput> & { key: string }

let counter = 0
export const nextKey = () => String(counter++)

export function newRow(prev?: MeetingRow): MeetingRow {
  return prev
    ? { ...prev, key: nextKey(), weekday: (prev.weekday % 7) + 1 }
    : { key: nextKey(), weekday: 1, startsAt: '', endsAt: '', location: '' }
}

export const toInput = ({
  weekday,
  startsAt,
  endsAt,
  location,
}: MeetingRow | components['schemas']['Meeting']): MeetingInput => ({
  weekday,
  startsAt,
  endsAt,
  location: location.trim(),
})

const MAX = 20

export function MeetingsEditor(props: {
  rows: MeetingRow[]
  onChange: (rows: MeetingRow[]) => void
  /** useFieldErrors(m.error, copy) from the dialog */
  fields: Record<string, string>
  /** row keys in the order last submitted (maps details index -> row) */
  sentKeys: string[]
}) {
  const { rows, onChange, fields, sentKeys } = props
  const metaRef = useRef<HTMLParagraphElement>(null)
  // no aria-invalid control for a list-level error, so focus the message itself
  useEffect(() => {
    if (fields.meetings) metaRef.current?.focus()
  }, [fields])
  const set = (i: number, patch: Partial<MeetingRow>) =>
    onChange(rows.map((r, k) => (k === i ? { ...r, ...patch } : r)))
  return (
    <fieldset className="flex flex-col gap-3">
      <legend className="mb-2 text-sm font-medium">Meeting times</legend>
      {fields.meetings && (
        <p ref={metaRef} tabIndex={-1} role="alert" className="text-sm text-danger">
          {fields.meetings}
        </p>
      )}
      {rows.length === 0 && <p className="text-sm text-ink-muted">No meeting times.</p>}
      {rows.map((r, i) => {
        const j = sentKeys.indexOf(r.key)
        const has = (f: string) => j >= 0 && fields[`meetings[${j}].${f}`]
        return (
          <div key={r.key} role="group" aria-label={`Meeting ${i + 1}`} className="flex flex-wrap items-start gap-2">
            <div className="w-36">
              <FormField
                label="Day"
                name={`meetings[${i}].weekday`}
                error={has('weekday') ? 'Choose a day.' : undefined}
              >
                <select
                  className={inputClass}
                  value={r.weekday}
                  onChange={(e) => set(i, { weekday: Number(e.target.value) })}
                >
                  {dayLong.map((d, k) => (
                    <option key={d} value={k + 1}>
                      {d}
                    </option>
                  ))}
                </select>
              </FormField>
            </div>
            <div className="w-32">
              <FormField
                label="Starts"
                name={`meetings[${i}].startsAt`}
                error={has('startsAt') ? 'Enter a start time.' : undefined}
              >
                <input
                  type="time"
                  className={inputClass}
                  value={r.startsAt}
                  onChange={(e) => set(i, { startsAt: e.target.value })}
                />
              </FormField>
            </div>
            <div className="w-32">
              <FormField
                label="Ends"
                name={`meetings[${i}].endsAt`}
                error={has('endsAt') ? 'Enter an end time after the start time.' : undefined}
              >
                <input
                  type="time"
                  className={inputClass}
                  value={r.endsAt}
                  onChange={(e) => set(i, { endsAt: e.target.value })}
                />
              </FormField>
            </div>
            <div className="w-48">
              <FormField
                label="Location"
                name={`meetings[${i}].location`}
                error={has('location') ? 'Enter a location of at most 100 characters.' : undefined}
              >
                <input
                  type="text"
                  maxLength={100}
                  className={inputClass}
                  value={r.location}
                  onChange={(e) => set(i, { location: e.target.value })}
                />
              </FormField>
            </div>
            <Button
              variant="ghost"
              size="sm"
              className="mt-7.5"
              aria-label={`Remove meeting ${i + 1}`}
              onClick={() => onChange(rows.filter((_, k) => k !== i))}
            >
              Remove
            </Button>
          </div>
        )
      })}
      <div>
        <Button
          variant="secondary"
          size="sm"
          disabled={rows.length >= MAX}
          onClick={() => onChange([...rows, newRow(rows.at(-1))])}
        >
          <Plus />
          Add meeting time
        </Button>
        {rows.length >= MAX && <p className="mt-1 text-xs text-ink-muted">At most 20 meeting times.</p>}
      </div>
    </fieldset>
  )
}
