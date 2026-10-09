import { useEffect, useState } from 'react'
import { formatDateTime } from '../../lib/format'

export type WindowState = 'upcoming' | 'open' | 'closed'
type Win = { opensAt?: string; closesAt?: string; timeLimitMinutes?: number; lateUntil?: string }

/** `1 minute` / `{n} minutes`. */
export const minutes = (n: number) => `${n} ${n === 1 ? 'minute' : 'minutes'}`

/** Display only. `late` = from closesAt until lateUntil (the assignment's late window). */
export function assignmentState(a: Win, now: number): 'upcoming' | 'open' | 'late' | 'closed' {
  const s = windowState(a, now)
  if (s === 'closed' && a.lateUntil && now < Date.parse(a.lateUntil)) return 'late'
  return s
}

/** Display only; the server decides (D140). */
export function windowState(a: Win, now: number): WindowState {
  if (a.opensAt && now < Date.parse(a.opensAt)) return 'upcoming'
  if (a.closesAt && now >= Date.parse(a.closesAt)) return 'closed'
  return 'open'
}

export function windowLine(a: Win, kind: 'quiz' | 'assignment' = 'quiz'): string {
  const opens = a.opensAt ? formatDateTime(a.opensAt) : ''
  const closes = a.closesAt ? formatDateTime(a.closesAt) : ''
  const limit = a.timeLimitMinutes ? `, ${a.timeLimitMinutes} min each` : ''
  const late = a.lateUntil ? `, late until ${formatDateTime(a.lateUntil)}` : ''
  return `Opens ${opens} · ${kind === 'quiz' ? 'Closes' : 'Due'} ${closes}${limit}${late}`
}

const pad = (n: number) => String(n).padStart(2, '0')

/** ISO instant -> `YYYY-MM-DDTHH:mm` in local time, for datetime-local. */
export function toLocalInput(iso: string): string {
  const d = new Date(iso)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** datetime-local value -> ISO instant; undefined for '' or invalid. */
export function fromLocalInput(v: string): string | undefined {
  if (!v) return undefined
  const d = new Date(v)
  return Number.isNaN(d.getTime()) ? undefined : d.toISOString()
}

export const timeZone = () => Intl.DateTimeFormat().resolvedOptions().timeZone

/** Date.now(), re-read every `ms`. */
export function useNow(ms: number): number {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), ms)
    return () => clearInterval(t)
  }, [ms])
  return now
}
